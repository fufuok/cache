//go:build go1.18
// +build go1.18

package main

import (
	"fmt"
	"time"

	"github.com/fufuok/cache"
)

func OnEvicted(k string, v int) {
	fmt.Printf("Evicted: K -> %s | V -> %d\n", k, v)
}

func main() {
	c := cache.NewOf[string, int](
		cache.WithDefaultExpirationOf[string, int](1*time.Second),
		cache.WithCleanupIntervalOf[string, int](500*time.Millisecond),
		cache.WithEvictedCallbackOf[string, int](OnEvicted),
	)
	// or:
	c = cache.NewOfDefault(1*time.Second, 500*time.Millisecond, OnEvicted)

	c.Set("A", 1, 1*time.Second)

	val, ok := c.Get("A")
	// A 1 true
	fmt.Println("A", val, ok)

	val, ok = c.GetOrSet("B", 2, 1*time.Second)
	// B 2 false
	fmt.Println("B", val, ok)

	val, ok = c.GetAndSet("B", 3, 1*time.Second)
	// B 2 true
	fmt.Println("B", val, ok)

	val, ok = c.GetAndDelete("B")
	// B 3 true
	fmt.Println("B", val, ok)

	c.SetDefault("C", 2)
	c.SetForever("D", 3)

	time.Sleep(2 * time.Second)

	c.Range(func(key string, value int) bool {
		fmt.Printf("Key -> %s | Value -> %d\n", key, value)
		return true
	})

	c.Delete("D")
	// delete is safe even if a key doesn't exists
	c.Delete("A")
	c.Delete("B")
	c.Delete("C")
	c.Delete("E")

	if c.Count() == 0 {
		fmt.Println("cleanup complete")
	}
}

// Output:
// A 1 true
// B 2 false
// B 2 true
// Evicted: K -> B | V -> 3
// B 3 true
// Evicted: K -> C | V -> 2
// Evicted: K -> A | V -> 1
// Key -> D | Value -> 3
// Evicted: K -> D | V -> 3
// cleanup complete
