//go:build go1.18
// +build go1.18

package cache

import (
	"runtime"
	"sync/atomic"
	"time"
)

var (
	_ CacheOf[string, any] = (*xsyncMapOfWrapper[string, any])(nil)
	_ CacheOf[int, any]    = (*xsyncMapOfWrapper[int, any])(nil)
)

type xsyncMapOfWrapper[K comparable, V any] struct {
	*xsyncMapOf[K, V]
}

type xsyncMapOf[K comparable, V any] struct {
	defaultExpiration atomic.Value
	evictedCallback   atomic.Value
	items             MapOf[K, itemOf[V]]
	stop              chan struct{}
}

// Creates a new MapOf instance with capacity enough to hold sizeHint entries.
func newXsyncMapOf[K comparable, V any](
	config ...ConfigOf[K, V],
) CacheOf[K, V] {
	cfg := configDefaultOf(config...)
	c := &xsyncMapOf[K, V]{
		items: NewMapOfPresized[K, itemOf[V]](cfg.MinCapacity),
		stop:  make(chan struct{}),
	}
	c.defaultExpiration.Store(cfg.DefaultExpiration)
	c.evictedCallback.Store(cfg.EvictedCallback)

	if cfg.CleanupInterval > 0 {
		go func() {
			ticker := time.NewTicker(cfg.CleanupInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					c.DeleteExpired()
				case <-c.stop:
					return
				}
			}
		}()
	}

	cache := &xsyncMapOfWrapper[K, V]{c}
	runtime.SetFinalizer(cache, func(m *xsyncMapOfWrapper[K, V]) { close(m.stop) })
	return cache
}

// Create a new cache with arbitrarily typed keys,
// specifying the default expiration duration and cleanup interval.
// If the cleanup interval is less than 1, the cleanup needs to be performed manually,
// calling c.DeleteExpired()
func newXsyncMapOfDefault[K comparable, V any](
	defaultExpiration,
	cleanupInterval time.Duration,
	evictedCallback ...EvictedCallbackOf[K, V],
) CacheOf[K, V] {
	cfg := ConfigOf[K, V]{
		DefaultExpiration: defaultExpiration,
		CleanupInterval:   cleanupInterval,
	}
	if len(evictedCallback) > 0 {
		cfg.EvictedCallback = evictedCallback[0]
	}
	return newXsyncMapOf[K, V](cfg)
}

// Set add item to the cache, replacing any existing items.
// (DefaultExpiration), the item uses a cached default expiration time.
// (NoExpiration), the item never expires.
// All values less than or equal to 0 are the same except DefaultExpiration,
// which means never expires.
func (c *xsyncMapOf[K, V]) Set(k K, v V, d time.Duration) {
	c.items.Store(k, itemOf[V]{
		v: v,
		e: c.expiration(d),
	})
}

func (c *xsyncMapOf[K, V]) expiration(d time.Duration) (e int64) {
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
func (c *xsyncMapOf[K, V]) SetDefault(k K, v V) {
	c.Set(k, v, DefaultExpiration)
}

// SetForever add item to cache and set to never expire, replacing any existing items.
func (c *xsyncMapOf[K, V]) SetForever(k K, v V) {
	c.Set(k, v, NoExpiration)
}

// Get an item from the cache.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
func (c *xsyncMapOf[K, V]) Get(k K) (V, bool) {
	i, ok := c.get(k)
	if ok {
		return i.v, true
	}
	return i.v, false
}

func (c *xsyncMapOf[K, V]) get(k K) (itemOf[V], bool) {
	var zeroedV itemOf[V]
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
		func(value itemOf[V], loaded bool) (itemOf[V], bool) {
			if loaded && !value.expired() {
				// k has a new value
				return value, false
			}
			// delete
			return zeroedV, true
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
func (c *xsyncMapOf[K, V]) GetWithExpiration(k K) (V, time.Time, bool) {
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
func (c *xsyncMapOf[K, V]) GetWithTTL(k K) (V, time.Duration, bool) {
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
func (c *xsyncMapOf[K, V]) GetOrSet(k K, v V, d time.Duration) (V, bool) {
	var ok bool
	i, _ := c.items.Compute(
		k,
		func(value itemOf[V], loaded bool) (itemOf[V], bool) {
			if loaded && !value.expired() {
				ok = true
				return value, false
			}
			return itemOf[V]{
				v: v,
				e: c.expiration(d),
			}, false
		},
	)
	return i.v, ok
}

// GetAndSet returns the existing value for the key if present,
// while setting the new value for the key.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false otherwise.
func (c *xsyncMapOf[K, V]) GetAndSet(k K, v V, d time.Duration) (V, bool) {
	var (
		ok  bool
		old itemOf[V]
	)
	i, _ := c.items.Compute(
		k,
		func(value itemOf[V], loaded bool) (itemOf[V], bool) {
			if loaded && !value.expired() {
				ok = true
				old = value
			}
			return itemOf[V]{
				v: v,
				e: c.expiration(d),
			}, false
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
func (c *xsyncMapOf[K, V]) GetAndRefresh(k K, d time.Duration) (V, bool) {
	var zeroedV itemOf[V]
	i, ok := c.items.Compute(
		k,
		func(value itemOf[V], loaded bool) (itemOf[V], bool) {
			if loaded && !value.expired() {
				// store new value
				value.e = c.expiration(d)
				return value, false
			}
			// delete
			return zeroedV, true
		},
	)
	if ok {
		return i.v, true
	}
	return zeroedV.v, false
}

// GetOrCompute returns the existing value for the key if present.
// Otherwise, it computes the value using the provided function and
// returns the computed value. The loaded result is true if the value
// was loaded, false if stored.
func (c *xsyncMapOf[K, V]) GetOrCompute(k K, valueFn func() V, d time.Duration) (V, bool) {
	var ok bool
	i, _ := c.items.Compute(
		k,
		func(value itemOf[V], loaded bool) (itemOf[V], bool) {
			if loaded && !value.expired() {
				ok = true
				return value, false
			}
			return itemOf[V]{
				v: valueFn(),
				e: c.expiration(d),
			}, false
		},
	)
	return i.v, ok
}

// Compute either sets the computed new value for the key or deletes
// the value for the key. When the delete result of the valueFn function
// is set to true, the value will be deleted, if it exists. When delete
// is set to false, the value is updated to the newValue.
// The ok result indicates whether value was computed and stored, thus, is
// present in the map. The actual result contains the new value in cases where
// the value was computed and stored. See the example for a few use cases.
func (c *xsyncMapOf[K, V]) Compute(
	k K,
	valueFn func(oldValue V, loaded bool) (newValue V, delete bool),
	d time.Duration,
) (V, bool) {
	var old V
	i, ok := c.items.Compute(
		k,
		func(ov itemOf[V], lok bool) (nv itemOf[V], del bool) {
			var v V
			if lok && !ov.expired() {
				// current value
				old = ov.v
			} else {
				lok = false
			}
			v, del = valueFn(old, lok)
			if del {
				return
			}
			return itemOf[V]{
				v: v,
				e: c.expiration(d),
			}, false
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
func (c *xsyncMapOf[K, V]) GetAndDelete(k K) (V, bool) {
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
func (c *xsyncMapOf[K, V]) Delete(k K) {
	c.GetAndDelete(k)
}

type kvOf[K comparable, V any] struct {
	k K
	v V
}

// DeleteExpired delete all expired items from the cache.
func (c *xsyncMapOf[K, V]) DeleteExpired() {
	var evictedItems []kvOf[K, V]
	ec := c.EvictedCallback()
	now := time.Now().UnixNano()
	c.items.Range(func(k K, v itemOf[V]) bool {
		i := v
		if i.expiredWithNow(now) {
			c.items.Delete(k)
			if ec != nil {
				evictedItems = append(evictedItems, kvOf[K, V]{k, i.v})
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
func (c *xsyncMapOf[K, V]) Range(f func(k K, v V) bool) {
	if f == nil {
		return
	}
	now := time.Now().UnixNano()
	c.items.Range(func(k K, v itemOf[V]) bool {
		i := v
		if i.expiredWithNow(now) {
			return true
		}
		return f(k, i.v)
	})
}

// Items return the items in the cache.
// This is a snapshot, which may include items that are about to expire.
func (c *xsyncMapOf[K, V]) Items() map[K]V {
	items := make(map[K]V, c.items.Size())
	c.Range(func(k K, v V) bool {
		items[k] = v
		return true
	})
	return items
}

// Clear deletes all keys and values currently stored in the map.
func (c *xsyncMapOf[K, V]) Clear() {
	c.items.Clear()
}

// Count returns the number of items in the cache.
// This may include items that have expired but have not been cleaned up.
func (c *xsyncMapOf[K, V]) Count() int {
	return c.items.Size()
}

// DefaultExpiration returns the default expiration time of the cache.
func (c *xsyncMapOf[K, V]) DefaultExpiration() time.Duration {
	return c.defaultExpiration.Load().(time.Duration)
}

// SetDefaultExpiration sets the default expiration time for the cache.
// Atomic safety.
func (c *xsyncMapOf[K, V]) SetDefaultExpiration(defaultExpiration time.Duration) {
	c.defaultExpiration.Store(defaultExpiration)
}

// EvictedCallback returns the callback function to execute
// when a key-value pair expires and is evicted.
func (c *xsyncMapOf[K, V]) EvictedCallback() EvictedCallbackOf[K, V] {
	return c.evictedCallback.Load().(EvictedCallbackOf[K, V])
}

// SetEvictedCallback Set the callback function to be executed
// when the key-value pair expires and is evicted.
// Atomic safety.
func (c *xsyncMapOf[K, V]) SetEvictedCallback(evictedCallback EvictedCallbackOf[K, V]) {
	c.evictedCallback.Store(evictedCallback)
}
