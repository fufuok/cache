package cache

import (
	"github.com/fufuok/cache/xsync"
)

type ComputeOp = xsync.ComputeOp

const (
	// CancelOp signals to Compute to not do anything as a result
	// of executing the lambda. If the entry was not present in
	// the map, nothing happens, and if it was present, the
	// returned value is ignored.
	CancelOp ComputeOp = iota
	// UpdateOp signals to Compute to update the entry to the
	// value returned by the lambda, creating it if necessary.
	UpdateOp
	// DeleteOp signals to Compute to always delete the entry
	// from the map.
	DeleteOp
)

type Map[K comparable, V any] interface {
	// Load returns the value stored in the map for a key, or zero value
	// of type V if no value is present.
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

	// LoadOrCompute returns the existing value for the key if
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
	LoadOrCompute(
		key K,
		valueFn func() (newValue V, cancel bool),
	) (value V, loaded bool)

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
		key K,
		valueFn func(oldValue V, loaded bool) (newValue V, op ComputeOp),
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
	// It is safe to modify the map while iterating it, including entry
	// creation, modification and deletion. However, the concurrent
	// modification rule apply, i.e. the changes may be not reflected
	// in the subsequently iterated entries.
	Range(f func(key K, value V) bool)

	// Clear deletes all keys and values currently stored in the map.
	Clear()

	// Size returns current size of the map.
	Size() int
}

// Usage: func NewMap[K comparable, V any](options ...func(*MapConfig)) *Map[K, V]
//
// import: "github.com/fufuok/cache/xsync"
// m := xsync.NewMap[string, int]()
//
// Example: examples/map-usage/main.go
