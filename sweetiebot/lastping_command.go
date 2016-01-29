package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strconv"
  "time"
)

type LastPingCommand struct {
}

func (c *LastPingCommand) Name() string {
  return "LastPing";  
}
func Pluralize(i int64, s string) string {
  if i == 1 { return strconv.FormatInt(i, 10) + s }
  return strconv.FormatInt(i, 10) + s + "s"
}
func TimeDiff(d time.Duration) string {
  seconds := int64(d.Seconds())
  if seconds <= 60 { return Pluralize(seconds, " second") }
  if seconds <= 60*60 { return Pluralize(seconds/60, " minute") }
  if seconds <= 60*60*24 { return Pluralize(seconds/3600, " hour") }
  return Pluralize(seconds/86400, " day")
}
func (c *LastPingCommand) Process(args []string, user *discordgo.User) string {
  index := 1
  maxrows := 2
  if len(args) > 0 {
    index, _ = strconv.Atoi(args[0])
  }
  if len(args) > 1 {
    maxrows, _ = strconv.Atoi(args[1])
  }
  if index < 1 { index = 1 }
  if maxrows < 0 { maxrows = 0 }
  if maxrows > 3 { maxrows = 3 }
  id, channel := sb.db.GetPing(SBatoi(user.ID), index - 1)
  if id == 0 { return "```No recent pings in the chat log.```" }
  
  after := sb.db.GetPingContext(id, channel, maxrows + 1)
  before := sb.db.GetPingContextBefore(id, channel, maxrows)
  s := "```Pinged " + TimeDiff(time.Now().UTC().Sub(after[0].Timestamp.Add(8*time.Hour))) + " ago, on " + after[0].Timestamp.Format(time.RFC822) + "```\n"
  
  for i := len(before) - 1; i >= 0; i-- {
    s += before[i].Author + ": " + before[i].Message + "\n"
  }
  s += "**" + after[0].Author + ": " + after[0].Message + "**\n"
  for i := 1; i < len(after); i++ {
    s += after[i].Author + ": " + after[i].Message + "\n"
  }
  return s
}
func (c *LastPingCommand) Usage() string { 
  return FormatUsage(c, "[ping index] [max context rows]", "Returns the nth most recent ping (where n is the ping index) in the chat, plus up to [max context rows] messages before and after it. Max context rows is 2 by default and 3 at maximum.") 
}
func (c *LastPingCommand) UsageShort() string { return "Returns the last message that pinged you." }
func (c *LastPingCommand) Roles() []string { return []string{} }
func (c *LastPingCommand) UsePM() bool { return true }