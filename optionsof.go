//go:build go1.18
// +build go1.18

package cache

import (
	"time"
)

type OptionOf[V any] func(config *ConfigOf[V])

func WithDefaultExpirationOf[V any](duration time.Duration) OptionOf[V] {
	return func(config *ConfigOf[V]) {
		config.DefaultExpiration = duration
	}
}

func WithCleanupIntervalOf[V any](interval time.Duration) OptionOf[V] {
	return func(config *ConfigOf[V]) {
		config.CleanupInterval = interval
	}
}

func WithEvictedCallbackOf[V any](ec EvictedCallbackOf[V]) OptionOf[V] {
	return func(config *ConfigOf[V]) {
		config.EvictedCallback = ec
	}
}
