package spammodule

import (
	"container/heap"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	bot "../sweetiebot"
	"github.com/erikmcclure/discordgo"
)

type userPressure struct {
	pressure    float32
	lastmessage int64
	lastcache   string
}

type userTimeout struct {
	user bot.DiscordUser
	time time.Time
}

type userTimeoutHeap []userTimeout

func (h userTimeoutHeap) Len() int           { return len(h) }
func (h userTimeoutHeap) Less(i, j int) bool { return h[i].time.Before(h[j].time) }
func (h userTimeoutHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h userTimeoutHeap) Peek() *userTimeout { return &h[0] }

func (h *userTimeoutHeap) Push(x interface{}) {
	*h = append(*h, x.(userTimeout))
}

func (h *userTimeoutHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// SpamModule detects banned emotes and deletes them
type SpamModule struct {
	tracker      sync.Map                    //map[bot.DiscordUser]*userPressure
	lockdown     discordgo.VerificationLevel // if -1 no lockdown was initiated, otherwise remembers the previous lockdown setting
	lastlockdown time.Time
	timeouts     *userTimeoutHeap
	timeoutLock  sync.Mutex
	silenced     map[bot.DiscordUser]bool // Users that have been silenced
	resilence    map[bot.DiscordUser]bool // Users that left while silenced
	silenceLock  sync.Mutex
}

// New spam module
func New() *SpamModule {
	w := &SpamModule{
		lockdown:  -1,
		timeouts:  &userTimeoutHeap{},
		silenced:  make(map[bot.DiscordUser]bool),
		resilence: make(map[bot.DiscordUser]bool),
	}
	heap.Init(w.timeouts)
	return w
}

// Name of the module
func (w *SpamModule) Name() string {
	return "Spam"
}

// Commands in the module
func (w *SpamModule) Commands() []bot.Command {
	return []bot.Command{
		&raidSilenceCommand{w},
		&wipeCommand{},
		&getPressureCommand{w},
		&getRaidCommand{w},
		&banRaidCommand{w},
	}
}

// Description of the module
func (w *SpamModule) Description(info *bot.GuildInfo) string {
	return fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_DESCRIPTION], info.Config.Basic.CommandPrefix, info.Config.Basic.CommandPrefix, info.Config.Basic.CommandPrefix, info.Config.Basic.CommandPrefix)
}

// OnTick discord hook
func (w *SpamModule) OnTick(info *bot.GuildInfo, t time.Time) {
	if w.lockdown != -1 && t.Sub(w.lastlockdown) > (time.Duration(info.Config.Spam.LockdownDuration)*time.Second) {
		w.DisableLockdown(info)
	}
	w.timeoutLock.Lock()
	for w.timeouts.Len() > 0 && w.timeouts.Peek().time.Before(t) {
		u := heap.Pop(w.timeouts).(userTimeout).user
		err := info.Bot.DG.RemoveRole(info.ID, u, info.Config.Basic.SilenceRole)
		if err != nil {
			info.SendMessage(info.Config.Basic.ModChannel, fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_ERROR_UNSILENCING], err.Error()))
		}
		info.SendMessage(info.Config.Basic.ModChannel, fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_UNSILENCING], info.GetUserName(u)))
	}
	w.timeoutLock.Unlock()
}

func silenceMember(user *discordgo.User, info *bot.GuildInfo) int8 {
	defer info.Bot.DG.GuildMemberRoleAdd(info.ID, user.ID, info.Config.Basic.SilenceRole.String()) // No matter what, tell discord to make this spammer silent even if we've already done this, because discord is fucking stupid and sometimes fails for no reason
	m := info.Bot.DG.GetMemberCreate(user, info.ID)
	if bot.MemberHasRole(m, info.Config.Basic.SilenceRole) {
		return 1
	}
	nroles := make([]string, len(m.Roles)) // We set this to a new slice so we can atomically replace it on x86 architectures, avoiding a lock
	copy(nroles, m.Roles)
	m.Roles = append(nroles, info.Config.Basic.SilenceRole.String())

	return 0
}

func (w *SpamModule) killSpammer(u *discordgo.User, info *bot.GuildInfo, msg *discordgo.Message, reason string, oldpressure float32, newpressure float32) {
	// Before anything else happens, we delete this message. This ensures that even if we get rate-limited, we can still delete any new messages
	if info.Config.Spam.MaxRemoveLookback >= 0 {
		time.Sleep(bot.DelayTime)
		info.Bot.DG.ChannelMessageDelete(msg.ChannelID, msg.ID)
	}

	timestamp := bot.GetTimestamp(msg)
	msgembeds := ""
	if len(msg.Embeds) > 0 {
		msgembeds = bot.StringMap[bot.STRING_SPAM_EMBEDDED_URLS]
		for _, v := range msg.Embeds {
			msgembeds += "\n<" + v.URL + ">"
		}
	}

	ch, err := info.Bot.DG.State.Channel(msg.ChannelID)
	chname := msg.ChannelID
	if err == nil {
		chname = ch.Name
	}
	lastmsg := info.Sanitize(msg.Content, bot.CleanAll)
	split := strings.SplitAfterN(lastmsg, "\n", 10)
	if len(split) > 9 {
		lastmsg = strings.Join(split[:9], "\n")
		if len(lastmsg) > 300 {
			lastmsg = lastmsg[:300]
		}
		lastmsg += bot.StringMap[bot.STRING_SPAM_TRUNCATED]
	} else if len(lastmsg) > 300 {
		lastmsg = lastmsg[:300] + bot.StringMap[bot.STRING_SPAM_TRUNCATED]
	}
	logmsg := fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_KILLING_SPAMMER_DETAIL], u.Username, oldpressure, newpressure, chname, info.Name, lastmsg, msgembeds)
	if info.Config.Users.WelcomeChannel.Equals(msg.ChannelID) || info.Config.Users.JailChannel.Equals(msg.ChannelID) {
		info.Bot.DG.GuildBanCreateWithReason(info.ID, u.ID, fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_AUTOBANNED_REASON], reason), 1)
		info.SendMessage(info.Config.Basic.ModChannel, fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_BAN_ALERT], u.ID, reason))
		info.Log(logmsg)
		return
	}
	silenced := silenceMember(u, info) > 0

	if info.Config.Spam.MaxRemoveLookback > 0 && !silenced {
		IDs := []string{msg.ID}
		lastid := msg.ID
		endtime := timestamp.Add(time.Duration(-info.Config.Spam.MaxRemoveLookback) * time.Second)

	EndLoop: // Even though this label is defined above the for loop, breaking to this label will actually skip the for loop entirely. Don't ask.
		for {
			messages, err := info.Bot.DG.ChannelMessages(msg.ChannelID, 99, lastid, "", "")
			info.LogError(bot.StringMap[bot.STRING_SPAM_ERROR_RETRIEVE_MESSAGES], err)
			if len(messages) == 0 || err != nil {
				break
			}
			lastid = messages[len(messages)-1].ID
			for _, v := range messages {
				tm, terr := v.Timestamp.Parse()
				if terr != nil || tm.Before(endtime) {
					break EndLoop // break out of both loops
				}
				if v.Author.ID == u.ID {
					IDs = append(IDs, v.ID)
				}
			}
		}

		info.Bot.DG.BulkDeleteBypass(msg.ChannelID, IDs) // We use the bypass because we can't risk the channel not being in the state for some reason
	} // otherwise we don't delete anything

	if !silenced { // Only send the alert if they weren't silenced already
		addmsg := "."
		if info.Config.Spam.SilenceTimeout > 0 {
			timeout := time.Duration(info.Config.Spam.SilenceTimeout) * time.Second
			addmsg = fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_WILL_BE_UNSILENCED], bot.TimeDiff(timeout))
			w.timeoutLock.Lock()
			heap.Push(w.timeouts, userTimeout{bot.DiscordUser(u.ID), timestamp.Add(timeout)})
			w.timeoutLock.Unlock()
		}
		info.SendMessage(info.Config.Basic.ModChannel, fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_SILENCE_ALERT], u.ID, reason, addmsg)) // Alert admins
		info.Log(logmsg)
	} else {
		info.Log(fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_KILLING_SPAMMER], u.Username))
	}
}

// TrackUser gets or creates the user tracking object for a given author
func (w *SpamModule) TrackUser(author bot.DiscordUser, timestamp time.Time) *userPressure {
	v, _ := w.tracker.LoadOrStore(author, &userPressure{0, timestamp.Unix()*1000 + int64(timestamp.Nanosecond()/1000000), ""})
	return v.(*userPressure)
}

// AddPressure to a user and checks to see if it goes over the limit. Used to supplement spam module via filter module
func (w *SpamModule) AddPressure(info *bot.GuildInfo, m *discordgo.Message, track *userPressure, p float32, reason string) bool {
	old := track.pressure

	override, ok := info.Config.Spam.MaxChannelPressure[bot.DiscordChannel(m.ChannelID)]
	if ok && override > 0.0 {
		p *= (info.Config.Spam.MaxPressure / override)
	}

	track.pressure += p
	if track.pressure > info.Config.Spam.MaxPressure {
		w.killSpammer(m.Author, info, m, reason, old, track.pressure)
		return true
	}
	return false
}

func (w *SpamModule) checkSpam(info *bot.GuildInfo, m *discordgo.Message) bool {
	if m.Author != nil {
		author := bot.DiscordUser(m.Author.ID)

		if info.UserIsMod(author) || info.UserIsAdmin(author) || m.Author.Bot {
			return false
		}
		if info.UserHasRole(author, info.Config.Basic.SilenceRole) && !info.Config.Users.JailChannel.Equals(m.ChannelID) {
			ch, _ := info.Bot.DG.Channel(m.ChannelID)
			time.Sleep(bot.DelayTime)
			info.ChannelMessageDelete(ch, m.ID)
			return true
		}
		if info.Config.Spam.IgnoreRole != bot.RoleEmpty && info.UserHasRole(author, info.Config.Spam.IgnoreRole) {
			return false
		}

		timestamp := bot.GetTimestamp(m)
		track := w.TrackUser(author, timestamp)
		last := track.lastmessage
		track.lastmessage = timestamp.Unix()*1000 + int64(timestamp.Nanosecond()/1000000)
		if track.lastmessage < last { // This can happen because discord has a bad habit of re-sending timestamps if anything so much as touches a message
			track.lastmessage = last
			return false // An invalid timestamp is never spam
		}
		interval := track.lastmessage - last

		track.pressure -= info.Config.Spam.BasePressure * (float32(interval) / (info.Config.Spam.PressureDecay * 1000.0))
		if track.pressure < 0 {
			track.pressure = 0
		}

		if w.AddPressure(info, m, track, info.Config.Spam.BasePressure, bot.StringMap[bot.STRING_SPAM_REASON_MESSAGES]) {
			return true
		}
		if w.AddPressure(info, m, track, info.Config.Spam.ImagePressure*float32(len(m.Attachments)), bot.StringMap[bot.STRING_SPAM_REASON_FILES]) {
			return true
		}
		if w.AddPressure(info, m, track, info.Config.Spam.ImagePressure*float32(len(m.Embeds)), bot.StringMap[bot.STRING_SPAM_REASON_IMAGES]) {
			return true
		}
		if w.AddPressure(info, m, track, info.Config.Spam.PingPressure*float32(len(m.Mentions)), bot.StringMap[bot.STRING_SPAM_REASON_PINGS]) {
			return true
		}
		if w.AddPressure(info, m, track, info.Config.Spam.LengthPressure*float32(len(m.Content)), bot.StringMap[bot.STRING_SPAM_REASON_LENGTH]) {
			return true
		}
		if w.AddPressure(info, m, track, info.Config.Spam.LinePressure*float32(strings.Count(m.Content, "\n")), bot.StringMap[bot.STRING_SPAM_REASON_NEWLINES]) {
			return true
		}
		if len(m.Content) > 0 && strings.ToLower(m.Content) == track.lastcache {
			if w.AddPressure(info, m, track, info.Config.Spam.RepeatPressure, bot.StringMap[bot.STRING_SPAM_REASON_COPY]) {
				return true
			}
		}
		track.lastcache = strings.ToLower(m.Content)
	}
	return false
}

// OnMessageCreate discord hook
func (w *SpamModule) OnMessageCreate(info *bot.GuildInfo, m *discordgo.Message) {
	w.checkSpam(info, m)
}

// OnCommand discord hook
func (w *SpamModule) OnCommand(info *bot.GuildInfo, m *discordgo.Message) bool {
	return w.checkSpam(info, m)
}

// DisableLockdown disables the guild lockdown, if there is one
func (w *SpamModule) DisableLockdown(info *bot.GuildInfo) {
	if w.lockdown != -1 {
		modchan := info.Config.Basic.ModChannel
		if info.Bot.Debug {
			modchan, _ = info.Bot.DebugChannels[bot.DiscordGuild(info.ID)]
		}
		guild, err := info.GetGuild()
		if err != nil {
			info.SendMessage(modchan, bot.StringMap[bot.STRING_SPAM_GUILD_NOT_FOUND])
		} else if guild.VerificationLevel != discordgo.VerificationLevelHigh {
			info.SendMessage(modchan, fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_VERIFICATION_LEVEL_ERROR], guild.VerificationLevel, discordgo.VerificationLevelHigh, info.GetBotName()))
		} else {
			g := discordgo.GuildParams{
				Name:                        "",
				Region:                      "",
				VerificationLevel:           &w.lockdown,
				DefaultMessageNotifications: 0,
				AfkChannelID:                "",
				AfkTimeout:                  0,
				Icon:                        "",
				OwnerID:                     "",
				Splash:                      "",
			}
			_, err = info.Bot.DG.GuildEdit(info.ID, g)
		}
		if err != nil {
			info.SendMessage(modchan, fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_LOCKDOWN_DISENGAGE_FAILURE], info.Bot.AppName))
		} else {
			info.SendMessage(modchan, bot.StringMap[bot.STRING_SPAM_LOCKDOWN_DISENGAGE])
		}
		w.lockdown = -1
	}
}

func (w *SpamModule) checkRaid(info *bot.GuildInfo, m *discordgo.Member, t time.Time) {
	if !info.Bot.DB.CheckStatus() {
		return
	}
	raidsize := info.Bot.DB.CountNewUsers(info.Config.Spam.RaidTime, bot.SBatoi(info.ID))
	if info.Config.Spam.RaidSize > 0 && raidsize >= info.Config.Spam.RaidSize && bot.RateLimit(&info.LastRaid, info.Config.Spam.RaidTime*2, t.Unix()) {
		r := info.Bot.DB.GetNewestUsers(raidsize, bot.SBatoi(info.ID))
		s := make([]string, 0, len(r))

		for _, v := range r {
			s = append(s, fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_USER_JOINED], v.User.Username, info.ApplyTimezone(v.FirstSeen, bot.UserEmpty).Format(time.ANSIC)))
			if info.Config.Spam.RaidSilence >= 1 {
				if info.Config.Basic.MemberRole != bot.RoleEmpty {
					info.Bot.DG.GuildMemberRoleRemove(info.ID, v.User.ID, info.Config.Basic.MemberRole.String())
				} else {
					silenceMember(v.User, info)
				}
			}
		}
		ch := info.Config.Basic.ModChannel
		if info.Bot.Debug {
			ch, _ = info.Bot.DebugChannels[bot.DiscordGuild(info.ID)]
		}
		message := fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_ALL_POSTFIX], info.Config.Basic.CommandPrefix)
		if info.Config.Spam.RaidSilence > 0 {
			message = bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_ENGAGED]
		}
		go info.SendMessage(ch, info.Config.Basic.ModRole.Display()+bot.StringMap[bot.STRING_SPAM_RAID_DETECTED]+message+"\n```"+strings.Join(s, "\n")+"```")
		if info.Config.Spam.LockdownDuration > 0 {
			if w.lockdown == -1 { // Only engage lockdown if it wasn't already engaged
				guild, err := info.GetGuild()
				if err != nil {
					w.lockdown = discordgo.VerificationLevelHigh
				} else {
					w.lockdown = guild.VerificationLevel
				}
				level := discordgo.VerificationLevelHigh
				g := discordgo.GuildParams{"", "", &level, 0, "", 0, "", "", ""}
				_, err = info.Bot.DG.GuildEdit(info.ID, g)
				if err != nil {
					info.SendMessage(ch, fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_LOCKDOWN_ENGAGE_FAILURE], info.GetBotName(), info.Config.Basic.CommandPrefix))
				} else {
					info.SendMessage(ch, fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_LOCKDOWN_ENGAGE], info.Config.Spam.LockdownDuration, info.Config.Basic.CommandPrefix))
				}
			}
			// Otherwise just reset the timer
			w.lastlockdown = t
		}
	}
}

// OnGuildMemberAdd discord hook
func (w *SpamModule) OnGuildMemberAdd(info *bot.GuildInfo, m *discordgo.Member, t time.Time) {
	w.silenceLock.Lock()
	if _, ok := w.resilence[bot.DiscordUser(m.User.ID)]; ok || info.Config.Spam.RaidSilence >= 2 || (info.Config.Spam.RaidSilence >= 1 && ((info.LastRaid + info.Config.Spam.RaidTime*2) > t.Unix())) {
		if info.Config.Basic.MemberRole == bot.RoleEmpty {
			silenceMember(m.User, info)
		}
		if len(info.Config.Users.WelcomeMessage) > 0 {
			defer info.SendMessage(info.Config.Users.WelcomeChannel, "<@"+m.User.ID+"> "+info.Config.Users.WelcomeMessage)
		}
	} else if info.Config.Basic.MemberRole != bot.RoleEmpty {
		defer info.Bot.DG.GuildMemberRoleAdd(info.ID, m.User.ID, info.Config.Basic.MemberRole.String())
	}
	w.silenceLock.Unlock()

	w.checkRaid(info, m, t)
}

// OnGuildMemberRemove discord hook
func (w *SpamModule) OnGuildMemberRemove(info *bot.GuildInfo, m *discordgo.Member, t time.Time) {
	w.silenceLock.Lock()
	defer w.silenceLock.Unlock()
	if _, ok := w.silenced[bot.DiscordUser(m.User.ID)]; ok {
		w.resilence[bot.DiscordUser(m.User.ID)] = true
		delete(w.silenced, bot.DiscordUser(m.User.ID))
	}
}

// OnGuildMemberUpdate discord hook
func (w *SpamModule) OnGuildMemberUpdate(info *bot.GuildInfo, m *discordgo.Member, t time.Time) {
	// Discord sends an OnGuildMemberUpdate *before* the OnGuildMemberAdd event, and OnGuildMemberRemove does not
	// include the roles of the member because the member has already been removed from the state, so we have to do
	// all the resilence logic in here using two maps instead of just one.
	w.silenceLock.Lock()
	if _, ok := w.resilence[bot.DiscordUser(m.User.ID)]; ok {
		delete(w.resilence, bot.DiscordUser(m.User.ID)) // Delete this first so the next OnGuildMemberUpdate will trigger the add/delete code path
		defer info.Bot.DG.GuildMemberRoleAdd(info.ID, m.User.ID, info.Config.Basic.SilenceRole.String())
	} else {
		if bot.MemberHasRole(m, info.Config.Basic.SilenceRole) {
			w.silenced[bot.DiscordUser(m.User.ID)] = true
		} else {
			delete(w.silenced, bot.DiscordUser(m.User.ID))
		}
	}
	w.silenceLock.Unlock()

	w.checkRaid(info, m, t)
}

func (w *SpamModule) getRaidUsers(info *bot.GuildInfo) []*discordgo.User {
	return info.Bot.DB.GetRecentUsers(time.Unix(info.LastRaid-info.Config.Spam.RaidTime, 0).UTC(), bot.SBatoi(info.ID))
}
func (w *SpamModule) isRecentRaid(info *bot.GuildInfo, t time.Time) bool {
	return info.LastRaid+info.Config.Spam.RaidTime*2 > t.Unix()
}

type raidSilenceCommand struct {
	s *SpamModule
}

func (c *raidSilenceCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "RaidSilence",
		Usage:     bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_USAGE],
		Sensitive: true,
	}
}
func (c *raidSilenceCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_ARGS_ERROR], false, nil
	}
	timestamp := bot.GetTimestamp(msg)

	switch strings.ToLower(args[0]) {
	case "all":
		info.Config.Spam.RaidSilence = 2
	case "raid":
		info.Config.Spam.RaidSilence = 1
	case "off":
		info.Config.Spam.RaidSilence = 0
	/*case "debug":
	var subtract int64
	if len(args) > 1 {
		subtract, _ = strconv.ParseInt(args[1], 10, 64)
	}
	info.LastRaid = timestamp.Unix() - subtract
	fmt.Println(time.Unix(info.LastRaid, 0))*/
	default:
		return bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_ARGS], false, nil
	}

	info.SaveConfig()

	if info.Config.Spam.RaidSilence <= 0 {
		c.s.DisableLockdown(info)
	} else if c.s.isRecentRaid(info, timestamp) { // If there has recently been a raid, silence everyone who joined or theoretically could have joined since the beginning of the raid.
		c.s.lastlockdown = timestamp // Reset lockdown timer just in case
		if !info.Bot.DB.CheckStatus() {
			return bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_DATABASE_ERROR], false, nil
		}
		// BEFORE we make any calls to discord, which could take some time, immediately respond with a silence set message so the admins know the command is functioning
		go info.SendMessage(bot.DiscordChannel(msg.ChannelID), fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_SET_RAID], strings.ToLower(args[0])))
		r := c.s.getRaidUsers(info)
		s := make([]string, 0, len(r))
		s = append(s, bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_DETECTION])
		for _, v := range r {
			s = append(s, v.Username)
			if info.Config.Basic.MemberRole != bot.RoleEmpty {
				info.Bot.DG.GuildMemberRoleRemove(info.ID, v.ID, info.Config.Basic.MemberRole.String())
			} else {
				silenceMember(v, info)
			}
		}
		return strings.Join(s, "\n") + "```", false, nil
	}
	return fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_SET], strings.ToLower(args[0])), false, nil
}
func (c *raidSilenceCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_DESCRIPTION],
		Params: []bot.CommandUsageParam{
			{Name: "all/raid/off", Desc: bot.StringMap[bot.STRING_SPAM_RAIDSILENCE_DESCRIPTION_NAME], Optional: false},
		},
	}
}

type wipeCommand struct {
}

func (c *wipeCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "Wipe",
		Usage:     bot.StringMap[bot.STRING_SPAM_WIPE_USAGE],
		Sensitive: true,
	}
}
func (c *wipeCommand) WipeMessages(ch *discordgo.Channel, num int, seconds int, timestamp time.Time, info *bot.GuildInfo) (int, error) {
	date := timestamp.Add(time.Duration(-seconds) * time.Second)

	ret := 0
	lastid := ""
	for ret < num {
		n := num - ret
		if n > 99 {
			n = 99
		}
		list, err := info.Bot.DG.ChannelMessages(ch.ID, n, lastid, "", "")
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
		if err = info.BulkDelete(ch, IDs); err != nil {
			return ret, err
		}

		lastid = IDs[len(IDs)-1]
	}
	return ret, nil
}
func (c *wipeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return bot.StringMap[bot.STRING_SPAM_WIPE_ARG_ERROR], false, nil
	}

	var err error
	messages := false
	ch := bot.DiscordChannel(msg.ChannelID)
	num := 0
	if len(args) > 1 {
		g, _ := info.GetGuild()
		ch, err = bot.ParseChannel(args[0], g)

		if err == nil {
			if args[1][len(args[1])-1] == 'm' {
				messages = true
				args[1] = args[1][:len(args[1])-1]
			}
			num, err = strconv.Atoi(args[1])
		}
	} else {
		if args[0][len(args[0])-1] == 'm' {
			messages = true
			args[0] = args[0][:len(args[0])-1]
		}
		num, err = strconv.Atoi(args[0])
	}
	if err != nil {
		return bot.ReturnError(err)
	}
	channel, private := info.Bot.ChannelIsPrivate(ch)
	if private {
		return bot.StringMap[bot.STRING_SPAM_WIPE_PM_ERROR], false, nil
	}
	if channel == nil || channel.GuildID != info.ID {
		return bot.StringMap[bot.STRING_SPAM_WIPE_CHANNEL_ERROR], false, nil
	}
	timestamp := bot.GetTimestamp(msg)
	if num <= 0 {
		return bot.StringMap[bot.STRING_SPAM_WIPE_NO_MESSAGES], false, nil
	}
	if messages {
		num, err = c.WipeMessages(channel, num, 0, timestamp, info)
	} else {
		num, err = c.WipeMessages(channel, 9999, num, timestamp, info)
	}
	if err != nil {
		return fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_WIPE_RETRIEVAL_ERROR], info.GetBotName(), err.Error()), false, nil
	}
	return fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_WIPE_DELETED], num, ch), false, nil
}
func (c *wipeCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_WIPE_DESCRIPTION], info.Config.Basic.CommandPrefix, info.Config.Basic.CommandPrefix),
		Params: []bot.CommandUsageParam{
			{Name: "channel", Desc: bot.StringMap[bot.STRING_SPAM_WIPE_CHANNEL], Optional: true},
			{Name: "seconds/messages", Desc: bot.StringMap[bot.STRING_SPAM_WIPE_MESSAGES], Optional: false},
		},
	}
}

type getPressureCommand struct {
	s *SpamModule
}

func (c *getPressureCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "GetPressure",
		Usage:     bot.StringMap[bot.STRING_SPAM_PRESSURE_USAGE],
		Sensitive: true,
	}
}

func (c *getPressureCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return bot.StringMap[bot.STRING_SPAM_PRESSURE_ARG_ERROR], false, nil
	}

	user, err := bot.ParseUser(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(err)
	}

	u, ok := c.s.tracker.Load(user)
	if !ok {
		return "0", false, nil
	}
	return fmt.Sprint(u.(*userPressure).pressure), false, nil
}
func (c *getPressureCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: bot.StringMap[bot.STRING_SPAM_PRESSURE_DESCRIPTION],
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: bot.StringMap[bot.STRING_SPAM_PRESSURE_USER], Optional: false},
		},
	}
}

type getRaidCommand struct {
	s *SpamModule
}

func (c *getRaidCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "GetRaid",
		Usage:     bot.StringMap[bot.STRING_SPAM_RAID_USAGE],
		Sensitive: true,
	}
}

func (c *getRaidCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !c.s.isRecentRaid(info, bot.GetTimestamp(msg)) {
		return fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_RAID_NONE], bot.TimeDiff(time.Duration(info.Config.Spam.RaidTime*2)*time.Second)), false, nil
	}
	s := []string{bot.StringMap[bot.STRING_SPAM_RAID_USERS]}
	for _, v := range c.s.getRaidUsers(info) {
		s = append(s, v.Username+"#"+v.Discriminator)
	}
	return "```\n" + strings.Join(s, "\n") + "```", false, nil
}
func (c *getRaidCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{Desc: bot.StringMap[bot.STRING_SPAM_RAID_DESCRIPTION]}
}

type banRaidCommand struct {
	s *SpamModule
}

func (c *banRaidCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "BanRaid",
		Usage:     bot.StringMap[bot.STRING_SPAM_BANRAID_USAGE],
		Sensitive: true,
	}
}
func (c *banRaidCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !c.s.isRecentRaid(info, bot.GetTimestamp(msg)) {
		return fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_RAID_NONE], bot.TimeDiff(time.Duration(info.Config.Spam.RaidTime*2)*time.Second)), false, nil
	}
	reason := fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_BANRAID_REASON], msg.Author.Username, msg.Author.Discriminator, info.Config.Basic.CommandPrefix)
	users := c.s.getRaidUsers(info)
	for _, v := range users {
		info.Bot.DG.GuildBanCreateWithReason(info.ID, v.ID, reason, 1)
	}
	return fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_BANRAID_RESULT], len(users)), false, nil
}
func (c *banRaidCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{Desc: fmt.Sprintf(bot.StringMap[bot.STRING_SPAM_BANRAID_DESCRIPTION], info.Config.Basic.CommandPrefix)}
}
