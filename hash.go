package cache

import (
	"hash/maphash"
	"reflect"
	"unsafe"

	"github.com/fufuok/cache/internal/xsync"
)

var (
	s1          = uint64(FastRand())
	s2          = uint64(FastRand())
	maphashSeed = uintptr(s1<<32 + s2)
)

// HashSeedString calculates a hash of s with the given seed.
func HashSeedString(seed maphash.Seed, s string) uint64 {
	return xsync.HashSeedString(seed, s)
}

// HashSeedUint64 calculates a hash of v with the given seed.
func HashSeedUint64(seed maphash.Seed, v uint64) uint64 {
	return xsync.HashSeedUint64(seed, v)
}

// StrHash64 is the built-in string hash function.
// It might be handy when writing a hasher function for NewTypedMapOf.
//
// Returned hash codes are is local to a single process and cannot
// be recreated in a different process.
func StrHash64(s string) uint64 {
	if s == "" {
		return uint64(maphashSeed)
	}
	strh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	return uint64(memhash(unsafe.Pointer(strh.Data), maphashSeed, uintptr(strh.Len)))
}

//go:noescape
//go:linkname memhash runtime.memhash
func memhash(p unsafe.Pointer, h, s uintptr) uintptr

//go:noescape
//go:linkname FastRand runtime.fastrand
func FastRand() uint32

//go:linkname FastRandn runtime.fastrandn
func FastRandn(n uint32) uint32
