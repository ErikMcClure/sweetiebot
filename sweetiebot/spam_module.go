package sweetiebot

import (
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// The emote module detects banned emotes and deletes them
type SpamModule struct {
	tracker  map[uint64]*SaturationLimit
	lastraid int64
}

func (w *SpamModule) Name() string {
	return "Anti-Spam"
}

func (w *SpamModule) Register(info *GuildInfo) {
	w.tracker = make(map[uint64]*SaturationLimit)
	w.lastraid = 0
	info.hooks.OnMessageCreate = append(info.hooks.OnMessageCreate, w)
	info.hooks.OnCommand = append(info.hooks.OnCommand, w)
	info.hooks.OnGuildMemberAdd = append(info.hooks.OnGuildMemberAdd, w)
	info.hooks.OnGuildMemberUpdate = append(info.hooks.OnGuildMemberUpdate, w)
}

func KillSpammer(u *discordgo.User, info *GuildInfo) {
	// Manually set our internal state to say this user has the Silent role, to prevent race conditions
	m, err := sb.dg.State.Member(info.Guild.ID, u.ID)
	if err == nil {
		srole := strconv.FormatUint(info.config.SilentRole, 10)
		for _, v := range m.Roles {
			if v == srole {
				return // Spammer was already killed, so don't try killing it again
			}
		}
		m.Roles = append(m.Roles, srole)
	} else {
		info.log.Log("Tried to kill spammer ", u.Username, " but they were already banned??? (Error: ", err.Error(), ")")
		return
	}

	info.log.Log("Killing spammer ", u.Username)

	sb.dg.GuildMemberEdit(info.Guild.ID, u.ID, m.Roles)   // Tell discord to make this spammer silent
	messages := sb.db.GetRecentMessages(SBatoi(u.ID), 60) // Retrieve all messages in the past 60 seconds and delete them.

	for _, v := range messages {
		sb.dg.ChannelMessageDelete(strconv.FormatUint(v.channel, 10), strconv.FormatUint(v.message, 10))
	}

	info.SendMessage(strconv.FormatUint(info.config.ModChannel, 10), "`Alert: "+u.Username+" was silenced for spamming. Please investigate.`") // Alert admins
}
func (w *SpamModule) CheckSpam(info *GuildInfo, m *discordgo.Message) bool {
	if m.Author != nil {
		if info.UserHasRole(m.Author.ID, strconv.FormatUint(info.config.SilentRole, 10)) {
			sb.dg.ChannelMessageDelete(m.ChannelID, m.ID)
			return true
		}
		id := SBatoi(m.Author.ID)
		_, ok := w.tracker[id]
		if !ok {
			w.tracker[id] = &SaturationLimit{make([]int64, 20, 20), 0, AtomicFlag{0}}
		}
		limit := w.tracker[id]
		limit.append(time.Now().UTC().Unix())
		if limit.checkafter(5, 1) || limit.checkafter(7, 4) || limit.checkafter(10, 9) {
			KillSpammer(m.Author, info)
			return true
		}
	}
	return false
}
func (w *SpamModule) OnMessageCreate(info *GuildInfo, m *discordgo.Message) {
	w.CheckSpam(info, m)
}
func (w *SpamModule) OnCommand(info *GuildInfo, m *discordgo.Message) bool {
	return w.CheckSpam(info, m)
}
func (w *SpamModule) OnGuildMemberAdd(info *GuildInfo, m *discordgo.Member) {
	raidsize := sb.db.CountNewUsers(info.config.MaxRaidTime, SBatoi(info.Guild.ID))
	if info.config.RaidSize > 0 && raidsize >= info.config.RaidSize && RateLimit(&w.lastraid, info.config.MaxRaidTime*2) {
		r := sb.db.GetNewestUsers(raidsize, SBatoi(info.Guild.ID))
		s := make([]string, 0, len(r))

		for _, v := range r {
			s = append(s, v.Username+"  (joined: "+v.FirstSeen.Format(time.ANSIC)+")")
		}
		ch := strconv.FormatUint(info.config.ModChannel, 10)
		if info.config.Debug {
			ch, _ = sb.DebugChannels[info.Guild.ID]
		}
		info.SendMessage(ch, "<@&"+strconv.FormatUint(info.config.AlertRole, 10)+"> Possible Raid Detected!\n```"+strings.Join(s, "\n")+"```")
	}
}
func (w *SpamModule) OnGuildMemberUpdate(info *GuildInfo, m *discordgo.Member) {
	w.OnGuildMemberAdd(info, m)
}
