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
func (c *NewUsersCommand) Channels() []string { return []string{} }

type AKACommand struct {
}

func (c *AKACommand) Name() string {
  return "aka";  
}
func (c *AKACommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    return "```You must provide a user to search for.```", false
  }
  arg := strings.Join(args, " ")
  var id uint64
  if userregex.MatchString(arg) {
    id = SBatoi(arg[2:len(arg)-1])   
  } else {
      IDs := sb.db.FindUsers("%" + arg + "%", 20, 0)
      if len(IDs) == 0 { // no matches!
        return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
      }
      if len(IDs) > 1 {
        s := []string{}
        
        for _, v := range IDs {
          u, _ := sb.db.GetUser(v)
          s = append(s, u.Username)
        }
        
        return "```Could be any of the following users or their aliases:\n" + strings.Join(s, "\n") + "```", len(s) > 5
      }
      id = IDs[0]
  }
  
  r := sb.db.GetAliases(id)
  u, _ := sb.db.GetUser(id)
  return "```All known aliases for " + u.Username + "\n  " + strings.Join(r, "\n  ") + "```", !CheckShutup(msg.ChannelID)
}
func (c *AKACommand) Usage() string { 
  return FormatUsage(c, "[@user]", "Lists all known aliases of the user in question, up to a maximum of 10, with the names used the longest first.") 
}
func (c *AKACommand) UsageShort() string { return "Lists all known aliases of a user." }
func (c *AKACommand) Roles() []string { return []string{} }
func (c *AKACommand) Channels() []string { return []string{} }