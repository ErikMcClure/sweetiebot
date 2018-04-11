package sweetiebot

import (
	"sync/atomic"
)

// RateLimit checks the rate limit, returns false if it was violated, and updates the rate limit
func RateLimit(prevtime *int64, interval int64, curtime int64) bool {
	d := (*prevtime) // perform a read so it doesn't change on us
	if curtime-d > interval {
		return atomic.CompareAndSwapInt64(prevtime, d, curtime)
	}
	return false
}
