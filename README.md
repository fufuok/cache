# 🗃️ go-cache

Goroutine-safe, high-performance in-memory cache, optimized for reads over writes, with expiration, rich API, and support for generics.

Based on [puzpuzpuz/xsync](https://github.com/puzpuzpuz/xsync). thx.

See: [Benchmarks](#-benchmarks).

## ⚙️ Installation

```go
go get github.com/fufuok/cache
```

## ⚡️ Quickstart

[DOC.md](DOC.md), Please see: [examples](examples)

**Cache or CacheOf usage**

```go
type Cache interface{ ... }
    func New(opts ...Option) Cache
    func NewDefault(defaultExpiration, cleanupInterval time.Duration, ...) Cache
type CacheOf[K comparable, V any] interface{ ... }
    func NewHashOf[K comparable, V any](opts ...OptionOf[K, V]) CacheOf[K, V]
    func NewHashOfDefault[K comparable, V any](defaultExpiration, cleanupInterval time.Duration, ...) CacheOf[K, V]
    func NewIntegerOf[K IntegerConstraint, V any](opts ...OptionOf[K, V]) CacheOf[K, V]
    func NewIntegerOfDefault[K IntegerConstraint, V any](defaultExpiration, cleanupInterval time.Duration, ...) CacheOf[K, V]
    func NewOf[V any](opts ...OptionOf[string, V]) CacheOf[string, V]
    func NewOfDefault[V any](defaultExpiration, cleanupInterval time.Duration, ...) CacheOf[string, V]
    func NewTypedOf[K comparable, V any](hasher func(maphash.Seed, K) uint64, opts ...OptionOf[K, V]) CacheOf[K, V]
    func NewTypedOfDefault[K comparable, V any](hasher func(maphash.Seed, K) uint64, ...) CacheOf[K, V]
```

**Demo**

```go
package main

import (
	"time"

	"github.com/fufuok/cache"
)

func main() {
	// for generics
	// c := cache.NewOf[int]()
	// c := cache.NewHashOf[string, int]()
	c := cache.New()
	c.SetForever("A", 1)
	c.GetOrSet("B", 2, 1*time.Second) // 2 false
	time.Sleep(1 * time.Second)
	c.Get("A") // 1, true
	// for generics
	// c.Get("B") // 0, false
	c.Get("B") // nil, false
	c.Count()  // 1
	c.Clear()
}
```

**Map or MapOf usage (similar to sync.Map)**

```go
type Map interface{ ... }
    func NewMap() Map
type MapOf[K comparable, V any] interface{ ... }
    func NewHashMapOf[K comparable, V any](hasher ...func(maphash.Seed, K) uint64) MapOf[K, V]
    func NewIntegerMapOf[K IntegerConstraint, V any]() MapOf[K, V]
    func NewMapOf[V any]() MapOf[string, V]
    func NewTypedMapOf[K comparable, V any](hasher func(maphash.Seed, K) uint64) MapOf[K, V]
```

**Demo**

```go
package main

import (
	"github.com/fufuok/cache"
)

func main() {
	// for generics
	// m := cache.NewMapOf[int]()
	// m := cache.NewHashMapOf[string, int]()
	m := cache.NewMap()
	m.Store("A", 1)
	m.LoadOrStore("B", 2) // 2 false
	m.LoadAndDelete("B")  // 2, true
	m.Load("A")           // 1, true
	// for generics
	// m.Load("B") // 0, false
	m.Load("B") // nil, false
	m.Size()    // 1
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
}
```

## 🤖 Benchmarks

```go
go test -run=^$ -benchtime=1s -bench=^BenchmarkCache
goos: windows
goarch: amd64
pkg: github.com/fufuok/cache
cpu: Intel(R) Core(TM) i5-10400 CPU @ 2.90GHz
BenchmarkCache_NoWarmUp/99%-reads-12            63327879                19.99 ns/op
BenchmarkCache_NoWarmUp/90%-reads-12            53017690                26.44 ns/op
BenchmarkCache_NoWarmUp/75%-reads-12            42973477                33.68 ns/op
BenchmarkCache_NoWarmUp/50%-reads-12            39048008                40.00 ns/op
BenchmarkCache_NoWarmUp/0%-reads-12             28305625                51.09 ns/op
BenchmarkCache_Hash_NoWarmUp/99%-reads-12       55992489                21.45 ns/op
BenchmarkCache_Hash_NoWarmUp/90%-reads-12       42209230                27.90 ns/op
BenchmarkCache_Hash_NoWarmUp/75%-reads-12       43738472                35.28 ns/op
BenchmarkCache_Hash_NoWarmUp/50%-reads-12       26457242                43.70 ns/op
BenchmarkCache_Hash_NoWarmUp/0%-reads-12        23198097                52.73 ns/op
BenchmarkCache_Integer_NoWarmUp/99%-reads-12            119213346               13.26 ns/op
BenchmarkCache_Integer_NoWarmUp/90%-reads-12            87820084                23.77 ns/op
BenchmarkCache_Integer_NoWarmUp/75%-reads-12            42621664                28.25 ns/op
BenchmarkCache_Integer_NoWarmUp/50%-reads-12            35653345                34.12 ns/op
BenchmarkCache_Integer_NoWarmUp/0%-reads-12             27299094                43.52 ns/op
BenchmarkCache_Integer_Hash_NoWarmUp/99%-reads-12       121937115               13.40 ns/op
BenchmarkCache_Integer_Hash_NoWarmUp/90%-reads-12       55315520                22.51 ns/op
BenchmarkCache_Integer_Hash_NoWarmUp/75%-reads-12       58547437                28.28 ns/op
BenchmarkCache_Integer_Hash_NoWarmUp/50%-reads-12       33534188                34.66 ns/op
BenchmarkCache_Integer_Hash_NoWarmUp/0%-reads-12        25182330                49.54 ns/op
BenchmarkCache_WarmUp/100%-reads-12                     27032749                41.54 ns/op
BenchmarkCache_WarmUp/99%-reads-12                      28002771                42.03 ns/op
BenchmarkCache_WarmUp/90%-reads-12                      24318080                44.37 ns/op
BenchmarkCache_WarmUp/75%-reads-12                      25321423                46.43 ns/op
BenchmarkCache_WarmUp/50%-reads-12                      23165198                50.44 ns/op
BenchmarkCache_WarmUp/0%-reads-12                       20675056                60.69 ns/op
BenchmarkCache_Hash_WarmUp/100%-reads-12                27673744                42.87 ns/op
BenchmarkCache_Hash_WarmUp/99%-reads-12                 27030314                43.32 ns/op
BenchmarkCache_Hash_WarmUp/90%-reads-12                 24570778                45.62 ns/op
BenchmarkCache_Hash_WarmUp/75%-reads-12                 23616886                48.11 ns/op
BenchmarkCache_Hash_WarmUp/50%-reads-12                 21490394                54.07 ns/op
BenchmarkCache_Hash_WarmUp/0%-reads-12                  20145944                62.16 ns/op
BenchmarkCache_Integer_WarmUp/100%-reads-12             39697898                27.82 ns/op
BenchmarkCache_Integer_WarmUp/99%-reads-12              42488253                28.25 ns/op
BenchmarkCache_Integer_WarmUp/90%-reads-12              37295572                27.32 ns/op
BenchmarkCache_Integer_WarmUp/75%-reads-12              37291052                29.17 ns/op
BenchmarkCache_Integer_WarmUp/50%-reads-12              33256934                33.38 ns/op
BenchmarkCache_Integer_WarmUp/0%-reads-12               25684106                41.51 ns/op
BenchmarkCache_IntegerHash_WarmUp/100%-reads-12         42396534                27.99 ns/op
BenchmarkCache_IntegerHash_WarmUp/99%-reads-12          41481296                28.24 ns/op
BenchmarkCache_IntegerHash_WarmUp/90%-reads-12          39696848                27.51 ns/op
BenchmarkCache_IntegerHash_WarmUp/75%-reads-12          37289198                29.37 ns/op
BenchmarkCache_IntegerHash_WarmUp/50%-reads-12          34180827                33.49 ns/op
BenchmarkCache_IntegerHash_WarmUp/0%-reads-12           30007651                41.30 ns/op
BenchmarkCache_Range-12                                      118           9502503 ns/op
```

```go
go test -run=^$ -benchtime=1s -bench=^BenchmarkMap
goos: windows
goarch: amd64
pkg: github.com/fufuok/cache
cpu: Intel(R) Core(TM) i5-10400 CPU @ 2.90GHz
BenchmarkMap_NoWarmUp/99%-reads-12              63325206                19.15 ns/op
BenchmarkMap_NoWarmUp/90%-reads-12              53744052                25.50 ns/op
BenchmarkMap_NoWarmUp/75%-reads-12              37599992                30.76 ns/op
BenchmarkMap_NoWarmUp/50%-reads-12              40772086                39.04 ns/op
BenchmarkMap_NoWarmUp/0%-reads-12               29337844                49.40 ns/op
BenchmarkMap_Hash_NoWarmUp/99%-reads-12         58235559                20.64 ns/op
BenchmarkMap_Hash_NoWarmUp/90%-reads-12         52314478                27.29 ns/op
BenchmarkMap_Hash_NoWarmUp/75%-reads-12         37600227                32.09 ns/op
BenchmarkMap_Hash_NoWarmUp/50%-reads-12         31159927                40.23 ns/op
BenchmarkMap_Hash_NoWarmUp/0%-reads-12          27935626                52.54 ns/op
BenchmarkMap_StandardMap_NoWarmUp/99%-reads-12          25140050               163.9 ns/op
BenchmarkMap_StandardMap_NoWarmUp/90%-reads-12           6178712               294.3 ns/op
BenchmarkMap_StandardMap_NoWarmUp/75%-reads-12           4757120               348.8 ns/op
BenchmarkMap_StandardMap_NoWarmUp/50%-reads-12           4043784               390.8 ns/op
BenchmarkMap_StandardMap_NoWarmUp/0%-reads-12            3417021               401.5 ns/op
BenchmarkMap_Integer_NoWarmUp/99%-reads-12              124007650               12.70 ns/op
BenchmarkMap_Integer_NoWarmUp/90%-reads-12              56774936                20.68 ns/op
BenchmarkMap_Integer_NoWarmUp/75%-reads-12              45362900                26.98 ns/op
BenchmarkMap_Integer_NoWarmUp/50%-reads-12              34489407                34.47 ns/op
BenchmarkMap_Integer_NoWarmUp/0%-reads-12               27501502                43.51 ns/op
BenchmarkMap_Integer_Hash_NoWarmUp/99%-reads-12         100000000               12.40 ns/op
BenchmarkMap_Integer_Hash_NoWarmUp/90%-reads-12         81966092                22.46 ns/op
BenchmarkMap_Integer_Hash_NoWarmUp/75%-reads-12         45658306                25.57 ns/op
BenchmarkMap_Integer_Hash_NoWarmUp/50%-reads-12         38508940                31.35 ns/op
BenchmarkMap_Integer_Hash_NoWarmUp/0%-reads-12          29092020                42.06 ns/op
BenchmarkMap_WarmUp/100%-reads-12                       29704293                39.97 ns/op
BenchmarkMap_WarmUp/99%-reads-12                        29366491                40.35 ns/op
BenchmarkMap_WarmUp/90%-reads-12                        27990949                42.13 ns/op
BenchmarkMap_WarmUp/75%-reads-12                        25609778                44.32 ns/op
BenchmarkMap_WarmUp/50%-reads-12                        24317341                48.40 ns/op
BenchmarkMap_WarmUp/0%-reads-12                         20894202                56.75 ns/op
BenchmarkMap_Hash_WarmUp/100%-reads-12                  28667463                42.39 ns/op
BenchmarkMap_Hash_WarmUp/99%-reads-12                   27655759                44.14 ns/op
BenchmarkMap_Hash_WarmUp/90%-reads-12                   27361714                44.41 ns/op
BenchmarkMap_Hash_WarmUp/75%-reads-12                   26149032                45.68 ns/op
BenchmarkMap_Hash_WarmUp/50%-reads-12                   24309853                50.35 ns/op
BenchmarkMap_Hash_WarmUp/0%-reads-12                    20230082                62.05 ns/op
BenchmarkMap_StandardMap_WarmUp/100%-reads-12           10964797                93.34 ns/op
BenchmarkMap_StandardMap_WarmUp/99%-reads-12            10617907               106.0 ns/op
BenchmarkMap_StandardMap_WarmUp/90%-reads-12             9165158               134.9 ns/op
BenchmarkMap_StandardMap_WarmUp/75%-reads-12             6948925               144.5 ns/op
BenchmarkMap_StandardMap_WarmUp/50%-reads-12             4192543               249.5 ns/op
BenchmarkMap_StandardMap_WarmUp/0%-reads-12              2891802               466.0 ns/op
BenchmarkMap_Integer_WarmUp/100%-reads-12               400912208                3.060 ns/op
BenchmarkMap_Integer_WarmUp/99%-reads-12                100000000               11.58 ns/op
BenchmarkMap_Integer_WarmUp/90%-reads-12                57717574                21.07 ns/op
BenchmarkMap_Integer_WarmUp/75%-reads-12                55887255                26.45 ns/op
BenchmarkMap_Integer_WarmUp/50%-reads-12                37842610                32.06 ns/op
BenchmarkMap_Integer_WarmUp/0%-reads-12                 29015722                41.02 ns/op
BenchmarkMap_Integer_Hash_WarmUp/100%-reads-12          399228955                3.112 ns/op
BenchmarkMap_Integer_Hash_WarmUp/99%-reads-12           132505498               12.78 ns/op
BenchmarkMap_Integer_Hash_WarmUp/90%-reads-12           91849152                22.55 ns/op
BenchmarkMap_Integer_Hash_WarmUp/75%-reads-12           55890379                26.38 ns/op
BenchmarkMap_Integer_Hash_WarmUp/50%-reads-12           31525850                32.60 ns/op
BenchmarkMap_Integer_Hash_WarmUp/0%-reads-12            29073272                41.06 ns/op
BenchmarkMap_Range-12                                        158           9017278 ns/op
BenchmarkMap_RangeStandardMap-12                              80          19778909 ns/op
```







*ff*

