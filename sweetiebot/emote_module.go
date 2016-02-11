package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "regexp"
  "strings"
)

// The emote module detects banned emotes and deletes them
type EmoteModule struct {
  ModuleEnabled
  emoteban *regexp.Regexp
  lastmsg int64
}

func (w *EmoteModule) Name() string {
  return "Emote"
}

func (w *EmoteModule) Register(hooks *ModuleHooks) {
  w.lastmsg = 0
  w.UpdateRegex()
  hooks.OnMessageCreate = append(hooks.OnMessageCreate, w)
  hooks.OnMessageUpdate = append(hooks.OnMessageUpdate, w)
  hooks.OnCommand = append(hooks.OnCommand, w)
}
func (w *EmoteModule) Channels() []string {
  return []string{}
}

func (w *EmoteModule) HasBigEmote(s *discordgo.Session, m *discordgo.Message) bool {
  if w.emoteban.MatchString(m.Content) {
    s.ChannelMessageDelete(m.ChannelID, m.ID)
    if RateLimit(&w.lastmsg, 5) {
      s.ChannelMessageSend(m.ChannelID, "`That emote was way too big! Try to avoid using large emotes, as they can clutter up the chatroom.`")
    }
    return true
  }
  return false
}

func (w *EmoteModule) OnMessageCreate(s *discordgo.Session, m *discordgo.Message) {
  w.HasBigEmote(s, m)
}
  
func (w *EmoteModule) OnMessageUpdate(s *discordgo.Session, m *discordgo.Message) {
  w.HasBigEmote(s, m)
}

func (w *EmoteModule) OnCommand(s *discordgo.Session, m *discordgo.Message) bool {
  return w.HasBigEmote(s, m)
}

func (w *EmoteModule) UpdateRegex() bool {
  var err error
  w.emoteban, err = regexp.Compile("\\[\\]\\(\\/r?(" + strings.Join(sb.config.Emotes, "|") + ")[-) \"]")
  return err == nil
}


type BanEmoteCommand struct {
  emotes EmoteModule
}

func (c *BanEmoteCommand) Name() string {
  return "BanEmote";  
}
func (c *BanEmoteCommand) Unban(emote string) bool {
  for i := 0; i < len(sb.config.Emotes); i++ {
    if sb.config.Emotes[i] == emote {
      sb.config.Emotes = append(sb.config.Emotes[:i], sb.config.Emotes[i+1:]...)
      return true
    }
  }
  return false
}
func (c *BanEmoteCommand) Process(args []string, user *discordgo.User, channel string) (string, bool) {
  if len(args) < 1 {
    return "```No emote specified.```", false
  }
  if len(args) >= 2 {
    if strings.ToLower(args[1]) == "unban" {
      if !c.Unban(args[0]) {
        return "```Could not find " + args[0] + "! Remember that emotes are case-sensitive.```", false
      }
      sb.SaveConfig()
      c.emotes.UpdateRegex()
      return "```Unbanned " + args[0] + " and recompiled the emote regex.```", false
    }
    return "```Unrecognized second argument. Did you mean to type 'unban'?```", false
  }
  
  sb.config.Emotes = append(sb.config.Emotes, args[0])
  sb.SaveConfig()
  r := c.emotes.UpdateRegex()
  if !r {
    c.Unban(args[0])
    c.emotes.UpdateRegex()
    return "```Failed to ban " + args[0] + " because regex compilation failed.```", false
  }
  return "```Banned " + args[0] + " and recompiled the emote regex.```", false
}
func (c *BanEmoteCommand) Usage() string { 
  return FormatUsage(c, "[emote] [unban]", "Bans the given emote code, unless 'unban' is specified, in which case it unbans the emote.") 
}
func (c *BanEmoteCommand) UsageShort() string { return "Bans an emote." }
func (c *BanEmoteCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }