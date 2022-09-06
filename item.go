package cache

import (
	"time"
)

type item struct {
	v interface{}
	e int64
}

// returns true if the item has expired.
func (i *item) expired() bool {
	return i.e > 0 && time.Now().UnixNano() > i.e
}

// returns true if the item has expired.
func (i *item) expiredWithNow(now int64) bool {
	return i.e > 0 && now > i.e
}
