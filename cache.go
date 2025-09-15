package cache

import (
	"time"
)

type Cache[K comparable, V any] interface {
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
	GetOrCompute(k K, valueFn func() (newValue V, cancel bool), d time.Duration) (value V, loaded bool)

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
	Compute(
		k K,
		valueFn func(oldValue V, loaded bool) (newValue V, op ComputeOp),
		d time.Duration,
	) (actual V, ok bool)

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

	// ItemsWithExpiration return the items in the cache with their expiration times.
	// This is a snapshot, which may include items that are about to expire.
	// The returned map contains items where the time.Time is zero for items that never expire.
	ItemsWithExpiration() map[K]ItemWithExpiration[V]

	// LoadItems loads multiple items into the cache.
	// This is useful for bulk loading data from external sources.
	LoadItems(items map[K]V, defaultExpiration time.Duration)

	// LoadItemsWithExpiration loads multiple items with their expiration times into the cache.
	// Items with zero expiration time will never expire.
	LoadItemsWithExpiration(items map[K]ItemWithExpiration[V])

	// Clear deletes all keys and values currently stored in the map.
	Clear()

	// Close closes the cache and releases any resources associated with it.
	Close()

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
	EvictedCallback() EvictedCallback[K, V]

	// SetEvictedCallback Set the callback function to be executed
	// when the key-value pair expires and is evicted.
	// Atomic safety.
	SetEvictedCallback(evictedCallback EvictedCallback[K, V])
}

// ItemWithExpiration represents a cache item with its expiration time
// Zero time means never expires
type ItemWithExpiration[V any] struct {
	Value      V         `json:"value"`
	Expiration time.Time `json:"expiration"`
}

func New[K comparable, V any](opts ...Option[K, V]) Cache[K, V] {
	cfg := DefaultConfig[K, V]()
	for _, opt := range opts {
		opt(&cfg)
	}
	return newXsyncMap[K, V](cfg)
}

func NewDefault[K comparable, V any](
	defaultExpiration,
	cleanupInterval time.Duration,
	evictedCallback ...EvictedCallback[K, V],
) Cache[K, V] {
	return newXsyncMapDefault[K, V](defaultExpiration, cleanupInterval, evictedCallback...)
}
