package cache

import (
	"time"
)

type Option func(config *Config)

func WithDefaultExpiration(duration time.Duration) Option {
	return func(config *Config) {
		config.DefaultExpiration = duration
	}
}

func WithCleanupInterval(interval time.Duration) Option {
	return func(config *Config) {
		config.CleanupInterval = interval
	}
}

func WithEvictedCallback(ec EvictedCallback) Option {
	return func(config *Config) {
		config.EvictedCallback = ec
	}
}
