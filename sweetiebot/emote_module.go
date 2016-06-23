package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "regexp"
  "strings"
  "strconv"
)

// The emote module detects banned emotes and deletes them
type EmoteModule struct {
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

func (w *EmoteModule) HasBigEmote(s *discordgo.Session, m *discordgo.Message) bool {
  if w.emoteban.MatchString(m.Content) {
    s.ChannelMessageDelete(m.ChannelID, m.ID)
    if RateLimit(&w.lastmsg, 5) {
      sb.SendMessage(m.ChannelID, "`That emote isn't allowed here! Try to avoid using large or disturbing emotes, as they can be problematic.`")
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
  if UserHasRole(m.Author.ID, strconv.FormatUint(sb.config.AlertRole, 10)) { return false }
  return w.HasBigEmote(s, m)
}

func (w *EmoteModule) UpdateRegex() bool {
  var err error
  w.emoteban, err = regexp.Compile("\\[\\]\\(\\/r?(" + strings.Join(MapToSlice(sb.config.Collections["emote"]), "|") + ")[-) \"]")
  return err == nil
}