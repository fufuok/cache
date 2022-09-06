package cache

import (
	"time"
)

type Cache interface {
	// Set add item to the cache, replacing any existing items.
	// (DefaultExpiration), the item uses a cached default expiration time.
	// (NoExpiration), the item never expires.
	// All values less than or equal to 0 are the same except DefaultExpiration, which means never expires.
	Set(k string, v interface{}, d time.Duration)

	// SetDefault add item to the cache with the default expiration time, replacing any existing items.
	SetDefault(k string, v interface{})

	// SetForever add item to cache and set to never expire, replacing any existing items.
	SetForever(k string, v interface{})

	// Get an item from the cache.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	Get(k string) (interface{}, bool)

	// GetWithExpiration get an item from the cache.
	// Returns the item or nil,
	// along with the expiration time, and a boolean indicating whether the key was found.
	GetWithExpiration(k string) (interface{}, time.Time, bool)

	// GetWithTTL get an item from the cache.
	// Returns the item or nil,
	// with the remaining lifetime and a boolean indicating whether the key was found.
	GetWithTTL(k string) (interface{}, time.Duration, bool)

	// GetOrSet returns the existing value for the key if present.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false if stored.
	GetOrSet(k string, v interface{}, d time.Duration) (interface{}, bool)

	// GetAndSet returns the existing value for the key if present,
	// while setting the new value for the key.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false otherwise.
	GetAndSet(k string, v interface{}, d time.Duration) (interface{}, bool)

	// GetAndRefresh Get an item from the cache, and refresh the item's expiration time.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	// Allows getting keys that have expired but not been evicted.
	// Not atomic synchronization.
	GetAndRefresh(k string, d time.Duration) (interface{}, bool)

	// GetAndDelete Get an item from the cache, and delete the key.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	GetAndDelete(k string) (interface{}, bool)

	// Delete an item from the cache.
	// Does nothing if the key is not in the cache.
	Delete(k string)

	// DeleteExpired delete all expired items from the cache.
	DeleteExpired()

	// Range calls f sequentially for each key and value present in the map.
	// If f returns false, range stops the iteration.
	Range(f func(k string, v interface{}) bool)

	// Items return the items in the cache.
	// This is a snapshot, which may include items that are about to expire.
	Items() map[string]interface{}

	// Count returns the number of items in the cache.
	// This may include items that have expired but have not been cleaned up.
	Count() int
}

func New(opts ...Option) Cache {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return NewXsyncMap(cfg)
}

func NewDefault(
	defaultExpiration,
	cleanupInterval time.Duration,
	evictedCallback ...EvictedCallback,
) Cache {
	opts := []Option{
		WithDefaultExpiration(defaultExpiration),
		WithCleanupInterval(cleanupInterval),
	}
	if len(evictedCallback) > 0 {
		opts = append(opts, WithEvictedCallback(evictedCallback[0]))
	}
	return New(opts...)
}