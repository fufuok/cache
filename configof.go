//go:build go1.18
// +build go1.18

package cache

import (
	"time"
)

// EvictedCallbackOf callback function to execute when the key-value pair expires and is evicted.
// Warning: cannot block, it is recommended to use goroutine.
type EvictedCallbackOf[V any] func(k string, v V)

type ConfigOf[V any] struct {
	// DefaultExpiration default expiration time for key-value pairs.
	DefaultExpiration time.Duration

	// CleanupInterval the interval at which expired key-value pairs are automatically cleaned up.
	CleanupInterval time.Duration

	// EvictedCallback executed when the key-value pair expires.
	EvictedCallback EvictedCallbackOf[V]
}

func DefaultConfigOf[V any]() ConfigOf[V] {
	return ConfigOf[V]{
		DefaultExpiration: NoExpiration,
		CleanupInterval:   DefaultCleanupInterval,
		EvictedCallback:   nil,
	}
}

// Helper function to set default values.
func configDefaultOf[V any](config ...ConfigOf[V]) ConfigOf[V] {
	if len(config) < 1 {
		return DefaultConfigOf[V]()
	}

	cfg := config[0]

	if cfg.DefaultExpiration < 1 {
		cfg.DefaultExpiration = NoExpiration
	}
	if cfg.CleanupInterval < 0 {
		cfg.CleanupInterval = 0
	}

	return cfg
}
