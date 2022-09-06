//go:build go1.18
// +build go1.18

package cache

import (
	"time"
)

type CacheOf[V any] interface {
	// Set add item to the cache, replacing any existing items.
	// (DefaultExpiration), the item uses a cached default expiration time.
	// (NoExpiration), the item never expires.
	// All values less than or equal to 0 are the same except DefaultExpiration, which means never expires.
	Set(k string, v V, d time.Duration)

	// SetDefault add item to the cache with the default expiration time, replacing any existing items.
	SetDefault(k string, v V)

	// SetForever add item to cache and set to never expire, replacing any existing items.
	SetForever(k string, v V)

	// Get an item from the cache.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	Get(k string) (V, bool)

	// GetWithExpiration get an item from the cache.
	// Returns the item or nil,
	// along with the expiration time, and a boolean indicating whether the key was found.
	GetWithExpiration(k string) (V, time.Time, bool)

	// GetWithTTL get an item from the cache.
	// Returns the item or nil,
	// with the remaining lifetime and a boolean indicating whether the key was found.
	GetWithTTL(k string) (V, time.Duration, bool)

	// GetOrSet returns the existing value for the key if present.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false if stored.
	GetOrSet(k string, v V, d time.Duration) (V, bool)

	// GetAndSet returns the existing value for the key if present,
	// while setting the new value for the key.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false otherwise.
	GetAndSet(k string, v V, d time.Duration) (V, bool)

	// GetAndRefresh Get an item from the cache, and refresh the item's expiration time.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	// Allows getting keys that have expired but not been evicted.
	// Not atomic synchronization.
	GetAndRefresh(k string, d time.Duration) (V, bool)

	// GetAndDelete Get an item from the cache, and delete the key.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	GetAndDelete(k string) (V, bool)

	// Delete an item from the cache.
	// Does nothing if the key is not in the cache.
	Delete(k string)

	// DeleteExpired delete all expired items from the cache.
	DeleteExpired()

	// Range calls f sequentially for each key and value present in the map.
	// If f returns false, range stops the iteration.
	Range(f func(k string, v V) bool)

	// Items return the items in the cache.
	// This is a snapshot, which may include items that are about to expire.
	Items() map[string]V

	// Count returns the number of items in the cache.
	// This may include items that have expired but have not been cleaned up.
	Count() int
}

func NewOf[V any](opts ...OptionOf[V]) CacheOf[V] {
	cfg := DefaultConfigOf[V]()
	for _, opt := range opts {
		opt(&cfg)
	}
	return NewXsyncMapOf[V](cfg)
}

func NewOfDefault[V any](
	defaultExpiration,
	cleanupInterval time.Duration,
	evictedCallback ...EvictedCallbackOf[V],
) CacheOf[V] {
	opts := []OptionOf[V]{
		WithDefaultExpirationOf[V](defaultExpiration),
		WithCleanupIntervalOf[V](cleanupInterval),
	}
	if len(evictedCallback) > 0 {
		opts = append(opts, WithEvictedCallbackOf(evictedCallback[0]))
	}
	return NewOf[V](opts...)
}
