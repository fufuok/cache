package cache

import (
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func mockXsyncMap(cfg ...Config) Cache {
	if len(cfg) == 0 {
		cfg = []Config{
			{
				DefaultExpiration: testDefaultExpiration,
				CleanupInterval:   testCleanupInterval,
			},
		}
	}
	c := newXsyncMap(cfg...)
	for _, x := range testKV {
		c.SetDefault(x.k, x.v)
	}
	c.Set("70ms", 1, 70*time.Millisecond)
	return c
}

func TestXsyncMap_Expire(t *testing.T) {
	c := newXsyncMapDefault(20*time.Millisecond, 1*time.Millisecond)
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

	var v interface{}
	v, ok = c.GetOrSet("e", 6, 50*time.Millisecond)
	if ok || v.(int) != 6 {
		t.Fatalf("key e should not be loaded, expected result is 6, got: %v", v)
	}
	v, ok = c.GetAndSet("e", 7, 150*time.Millisecond)
	if !ok || v.(int) != 6 {
		t.Fatalf("key e should be loaded, expected result is 6, got: %v", v)
	}

	var ttl time.Duration
	v, ttl, ok = c.GetWithTTL("e")
	if !ok || v.(int) != 7 || ttl < 100*time.Millisecond {
		t.Fatalf("key e should be loaded, expected result is 7, got: %v, ttl: %s", v, ttl)
	}
}

func TestXsyncMap_SetAndGet(t *testing.T) {
	c := mockXsyncMap()
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

func TestXsyncMap_SetDefault_WithoutCleanup(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := newXsyncMapDefault(defaultExpiration, 0)
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

func TestXsyncMap_SetDefault(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := newXsyncMapDefault(defaultExpiration, testCleanupInterval)
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

func TestXsyncMap_SetForever(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := newXsyncMapDefault(defaultExpiration, testCleanupInterval)
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

func TestXsyncMap_GetOrSet(t *testing.T) {
	exp := 20 * time.Millisecond
	c := newXsyncMapDefault(20*time.Millisecond, testCleanupInterval)
	v, ok := c.GetOrSet("x", 1, 0)
	if ok {
		t.Fatal("key x should not loaded")
	}
	x, ok := v.(int)
	if !ok || x != 1 {
		t.Fatalf("key x, expected %d, got %v", 1, x)
	}

	time.Sleep(exp * 2)

	v, ok = c.GetOrSet("x", 2, exp)
	if !ok || v.(int) != 1 {
		t.Fatalf("key x, expected %d, got %v", 1, v)
	}

	time.Sleep(exp * 2)

	y, ok := c.Get("x")
	if !ok || y.(int) != 1 {
		t.Fatalf("key x, expected %d, got %v", 1, y)
	}
}

func TestXsyncMap_GetAndSet(t *testing.T) {
	exp := 20 * time.Millisecond
	c := newXsyncMapDefault(20*time.Millisecond, testCleanupInterval)
	v, ok := c.GetAndSet("x", 1, 0)
	if ok {
		t.Fatal("key x should not loaded")
	}
	x, ok := v.(int)
	if !ok || x != 1 {
		t.Fatalf("key x, expected %d, got %v", 1, x)
	}

	time.Sleep(exp * 2)

	v, ok = c.GetAndSet("x", 2, exp)
	if !ok || v.(int) != 1 {
		t.Fatalf("key x, expected %d, got %v", 1, v)
	}

	y, ok := c.Get("x")
	if !ok || y.(int) != 2 {
		t.Fatalf("key x, expected %d, got %v", 2, y)
	}

	time.Sleep(exp * 2)

	_, ok = c.Get("x")
	if ok {
		t.Fatal("key x should not loaded")
	}
}

func TestXsyncMap_GetAndRefresh(t *testing.T) {
	c := newXsyncMapDefault(100*time.Millisecond, testCleanupInterval)
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

	v, ok = c.GetAndRefresh("x", 800*time.Millisecond)
	if !ok || v.(int) != 1 {
		t.Fatalf("expect the result to be true and the value to be 1, got %d", v)
	}

	v, ttl, ok = c.GetWithTTL("x")
	if !ok || v == nil || ttl < 500*time.Millisecond {
		t.Fatalf("key X lifetime is incorrect, expected >= 500ms, got %d", ttl)
	}
	v, tm, ok = c.GetWithExpiration("x")
	if !ok || v == nil || v.(int) != 1 || tm.Before(time.Now()) {
		t.Fatal("failed to get the value and expiration time of key x")
	}

	<-time.After(1 * time.Second)
	v, ok = c.GetAndRefresh("x", 1*time.Second)
	if ok || v != nil {
		t.Fatalf("expect the result to be false and the value to be nil, got %v", v)
	}
	v, ttl, ok = c.GetWithTTL("x")
	if ok || v != nil || ttl != 0 {
		t.Fatalf("expect the result to be false and the value to be nil, got %v, ttl: %s", v, ttl)
	}
}

func TestXsyncMap_GetOrCompute(t *testing.T) {
	const numEntries = 1000
	c := newXsyncMap(Config{MinCapacity: numEntries})
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

func TestXsyncMap_GetOrCompute_WithKeyExpired(t *testing.T) {
	c := newXsyncMap()
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

func TestXsyncMap_GetOrCompute_FunctionCalledOnce(t *testing.T) {
	c := newXsyncMap()
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

func TestXsyncMap_Compute(t *testing.T) {
	var zeroedV interface{}
	c := newXsyncMap()
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

func TestXsyncMap_GetAndDelete(t *testing.T) {
	c := newXsyncMap()
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

func TestXsyncMap_Delete(t *testing.T) {
	c := newXsyncMap()
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

func TestXsyncMap_DeleteExpired(t *testing.T) {
	var n int64
	testEvictedCallback := func(k string, v interface{}) {
		atomic.AddInt64(&n, v.(int64))
	}
	c := newXsyncMapDefault(10*time.Millisecond, 5*time.Millisecond, testEvictedCallback)
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

func TestXsyncMap_Range(t *testing.T) {
	var n int64
	testRange := func(k string, v interface{}) bool {
		atomic.AddInt64(&n, v.(int64))
		return true
	}
	c := newXsyncMap()
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
