package cache

import (
	"time"
)

type item[V any] struct {
	v V
	e int64
}

// returns true if the item has expired.
func (i *item[V]) expired() bool {
	return i.e > 0 && time.Now().UnixNano() > i.e
}

// returns true if the item has expired.
func (i *item[V]) expiredWithNow(now int64) bool {
	return i.e > 0 && now > i.e
}
