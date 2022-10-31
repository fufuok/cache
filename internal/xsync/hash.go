package xsync

import (
	"hash/maphash"
)

// HashString calculates a hash of s with the given seed.
func HashString(seed maphash.Seed, s string) uint64 {
	return hashString(seed, s)
}
