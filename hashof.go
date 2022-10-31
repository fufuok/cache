//go:build go1.18
// +build go1.18

package cache

import (
	"hash/maphash"

	"github.com/fufuok/cache/internal/xsync"
	"github.com/fufuok/cache/internal/xxhash"
)

// IntegerConstraint represents any integer type.
type IntegerConstraint interface{ xsync.IntegerConstraint }

// Hashable allowed map key types constraint
type Hashable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string | ~complex64 | ~complex128
}

// GenHasher64 use xxHash.
// Same as NewHashMapOf, NewHashOf hashing algorithm
func GenHasher64[K comparable]() func(K) uint64 {
	return xxhash.GenHasher64[K]()
}

func GenSeedHasher64[K comparable]() func(maphash.Seed, K) uint64 {
	return xxhash.GenSeedHasher64[K]()
}

// Hash64 calculates a hash of v with the given seed.
func Hash64[T IntegerConstraint](seed maphash.Seed, v T) uint64 {
	return xsync.Hash64(seed, v)
}
