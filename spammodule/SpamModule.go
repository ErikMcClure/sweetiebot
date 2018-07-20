package spammodule

import (
	"container/heap"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	bot "../sweetiebot"
	"github.com/blackhole12/discordgo"
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
}

// New spam module
func New() *SpamModule {
	w := &SpamModule{
		lockdown: -1,
		timeouts: &userTimeoutHeap{},
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
func (w *SpamModule) Description() string {
	return "Tracks all channels it is active on for spammers. Each message someone sends generates \"pressure\", which decays rapidly. Long messages, messages with links, or messages with pings will generate more pressure. If a user generates too much pressure, they will be silenced and the moderators notified. Also detects groups of people joining at the same time and alerts the moderators of a potential raid."
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
			info.SendMessage(info.Config.Basic.ModChannel, "```\nError unsilencing member: "+err.Error()+"```")
		}
		info.SendMessage(info.Config.Basic.ModChannel, "```\nUnsilenced "+info.GetUserName(u)+".```")
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
		msgembeds = "\nEmbedded URLs: "
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
		lastmsg += "... [truncated]"
	} else if len(lastmsg) > 300 {
		lastmsg = lastmsg[:300] + "... [truncated]"
	}
	logmsg := fmt.Sprintf("Killing spammer %s (pressure: %v -> %v). Last message sent on #%s in %s: \n%s%s", u.Username, oldpressure, newpressure, chname, info.Name, lastmsg, msgembeds)
	if info.Config.Users.WelcomeChannel.Equals(msg.ChannelID) {
		info.Bot.DG.GuildBanCreateWithReason(info.ID, u.ID, "Autobanned for "+reason+" in the welcome channel.", 1)
		info.SendMessage(info.Config.Basic.ModChannel, "Alert: <@"+u.ID+"> was banned for "+reason+" in the welcome channel.")
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
			info.LogError("Error encountered while attempting to retrieve messages: ", err)
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
			addmsg = ", or they will be unsilenced automatically in " + bot.TimeDiff(timeout)
			w.timeoutLock.Lock()
			heap.Push(w.timeouts, userTimeout{bot.DiscordUser(u.ID), timestamp.Add(timeout)})
			w.timeoutLock.Unlock()
		}
		info.SendMessage(info.Config.Basic.ModChannel, "Alert: <@"+u.ID+"> was silenced for "+reason+". Please investigate"+addmsg) // Alert admins
		info.Log(logmsg)
	} else {
		info.Log("Killing spammer " + u.Username)
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
		if info.UserHasRole(author, info.Config.Basic.SilenceRole) && !info.Config.Users.WelcomeChannel.Equals(m.ChannelID) {
			ch, _ := info.Bot.DG.Channel(m.ChannelID)
			time.Sleep(bot.DelayTime)
			info.ChannelMessageDelete(ch, m.ID)
			return true
		}
		if info.UserIsMod(author) || info.UserIsAdmin(author) ||
			(info.Config.Spam.IgnoreRole != bot.RoleEmpty && info.UserHasRole(author, info.Config.Spam.IgnoreRole)) ||
			m.Author.Bot {
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

		if w.AddPressure(info, m, track, info.Config.Spam.BasePressure, "spamming too many messages") {
			return true
		}
		if w.AddPressure(info, m, track, info.Config.Spam.ImagePressure*float32(len(m.Attachments)), "attaching too many files") {
			return true
		}
		if w.AddPressure(info, m, track, info.Config.Spam.ImagePressure*float32(len(m.Embeds)), "spamming too many images") {
			return true
		}
		if w.AddPressure(info, m, track, info.Config.Spam.PingPressure*float32(len(m.Mentions)), "pinging too many people") {
			return true
		}
		if w.AddPressure(info, m, track, info.Config.Spam.LengthPressure*float32(len(m.Content)), "sending a really long message") {
			return true
		}
		if w.AddPressure(info, m, track, info.Config.Spam.LinePressure*float32(strings.Count(m.Content, "\n")), "Using too many newlines") {
			return true
		}
		if len(m.Content) > 0 && strings.ToLower(m.Content) == track.lastcache {
			if w.AddPressure(info, m, track, info.Config.Spam.RepeatPressure, "copy+pasting the same message") {
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
			info.SendMessage(modchan, "Guild cannot be found in state?!")
		} else if guild.VerificationLevel != discordgo.VerificationLevelHigh {
			info.SendMessage(modchan, fmt.Sprintf("The verification level is at %v instead of %v, which means it was manually changed by someone other than "+info.GetBotName()+", so it has not been restored.", guild.VerificationLevel, discordgo.VerificationLevelHigh))
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
			info.SendMessage(modchan, "Could not disengage lockdown! Make sure you've given the "+info.Bot.AppName+" role the Manage Server permission, you'll have to manually restore it yourself this time.")
		} else {
			info.SendMessage(modchan, "Lockdown disengaged, server verification levels restored.")
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
			s = append(s, v.User.Username+"  (joined: "+info.ApplyTimezone(v.FirstSeen, bot.UserEmpty).Format(time.ANSIC)+")")
			if info.Config.Spam.RaidSilence >= 1 {
				silenceMember(v.User, info)
			}
		}
		ch := info.Config.Basic.ModChannel
		if info.Bot.Debug {
			ch, _ = info.Bot.DebugChannels[bot.DiscordGuild(info.ID)]
		}
		message := "Use `" + info.Config.Basic.CommandPrefix + "raidsilence all` to silence them!"
		if info.Config.Spam.RaidSilence > 0 {
			message = "RaidSilence has been engaged and the following users silenced:"
		}
		go info.SendMessage(ch, info.Config.Basic.ModRole.Display()+" Possible Raid Detected! "+message+"\n```"+strings.Join(s, "\n")+"```")
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
					info.SendMessage(ch, "Could not engage lockdown! Make sure you've given "+info.GetBotName()+" the Manage Server permission, or disable the lockdown entirely via `"+info.Config.Basic.CommandPrefix+"setconfig spam.lockdownduration 0`.")
				} else {
					info.SendMessage(ch, fmt.Sprintf("Lockdown engaged! Server verification level will be reset in %v seconds. This lockdown can be manually ended via `"+info.Config.Basic.CommandPrefix+"raidsilence off/alert/log`.", info.Config.Spam.LockdownDuration))
				}
			}
			// Otherwise just reset the timer
			w.lastlockdown = t
		}
	}
}

// OnGuildMemberAdd discord hook
func (w *SpamModule) OnGuildMemberAdd(info *bot.GuildInfo, m *discordgo.Member, t time.Time) {
	if info.Config.Spam.RaidSilence >= 2 || (info.Config.Spam.RaidSilence >= 1 && ((info.LastRaid + info.Config.Spam.RaidTime*2) > t.Unix())) {
		silenceMember(m.User, info)
		if len(info.Config.Users.WelcomeMessage) > 0 {
			info.SendMessage(info.Config.Users.WelcomeChannel, "<@"+m.User.ID+"> "+info.Config.Users.WelcomeMessage)
		}
	}
	w.checkRaid(info, m, t)
}

// OnGuildMemberUpdate discord hook
func (w *SpamModule) OnGuildMemberUpdate(info *bot.GuildInfo, m *discordgo.Member, t time.Time) {
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
		Usage:     "Toggle raid silencing.",
		Sensitive: true,
	}
}
func (c *raidSilenceCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou must provide a raid silence level (either all, raid, or off).```", false, nil
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
		return "```\nOnly all, raid, and off are valid raid silence levels.```", false, nil
	}

	info.SaveConfig()

	if info.Config.Spam.RaidSilence <= 0 {
		c.s.DisableLockdown(info)
	} else if c.s.isRecentRaid(info, timestamp) { // If there has recently been a raid, silence everyone who joined or theoretically could have joined since the beginning of the raid.
		c.s.lastlockdown = timestamp // Reset lockdown timer just in case
		if !info.Bot.DB.CheckStatus() {
			return "```\nRaidSilence was engaged, but a database error prevents me from retroactively applying it!```", false, nil
		}
		// BEFORE we make any calls to discord, which could take some time, immediately respond with a silence set message so the admins know the command is functioning
		go info.SendMessage(bot.DiscordChannel(msg.ChannelID), "```\nSet the raid silence level to "+strings.ToLower(args[0])+".```")
		r := c.s.getRaidUsers(info)
		s := make([]string, 0, len(r))
		s = append(s, "```\nDetected a recent raid. All users from the raid have been silenced:")
		for _, v := range r {
			s = append(s, v.Username)
			silenceMember(v, info)
		}
		return strings.Join(s, "\n") + "```", false, nil
	}
	return "```\nSet the raid silence level to " + strings.ToLower(args[0]) + ".```", false, nil
}
func (c *raidSilenceCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Toggles silencing new members during raids. This does not affect spam detection, only new members joining the server.",
		Params: []bot.CommandUsageParam{
			{Name: "all/raid/off", Desc: "`all` will always silence all new members. `raid` will only silence new members if a raid is detected, up to `spam.raidtime*2` seconds after the raid is detected. `off` disables raid silencing.", Optional: false},
		},
	}
}

type wipeCommand struct {
}

func (c *wipeCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "Wipe",
		Usage:     "Wipes a given channel",
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
		return "```\nYou must specify the duration.```", false, nil
	}

	var err error
	messages := false
	ch := bot.DiscordChannel(msg.ChannelID)
	num := 0
	if len(args) > 1 {
		if strings.ToLower(args[1]) == "messages" {
			messages = true
			num, err = strconv.Atoi(args[0])
		} else {
			g, _ := info.GetGuild()
			ch, err = bot.ParseChannel(args[0], g)
			if err == nil {
				num, err = strconv.Atoi(args[1])
			}

			if len(args) > 2 {
				messages = strings.ToLower(args[2]) == "messages"
			}
		}
		if err != nil {
			return bot.ReturnError(err)
		}
	}
	channel, private := info.Bot.ChannelIsPrivate(ch)
	if private {
		return "```\nCan't delete messages in a PM!```", false, nil
	}
	if channel == nil || channel.GuildID != info.ID {
		return "```\nThat channel isn't on this server!```", false, nil
	}
	timestamp := bot.GetTimestamp(msg)
	if num <= 0 {
		return "```\nThere's no point deleting 0 messages!.```", false, nil
	}
	if messages {
		num, err = c.WipeMessages(channel, num, 0, timestamp, info)
	} else {
		num, err = c.WipeMessages(channel, 9999, num, timestamp, info)
	}
	if err != nil {
		return "```\nError retrieving messages. Are you sure you gave " + info.GetBotName() + " a channel that exists? This won't work in PMs! " + err.Error() + "```", false, nil
	}
	return fmt.Sprintf("Deleted %v messages in <#%s>.", num, ch), false, nil
}
func (c *wipeCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Removes all messages in a channel sent within the last N seconds, or simply removes the last N messages if \"messages\" is appended.",
		Params: []bot.CommandUsageParam{
			{Name: "channel", Desc: "The channel to delete from. You must use the #channel format so discord actually highlights the channel, otherwise it won't work. If omitted, uses the current channel", Optional: true},
			{Name: "seconds", Desc: "Specifies the number of seconds to look back. The command deletes all messages sent up to this many seconds ago.", Optional: false},
			{Name: "MESSAGES", Desc: "If you append \"MESSAGES\" to the end of the command, it will remove that many messages, instead of looking back that many seconds.", Optional: true},
		},
	}
}

type getPressureCommand struct {
	s *SpamModule
}

func (c *getPressureCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "GetPressure",
		Usage:     "Gets a user's pressure.",
		Sensitive: true,
	}
}

func (c *getPressureCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou must provide a user to search for.```", false, nil
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
		Desc: "Gets the current spam pressure of a user.",
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: "User to retrieve pressure from.", Optional: false},
		},
	}
}

type getRaidCommand struct {
	s *SpamModule
}

func (c *getRaidCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "GetRaid",
		Usage:     "Lists users in most recent raid.",
		Sensitive: true,
	}
}

func (c *getRaidCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !c.s.isRecentRaid(info, bot.GetTimestamp(msg)) {
		return fmt.Sprintf("```\nNo raid has occurred within the past %s.```", bot.TimeDiff(time.Duration(info.Config.Spam.RaidTime*2)*time.Second)), false, nil
	}
	s := []string{"Users in latest raid: "}
	for _, v := range c.s.getRaidUsers(info) {
		s = append(s, v.Username+"#"+v.Discriminator)
	}
	return "```\n" + strings.Join(s, "\n") + "```", false, nil
}
func (c *getRaidCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{Desc: "Lists all users that are considered part of the most recent raid, if there was one."}
}

type banRaidCommand struct {
	s *SpamModule
}

func (c *banRaidCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "BanRaid",
		Usage:     "Bans all users in most recent raid.",
		Sensitive: true,
	}
}
func (c *banRaidCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !c.s.isRecentRaid(info, bot.GetTimestamp(msg)) {
		return fmt.Sprintf("```\nNo raid has occurred within the past %s.```", bot.TimeDiff(time.Duration(info.Config.Spam.RaidTime*2)*time.Second)), false, nil
	}
	reason := fmt.Sprintf("Banned by %s#%s via the !banraid command.", msg.Author.Username, msg.Author.Discriminator)
	users := c.s.getRaidUsers(info)
	for _, v := range users {
		info.Bot.DG.GuildBanCreateWithReason(info.ID, v.ID, reason, 1)
	}
	return fmt.Sprintf("```\nBanned %v users. The ban log will reflect who ran this command.```", len(users)), false, nil
}
func (c *banRaidCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{Desc: "Bans all users that are considered part of the most recent raid, if there was one. Use " + info.Config.Basic.CommandPrefix + "getraid to check who will be banned before using this command."}
}
