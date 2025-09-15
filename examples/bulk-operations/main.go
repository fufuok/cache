package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fufuok/cache"
)

func main() {
	// Create a new cache instance
	c := cache.NewDefault[string, string](5*time.Minute, 1*time.Minute)
	defer c.Close()

	fmt.Println("=== Bulk Operations Example ===")

	// Example 1: LoadItems - Load multiple items with same expiration
	fmt.Println("1. LoadItems Example:")
	bulkData := map[string]string{
		"user:1001": "Alice",
		"user:1002": "Bob",
		"user:1003": "Charlie",
		"user:1004": "Diana",
	}

	// Load all items with 2 minutes expiration
	c.LoadItems(bulkData, 2*time.Minute)
	fmt.Printf("Loaded %d users into cache\n", len(bulkData))

	// Verify the data
	for key := range bulkData {
		if value, ok := c.Get(key); ok {
			fmt.Printf("  %s: %s\n", key, value)
		} else {
			log.Printf("Failed to retrieve %s", key)
		}
	}
	fmt.Printf("Total items in cache: %d\n\n", c.Count())

	// Example 2: ItemsWithExpiration - Get all items with their expiration info
	fmt.Println("2. ItemsWithExpiration Example:")
	items := c.ItemsWithExpiration()

	fmt.Println("Current cache contents with expiration times:")
	for key, item := range items {
		if item.Expiration.IsZero() {
			fmt.Printf("  %s: %s (never expires)\n", key, item.Value)
		} else {
			ttl := time.Until(item.Expiration)
			fmt.Printf("  %s: %s (expires in %v)\n", key, item.Value, ttl.Round(time.Second))
		}
	}
	fmt.Println()

	// Example 3: LoadItemsWithExpiration - Load items with individual expiration times
	fmt.Println("3. LoadItemsWithExpiration Example:")

	// Prepare data with different expiration times
	now := time.Now()
	configData := map[string]cache.ItemWithExpiration[string]{
		"config:session_timeout": {
			Value:      "30m",
			Expiration: now.Add(1 * time.Hour), // Expires in 1 hour
		},
		"config:max_connections": {
			Value:      "1000",
			Expiration: now.Add(24 * time.Hour), // Expires in 24 hours
		},
		"config:debug_mode": {
			Value:      "false",
			Expiration: time.Time{}, // Never expires
		},
		"config:api_version": {
			Value:      "v2.1.0",
			Expiration: now.Add(10 * time.Second), // Expires soon for demo
		},
	}

	// Load configuration data
	c.LoadItemsWithExpiration(configData)
	fmt.Printf("Loaded %d configuration items\n", len(configData))

	// Show all items with their expiration info
	fmt.Println("All cache items after loading config:")
	allItems := c.ItemsWithExpiration()
	for key, item := range allItems {
		if item.Expiration.IsZero() {
			fmt.Printf("  %s: %s (never expires)\n", key, item.Value)
		} else {
			ttl := time.Until(item.Expiration)
			if ttl > 0 {
				fmt.Printf("  %s: %s (expires in %v)\n", key, item.Value, ttl.Round(time.Second))
			} else {
				fmt.Printf("  %s: %s (expired)\n", key, item.Value)
			}
		}
	}
	fmt.Printf("Total items in cache: %d\n\n", c.Count())

	// Example 4: Demonstrate expiration behavior
	fmt.Println("4. Expiration Behavior Demo:")
	fmt.Println("Waiting 12 seconds to demonstrate expiration...")
	time.Sleep(12 * time.Second)

	// Check which items are still accessible
	fmt.Println("Items still accessible after waiting:")
	accessibleCount := 0
	for key := range allItems {
		if value, ok := c.Get(key); ok {
			fmt.Printf("  %s: %s\n", key, value)
			accessibleCount++
		}
	}
	fmt.Printf("Accessible items: %d/%d\n", accessibleCount, len(allItems))
	fmt.Printf("Total items in cache: %d\n\n", c.Count())

	// Example 5: Bulk operations with existing data
	fmt.Println("5. Bulk Operations with Existing Data:")

	// Add some individual items first
	c.Set("temp:data1", "temporary1", 30*time.Second)
	c.Set("temp:data2", "temporary2", 30*time.Second)
	fmt.Println("Added 2 temporary items")

	// Load more bulk data
	moreUsers := map[string]string{
		"user:1005": "Eve",
		"user:1006": "Frank",
	}
	c.LoadItems(moreUsers, cache.NoExpiration) // Never expires
	fmt.Printf("Added %d more users (never expire)\n", len(moreUsers))

	// Show final cache state
	fmt.Println("Final cache state:")
	finalItems := c.ItemsWithExpiration()
	for key, item := range finalItems {
		if item.Expiration.IsZero() {
			fmt.Printf("  %s: %s (never expires)\n", key, item.Value)
		} else {
			ttl := time.Until(item.Expiration)
			if ttl > 0 {
				fmt.Printf("  %s: %s (expires in %v)\n", key, item.Value, ttl.Round(time.Second))
			} else {
				fmt.Printf("  %s: %s (expired)\n", key, item.Value)
			}
		}
	}
	fmt.Printf("Final count: %d items\n", c.Count())

	fmt.Println("\n=== Bulk Operations Example Complete ===")
}
