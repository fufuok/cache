# 🗃️ go-cache

Goroutine-safe, high-performance in-memory cache, optimized for reads over writes, with expiration, rich API, and support for generics.

Based on [puzpuzpuz/xsync]([puzpuzpuz/xsync: Concurrent data structures for Go (github.com)](https://github.com/puzpuzpuz/xsync)).

## ⚙️ Installation

```go
go get github.com/fufuok/cache
```

## ⚡️ Quickstart

Please see: [examples](examples)

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

## ✨ Cache Interface

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

## Config

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

// EvictedCallback callback function to execute when the key-value pair expires and is evicted.
// Warning: cannot block, it is recommended to use goroutine.
type EvictedCallback func(k string, v interface{})

type Config struct {
	// DefaultExpiration default expiration time for key-value pairs.
	DefaultExpiration time.Duration

	// CleanupInterval the interval at which expired key-value pairs are automatically cleaned up.
	CleanupInterval time.Duration

	// EvictedCallback executed when the key-value pair expires.
	EvictedCallback EvictedCallback
}
```

## 🤖 Benchmarks

```go
go test -run=^$ -benchtime=1s -bench=.
goos: linux
goarch: amd64
pkg: github.com/fufuok/cache
cpu: Intel(R) Core(TM) i5-10400 CPU @ 2.90GHz
BenchmarkCache_NoWarmUp/99%-reads-12            50074066                28.86 ns/op
BenchmarkCache_NoWarmUp/90%-reads-12            29898792                48.06 ns/op
BenchmarkCache_NoWarmUp/75%-reads-12            21478354                52.33 ns/op
BenchmarkCache_NoWarmUp/50%-reads-12            13630849                92.10 ns/op
BenchmarkCache_NoWarmUp/0%-reads-12             13473494               100.8 ns/op
BenchmarkCache_StandardMap_NoWarmUp/99%-reads-12                 3524418               379.9 ns/op
BenchmarkCache_StandardMap_NoWarmUp/90%-reads-12                 2389998               532.9 ns/op
BenchmarkCache_StandardMap_NoWarmUp/75%-reads-12                 2001315               590.6 ns/op
BenchmarkCache_StandardMap_NoWarmUp/50%-reads-12                 1825378               625.8 ns/op
BenchmarkCache_StandardMap_NoWarmUp/0%-reads-12                  1750435               671.6 ns/op
BenchmarkCache_WarmUp/100%-reads-12                             23317450                51.59 ns/op
BenchmarkCache_WarmUp/99%-reads-12                              23086156                51.77 ns/op
BenchmarkCache_WarmUp/90%-reads-12                              22214446                47.62 ns/op
BenchmarkCache_WarmUp/75%-reads-12                              25033112                46.03 ns/op
BenchmarkCache_WarmUp/50%-reads-12                              20050191                51.29 ns/op
BenchmarkCache_WarmUp/0%-reads-12                               17511268                61.26 ns/op
BenchmarkCache_StandardMap_WarmUp/100%-reads-12                  8565800               120.9 ns/op
BenchmarkCache_StandardMap_WarmUp/99%-reads-12                   2573850               428.1 ns/op
BenchmarkCache_StandardMap_WarmUp/90%-reads-12                   1740759               623.4 ns/op
BenchmarkCache_StandardMap_WarmUp/75%-reads-12                   1877916               550.3 ns/op
BenchmarkCache_StandardMap_WarmUp/50%-reads-12                   1990254               602.1 ns/op
BenchmarkCache_StandardMap_WarmUp/0%-reads-12                    1865744               703.9 ns/op
BenchmarkCache_Range-12                                               76          15280929 ns/op
BenchmarkCache_RangeStandardMap-12                                    81          13914793 ns/op
```





*ff*

