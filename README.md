# 🗃️ go-cache

Goroutine-safe, high-performance in-memory cache, optimized for reads over writes, with expiration, rich API, and support for generics.

Based on [puzpuzpuz/xsync](https://github.com/puzpuzpuz/xsync).

## ⚙️ Installation

```go
go get github.com/fufuok/cache
```

## ⚡️ Quickstart

Please see: [examples](examples)

**Cache or CacheOf usage**

```go
package main

import (
	"time"

	"github.com/fufuok/cache"
)

func main() {
	// for generics
	// c := cache.NewOf[int]()
	c := cache.New()
	c.SetForever("A", 1)
	c.GetOrSet("B", 2, 1*time.Second) // 2 false
	time.Sleep(1 * time.Second)
	c.Get("A") // 1, true
	// for generics
	// c.Get("B") // 0, false
	c.Get("B") // nil, false
}
```

**Map or MapOf usage (similar to sync.Map)**

```go
package main

import (
	"time"

	"github.com/fufuok/cache"
)

func main() {
	// for generics
	// m := cache.NewMapOf[int]()
	m := cache.NewMap()
	m.Store("A", 1)
	m.LoadOrStore("B", 2) // 2 false
	m.LoadAndDelete("B")  // 2, true
	m.Load("A")           // 1, true
	// for generics
	// m.Load("B") // 0, false
	m.Load("B") // nil, false
}
```

## ✨ CacheOf Interface

```go
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
```

## 🔰 MapOf Interface

```go
type MapOf[V any] interface {
	// Store add item to the cache, replacing any existing items.
	Store(k string, v V)

	// Load an item from the cache.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	Load(k string) (V, bool)

	// LoadOrStore returns the existing value for the key if present.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false if stored.
	LoadOrStore(k string, v V) (V, bool)

	// LoadAndStore returns the existing value for the key if present,
	// while setting the new value for the key.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false otherwise.
	LoadAndStore(k string, v V) (V, bool)

	// LoadAndDelete Get an item from the cache, and delete the key.
	// Returns the item or nil,
	// and a boolean indicating whether the key was found.
	LoadAndDelete(k string) (V, bool)

	// Delete an item from the cache.
	// Does nothing if the key is not in the cache.
	Delete(k string)

	// Range calls f sequentially for each key and value present in the map.
	// If f returns false, range stops the iteration.
	Range(f func(k string, v V) bool)

	// Size returns the number of items in the cache.
	// This may include items that have expired but have not been cleaned up.
	Size() int
}
```

## 🛠 ConfigOf

```go
const (
	// NoExpiration mark cached item never expire.
	NoExpiration = -2 * time.Second

	// DefaultExpiration use the default expiration time set when the cache was created.
	// Equivalent to passing in the same e duration as was given to NewCache() or NewCacheDefault().
	DefaultExpiration = -1 * time.Second

	// DefaultCleanupInterval the default time interval for automatically cleaning up expired key-value pairs
	DefaultCleanupInterval = 10 * time.Second
)

// EvictedCallbackOf callback function to execute when the key-value pair expires and is evicted.
// Warning: cannot block, it is recommended to use goroutine.
type EvictedCallbackOf[V any] func(k string, v V)

type ConfigOf[V any] struct {
	// DefaultExpiration default expiration time for key-value pairs.
	DefaultExpiration time.Duration

	// CleanupInterval the interval at which expired key-value pairs are automatically cleaned up.
	CleanupInterval time.Duration

	// EvictedCallback executed when the key-value pair expires.
	EvictedCallback EvictedCallbackOf[V]
}
```

## 🤖 Benchmarks

```go
go test -run=^$ -benchtime=1s -bench=^BenchmarkCache
goos: windows
goarch: amd64
pkg: github.com/fufuok/cache
cpu: Intel(R) Core(TM) i5-10400 CPU @ 2.90GHz
BenchmarkCache_NoWarmUp/99%-reads-12            56162835                20.92 ns/op
BenchmarkCache_NoWarmUp/90%-reads-12            53462355                29.11 ns/op
BenchmarkCache_NoWarmUp/75%-reads-12            46278441                39.45 ns/op
BenchmarkCache_NoWarmUp/50%-reads-12            33421249                41.98 ns/op
BenchmarkCache_NoWarmUp/0%-reads-12             23138462                57.64 ns/op
BenchmarkCache_WarmUp/100%-reads-12             27940110                48.30 ns/op
BenchmarkCache_WarmUp/99%-reads-12              26440336                43.72 ns/op
BenchmarkCache_WarmUp/90%-reads-12              24064250                46.09 ns/op
BenchmarkCache_WarmUp/75%-reads-12              22281722                49.44 ns/op
BenchmarkCache_WarmUp/50%-reads-12              22203678                52.68 ns/op
BenchmarkCache_WarmUp/0%-reads-12               18652684                63.32 ns/op
BenchmarkCache_Range-12                               70          15780200 ns/op
```

```go
go test -run=^$ -benchtime=1s -bench=^BenchmarkMap
goos: windows
goarch: amd64
pkg: github.com/fufuok/cache
cpu: Intel(R) Core(TM) i5-10400 CPU @ 2.90GHz
BenchmarkMap_NoWarmUp/99%-reads-12              60161834                20.87 ns/op
BenchmarkMap_NoWarmUp/90%-reads-12              42707209                28.84 ns/op
BenchmarkMap_NoWarmUp/75%-reads-12              36461086                34.98 ns/op
BenchmarkMap_NoWarmUp/50%-reads-12              29346812                44.98 ns/op
BenchmarkMap_NoWarmUp/0%-reads-12               23323569                54.81 ns/op
BenchmarkMap_StandardMap_NoWarmUp/99%-reads-12          30080841               193.5 ns/op
BenchmarkMap_StandardMap_NoWarmUp/90%-reads-12           5978382               333.3 ns/op
BenchmarkMap_StandardMap_NoWarmUp/75%-reads-12           4484434               374.8 ns/op
BenchmarkMap_StandardMap_NoWarmUp/50%-reads-12           3568418               407.9 ns/op
BenchmarkMap_StandardMap_NoWarmUp/0%-reads-12            3304824               426.2 ns/op
BenchmarkMap_WarmUp/100%-reads-12                       28648095                42.65 ns/op
BenchmarkMap_WarmUp/99%-reads-12                        27972679                41.93 ns/op
BenchmarkMap_WarmUp/90%-reads-12                        25326820                44.15 ns/op
BenchmarkMap_WarmUp/75%-reads-12                        22229838                47.53 ns/op
BenchmarkMap_WarmUp/50%-reads-12                        22619359                49.83 ns/op
BenchmarkMap_WarmUp/0%-reads-12                         17453097                61.33 ns/op
BenchmarkMap_StandardMap_WarmUp/100%-reads-12           12967240                80.39 ns/op
BenchmarkMap_StandardMap_WarmUp/99%-reads-12            12137184               101.6 ns/op
BenchmarkMap_StandardMap_WarmUp/90%-reads-12             7324218               137.0 ns/op
BenchmarkMap_StandardMap_WarmUp/75%-reads-12             7573372               157.6 ns/op
BenchmarkMap_StandardMap_WarmUp/50%-reads-12             3191451               324.0 ns/op
BenchmarkMap_StandardMap_WarmUp/0%-reads-12              2642348               436.8 ns/op
BenchmarkMap_Range-12                                         88          16448449 ns/op
BenchmarkMap_RangeStandardMap-12                              79          14323286 ns/op
```





*ff*

