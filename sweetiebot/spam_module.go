package sweetiebot

import (
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
	info.hooks.OnMessageUpdate = append(info.hooks.OnMessageUpdate, w)
	info.hooks.OnCommand = append(info.hooks.OnCommand, w)
	info.hooks.OnGuildMemberAdd = append(info.hooks.OnGuildMemberAdd, w)
	info.hooks.OnGuildMemberUpdate = append(info.hooks.OnGuildMemberUpdate, w)
}

func IsSilenced(m *discordgo.Member, info *GuildInfo) bool {
	srole := SBitoa(info.config.SilentRole)
	for _, v := range m.Roles {
		if v == srole {
			return true
		}
	}
	return false
}

func SilenceMember(userID string, info *GuildInfo) int8 {
	// Manually set our internal state to say this user has the Silent role, to prevent race conditions
	m, err := sb.dg.GuildMember(info.Guild.ID, userID)
	if err == nil {
		if IsSilenced(m, info) {
			return 1
		}
		m.Roles = append(m.Roles, SBitoa(info.config.SilentRole))
	} else {
		info.log.Log("Could not silence <@"+userID+"> because discordgo can't find them. (Error: ", err.Error(), ")")
		return -1
	}
	err = sb.dg.GuildMemberEdit(info.Guild.ID, userID, m.Roles) // Tell discord to make this spammer silent
	if err == nil {
		return 0
	}
	info.log.Log("GuildMemberEdit returned error: ", err.Error())
	return -2
}

func BanMember(u *discordgo.User, info *GuildInfo) {
	m, err := sb.dg.GuildMember(info.Guild.ID, u.ID)
	if err != nil || IsSilenced(m, info) {
		sb.dg.GuildBanCreate(info.Guild.ID, u.ID, 1)
	}
}

func KillSpammer(u *discordgo.User, info *GuildInfo, msg *discordgo.Message, reason string) {
	info.log.Log("Killing spammer ", u.Username, ". Last message sent: \n", msg.ContentWithMentionsReplaced())
	if SBatoi(msg.ChannelID) == info.config.WelcomeChannel {
		BanMember(u, info)
		info.SendMessage(SBitoa(info.config.ModChannel), "Alert: <@"+u.ID+"> was banned for "+reason+" in the welcome channel.")
		return
	}
	SilenceMember(u.ID, info)

	if info.config.MaxSpamRemoveLookback > 0 {
		if sb.IsDBGuild(info) {
			messages := sb.db.GetRecentMessages(SBatoi(u.ID), uint64(info.config.MaxSpamRemoveLookback), SBatoi(info.Guild.ID)) // Retrieve all messages in the past X seconds and delete them.

			for _, v := range messages {
				sb.dg.ChannelMessageDelete(SBitoa(v.channel), SBitoa(v.message))
			}
		}
	} else if info.config.MaxSpamRemoveLookback == 0 {
		sb.dg.ChannelMessageDelete(msg.ChannelID, msg.ID)
	} // otherwise we don't delete anything

	info.SendMessage(SBitoa(info.config.ModChannel), "Alert: <@"+u.ID+"> was silenced for "+reason+". Please investigate.") // Alert admins
}
func (w *SpamModule) CheckSpam(info *GuildInfo, m *discordgo.Message) bool {
	if m.Author != nil {
		if info.UserHasRole(m.Author.ID, SBitoa(info.config.SilentRole)) && SBatoi(m.ChannelID) != info.config.WelcomeChannel {
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
		//if limit.checkafter(5, 1) || limit.checkafter(7, 4) || limit.checkafter(10, 9) {
		for k, v := range info.config.MaxMessageSpam {
			if limit.checkafter(v, k) {
				KillSpammer(m.Author, info, m, "spamming too many messages")
				return true
			}
		}
		if len(m.Mentions) > info.config.MaxPingSpam {
			KillSpammer(m.Author, info, m, "pinging too many people")
			return true
		}
		if len(m.Embeds) > info.config.MaxImageSpam || len(m.Attachments) > info.config.MaxAttachSpam {
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
			s = append(s, v.User.Username+"  (joined: "+ApplyTimezone(v.FirstSeen, info, nil).Format(time.ANSIC)+")")
			if info.config.AutoSilence >= 1 {
				SilenceMember(v.User.ID, info)
			}
		}
		ch := SBitoa(info.config.ModChannel)
		if sb.Debug {
			ch, _ = sb.DebugChannels[info.Guild.ID]
		}
		info.SendMessage(ch, "<@&"+SBitoa(info.config.AlertRole)+"> Possible Raid Detected!\n```"+strings.Join(s, "\n")+"```")
	}
}
func (w *SpamModule) OnGuildMemberAdd(info *GuildInfo, m *discordgo.Member) {
	if info.config.AutoSilence >= 2 || (info.config.AutoSilence >= 1 && w.lastraid+info.config.MaxRaidTime*2 > time.Now().UTC().Unix()) {
		SilenceMember(m.User.ID, info)
		info.SendMessage(SBitoa(info.config.ModChannel), "<@"+m.User.ID+"> joined the server and was autosilenced. Please vet them before unsilencing them.")
		if len(info.config.WelcomeMessage) > 0 {
			info.SendMessage(SBitoa(info.config.WelcomeChannel), "<@"+m.User.ID+"> "+info.config.WelcomeMessage)
		}
	}
	if info.config.AutoSilence == -1 {
		info.SendMessage(SBitoa(info.config.ModChannel), "<@"+m.User.ID+"> joined the server.")
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
		info.config.AutoSilence = 2
	case "raid":
		info.config.AutoSilence = 1
	case "off":
		info.config.AutoSilence = 0
	case "alert":
		info.config.AutoSilence = -1
	//case "debug":
	//	subtract, _ := strconv.ParseInt(args[1], 10, 64)
	//	c.s.lastraid = time.Now().UTC().Unix() - subtract
	default:
		return "```Only all, raid, and off are valid auto silence levels.```", false
	}

	info.SaveConfig()

	if info.config.AutoSilence == 0 {
		// unsilence everyone
	} else if c.s.lastraid+info.config.MaxRaidTime*2 > time.Now().UTC().Unix() { // If there has recently been a raid, silence everyone who joined or theoretically could have joined since the beginning of the raid.
		r := sb.db.GetRecentUsers(time.Unix(c.s.lastraid-info.config.MaxRaidTime, 0).UTC(), SBatoi(info.Guild.ID))
		s := make([]string, 0, len(r))
		s = append(s, "```Detected a recent raid. All users from the raid have been silenced:")
		for _, v := range r {
			s = append(s, v.Username)
			SilenceMember(v.ID, info)
		}
		return strings.Join(s, "\n") + "```", false
	}
	return "```Set the auto silence level to " + strings.ToLower(args[0]) + "```", false
}
func (c *AutoSilenceCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[all/raid/alert/off]", "Toggles the auto silence level for anti-spam. All will autosilence all new members. Raid will only silence raiders. Alert does not auto-silence anyone, but sends an alert to the mod channel whenever anyone joins the server. Off disables auto-silence and unsilences everyone.")
}
func (c *AutoSilenceCommand) UsageShort() string { return "Toggle auto silence." }

type WipeWelcomeCommand struct {
}

func (c *WipeWelcomeCommand) Name() string {
	return "WipeWelcome"
}
func (c *WipeWelcomeCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	ch := SBitoa(info.config.WelcomeChannel)
	list, err := sb.dg.ChannelMessages(ch, 99, "", "")
	if err != nil {
		info.log.LogError("Error retrieving messages: ", err)
		return "```Error retrieving messages.```", false
	}
	for len(list) > 0 {
		IDs := make([]string, len(list), len(list))
		for i := 0; i < len(list); i++ {
			IDs[i] = list[i].ID
		}
		sb.dg.ChannelMessagesBulkDelete(ch, IDs)
		list, err = sb.dg.ChannelMessages(ch, 99, "", "")
	}
	return "Deleted all messages in <#" + ch + ">.", false
}
func (c *WipeWelcomeCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "", "Cleans out welcome channel.")
}
func (c *WipeWelcomeCommand) UsageShort() string { return "Cleans out welcome channel." }
