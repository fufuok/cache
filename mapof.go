//go:build go1.18
// +build go1.18

package cache

import (
	"hash/maphash"

	"github.com/fufuok/cache/internal/xsync"
)

type MapOf[K comparable, V any] interface {
	// Load returns the value stored in the map for a key, or nil if no
	// value is present.
	// The ok result indicates whether value was found in the map.
	Load(key K) (value V, ok bool)

	// Store sets the value for a key.
	Store(key K, value V)

	// LoadOrStore returns the existing value for the key if present.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false if stored.
	LoadOrStore(key K, value V) (actual V, loaded bool)

	// LoadAndStore returns the existing value for the key if present,
	// while setting the new value for the key.
	// It stores the new value and returns the existing one, if present.
	// The loaded result is true if the existing value was loaded,
	// false otherwise.
	LoadAndStore(key K, value V) (actual V, loaded bool)

	// LoadOrCompute returns the existing value for the key if present.
	// Otherwise, it computes the value using the provided function and
	// returns the computed value. The loaded result is true if the value
	// was loaded, false if stored.
	LoadOrCompute(key K, valueFn func() V) (actual V, loaded bool)

	// Compute either sets the computed new value for the key or deletes
	// the value for the key. When the delete result of the valueFn function
	// is set to true, the value will be deleted, if it exists. When delete
	// is set to false, the value is updated to the newValue.
	// The ok result indicates whether value was computed and stored, thus, is
	// present in the map. The actual result contains the new value in cases where
	// the value was computed and stored. See the example for a few use cases.
	Compute(
		key K,
		valueFn func(oldValue V, loaded bool) (newValue V, delete bool),
	) (actual V, ok bool)

	// LoadAndDelete deletes the value for a key, returning the previous
	// value if any. The loaded result reports whether the key was
	// present.
	LoadAndDelete(key K) (value V, loaded bool)

	// Delete deletes the value for a key.
	Delete(key K)

	// Range calls f sequentially for each key and value present in the
	// map. If f returns false, range stops the iteration.
	//
	// Range does not necessarily correspond to any consistent snapshot
	// of the Map's contents: no key will be visited more than once, but
	// if the value for any key is stored or deleted concurrently, Range
	// may reflect any mapping for that key from any point during the
	// Range call.
	//
	// It is safe to modify the map while iterating it. However, the
	// concurrent modification rule apply, i.e. the changes may be not
	// reflected in the subsequently iterated entries.
	Range(f func(key K, value V) bool)

	// Clear deletes all keys and values currently stored in the map.
	Clear()

	// Size returns current size of the map.
	Size() int
}

// NewMapOf creates a new HashMapOf instance with string keys.
// The keys never expire, similar to the use of sync.Map.
func NewMapOf[V any]() MapOf[string, V] {
	return xsync.NewMapOf[V]()
}

// NewIntegerMapOf creates a new HashMapOf instance with integer typed keys.
func NewIntegerMapOf[K IntegerConstraint, V any]() MapOf[K, V] {
	return xsync.NewIntegerMapOf[K, V]()
}

// NewHashMapOf creates a new HashMapOf instance with arbitrarily typed keys.
// If no hasher is specified, an automatic generation will be attempted.
// Hashable allowed map key types constraint.
// Automatically generated hashes for these types are safe:
//
//	type Hashable interface {
//		~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
//		~float32 | ~float64 | ~string | ~complex64 | ~complex128
//	}
func NewHashMapOf[K comparable, V any](hasher ...func(maphash.Seed, K) uint64) MapOf[K, V] {
	return xsync.NewHashMapOf[K, V](hasher...)
}

// NewTypedMapOf creates a new HashMapOf instance with arbitrarily typed keys.
// Keys are hashed to uint64 using the hasher function. Note that StrHash64
// function might be handy when writing the hasher function for structs with
// string fields.
func NewTypedMapOf[K comparable, V any](hasher func(maphash.Seed, K) uint64) MapOf[K, V] {
	return xsync.NewTypedMapOf[K, V](hasher)
}
