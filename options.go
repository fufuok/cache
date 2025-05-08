package cache

import (
	"time"
)

type Option[K comparable, V any] func(config *Config[K, V])

func WithDefaultExpiration[K comparable, V any](duration time.Duration) Option[K, V] {
	return func(config *Config[K, V]) {
		config.DefaultExpiration = duration
	}
}

func WithCleanupInterval[K comparable, V any](interval time.Duration) Option[K, V] {
	return func(config *Config[K, V]) {
		config.CleanupInterval = interval
	}
}

func WithEvictedCallback[K comparable, V any](ec EvictedCallback[K, V]) Option[K, V] {
	return func(config *Config[K, V]) {
		config.EvictedCallback = ec
	}
}

func WithMinCapacity[K comparable, V any](sizeHint int) Option[K, V] {
	return func(config *Config[K, V]) {
		config.MinCapacity = sizeHint
	}
}
