package cache

import (
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func mockXsyncMap(cfg ...Config[string, any]) Cache[string, any] {
	if len(cfg) == 0 {
		cfg = []Config[string, any]{
			{
				DefaultExpiration: testDefaultExpiration,
				CleanupInterval:   testCleanupInterval,
			},
		}
	}
	c := newXsyncMap[string, any](cfg...)
	for _, x := range testKV {
		c.SetDefault(x.k, x.v)
	}
	c.Set("70ms", 1, 70*time.Millisecond)
	return c
}

func TestXsyncMap_Expire(t *testing.T) {
	exp := 20 * time.Millisecond
	interval := 1 * time.Millisecond
	cfg := []Config[string, int]{
		{
			DefaultExpiration: exp,
			CleanupInterval:   interval,
		},
	}
	caches := []Cache[string, int]{
		newXsyncMap[string, int](cfg...),
		newXsyncMapDefault[string, int](exp, interval),
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
	c := newXsyncMapDefault[string, int](defaultExpiration, 0)
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

func TestXsyncMap_SetDefault(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := newXsyncMapDefault[string, int](defaultExpiration, testCleanupInterval)
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

func TestXsyncMap_SetForever(t *testing.T) {
	defaultExpiration := 50 * time.Millisecond
	c := newXsyncMapDefault[string, int](defaultExpiration, testCleanupInterval)
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

func TestXsyncMap_GetOrSet(t *testing.T) {
	exp := 20 * time.Millisecond
	c := newXsyncMapDefault[string, int](exp, testCleanupInterval)
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

func TestXsyncMap_GetAndSet(t *testing.T) {
	exp := 20 * time.Millisecond
	c := newXsyncMapDefault[string, int](exp, testCleanupInterval)
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

func TestXsyncMap_GetAndRefresh(t *testing.T) {
	c := newXsyncMapDefault[string, int](100*time.Millisecond, testCleanupInterval)
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

func TestXsyncMap_GetOrCompute(t *testing.T) {
	const numEntries = 1000
	c := newXsyncMap[string, int](Config[string, int]{MinCapacity: numEntries})
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

func TestXsyncMap_GetOrCompute_WithKeyExpired(t *testing.T) {
	c := newXsyncMap[string, int]()
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
}

func TestXsyncMap_GetOrCompute_FunctionCalledOnce(t *testing.T) {
	c := newXsyncMap[int, int]()
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

func TestXsyncMap_Compute(t *testing.T) {
	c := newXsyncMap[string, int]()
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

func TestXsyncMap_GetAndDelete(t *testing.T) {
	c := newXsyncMap[string, int]()
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

func TestXsyncMap_Delete(t *testing.T) {
	c := newXsyncMap[string, int]()
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

func TestXsyncMap_DeleteExpired(t *testing.T) {
	var n int64
	testEvictedCallback := func(k string, v int64) {
		atomic.AddInt64(&n, v)
	}
	c := newXsyncMapDefault[string, int64](10*time.Millisecond, 5*time.Millisecond, testEvictedCallback)
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

func TestXsyncMap_Range(t *testing.T) {
	var n int64
	testRange := func(k string, v int64) bool {
		atomic.AddInt64(&n, v)
		return true
	}
	c := newXsyncMap[string, int64]()
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

func TestXsyncMap_ItemsWithExpiration(t *testing.T) {
	c := newXsyncMapDefault[string, int](100*time.Millisecond, testCleanupInterval)

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

func TestXsyncMap_LoadItems(t *testing.T) {
	c := newXsyncMap[string, int]()

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

func TestXsyncMap_LoadItemsWithExpiration(t *testing.T) {
	c := newXsyncMap[string, string]()

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

func TestXsyncMap_LoadItems_EdgeCases(t *testing.T) {
	c := newXsyncMap[string, int]()

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

func TestXsyncMap_LoadItemsWithExpiration_EdgeCases(t *testing.T) {
	c := newXsyncMap[string, int]()

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
