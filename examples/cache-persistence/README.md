# Cache Persistence Example

This example demonstrates how to use `ItemsWithExpiration()` and `LoadItemsWithExpiration()` to persist cache data to JSON files and restore it later.

## Features Demonstrated

1. **Export Cache to JSON**: Use `ItemsWithExpiration()` to get all cache items with their expiration times and serialize to JSON
2. **Import from JSON**: Load JSON data back into cache using `LoadItemsWithExpiration()`
3. **Flexible Loading**: Show different strategies for loading cached data:
   - Load with original expiration times
   - Load with default expiration for all items
   - Load with custom/extended expiration times

## What This Example Shows

- Populating cache with various expiration settings
- Exporting cache contents to a JSON file preserving expiration information
- Clearing cache (simulating application restart)
- Restoring cache from JSON with different expiration strategies
- Handling expired items during restoration

## Key Methods Used

- `cache.ItemsWithExpiration()` - Get all items with expiration info
- `cache.LoadItemsWithExpiration(items)` - Load items with specific expiration times
- `cache.LoadItems(items, expiration)` - Load items with uniform expiration
- JSON serialization/deserialization for persistence

## Usage

```bash
cd examples/cache-persistence
go run main.go
```

The example will create a temporary `cache_backup.json` file during execution and clean it up automatically.
