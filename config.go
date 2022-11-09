package cache

import (
	"time"
)

const (
	// NoExpiration mark cached item never expire.
	NoExpiration = -2 * time.Second

	// DefaultExpiration use the default expiration time set when the cache was created.
	// Equivalent to passing in the same e duration as was given to NewCache() or NewCacheDefault().
	DefaultExpiration = -1 * time.Second

	// DefaultCleanupInterval the default time interval for automatically cleaning up expired key-value pairs
	DefaultCleanupInterval = 10 * time.Second

	// DefaultMinCapacity specify the initial cache capacity (minimum capacity)
	DefaultMinCapacity = 32 * 3
)

// EvictedCallback callback function to execute when the key-value pair expires and is evicted.
// Warning: cannot block, it is recommended to use goroutine.
type EvictedCallback func(k string, v interface{})

type Config struct {
	// DefaultExpiration default expiration time for key-value pairs.
	DefaultExpiration time.Duration

	// CleanupInterval the interval at which expired key-value pairs are automatically cleaned up.
	CleanupInterval time.Duration

	// EvictedCallback executed when the key-value pair expires.
	EvictedCallback EvictedCallback

	// MinCapacity specify the initial cache capacity (minimum capacity)
	MinCapacity int
}

func DefaultConfig() Config {
	return Config{
		DefaultExpiration: NoExpiration,
		CleanupInterval:   DefaultCleanupInterval,
		EvictedCallback:   nil,
		MinCapacity:       DefaultMinCapacity,
	}
}

// Helper function to set default values.
func configDefault(config ...Config) Config {
	if len(config) < 1 {
		return DefaultConfig()
	}

	cfg := config[0]

	if cfg.DefaultExpiration < 1 {
		cfg.DefaultExpiration = NoExpiration
	}
	if cfg.CleanupInterval < 0 {
		cfg.CleanupInterval = 0
	}
	if cfg.MinCapacity < DefaultMinCapacity {
		cfg.MinCapacity = DefaultMinCapacity
	}

	return cfg
}
