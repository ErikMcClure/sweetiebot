package sweetiebot

import (
	"sync/atomic"
)

// AtomicFlag represents an atomic bit that can be set or cleared
type AtomicFlag struct {
	flag uint32
}

// SaturationLimit tracks when events occured and implements a saturation limit on them
type SaturationLimit struct {
	times []int64
	index int
	lock  AtomicFlag
}

func realmod(x int, m int) int {
	x %= m
	if x < 0 {
		x += m
	}
	return x
}

// TestAndSet returns the old value and sets the flag to 1
func (f *AtomicFlag) TestAndSet() bool {
	return atomic.SwapUint32(&f.flag, 1) != 0
}

// Clear sets the flag to 0
func (f *AtomicFlag) Clear() {
	atomic.SwapUint32(&f.flag, 0)
}

func (s *SaturationLimit) append(time int64) {
	for s.lock.TestAndSet() {
	}
	s.index = realmod(s.index+1, len(s.times))
	s.times[s.index] = time
	s.lock.Clear()
}

// Used for our own saturation limits, where we check to see if sending the message would violate our limit BEFORE we actually send it.
func (s *SaturationLimit) check(num int, period int64, curtime int64) bool {
	for s.lock.TestAndSet() {
	}
	i := realmod(s.index-(num-1), len(s.times))
	b := (curtime - s.times[i]) <= period
	s.lock.Clear()
	return b
}

// Used for spam detection, where we always insert the message first (because it's already happened) and THEN check to see if it violated the limit.
func (s *SaturationLimit) checkafter(num int, period int64) bool {
	for s.lock.TestAndSet() {
	}
	i := realmod(s.index-num, len(s.times))
	b := (s.times[s.index] - s.times[i]) <= period
	s.lock.Clear()
	return b
}

func (s *SaturationLimit) resize(size int) {
	for s.lock.TestAndSet() {
	}
	n := make([]int64, size, size)
	copy(n, s.times)
	s.times = n
	s.lock.Clear()
}

// CheckRateLimit performs a check on the rate limit without updating it
func CheckRateLimit(prevtime *int64, interval int64, curtime int64) bool {
	return curtime-(*prevtime) > interval
}

// RateLimit checks the rate limit, returns false if it was violated, and updates the rate limit
func RateLimit(prevtime *int64, interval int64, curtime int64) bool {
	d := (*prevtime) // perform a read so it doesn't change on us
	if curtime-d > interval {
		*prevtime = curtime // CompareAndSwapInt64 doesn't work on x86, temporarily removing this
		return true
		//return atomic.CompareAndSwapInt64(prevtime, d, t) // If the swapped failed, it means another thread already sent a message and swapped it out, so don't send a message.
	}
	return false
}

// AtomicBool represents an atomic boolean that can be set to true or false
type AtomicBool struct {
	flag uint32
}

// Get the current value of the bool
func (b *AtomicBool) Get() bool {
	return atomic.LoadUint32(&b.flag) != 0
}

// Set the value of the bool
func (b *AtomicBool) Set(value bool) {
	var v uint32
	if value {
		v = 1
	}
	atomic.StoreUint32(&b.flag, v)
}
