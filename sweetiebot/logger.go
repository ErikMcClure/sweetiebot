package sweetiebot

import (
  "fmt"
)

type Logger interface {
  Log(args ...interface{})
  LogError(msg string, err error)
  Error(message string, channelID string)
}

type Log struct {
  bot *SweetieBot
  lasterr int64
}

func (l *Log) Log(args ...interface{}) {
  s := fmt.Sprint(args...)
  fmt.Println(s)
  l.bot.db.Log(s)
  if len(l.bot.LogChannelID) > 0 {
    l.bot.dg.ChannelMessageSend(l.bot.LogChannelID, "```" + s + "```") 
  }
}

func (l *Log) LogError(msg string, err error) {
    if err != nil {
        l.Log(msg, err.Error());
    }
}

func (l *Log) Error(channelID string, message string) {
  if RateLimit(&l.lasterr, sb.config.Maxerror) { // Don't print more than one error message every n seconds.
    l.bot.dg.ChannelMessageSend(channelID, "```" + message + "```") 
  }
  //l.Log(message); // Always log it to the debug log. TODO: This is really annoying, maybe we shouldn't do this
}

func (l *Log) Init(bot *SweetieBot) {
  l.bot = bot
  l.lasterr = 0   
}