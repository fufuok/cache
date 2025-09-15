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

func TestCache_ItemsWithExpiration(t *testing.T) {
	c := NewDefault[string, int](100*time.Millisecond, testCleanupInterval)

	// Add test data
	c.Set("never_expire", 1, NoExpiration)
	c.Set("expire_100ms", 2, 100*time.Millisecond)
	c.Set("expire_200ms", 3, 200*time.Millisecond)
	c.SetDefault("default_expire", 4) // Use default expiration time

	// Wait for some items to expire
	time.Sleep(50 * time.Millisecond)

	items := c.ItemsWithExpiration()

	// Verify v count
	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}

	// Verify never-expire v
	v, ok := items["never_expire"]
	if !ok {
		t.Fatal("never_expire v should exist")
	}
	if v.Value != 1 {
		t.Fatalf("never_expire value should be 1, got %d", v.Value)
	}
	if !v.Expiration.IsZero() {
		t.Fatal("never_expire should have zero expiration time")
	}

	// Verify v with expiration time
	v, ok = items["expire_100ms"]
	if !ok {
		t.Fatal("expire_100ms v should exist")
	}
	if v.Value != 2 {
		t.Fatalf("expire_100ms value should be 2, got %d", v.Value)
	}
	if v.Expiration.IsZero() {
		t.Fatal("expire_100ms should have non-zero expiration time")
	}

	// Verify v with default expiration time
	v, ok = items["default_expire"]
	if !ok {
		t.Fatal("default_expire v should exist")
	}
	if v.Value != 4 {
		t.Fatalf("default_expire value should be 4, got %d", v.Value)
	}
	if v.Expiration.IsZero() {
		t.Fatal("default_expire should have non-zero expiration time")
	}

	// Wait for more items to expire
	time.Sleep(100 * time.Millisecond)

	items = c.ItemsWithExpiration()

	// Verify expired items are not in result
	if _, ok := items["expire_100ms"]; ok {
		t.Fatal("expire_100ms should have expired and not be in items")
	}

	// Verify never-expire v still exists
	if _, ok := items["never_expire"]; !ok {
		t.Fatal("never_expire should still exist")
	}

	c.Close()
}

func TestCache_LoadItems(t *testing.T) {
	c := New[string, int]()

	// Prepare data to load
	itemsToLoad := map[string]int{
		"item1": 100,
		"item2": 200,
		"item3": 300,
	}

	// Load data
	c.LoadItems(itemsToLoad, 50*time.Millisecond)

	// Verify data is loaded correctly
	for k, expectedV := range itemsToLoad {
		v, ok := c.Get(k)
		if !ok {
			t.Fatalf("item %s should be loaded", k)
		}
		if v != expectedV {
			t.Fatalf("item %s: expected %d, got %d", k, expectedV, v)
		}

		// Verify expiration time
		_, ttl, ok := c.GetWithTTL(k)
		if !ok {
			t.Fatalf("item %s should exist", k)
		}
		if ttl <= 0 || ttl > 50*time.Millisecond {
			t.Fatalf("item %s: incorrect TTL %v", k, ttl)
		}
	}

	// Verify item count
	if c.Count() != len(itemsToLoad) {
		t.Fatalf("expected %d items, got %d", len(itemsToLoad), c.Count())
	}

	// Test loading with existing data
	c.Set("existing", 999, NoExpiration)

	moreItems := map[string]int{
		"item4": 400,
		"item5": 500,
	}

	c.LoadItems(moreItems, NoExpiration)

	// Verify all data exists
	totalExpected := len(itemsToLoad) + len(moreItems) + 1 // +1 for existing
	if c.Count() != totalExpected {
		t.Fatalf("expected %d items, got %d", totalExpected, c.Count())
	}

	// Verify existing data is not affected
	v, ok := c.Get("existing")
	if !ok || v != 999 {
		t.Fatal("existing item should not be affected")
	}

	c.Close()
}

func TestCache_LoadItemsWithExpiration(t *testing.T) {
	c := New[string, string]()

	// Prepare data with expiration times
	now := time.Now()
	itemsToLoad := map[string]ItemWithExpiration[string]{
		"never_expire": {
			Value:      "never expires",
			Expiration: time.Time{}, // Zero value means never expires
		},
		"expire_soon": {
			Value:      "expires soon",
			Expiration: now.Add(50 * time.Millisecond),
		},
		"expire_later": {
			Value:      "expires later",
			Expiration: now.Add(200 * time.Millisecond),
		},
	}

	// Load data
	c.LoadItemsWithExpiration(itemsToLoad)

	// Verify data is loaded correctly
	for k, expectedItem := range itemsToLoad {
		v, expiration, ok := c.GetWithExpiration(k)
		if !ok {
			t.Fatalf("item %s should be loaded", k)
		}
		if v != expectedItem.Value {
			t.Fatalf("item %s: expected value %s, got %s", k, expectedItem.Value, v)
		}

		// Verify expiration time
		if expectedItem.Expiration.IsZero() {
			// Never expires
			if !expiration.IsZero() {
				t.Fatalf("item %s should never expire", k)
			}
		} else {
			// Has expiration time
			if expiration.IsZero() {
				t.Fatalf("item %s should have expiration time", k)
			}
			// Allow some time tolerance
			diff := expiration.Sub(expectedItem.Expiration)
			if diff < -10*time.Millisecond || diff > 10*time.Millisecond {
				t.Fatalf("item %s: expiration time mismatch, expected %v, got %v",
					k, expectedItem.Expiration, expiration)
			}
		}
	}

	// Verify item count
	if c.Count() != len(itemsToLoad) {
		t.Fatalf("expected %d items, got %d", len(itemsToLoad), c.Count())
	}

	// Wait for some items to expire
	time.Sleep(80 * time.Millisecond)

	// Verify expired items
	_, ok := c.Get("expire_soon")
	if ok {
		t.Fatal("expire_soon should have expired")
	}

	// Verify non-expired items
	v, ok := c.Get("never_expire")
	if !ok || v != "never expires" {
		t.Fatal("never_expire should still exist")
	}

	v, ok = c.Get("expire_later")
	if !ok || v != "expires later" {
		t.Fatal("expire_later should still exist")
	}

	// Wait for remaining items to expire
	time.Sleep(150 * time.Millisecond)

	_, ok = c.Get("expire_later")
	if ok {
		t.Fatal("expire_later should have expired")
	}

	// Never-expire item should still exist
	v, ok = c.Get("never_expire")
	if !ok || v != "never expires" {
		t.Fatal("never_expire should still exist after all time")
	}

	// Test overwriting existing data
	c.Set("existing", "existing data", NoExpiration)

	newItems := map[string]ItemWithExpiration[string]{
		"existing": {
			Value:      "overwritten data",
			Expiration: time.Now().Add(100 * time.Millisecond),
		},
	}

	c.LoadItemsWithExpiration(newItems)

	v, _, ok = c.GetWithExpiration("existing")
	if !ok || v != "overwritten data" {
		t.Fatal("existing item should be overwritten")
	}

	c.Close()
}

func TestCache_LoadItems_EdgeCases(t *testing.T) {
	c := New[string, int]()

	// Test loading empty map
	emptyItems := make(map[string]int)
	c.LoadItems(emptyItems, testDefaultExpiration)
	if c.Count() != 0 {
		t.Fatal("cache should be empty after loading empty items")
	}

	// Test loading nil map (should not panic)
	var nilItems map[string]int
	c.LoadItems(nilItems, testDefaultExpiration)
	if c.Count() != 0 {
		t.Fatal("cache should be empty after loading nil items")
	}

	c.Close()
}

func TestCache_LoadItemsWithExpiration_EdgeCases(t *testing.T) {
	c := New[string, int]()

	// Test loading empty map
	emptyItems := make(map[string]ItemWithExpiration[int])
	c.LoadItemsWithExpiration(emptyItems)
	if c.Count() != 0 {
		t.Fatal("cache should be empty after loading empty items")
	}

	// Test loading nil map (should not panic)
	var nilItems map[string]ItemWithExpiration[int]
	c.LoadItemsWithExpiration(nilItems)
	if c.Count() != 0 {
		t.Fatal("cache should be empty after loading nil items")
	}

	// Test loading already expired items - they should be skipped
	pastTime := time.Now().Add(-100 * time.Millisecond)
	expiredItems := map[string]ItemWithExpiration[int]{
		"expired": {
			Value:      42,
			Expiration: pastTime,
		},
		"valid": {
			Value:      100,
			Expiration: time.Now().Add(100 * time.Millisecond),
		},
	}

	c.LoadItemsWithExpiration(expiredItems)

	// Expired items should be skipped and not loaded
	_, ok := c.Get("expired")
	if ok {
		t.Fatal("expired item should not be loaded")
	}

	// Valid items should be loaded normally
	v, ok := c.Get("valid")
	if !ok || v != 100 {
		t.Fatal("valid item should be loaded correctly")
	}

	// Cache should only contain the valid item
	if c.Count() != 1 {
		t.Fatalf("cache should contain 1 item, got %d", c.Count())
	}

	// Test expired items deletion: if an existing item has the same key as expired item
	c.Set("existing_key", 999, NoExpiration)

	expiredWithExistingKey := map[string]ItemWithExpiration[int]{
		"existing_key": {
			Value:      888,
			Expiration: pastTime, // Already expired
		},
	}

	c.LoadItemsWithExpiration(expiredWithExistingKey)

	// The existing item should be deleted since we tried to load an expired item with same key
	_, ok = c.Get("existing_key")
	if ok {
		t.Fatal("existing item should be deleted when trying to load expired item with same key")
	}

	c.Close()
}
