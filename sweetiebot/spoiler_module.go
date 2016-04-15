package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "regexp"
)

// This module picks a random action to do whenever #example has been idle for several minutes (configurable)
type SpoilerModule struct {
  ModuleEnabled
  spoilerban *regexp.Regexp
  lastmsg int64 // Sanity rate limiter
}

func (w *SpoilerModule) Name() string {
  return "Spoiler"
}

func (w *SpoilerModule) Register(hooks *ModuleHooks) {
  w.lastmsg = 0
  w.UpdateRegex()
  hooks.OnMessageCreate = append(hooks.OnMessageCreate, w)
  hooks.OnMessageUpdate = append(hooks.OnMessageUpdate, w)
  hooks.OnCommand = append(hooks.OnCommand, w)
}
func (w *SpoilerModule) Channels() []string {
  return []string{"example", "mylittleactivities", "mylittleroleplay", "mylittlenerds", "mylittlebot", "bot-debug" }
}

func (w *SpoilerModule) HasSpoiler(s *discordgo.Session, m *discordgo.Message) bool {
  if w.spoilerban != nil && w.spoilerban.MatchString(strings.ToLower(m.Content)) {
    s.ChannelMessageDelete(m.ChannelID, m.ID)
    if RateLimit(&w.lastmsg, sb.config.Maxspoiltime) {
      sb.SendMessage(m.ChannelID, "[](/sbtarget) ```POSTING SPOILERS IS A BANNABLE OFFENSE. All discussion about future episodes or seasons MUST be in #mylittlespoilers.```")
    }
    return true
  }
  return false
}

func (w *SpoilerModule) OnMessageCreate(s *discordgo.Session, m *discordgo.Message) {
  w.HasSpoiler(s, m)
}
  
func (w *SpoilerModule) OnMessageUpdate(s *discordgo.Session, m *discordgo.Message) {
  w.HasSpoiler(s, m)
}

func (w *SpoilerModule) OnCommand(s *discordgo.Session, m *discordgo.Message) bool {
  if UserHasAnyRole(m.Author.ID, sb.princessrole) { return false } // If we are a princess, always allow us to run this command, otherwise we can't unspoil things
  return w.HasSpoiler(s, m)
}

func (w *SpoilerModule) UpdateRegex() bool {
  if len(sb.config.Spoilers) < 1 {
    w.spoilerban = nil
    return true
  }
  var err error
  w.spoilerban, err = regexp.Compile("(" + strings.Join(sb.config.Spoilers, "|") + ")")
  return err == nil
}


type AddSpoilerCommand struct {
  spoilers *SpoilerModule
}

func (c *AddSpoilerCommand) Name() string {
  return "AddSpoiler";  
}
func (c *AddSpoilerCommand) Unban(spoiler string) bool {
  spoiler = strings.ToLower(spoiler)
  for i := 0; i < len(sb.config.Spoilers); i++ {
    if strings.ToLower(sb.config.Spoilers[i]) == spoiler {
      sb.config.Spoilers = append(sb.config.Spoilers[:i], sb.config.Spoilers[i+1:]...)
      return true
    }
  }
  return false
}
func (c *AddSpoilerCommand) Process(args []string, msg *discordgo.Message) (string, bool) {  
  if len(args) < 1 {
    return "```Nothing specified.```", false
  }
  if strings.ToLower(args[0]) == "unspoiler" {
    arg := strings.Join(args[1:], " ")
    if !c.Unban(arg) {
      return "```Could not find " + arg + "!```", false
    }
    sb.SaveConfig()
    c.spoilers.UpdateRegex()
    return "```Unspoilered " + arg + " and recompiled the spoiler regex.```", false
  }
  
  arg := strings.Join(args, " ")
  sb.config.Spoilers = append(sb.config.Spoilers, arg)
  sb.SaveConfig()
  r := c.spoilers.UpdateRegex()
  if !r {
    c.Unban(arg)
    c.spoilers.UpdateRegex()
    return "```Failed to ban " + arg + " because regex compilation failed.```", false
  }
  return "```Banned " + arg + " and recompiled the spoiler regex.```", false
}
func (c *AddSpoilerCommand) Usage() string { 
  return FormatUsage(c, "[unspoiler] [arbitrary string]", "Adds a line to spoilers (no quotes are required). This is used in a regex, so regex symbols are valid. If unspoiler is specified as the first word, unspoilers the spoiler instead.") 
}
func (c *AddSpoilerCommand) UsageShort() string { return "Adds a line to spoilers." }
func (c *AddSpoilerCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *AddSpoilerCommand) Channels() []string { return []string{} }