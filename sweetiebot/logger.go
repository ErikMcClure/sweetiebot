package sweetiebot

import (
	"fmt"
	"time"
)

type logger interface {
	Log(args ...interface{})
	LogError(msg string, err error)
}

type emptyLog struct{}

func (log *emptyLog) Log(args ...interface{}) {
	s := fmt.Sprint(args...)
	fmt.Printf("[%s] %s\n", time.Now().Format(time.Stamp), s)
}

func (log *emptyLog) LogError(msg string, err error) {
	if err != nil {
		log.Log(msg, err.Error())
	}
}
