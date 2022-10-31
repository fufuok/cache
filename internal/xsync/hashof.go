//go:build go1.18
// +build go1.18

package xsync

import (
	"hash/maphash"
)

// Hash64 calculates a hash of v with the given seed.
func Hash64[T IntegerConstraint](seed maphash.Seed, v T) uint64 {
	return hash64(seed, v)
}
