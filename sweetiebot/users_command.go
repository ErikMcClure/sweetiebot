package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strconv"
  "time"
  "strings"
)

type NewUsersCommand struct {
}

func (c *NewUsersCommand) Name() string {
  return "newusers";  
}
func (c *NewUsersCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  maxresults := 5
  if len(args) > 0 { maxresults, _ = strconv.Atoi(args[0]) }
  if maxresults < 1 { return "```How I return no results???```", false }
  if maxresults > 30 { maxresults = 30 }
  r := sb.db.GetNewestUsers(maxresults)
  s := make([]string, 0, len(r))
  
  for _, v := range r {
    s = append(s, v.Username + "  (joined: " + v.FirstSeen.Format(time.ANSIC) + ")") 
  }
  return "```" + strings.Join(s, "\n") + "```", true
}
func (c *NewUsersCommand) Usage() string { 
  return FormatUsage(c, "[maxresults]", "Lists up to maxresults users, starting with the newest user to join the server. Defaults to 5 results, returns a maximum of 30.") 
}
func (c *NewUsersCommand) UsageShort() string { return "[PM Only] Gets a list of the most recent users to join the server." }
func (c *NewUsersCommand) Roles() []string { return []string{} }

type AKACommand struct {
}

func (c *AKACommand) Name() string {
  return "aka";  
}
func (c *AKACommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  id, fail := ReadUserPingArg(args)
  if fail != "" { return fail, false }
  r := sb.db.GetAliases(id)
  u, _ := sb.db.GetUser(id)
  return "```All known aliases for " + u.Username + "\n  " + strings.Join(r, "\n  ") + "```", !CheckShutup(msg.ChannelID)
}
func (c *AKACommand) Usage() string { 
  return FormatUsage(c, "[@user]", "Lists all known aliases of the user in question, up to a maximum of 10, with the names used the longest first.") 
}
func (c *AKACommand) UsageShort() string { return "Lists all known aliases of a user." }
func (c *AKACommand) Roles() []string { return []string{} }