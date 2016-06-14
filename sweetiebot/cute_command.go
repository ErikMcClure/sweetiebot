package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "strconv"
  "math/rand"
)

type CuteCommand struct {
  int64 lastpic
}

func (c *CuteCommand) Name() string {
  return "Cute";  
}
func (c *CuteCommand) Process(args []string, msg *discordgo.Message) (string, bool) {  
  if len(args) > 0 {
    arg := args[0]
    if !channelregex.MatchString(arg) {
      return "```That's not a channel!```", false
    }
    if !UserHasAnyRole(msg.Author.ID, sb.princessrole) {
      return "```Only mods can specify a channel.```", false
    }
    sb.SendMessage(arg[2:len(arg)-1], MapGetRandomItem(sb.config.CutePics))
  }

  if !RateLimit(&c.lastpic, sb.config.MaxCute) {
    return "```Only one cute pic every " + TimeDiff(time.Duration(sb.config.MaxCute) * time.Second) + ".```"
  }
  return MapGetRandomItem(sb.config.CutePics), false
}
func (c *CuteCommand) Usage() string { 
  return FormatUsage(c, "[channel]", "Posts a cute pony picture. An optional channel argument can be used to send it somewhere else, but is only available to mods.") 
}
func (c *CuteCommand) UsageShort() string { return "Posts a cute pony picture." }
func (c *CuteCommand) Roles() []string { return []string{} }
func (c *CuteCommand) Channels() []string { return []string{} }



type AddSpoilerCommand struct {
  spoilers *SpoilerModule
}

func (c *AddSpoilerCommand) Name() string {
  return "AddSpoiler";  
}
func (c *AddSpoilerCommand) Process(args []string, msg *discordgo.Message) (string, bool) {  
  if len(args) < 1 {
    return "```Nothing specified.```", false
  }
  
  arg := strings.Join(args, " ")
  if len(sb.config.Spoilers) <= 0 {
    sb.config.Spoilers = make(map[string]bool)
  }
  sb.config.Spoilers[arg] = true
  sb.SaveConfig()
  r := c.spoilers.UpdateRegex()
  if !r {
    delete(sb.config.Spoilers, arg)
    c.spoilers.UpdateRegex()
    return "```Failed to ban " + arg + " because regex compilation failed.```", false
  }
  return "```Banned " + arg + " and recompiled the spoiler regex.```", false
}
func (c *AddSpoilerCommand) Usage() string { 
  return FormatUsage(c, "[arbitrary string]", "Adds a line to spoilers (no quotes are required). This is used in a regex, so regex symbols are valid.") 
}
func (c *AddSpoilerCommand) UsageShort() string { return "Adds a line to spoilers." }
func (c *AddSpoilerCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *AddSpoilerCommand) Channels() []string { return []string{} }

type RemoveSpoilerCommand struct {
  spoilers *SpoilerModule
}

func (c *RemoveCuteCommand) Name() string {
  return "RemoveCute";  
}
func (c *RemoveCuteCommand) Process(args []string, msg *discordgo.Message) (string, bool) {  
  if len(args) < 1 {
    return "```Nothing specified.```", false
  }
  arg := strings.Join(args, " ")
  _, ok := sb.config.Spoilers[arg]
  if !ok {
    return "```Could not find " + arg + "!```", false
  }
  delete(sb.config.Spoilers, arg)
  sb.SaveConfig()
  c.spoilers.UpdateRegex()
  return "```Unspoilered " + arg + " and recompiled the spoiler regex.```", false
}
func (c *RemoveCuteCommand) Usage() string { 
  return FormatUsage(c, "[arbitrary string]", "Removes a line from spoilers (no quotes are required).") 
}
func (c *RemoveCuteCommand) UsageShort() string { return "Removes a line from spoilers." }
func (c *RemoveCuteCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *RemoveCuteCommand) Channels() []string { return []string{} }