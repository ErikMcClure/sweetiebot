package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "time"
  "strconv"
)

// The emote module detects banned emotes and deletes them
type SpamModule struct {
  maxlimit uint
  tracker map[uint64]*SaturationLimit
}

func (w *SpamModule) Name() string {
  return "Anti-Spam"
}

func (w *SpamModule) Register(hooks *ModuleHooks) {
  w.maxlimit = 30 // this must be at least 1 larger than the largest amount you check for
  w.tracker = make(map[uint64]*SaturationLimit)
  hooks.OnMessageCreate = append(hooks.OnMessageCreate, w)
  hooks.OnCommand = append(hooks.OnCommand, w)
}
func (w *SpamModule) Channels() []string {
  return []string{}
}

func KillSpammer(u *discordgo.User) {  
  // Manually set our internal state to say this user has the Silent role, to prevent race conditions
  m, err := sb.dg.State.Member(sb.GuildID, u.ID)
  if err == nil {
    for _, v := range m.Roles {
      if v == sb.SilentRole {
        return // Spammer was already killed, so don't try killing it again
      }
    }
    m.Roles = append(m.Roles, sb.SilentRole)
  }
  
  sb.log.Log("Killing spammer ", u.Username)
  
  sb.dg.GuildMemberEdit(sb.GuildID, m.User.ID, m.Roles) // Tell discord to make this spammer silent
  messages := sb.db.GetRecentMessages(SBatoi(m.User.ID), 60) // Retrieve all messages in the past 60 seconds and delete them.

  for _, v := range messages {
    sb.dg.ChannelMessageDelete(strconv.FormatUint(v.channel, 10), strconv.FormatUint(v.message, 10))
  }
  
  sb.dg.ChannelMessageSend(sb.ModChannelID, "`Alert: " + u.Username + " was silenced for spamming. Please investigate.`") // Alert admins
}
func (w *SpamModule) CheckSpam(s *discordgo.Session, m *discordgo.Message) bool {
  if m.Author != nil {
    if UserHasRole(m.Author.ID, sb.SilentRole) {
      s.ChannelMessageDelete(m.ChannelID, m.ID);
      return true
    }
    id := SBatoi(m.Author.ID)
    _, ok := w.tracker[id]
    if !ok {
      w.tracker[id] = &SaturationLimit{make([]int64, w.maxlimit, w.maxlimit), 0, AtomicFlag{0}};
    }
    limit := w.tracker[id]
    limit.append(time.Now().UTC().Unix())
    if limit.checkafter(5, 1) || limit.checkafter(10, 5) || limit.checkafter(12, 10) {
      KillSpammer(m.Author)
      return true
    }
  }
  return false
}
func (w *SpamModule) OnMessageCreate(s *discordgo.Session, m *discordgo.Message) {
  w.CheckSpam(s, m)
}
func (w *SpamModule) OnCommand(s *discordgo.Session, m *discordgo.Message) bool {
  return w.CheckSpam(s, m)
}
func (w *SpamModule) IsEnabled() bool {
  return true // always enabled
}
func (w *SpamModule) Enable(b bool) {}