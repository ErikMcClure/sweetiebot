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

func (w *SpoilerModule) Register(hooks *ModuleHooks) {
  w.lastmsg = 0
  w.UpdateRegex()
  hooks.OnMessageCreate = append(hooks.OnMessageCreate, w)
  hooks.OnMessageUpdate = append(hooks.OnMessageUpdate, w)
  hooks.OnCommand = append(hooks.OnCommand, w)
}

func (w *SpoilerModule) HasSpoiler(s *discordgo.Session, m *discordgo.Message) bool {
  cid := SBatoi(m.ChannelID)
  for _, v := range sb.config.SpoilChannels {
    if cid == v {
      return false // this is a spoiler channel so we don't monitor it
    }
  }
  if w.spoilerban != nil && w.spoilerban.MatchString(strings.ToLower(m.Content)) {
    s.ChannelMessageDelete(m.ChannelID, m.ID)
    if RateLimit(&w.lastmsg, sb.config.Maxspoiltime) {
      sb.SendMessage(m.ChannelID, "[](/nospoilers) ```NO SPOILERS! Posting spoilers is a bannable offense. All discussion about new and future content MUST be in #mylittlespoilers.```")
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
  if UserHasRole(m.Author.ID, strconv.FormatUint(sb.config.AlertRole, 10)) { return false } // If we are a princess, always allow us to run this command, otherwise we can't unspoil things
  return w.HasSpoiler(s, m)
}

func (w *SpoilerModule) UpdateRegex() bool {
  if len(sb.config.Collections["spoiler"]) < 1 {
    w.spoilerban = nil
    return true
  }
  var err error
  w.spoilerban, err = regexp.Compile("(" + strings.Join(MapToSlice(sb.config.Collections["spoiler"]), "|") + ")")
  return err == nil
}