package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
)

type LastSeenCommand struct {
}

func (c *LastSeenCommand) Name() string {
  return "LastSeen";  
}
func (c *LastSeenCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  id, fail := ReadUserPingArg(args)
  if fail != "" { return fail, false }
  u, lastseen := sb.db.GetUser(id)
  return "```" + u.Username + " last seen " + TimeDiff(SinceUTC(lastseen)) + " ago.```", false
}
func (c *LastSeenCommand) Usage() string { 
  return FormatUsage(c, "[@user]", "Returns when a user was last seen on discord, which is usually their last status change.") 
}
func (c *LastSeenCommand) UsageShort() string { return "Returns when a user was last seen." }
func (c *LastSeenCommand) Roles() []string { return []string{} }
func (c *LastSeenCommand) Channels() []string { return []string{} }