package sweetiebot

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"math"

	"github.com/blackhole12/discordgo"
)

type UserPressure struct {
	pressure    float32
	lastmessage int64
	lastcache   string
}

// The emote module detects banned emotes and deletes them
type SpamModule struct {
	sync.Mutex
	tracker  map[uint64]*UserPressure
	lastraid int64
}

func (w *SpamModule) Name() string {
	return "Anti-Spam"
}

func (w *SpamModule) Register(info *GuildInfo) {
	w.tracker = make(map[uint64]*UserPressure)
	w.lastraid = 0
	info.hooks.OnMessageCreate = append(info.hooks.OnMessageCreate, w)
	info.hooks.OnCommand = append(info.hooks.OnCommand, w)
	info.hooks.OnGuildMemberAdd = append(info.hooks.OnGuildMemberAdd, w)
	info.hooks.OnGuildMemberUpdate = append(info.hooks.OnGuildMemberUpdate, w)
	info.hooks.OnGuildMemberRemove = append(info.hooks.OnGuildMemberRemove, w)
}

func (w *SpamModule) Commands() []Command {
	return []Command{
		&AutoSilenceCommand{w},
		&WipeCommand{},
		&GetPressureCommand{w},
		&GetRaidCommand{w},
		&BanRaidCommand{w},
	}
}

func (w *SpamModule) Description() string {
	return "Tracks all channels it is active on for spammers. Each message someone sends generates \"pressure\", which decays rapidly. Long messages, messages with links, or messages with pings will generate more pressure. If a user generates too much pressure, they will be silenced and the moderators notified. Also detects groups of people joining at the same time and alerts the moderators of a potential raid."
}

func IsSilenced(m *discordgo.Member, info *GuildInfo) bool {
	srole := SBitoa(info.config.Spam.SilentRole)
	for _, v := range m.Roles {
		if v == srole {
			return true
		}
	}
	return false
}

func DoDiscordSilence(userID string, info *GuildInfo) {
	err := sb.dg.GuildMemberRoleAdd(info.ID, userID, SBitoa(info.config.Spam.SilentRole))
	info.log.LogError(fmt.Sprintf("GuildMemberRoleAdd(%s, %s, %v) return error: ", info.ID, userID, info.config.Spam.SilentRole), err)
}
func SilenceMember(user *discordgo.User, info *GuildInfo) int8 {
	defer DoDiscordSilence(user.ID, info) // No matter what, tell discord to make this spammer silent even if we've already done this, because discord is fucking stupid and sometimes fails for no reason
	m := info.GetMemberCreate(user)
	sb.dg.State.Lock()         // Manually set our internal state to say this spammer is silent to prevent race conditions
	defer sb.dg.State.Unlock() // this defer will execute BEFORE our DoDiscordSilence defer, minimizing lock time
	if IsSilenced(m, info) {
		return 1
	}
	m.Roles = append(m.Roles, SBitoa(info.config.Spam.SilentRole))

	return 0
}

func SilenceMemberSimple(userID string, info *GuildInfo) int8 {
	defer DoDiscordSilence(userID, info)
	m, merr := info.GetMember(userID)
	if merr == nil { // Manually set our internal state to say this spammer is silent to prevent race conditions
		sb.dg.State.Lock()
		defer sb.dg.State.Unlock()
		if IsSilenced(m, info) {
			return 1
		}
		m.Roles = append(m.Roles, SBitoa(info.config.Spam.SilentRole))
	}

	return 0
}

func KillSpammer(u *discordgo.User, info *GuildInfo, msg *discordgo.Message, reason string, oldpressure float32, newpressure float32) {
	// Before anything else happens, we delete this message. This ensures that even if we get rate-limited, we can still delete any new messages
	if info.config.Spam.MaxRemoveLookback >= 0 {
		sb.dg.ChannelMessageDelete(msg.ChannelID, msg.ID)
	}

	msgembeds := ""
	if len(msg.Embeds) > 0 {
		msgembeds = "\nEmbedded URLs: "
		for _, v := range msg.Embeds {
			msgembeds += "\n<" + v.URL + ">"
		}
	}

	chname := msg.ChannelID
	ch, err := sb.dg.Channel(msg.ChannelID)
	if err == nil {
		chname = ch.Name
	}
	lastmsg := SanitizeMentions(msg.ContentWithMentionsReplaced())
	if len(lastmsg) > 300 {
		lastmsg = lastmsg[:300] + " [truncated]"
	}
	logmsg := fmt.Sprintf("Killing spammer %s (pressure: %v -> %v). Last message sent on #%s in %s: \n%s%s", u.Username, oldpressure, newpressure, chname, info.Name, lastmsg, msgembeds)
	if SBatoi(msg.ChannelID) == info.config.Users.WelcomeChannel {
		sb.dg.GuildBanCreateWithReason(info.ID, u.ID, "Autobanned for "+reason+" in the welcome channel.", 1)
		info.SendMessage(SBitoa(info.config.Basic.ModChannel), "Alert: <@"+u.ID+"> was banned for "+reason+" in the welcome channel.")
		info.log.Log(logmsg)
		return
	}
	silenced := SilenceMember(u, info) > 0

	if info.config.Spam.MaxRemoveLookback > 0 && !silenced {
		IDs := []string{msg.ID}
		lastid := msg.ID
		endtime := time.Now().UTC().Add(time.Duration(-info.config.Spam.MaxRemoveLookback) * time.Second)

	EndLoop: // Even though this label is defined above the for loop, breaking to this label will actually skip the for loop entirely. Don't ask.
		for {
			messages, err := sb.dg.ChannelMessages(msg.ChannelID, 99, lastid, "", "")
			info.log.LogError("Error encountered while attempting to retrieve messages: ", err)
			if len(messages) == 0 || err != nil {
				break
			}
			lastid = messages[len(messages)-1].ID
			for _, v := range messages {
				tm, terr := v.Timestamp.Parse()
				info.log.LogError("Error encountered while attempting to parse timestamp: ", terr)
				if terr != nil || tm.Before(endtime) {
					break EndLoop // break out of both loops
				}
				if v.Author.ID == u.ID {
					IDs = append(IDs, v.ID)
				}
			}
		}

		sb.BulkDelete(msg.ChannelID, IDs)
	} // otherwise we don't delete anything

	if !silenced { // Only send the alert if they weren't silenced already
		info.SendMessage(SBitoa(info.config.Basic.ModChannel), "Alert: <@"+u.ID+"> was silenced for "+reason+". Please investigate.") // Alert admins
		info.log.Log(logmsg)
	} else {
		info.log.Log("Killing spammer " + u.Username)
	}
}

// Gets the pressure generated from an isolated message, ignoring the context.
func GetPressure(info *GuildInfo, m *discordgo.Message, edited bool) float32 {
	p := info.config.Spam.ImagePressure * float32(len(m.Attachments))
	p += info.config.Spam.PingPressure * float32(len(m.Mentions))
	p += info.config.Spam.ImagePressure * float32(len(m.Embeds))
	p += info.config.Spam.LengthPressure * float32(len(m.Content))
	p += info.config.Spam.LinePressure * float32(strings.Count(m.Content, "\n"))
	p += info.config.Spam.BasePressure
	if edited { // Editing a message contributes only the square root of the total (so you can edit a post with lots of pictures and not get instabanned)
		p = float32(math.Sqrt(float64(p)))
	}
	return p
}

func (w *SpamModule) CheckSpam(info *GuildInfo, m *discordgo.Message, edited bool) bool {
	if m.Author != nil {
		if info.UserHasRole(m.Author.ID, SBitoa(info.config.Spam.SilentRole)) && SBatoi(m.ChannelID) != info.config.Users.WelcomeChannel {
			sb.dg.ChannelMessageDelete(m.ChannelID, m.ID)
			return true
		}
		if (info.config.Basic.AlertRole != 0 && info.UserHasRole(m.Author.ID, SBitoa(info.config.Basic.AlertRole))) ||
			(info.config.Spam.IgnoreRole != 0 && info.UserHasRole(m.Author.ID, SBitoa(info.config.Spam.IgnoreRole))) ||
			m.Author.Bot {
			return false
		}
		id := SBatoi(m.Author.ID)
		tm, err := m.Timestamp.Parse()
		if len(m.EditedTimestamp) > 0 {
			tm, err = m.EditedTimestamp.Parse()
		}
		if err != nil {
			fmt.Println("Error parsing discord timestamp: ", m)
			tm = time.Now().UTC()
		}
		w.Lock()
		_, ok := w.tracker[id]
		if !ok {
			w.tracker[id] = &UserPressure{0, tm.Unix()*1000 + int64(tm.Nanosecond()/1000000), ""}
		}
		track := w.tracker[id]
		w.Unlock()
		p := GetPressure(info, m, edited)
		if len(m.Content) > 0 && strings.ToLower(m.Content) == track.lastcache {
			p += info.config.Spam.RepeatPressure
		}
		track.lastcache = strings.ToLower(m.Content)
		last := track.lastmessage
		track.lastmessage = tm.Unix()*1000 + int64(tm.Nanosecond()/1000000)
		if track.lastmessage < last { // This can happen because discord has a bad habit of re-sending timestamps if anything so much as touches a message
			track.lastmessage = last
			return false // An invalid timestamp is never spam
		}
		interval := track.lastmessage - last

		override, ok := info.config.Spam.MaxChannelPressure[SBatoi(m.ChannelID)]
		if ok && override > 0.0 {
			p *= (info.config.Spam.MaxPressure / override)
		}
		oldpressure := track.pressure
		track.pressure -= info.config.Spam.BasePressure * (float32(interval) / (info.config.Spam.PressureDecay * 1000.0))
		if track.pressure < 0 {
			track.pressure = 0
		}
		track.pressure += p
		//fmt.Println("Current Pressure: ", track.pressure)
		if track.pressure > info.config.Spam.MaxPressure {
			KillSpammer(m.Author, info, m, "spamming too many messages", oldpressure, track.pressure)
			return true
		}
	}
	return false
}
func (w *SpamModule) OnMessageCreate(info *GuildInfo, m *discordgo.Message) {
	w.CheckSpam(info, m, false)
}
func (w *SpamModule) OnCommand(info *GuildInfo, m *discordgo.Message) bool {
	return w.CheckSpam(info, m, false)
}

func DisableLockdown(info *GuildInfo) {
	if info.lockdown != -1 {
		modchan := SBitoa(info.config.Basic.ModChannel)
		if sb.Debug {
			modchan, _ = sb.DebugChannels[info.ID]
		}
		guild, err := sb.dg.State.Guild(info.ID)
		if err != nil {
			info.SendMessage(modchan, "Guild cannot be found in state?!")
		} else if guild.VerificationLevel != discordgo.VerificationLevelHigh {
			info.SendMessage(modchan, fmt.Sprintf("The verification level is at %v instead of %v, which means it was manually changed by someone other than sweetiebot, so it has not been restored.", guild.VerificationLevel, discordgo.VerificationLevelHigh))
		} else {
			g := discordgo.GuildParams{"", "", &info.lockdown, 0, "", 0, "", "", ""}
			_, err = sb.dg.GuildEdit(info.ID, g)
		}
		if err != nil {
			info.SendMessage(modchan, "Could not disengage lockdown! Make sure you've given the Sweetie Bot role the Manage Server permission, you'll have to manually restore it yourself this time.")
		} else {
			info.SendMessage(modchan, "Lockdown disengaged, server verification levels restored.")
		}
		info.lockdown = -1
	}
}
func (w *SpamModule) checkRaid(info *GuildInfo, m *discordgo.Member) {
	if !sb.db.CheckStatus() {
		return
	}
	raidsize := sb.db.CountNewUsers(info.config.Spam.RaidTime, SBatoi(info.ID))
	if info.config.Spam.RaidSize > 0 && raidsize >= info.config.Spam.RaidSize && RateLimit(&w.lastraid, info.config.Spam.RaidTime*2) {
		r := sb.db.GetNewestUsers(raidsize, SBatoi(info.ID))
		s := make([]string, 0, len(r))

		for _, v := range r {
			s = append(s, v.User.Username+"  (joined: "+ApplyTimezone(v.FirstSeen, info, nil).Format(time.ANSIC)+")")
			if info.config.Spam.AutoSilence >= 1 {
				SilenceMember(v.User, info)
			}
		}
		ch := SBitoa(info.config.Basic.ModChannel)
		if sb.Debug {
			ch, _ = sb.DebugChannels[info.ID]
		}
		info.SendMessage(ch, "<@&"+SBitoa(info.config.Basic.AlertRole)+"> Possible Raid Detected! Use `"+info.config.Basic.CommandPrefix+"autosilence all` to silence them!\n```"+strings.Join(s, "\n")+"```")
		if info.config.Spam.LockdownDuration > 0 {
			if info.lockdown == -1 { // Only engage lockdown if it wasn't already engaged
				guild, err := sb.dg.State.Guild(info.ID)
				if err != nil {
					info.lockdown = discordgo.VerificationLevelHigh
				} else {
					info.lockdown = guild.VerificationLevel
				}
				level := discordgo.VerificationLevelHigh
				g := discordgo.GuildParams{"", "", &level, 0, "", 0, "", "", ""}
				_, err = sb.dg.GuildEdit(info.ID, g)
				if err != nil {
					info.SendMessage(ch, "Could not engage lockdown! Make sure you've given Sweetie Bot the Manage Server permission, or disable the lockdown entirely via `"+info.config.Basic.CommandPrefix+"setconfig spam.lockdownduration 0`.")
				} else {
					info.SendMessage(ch, fmt.Sprintf("Lockdown engaged! Server verification level will be reset in %v seconds. This lockdown can be manually ended via `"+info.config.Basic.CommandPrefix+"autosilence off/alert/log`.", info.config.Spam.LockdownDuration))
				}
			}
			// Otherwise just reset the timer
			info.lastlockdown = time.Now().UTC()
		}
	}
}
func (w *SpamModule) OnGuildMemberAdd(info *GuildInfo, m *discordgo.Member) {
	created := "(Created " + TimeDiff(time.Now().UTC().Sub(snowflakeTime(SBatoi(m.User.ID)))) + " ago)"
	if info.config.Spam.AutoSilence >= 2 || (info.config.Spam.AutoSilence >= 1 && w.lastraid+info.config.Spam.RaidTime*2 > time.Now().UTC().Unix()) {
		SilenceMember(m.User, info)
		info.SendMessage(SBitoa(info.config.Basic.ModChannel), "<@"+m.User.ID+"> "+created+" joined the server and was autosilenced. Please vet them before unsilencing them.")
		if len(info.config.Users.WelcomeMessage) > 0 {
			info.SendMessage(SBitoa(info.config.Users.WelcomeChannel), "<@"+m.User.ID+"> "+info.config.Users.WelcomeMessage)
		}
	}
	if info.config.Spam.AutoSilence == -1 {
		info.SendMessage(SBitoa(info.config.Basic.ModChannel), "<@"+m.User.ID+"> "+created+" joined the server.")
	}
	if info.config.Spam.AutoSilence == -2 {
		info.SendMessage(SBitoa(info.config.Log.Channel), "<@"+m.User.ID+"> "+created+" joined the server.")
	}
	w.checkRaid(info, m)
}
func (w *SpamModule) OnGuildMemberUpdate(info *GuildInfo, m *discordgo.Member) {
	w.checkRaid(info, m)
}
func (w *SpamModule) OnGuildMemberRemove(info *GuildInfo, m *discordgo.Member) {
	if info.config.Basic.TrackUserLeft {
		text := m.User.Username + "#" + m.User.Discriminator + " left the server."
		if info.config.Spam.AutoSilence == -1 || info.config.Spam.AutoSilence >= 2 {
			info.SendMessage(SBitoa(info.config.Basic.ModChannel), text)
		} else if info.config.Spam.AutoSilence == -2 {
			info.SendMessage(SBitoa(info.config.Log.Channel), text)
		}
	}
}
func (w *SpamModule) GetRaidUsers(info *GuildInfo) []*discordgo.User {
	return sb.db.GetRecentUsers(time.Unix(w.lastraid-info.config.Spam.RaidTime, 0).UTC(), SBatoi(info.ID))
}
func (w *SpamModule) IsRecentRaid(info *GuildInfo) bool {
	return w.lastraid+info.config.Spam.RaidTime*2 > time.Now().UTC().Unix()
}

type AutoSilenceCommand struct {
	s *SpamModule
}

func (c *AutoSilenceCommand) Name() string {
	return "AutoSilence"
}
func (c *AutoSilenceCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You must provide an auto silence level (either alert, log, all, raid, or off).```", false, nil
	}
	switch strings.ToLower(args[0]) {
	case "all":
		info.config.Spam.AutoSilence = 2
	case "raid":
		info.config.Spam.AutoSilence = 1
	case "off":
		info.config.Spam.AutoSilence = 0
	case "alert":
		info.config.Spam.AutoSilence = -1
	case "log":
		info.config.Spam.AutoSilence = -2
	//case "debug":
	//	subtract, _ := strconv.ParseInt(args[1], 10, 64)
	//	c.s.lastraid = time.Now().UTC().Unix() - subtract
	default:
		return "```Only alert, log, all, raid, and off are valid auto silence levels.```", false, nil
	}

	info.SaveConfig()

	if info.config.Spam.AutoSilence <= 0 {
		DisableLockdown(info)
	} else if c.s.IsRecentRaid(info) { // If there has recently been a raid, silence everyone who joined or theoretically could have joined since the beginning of the raid.
		info.lastlockdown = time.Now().UTC() // Reset lockdown timer just in case
		if !sb.db.CheckStatus() {
			return "```Autosilence was engaged, but a database error prevents me from retroactively applying it!```", false, nil
		}
		// BEFORE we make any calls to discord, which could take some time, immediately respond with a silence set message so the admins know the command is functioning
		info.SendMessage(msg.ChannelID, "```Set the auto silence level to "+strings.ToLower(args[0])+".```")
		r := c.s.GetRaidUsers(info)
		s := make([]string, 0, len(r))
		s = append(s, "```Detected a recent raid. All users from the raid have been silenced:")
		for _, v := range r {
			s = append(s, v.Username)
			SilenceMember(v, info)
		}
		return strings.Join(s, "\n") + "```", false, nil
	}
	return "```Set the auto silence level to " + strings.ToLower(args[0]) + ".```", false, nil
}
func (c *AutoSilenceCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Toggles the auto silence level for anti-spam.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "all/raid/alert/log/off", Desc: "`all` will autosilence all new members. `raid` will turn on autosilence if a raid is detected (not recommended). `alert` does not auto-silence anyone, but sends an alert to the mod channel whenever anyone joins the server. `log` sends the alerts to the log channel instead. `off` disables auto-silence and unsilences everyone.", Optional: false},
		},
	}
}
func (c *AutoSilenceCommand) UsageShort() string { return "Toggle auto silence." }

type WipeCommand struct {
}

func (c *WipeCommand) Name() string {
	return "Wipe"
}
func (c *WipeCommand) WipeMessages(ch string, num int, seconds int) (int, error) {
	date := time.Now().UTC().Add(time.Duration(-seconds) * time.Second)

	ret := 0
	lastid := ""
	for ret < num {
		n := num - ret
		if n > 99 {
			n = 99
		}
		list, err := sb.dg.ChannelMessages(ch, n, lastid, "", "")
		if err != nil || len(list) == 0 {
			return ret, err
		}
		IDs := make([]string, 0, len(list))
		for i := 0; i < len(list) && ret < num; i++ {
			if seconds > 0 {
				t, err := list[i].Timestamp.Parse()
				if err != nil || t.Before(date) {
					break
				}
			}
			IDs = append(IDs, list[i].ID)
			ret++
		}
		if len(IDs) == 0 {
			break
		}
		sb.dg.ChannelMessagesBulkDelete(ch, IDs)
		lastid = IDs[len(IDs)-1]
	}
	return ret, nil
}
func (c *WipeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 2 {
		return "```You must specify the channel and the duration.```", false, nil
	}

	ch := StripPing(args[0])
	num, err := strconv.Atoi(args[1])
	if err != nil || num <= 0 {
		return "```There's no point deleting 0 messages!.```", false, nil
	}
	if len(args) > 2 && strings.ToLower(args[2]) == "messages" {
		num, err = c.WipeMessages(ch, num, 0)
	} else {
		num, err = c.WipeMessages(ch, 9999, num)
	}
	if err != nil {
		return "```Error retrieving messages. Are you sure you gave sweetiebot a channel that exists? This won't work in PMs! " + err.Error() + "```", false, nil
	}
	return fmt.Sprintf("Deleted %v messages in <#%s>.", num, ch), false, nil
}
func (c *WipeCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Removes all messages in a channel sent within the last N seconds, or simply removes the last N messages if \"messages\" is appended.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "channel", Desc: "The channel to delete from. You must use the #channel format so discord actually highlights the channel, otherwise it won't work.", Optional: false},
			CommandUsageParam{Name: "seconds", Desc: "Specifies the number of seconds to look back. The command deletes all messages sent up to this many seconds ago.", Optional: false},
			CommandUsageParam{Name: "MESSAGES", Desc: "If you append \"MESSAGES\" to the end of the command, it will remove that many messages, instead of looking back that many seconds.", Optional: true},
		},
	}
}
func (c *WipeCommand) UsageShort() string { return "Cleans out welcome channel." }

type GetPressureCommand struct {
	s *SpamModule
}

func (c *GetPressureCommand) Name() string {
	return "GetPressure"
}
func (c *GetPressureCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	_, isOwner := sb.Owners[SBatoi(msg.Author.ID)]
	if !isOwner {
		return "```Only the owner of the bot itself can call this!```", false, nil
	}
	if len(args) < 1 {
		return "```You must provide a user to search for.```", false, nil
	}
	arg := msg.Content[indices[0]:]
	IDs := FindUsername(arg, info)

	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false, nil
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info, true), "\n") + "```", len(IDs) > 5, nil
	}

	c.s.Lock()
	u, ok := c.s.tracker[IDs[0]]
	c.s.Unlock()
	if !ok {
		return "0", false, nil
	}
	return fmt.Sprint(u.pressure), false, nil
}
func (c *GetPressureCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Restricted command that gets the current spam pressure of a user.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "user", Desc: "User to retrieve pressure from.", Optional: false},
		},
	}
}
func (c *GetPressureCommand) UsageShort() string { return "[RESTRICTED] Gets user's spam pressure." }

type GetRaidCommand struct {
	s *SpamModule
}

func (c *GetRaidCommand) Name() string {
	return "GetRaid"
}
func (c *GetRaidCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !c.s.IsRecentRaid(info) {
		return fmt.Sprintf("```No raid has occured within the past %s.```", TimeDiff(time.Duration(info.config.Spam.RaidTime*2)*time.Second)), false, nil
	}
	s := []string{"Users in latest raid: "}
	for _, v := range c.s.GetRaidUsers(info) {
		s = append(s, v.Username+"#"+v.Discriminator)
	}
	return "```" + strings.Join(s, "\n") + "```", false, nil
}
func (c *GetRaidCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{Desc: "Lists all users that are considered part of the most recent raid, if there was one."}
}
func (c *GetRaidCommand) UsageShort() string { return "Lists users in most recent raid." }

type BanRaidCommand struct {
	s *SpamModule
}

func (c *BanRaidCommand) Name() string {
	return "BanRaid"
}
func (c *BanRaidCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !c.s.IsRecentRaid(info) {
		return fmt.Sprintf("```No raid has occured within the past %s.```", TimeDiff(time.Duration(info.config.Spam.RaidTime*2)*time.Second)), false, nil
	}
	reason := fmt.Sprintf("Banned by %s#%s via the !banraid command.", msg.Author.Username, msg.Author.Discriminator)
	users := c.s.GetRaidUsers(info)
	for _, v := range users {
		sb.dg.GuildBanCreateWithReason(info.ID, v.ID, reason, 1)
	}
	return fmt.Sprintf("```Banned %v users. The ban log will reflect who ran this command.```", len(users)), false, nil
}
func (c *BanRaidCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{Desc: "Bans all users that are considered part of the most recent raid, if there was one. Use !getraid to check who will be banned before using this command."}
}
func (c *BanRaidCommand) UsageShort() string { return "Bans all users in most recent raid." }
