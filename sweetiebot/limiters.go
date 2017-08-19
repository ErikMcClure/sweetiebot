package sweetiebot

import (
	"sync/atomic"
	"time"
)

type AtomicFlag struct {
	flag uint32
}

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

func (f *AtomicFlag) test_and_set() bool {
	return atomic.SwapUint32(&f.flag, 1) != 0
}

func (f *AtomicFlag) clear() {
	atomic.SwapUint32(&f.flag, 0)
}

func (s *SaturationLimit) append(time int64) {
	for s.lock.test_and_set() {
	}
	s.index = realmod(s.index+1, len(s.times))
	s.times[s.index] = time
	s.lock.clear()
}

// Used for our own saturation limits, where we check to see if sending the message would violate our limit BEFORE we actually send it.
func (s *SaturationLimit) check(num int, period int64, curtime int64) bool {
	for s.lock.test_and_set() {
	}
	i := realmod(s.index-(num-1), len(s.times))
	b := (curtime - s.times[i]) <= period
	s.lock.clear()
	return b
}

// Used for spam detection, where we always insert the message first (because it's already happened) and THEN check to see if it violated the limit.
func (s *SaturationLimit) checkafter(num int, period int64) bool {
	for s.lock.test_and_set() {
	}
	i := realmod(s.index-num, len(s.times))
	b := (s.times[s.index] - s.times[i]) <= period
	s.lock.clear()
	return b
}

func (s *SaturationLimit) resize(size int) {
	for s.lock.test_and_set() {
	}
	n := make([]int64, size, size)
	copy(n, s.times)
	s.times = n
	s.lock.clear()
}

func CheckRateLimit(prevtime *int64, interval int64) bool {
	return time.Now().UTC().Unix()-(*prevtime) > interval
}

func RateLimit(prevtime *int64, interval int64) bool {
	t := time.Now().UTC().Unix()
	d := (*prevtime) // perform a read so it doesn't change on us
	if t-d > interval {
		*prevtime = t // CompareAndSwapInt64 doesn't work on x86, temporarily removing this
		return true
		//return atomic.CompareAndSwapInt64(prevtime, d, t) // If the swapped failed, it means another thread already sent a message and swapped it out, so don't send a message.
	}
	return false
}

func CheckShutup(channel string) bool {
	return true
}

type AtomicBool struct {
	flag uint32
}

func (b *AtomicBool) get() bool {
	return atomic.LoadUint32(&b.flag) != 0
}

func (b *AtomicBool) set(value bool) {
	var v uint32 = 0
	if value {
		v = 1
	}
	atomic.StoreUint32(&b.flag, v)
}
