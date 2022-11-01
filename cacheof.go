//go:build go1.18
// +build go1.18

package cache

import (
	"hash/maphash"
	"time"

	"github.com/fufuok/cache/internal/xxhash"
)

type CacheOf[K comparable, V any] interface {
	// Set add item to the cache, replacing any existing items.
	// (DefaultExpiration), the item uses a cached default expiration time.
	// (NoExpiration), the item never expires.
	// All values less than or equal to 0 are the same except DefaultExpiration,
	// which means never expires.
	Set(k K, v V, d time.Duration)

	// SetDefault add item to the cache with the default expiration time,
	// replacing any existing items.
	SetDefault(k K, v V)

	// SetForever add item to cache and set to never expire, replacing any existing items.
	SetForever(k K, v V)

	// Get an item from the cache.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	Get(k K) (value V, ok bool)

	// GetWithExpiration get an item from the cache.
	// Returns the item or nil,
	// along with the expiration time, and a boolean indicating whether the key was found.
	GetWithExpiration(k K) (value V, expiration time.Time, ok bool)

	// GetWithTTL get an item from the cache.
	// Returns the item or nil,
	// with the remaining lifetime and a boolean indicating whether the key was found.
	GetWithTTL(k K) (value V, ttl time.Duration, ok bool)

	// GetOrSet returns the existing value for the key if present.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false if stored.
	GetOrSet(k K, v V, d time.Duration) (value V, loaded bool)

	// GetAndSet returns the existing value for the key if present,
	// while setting the new value for the key.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false otherwise.
	GetAndSet(k K, v V, d time.Duration) (value V, loaded bool)

	// GetAndRefresh Get an item from the cache, and refresh the item's expiration time.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	GetAndRefresh(k K, d time.Duration) (value V, loaded bool)

	// GetOrCompute returns the existing value for the key if present.
	// Otherwise, it computes the value using the provided function and
	// returns the computed value. The loaded result is true if the value
	// was loaded, false if stored.
	GetOrCompute(k K, valueFn func() V, d time.Duration) (V, bool)

	// Compute either sets the computed new value for the key or deletes
	// the value for the key. When the delete result of the valueFn function
	// is set to true, the value will be deleted, if it exists. When delete
	// is set to false, the value is updated to the newValue.
	// The ok result indicates whether value was computed and stored, thus, is
	// present in the map. The actual result contains the new value in cases where
	// the value was computed and stored. See the example for a few use cases.
	Compute(
		k K,
		valueFn func(oldValue V, loaded bool) (newValue V, delete bool),
		d time.Duration,
	) (V, bool)

	// GetAndDelete Get an item from the cache, and delete the key.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	GetAndDelete(k K) (value V, loaded bool)

	// Delete an item from the cache.
	// Does nothing if the key is not in the cache.
	Delete(k K)

	// DeleteExpired delete all expired items from the cache.
	DeleteExpired()

	// Range calls f sequentially for each key and value present in the map.
	// If f returns false, range stops the iteration.
	Range(f func(k K, v V) bool)

	// Items return the items in the cache.
	// This is a snapshot, which may include items that are about to expire.
	Items() map[K]V

	// Clear deletes all keys and values currently stored in the map.
	Clear()

	// Count returns the number of items in the cache.
	// This may include items that have expired but have not been cleaned up.
	Count() int

	// DefaultExpiration returns the default expiration time for the cache.
	DefaultExpiration() time.Duration

	// SetDefaultExpiration sets the default expiration time for the cache.
	// Atomic safety.
	SetDefaultExpiration(defaultExpiration time.Duration)

	// EvictedCallback returns the callback function to execute
	// when a key-value pair expires and is evicted.
	EvictedCallback() EvictedCallbackOf[K, V]

	// SetEvictedCallback Set the callback function to be executed
	// when the key-value pair expires and is evicted.
	// Atomic safety.
	SetEvictedCallback(evictedCallback EvictedCallbackOf[K, V])
}

func NewOf[V any](opts ...OptionOf[string, V]) CacheOf[string, V] {
	return NewTypedOf[string, V](HashString, opts...)
}

func NewIntegerOf[K IntegerConstraint, V any](opts ...OptionOf[K, V]) CacheOf[K, V] {
	return NewTypedOf[K, V](Hash64[K], opts...)
}

func NewHashOf[K comparable, V any](opts ...OptionOf[K, V]) CacheOf[K, V] {
	hasher := xxhash.GenSeedHasher64[K]()
	return NewTypedOf[K, V](hasher, opts...)
}

func NewTypedOf[K comparable, V any](hasher func(maphash.Seed, K) uint64, opts ...OptionOf[K, V]) CacheOf[K, V] {
	cfg := DefaultConfigOf[K, V]()
	for _, opt := range opts {
		opt(&cfg)
	}
	return newXsyncTypedMapOf[K, V](hasher, cfg)
}

func NewOfDefault[V any](
	defaultExpiration,
	cleanupInterval time.Duration,
	evictedCallback ...EvictedCallbackOf[string, V],
) CacheOf[string, V] {
	return NewTypedOfDefault[string, V](HashString, defaultExpiration, cleanupInterval, evictedCallback...)
}

func NewIntegerOfDefault[K IntegerConstraint, V any](
	defaultExpiration,
	cleanupInterval time.Duration,
	evictedCallback ...EvictedCallbackOf[K, V],
) CacheOf[K, V] {
	return NewTypedOfDefault[K, V](Hash64[K], defaultExpiration, cleanupInterval, evictedCallback...)
}

func NewHashOfDefault[K comparable, V any](
	defaultExpiration,
	cleanupInterval time.Duration,
	evictedCallback ...EvictedCallbackOf[K, V],
) CacheOf[K, V] {
	hasher := xxhash.GenSeedHasher64[K]()
	return NewTypedOfDefault[K, V](hasher, defaultExpiration, cleanupInterval, evictedCallback...)
}

func NewTypedOfDefault[K comparable, V any](
	hasher func(maphash.Seed, K) uint64,
	defaultExpiration,
	cleanupInterval time.Duration,
	evictedCallback ...EvictedCallbackOf[K, V],
) CacheOf[K, V] {
	opts := []OptionOf[K, V]{
		WithDefaultExpirationOf[K, V](defaultExpiration),
		WithCleanupIntervalOf[K, V](cleanupInterval),
	}
	if len(evictedCallback) > 0 {
		opts = append(opts, WithEvictedCallbackOf[K, V](evictedCallback[0]))
	}
	return NewTypedOf[K, V](hasher, opts...)
}
