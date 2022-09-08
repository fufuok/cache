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
	c := NewXsyncMap(cfg...)
	for _, x := range testKV {
		c.SetDefault(x.k, x.v)
	}
	c.Set("70ms", 1, 70*time.Millisecond)
	return c
}

func TestXsyncMap_Expire(t *testing.T) {
	c := NewXsyncMapDefault(20*time.Millisecond, 1*time.Millisecond)
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

func TestXsyncMap_SetDefault(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := NewXsyncMapDefault(defaultExpiration, testCleanupInterval)
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
}

func TestXsyncMap_SetForever(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := NewXsyncMapDefault(defaultExpiration, testCleanupInterval)
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
	c := NewXsyncMap()
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

func TestXsyncMap_GetAndSet(t *testing.T) {
	c := NewXsyncMap()
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

func TestXsyncMap_GetAndRefresh(t *testing.T) {
	c := NewXsyncMapDefault(100*time.Millisecond, testCleanupInterval)
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

func TestXsyncMap_GetAndDelete(t *testing.T) {
	c := NewXsyncMap()
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
	c := NewXsyncMap()
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
	c := NewXsyncMapDefault(10*time.Millisecond, 5*time.Millisecond, testEvictedCallback)
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
	c := NewXsyncMap()
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
