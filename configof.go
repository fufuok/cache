//go:build go1.18
// +build go1.18

package cache

import (
	"time"
)

// EvictedCallbackOf callback function to execute when the key-value pair expires and is evicted.
// Warning: cannot block, it is recommended to use goroutine.
type EvictedCallbackOf[K comparable, V any] func(k K, v V)

type ConfigOf[K comparable, V any] struct {
	// DefaultExpiration default expiration time for key-value pairs.
	DefaultExpiration time.Duration

	// CleanupInterval the interval at which expired key-value pairs are automatically cleaned up.
	CleanupInterval time.Duration

	// EvictedCallback executed when the key-value pair expires.
	EvictedCallback EvictedCallbackOf[K, V]

	// MinCapacity specify the initial cache capacity (minimum capacity)
	MinCapacity int
}

func DefaultConfigOf[K comparable, V any]() ConfigOf[K, V] {
	return ConfigOf[K, V]{
		DefaultExpiration: NoExpiration,
		CleanupInterval:   DefaultCleanupInterval,
		EvictedCallback:   nil,
		MinCapacity:       DefaultMinCapacity,
	}
}

// Helper function to set default values.
func configDefaultOf[K comparable, V any](config ...ConfigOf[K, V]) ConfigOf[K, V] {
	if len(config) < 1 {
		return DefaultConfigOf[K, V]()
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
