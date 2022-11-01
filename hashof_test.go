//go:build go1.18
// +build go1.18

package cache

import (
	"hash/maphash"
	"strconv"
	"testing"
	"time"
)

func TestGenHasher64(t *testing.T) {
	seed := maphash.MakeSeed()
	hasher0 := GenHasher64[string]()
	var h0 = "ff"
	v := hasher0(h0)
	if v != hasher0(h0) {
		t.Log("expect the same hash result")
	}

	seedHasher0 := GenSeedHasher64[string]()
	v = seedHasher0(seed, h0)
	if v != seedHasher0(seed, h0) {
		t.Log("expect the same hash result")
	}

	hasher1 := GenHasher64[int]()
	var h1 = 123
	v = hasher1(h1)
	if v != hasher1(h1) {
		t.Log("expect the same hash result")
	}

	seedHasher1 := GenSeedHasher64[int]()
	v = seedHasher1(seed, h1)
	if v != seedHasher1(seed, h1) {
		t.Log("expect the same hash result")
	}

	type L1 int
	type L2 L1
	hasher2 := GenHasher64[L2]()
	var h2 L2 = 123
	v2 := hasher2(h2)
	if v2 != hasher2(h2) {
		t.Log("expect the same hash result")
	}
	if v != hasher2(h2) {
		t.Log("expect the same hash result")
	}

	seedHasher2 := GenSeedHasher64[L2]()
	v = seedHasher2(seed, h2)
	if v != seedHasher2(seed, h2) {
		t.Log("expect the same hash result")
	}
	if v != seedHasher1(seed, h1) {
		t.Log("expect the same hash result")
	}

	//lint:ignore U1000 prevents false sharing
	type foo struct {
		x int
		y int
	}
	hasher3 := GenHasher64[*foo]()
	var h3 = new(foo)
	h31 := h3
	if hasher3(h3) != hasher3(h31) {
		t.Log("expect the same hash result")
	}

	seedHasher3 := GenSeedHasher64[*foo]()
	if seedHasher3(seed, h3) != seedHasher3(seed, h31) {
		t.Log("expect the same hash result")
	}

	hasher4 := GenHasher64[float64]()
	v = hasher4(3.1415926)
	if v != hasher4(3.1415926) {
		t.Log("expect the same hash result")
	}
	if v == hasher4(3.1415927) {
		t.Log("expect different hash results")
	}

	seedHasher4 := GenSeedHasher64[float64]()
	v = seedHasher4(seed, 3.1415926)
	if v != seedHasher4(seed, 3.1415926) {
		t.Log("expect the same hash result")
	}
	if v == seedHasher4(seed, 3.1415927) {
		t.Log("expect different hash results")
	}

	hasher5 := GenHasher64[complex128]()
	v = hasher5(complex(3, 5))
	if v != hasher5(complex(3, 5)) {
		t.Log("expect the same hash result")
	}
	if v == hasher5(complex(4, 5)) {
		t.Log("expect different hash results")
	}

	seedHasher5 := GenSeedHasher64[complex128]()
	v = seedHasher5(seed, complex(3, 5))
	if v != seedHasher5(seed, complex(3, 5)) {
		t.Log("expect the same hash result")
	}
	if v == seedHasher5(seed, complex(4, 5)) {
		t.Log("expect different hash results")
	}

	hasher6 := GenHasher64[byte]()
	v = hasher6('\n')
	if v != hasher6(10) {
		t.Log("expect the same hash result")
	}
	if v == hasher6('\r') {
		t.Log("expect different hash results")
	}

	seedHasher6 := GenSeedHasher64[byte]()
	v = seedHasher6(seed, '\n')
	if v != seedHasher6(seed, 10) {
		t.Log("expect the same hash result")
	}
	if v == seedHasher6(seed, '\r') {
		t.Log("expect different hash results")
	}

	hasher7 := GenHasher64[uintptr]()
	v = hasher7(10)
	if v != hasher7(10) {
		t.Log("expect the same hash result")
	}
	if v == hasher7(7) {
		t.Log("expect different hash results")
	}

	seedHasher7 := GenSeedHasher64[uintptr]()
	v = seedHasher7(seed, 10)
	if v != seedHasher7(seed, 10) {
		t.Log("expect the same hash result")
	}
	if v == seedHasher7(seed, 7) {
		t.Log("expect different hash results")
	}
}

func TestHashString(t *testing.T) {
	const numEntries = 1000
	c := NewIntegerOf[uint64, int]()
	exp := 10 * time.Second
	seed := maphash.MakeSeed()
	for i := 0; i < numEntries; i++ {
		if _, ok := c.GetOrSet(HashSeedString(seed, strconv.Itoa(i)), i, exp); ok {
			t.Fatal("value was not expected")
		}
	}
	if c.Count() != numEntries {
		t.Fatalf("expect count of 10000, but got: %d", c.Count())
	}
}

func TestHashSeedUint64(t *testing.T) {
	const numEntries = 1000
	c := NewIntegerOf[uint64, int]()
	exp := 10 * time.Second
	seed := maphash.MakeSeed()
	for i := 0; i < numEntries; i++ {
		if _, ok := c.GetOrSet(HashSeedUint64(seed, uint64(i)), i, exp); ok {
			t.Fatal("value was not expected")
		}
	}
	if c.Count() != numEntries {
		t.Fatalf("expect count of 10000, but got: %d", c.Count())
	}
}

func TestStrHash64(t *testing.T) {
	const numEntries = 1000
	c := NewIntegerOf[uint64, int]()
	exp := 10 * time.Second
	for i := 0; i < numEntries; i++ {
		if _, ok := c.GetOrSet(StrHash64(strconv.Itoa(i)), i, exp); ok {
			t.Fatal("value was not expected")
		}
	}
	if c.Count() != numEntries {
		t.Fatalf("expect count of 10000, but got: %d", c.Count())
	}
}
