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

// experimental ban command for admins to ban users from the server with extreme prejudice
type BanCommand struct{
}

func (c *BanCommand) Name() string{
  return "ban";
}

func (c *BanCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  // make sure we passed a valid argument to the command
  if len(args) < 1 {
    return "```You didn't tell me who to zap with the friendship gun, silly.```", false
  }
  // get the user ID and deal with Discord's alias bullshit
  arg := strings.Join(args, " ")
  var id uint64
  if userregex.MatchString(arg) {
    id = SBatoi(arg[2:len(arg)-1])
  } else {
    IDs := sb.db.FindUsers("%" + arg + "%", 20, 0) // how exactly does this work?
    if len(IDs) == 0 { // no matches
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
  // we're done with our checks
  // actually ban the user here and send the output. This is probably poorly done.
  gID := sb.Guild.ID
  u, _ := sb.db.GetUser(id)
  uID := strconv.FormatUint(id, 10)
  sb.dg.GuildBanCreate(gID, uID, 1)
  
  return "```Banned " + u.Username + " from the server. Harmony restored.```",  !CheckShutup(msg.ChannelID)
}
func (c *BanCommand) Usage() string {
  return FormatUsage(c, "[@user]", "Commands Sweetie Bot to ban a given user.")
}
func (c *BanCommand) UsageShort() string { return "Commands Sweetie Bot to ban a given user." }