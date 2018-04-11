package sweetiebot

// RateLimit checks the rate limit, returns false if it was violated, and updates the rate limit
func RateLimit(prevtime *int64, interval int64, curtime int64) bool {
	d := (*prevtime) // perform a read so it doesn't change on us
	if curtime-d > interval {
		*prevtime = curtime // CompareAndSwapInt64 doesn't work on x86 so we just assume this worked
		return true
	}
	return false
}
