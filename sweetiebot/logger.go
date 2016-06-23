package sweetiebot

import (
  "fmt"
  "strconv"
)

type Logger interface {
  Log(args ...interface{})
  LogError(msg string, err error)
  Error(message string, channelID string)
}

type Log struct {
  lasterr int64
}

func (l *Log) Log(args ...interface{}) {
  s := fmt.Sprint(args...)
  fmt.Println(s)
  if sb.db != nil {
    sb.db.Log(s)
    if sb.config.LogChannel > 0 {
      sb.SendMessage(strconv.FormatUint(sb.config.LogChannel, 10), "```" + s + "```") 
    }
  }
}

func (l *Log) LogError(msg string, err error) {
    if err != nil {
        l.Log(msg, err.Error());
    }
}

func (l *Log) Error(channelID string, message string) {
  if RateLimit(&l.lasterr, sb.config.Maxerror) { // Don't print more than one error message every n seconds.
    sb.SendMessage(channelID, "```" + message + "```") 
  }
  //l.Log(message); // Always log it to the debug log. TODO: This is really annoying, maybe we shouldn't do this
}