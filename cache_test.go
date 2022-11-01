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
	testKV = []kv{
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

func mockCache(opts ...Option) Cache {
	if len(opts) == 0 {
		opts = []Option{
			WithDefaultExpiration(testDefaultExpiration),
			WithCleanupInterval(testCleanupInterval),
		}
	}
	c := New(opts...)
	for _, x := range testKV {
		c.SetDefault(x.k, x.v)
	}
	c.Set("70ms", 1, 70*time.Millisecond)
	return c
}

func TestCache_Expire(t *testing.T) {
	c := NewDefault(20*time.Millisecond, 1*time.Millisecond)
	c.Set("a", 1, 0)
	c.Set("b", 2, DefaultExpiration)
	c.Set("c", 3, NoExpiration)
	c.Set("d", 4, 20*time.Millisecond)
	c.Set("e", 5, 100*time.Millisecond)

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

	<-time.After(50 * time.Millisecond)
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
	c := NewDefault(defaultExpiration, 0)
	c.SetDefault("x", 1)
	v, ok := c.Get("x")
	if !ok || v == nil {
		t.Fatal("key x should have a value")
	}

	v, ttl, ok := c.GetWithTTL("x")
	if !ok || v == nil {
		t.Fatal("key x should have a value")
	}
	if ttl < 30*time.Millisecond || ttl > defaultExpiration {
		t.Fatal("incorrect lifetime for key x")
	}

	<-time.After(55 * time.Millisecond)
	v, ok = c.Get("x")
	if ok || v != nil {
		t.Fatal("since key x is expired, it should be automatically deleted on get().")
	}

	if c.Count() != 0 {
		t.Fatalf("incorrect number of items in cache, expected %d, got %d", 0, c.Count())
	}
}

func TestCache_SetDefault(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := NewDefault(defaultExpiration, testCleanupInterval)
	c.SetDefault("x", 1)
	v, ok := c.Get("x")
	if !ok || v == nil {
		t.Fatal("key x should have a value")
	}

	v, ttl, ok := c.GetWithTTL("x")
	if !ok || v == nil {
		t.Fatal("key x should have a value")
	}
	if ttl < 30*time.Millisecond || ttl > defaultExpiration {
		t.Fatal("incorrect lifetime for key x")
	}

	<-time.After(55 * time.Millisecond)
	v, ok = c.Get("x")
	if ok || v != nil {
		t.Fatal("key x should be automatically deleted")
	}

	if c.Count() != 0 {
		t.Fatalf("incorrect number of items in cache, expected %d, got %d", 0, c.Count())
	}
}

func TestCache_SetForever(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := NewDefault(defaultExpiration, testCleanupInterval)
	c.SetForever("x", 1)
	v, ok := c.Get("x")
	if !ok || v == nil {
		t.Fatal("key x should have a value")
	}

	<-time.After(55 * time.Millisecond)
	v, ttl, ok := c.GetWithTTL("x")
	if !ok || v == nil || ttl != NoExpiration {
		t.Fatal("the lifetime of key x should be forever")
	}
}

func TestCache_GetOrSet(t *testing.T) {
	c := New()
	v, ok := c.GetOrSet("x", 1, testDefaultExpiration)
	if ok {
		t.Fatal("key x should not loaded")
	}
	x, ok := v.(int)
	if !ok || x != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	v, ok = c.GetOrSet("x", 2, testDefaultExpiration)
	if !ok || v.(int) != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	y, ok := c.Get("x")
	if !ok || y.(int) != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, y)
	}
}

func TestCache_GetAndSet(t *testing.T) {
	c := New()
	v, ok := c.GetAndSet("x", 1, testDefaultExpiration)
	if ok {
		t.Fatal("key x should not loaded")
	}
	x, ok := v.(int)
	if !ok || x != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	v, ok = c.GetAndSet("x", 2, testDefaultExpiration)
	if !ok || v.(int) != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	y, ok := c.Get("x")
	if !ok || y.(int) != 2 {
		t.Fatalf("key x, expected %d, got %d", 2, y)
	}
}

func TestCache_GetAndRefresh(t *testing.T) {
	c := NewDefault(100*time.Millisecond, testCleanupInterval)
	c.SetDefault("x", 1)
	v, tm, ok := c.GetWithExpiration("x")
	if !ok || v == nil || tm.Before(time.Now()) {
		t.Fatal("failed to get the value and expiration time of key x")
	}

	<-time.After(50 * time.Millisecond)
	v, ttl, ok := c.GetWithTTL("x")
	if !ok || v == nil || ttl > 50*time.Millisecond {
		t.Fatalf("key X lifetime is incorrect, expected <= 50ms, got %d", ttl)
	}

	c.GetAndRefresh("x", 1*time.Second)
	v, ttl, ok = c.GetWithTTL("x")
	if !ok || v == nil || ttl < 500*time.Millisecond {
		t.Fatalf("key X lifetime is incorrect, expected >= 500ms, got %d", ttl)
	}
	v, tm, ok = c.GetWithExpiration("x")
	if !ok || v == nil || v.(int) != 1 || tm.Before(time.Now()) {
		t.Fatal("failed to get the value and expiration time of key x")
	}
}

func TestCache_GetOrCompute(t *testing.T) {
	const numEntries = 1000
	c := New()
	for i := 0; i < numEntries; i++ {
		v, loaded := c.GetOrCompute(strconv.Itoa(i), func() interface{} {
			return i
		}, 0)
		if loaded {
			t.Fatalf("value not computed for %d", i)
		}
		if vi, ok := v.(int); ok && vi != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
	for i := 0; i < numEntries; i++ {
		v, loaded := c.GetOrCompute(strconv.Itoa(i), func() interface{} {
			return i
		}, 0)
		if !loaded {
			t.Fatalf("value not loaded for %d", i)
		}
		if vi, ok := v.(int); ok && vi != i {
			t.Fatalf("values do not match for %d: %v", i, v)
		}
	}
}

func TestCache_GetOrCompute_WithKeyExpired(t *testing.T) {
	c := New()
	v, loaded := c.GetOrCompute("1", func() interface{} {
		return 1
	}, 0)
	if loaded {
		t.Fatal("value not computed for 1")
	}
	if vi, ok := v.(int); ok && vi != 1 {
		t.Fatalf("values do not match for 1: %v", v)
	}

	v, loaded = c.GetAndRefresh("1", 10*time.Millisecond)
	if !loaded {
		t.Fatal("value not loaded for 1")
	}
	if vi, ok := v.(int); ok && vi != 1 {
		t.Fatalf("values do not match for 1: %v", v)
	}

	v, loaded = c.GetOrCompute("1", func() interface{} {
		return 2
	}, 0)
	if !loaded {
		t.Fatal("value not loaded for 1")
	}
	if vi, ok := v.(int); ok && vi != 1 {
		t.Fatalf("values do not match for 1: %v", v)
	}

	time.Sleep(10 * time.Millisecond)

	v, loaded = c.GetOrCompute("1", func() interface{} {
		return 1
	}, 0)
	if loaded {
		t.Fatal("value not computed for 1")
	}
	if vi, ok := v.(int); ok && vi != 1 {
		t.Fatalf("values do not match for 1: %v", v)
	}
}

func TestCache_GetOrCompute_FunctionCalledOnce(t *testing.T) {
	c := New()
	for i := 0; i < 100; {
		c.GetOrCompute(strconv.Itoa(i), func() (v interface{}) {
			v, i = i, i+1
			return v
		}, 0)
	}
	c.Range(func(k string, v interface{}) bool {
		if vi, ok := v.(int); !ok || strconv.Itoa(vi) != k {
			t.Fatalf("%sth key is not equal to value %d", k, v)
		}
		return true
	})
}

func TestCache_Compute(t *testing.T) {
	var zeroedV interface{}
	c := New()
	// Store a new value.
	v, ok := c.Compute("foobar", func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
		if oldValue != zeroedV {
			t.Fatalf("oldValue should be empty interface{} when computing a new value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when computing a new value")
		}
		newValue = 42
		delete = false
		return
	}, 0)
	if v.(int) != 42 {
		t.Fatalf("v should be 42 when computing a new value: %d", v)
	}
	if !ok {
		t.Fatal("ok should be true when computing a new value")
	}
	// Update an existing value.
	v, ok = c.Compute("foobar", func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
		if oldValue.(int) != 42 {
			t.Fatalf("oldValue should be 42 when updating the value: %d", oldValue)
		}
		if !loaded {
			t.Fatal("loaded should be true when updating the value")
		}
		newValue = oldValue.(int) + 42
		delete = false
		return
	}, 0)
	if v.(int) != 84 {
		t.Fatalf("v should be 84 when updating the value: %d", v)
	}
	if !ok {
		t.Fatal("ok should be true when updating the value")
	}
	// Delete an existing value.
	v, ok = c.Compute("foobar", func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
		if oldValue != 84 {
			t.Fatalf("oldValue should be 84 when deleting the value: %d", oldValue)
		}
		if !loaded {
			t.Fatal("loaded should be true when deleting the value")
		}
		delete = true
		return
	}, 0)
	if v.(int) != 84 {
		t.Fatalf("v should be 84 when deleting the value: %d", v)
	}
	if ok {
		t.Fatal("ok should be false when deleting the value")
	}
	// Try to delete a non-existing value. Notice different key.
	v, ok = c.Compute("barbaz", func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
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
	}, 0)
	if v != zeroedV {
		t.Fatalf("v should be empty interface{} when trying to delete a non-existing value: %d", v)
	}
	if ok {
		t.Fatal("ok should be false when trying to delete a non-existing value")
	}
	// Store a new value.
	v, ok = c.Compute("expires soon", func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
		if oldValue != zeroedV {
			t.Fatalf("oldValue should be empty interface{} when computing a new value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when computing a new value")
		}
		newValue = 42
		delete = false
		return
	}, 10*time.Millisecond)
	if v.(int) != 42 {
		t.Fatalf("v should be 42 when computing a new value: %d", v)
	}
	if !ok {
		t.Fatal("ok should be true when computing a new value")
	}
	time.Sleep(10 * time.Millisecond)
	// Try to delete a expired value. Notice different key.
	v, ok = c.Compute("expires soon", func(oldValue interface{}, loaded bool) (newValue interface{}, delete bool) {
		var zeroedV interface{}
		if oldValue != zeroedV {
			t.Fatalf("oldValue should be empty interface{} when trying to delete a expired value: %d", oldValue)
		}
		if loaded {
			t.Fatal("loaded should be false when trying to delete a expired value")
		}
		// We're returning a non-zero value, but the map should ignore it.
		newValue = 42
		delete = true
		return
	}, 10*time.Millisecond)
	if v != zeroedV {
		t.Fatalf("v should be empty interface{} when trying to delete a expired value: %d", v)
	}
	if ok {
		t.Fatal("ok should be false when trying to delete a expired value")
	}
}

func TestCache_GetAndDelete(t *testing.T) {
	c := New()
	v, ok := c.GetAndDelete("x")
	if ok || v != nil {
		t.Fatal("key a should not exist")
	}

	c.SetDefault("x", 1)

	v, ok = c.GetAndDelete("x")
	if !ok || v.(int) != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	v, ok = c.Get("x")
	if ok || v != nil {
		t.Fatal("key x should be deleted")
	}
}

func TestCache_Delete(t *testing.T) {
	c := New()
	c.Delete("x")

	c.SetForever("x", 1)
	v, ok := c.Get("x")
	if !ok || v.(int) != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	c.Delete("x")

	v, ok = c.Get("x")
	if ok || v != nil {
		t.Fatal("key x should be deleted")
	}
}

func TestCache_DeleteExpired(t *testing.T) {
	var n int64
	testEvictedCallback := func(k string, v interface{}) {
		atomic.AddInt64(&n, v.(int64))
	}
	c := NewDefault(10*time.Millisecond, 5*time.Millisecond, testEvictedCallback)
	for i := 0; i < 10; i++ {
		c.SetDefault(strconv.Itoa(i), int64(i))
	}

	<-time.After(200 * time.Millisecond)
	m := atomic.LoadInt64(&n)
	if m != 45 {
		t.Fatalf("evicted callback executes incorrectly, expected %d, got %d", 45, m)
	}

	v, ok := c.Get("1")
	if ok || v != nil {
		t.Fatal("key 1 should have expired, but was fetched")
	}
}

func countBasedOnRange(c Cache) int {
	size := 0
	c.Range(func(key string, value interface{}) bool {
		size++
		return true
	})
	return size
}

func TestCache_Size(t *testing.T) {
	const numEntries = 1000
	c := New()
	size := c.Count()
	if size != 0 {
		t.Errorf("zero size expected: %d", size)
	}
	expectedSize := 0
	for i := 0; i < numEntries; i++ {
		c.SetDefault(strconv.Itoa(i), i)
		expectedSize++
		size := c.Count()
		if size != expectedSize {
			t.Errorf("size of %d was expected, got: %d", expectedSize, size)
		}
		rsize := countBasedOnRange(c)
		if size != rsize {
			t.Errorf("size does not match number of entries in Range: %v, %v", size, rsize)
		}
	}
	for i := 0; i < numEntries; i++ {
		c.Delete(strconv.Itoa(i))
		expectedSize--
		size := c.Count()
		if size != expectedSize {
			t.Errorf("size of %d was expected, got: %d", expectedSize, size)
		}
		rsize := countBasedOnRange(c)
		if size != rsize {
			t.Errorf("size does not match number of entries in Range: %v, %v", size, rsize)
		}
	}
}

func TestCache_Clear(t *testing.T) {
	const numEntries = 1000
	c := New()
	for i := 0; i < numEntries; i++ {
		c.SetDefault(strconv.Itoa(i), i)
	}
	size := c.Count()
	if size != numEntries {
		t.Errorf("size of %d was expected, got: %d", numEntries, size)
	}
	c.Clear()
	size = c.Count()
	if size != 0 {
		t.Errorf("zero size was expected, got: %d", size)
	}
	rsize := countBasedOnRange(c)
	if rsize != 0 {
		t.Errorf("zero number of entries in Range was expected, got: %d", rsize)
	}
}

func TestCache_Range(t *testing.T) {
	var n int64
	testRange := func(k string, v interface{}) bool {
		atomic.AddInt64(&n, v.(int64))
		return true
	}
	c := New()
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
