package spammodule

import (
	"container/heap"
	"testing"
	"time"

	bot "github.com/erikmcclure/sweetiebot/sweetiebot"
)

func TestHeap(t *testing.T) {
	t.Parallel()

	spam := New()

	insert := []int64{1000, 1300, 1400, 1100, 1200}
	for _, v := range insert {
		heap.Push(spam.timeouts, userTimeout{bot.DiscordUser(""), time.Unix(v, 0)})
	}

	out := []int64{1000, 1100, 1200, 1300, 1400}
	for _, v := range out {
		if heap.Pop(spam.timeouts).(userTimeout).time != time.Unix(v, 0) {
			t.Error(v)
		}
	}

	tmp := userTimeout{bot.DiscordUser(""), time.Unix(900, 0)}
	heap.Push(spam.timeouts, tmp)
	if heap.Pop(spam.timeouts).(userTimeout).time != tmp.time {
		t.Error("900 did not match")
	}
}
