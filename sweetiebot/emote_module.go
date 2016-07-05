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

func (w *EmoteModule) Register(info *GuildInfo) {
  w.lastmsg = 0
  w.UpdateRegex(info)
  info.hooks.OnMessageCreate = append(info.hooks.OnMessageCreate, w)
  info.hooks.OnMessageUpdate = append(info.hooks.OnMessageUpdate, w)
  info.hooks.OnCommand = append(info.hooks.OnCommand, w)
}

func (w *EmoteModule) HasBigEmote(info *GuildInfo, m *discordgo.Message) bool {
  if w.emoteban.MatchString(m.Content) {
    sb.dg.ChannelMessageDelete(m.ChannelID, m.ID)
    if RateLimit(&w.lastmsg, 5) {
      info.SendMessage(m.ChannelID, "`That emote isn't allowed here! Try to avoid using large or disturbing emotes, as they can be problematic.`")
    }
    return true
  }
  return false
}

func (w *EmoteModule) OnMessageCreate(info *GuildInfo, m *discordgo.Message) {
  w.HasBigEmote(info, m)
}
  
func (w *EmoteModule) OnMessageUpdate(info *GuildInfo, m *discordgo.Message) {
  w.HasBigEmote(info, m)
}

func (w *EmoteModule) OnCommand(info *GuildInfo, m *discordgo.Message) bool {
  if info.UserHasRole(m.Author.ID, strconv.FormatUint(info.config.AlertRole, 10)) { return false }
  return w.HasBigEmote(info, m)
}

func (w *EmoteModule) UpdateRegex(info *GuildInfo) bool {
  var err error
  w.emoteban, err = regexp.Compile("\\[\\]\\(\\/r?(" + strings.Join(MapToSlice(info.config.Collections["emote"]), "|") + ")[-) \"]")
  return err == nil
}