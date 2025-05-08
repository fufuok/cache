package cache

import (
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

type testStruct struct {
	num int
	sub *testStruct
}

const (
	testDefaultExpiration = 1 * time.Second
	testCleanupInterval   = 1 * time.Millisecond
)

var (
	t1 = testStruct{
		num: 1,
		sub: nil,
	}
	t2 = testStruct{
		num: 2,
		sub: &t1,
	}
	testKV = []kv[string, any]{
		{"string", "s"},
		{"int", 1},
		{"int32", int32(32)},
		{"int64", int64(64)},
		{"float32", float32(3.14)},
		{"float64", 3.14},
		{"nil", nil},
		{"pointer", &t1},
		{"struct", t2},
	}
)

func mockCache(opts ...Option[string, any]) Cache[string, any] {
	if len(opts) == 0 {
		opts = []Option[string, any]{
			WithDefaultExpiration[string, any](testDefaultExpiration),
			WithCleanupInterval[string, any](testCleanupInterval),
		}
	}
	c := New[string, any](opts...)
	for _, x := range testKV {
		c.SetDefault(x.k, x.v)
	}
	c.Set("70ms", 1, 70*time.Millisecond)
	return c
}

func TestCache_Expire(t *testing.T) {
	exp := 20 * time.Millisecond
	interval := 1 * time.Millisecond
	opts := []Option[string, int]{
		WithDefaultExpiration[string, int](exp),
		WithCleanupInterval[string, int](interval),
	}
	caches := []Cache[string, int]{
		New[string, int](opts...),
		NewDefault[string, int](exp, interval),
	}
	for _, c := range caches {
		c.Set("a", 1, 0)
		c.Set("b", 2, DefaultExpiration)
		c.Set("c", 3, NoExpiration)
		c.Set("d", 4, 20*time.Millisecond)
		c.Set("e", 5, 150*time.Millisecond)

		<-time.After(25 * time.Millisecond)
		_, ok := c.Get("d")
		if ok {
			t.Fatal("key d should be automatically deleted")
		}

		<-time.After(30 * time.Millisecond)
		_, ok = c.Get("b")
		if ok {
			t.Fatal("key b should be automatically deleted")
		}
		_, ok = c.Get("a")
		if !ok {
			t.Fatal("key a is set to never expire, but not found")
		}
		_, ok = c.Get("c")
		if !ok {
			t.Fatal("key c is set to never expire, but not found")
		}
		_, ok = c.Get("e")
		if !ok {
			t.Fatal("key e has not expired but was not found")
		}

		<-time.After(100 * time.Millisecond)
		_, ok = c.Get("e")
		if ok {
			t.Fatal("key e should be automatically deleted")
		}

		n := c.Count()
		if n != 2 {
			t.Fatalf("expected number of items in cache to be 2, got: %d", n)
		}

		c.Clear()

		n = c.Count()
		if n != 0 {
			t.Fatalf("expected number of items in cache to be 0, got: %d", n)
		}
	}
}

func TestCache_Integer_Expire(t *testing.T) {
	exp := 20 * time.Millisecond
	interval := 1 * time.Millisecond
	opts := []Option[int, int]{
		WithDefaultExpiration[int, int](exp),
		WithCleanupInterval[int, int](interval),
	}
	caches := []Cache[int, int]{
		New[int, int](opts...),
		NewDefault[int, int](exp, interval),
	}
	for _, c := range caches {
		c.Set(1, 1, 0)
		c.Set(2, 2, DefaultExpiration)
		c.Set(3, 3, NoExpiration)
		c.Set(4, 4, 20*time.Millisecond)
		c.Set(5, 5, 100*time.Millisecond)

		<-time.After(25 * time.Millisecond)
		_, ok := c.Get(4)
		if ok {
			t.Fatal("key 4 should be automatically deleted")
		}

		<-time.After(30 * time.Millisecond)
		_, ok = c.Get(2)
		if ok {
			t.Fatal("key 2 should be automatically deleted")
		}
		_, ok = c.Get(1)
		if !ok {
			t.Fatal("key 1 is set to never expire, but not found")
		}
		_, ok = c.Get(3)
		if !ok {
			t.Fatal("key 3 is set to never expire, but not found")
		}
		_, ok = c.Get(5)
		if !ok {
			t.Fatal("key 5 has not expired but was not found")
		}

		<-time.After(50 * time.Millisecond)
		_, ok = c.Get(5)
		if ok {
			t.Fatal("key 5 should be automatically deleted")
		}
	}
}

func TestCache_SetAndGet(t *testing.T) {
	c := mockCache()
	for _, x := range testKV {
		v, ok := c.Get(x.k)
		if !ok {
			t.Fatalf("key `%s` should have a value: %v", x.k, x.v)
		}
		if !reflect.DeepEqual(v, x.v) {
			t.Fatalf("deep equal: %v != %v", v, x.v)
		}
	}

	_, ok := c.Get("70ms")
	if !ok {
		t.Fatal("key `70ms` should have a value")
	}
	_, ok = c.Get("not exist")
	if ok {
		t.Fatal("key `not exist` should not have a value")
	}
}

func TestCache_SetDefault_WithoutCleanup(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := NewDefault[string, int](defaultExpiration, 0)
	c.SetDefault("x", 1)
	v, ok := c.Get("x")
	if !ok || v != 1 {
		t.Fatal("key x should have a value")
	}

	v, ttl, ok := c.GetWithTTL("x")
	if !ok || v != 1 {
		t.Fatal("key x should have a value")
	}
	if ttl < 30*time.Millisecond || ttl > defaultExpiration {
		t.Fatal("incorrect lifetime for key x")
	}

	<-time.After(55 * time.Millisecond)
	v, ok = c.Get("x")
	if ok || v != 0 {
		t.Fatal("since key x is expired, it should be automatically deleted on get().")
	}

	if c.Count() != 0 {
		t.Fatalf("incorrect number of items in cache, expected %d, got %d", 0, c.Count())
	}
}

func TestCache_SetDefault(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := NewDefault[string, int](defaultExpiration, testCleanupInterval)
	c.SetDefault("x", 1)
	v, ok := c.Get("x")
	if !ok || v != 1 {
		t.Fatal("key x should have a value")
	}

	v, ttl, ok := c.GetWithTTL("x")
	if !ok || v != 1 {
		t.Fatal("key x should have a value")
	}
	if ttl < 30*time.Millisecond || ttl > defaultExpiration {
		t.Fatal("incorrect lifetime for key x")
	}

	<-time.After(55 * time.Millisecond)
	v, ok = c.Get("x")
	if ok || v != 0 {
		t.Fatal("key x should be automatically deleted")
	}

	if c.Count() != 0 {
		t.Fatalf("incorrect number of items in cache, expected %d, got %d", 0, c.Count())
	}
}

func TestCache_SetForever(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := NewDefault[string, int](defaultExpiration, testCleanupInterval)
	c.SetForever("x", 1)
	v, ok := c.Get("x")
	if !ok || v != 1 {
		t.Fatal("key x should have a value")
	}

	<-time.After(55 * time.Millisecond)
	v, ttl, ok := c.GetWithTTL("x")
	if !ok || v != 1 || ttl != NoExpiration {
		t.Fatal("the lifetime of key x should be forever")
	}
}

func TestCache_GetOrSet(t *testing.T) {
	c := New[string, int]()
	v, ok := c.GetOrSet("x", 1, testDefaultExpiration)
	if ok {
		t.Fatal("key x should not loaded")
	}
	if v != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	v, ok = c.GetOrSet("x", 2, testDefaultExpiration)
	if !ok || v != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	y, ok := c.Get("x")
	if !ok || y != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, y)
	}
}

func TestCache_GetAndSet(t *testing.T) {
	c := New[string, int]()
	v, ok := c.GetAndSet("x", 1, testDefaultExpiration)
	if ok {
		t.Fatal("key x should not loaded")
	}
	if v != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	v, ok = c.GetAndSet("x", 2, 50*time.Millisecond)
	if !ok || v != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	y, ok := c.Get("x")
	if !ok || y != 2 {
		t.Fatalf("key x, expected %d, got %d", 2, y)
	}

	// Always reset expiration time
	c.GetAndSet("x", 2, 200*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	z, ok := c.Get("x")
	if !ok || z != 2 {
		t.Fatalf("key x, expected %d, got %d", 3, z)
	}

	time.Sleep(110 * time.Millisecond)
	_, ok = c.Get("x")
	if ok {
		t.Fatal("key x should not loaded")
	}
}

func TestCache_GetAndRefresh(t *testing.T) {
	c := NewDefault[string, int](100*time.Millisecond, testCleanupInterval)
	c.SetDefault("x", 1)
	v, tm, ok := c.GetWithExpiration("x")
	if !ok || v != 1 || tm.Before(time.Now()) {
		t.Fatal("failed to get the value and expiration time of key x")
	}

	<-time.After(50 * time.Millisecond)
	v, ttl, ok := c.GetWithTTL("x")
	if !ok || v != 1 || ttl > 50*time.Millisecond {
		t.Fatalf("key X lifetime is incorrect, expected <= 50ms, got %d", ttl)
	}

	c.GetAndRefresh("x", 1*time.Second)
	v, ttl, ok = c.GetWithTTL("x")
	if !ok || v != 1 || ttl < 500*time.Millisecond {
		t.Fatalf("key X lifetime is incorrect, expected >= 500ms, got %d", ttl)
	}
	v, tm, ok = c.GetWithExpiration("x")
	if !ok || v != 1 || tm.Before(time.Now()) {
		t.Fatal("failed to get the value and expiration time of key x")
	}
}

func TestCache_GetOrCompute(t *testing.T) {
	const numEntries = 1000
	c := New[string, int](WithMinCapacity[string, int](numEntries))
	for i := 0; i < numEntries; i++ {
		v, loaded := c.GetOrCompute(strconv.Itoa(i), func() (int, bool) {
			return i, false
		}, 0)
		if loaded {
			t.Fatalf("value not computed for %d", i)
		}
		if v != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
	for i := 0; i < numEntries; i++ {
		v, loaded := c.GetOrCompute(strconv.Itoa(i), func() (int, bool) {
			return i, false
		}, 0)
		if !loaded {
			t.Fatalf("value not loaded for %d", i)
		}
		if v != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}

func TestCache_GetOrCompute_WithKeyExpired(t *testing.T) {
	c := New[string, int]()
	v, loaded := c.GetOrCompute("1", func() (int, bool) {
		return 1, false
	}, 0)
	if loaded {
		t.Fatal("value not computed for 1")
	}
	if v != 1 {
		t.Fatalf("values do not match for 1: %v", v)
	}

	v, loaded = c.GetAndRefresh("1", 10*time.Millisecond)
	if !loaded {
		t.Fatal("value not loaded for 1")
	}
	if v != 1 {
		t.Fatalf("values do not match for 1: %v", v)
	}

	v, loaded = c.GetOrCompute("1", func() (int, bool) {
		return 2, false
	}, 0)
	if !loaded {
		t.Fatal("value not loaded for 1")
	}
	if v != 1 {
		t.Fatalf("values do not match for 1: %v", v)
	}

	time.Sleep(10 * time.Millisecond)

	v, loaded = c.GetOrCompute("1", func() (int, bool) {
		return 1, false
	}, 0)
	if loaded {
		t.Fatal("value not computed for 1")
	}
	if v != 1 {
		t.Fatalf("values do not match for 1: %v", v)
	}

	v, loaded = c.GetOrCompute("3", func() (int, bool) {
		return 3, true
	}, 0)
	if loaded {
		t.Fatal("value not computed for 3")
	}
	if v != 0 {
		t.Fatalf("values do not match for 0: %v", v)
	}

	v, loaded = c.GetOrCompute("3", func() (int, bool) {
		return 33, false
	}, 0)
	if loaded {
		t.Fatal("value not computed for 3")
	}
	if v != 33 {
		t.Fatalf("values do not match for 33: %v", v)
	}

	v, loaded = c.GetOrCompute("3", func() (int, bool) {
		return 333, false
	}, 0)
	if !loaded {
		t.Fatal("value not loaded for 3")
	}
	if v != 33 {
		t.Fatalf("values do not match for 33: %v", v)
	}
}

func TestCache_GetOrCompute_FunctionCalledOnce(t *testing.T) {
	c := New[int, int]()
	for i := 0; i < 100; {
		c.GetOrCompute(i, func() (v int, cancel bool) {
			v, i = i, i+1
			return v, false
		}, 0)
	}
	c.Range(func(k, v int) bool {
		if k != v {
			t.Fatalf("%dth key is not equal to value %d", k, v)
		}
		return true
	})
}

func TestCache_Compute(t *testing.T) {
	c := New[string, int]()
	// Store a new value.
	v, ok := c.Compute("foobar", func(oldValue int, loaded bool) (newValue int, op ComputeOp) {
		if oldValue != 0 {
			t.Fatalf("oldValue should be 0 when computing a new value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when computing a new value")
		}
		newValue = 42
		op = UpdateOp
		return
	}, 0)
	if v != 42 {
		t.Fatalf("v should be 42 when computing a new value: %d", v)
	}
	if !ok {
		t.Fatal("ok should be true when computing a new value")
	}
	// Update an existing value.
	v, ok = c.Compute("foobar", func(oldValue int, loaded bool) (newValue int, op ComputeOp) {
		if oldValue != 42 {
			t.Fatalf("oldValue should be 42 when updating the value: %d", oldValue)
		}
		if !loaded {
			t.Fatal("loaded should be true when updating the value")
		}
		newValue = oldValue + 42
		op = UpdateOp
		return
	}, 0)
	if v != 84 {
		t.Fatalf("v should be 84 when updating the value: %d", v)
	}
	if !ok {
		t.Fatal("ok should be true when updating the value")
	}
	// Delete an existing value.
	v, ok = c.Compute("foobar", func(oldValue int, loaded bool) (newValue int, op ComputeOp) {
		if oldValue != 84 {
			t.Fatalf("oldValue should be 84 when deleting the value: %d", oldValue)
		}
		if !loaded {
			t.Fatal("loaded should be true when deleting the value")
		}
		op = DeleteOp
		return
	}, 0)
	if v != 84 {
		t.Fatalf("v should be 84 when deleting the value: %d", v)
	}
	if ok {
		t.Fatal("ok should be false when deleting the value")
	}
	// Try to delete a non-existing value. Notice different key.
	v, ok = c.Compute("barbaz", func(oldValue int, loaded bool) (newValue int, op ComputeOp) {
		if oldValue != 0 {
			t.Fatalf("oldValue should be 0 when trying to delete a non-existing value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when trying to delete a non-existing value")
		}
		// We're returning a non-zero value, but the map should ignore it.
		newValue = 42
		op = DeleteOp
		return
	}, 0)
	if v != 0 {
		t.Fatalf("v should be 0 when trying to delete a non-existing value: %d", v)
	}
	if ok {
		t.Fatal("ok should be false when trying to delete a non-existing value")
	}
	// Store a new value.
	v, ok = c.Compute("expires soon", func(oldValue int, loaded bool) (newValue int, op ComputeOp) {
		if oldValue != 0 {
			t.Fatalf("oldValue should be 0 when computing a new value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when computing a new value")
		}
		newValue = 42
		op = UpdateOp
		return
	}, 10*time.Millisecond)
	if v != 42 {
		t.Fatalf("v should be 42 when computing a new value: %d", v)
	}
	if !ok {
		t.Fatal("ok should be true when computing a new value")
	}
	time.Sleep(10 * time.Millisecond)
	// Try to delete a expired value. Notice different key.
	v, ok = c.Compute("expires soon", func(oldValue int, loaded bool) (newValue int, op ComputeOp) {
		if oldValue != 0 {
			t.Fatalf("oldValue should be 0 when trying to delete a expired value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when trying to delete a expired value")
		}
		// We're returning a non-zero value, but the map should ignore it.
		newValue = 42
		op = DeleteOp
		return
	}, 0)
	if v != 0 {
		t.Fatalf("v should be 0 when trying to delete a expired value: %d", v)
	}
	if ok {
		t.Fatal("ok should be false when trying to delete a expired value")
	}
}

func TestCache_GetAndDelete(t *testing.T) {
	c := New[string, int]()
	v, ok := c.GetAndDelete("x")
	if ok || v != 0 {
		t.Fatal("key a should not exist")
	}

	c.SetDefault("x", 1)

	v, ok = c.GetAndDelete("x")
	if !ok || v != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	v, ok = c.Get("x")
	if ok || v != 0 {
		t.Fatal("key x should be deleted")
	}
}

func TestCache_Delete(t *testing.T) {
	c := New[string, int]()
	c.Delete("x")

	c.SetForever("x", 1)
	v, ok := c.Get("x")
	if !ok || v != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	c.Delete("x")

	v, ok = c.Get("x")
	if ok || v != 0 {
		t.Fatal("key x should be deleted")
	}
}

func TestCache_DeleteExpired(t *testing.T) {
	var n int64
	testEvictedCallback := func(k string, v int64) {
		atomic.AddInt64(&n, v)
	}
	c := NewDefault[string, int64](10*time.Millisecond, 5*time.Millisecond, testEvictedCallback)
	for i := 0; i < 10; i++ {
		c.SetDefault(strconv.Itoa(i), int64(i))
	}

	<-time.After(200 * time.Millisecond)
	m := atomic.LoadInt64(&n)
	if m != 45 {
		t.Fatalf("evicted callback executes incorrectly, expected %d, got %d", 45, m)
	}

	v, ok := c.Get("1")
	if ok || v != 0 {
		t.Fatal("key 1 should have expired, but was fetched")
	}
}

func countBasedOnTypedRange(c Cache[string, int]) int {
	size := 0
	c.Range(func(key string, value int) bool {
		size++
		return true
	})
	return size
}

func TestCache_Size(t *testing.T) {
	const numEntries = 1000
	c := New[string, int]()
	size := c.Count()
	if size != 0 {
		t.Fatalf("zero size expected: %d", size)
	}
	expectedSize := 0
	for i := 0; i < numEntries; i++ {
		c.SetDefault(strconv.Itoa(i), i)
		expectedSize++
		size := c.Count()
		if size != expectedSize {
			t.Fatalf("size of %d was expected, got: %d", expectedSize, size)
		}
		rsize := countBasedOnTypedRange(c)
		if size != rsize {
			t.Fatalf("size does not match number of entries in Range: %v, %v", size, rsize)
		}
	}
	for i := 0; i < numEntries; i++ {
		c.Delete(strconv.Itoa(i))
		expectedSize--
		size := c.Count()
		if size != expectedSize {
			t.Fatalf("size of %d was expected, got: %d", expectedSize, size)
		}
		rsize := countBasedOnTypedRange(c)
		if size != rsize {
			t.Fatalf("size does not match number of entries in Range: %v, %v", size, rsize)
		}
	}
}

func TestCache_Clear(t *testing.T) {
	const numEntries = 1000
	c := New[string, int]()
	for i := 0; i < numEntries; i++ {
		c.SetDefault(strconv.Itoa(i), i)
	}
	size := c.Count()
	if size != numEntries {
		t.Fatalf("size of %d was expected, got: %d", numEntries, size)
	}
	c.Clear()
	size = c.Count()
	if size != 0 {
		t.Fatalf("zero size was expected, got: %d", size)
	}
	rsize := countBasedOnTypedRange(c)
	if rsize != 0 {
		t.Fatalf("zero number of entries in Range was expected, got: %d", rsize)
	}
}

func TestCache_Range(t *testing.T) {
	var n int64
	testRange := func(k string, v int64) bool {
		atomic.AddInt64(&n, v)
		return true
	}
	c := New[string, int64]()
	for i := 0; i < 10; i++ {
		c.SetDefault(strconv.Itoa(i), int64(i))
	}
	c.Range(testRange)
	m := atomic.LoadInt64(&n)
	if m != 45 {
		t.Fatalf("the traversal is executed incorrectly, expected %d, got %d", 45, m)
	}

	if c.Count() != 10 {
		t.Fatalf("incorrect number of items in cache, expected %d, got %d", 10, c.Count())
	}
}
