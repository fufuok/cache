package cache

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestMap_UniqueValuePointers_Int(t *testing.T) {
	m := NewMap()
	v := 42
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMap_UniqueValuePointers_Struct(t *testing.T) {
	type foo struct{}
	m := NewMap()
	v := foo{}
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMap_UniqueValuePointers_Pointer(t *testing.T) {
	type foo struct{}
	m := NewMap()
	v := &foo{}
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMap_UniqueValuePointers_Slice(t *testing.T) {
	m := NewMap()
	v := make([]int, 13)
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMap_UniqueValuePointers_String(t *testing.T) {
	m := NewMap()
	v := "bar"
	m.Store("foo", v)
	m.Store("foo", v)
}

func TestMap_UniqueValuePointers_Nil(t *testing.T) {
	m := NewMap()
	m.Store("foo", nil)
	m.Store("foo", nil)
}
func TestMap_MissingEntry(t *testing.T) {
	m := NewMap()
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

func TestMap_EmptyStringKey(t *testing.T) {
	m := NewMap()
	m.Store("", "foobar")
	v, ok := m.Load("")
	if !ok {
		t.Error("value was expected")
	}
	if vs, ok := v.(string); ok && vs != "foobar" {
		t.Fatalf("value does not match: %v", v)
	}
}

func TestMapStore_NilValue(t *testing.T) {
	m := NewMap()
	m.Store("foo", nil)
	v, ok := m.Load("foo")
	if !ok {
		t.Error("nil value was expected")
	}
	if v != nil {
		t.Fatalf("value was not nil: %v", v)
	}
}

func TestMapLoadOrStore_NilValue(t *testing.T) {
	m := NewMap()
	m.LoadOrStore("foo", nil)
	v, loaded := m.LoadOrStore("foo", nil)
	if !loaded {
		t.Error("nil value was expected")
	}
	if v != nil {
		t.Fatalf("value was not nil: %v", v)
	}
}

func TestMapLoadOrStore_NonNilValue(t *testing.T) {
	type foo struct{}
	m := NewMap()
	newv := &foo{}
	v, loaded := m.LoadOrStore("foo", newv)
	if loaded {
		t.Error("no value was expected")
	}
	if v != newv {
		t.Fatalf("value does not match: %v", v)
	}
	newv2 := &foo{}
	v, loaded = m.LoadOrStore("foo", newv2)
	if !loaded {
		t.Error("value was expected")
	}
	if v != newv {
		t.Fatalf("value does not match: %v", v)
	}
}

func TestMapLoadAndStore_NilValue(t *testing.T) {
	m := NewMap()
	m.LoadAndStore("foo", nil)
	v, loaded := m.LoadAndStore("foo", nil)
	if !loaded {
		t.Error("nil value was expected")
	}
	if v != nil {
		t.Fatalf("value was not nil: %v", v)
	}
	v, loaded = m.Load("foo")
	if !loaded {
		t.Error("nil value was expected")
	}
	if v != nil {
		t.Fatalf("value was not nil: %v", v)
	}
}

func TestMapLoadAndStore_NonNilValue(t *testing.T) {
	type foo struct{}
	m := NewMap()
	v1 := &foo{}
	v, loaded := m.LoadAndStore("foo", v1)
	if loaded {
		t.Error("no value was expected")
	}
	if v != v1 {
		t.Fatalf("value does not match: %v", v)
	}
	v2 := 2
	v, loaded = m.LoadAndStore("foo", v2)
	if !loaded {
		t.Error("value was expected")
	}
	if v != v1 {
		t.Fatalf("value does not match: %v", v)
	}
	v, loaded = m.Load("foo")
	if !loaded {
		t.Error("value was expected")
	}
	if v != v2 {
		t.Fatalf("value does not match: %v", v)
	}
}

func TestMapRange(t *testing.T) {
	const numEntries = 1000
	m := NewMap()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	iters := 0
	met := make(map[string]int)
	m.Range(func(key string, value interface{}) bool {
		if key != strconv.Itoa(value.(int)) {
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

func TestMapRange_FalseReturned(t *testing.T) {
	m := NewMap()
	for i := 0; i < 100; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	iters := 0
	m.Range(func(key string, value interface{}) bool {
		iters++
		return iters != 13
	})
	if iters != 13 {
		t.Fatalf("got unexpected number of iterations: %d", iters)
	}
}

func TestMapRange_NestedDelete(t *testing.T) {
	const numEntries = 256
	m := NewMap()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	m.Range(func(key string, value interface{}) bool {
		m.Delete(key)
		return true
	})
	for i := 0; i < numEntries; i++ {
		if _, ok := m.Load(strconv.Itoa(i)); ok {
			t.Fatalf("value found for %d", i)
		}
	}
}

func TestMapStore(t *testing.T) {
	const numEntries = 128
	m := NewMap()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		v, ok := m.Load(strconv.Itoa(i))
		if !ok {
			t.Fatalf("value not found for %d", i)
		}
		if vi, ok := v.(int); ok && vi != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}

func TestMapLoadOrStore(t *testing.T) {
	const numEntries = 1000
	m := NewMap()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		if _, loaded := m.LoadOrStore(strconv.Itoa(i), i); !loaded {
			t.Fatalf("value not found for %d", i)
		}
	}
}

func TestMapLoadOrCompute(t *testing.T) {
	const numEntries = 1000
	m := NewMap()
	for i := 0; i < numEntries; i++ {
		v, loaded := m.LoadOrCompute(strconv.Itoa(i), func() interface{} {
			return i
		})
		if loaded {
			t.Fatalf("value not computed for %d", i)
		}
		if vi, ok := v.(int); ok && vi != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
	for i := 0; i < numEntries; i++ {
		v, loaded := m.LoadOrCompute(strconv.Itoa(i), func() interface{} {
			return i
		})
		if !loaded {
			t.Fatalf("value not loaded for %d", i)
		}
		if vi, ok := v.(int); ok && vi != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}

func TestMapLoadOrCompute_FunctionCalledOnce(t *testing.T) {
	m := NewMap()
	for i := 0; i < 100; {
		m.LoadOrCompute(strconv.Itoa(i), func() (v interface{}) {
			v, i = i, i+1
			return v
		})
	}

	m.Range(func(k string, v interface{}) bool {
		if vi, ok := v.(int); !ok || strconv.Itoa(vi) != k {
			t.Fatalf("%sth key is not equal to value %d", k, v)
		}
		return true
	})
}

func TestMapCompute(t *testing.T) {
	var zeroedV interface{}
	m := NewMap()
	// Store a new value.
	v, ok := m.Compute("foobar", func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
		if oldValue != zeroedV {
			t.Fatalf("oldValue should be empty interface{} when computing a new value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when computing a new value")
		}
		newValue = 42
		delete = false
		return
	})
	if v.(int) != 42 {
		t.Fatalf("v should be 42 when computing a new value: %d", v)
	}
	if !ok {
		t.Fatal("ok should be true when computing a new value")
	}
	// Update an existing value.
	v, ok = m.Compute("foobar", func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
		if oldValue.(int) != 42 {
			t.Fatalf("oldValue should be 42 when updating the value: %d", oldValue)
		}
		if !loaded {
			t.Fatal("loaded should be true when updating the value")
		}
		newValue = oldValue.(int) + 42
		delete = false
		return
	})
	if v.(int) != 84 {
		t.Fatalf("v should be 84 when updating the value: %d", v)
	}
	if !ok {
		t.Fatal("ok should be true when updating the value")
	}
	// Delete an existing value.
	v, ok = m.Compute("foobar", func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
		if oldValue != 84 {
			t.Fatalf("oldValue should be 84 when deleting the value: %d", oldValue)
		}
		if !loaded {
			t.Fatal("loaded should be true when deleting the value")
		}
		delete = true
		return
	})
	if v.(int) != 84 {
		t.Fatalf("v should be 84 when deleting the value: %d", v)
	}
	if ok {
		t.Fatal("ok should be false when deleting the value")
	}
	// Try to delete a non-existing value. Notice different key.
	v, ok = m.Compute("barbaz", func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
		var zeroedV interface{}
		if oldValue != zeroedV {
			t.Fatalf("oldValue should be empty interface{} when trying to delete a non-existing value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when trying to delete a non-existing value")
		}
		// We're returning a non-zero value, but the map should ignore it.
		newValue = 42
		delete = true
		return
	})
	if v != zeroedV {
		t.Fatalf("v should be empty interface{} when trying to delete a non-existing value: %d", v)
	}
	if ok {
		t.Fatal("ok should be false when trying to delete a non-existing value")
	}
}

func TestMapStoreThenDelete(t *testing.T) {
	const numEntries = 1000
	m := NewMap()
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

func TestMapStoreThenLoadAndDelete(t *testing.T) {
	const numEntries = 1000
	m := NewMap()
	for i := 0; i < numEntries; i++ {
		m.Store(strconv.Itoa(i), i)
	}
	for i := 0; i < numEntries; i++ {
		if v, loaded := m.LoadAndDelete(strconv.Itoa(i)); !loaded || v.(int) != i {
			t.Fatalf("value was not found or different for %d: %v", i, v)
		}
		if _, ok := m.Load(strconv.Itoa(i)); ok {
			t.Fatalf("value was not expected for %d", i)
		}
	}
}

func sizeBasedOnRange(m Map) int {
	size := 0
	m.Range(func(key string, value interface{}) bool {
		size++
		return true
	})
	return size
}

func TestMapSize(t *testing.T) {
	const numEntries = 1000
	m := NewMap()
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
		rsize := sizeBasedOnRange(m)
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
		rsize := sizeBasedOnRange(m)
		if size != rsize {
			t.Fatalf("size does not match number of entries in Range: %v, %v", size, rsize)
		}
	}
}

func TestMapClear(t *testing.T) {
	const numEntries = 1000
	m := NewMap()
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
	rsize := sizeBasedOnRange(m)
	if rsize != 0 {
		t.Fatalf("zero number of entries in Range was expected, got: %d", rsize)
	}
}

func parallelSeqResizer(t *testing.T, m Map, numEntries int, positive bool, cdone chan bool) {
	for i := 0; i < numEntries; i++ {
		if positive {
			m.Store(strconv.Itoa(i), i)
		} else {
			m.Store(strconv.Itoa(-i), -i)
		}
	}
	cdone <- true
}

func TestMapParallelResize_GrowOnly(t *testing.T) {
	const numEntries = 100_000
	m := NewMap()
	cdone := make(chan bool)
	go parallelSeqResizer(t, m, numEntries, true, cdone)
	go parallelSeqResizer(t, m, numEntries, false, cdone)
	// Wait for the goroutines to finish.
	<-cdone
	<-cdone
	// Verify map contents.
	for i := -numEntries + 1; i < numEntries; i++ {
		v, ok := m.Load(strconv.Itoa(i))
		if !ok {
			t.Fatalf("value not found for %d", i)
		}
		if vi, ok := v.(int); ok && vi != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
	if s := m.Size(); s != 2*numEntries-1 {
		t.Fatalf("unexpected size: %v", s)
	}
}

func parallelRandResizer(t *testing.T, m Map, numIters, numEntries int, cdone chan bool) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < numIters; i++ {
		coin := r.Int63n(2)
		for j := 0; j < numEntries; j++ {
			if coin == 1 {
				m.Store(strconv.Itoa(j), j)
			} else {
				m.Delete(strconv.Itoa(j))
			}
		}
	}
	cdone <- true
}

func TestMapParallelResize(t *testing.T) {
	const numIters = 1_000
	const numEntries = 2 * 3 * 32
	m := NewMap()
	cdone := make(chan bool)
	go parallelRandResizer(t, m, numIters, numEntries, cdone)
	go parallelRandResizer(t, m, numIters, numEntries, cdone)
	// Wait for the goroutines to finish.
	<-cdone
	<-cdone
	// Verify map contents.
	for i := 0; i < numEntries; i++ {
		v, ok := m.Load(strconv.Itoa(i))
		if !ok {
			// The entry may be deleted and that's ok.
			continue
		}
		if vi, ok := v.(int); ok && vi != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
	s := m.Size()
	if s > numEntries {
		t.Fatalf("unexpected size: %v", s)
	}
	rs := sizeBasedOnRange(m)
	if s != rs {
		t.Fatalf("size does not match number of entries in Range: %v, %v", s, rs)
	}
}

func parallelRandClearer(t *testing.T, m Map, numIters, numEntries int, cdone chan bool) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < numIters; i++ {
		coin := r.Int63n(2)
		for j := 0; j < numEntries; j++ {
			if coin == 1 {
				m.Store(strconv.Itoa(j), j)
			} else {
				m.Clear()
			}
		}
	}
	cdone <- true
}

func TestMapParallelClear(t *testing.T) {
	const numIters = 100
	const numEntries = 1_000
	m := NewMap()
	cdone := make(chan bool)
	go parallelRandClearer(t, m, numIters, numEntries, cdone)
	go parallelRandClearer(t, m, numIters, numEntries, cdone)
	// Wait for the goroutines to finish.
	<-cdone
	<-cdone
	// Verify map size.
	s := m.Size()
	if s > numEntries {
		t.Fatalf("unexpected size: %v", s)
	}
	rs := sizeBasedOnRange(m)
	if s != rs {
		t.Fatalf("size does not match number of entries in Range: %v, %v", s, rs)
	}
}

func parallelSeqStorer(t *testing.T, m Map, storeEach, numIters, numEntries int, cdone chan bool) {
	for i := 0; i < numIters; i++ {
		for j := 0; j < numEntries; j++ {
			if storeEach == 0 || j%storeEach == 0 {
				m.Store(strconv.Itoa(j), j)
				// Due to atomic snapshots we must see a "<j>"/j pair.
				v, ok := m.Load(strconv.Itoa(j))
				if !ok {
					t.Errorf("value was not found for %d", j)
					break
				}
				if vi, ok := v.(int); !ok || vi != j {
					t.Errorf("value was not expected for %d: %d", j, vi)
					break
				}
			}
		}
	}
	cdone <- true
}

func TestMapParallelStores(t *testing.T) {
	const numStorers = 4
	const numIters = 10_000
	const numEntries = 100
	m := NewMap()
	cdone := make(chan bool)
	for i := 0; i < numStorers; i++ {
		go parallelSeqStorer(t, m, i, numIters, numEntries, cdone)
	}
	// Wait for the goroutines to finish.
	for i := 0; i < numStorers; i++ {
		<-cdone
	}
	// Verify map contents.
	for i := 0; i < numEntries; i++ {
		v, ok := m.Load(strconv.Itoa(i))
		if !ok {
			t.Fatalf("value not found for %d", i)
		}
		if vi, ok := v.(int); ok && vi != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}

func parallelRandStorer(t *testing.T, m Map, numIters, numEntries int, cdone chan bool) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < numIters; i++ {
		j := r.Intn(numEntries)
		if v, loaded := m.LoadOrStore(strconv.Itoa(j), j); loaded {
			if vi, ok := v.(int); !ok || vi != j {
				t.Errorf("value was not expected for %d: %d", j, vi)
			}
		}
	}
	cdone <- true
}

func parallelRandDeleter(t *testing.T, m Map, numIters, numEntries int, cdone chan bool) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < numIters; i++ {
		j := r.Intn(numEntries)
		if v, loaded := m.LoadAndDelete(strconv.Itoa(j)); loaded {
			if vi, ok := v.(int); !ok || vi != j {
				t.Errorf("value was not expected for %d: %d", j, vi)
			}
		}
	}
	cdone <- true
}

func parallelLoader(t *testing.T, m Map, numIters, numEntries int, cdone chan bool) {
	for i := 0; i < numIters; i++ {
		for j := 0; j < numEntries; j++ {
			// Due to atomic snapshots we must either see no entry, or a "<j>"/j pair.
			if v, ok := m.Load(strconv.Itoa(j)); ok {
				if vi, ok := v.(int); !ok || vi != j {
					t.Errorf("value was not expected for %d: %d", j, vi)
				}
			}
		}
	}
	cdone <- true
}

func TestMapAtomicSnapshot(t *testing.T) {
	const numIters = 100_000
	const numEntries = 100
	m := NewMap()
	cdone := make(chan bool)
	// Update or delete random entry in parallel with loads.
	go parallelRandStorer(t, m, numIters, numEntries, cdone)
	go parallelRandDeleter(t, m, numIters, numEntries, cdone)
	go parallelLoader(t, m, numIters, numEntries, cdone)
	// Wait for the goroutines to finish.
	for i := 0; i < 3; i++ {
		<-cdone
	}
}

func TestMapParallelStoresAndDeletes(t *testing.T) {
	const numWorkers = 2
	const numIters = 100_000
	const numEntries = 1000
	m := NewMap()
	cdone := make(chan bool)
	// Update random entry in parallel with deletes.
	for i := 0; i < numWorkers; i++ {
		go parallelRandStorer(t, m, numIters, numEntries, cdone)
		go parallelRandDeleter(t, m, numIters, numEntries, cdone)
	}
	// Wait for the goroutines to finish.
	for i := 0; i < 2*numWorkers; i++ {
		<-cdone
	}
}

func parallelComputer(t *testing.T, m Map, numIters, numEntries int, cdone chan bool) {
	for i := 0; i < numIters; i++ {
		for j := 0; j < numEntries; j++ {
			m.Compute(strconv.Itoa(j), func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
				if !loaded {
					return uint64(1), false
				}
				return uint64(oldValue.(uint64) + 1), false
			})
		}
	}
	cdone <- true
}

func TestMapParallelComputes(t *testing.T) {
	const numWorkers = 4 // Also stands for numEntries.
	const numIters = 10_000
	m := NewMap()
	cdone := make(chan bool)
	for i := 0; i < numWorkers; i++ {
		go parallelComputer(t, m, numIters, numWorkers, cdone)
	}
	// Wait for the goroutines to finish.
	for i := 0; i < numWorkers; i++ {
		<-cdone
	}
	// Verify map contents.
	for i := 0; i < numWorkers; i++ {
		v, ok := m.Load(strconv.Itoa(i))
		if !ok {
			t.Fatalf("value not found for %d", i)
		}
		if v.(uint64) != numWorkers*numIters {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}
