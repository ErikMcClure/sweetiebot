package sweetiebot

import "fmt"
import "time"

type Logger interface {
	Log(args ...interface{})
	LogError(msg string, err error)
	Error(message string, channelID string)
}

type Log struct {
	lasterr int64
	info    *GuildInfo
}

func (l *Log) Log(args ...interface{}) {
	s := fmt.Sprint(args...)
	fmt.Printf("[%s] %s\n", time.Now().Format(time.Stamp), s)
	if sb.db != nil && l.info != nil && sb.IsMainGuild(l.info) && sb.db.status.get() {
		sb.db.Audit(AUDIT_TYPE_LOG, nil, s, SBatoi(l.info.ID))
	}
	if l.info != nil && l.info.config.Log.Channel > 0 {
		l.info.SendMessage(SBitoa(l.info.config.Log.Channel), "```\n"+s+"```")
	}
}

func (l *Log) LogError(msg string, err error) {
	if err != nil {
		l.Log(msg, err.Error())
	}
}

func (l *Log) Error(channelID string, message string) {
	if l.info != nil && RateLimit(&l.lasterr, l.info.config.Log.Cooldown) { // Don't print more than one error message every n seconds.
		l.info.SendMessage(channelID, "```\n"+message+"```")
	}
	//l.Log(message); // Always log it to the debug log. TODO: This is really annoying, maybe we shouldn't do this
}
