# 🗃️ go-cache

Goroutine-safe, high-performance in-memory cache, optimized for reads over writes, with expiration, rich API, and support for generics.

Based on [puzpuzpuz/xsync](https://github.com/puzpuzpuz/xsync). thx.

See: [Benchmarks](#-benchmarks).

## ⚙️ Installation

```go
go get -u github.com/fufuok/cache
```

## ⚡️ Quickstart

[DOC.md](DOC.md), Please see: [examples](examples)

**Cache or CacheOf usage**

```go
type Cache interface{ ... }
    func New(opts ...Option) Cache
    func NewDefault(defaultExpiration, cleanupInterval time.Duration, ...) Cache
type CacheOf[K comparable, V any] interface{ ... }
    func NewOf[K comparable, V any](opts ...OptionOf[K, V]) CacheOf[K, V]
    func NewOfDefault[K comparable, V any](defaultExpiration, cleanupInterval time.Duration, ...) CacheOf[K, V]

type Option func(config *Config)
    func WithCleanupInterval(interval time.Duration) Option
    func WithDefaultExpiration(duration time.Duration) Option
    func WithEvictedCallback(ec EvictedCallback) Option
    func WithMinCapacity(sizeHint int) Option
type OptionOf[K comparable, V any] func(config *ConfigOf[K, V])
    func WithCleanupIntervalOf[K comparable, V any](interval time.Duration) OptionOf[K, V]
    func WithDefaultExpirationOf[K comparable, V any](duration time.Duration) OptionOf[K, V]
    func WithEvictedCallbackOf[K comparable, V any](ec EvictedCallbackOf[K, V]) OptionOf[K, V]
    func WithMinCapacityOf[K comparable, V any](sizeHint int) OptionOf[K, V]
```

**Demo**

```go
package main

import (
	"fmt"
	"time"

	"github.com/fufuok/cache"
)

func main() {
	// for generics
	// c := cache.NewOf[string, int]()
	c := cache.New()
	c.SetForever("A", 1)
	fmt.Println(c.GetOrSet("B", 2, 1*time.Second)) // 2 false
	time.Sleep(1 * time.Second)
	fmt.Println(c.Get("A")) // 1, true
	// for generics
	// fmt.Println(c.Get("B")) // 0, false
	fmt.Println(c.Get("B")) // nil, false
	fmt.Println(c.Count())  // 1
	c.Clear()
}
```

**Map or MapOf usage (similar to sync.Map)**

```go
type Map interface{ ... }
    func NewMap() Map
    func NewMapPresized(sizeHint int) Map
type MapOf[K comparable, V any] interface{ ... }
    func NewMapOf[K comparable, V any]() MapOf[K, V]
    func NewMapOfPresized[K comparable, V any](sizeHint int) MapOf[K, V]
```

**Demo**

```go
package main

import (
	"fmt"

	"github.com/fufuok/cache"
)

func main() {
	// for generics
	// m := cache.NewMapOf[string, int]()
	m := cache.NewMap()
	m.Store("A", 1)
	fmt.Println(m.LoadOrStore("B", 2)) // 2 false
	fmt.Println(m.LoadAndDelete("B"))  // 2, true
	fmt.Println(m.Load("A"))           // 1, true
	// for generics
	// fmt.Println(m.Load("B")) // 0, false
	fmt.Println(m.Load("B")) // nil, false
	fmt.Println(m.Size())    // 1
	m.Clear()
}
```

## ✨ CacheOf Interface

```go
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
```

## 🔰 MapOf Interface

```go
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

	// DefaultMinCapacity specify the initial cache capacity (minimum capacity)
	DefaultMinCapacity = 32 * 3
)

// EvictedCallbackOf callback function to execute when the key-value pair expires and is evicted.
// Warning: cannot block, it is recommended to use goroutine.
type EvictedCallbackOf[K comparable, V any] func(k K, v V)

type ConfigOf[K comparable, V any] struct {
	// DefaultExpiration default expiration time for key-value pairs.
	DefaultExpiration time.Duration

	// CleanupInterval the interval at which expired key-value pairs are automatically cleaned up.
	CleanupInterval time.Duration

	// EvictedCallback executed when the key-value pair expires.
	EvictedCallback EvictedCallbackOf[K, V]

	// MinCapacity specify the initial cache capacity (minimum capacity)
	MinCapacity int
}
```

## 🤖 Benchmarks

- Number of entries used in benchmark: `1_000_000`

- ```go
  {"100%-reads", 100}, // 100% loads,    0% stores,    0% deletes
  {"99%-reads", 99},   //  99% loads,  0.5% stores,  0.5% deletes
  {"90%-reads", 90},   //  90% loads,    5% stores,    5% deletes
  {"75%-reads", 75},   //  75% loads, 12.5% stores, 12.5% deletes
  {"50%-reads", 50},   //  50% loads,   25% stores,   25% deletes
  {"0%-reads", 0},     //   0% loads,   50% stores,   50% deletes
  ```

```go
# go test -run=^$ -benchtime=1s -bench=^BenchmarkCache
goos: linux
goarch: amd64
pkg: github.com/fufuok/cache
cpu: AMD Ryzen 7 5700G with Radeon Graphics
BenchmarkCache_NoWarmUp/99%-reads-16    63965202                17.95 ns/op
BenchmarkCache_NoWarmUp/90%-reads-16    67328943                24.10 ns/op
BenchmarkCache_NoWarmUp/75%-reads-16    58756623                23.86 ns/op
BenchmarkCache_NoWarmUp/50%-reads-16    56851326                27.52 ns/op
BenchmarkCache_NoWarmUp/0%-reads-16     48408231                30.44 ns/op
BenchmarkCache_Integer_NoWarmUp/99%-reads-16            120131253               10.23 ns/op
BenchmarkCache_Integer_NoWarmUp/90%-reads-16            100000000               13.78 ns/op
BenchmarkCache_Integer_NoWarmUp/75%-reads-16            100000000               15.84 ns/op
BenchmarkCache_Integer_NoWarmUp/50%-reads-16            100000000               17.66 ns/op
BenchmarkCache_Integer_NoWarmUp/0%-reads-16             77839507                20.96 ns/op
BenchmarkCache_WarmUp/100%-reads-16                     32617021                38.83 ns/op
BenchmarkCache_WarmUp/99%-reads-16                      34722482                34.13 ns/op
BenchmarkCache_WarmUp/90%-reads-16                      32676339                32.14 ns/op
BenchmarkCache_WarmUp/75%-reads-16                      41837350                28.35 ns/op
BenchmarkCache_WarmUp/50%-reads-16                      40624537                31.01 ns/op
BenchmarkCache_WarmUp/0%-reads-16                       31336846                32.22 ns/op
BenchmarkCache_Integer_WarmUp/100%-reads-16             67382474                18.11 ns/op
BenchmarkCache_Integer_WarmUp/99%-reads-16              69214749                18.84 ns/op
BenchmarkCache_Integer_WarmUp/90%-reads-16              71938634                16.10 ns/op
BenchmarkCache_Integer_WarmUp/75%-reads-16              61299493                16.38 ns/op
BenchmarkCache_Integer_WarmUp/50%-reads-16              58511590                18.37 ns/op
BenchmarkCache_Integer_WarmUp/0%-reads-16               49336832                20.99 ns/op
BenchmarkCache_Range-16                                      318           4516644 ns/op
```

```go
# go test -run=^$ -benchtime=1s -bench=^BenchmarkMap
goos: linux
goarch: amd64
pkg: github.com/fufuok/cache
cpu: AMD Ryzen 7 5700G with Radeon Graphics
BenchmarkMap_NoWarmUp/99%-reads-16              72496048                18.52 ns/op
BenchmarkMap_NoWarmUp/90%-reads-16              63024735                21.56 ns/op
BenchmarkMap_NoWarmUp/75%-reads-16              59399750                25.12 ns/op
BenchmarkMap_NoWarmUp/50%-reads-16              51399138                24.23 ns/op
BenchmarkMap_NoWarmUp/0%-reads-16               51073983                28.00 ns/op
BenchmarkMap_Integer_NoWarmUp/99%-reads-16              148085829                8.401 ns/op
BenchmarkMap_Integer_NoWarmUp/90%-reads-16              100000000               12.25 ns/op
BenchmarkMap_Integer_NoWarmUp/75%-reads-16              100000000               13.06 ns/op
BenchmarkMap_Integer_NoWarmUp/50%-reads-16              100000000               14.64 ns/op
BenchmarkMap_Integer_NoWarmUp/0%-reads-16               81655238                16.92 ns/op
BenchmarkMap_StandardMap_NoWarmUp/99%-reads-16           5306242               319.2 ns/op
BenchmarkMap_StandardMap_NoWarmUp/90%-reads-16           2906460               461.2 ns/op
BenchmarkMap_StandardMap_NoWarmUp/75%-reads-16           2386760               504.1 ns/op
BenchmarkMap_StandardMap_NoWarmUp/50%-reads-16           2469435               515.5 ns/op
BenchmarkMap_StandardMap_NoWarmUp/0%-reads-16            1997181               575.5 ns/op
BenchmarkMap_WarmUp/100%-reads-16                       37755207                31.90 ns/op
BenchmarkMap_WarmUp/99%-reads-16                        36679430                31.53 ns/op
BenchmarkMap_WarmUp/90%-reads-16                        45285555                26.23 ns/op
BenchmarkMap_WarmUp/75%-reads-16                        45376821                25.98 ns/op
BenchmarkMap_WarmUp/50%-reads-16                        35553566                28.31 ns/op
BenchmarkMap_WarmUp/0%-reads-16                         33345013                34.30 ns/op
BenchmarkMap_Integer_WarmUp/100%-reads-16               420121954                2.787 ns/op
BenchmarkMap_Integer_WarmUp/99%-reads-16                161534372                7.975 ns/op
BenchmarkMap_Integer_WarmUp/90%-reads-16                100000000               14.23 ns/op
BenchmarkMap_Integer_WarmUp/75%-reads-16                80478811                15.98 ns/op
BenchmarkMap_Integer_WarmUp/50%-reads-16                67574710                17.81 ns/op
BenchmarkMap_Integer_WarmUp/0%-reads-16                 57477188                22.04 ns/op
BenchmarkMap_StandardMap_WarmUp/100%-reads-16           13763970                77.03 ns/op
BenchmarkMap_StandardMap_WarmUp/99%-reads-16             4539453               242.3 ns/op
BenchmarkMap_StandardMap_WarmUp/90%-reads-16             4369597               240.1 ns/op
BenchmarkMap_StandardMap_WarmUp/75%-reads-16             2665832               380.1 ns/op
BenchmarkMap_StandardMap_WarmUp/50%-reads-16             2142928               500.8 ns/op
BenchmarkMap_StandardMap_WarmUp/0%-reads-16              2104003               599.6 ns/op
BenchmarkMap_Range-16                                        392           3183282 ns/op
BenchmarkMap_RangeStandardMap-16                             188           6469273 ns/op
```







*ff*

