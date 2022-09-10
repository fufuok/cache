package cache

import (
	"github.com/fufuok/cache/internal/xsync"
)

type Map interface {
	// Store add item to the cache, replacing any existing items.
	Store(k string, v interface{})

	// Load an item from the cache.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	Load(k string) (interface{}, bool)

	// LoadOrStore returns the existing value for the key if present.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false if stored.
	LoadOrStore(k string, v interface{}) (interface{}, bool)

	// LoadAndStore returns the existing value for the key if present,
	// while setting the new value for the key.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false otherwise.
	LoadAndStore(k string, v interface{}) (interface{}, bool)

	// LoadAndDelete Get an item from the cache, and delete the key.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	LoadAndDelete(k string) (interface{}, bool)

	// Delete an item from the cache.
	// Does nothing if the key is not in the cache.
	Delete(k string)

	// Range calls f sequentially for each key and value present in the map.
	// If f returns false, range stops the iteration.
	Range(f func(k string, v interface{}) bool)

	// Size returns the number of items in the cache.
	// This may include items that have expired but have not been cleaned up.
	Size() int
}

// NewMap the keys never expire, similar to the use of sync.Map.
func NewMap() Map {
	return xsync.NewMap()
}
