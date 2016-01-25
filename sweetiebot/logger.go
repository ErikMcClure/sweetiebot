package sweetiebot

import (
  "fmt"
)

type Logger interface {
  Log(args ...interface{})
  LogError(msg string, err error)
  Error(message string)
}

type Log struct {
  bot *SweetieBot
  lasterr uint64
}

func (l *Log) Log(args ...interface{}) {
  s := fmt.Sprint(args...)
  fmt.Println(s)
  l.bot.db.Log(s)
  if len(l.bot.LogChannelID) > 0 { 
    fmt.Println("trying to log discord") 
    l.bot.dg.ChannelMessageSend(l.bot.LogChannelID, s) 
  }
}

func (l *Log) LogError(msg string, err error) {
    if err != nil {
        l.Log(msg, err.Error());
    }
}

func (l *Log) Error(message string) {
  if l.lasterr > 0 { // Check if we have not emitted an error message for 5 seconds.
  
    // Emit the error message to general chat
  }
  l.Log(message); // Always log it to the debug log.
}

func (l *Log) Init(bot *SweetieBot) {
  l.bot = bot
  l.lasterr = 0   
}