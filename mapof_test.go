//go:build go1.18
// +build go1.18

package cache

import (
	"strconv"
	"testing"
)

func TestMapOf_UniqueValuePointers_Int(t *testing.T) {
	m := NewMapOf[int]()
	v := 42
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMapOf_UniqueValuePointers_Struct(t *testing.T) {
	type foo struct{}
	m := NewMapOf[foo]()
	v := foo{}
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMapOf_UniqueValuePointers_Pointer(t *testing.T) {
	type foo struct{}
	m := NewMapOf[*foo]()
	v := &foo{}
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMapOf_UniqueValuePointers_Slice(t *testing.T) {
	m := NewMapOf[[]int]()
	v := make([]int, 13)
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMapOf_UniqueValuePointers_String(t *testing.T) {
	m := NewMapOf[string]()
	v := "bar"
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMapOf_UniqueValuePointers_Nil(t *testing.T) {
	m := NewMapOf[*struct{}]()
	m.Store("foo", nil)
	m.Store("foo", nil)
}

func TestMapOf_MissingEntry(t *testing.T) {
	m := NewMapOf[string]()
	v, ok := m.Load("foo")
	if ok {
		t.Errorf("value was not expected: %v", v)
	}
	if deleted, loaded := m.LoadAndDelete("foo"); loaded {
		t.Errorf("value was not expected %v", deleted)
	}
	if actual, loaded := m.LoadOrStore("foo", "bar"); loaded {
		t.Errorf("value was not expected %v", actual)
	}
}

func TestMapOf_EmptyStringKey(t *testing.T) {
	m := NewMapOf[string]()
	m.Store("", "foobar")
	v, ok := m.Load("")
	if !ok {
		t.Error("value was expected")
	}
	if v != "foobar" {
		t.Errorf("value does not match: %v", v)
	}
}

func TestMapOfStore_NilValue(t *testing.T) {
	m := NewMapOf[*struct{}]()
	m.Store("foo", nil)
	v, ok := m.Load("foo")
	if !ok {
		t.Error("nil value was expected")
	}
	if v != nil {
		t.Errorf("value was not nil: %v", v)
	}
}

func TestMapOfLoadOrStore_NilValue(t *testing.T) {
	m := NewMapOf[*struct{}]()
	m.LoadOrStore("foo", nil)
	v, loaded := m.LoadOrStore("foo", nil)
	if !loaded {
		t.Error("nil value was expected")
	}
	if v != nil {
		t.Errorf("value was not nil: %v", v)
	}
}

func TestMapOfLoadOrStore_NonNilValue(t *testing.T) {
	type foo struct{}
	m := NewMapOf[*foo]()
	newv := &foo{}
	v, loaded := m.LoadOrStore("foo", newv)
	if loaded {
		t.Error("no value was expected")
	}
	if v != newv {
		t.Errorf("value does not match: %v", v)
	}
}

func TestMapOfLoadAndStore_NilValue(t *testing.T) {
	m := NewMapOf[*struct{}]()
	m.LoadAndStore("foo", nil)
	v, loaded := m.LoadAndStore("foo", nil)
	if !loaded {
		t.Error("nil value was expected")
	}
	if v != nil {
		t.Errorf("value was not nil: %v", v)
	}
	v, loaded = m.Load("foo")
	if !loaded {
		t.Error("nil value was expected")
	}
	if v != nil {
		t.Errorf("value was not nil: %v", v)
	}
}

func TestMapOfLoadAndStore_NonNilValue(t *testing.T) {
	m := NewMapOf[int]()
	v1 := 1
	v, loaded := m.LoadAndStore("foo", v1)
	if loaded {
		t.Error("no value was expected")
	}
	if v != v1 {
		t.Errorf("value does not match: %v", v)
	}
	v2 := 2
	v, loaded = m.LoadAndStore("foo", v2)
	if !loaded {
		t.Error("value was expected")
	}
	if v != v1 {
		t.Errorf("value does not match: %v", v)
	}
	v, loaded = m.Load("foo")
	if !loaded {
		t.Error("value was expected")
	}
	if v != v2 {
		t.Errorf("value does not match: %v", v)
	}
}

func TestMapOfRange(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	iters := 0
	met := make(map[string]int)
	m.Range(func(key string, value int) bool {
		if key != strconv.Itoa(value) {
			t.Errorf("got unexpected key/value for iteration %d: %v/%v", iters, key, value)
			return false
		}
		met[key] += 1
		iters++
		return true
	})
	if iters != numEntries {
		t.Errorf("got unexpected number of iterations: %d", iters)
	}
	for i := 0; i < numEntries; i++ {
		if c := met[strconv.Itoa(i)]; c != 1 {
			t.Errorf("range did not iterate correctly over %d: %d", i, c)
		}
	}
}

func TestMapOfRange_FalseReturned(t *testing.T) {
	m := NewMapOf[int]()
	for i := 0; i < 100; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	iters := 0
	m.Range(func(key string, value int) bool {
		iters++
		return iters != 13
	})
	if iters != 13 {
		t.Errorf("got unexpected number of iterations: %d", iters)
	}
}

func TestMapOfRange_NestedDelete(t *testing.T) {
	const numEntries = 256
	m := NewMapOf[int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	m.Range(func(key string, value int) bool {
		m.Delete(key)
		return true
	})
	for i := 0; i < numEntries; i++ {
		if _, ok := m.Load(strconv.Itoa(i)); ok {
			t.Errorf("value found for %d", i)
		}
	}
}

func TestMapOfSerialStore(t *testing.T) {
	const numEntries = 128
	m := NewMapOf[int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		v, ok := m.Load(strconv.Itoa(i))
		if !ok {
			t.Errorf("value not found for %d", i)
		}
		if v != i {
			t.Errorf("values do not match for %d: %v", i, v)
		}
	}
}

func TestMapOfSerialLoadOrStore(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		if _, loaded := m.LoadOrStore(strconv.Itoa(i), i); !loaded {
			t.Errorf("value not found for %d", i)
		}
	}
}

func TestMapOfSerialStoreThenDelete(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		m.Delete(strconv.Itoa(i))
		if _, ok := m.Load(strconv.Itoa(i)); ok {
			t.Errorf("value was not expected for %d", i)
		}
	}
}

func TestMapOfSerialStoreThenLoadAndDelete(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[int]()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		if _, loaded := m.LoadAndDelete(strconv.Itoa(i)); !loaded {
			t.Errorf("value was not found for %d", i)
		}
		if _, ok := m.Load(strconv.Itoa(i)); ok {
			t.Errorf("value was not expected for %d", i)
		}
	}
}

func TestMapOfSize(t *testing.T) {
	const numEntries = 1000
	m := NewMapOf[int]()
	size := m.Size()
	if size != 0 {
		t.Errorf("zero size expected: %d", size)
	}
	expectedSize := 0
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
		expectedSize++
		size := m.Size()
		if size != expectedSize {
			t.Errorf("size of %d was expected, got: %d", expectedSize, size)
		}
	}
	for i := 0; i < numEntries; i++ {
		m.Delete(strconv.Itoa(i))
		expectedSize--
		size := m.Size()
		if size != expectedSize {
			t.Errorf("size of %d was expected, got: %d", expectedSize, size)
		}
	}
}
