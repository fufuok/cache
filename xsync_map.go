package cache

import (
	"runtime"
	"time"

	"github.com/fufuok/cache/internal/xsync"
)

var _ Cache = (*xsyncMapWrapper)(nil)

type xsyncMapWrapper struct {
	*xsyncMap
}

type xsyncMap struct {
	defaultExpiration time.Duration
	evictedCallback   EvictedCallback
	items             *xsync.Map
	stop              chan struct{}
}

// create a new cache, optionally specifying configuration items.
func newXsyncMap(config ...Config) Cache {
	cfg := configDefault(config...)
	c := &xsyncMap{
		defaultExpiration: cfg.DefaultExpiration,
		evictedCallback:   cfg.EvictedCallback,
		items:             xsync.NewMap(),
		stop:              make(chan struct{}),
	}

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

// creates a new cache with the given default expiration duration and cleanup interval.
// If the cleanup interval is less than 1, the cleanup needs to be performed manually, calling c.DeleteExpired()
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
// All values less than or equal to 0 are the same except DefaultExpiration, which means never expires.
func (c *xsyncMap) Set(k string, v interface{}, d time.Duration) {
	c.items.Store(k, item{
		v: v,
		e: c.expiration(d),
	})
}

func (c *xsyncMap) expiration(d time.Duration) (e int64) {
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	return
}

// SetDefault add item to the cache with the default expiration time, replacing any existing items.
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
	if i, ok := c.get(k); ok {
		return i.v, true
	}
	return nil, false
}

func (c *xsyncMap) get(k string) (item, bool) {
	v, ok := c.items.Load(k)
	if !ok {
		return item{}, false
	}
	i := v.(item)
	if i.expired() {
		c.GetAndDelete(k)
		return item{}, false
	}
	return i, true
}

// GetWithExpiration get an item from the cache.
// Returns the item or nil,
// along with the expiration time, and a boolean indicating whether the key was found.
func (c *xsyncMap) GetWithExpiration(k string) (interface{}, time.Time, bool) {
	i, ok := c.get(k)
	if !ok {
		// not found
		return nil, time.Time{}, false
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
func (c *xsyncMap) GetWithTTL(k string) (interface{}, time.Duration, bool) {
	i, ok := c.get(k)
	if !ok {
		// not found
		return nil, 0, false
	}
	if i.e > 0 {
		// with ttl
		return i.v, time.Unix(0, i.e).Sub(time.Now()), true
	}
	// never expires
	return i.v, NoExpiration, true
}

// GetOrSet returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (c *xsyncMap) GetOrSet(k string, v interface{}, d time.Duration) (interface{}, bool) {
	r, ok := c.items.LoadOrStore(k, item{
		v: v,
		e: c.expiration(d),
	})
	return r.(item).v, ok
}

// GetAndSet returns the existing value for the key if present,
// while setting the new value for the key.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false otherwise.
func (c *xsyncMap) GetAndSet(k string, v interface{}, d time.Duration) (interface{}, bool) {
	r, ok := c.items.LoadAndStore(k, item{
		v: v,
		e: c.expiration(d),
	})
	return r.(item).v, ok
}

// GetAndRefresh Get an item from the cache, and refresh the item's expiration time.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
// Allows getting keys that have expired but not been evicted.
// Not atomic synchronization.
func (c *xsyncMap) GetAndRefresh(k string, d time.Duration) (interface{}, bool) {
	v, ok := c.items.Load(k)
	if !ok {
		return nil, false
	}

	i := v.(item)
	e := c.expiration(d)
	if i.e == e {
		return i.v, true
	}

	c.items.Store(k, item{
		v: i.v,
		e: e,
	})
	return i.v, true
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
	if c.evictedCallback != nil {
		c.evictedCallback(k, i.v)
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
	now := time.Now().UnixNano()
	c.items.Range(func(k string, v interface{}) bool {
		i := v.(item)
		if i.expiredWithNow(now) {
			c.items.Delete(k)
			if c.evictedCallback != nil {
				evictedItems = append(evictedItems, kv{k, i.v})
			}
		}
		return true
	})
	for _, v := range evictedItems {
		c.evictedCallback(v.k, v.v)
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

// Count returns the number of items in the cache.
// This may include items that have expired but have not been cleaned up.
func (c *xsyncMap) Count() int {
	return c.items.Size()
}
