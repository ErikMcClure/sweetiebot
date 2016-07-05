package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "strconv"
  "regexp"
)

// This module picks a random action to do whenever #example has been idle for several minutes (configurable)
type SpoilerModule struct {
  spoilerban *regexp.Regexp
  lastmsg int64 // Sanity rate limiter
}

func (w *SpoilerModule) Name() string {
  return "Spoiler"
}

func (w *SpoilerModule) Register(info *GuildInfo) {
  w.lastmsg = 0
  w.UpdateRegex(info)
  info.hooks.OnMessageCreate = append(info.hooks.OnMessageCreate, w)
  info.hooks.OnMessageUpdate = append(info.hooks.OnMessageUpdate, w)
  info.hooks.OnCommand = append(info.hooks.OnCommand, w)
}

func (w *SpoilerModule) HasSpoiler(info *GuildInfo, m *discordgo.Message) bool {
  cid := SBatoi(m.ChannelID)
  for _, v := range info.config.SpoilChannels {
    if cid == v {
      return false // this is a spoiler channel so we don't monitor it
    }
  }
  if w.spoilerban != nil && w.spoilerban.MatchString(strings.ToLower(m.Content)) {
    sb.dg.ChannelMessageDelete(m.ChannelID, m.ID)
    if RateLimit(&w.lastmsg, info.config.Maxspoiltime) {
      info.SendMessage(m.ChannelID, "[](/nospoilers) ```NO SPOILERS! Posting spoilers is a bannable offense. All discussion about new and future content MUST be in #mylittlespoilers.```")
    }
    return true
  }
  return false
}

func (w *SpoilerModule) OnMessageCreate(info *GuildInfo, m *discordgo.Message) {
  w.HasSpoiler(info, m)
}
  
func (w *SpoilerModule) OnMessageUpdate(info *GuildInfo, m *discordgo.Message) {
  w.HasSpoiler(info, m)
}

func (w *SpoilerModule) OnCommand(info *GuildInfo, m *discordgo.Message) bool {
  if info.UserHasRole(m.Author.ID, strconv.FormatUint(info.config.AlertRole, 10)) { return false } // If we are a princess, always allow us to run this command, otherwise we can't unspoil things
  return w.HasSpoiler(info, m)
}

func (w *SpoilerModule) UpdateRegex(info *GuildInfo) bool {
  if len(info.config.Collections["spoiler"]) < 1 {
    w.spoilerban = nil
    return true
  }
  var err error
  w.spoilerban, err = regexp.Compile("(" + strings.Join(MapToSlice(info.config.Collections["spoiler"]), "|") + ")")
  return err == nil
}