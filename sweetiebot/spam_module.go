package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "time"
  "strconv"
  "strings"
)

// The emote module detects banned emotes and deletes them
type SpamModule struct {
  tracker map[uint64]*SaturationLimit
  lastraid int64
}

func (w *SpamModule) Name() string {
  return "Anti-Spam"
}

func (w *SpamModule) Register(hooks *ModuleHooks) {
  w.tracker = make(map[uint64]*SaturationLimit)
  w.lastraid = 0
  hooks.OnMessageCreate = append(hooks.OnMessageCreate, w)
  hooks.OnCommand = append(hooks.OnCommand, w)
  hooks.OnGuildMemberAdd = append(hooks.OnGuildMemberAdd, w)
  hooks.OnGuildMemberUpdate = append(hooks.OnGuildMemberUpdate, w)
}

func KillSpammer(u *discordgo.User) {  
  // Manually set our internal state to say this user has the Silent role, to prevent race conditions
  m, err := sb.dg.State.Member(sb.Guild.ID, u.ID)
  if err == nil {
    srole := strconv.FormatUint(sb.config.SilentRole, 10)
    for _, v := range m.Roles {
      if v == srole {
        return // Spammer was already killed, so don't try killing it again
      }
    }
    m.Roles = append(m.Roles, srole)
  } else {
    sb.log.Log("Tried to kill spammer ", u.Username, " but they were already banned??? (Error: ", err.Error(), ")")
    return
  }
  
  sb.log.Log("Killing spammer ", u.Username)
  
  sb.dg.GuildMemberEdit(sb.Guild.ID, u.ID, m.Roles) // Tell discord to make this spammer silent
  messages := sb.db.GetRecentMessages(SBatoi(u.ID), 60) // Retrieve all messages in the past 60 seconds and delete them.

  for _, v := range messages {
    sb.dg.ChannelMessageDelete(strconv.FormatUint(v.channel, 10), strconv.FormatUint(v.message, 10))
  }
  
  sb.SendMessage(strconv.FormatUint(sb.config.ModChannel, 10), "`Alert: " + u.Username + " was silenced for spamming. Please investigate.`") // Alert admins
}
func (w *SpamModule) CheckSpam(s *discordgo.Session, m *discordgo.Message) bool {
  if m.Author != nil {
    if UserHasRole(m.Author.ID, strconv.FormatUint(sb.config.SilentRole, 10)) {
      s.ChannelMessageDelete(m.ChannelID, m.ID);
      return true
    }
    id := SBatoi(m.Author.ID)
    _, ok := w.tracker[id]
    if !ok {
      w.tracker[id] = &SaturationLimit{make([]int64, sb.config.Maxspam, sb.config.Maxspam), 0, AtomicFlag{0}};
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
func (w *SpamModule) OnGuildMemberAdd(s *discordgo.Session, m *discordgo.Member) {
  raidsize := sb.db.CountNewUsers(sb.config.MaxRaidTime);
  if sb.config.RaidSize > 0 && raidsize >= sb.config.RaidSize && RateLimit(&w.lastraid, sb.config.MaxRaidTime*2) {  
    r := sb.db.GetNewestUsers(raidsize)
    s := make([]string, 0, len(r))
    
    for _, v := range r {
      s = append(s, v.Username + "  (joined: " + v.FirstSeen.Format(time.ANSIC) + ")") 
    }
    ch := strconv.FormatUint(sb.config.ModChannel, 10)
    if sb.config.Debug { ch = sb.DebugChannelID }
    sb.SendMessage(ch, "<@&" + strconv.FormatUint(sb.config.AlertRole, 10) + "> Possible Raid Detected!\n```" + strings.Join(s, "\n") + "```")
  }
}
func (w *SpamModule) OnGuildMemberUpdate(s *discordgo.Session, m *discordgo.Member) {
  w.OnGuildMemberAdd(s, m)
}
  