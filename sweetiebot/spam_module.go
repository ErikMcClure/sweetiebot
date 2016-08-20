package sweetiebot

import (
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// The emote module detects banned emotes and deletes them
type SpamModule struct {
	tracker     map[uint64]*SaturationLimit
	lastraid    int64
	AutoSilence int8
}

func (w *SpamModule) Name() string {
	return "Anti-Spam"
}

func (w *SpamModule) Register(info *GuildInfo) {
	w.tracker = make(map[uint64]*SaturationLimit)
	w.lastraid = 0
	w.AutoSilence = 0
	info.hooks.OnMessageCreate = append(info.hooks.OnMessageCreate, w)
	info.hooks.OnMessageUpdate = append(info.hooks.OnMessageUpdate, w)
	info.hooks.OnCommand = append(info.hooks.OnCommand, w)
	info.hooks.OnGuildMemberAdd = append(info.hooks.OnGuildMemberAdd, w)
	info.hooks.OnGuildMemberUpdate = append(info.hooks.OnGuildMemberUpdate, w)
}

func SilenceMember(u *discordgo.User, info *GuildInfo) {
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
	sb.dg.GuildMemberEdit(info.Guild.ID, u.ID, m.Roles) // Tell discord to make this spammer silent
}
func KillSpammer(u *discordgo.User, info *GuildInfo, msg *discordgo.Message, reason string) {
	info.log.Log("Killing spammer ", u.Username, ". Last message sent: \n", msg.ContentWithMentionsReplaced())
	SilenceMember(u, info)

	if sb.IsMainGuild(info) {
		messages := sb.db.GetRecentMessages(SBatoi(u.ID), 60) // Retrieve all messages in the past 60 seconds and delete them.

		for _, v := range messages {
			sb.dg.ChannelMessageDelete(strconv.FormatUint(v.channel, 10), strconv.FormatUint(v.message, 10))
		}
	}

	info.SendMessage(strconv.FormatUint(info.config.ModChannel, 10), "Alert: <@"+u.ID+"> was silenced for "+reason+". Please investigate.") // Alert admins
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
			KillSpammer(m.Author, info, m, "spamming too many messages")
			return true
		}
		if len(m.Mentions) > 4 {
			KillSpammer(m.Author, info, m, "pinging too many people")
			return true
		}
		if len(m.Embeds) > 3 || len(m.Attachments) > 1 {
			KillSpammer(m.Author, info, m, "embedding too many images")
			return true
		}
	}
	return false
}
func (w *SpamModule) OnMessageCreate(info *GuildInfo, m *discordgo.Message) {
	w.CheckSpam(info, m)
}
func (w *SpamModule) OnMessageUpdate(info *GuildInfo, m *discordgo.Message) {
	w.CheckSpam(info, m)
}
func (w *SpamModule) OnCommand(info *GuildInfo, m *discordgo.Message) bool {
	return w.CheckSpam(info, m)
}
func (w *SpamModule) checkRaid(info *GuildInfo, m *discordgo.Member) {
	raidsize := sb.db.CountNewUsers(info.config.MaxRaidTime, SBatoi(info.Guild.ID))
	if info.config.RaidSize > 0 && raidsize >= info.config.RaidSize && RateLimit(&w.lastraid, info.config.MaxRaidTime*2) {
		r := sb.db.GetNewestUsers(raidsize, SBatoi(info.Guild.ID))
		s := make([]string, 0, len(r))

		for _, v := range r {
			s = append(s, v.User.Username+"  (joined: "+ApplyTimezone(v.FirstSeen, info).Format(time.ANSIC)+")")
			if w.AutoSilence >= 1 {
				SilenceMember(v.User, info)
			}
		}
		ch := strconv.FormatUint(info.config.ModChannel, 10)
		if info.config.Debug {
			ch, _ = sb.DebugChannels[info.Guild.ID]
		}
		info.SendMessage(ch, "<@&"+strconv.FormatUint(info.config.AlertRole, 10)+"> Possible Raid Detected!\n```"+strings.Join(s, "\n")+"```")
	}
}
func (w *SpamModule) OnGuildMemberAdd(info *GuildInfo, m *discordgo.Member) {
	if w.AutoSilence >= 2 || (w.AutoSilence >= 1 && w.lastraid+info.config.MaxRaidTime*2 > time.Now().UTC().Unix()) {
		SilenceMember(m.User, info)
		info.SendMessage(strconv.FormatUint(info.config.ModChannel, 10), "<@"+m.User.ID+"> joined the server and was autosilenced. Please vet them before unsilencing them.")
	}
	w.checkRaid(info, m)
}
func (w *SpamModule) OnGuildMemberUpdate(info *GuildInfo, m *discordgo.Member) {
	w.checkRaid(info, m)
}

type AutoSilenceCommand struct {
	s *SpamModule
}

func (c *AutoSilenceCommand) Name() string {
	return "AutoSilence"
}
func (c *AutoSilenceCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You must provide an auto silence level (either all, raid, or off).```", false
	}
	switch strings.ToLower(args[0]) {
	case "all":
		c.s.AutoSilence = 2
	case "raid":
		c.s.AutoSilence = 1
	case "off":
		c.s.AutoSilence = 0
	//case "debug":
	//	subtract, _ := strconv.ParseInt(args[1], 10, 64)
	//	c.s.lastraid = time.Now().UTC().Unix() - subtract
	default:
		return "```Only all, raid, and off are valid auto silence levels.```", false
	}

	if c.s.AutoSilence == 0 {
		// unsilence everyone
	} else if c.s.lastraid+info.config.MaxRaidTime*2 > time.Now().UTC().Unix() { // If there has recently been a raid, silence everyone who joined or theoretically could have joined since the beginning of the raid.
		r := sb.db.GetRecentUsers(time.Unix(c.s.lastraid-info.config.MaxRaidTime, 0).UTC(), SBatoi(info.Guild.ID))
		s := make([]string, 0, len(r))
		s = append(s, "```Detected a recent raid. All users from the raid have been silenced:")
		for _, v := range r {
			s = append(s, v.Username)
			SilenceMember(v, info)
		}
		return strings.Join(s, "\n") + "```", false
	}
	return "```Set the auto silence level to " + strings.ToLower(args[0]) + "```", false
}
func (c *AutoSilenceCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[all/raid/off]", "Toggles the auto silence level for anti-spam. All will autosilence all new members. Raid will only silence raiders. Off disables auto-silence and unsilences everyone.")
}
func (c *AutoSilenceCommand) UsageShort() string { return "Toggle auto silence." }
