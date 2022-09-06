//go:build go1.18
// +build go1.18

package cache

import (
	"time"
)

type itemOf[V any] struct {
	v V
	e int64
}

// returns true if the item has expired.
func (i *itemOf[V]) expired() bool {
	return i.e > 0 && time.Now().UnixNano() > i.e
}

// returns true if the item has expired.
func (i *itemOf[V]) expiredWithNow(now int64) bool {
	return i.e > 0 && now > i.e
}
