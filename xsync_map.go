package cache

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/fufuok/cache/internal/xsync"
)

var _ Cache = (*xsyncMapWrapper)(nil)

type xsyncMapWrapper struct {
	*xsyncMap
}

type xsyncMap struct {
	defaultExpiration atomic.Value
	evictedCallback   atomic.Value
	items             *xsync.Map
	stop              chan struct{}
}

// Create a new cache, optionally specifying configuration items.
func newXsyncMap(config ...Config) Cache {
	cfg := configDefault(config...)
	c := &xsyncMap{
		items: xsync.NewMapPresized(cfg.MinCapacity),
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

	cache := &xsyncMapWrapper{c}
	runtime.SetFinalizer(cache, func(m *xsyncMapWrapper) { close(m.stop) })
	return cache
}

// Creates a new cache with the given default expiration duration and cleanup interval.
// If the cleanup interval is less than 1, the cleanup needs to be performed manually,
// calling c.DeleteExpired()
func newXsyncMapDefault(defaultExpiration, cleanupInterval time.Duration, evictedCallback ...EvictedCallback) Cache {
	cfg := Config{
		DefaultExpiration: defaultExpiration,
		CleanupInterval:   cleanupInterval,
	}
	if len(evictedCallback) > 0 {
		cfg.EvictedCallback = evictedCallback[0]
	}
	return newXsyncMap(cfg)
}

// Set add item to the cache, replacing any existing items.
// (DefaultExpiration), the item uses a cached default expiration time.
// (NoExpiration), the item never expires.
// All values less than or equal to 0 are the same except DefaultExpiration,
// which means never expires.
func (c *xsyncMap) Set(k string, v interface{}, d time.Duration) {
	c.items.Store(k, item{
		v: v,
		e: c.expiration(d),
	})
}

func (c *xsyncMap) expiration(d time.Duration) (e int64) {
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
func (c *xsyncMap) SetDefault(k string, v interface{}) {
	c.Set(k, v, DefaultExpiration)
}

// SetForever add item to cache and set to never expire, replacing any existing items.
func (c *xsyncMap) SetForever(k string, v interface{}) {
	c.Set(k, v, NoExpiration)
}

// Get an item from the cache.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
func (c *xsyncMap) Get(k string) (interface{}, bool) {
	if v, ok := c.get(k); ok {
		return v.(item).v, true
	}
	return nil, false
}

func (c *xsyncMap) get(k string) (interface{}, bool) {
	v, ok := c.items.Load(k)
	if !ok {
		return nil, false
	}

	i := v.(item)
	if !i.expired() {
		return i, true
	}

	// double check or delete
	v, ok = c.items.Compute(
		k,
		func(value interface{}, loaded bool) (interface{}, bool) {
			if loaded {
				i = value.(item)
				if !i.expired() {
					// k has a new value
					return i, false
				}
			}
			// delete
			return nil, true
		},
	)
	if ok {
		return v, true
	}
	return nil, false
}

// GetWithExpiration get an item from the cache.
// Returns the item or nil,
// along with the expiration time, and a boolean indicating whether the key was found.
func (c *xsyncMap) GetWithExpiration(k string) (interface{}, time.Time, bool) {
	v, ok := c.get(k)
	if !ok {
		// not found
		return nil, time.Time{}, false
	}
	i := v.(item)
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
func (c *xsyncMap) GetWithTTL(k string) (interface{}, time.Duration, bool) {
	v, ok := c.get(k)
	if !ok {
		// not found
		return nil, 0, false
	}
	i := v.(item)
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
func (c *xsyncMap) GetOrSet(k string, v interface{}, d time.Duration) (interface{}, bool) {
	var ok bool
	r, _ := c.items.Compute(
		k,
		func(value interface{}, loaded bool) (interface{}, bool) {
			if loaded {
				old := value.(item)
				if !old.expired() {
					ok = true
					return old, false
				}
			}
			return item{
				v: v,
				e: c.expiration(d),
			}, false
		},
	)
	return r.(item).v, ok
}

// GetAndSet returns the existing value for the key if present,
// while setting the new value for the key.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false otherwise.
func (c *xsyncMap) GetAndSet(k string, v interface{}, d time.Duration) (interface{}, bool) {
	var (
		ok  bool
		old item
	)
	r, _ := c.items.Compute(
		k,
		func(value interface{}, loaded bool) (interface{}, bool) {
			if loaded {
				old = value.(item)
				if !old.expired() {
					ok = true
				}
			}
			return item{
				v: v,
				e: c.expiration(d),
			}, false
		},
	)
	if ok {
		return old.v, true
	}
	return r.(item).v, false
}

// GetAndRefresh Get an item from the cache, and refresh the item's expiration time.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
func (c *xsyncMap) GetAndRefresh(k string, d time.Duration) (interface{}, bool) {
	r, ok := c.items.Compute(
		k,
		func(value interface{}, loaded bool) (interface{}, bool) {
			if loaded {
				i := value.(item)
				if !i.expired() {
					// store new value
					i.e = c.expiration(d)
					return i, false
				}
			}
			// delete
			return nil, true
		},
	)
	if ok {
		return r.(item).v, true
	}
	return nil, false
}

// GetOrCompute returns the existing value for the key if present.
// Otherwise, it computes the value using the provided function and
// returns the computed value. The loaded result is true if the value
// was loaded, false if stored.
func (c *xsyncMap) GetOrCompute(k string, valueFn func() interface{}, d time.Duration) (interface{}, bool) {
	var ok bool
	v, _ := c.items.Compute(
		k,
		func(value interface{}, loaded bool) (interface{}, bool) {
			if loaded {
				i := value.(item)
				if !i.expired() {
					ok = true
					return value, false
				}
			}
			return item{
				v: valueFn(),
				e: c.expiration(d),
			}, false
		},
	)
	return v.(item).v, ok
}

// Compute either sets the computed new value for the key or deletes
// the value for the key. When the delete result of the valueFn function
// is set to true, the value will be deleted, if it exists. When delete
// is set to false, the value is updated to the newValue.
// The ok result indicates whether value was computed and stored, thus, is
// present in the map. The actual result contains the new value in cases where
// the value was computed and stored. See the example for a few use cases.
func (c *xsyncMap) Compute(
	k string,
	valueFn func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool),
	d time.Duration,
) (interface{}, bool) {
	var old interface{}
	v, ok := c.items.Compute(
		k,
		func(ov interface{}, lok bool) (nv interface{}, del bool) {
			var v interface{}
			if lok {
				i := ov.(item)
				if !i.expired() {
					old = i.v
				} else {
					lok = false
				}
			}
			v, del = valueFn(old, lok)
			if del {
				return
			}
			return item{
				v: v,
				e: c.expiration(d),
			}, false
		},
	)
	if ok {
		return v.(item).v, true
	}
	return old, false
}

// GetAndDelete Get an item from the cache, and delete the key.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
func (c *xsyncMap) GetAndDelete(k string) (interface{}, bool) {
	v, ok := c.items.LoadAndDelete(k)
	if !ok {
		return nil, false
	}
	i := v.(item)
	ec := c.EvictedCallback()
	if ec != nil {
		ec(k, i.v)
	}
	return i.v, true
}

// Delete an item from the cache.
// Does nothing if the key is not in the cache.
func (c *xsyncMap) Delete(k string) {
	c.GetAndDelete(k)
}

type kv struct {
	k string
	v interface{}
}

// DeleteExpired delete all expired items from the cache.
func (c *xsyncMap) DeleteExpired() {
	var evictedItems []kv
	ec := c.EvictedCallback()
	now := time.Now().UnixNano()
	c.items.Range(func(k string, v interface{}) bool {
		i := v.(item)
		if i.expiredWithNow(now) {
			c.items.Delete(k)
			if ec != nil {
				evictedItems = append(evictedItems, kv{k, i.v})
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
func (c *xsyncMap) Range(f func(k string, v interface{}) bool) {
	if f == nil {
		return
	}
	now := time.Now().UnixNano()
	c.items.Range(func(k string, v interface{}) bool {
		i := v.(item)
		if i.expiredWithNow(now) {
			return true
		}
		return f(k, i.v)
	})
}

// Items return the items in the cache.
// This is a snapshot, which may include items that are about to expire.
func (c *xsyncMap) Items() map[string]interface{} {
	items := make(map[string]interface{}, c.items.Size())
	c.Range(func(k string, v interface{}) bool {
		items[k] = v
		return true
	})
	return items
}

// Clear deletes all keys and values currently stored in the map.
func (c *xsyncMap) Clear() {
	c.items.Clear()
}

// Count returns the number of items in the cache.
// This may include items that have expired but have not been cleaned up.
func (c *xsyncMap) Count() int {
	return c.items.Size()
}

// DefaultExpiration returns the default expiration time for the cache.
func (c *xsyncMap) DefaultExpiration() time.Duration {
	return c.defaultExpiration.Load().(time.Duration)
}

// SetDefaultExpiration sets the default expiration time for the cache.
// Atomic safety.
func (c *xsyncMap) SetDefaultExpiration(defaultExpiration time.Duration) {
	c.defaultExpiration.Store(defaultExpiration)
}

// EvictedCallback returns the callback function to execute
// when a key-value pair expires and is evicted.
func (c *xsyncMap) EvictedCallback() EvictedCallback {
	return c.evictedCallback.Load().(EvictedCallback)
}

// SetEvictedCallback Set the callback function to be executed
// when the key-value pair expires and is evicted.
// Atomic safety.
func (c *xsyncMap) SetEvictedCallback(evictedCallback EvictedCallback) {
	c.evictedCallback.Store(evictedCallback)
}
