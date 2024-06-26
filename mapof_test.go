//go:build go1.18
// +build go1.18

package cache

import (
	"strconv"
	"testing"
)

func TestMapOf_UniqueValuePointers_Int(t *testing.T) {
	m := NewMapOf[string, int]()
	v := 42
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMapOf_UniqueValuePointers_Struct(t *testing.T) {
	type foo struct{}
	m := NewMapOf[string, foo]()
	v := foo{}
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMapOf_UniqueValuePointers_Pointer(t *testing.T) {
	type foo struct{}
	m := NewMapOf[string, *foo]()
	v := &foo{}
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMapOf_UniqueValuePointers_Slice(t *testing.T) {
	m := NewMapOf[string, []int]()
	v := make([]int, 13)
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMapOf_UniqueValuePointers_String(t *testing.T) {
	m := NewMapOf[string, string]()
	v := "bar"
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMapOf_UniqueValuePointers_Nil(t *testing.T) {
	m := NewMapOf[string, *struct{}]()
	m.Store("foo", nil)
	m.Store("foo", nil)
}

func TestMapOf_MissingEntry(t *testing.T) {
	m := NewMapOf[string, string]()
	v, ok := m.Load("foo")
	if ok {
		t.Fatalf("value was not expected: %v", v)
	}
	if deleted, loaded := m.LoadAndDelete("foo"); loaded {
		t.Fatalf("value was not expected %v", deleted)
	}
	if actual, loaded := m.LoadOrStore("foo", "bar"); loaded {
		t.Fatalf("value was not expected %v", actual)
	}
}

func TestMapOf_EmptyStringKey(t *testing.T) {
	m := NewMapOf[string, string]()
	m.Store("", "foobar")
	v, ok := m.Load("")
	if !ok {
		t.Fatal("value was expected")
	}
	if v != "foobar" {
		t.Fatalf("value does not match: %v", v)
	}
}

func TestMapOfStore_NilValue(t *testing.T) {
	m := NewMapOf[string, *struct{}]()
	m.Store("foo", nil)
	v, ok := m.Load("foo")
	if !ok {
		t.Fatal("nil value was expected")
	}
	if v != nil {
		t.Fatalf("value was not nil: %v", v)
	}
}

func TestMapOfLoadOrStore_NilValue(t *testing.T) {
	m := NewMapOf[string, *struct{}]()
	m.LoadOrStore("foo", nil)
	v, loaded := m.LoadOrStore("foo", nil)
	if !loaded {
		t.Fatal("nil value was expected")
	}
	if v != nil {
		t.Fatalf("value was not nil: %v", v)
	}
}

func TestMapOfLoadOrStore_NonNilValue(t *testing.T) {
	type foo struct{}
	m := NewMapOf[string, *foo]()
	newv := &foo{}
	v, loaded := m.LoadOrStore("foo", newv)
	if loaded {
		t.Fatal("no value was expected")
	}
	if v != newv {
		t.Fatalf("value does not match: %v", v)
	}
	newv2 := &foo{}
	v, loaded = m.LoadOrStore("foo", newv2)
	if !loaded {
		t.Fatal("value was expected")
	}
	if v != newv {
		t.Fatalf("value does not match: %v", v)
	}
}

func TestMapOfLoadAndStore_NilValue(t *testing.T) {
	m := NewMapOf[string, *struct{}]()
	m.LoadAndStore("foo", nil)
	v, loaded := m.LoadAndStore("foo", nil)
	if !loaded {
		t.Fatal("nil value was expected")
	}
	if v != nil {
		t.Fatalf("value was not nil: %v", v)
	}
	v, loaded = m.Load("foo")
	if !loaded {
		t.Fatal("nil value was expected")
	}
	if v != nil {
		t.Fatalf("value was not nil: %v", v)
	}
}

func TestMapOfLoadAndStore_NonNilValue(t *testing.T) {
	m := NewMapOf[string, int]()
	v1 := 1
	v, loaded := m.LoadAndStore("foo", v1)
	if loaded {
		t.Fatal("no value was expected")
	}
	if v != v1 {
		t.Fatalf("value does not match: %v", v)
	}
	v2 := 2
	v, loaded = m.LoadAndStore("foo", v2)
	if !loaded {
		t.Fatal("value was expected")
	}
	if v != v1 {
		t.Fatalf("value does not match: %v", v)
	}
	v, loaded = m.Load("foo")
	if !loaded {
		t.Fatal("value was expected")
	}
	if v != v2 {
		t.Fatalf("value does not match: %v", v)
	}
}

func TestMapOfRange(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[string, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	iters := 0
	met := make(map[string]int)
	m.Range(func(key string, value int) bool {
		if key != strconv.Itoa(value) {
			t.Fatalf("got unexpected key/value for iteration %d: %v/%v", iters, key, value)
			return false
		}
		met[key] += 1
		iters++
		return true
	})
	if iters != numEntries {
		t.Fatalf("got unexpected number of iterations: %d", iters)
	}
	for i := 0; i < numEntries; i++ {
		if c := met[strconv.Itoa(i)]; c != 1 {
			t.Fatalf("range did not iterate correctly over %d: %d", i, c)
		}
	}
}

func TestMapOfRange_FalseReturned(t *testing.T) {
	m := NewMapOf[string, int]()
	for i := 0; i < 100; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	iters := 0
	m.Range(func(key string, value int) bool {
		iters++
		return iters != 13
	})
	if iters != 13 {
		t.Fatalf("got unexpected number of iterations: %d", iters)
	}
}

func TestMapOfRange_NestedDelete(t *testing.T) {
	const numEntries = 256
	m := NewMapOf[string, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	m.Range(func(key string, value int) bool {
		m.Delete(key)
		return true
	})
	for i := 0; i < numEntries; i++ {
		if _, ok := m.Load(strconv.Itoa(i)); ok {
			t.Fatalf("value found for %d", i)
		}
	}
}

func TestMapOfStore(t *testing.T) {
	const numEntries = 128
	m := NewMapOf[string, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		v, ok := m.Load(strconv.Itoa(i))
		if !ok {
			t.Fatalf("value not found for %d", i)
		}
		if v != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}

func TestIntegerMapOfStore(t *testing.T) {
	const numEntries = 128
	m := NewMapOf[int, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(i, i)
	}
	for i := 0; i < numEntries; i++ {
		v, ok := m.Load(i)
		if !ok {
			t.Fatalf("value not found for %d", i)
		}
		if v != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}

func TestTypedMapOfStore_StructKeys_IntValues(t *testing.T) {
	type foo struct {
		x int32
		y int32
	}
	const numEntries = 128
	m := NewMapOf[foo, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(foo{int32(i), -int32(i)}, i)
	}
	for i := 0; i < numEntries; i++ {
		v, ok := m.Load(foo{int32(i), -int32(i)})
		if !ok {
			t.Fatalf("value not found for %d", i)
		}
		if v != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}

func TestTypedMapOfStore_StructKeys_StructValues(t *testing.T) {
	type foo struct {
		x int32
		y int32
	}
	const numEntries = 128
	m := NewMapOf[foo, foo]()
	for i := 0; i < numEntries; i++ {
		m.Store(foo{int32(i), -int32(i)}, foo{-int32(i), int32(i)})
	}
	for i := 0; i < numEntries; i++ {
		v, ok := m.Load(foo{int32(i), -int32(i)})
		if !ok {
			t.Fatalf("value not found for %d", i)
		}
		if v.x != -int32(i) {
			t.Fatalf("x value does not match for %d: %v", i, v)
		}
		if v.y != int32(i) {
			t.Fatalf("y value does not match for %d: %v", i, v)
		}
	}
}

func TestTypedMapOfStore_HashCodeCollisions(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[int, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(i, i)
	}
	for i := 0; i < numEntries; i++ {
		v, ok := m.Load(i)
		if !ok {
			t.Fatalf("value not found for %d", i)
		}
		if v != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}

func TestMapOfLoadOrStore(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[string, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		if _, loaded := m.LoadOrStore(strconv.Itoa(i), i); !loaded {
			t.Fatalf("value not found for %d", i)
		}
	}
}

func TestMapOfLoadOrCompute(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[string, int]()
	for i := 0; i < numEntries; i++ {
		v, loaded := m.LoadOrCompute(strconv.Itoa(i), func() int {
			return i
		})
		if loaded {
			t.Fatalf("value not computed for %d", i)
		}
		if v != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
	for i := 0; i < numEntries; i++ {
		v, loaded := m.LoadOrCompute(strconv.Itoa(i), func() int {
			return i
		})
		if !loaded {
			t.Fatalf("value not loaded for %d", i)
		}
		if v != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}

func TestMapOfLoadOrCompute_FunctionCalledOnce(t *testing.T) {
	m := NewMapOf[int, int]()
	for i := 0; i < 100; {
		m.LoadOrCompute(i, func() (v int) {
			v, i = i, i+1
			return v
		})
	}

	m.Range(func(k, v int) bool {
		if k != v {
			t.Fatalf("%dth key is not equal to value %d", k, v)
		}
		return true
	})
}

func TestMapOfCompute(t *testing.T) {
	m := NewMapOf[string, int]()
	// Store a new value.
	v, ok := m.Compute("foobar", func(oldValue int, loaded bool) (newValue int, delete bool) {
		if oldValue != 0 {
			t.Fatalf("oldValue should be 0 when computing a new value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when computing a new value")
		}
		newValue = 42
		delete = false
		return
	})
	if v != 42 {
		t.Fatalf("v should be 42 when computing a new value: %d", v)
	}
	if !ok {
		t.Fatal("ok should be true when computing a new value")
	}
	// Update an existing value.
	v, ok = m.Compute("foobar", func(oldValue int, loaded bool) (newValue int, delete bool) {
		if oldValue != 42 {
			t.Fatalf("oldValue should be 42 when updating the value: %d", oldValue)
		}
		if !loaded {
			t.Fatal("loaded should be true when updating the value")
		}
		newValue = oldValue + 42
		delete = false
		return
	})
	if v != 84 {
		t.Fatalf("v should be 84 when updating the value: %d", v)
	}
	if !ok {
		t.Fatal("ok should be true when updating the value")
	}
	// Delete an existing value.
	v, ok = m.Compute("foobar", func(oldValue int, loaded bool) (newValue int, delete bool) {
		if oldValue != 84 {
			t.Fatalf("oldValue should be 84 when deleting the value: %d", oldValue)
		}
		if !loaded {
			t.Fatal("loaded should be true when deleting the value")
		}
		delete = true
		return
	})
	if v != 84 {
		t.Fatalf("v should be 84 when deleting the value: %d", v)
	}
	if ok {
		t.Fatal("ok should be false when deleting the value")
	}
	// Try to delete a non-existing value. Notice different key.
	v, ok = m.Compute("barbaz", func(oldValue int, loaded bool) (newValue int, delete bool) {
		if oldValue != 0 {
			t.Fatalf("oldValue should be 0 when trying to delete a non-existing value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when trying to delete a non-existing value")
		}
		// We're returning a non-zero value, but the map should ignore it.
		newValue = 42
		delete = true
		return
	})
	if v != 0 {
		t.Fatalf("v should be 0 when trying to delete a non-existing value: %d", v)
	}
	if ok {
		t.Fatal("ok should be false when trying to delete a non-existing value")
	}
}

func TestMapOfStoreThenDelete(t *testing.T) {
	const numEntries = 1000
	m := NewMapOfPresized[string, int](numEntries)
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		m.Delete(strconv.Itoa(i))
		if _, ok := m.Load(strconv.Itoa(i)); ok {
			t.Fatalf("value was not expected for %d", i)
		}
	}
}

func TestIntegerMapOfStoreThenDelete(t *testing.T) {
	const numEntries = 1000
	m := NewMapOfPresized[int32, int32](numEntries)
	for i := 0; i < numEntries; i++ {
		m.Store(int32(i), int32(i))
	}
	for i := 0; i < numEntries; i++ {
		m.Delete(int32(i))
		if _, ok := m.Load(int32(i)); ok {
			t.Fatalf("value was not expected for %d", i)
		}
	}
}

func TestTypedMapOfStoreThenDelete(t *testing.T) {
	type foo struct {
		x int32
		y int32
	}
	const numEntries = 1000
	m := NewMapOfPresized[foo, string](numEntries)
	for i := 0; i < numEntries; i++ {
		m.Store(foo{int32(i), 42}, strconv.Itoa(i))
	}
	for i := 0; i < numEntries; i++ {
		m.Delete(foo{int32(i), 42})
		if _, ok := m.Load(foo{int32(i), 42}); ok {
			t.Fatalf("value was not expected for %d", i)
		}
	}
}

func TestMapOfStoreThenLoadAndDelete(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[string, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		if v, loaded := m.LoadAndDelete(strconv.Itoa(i)); !loaded || v != i {
			t.Fatalf("value was not found or different for %d: %v", i, v)
		}
		if _, ok := m.Load(strconv.Itoa(i)); ok {
			t.Fatalf("value was not expected for %d", i)
		}
	}
}

func TestIntegerMapOfStoreThenLoadAndDelete(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[int, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(i, i)
	}
	for i := 0; i < numEntries; i++ {
		if _, loaded := m.LoadAndDelete(i); !loaded {
			t.Fatalf("value was not found for %d", i)
		}
		if _, ok := m.Load(i); ok {
			t.Fatalf("value was not expected for %d", i)
		}
	}
}

func TestTypedMapOfStoreThenLoadAndDelete(t *testing.T) {
	type foo struct {
		x int32
		y int32
	}
	const numEntries = 1000
	m := NewMapOf[foo, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(foo{42, int32(i)}, i)
	}
	for i := 0; i < numEntries; i++ {
		if _, loaded := m.LoadAndDelete(foo{42, int32(i)}); !loaded {
			t.Fatalf("value was not found for %d", i)
		}
		if _, ok := m.Load(foo{42, int32(i)}); ok {
			t.Fatalf("value was not expected for %d", i)
		}
	}
}

func sizeBasedOnTypedRange(m MapOf[string, int]) int {
	size := 0
	m.Range(func(key string, value int) bool {
		size++
		return true
	})
	return size
}

func TestMapOfSize(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[string, int]()
	size := m.Size()
	if size != 0 {
		t.Fatalf("zero size expected: %d", size)
	}
	expectedSize := 0
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
		expectedSize++
		size := m.Size()
		if size != expectedSize {
			t.Fatalf("size of %d was expected, got: %d", expectedSize, size)
		}
		rsize := sizeBasedOnTypedRange(m)
		if size != rsize {
			t.Fatalf("size does not match number of entries in Range: %v, %v", size, rsize)
		}
	}
	for i := 0; i < numEntries; i++ {
		m.Delete(strconv.Itoa(i))
		expectedSize--
		size := m.Size()
		if size != expectedSize {
			t.Fatalf("size of %d was expected, got: %d", expectedSize, size)
		}
		rsize := sizeBasedOnTypedRange(m)
		if size != rsize {
			t.Fatalf("size does not match number of entries in Range: %v, %v", size, rsize)
		}
	}
}

func TestMapOfClear(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[string, int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	size := m.Size()
	if size != numEntries {
		t.Fatalf("size of %d was expected, got: %d", numEntries, size)
	}
	m.Clear()
	size = m.Size()
	if size != 0 {
		t.Fatalf("zero size was expected, got: %d", size)
	}
	rsize := sizeBasedOnTypedRange(m)
	if rsize != 0 {
		t.Fatalf("zero number of entries in Range was expected, got: %d", rsize)
	}
}

func TestMapOf_StructKey_CustomHasher(t *testing.T) {
	const num = 200
	type location struct {
		lon float32
		lat float32
	}
	m := NewMapOf[location, int]()
	for i := 0; i < num; i++ {
		m.Store(location{float32(i), float32(-i)}, i)
	}
	for i := 0; i < num; i++ {
		v, ok := m.Load(location{float32(i), float32(-i)})
		if !ok {
			t.Fatalf("value not found: %d", i)
		}
		if v != i {
			t.Fatalf("values do not match, %d: %v", i, v)
		}
	}
}

func TestMapOf_StructKey_GenHasher(t *testing.T) {
	const num = 200
	type location struct {
		lon float32
		lat float32
	}
	// Warning: panic: unsupported key type xsync_test.location of kind struct
	// m := NewMapOf[location, int]()
	m := NewMapOf[*location, int]()
	keys := make([]*location, 0, num)

	for i := 0; i < num; i++ {
		key := &location{float32(i), float32(-i)}
		m.Store(key, i)
		keys = append(keys, key)
	}
	i := 0
	for _, k := range keys {
		v, ok := m.Load(k)
		if !ok {
			t.Fatalf("value not found: %d", i)
		}
		if v != i {
			t.Fatalf("values do not match, %d: %v", i, v)
		}
		i++
	}
	for i := 0; i < num; i++ {
		v, ok := m.Load(keys[i])
		if !ok {
			t.Fatalf("value not found: %d", i)
		}
		if v != i {
			t.Fatalf("values do not match, %d: %v", i, v)
		}
	}
	key := &location{float32(1), float32(-1)}
	v, ok := m.Load(key)
	if ok {
		t.Fatalf("find value, expect or not: %v", v)
	}
}
