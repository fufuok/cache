package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fufuok/cache"
)

func main() {
	fmt.Println("=== Cache Persistence Example ===")

	// Create a cache with 5 minute default expiration
	c := cache.NewDefault[string, string](5*time.Minute, 1*time.Minute)
	defer c.Close()

	// Step 1: Populate the cache with some data
	fmt.Println("1. Populating cache with sample data...")
	c.Set("user:1001", "Alice Johnson", cache.DefaultExpiration)
	c.Set("user:1002", "Bob Smith", cache.DefaultExpiration)
	c.Set("user:1003", "Charlie Brown", cache.NoExpiration) // Never expires
	c.Set("config:timeout", "30s", 2*time.Minute)           // Custom expiration
	c.Set("session:abc123", "active", 10*time.Second)       // Short expiration for demo

	fmt.Printf("Cache populated with %d items\n", c.Count())

	// Show current cache contents
	fmt.Println("\nCurrent cache contents:")
	items := c.ItemsWithExpiration()
	for key, item := range items {
		if item.Expiration.IsZero() {
			fmt.Printf("  %s: %s (never expires)\n", key, item.Value)
		} else {
			ttl := time.Until(item.Expiration)
			fmt.Printf("  %s: %s (expires in %v)\n", key, item.Value, ttl.Round(time.Second))
		}
	}

	// Step 2: Export cache to JSON file
	fmt.Println("\n2. Exporting cache to JSON file...")
	cacheData := c.ItemsWithExpiration()

	jsonData, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal cache data: %v", err)
	}

	filename := "cache_backup.json"
	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		log.Fatalf("Failed to write cache file: %v", err)
	}

	fmt.Printf("Cache exported to %s\n", filename)
	fmt.Printf("File size: %d bytes\n", len(jsonData))

	// Show JSON content (first 500 characters)
	fmt.Println("\nJSON content preview:")
	preview := string(jsonData)
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	fmt.Println(preview)

	// Step 3: Clear the cache and simulate restart
	fmt.Println("\n3. Simulating application restart...")
	c.Clear()
	fmt.Printf("Cache cleared. Current item count: %d\n", c.Count())

	// Wait a moment to demonstrate time passage
	fmt.Println("Waiting 3 seconds to simulate time passage...")
	time.Sleep(3 * time.Second)

	// Step 4: Load cache from JSON file with default expiration
	fmt.Println("\n4. Loading cache from JSON file...")

	// Read JSON file
	jsonData, err = os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read cache file: %v", err)
	}

	// Parse JSON into cache items
	var loadedItems map[string]cache.ItemWithExpiration[string]
	err = json.Unmarshal(jsonData, &loadedItems)
	if err != nil {
		log.Fatalf("Failed to unmarshal cache data: %v", err)
	}

	fmt.Printf("Loaded %d items from JSON\n", len(loadedItems))

	// Method 1: Load with original expiration times
	fmt.Println("\nMethod 1: Loading with original expiration times...")
	c.LoadItemsWithExpiration(loadedItems)
	fmt.Printf("Cache restored with %d items\n", c.Count())

	// Show restored cache contents
	fmt.Println("Restored cache contents:")
	restoredItems := c.ItemsWithExpiration()
	for key, item := range restoredItems {
		if item.Expiration.IsZero() {
			fmt.Printf("  %s: %s (never expires)\n", key, item.Value)
		} else {
			ttl := time.Until(item.Expiration)
			if ttl > 0 {
				fmt.Printf("  %s: %s (expires in %v)\n", key, item.Value, ttl.Round(time.Second))
			} else {
				fmt.Printf("  %s: %s (already expired)\n", key, item.Value)
			}
		}
	}

	// Step 5: Alternative approach - Load with default expiration
	fmt.Println("\n5. Alternative: Loading all items with default expiration...")
	c.Clear()

	// Convert ItemWithExpiration to simple map and load with default expiration
	simpleItems := make(map[string]string)
	for key, item := range loadedItems {
		simpleItems[key] = item.Value
	}

	c.LoadItems(simpleItems, cache.DefaultExpiration)
	fmt.Printf("Cache reloaded with default expiration: %d items\n", c.Count())

	// Show cache with default expiration
	fmt.Println("Cache contents with default expiration:")
	defaultItems := c.ItemsWithExpiration()
	for key, item := range defaultItems {
		ttl := time.Until(item.Expiration)
		fmt.Printf("  %s: %s (expires in %v)\n", key, item.Value, ttl.Round(time.Second))
	}

	// Step 6: Demonstrate selective loading
	fmt.Println("\n6. Selective loading example...")
	c.Clear()

	// Only load non-expired items with extended expiration
	selectiveItems := make(map[string]cache.ItemWithExpiration[string])
	now := time.Now()

	for key, item := range loadedItems {
		// Skip items that were already expired or extend expiration for all
		selectiveItems[key] = cache.ItemWithExpiration[string]{
			Value:      item.Value,
			Expiration: now.Add(10 * time.Minute), // Give all items 10 minutes
		}
	}

	c.LoadItemsWithExpiration(selectiveItems)
	fmt.Printf("Selectively loaded %d items with extended expiration\n", c.Count())

	// Show final state
	fmt.Println("Final cache state:")
	finalItems := c.ItemsWithExpiration()
	for key, item := range finalItems {
		ttl := time.Until(item.Expiration)
		fmt.Printf("  %s: %s (expires in %v)\n", key, item.Value, ttl.Round(time.Second))
	}

	// Cleanup
	fmt.Println("\n7. Cleanup...")
	err = os.Remove(filename)
	if err != nil {
		log.Printf("Warning: Failed to remove backup file: %v", err)
	} else {
		fmt.Printf("Removed %s\n", filename)
	}

	fmt.Println("\n=== Cache Persistence Example Complete ===")
}
