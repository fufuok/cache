//go:build go1.18
// +build go1.18

package cache

import (
	"runtime"
	"time"

	"github.com/fufuok/cache/internal/xsync"
)

var _ CacheOf[any] = (*xsyncMapOfWrapper[any])(nil)

type xsyncMapOfWrapper[V any] struct {
	*xsyncMapOf[V]
}

type xsyncMapOf[V any] struct {
	defaultExpiration time.Duration
	evictedCallback   EvictedCallbackOf[V]
	items             *xsync.MapOf[itemOf[V]]
	stop              chan struct{}
}

// NewXsyncMapOf create a new cache, optionally specifying configuration items.
func NewXsyncMapOf[V any](config ...ConfigOf[V]) CacheOf[V] {
	cfg := configDefaultOf(config...)
	c := &xsyncMapOf[V]{
		defaultExpiration: cfg.DefaultExpiration,
		evictedCallback:   cfg.EvictedCallback,
		items:             xsync.NewMapOf[itemOf[V]](),
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

	cache := &xsyncMapOfWrapper[V]{c}
	runtime.SetFinalizer(cache, func(m *xsyncMapOfWrapper[V]) { close(m.stop) })
	return cache
}

// NewXsyncMapOfDefault creates a new cache with the given default expiration duration and cleanup interval.
// If the cleanup interval is less than 1, the cleanup needs to be performed manually, calling c.DeleteExpired()
func NewXsyncMapOfDefault[V any](
	defaultExpiration,
	cleanupInterval time.Duration,
	evictedCallback ...EvictedCallbackOf[V],
) CacheOf[V] {
	cfg := ConfigOf[V]{
		DefaultExpiration: defaultExpiration,
		CleanupInterval:   cleanupInterval,
	}
	if len(evictedCallback) > 0 {
		cfg.EvictedCallback = evictedCallback[0]
	}
	return NewXsyncMapOf[V](cfg)
}

// Set add item to the cache, replacing any existing items.
// (DefaultExpiration), the item uses a cached default expiration time.
// (NoExpiration), the item never expires.
// All values less than or equal to 0 are the same except DefaultExpiration, which means never expires.
func (c *xsyncMapOf[V]) Set(k string, v V, d time.Duration) {
	c.items.Store(k, itemOf[V]{
		v: v,
		e: c.expiration(d),
	})
}

func (c *xsyncMapOf[V]) expiration(d time.Duration) (e int64) {
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	return
}

// SetDefault add item to the cache with the default expiration time, replacing any existing items.
func (c *xsyncMapOf[V]) SetDefault(k string, v V) {
	c.Set(k, v, DefaultExpiration)
}

// SetForever add item to cache and set to never expire, replacing any existing items.
func (c *xsyncMapOf[V]) SetForever(k string, v V) {
	c.Set(k, v, NoExpiration)
}

// Get an item from the cache.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
func (c *xsyncMapOf[V]) Get(k string) (V, bool) {
	i, ok := c.get(k)
	if ok {
		return i.v, true
	}
	return i.v, false
}

func (c *xsyncMapOf[V]) get(k string) (itemOf[V], bool) {
	i, ok := c.items.Load(k)
	if !ok {
		return itemOf[V]{}, false
	}
	if i.expired() {
		c.GetAndDelete(k)
		return itemOf[V]{}, false
	}
	return i, true
}

// GetWithExpiration get an item from the cache.
// Returns the item or nil,
// along with the expiration time, and a boolean indicating whether the key was found.
func (c *xsyncMapOf[V]) GetWithExpiration(k string) (V, time.Time, bool) {
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
func (c *xsyncMapOf[V]) GetWithTTL(k string) (V, time.Duration, bool) {
	i, ok := c.get(k)
	if !ok {
		// not found
		var v V
		return v, 0, false
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
func (c *xsyncMapOf[V]) GetOrSet(k string, v V, d time.Duration) (V, bool) {
	i, ok := c.items.LoadOrStore(k, itemOf[V]{
		v: v,
		e: c.expiration(d),
	})
	return i.v, ok
}

// GetAndSet returns the existing value for the key if present,
// while setting the new value for the key.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false otherwise.
func (c *xsyncMapOf[V]) GetAndSet(k string, v V, d time.Duration) (V, bool) {
	i, ok := c.items.LoadAndStore(k, itemOf[V]{
		v: v,
		e: c.expiration(d),
	})
	return i.v, ok
}

// GetAndRefresh Get an item from the cache, and refresh the item's expiration time.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
// Allows getting keys that have expired but not been evicted.
// Not atomic synchronization.
func (c *xsyncMapOf[V]) GetAndRefresh(k string, d time.Duration) (V, bool) {
	i, ok := c.items.Load(k)
	if !ok {
		var v V
		return v, false
	}

	e := c.expiration(d)
	if i.e == e {
		return i.v, true
	}

	c.items.Store(k, itemOf[V]{
		v: i.v,
		e: e,
	})
	return i.v, true
}

// GetAndDelete Get an item from the cache, and delete the key.
// Returns the item or nil,
// and a boolean indicating whether the key was found.
func (c *xsyncMapOf[V]) GetAndDelete(k string) (V, bool) {
	i, ok := c.items.LoadAndDelete(k)
	if !ok {
		var v V
		return v, false
	}
	if c.evictedCallback != nil {
		c.evictedCallback(k, i.v)
	}
	return i.v, true
}

// Delete an item from the cache.
// Does nothing if the key is not in the cache.
func (c *xsyncMapOf[V]) Delete(k string) {
	c.GetAndDelete(k)
}

type kvOf[V any] struct {
	k string
	v V
}

// DeleteExpired delete all expired items from the cache.
func (c *xsyncMapOf[V]) DeleteExpired() {
	var evictedItems []kvOf[V]
	now := time.Now().UnixNano()
	c.items.Range(func(k string, v itemOf[V]) bool {
		i := v
		if i.expiredWithNow(now) {
			c.items.Delete(k)
			if c.evictedCallback != nil {
				evictedItems = append(evictedItems, kvOf[V]{k, i.v})
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
func (c *xsyncMapOf[V]) Range(f func(k string, v V) bool) {
	if f == nil {
		return
	}
	now := time.Now().UnixNano()
	c.items.Range(func(k string, v itemOf[V]) bool {
		i := v
		if i.expiredWithNow(now) {
			return true
		}
		return f(k, i.v)
	})
}

// Items return the items in the cache.
// This is a snapshot, which may include items that are about to expire.
func (c *xsyncMapOf[V]) Items() map[string]V {
	items := make(map[string]V, c.items.Size())
	c.Range(func(k string, v V) bool {
		items[k] = v
		return true
	})
	return items
}

// Count returns the number of items in the cache.
// This may include items that have expired but have not been cleaned up.
func (c *xsyncMapOf[V]) Count() int {
	return c.items.Size()
}
