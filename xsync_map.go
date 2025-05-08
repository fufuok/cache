package cache

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/fufuok/cache/xsync"
)

var (
	_ Cache[string, any] = (*xsyncMapWrapper[string, any])(nil)
	_ Cache[int, any]    = (*xsyncMapWrapper[int, any])(nil)
)

type xsyncMapWrapper[K comparable, V any] struct {
	*xsyncMap[K, V]
}

type xsyncMap[K comparable, V any] struct {
	defaultExpiration atomic.Value
	evictedCallback   atomic.Value
	items             Map[K, item[V]]
	stop              chan struct{}
	closed            atomic.Bool
}

// Creates a new Map instance with capacity enough to hold sizeHint entries.
func newXsyncMap[K comparable, V any](
	config ...Config[K, V],
) Cache[K, V] {
	cfg := configDefault(config...)
	c := &xsyncMap[K, V]{
		items: xsync.NewMap[K, item[V]](xsync.WithPresize(cfg.MinCapacity)),
		stop:  make(chan struct{}),
	}
	c.defaultExpiration.Store(cfg.DefaultExpiration)
	c.evictedCallback.Store(cfg.EvictedCallback)

	if cfg.CleanupInterval > 0 {
		go c.startCleanupLoop(cfg.CleanupInterval)
	}

	cache := &xsyncMapWrapper[K, V]{c}
	runtime.SetFinalizer(cache, func(m *xsyncMapWrapper[K, V]) { m.close() })
	return cache
}

// Create a new cache with arbitrarily typed keys,
// specifying the default expiration duration and cleanup interval.
// If the cleanup interval is less than 1, the cleanup needs to be performed manually,
// calling c.DeleteExpired()
func newXsyncMapDefault[K comparable, V any](
	defaultExpiration,
	cleanupInterval time.Duration,
	evictedCallback ...EvictedCallback[K, V],
) Cache[K, V] {
	cfg := Config[K, V]{
		DefaultExpiration: defaultExpiration,
		CleanupInterval:   cleanupInterval,
	}
	if len(evictedCallback) > 0 {
		cfg.EvictedCallback = evictedCallback[0]
	}
	return newXsyncMap[K, V](cfg)
}

func (c *xsyncMap[K, V]) startCleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-c.stop:
			return
		}
	}
}

// Set add item to the cache, replacing any existing items.
// (DefaultExpiration), the item uses a cached default expiration time.
// (NoExpiration), the item never expires.
// All values less than or equal to 0 are the same except DefaultExpiration,
// which means never expires.
func (c *xsyncMap[K, V]) Set(k K, v V, d time.Duration) {
	c.items.Store(k, item[V]{
		v: v,
		e: c.expiration(d),
	})
}

func (c *xsyncMap[K, V]) expiration(d time.Duration) (e int64) {
	if d == DefaultExpiration {
		d = c.DefaultExpiration()
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	return
}

// SetDefault add item to the cache with the default expiration time,
// replacing any existing items.
func (c *xsyncMap[K, V]) SetDefault(k K, v V) {
	c.Set(k, v, DefaultExpiration)
}

// SetForever add item to cache and set to never expire, replacing any existing items.
func (c *xsyncMap[K, V]) SetForever(k K, v V) {
	c.Set(k, v, NoExpiration)
}

// Get an item from the cache.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
func (c *xsyncMap[K, V]) Get(k K) (V, bool) {
	i, ok := c.get(k)
	if ok {
		return i.v, true
	}
	return i.v, false
}

func (c *xsyncMap[K, V]) get(k K) (item[V], bool) {
	var zeroedV item[V]
	i, ok := c.items.Load(k)
	if !ok {
		return zeroedV, false
	}

	if !i.expired() {
		return i, true
	}

	// double check or delete
	i, ok = c.items.Compute(
		k,
		func(value item[V], loaded bool) (item[V], ComputeOp) {
			if loaded && !value.expired() {
				// k has a new value
				return value, CancelOp
			}
			// delete
			return zeroedV, DeleteOp
		},
	)
	if ok {
		return i, true
	}
	return zeroedV, false
}

// GetWithExpiration get an item from the cache.
// Returns the item or nil,
// along with the expiration time, and a boolean indicating whether the key was found.
func (c *xsyncMap[K, V]) GetWithExpiration(k K) (V, time.Time, bool) {
	i, ok := c.get(k)
	if !ok {
		// not found
		var v V
		return v, time.Time{}, false
	}
	if i.e > 0 {
		// with expiration
		return i.v, time.Unix(0, i.e), true
	}
	// never expires
	return i.v, time.Time{}, true
}

// GetWithTTL get an item from the cache.
// Returns the item or nil,
// with the remaining lifetime and a boolean indicating whether the key was found.
func (c *xsyncMap[K, V]) GetWithTTL(k K) (V, time.Duration, bool) {
	i, ok := c.get(k)
	if !ok {
		// not found
		var zeroedV V
		return zeroedV, 0, false
	}
	if i.e > 0 {
		// with ttl
		return i.v, time.Until(time.Unix(0, i.e)), true
	}
	// never expires
	return i.v, NoExpiration, true
}

// GetOrSet returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (c *xsyncMap[K, V]) GetOrSet(k K, v V, d time.Duration) (V, bool) {
	var ok bool
	i, _ := c.items.Compute(
		k,
		func(value item[V], loaded bool) (item[V], ComputeOp) {
			if loaded && !value.expired() {
				ok = true
				return value, CancelOp
			}
			return item[V]{
				v: v,
				e: c.expiration(d),
			}, UpdateOp
		},
	)
	return i.v, ok
}

// GetAndSet returns the existing value for the key if present,
// while setting the new value for the key.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false otherwise.
func (c *xsyncMap[K, V]) GetAndSet(k K, v V, d time.Duration) (V, bool) {
	var (
		ok  bool
		old item[V]
	)
	i, _ := c.items.Compute(
		k,
		func(value item[V], loaded bool) (item[V], ComputeOp) {
			if loaded && !value.expired() {
				ok = true
				old = value
			}
			return item[V]{
				v: v,
				e: c.expiration(d),
			}, UpdateOp
		},
	)
	if ok {
		return old.v, true
	}
	return i.v, false
}

// GetAndRefresh Get an item from the cache, and refresh the item's expiration time.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
func (c *xsyncMap[K, V]) GetAndRefresh(k K, d time.Duration) (V, bool) {
	var zeroedV item[V]
	i, ok := c.items.Compute(
		k,
		func(value item[V], loaded bool) (item[V], ComputeOp) {
			if loaded && !value.expired() {
				// store new value
				value.e = c.expiration(d)
				return value, UpdateOp
			}
			// delete
			return zeroedV, DeleteOp
		},
	)
	if ok {
		return i.v, true
	}
	return zeroedV.v, false
}

// GetOrCompute returns the existing value for the key if
// present. Otherwise, it tries to compute the value using the
// provided function and, if successful, stores and returns
// the computed value. The loaded result is true if the value was
// loaded, or false if computed. If valueFn returns true as the
// cancel value, the computation is cancelled and the zero value
// for type V is returned.
//
// This call locks a hash table bucket while the compute function
// is executed. It means that modifications on other entries in
// the bucket will be blocked until the valueFn executes. Consider
// this when the function includes long-running operations.
func (c *xsyncMap[K, V]) GetOrCompute(k K, valueFn func() (newValue V, cancel bool), d time.Duration) (V, bool) {
	var ok bool
	i, _ := c.items.Compute(
		k,
		func(value item[V], loaded bool) (item[V], ComputeOp) {
			if loaded && !value.expired() {
				ok = true
				return value, CancelOp
			}
			newValue, cancel := valueFn()
			if !cancel {
				return item[V]{
					v: newValue,
					e: c.expiration(d),
				}, UpdateOp
			}
			return value, CancelOp
		},
	)
	return i.v, ok
}

// Compute either sets the computed new value for the key,
// deletes the value for the key, or does nothing, based on
// the returned [ComputeOp]. When the op returned by valueFn
// is [UpdateOp], the value is updated to the new value. If
// it is [DeleteOp], the entry is removed from the map
// altogether. And finally, if the op is [CancelOp] then the
// entry is left as-is. In other words, if it did not already
// exist, it is not created, and if it did exist, it is not
// updated. This is useful to synchronously execute some
// operation on the value without incurring the cost of
// updating the map every time. The ok result indicates
// whether the entry is present in the map after the compute
// operation. The actual result contains the value of the map
// if a corresponding entry is present, or the zero value
// otherwise. See the example for a few use cases.
//
// This call locks a hash table bucket while the compute function
// is executed. It means that modifications on other entries in
// the bucket will be blocked until the valueFn executes. Consider
// this when the function includes long-running operations.
func (c *xsyncMap[K, V]) Compute(
	k K,
	valueFn func(oldValue V, loaded bool) (newValue V, op ComputeOp),
	d time.Duration,
) (V, bool) {
	var old V
	i, ok := c.items.Compute(
		k,
		func(ov item[V], lok bool) (nv item[V], op ComputeOp) {
			var v V
			if lok && !ov.expired() {
				// current value
				old = ov.v
			} else {
				lok = false
			}
			v, op = valueFn(old, lok)
			switch op {
			case DeleteOp:
				nv = ov
			case UpdateOp:
				nv = item[V]{
					v: v,
					e: c.expiration(d),
				}
			case CancelOp:
				nv = ov
			}
			return
		},
	)
	if ok {
		return i.v, true
	}
	return old, false
}

// GetAndDelete Get an item from the cache, and delete the key.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
func (c *xsyncMap[K, V]) GetAndDelete(k K) (V, bool) {
	i, ok := c.items.LoadAndDelete(k)
	if !ok {
		var v V
		return v, false
	}
	ec := c.EvictedCallback()
	if ec != nil {
		ec(k, i.v)
	}
	return i.v, true
}

// Delete an item from the cache.
// Does nothing if the key is not in the cache.
func (c *xsyncMap[K, V]) Delete(k K) {
	c.GetAndDelete(k)
}

type kv[K comparable, V any] struct {
	k K
	v V
}

// DeleteExpired delete all expired items from the cache.
func (c *xsyncMap[K, V]) DeleteExpired() {
	var evictedItems []kv[K, V]
	ec := c.EvictedCallback()
	now := time.Now().UnixNano()
	c.items.Range(func(k K, v item[V]) bool {
		i := v
		if i.expiredWithNow(now) {
			c.items.Delete(k)
			if ec != nil {
				evictedItems = append(evictedItems, kv[K, V]{k, i.v})
			}
		}
		return true
	})
	for _, v := range evictedItems {
		ec(v.k, v.v)
	}
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (c *xsyncMap[K, V]) Range(f func(k K, v V) bool) {
	if f == nil {
		return
	}
	now := time.Now().UnixNano()
	c.items.Range(func(k K, v item[V]) bool {
		i := v
		if i.expiredWithNow(now) {
			return true
		}
		return f(k, i.v)
	})
}

// Items return the items in the cache.
// This is a snapshot, which may include items that are about to expire.
func (c *xsyncMap[K, V]) Items() map[K]V {
	items := make(map[K]V, c.items.Size())
	c.Range(func(k K, v V) bool {
		items[k] = v
		return true
	})
	return items
}

// Clear deletes all keys and values currently stored in the map.
func (c *xsyncMap[K, V]) Clear() {
	c.items.Clear()
}

// Close closes the cache and releases any resources associated with it.
func (c *xsyncMap[K, V]) Close() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.stop)
	}
}

// closes the cache and releases any resources associated with it.
func (c *xsyncMapWrapper[K, V]) close() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.stop)
	}
}

// Count returns the number of items in the cache.
// This may include items that have expired but have not been cleaned up.
func (c *xsyncMap[K, V]) Count() int {
	return c.items.Size()
}

// DefaultExpiration returns the default expiration time of the cache.
func (c *xsyncMap[K, V]) DefaultExpiration() time.Duration {
	return c.defaultExpiration.Load().(time.Duration)
}

// SetDefaultExpiration sets the default expiration time for the cache.
// Atomic safety.
func (c *xsyncMap[K, V]) SetDefaultExpiration(defaultExpiration time.Duration) {
	c.defaultExpiration.Store(defaultExpiration)
}

// EvictedCallback returns the callback function to execute
// when a key-value pair expires and is evicted.
func (c *xsyncMap[K, V]) EvictedCallback() EvictedCallback[K, V] {
	return c.evictedCallback.Load().(EvictedCallback[K, V])
}

// SetEvictedCallback Set the callback function to be executed
// when the key-value pair expires and is evicted.
// Atomic safety.
func (c *xsyncMap[K, V]) SetEvictedCallback(evictedCallback EvictedCallback[K, V]) {
	c.evictedCallback.Store(evictedCallback)
}
