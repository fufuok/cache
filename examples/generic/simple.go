//go:build go1.18
// +build go1.18

package main

import (
	"fmt"
	"time"

	"github.com/fufuok/cache"
)

func main() {
	c := cache.NewOf[int]()

	c.Set("A", 1, 1*time.Minute)

	val, ok := c.Get("A")
	// A 1 true
	fmt.Println("A", val, ok)

	val, ok = c.GetOrSet("B", 2, 1*time.Minute)
	// B 2 false
	fmt.Println("B", val, ok)

	val, ok = c.GetAndSet("B", 3, 1*time.Minute)
	// B 2 true
	fmt.Println("B", val, ok)

	val, ok = c.GetAndDelete("B")
	// B 3 true
	fmt.Println("B", val, ok)

	c.SetDefault("C", 2)
	c.SetForever("D", 3)

	c.Range(func(key string, value int) bool {
		fmt.Printf("Key -> %s | Value -> %d\n", key, value)
		return true
	})

	c.Delete("A")
	c.Delete("C")
	c.Delete("D")
	// delete is safe even if a key doesn't exists
	c.Delete("B")
	c.Delete("E")

	if c.Count() == 0 {
		fmt.Println("cleanup complete")
	}
}

// Output:
// A 1 true
// B 2 false
// B 2 true
// B 3 true
// Key -> D | Value -> 3
// Key -> C | Value -> 2
// Key -> A | Value -> 1
// cleanup complete
