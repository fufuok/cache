//go:build go1.18
// +build go1.18

package cache

import (
	"time"
)

type OptionOf[K comparable, V any] func(config *ConfigOf[K, V])

func WithDefaultExpirationOf[K comparable, V any](duration time.Duration) OptionOf[K, V] {
	return func(config *ConfigOf[K, V]) {
		config.DefaultExpiration = duration
	}
}

func WithCleanupIntervalOf[K comparable, V any](interval time.Duration) OptionOf[K, V] {
	return func(config *ConfigOf[K, V]) {
		config.CleanupInterval = interval
	}
}

func WithEvictedCallbackOf[K comparable, V any](ec EvictedCallbackOf[K, V]) OptionOf[K, V] {
	return func(config *ConfigOf[K, V]) {
		config.EvictedCallback = ec
	}
}

func WithMinCapacityOf[K comparable, V any](sizeHint int) OptionOf[K, V] {
	return func(config *ConfigOf[K, V]) {
		config.MinCapacity = sizeHint
	}
}
