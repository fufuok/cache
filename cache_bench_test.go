package cache

import (
	"strconv"
	"testing"
	_ "unsafe"
)

// Ref: https://github.com/puzpuzpuz/xsync/blob/main/map_test.go
const (
	// number of entries to use in benchmarks
	benchmarkNumEntries = 1_000_000
	// key prefix used in benchmarks
	benchmarkKeyPrefix = "what_a_looooooooooooooooooooooong_key_prefix_"
)

var benchmarkCases = []struct {
	name           string
	readPercentage int
}{
	{"100%-reads", 100}, // 100% loads,    0% stores,    0% deletes
	{"99%-reads", 99},   //  99% loads,  0.5% stores,  0.5% deletes
	{"90%-reads", 90},   //  90% loads,    5% stores,    5% deletes
	{"75%-reads", 75},   //  75% loads, 12.5% stores, 12.5% deletes
	{"50%-reads", 50},   //  50% loads,   25% stores,   25% deletes
	{"0%-reads", 0},     //   0% loads,   50% stores,   50% deletes
}

var (
	benchmarkKeys        []string
	benchmarkIntegerKeys []int
)

func init() {
	benchmarkKeys = make([]string, benchmarkNumEntries)
	benchmarkIntegerKeys = make([]int, benchmarkNumEntries)
	for i := 0; i < benchmarkNumEntries; i++ {
		benchmarkKeys[i] = benchmarkKeyPrefix + strconv.Itoa(i)
		benchmarkIntegerKeys[i] = i
	}
}

func BenchmarkCache_NoWarmUp(b *testing.B) {
	for _, bc := range benchmarkCases {
		if bc.readPercentage == 100 {
			// This benchmark doesn't make sense without a warm-up.
			continue
		}
		b.Run(bc.name, func(b *testing.B) {
			m := New[string, int](WithMinCapacity[string, int](benchmarkNumEntries))
			benchmarkCache(b, func(k string) (int, bool) {
				return m.Get(k)
			}, func(k string, v int) {
				m.SetForever(k, v)
			}, func(k string) {
				m.Delete(k)
			}, bc.readPercentage)
		})
	}
}

func BenchmarkCache_Integer_NoWarmUp(b *testing.B) {
	for _, bc := range benchmarkCases {
		if bc.readPercentage == 100 {
			// This benchmark doesn't make sense without a warm-up.
			continue
		}
		b.Run(bc.name, func(b *testing.B) {
			m := New[int, int](WithMinCapacity[int, int](benchmarkNumEntries))
			benchmarkIntegerCache(b, func(k int) (int, bool) {
				return m.Get(k)
			}, func(k int, v int) {
				m.SetForever(k, v)
			}, func(k int) {
				m.Delete(k)
			}, bc.readPercentage)
		})
	}
}

func BenchmarkCache_WarmUp(b *testing.B) {
	for _, bc := range benchmarkCases {
		b.Run(bc.name, func(b *testing.B) {
			m := New[string, int]()
			for i := 0; i < benchmarkNumEntries; i++ {
				m.SetForever(benchmarkKeyPrefix+strconv.Itoa(i), i)
			}
			benchmarkCache(b, func(k string) (int, bool) {
				return m.Get(k)
			}, func(k string, v int) {
				m.SetForever(k, v)
			}, func(k string) {
				m.Delete(k)
			}, bc.readPercentage)
		})
	}
}

func BenchmarkCache_Integer_WarmUp(b *testing.B) {
	for _, bc := range benchmarkCases {
		b.Run(bc.name, func(b *testing.B) {
			m := New[int, int]()
			for i := 0; i < benchmarkNumEntries; i++ {
				m.SetForever(i, i)
			}
			benchmarkIntegerCache(b, func(k int) (int, bool) {
				return m.Get(k)
			}, func(k int, v int) {
				m.SetForever(k, v)
			}, func(k int) {
				m.Delete(k)
			}, bc.readPercentage)
		})
	}
}

func benchmarkCache(
	b *testing.B,
	loadFn func(k string) (int, bool),
	storeFn func(k string, v int),
	deleteFn func(k string),
	readPercentage int,
) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// convert percent to permille to support 99% case
		storeThreshold := 10 * readPercentage
		deleteThreshold := 10*readPercentage + ((1000 - 10*readPercentage) / 2)
		for pb.Next() {
			op := int(runtimeFastrand() % 1000)
			i := int(runtimeFastrand() % benchmarkNumEntries)
			if op >= deleteThreshold {
				deleteFn(benchmarkKeys[i])
			} else if op >= storeThreshold {
				storeFn(benchmarkKeys[i], i)
			} else {
				loadFn(benchmarkKeys[i])
			}
		}
	})
}

func benchmarkIntegerCache(
	b *testing.B,
	loadFn func(k int) (int, bool),
	storeFn func(k int, v int),
	deleteFn func(k int),
	readPercentage int,
) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// convert percent to permille to support 99% case
		storeThreshold := 10 * readPercentage
		deleteThreshold := 10*readPercentage + ((1000 - 10*readPercentage) / 2)
		for pb.Next() {
			op := int(runtimeFastrand() % 1000)
			i := int(runtimeFastrand() % benchmarkNumEntries)
			if op >= deleteThreshold {
				deleteFn(benchmarkIntegerKeys[i])
			} else if op >= storeThreshold {
				storeFn(benchmarkIntegerKeys[i], i)
			} else {
				loadFn(benchmarkIntegerKeys[i])
			}
		}
	})
}

func BenchmarkCache_Range(b *testing.B) {
	m := New[string, int]()
	for i := 0; i < benchmarkNumEntries; i++ {
		m.SetForever(benchmarkKeys[i], i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		foo := 0
		for pb.Next() {
			m.Range(func(key string, value int) bool {
				foo++
				return true
			})
			_ = foo
		}
	})
}

//go:noescape
//go:linkname runtimeFastrand runtime.fastrand
func runtimeFastrand() uint32
