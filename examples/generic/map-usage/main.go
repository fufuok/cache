//go:build go1.18
// +build go1.18

package main

import (
	"fmt"

	"github.com/fufuok/cache"
)

func main() {
	m := cache.NewMapOf[int]()

	m.Store("A", 1)

	val, ok := m.Load("A")
	// A 1 true
	fmt.Println("A", val, ok)

	val, ok = m.LoadOrStore("B", 2)
	// B 2 false
	fmt.Println("B", val, ok)

	val, ok = m.LoadAndStore("B", 3)
	// B 2 true
	fmt.Println("B", val, ok)

	val, ok = m.LoadAndDelete("B")
	// B 3 true
	fmt.Println("B", val, ok)

	m.Store("C", 4)
	m.Store("C", 5)
	m.LoadOrStore("D", 6)

	m.Range(func(key string, value int) bool {
		fmt.Printf("Key -> %s | Value -> %d\n", key, value)
		return true
	})

	m.Delete("A")
	m.Delete("C")
	m.Delete("D")
	// delete is safe even if a key doesn't exists
	m.Delete("B")
	m.Delete("E")

	if m.Size() == 0 {
		fmt.Println("map size is 0")
	}
}

// Output:
// A 1 true
// B 2 false
// B 2 true
// B 3 true
// Key -> D | Value -> 6
// Key -> A | Value -> 1
// Key -> C | Value -> 5
// map size is 0
