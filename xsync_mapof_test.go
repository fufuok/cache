//go:build go1.18
// +build go1.18

package cache

import (
	"hash/maphash"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func mockXsyncMapOf(cfg ...ConfigOf[string, any]) CacheOf[string, any] {
	if len(cfg) == 0 {
		cfg = []ConfigOf[string, any]{
			{
				DefaultExpiration: testDefaultExpiration,
				CleanupInterval:   testCleanupInterval,
			},
		}
	}
	c := newXsyncMapOf[any](cfg...)
	for _, x := range testKVOf {
		c.SetDefault(x.k, x.v)
	}
	c.Set("70ms", 1, 70*time.Millisecond)
	return c
}

func TestXsyncMapOf_Expire(t *testing.T) {
	exp := 20 * time.Millisecond
	interval := 1 * time.Millisecond
	cfg := []ConfigOf[string, int]{
		{
			DefaultExpiration: exp,
			CleanupInterval:   interval,
		},
	}
	hasher := func(_ maphash.Seed, k string) uint64 {
		return StrHash64(k)
	}
	caches := []CacheOf[string, int]{
		newXsyncMapOf[int](cfg...),
		newXsyncMapOfDefault[int](exp, interval),
		newXsyncTypedMapOf[string, int](hasher, cfg...),
		newXsyncTypedMapOfDefault[string, int](hasher, exp, interval),
	}
	for _, c := range caches {
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

		var v int
		v, ok = c.GetOrSet("e", 6, 50*time.Millisecond)
		if ok || v != 6 {
			t.Fatalf("key e should not be loaded, expected result is 6, got: %v", v)
		}
		v, ok = c.GetAndSet("e", 7, 150*time.Millisecond)
		if !ok || v != 6 {
			t.Fatalf("key e should be loaded, expected result is 6, got: %v", v)
		}

		var ttl time.Duration
		v, ttl, ok = c.GetWithTTL("e")
		if !ok || v != 7 || ttl < 100*time.Millisecond {
			t.Fatalf("key e should be loaded, expected result is 7, got: %v, ttl: %s", v, ttl)
		}
	}
}

func TestXsyncMapOf_SetAndGet(t *testing.T) {
	c := mockXsyncMapOf()
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

func TestXsyncMapOf_SetDefault(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := newXsyncMapOfDefault[int](defaultExpiration, testCleanupInterval)
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
}

func TestXsyncMapOf_SetForever(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := newXsyncMapOfDefault[int](defaultExpiration, testCleanupInterval)
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

func TestXsyncMapOf_GetOrSet(t *testing.T) {
	exp := 20 * time.Millisecond
	c := newXsyncMapOfDefault[int](exp, testCleanupInterval)
	v, ok := c.GetOrSet("x", 1, 0)
	if ok {
		t.Fatal("key x should not loaded")
	}
	if v != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	time.Sleep(exp * 2)

	v, ok = c.GetOrSet("x", 2, exp)
	if !ok || v != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	time.Sleep(exp * 2)

	y, ok := c.Get("x")
	if !ok || y != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, y)
	}
}

func TestXsyncMapOf_GetAndSet(t *testing.T) {
	exp := 20 * time.Millisecond
	c := newXsyncMapOfDefault[int](exp, testCleanupInterval)
	v, ok := c.GetAndSet("x", 1, 0)
	if ok {
		t.Fatal("key x should not loaded")
	}
	if v != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	time.Sleep(exp * 2)

	v, ok = c.GetAndSet("x", 2, exp)
	if !ok || v != 1 {
		t.Fatalf("key x, expected %d, got %d", 1, v)
	}

	y, ok := c.Get("x")
	if !ok || y != 2 {
		t.Fatalf("key x, expected %d, got %d", 2, y)
	}

	time.Sleep(exp * 2)

	_, ok = c.Get("x")
	if ok {
		t.Fatal("key x should not loaded")
	}
}

func TestXsyncMapOf_GetAndRefresh(t *testing.T) {
	c := newXsyncMapOfDefault[int](100*time.Millisecond, testCleanupInterval)
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

	v, ok = c.GetAndRefresh("x", 800*time.Millisecond)
	if !ok || v != 1 {
		t.Fatalf("expect the result to be true and the value to be 1, got %d", v)
	}

	v, ttl, ok = c.GetWithTTL("x")
	if !ok || v != 1 || ttl < 500*time.Millisecond {
		t.Fatalf("key X lifetime is incorrect, expected >= 500ms, got %d", ttl)
	}
	v, tm, ok = c.GetWithExpiration("x")
	if !ok || v != 1 || tm.Before(time.Now()) {
		t.Fatal("failed to get the value and expiration time of key x")
	}

	<-time.After(1 * time.Second)
	v, ok = c.GetAndRefresh("x", 1*time.Second)
	if ok || v != 0 {
		t.Fatalf("expect the result to be false and the value to be 0, got %d", v)
	}
	v, ttl, ok = c.GetWithTTL("x")
	if ok || v != 0 || ttl != 0 {
		t.Fatalf("expect the result to be false and the value to be 0, got %d, ttl: %s", v, ttl)
	}
}

func TestXsyncMapOf_GetAndDelete(t *testing.T) {
	c := newXsyncMapOf[int]()
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

func TestXsyncMapOf_Delete(t *testing.T) {
	c := newXsyncMapOf[int]()
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

func TestXsyncMapOf_DeleteExpired(t *testing.T) {
	var n int64
	testEvictedCallback := func(k string, v int64) {
		atomic.AddInt64(&n, v)
	}
	c := newXsyncMapOfDefault[int64](10*time.Millisecond, 5*time.Millisecond, testEvictedCallback)
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

func TestXsyncMapOf_Range(t *testing.T) {
	var n int64
	testRange := func(k string, v int64) bool {
		atomic.AddInt64(&n, v)
		return true
	}
	c := newXsyncMapOf[int64]()
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
