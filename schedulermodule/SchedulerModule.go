package schedulermodule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	bot "../sweetiebot"
	"github.com/blackhole12/discordgo"
)

// SchedulerModule manages the scheduling system
type SchedulerModule struct {
}

var repeatregex = regexp.MustCompile("repeat -?[0-9]+ (second|minute|hour|day|week|month|quarter|year)s?")

const ( // We don't use iota here because this must match the database values exactly
	typeEventBan        = 0
	typeEventBirthday   = 1
	typeEventMessage    = 2
	typeEventEpisode    = 3
	typeEventUnbirthday = 4
	typeEvent           = 5
	typeEventReminder   = 6
	typeEventRole       = 7
	typeEventSilence    = 8
	typeEventRemoveRole = 9
)

// New SchedulerModule
func New() *SchedulerModule {
	return &SchedulerModule{}
}

// Name of the module
func (w *SchedulerModule) Name() string {
	return "Scheduler"
}

// Commands in the module
func (w *SchedulerModule) Commands() []bot.Command {
	return []bot.Command{
		&scheduleCommand{},
		&nextCommand{},
		&addEventCommand{},
		&removeEventCommand{},
		&remindMeCommand{},
		&addBirthdayCommand{},
	}
}

// Description of the module
func (w *SchedulerModule) Description(info *bot.GuildInfo) string {
	return "Manages the scheduling system, and periodically checks for events that need to be processed. To change what channel this module sends birthday notifications or other events to, use `!setconfig modules.channels scheduler #channelname`."
}

// OnTick discord hook
func (w *SchedulerModule) OnTick(info *bot.GuildInfo, t time.Time) {
	if !info.Bot.DB.CheckStatus() {
		return
	}
	events := info.Bot.DB.GetSchedule(bot.SBatoi(info.ID))
	if len(events) == 0 {
		return
	}
	channel := info.Config.Basic.ModChannel
	modulename := bot.ModuleID(strings.ToLower(w.Name()))
	if len(info.Config.Modules.Channels[modulename]) > 0 {
		for k := range info.Config.Modules.Channels[modulename] {
			if k != bot.ChannelEmpty && k != bot.ChannelExclusion {
				channel = k
				break
			}
		}
	} else if len(info.Config.Modules.Channels["bored"]) > 0 {
		for k := range info.Config.Modules.Channels["bored"] {
			if k != bot.ChannelEmpty && k != bot.ChannelExclusion {
				channel = k
				break
			}
		}
	} else if len(info.Config.Basic.FreeChannels) > 0 {
		for k := range info.Config.Basic.FreeChannels {
			if k != bot.ChannelEmpty && k != bot.ChannelExclusion {
				channel = k
				break
			}
		}
	}

	if channel == bot.ChannelEmpty {
		return
	}
	if ch, private := info.Bot.ChannelIsPrivate(channel); private || ch == nil || ch.GuildID != info.ID {
		channel = info.Config.Basic.ModChannel
	}

	for _, v := range events {
		switch v.Type {
		case typeEventBan:
			err := info.Bot.DG.GuildBanDelete(info.ID, v.Data)
			if err != nil {
				info.SendMessage(info.Config.Basic.ModChannel, "Error unbanning <@"+v.Data+">: "+err.Error())
			} else {
				info.SendMessage(info.Config.Basic.ModChannel, "Unbanned <@"+v.Data+">")
			}
		case typeEventBirthday:
			if info.Config.Scheduler.BirthdayRole == bot.RoleEmpty {
				info.Log("No birthday role set!")
			} else {
				err := info.ResolveRoleAddError(info.Bot.DG.GuildMemberRoleAdd(info.ID, v.Data, info.Config.Scheduler.BirthdayRole.String()))
				info.LogError("Failed to set birthday role: ", err)
			}
			info.SendMessage(channel, "Happy Birthday <@"+v.Data+">!")
		case typeEventMessage:
			info.SendMessage(channel, v.Data)
		case typeEvent, typeEventEpisode:
			info.SendMessage(channel, v.Data+" is starting now!")
		case typeEventUnbirthday:
			if info.Config.Scheduler.BirthdayRole == bot.RoleEmpty {
				info.Log("No birthday role set!")
			} else {
				err := info.ResolveRoleAddError(info.Bot.DG.GuildMemberRoleRemove(info.ID, v.Data, info.Config.Scheduler.BirthdayRole.String()))
				info.LogError("Failed to remove birthday role: ", err)
			}
		case typeEventReminder:
			dat := strings.SplitN(v.Data, "|", 2)
			ch, err := info.Bot.DG.UserChannelCreate(dat[0])
			info.LogError("Error opening private channel: ", err)
			if err == nil {
				info.SendMessage(bot.DiscordChannel(ch.ID), dat[1])
			}
		case typeEventRole:
			dat := strings.SplitN(v.Data, "|", 2)
			info.SendMessage(channel, dat[0]+" "+dat[1])
		case typeEventSilence:
			err := info.ResolveRoleAddError(info.Bot.DG.RemoveRole(info.ID, bot.DiscordUser(v.Data), info.Config.Basic.SilenceRole))
			if err != nil {
				info.SendMessage(info.Config.Basic.ModChannel, "Error unsilencing <@"+v.Data+">: "+err.Error())
			} else {
				info.SendMessage(info.Config.Basic.ModChannel, "Unsilenced <@"+v.Data+">")
			}
		case typeEventRemoveRole:
			dat := strings.SplitN(v.Data, "|", 2)
			if len(dat) != 2 {
				info.SendMessage(info.Config.Basic.ModChannel, "Invalid data in role removal event: "+v.Data)
			} else {
				role := bot.DiscordRole(dat[1])
				err := info.ResolveRoleAddError(info.Bot.DG.RemoveRole(info.ID, bot.DiscordUser(dat[0]), role))
				if err != nil {
					info.SendMessage(info.Config.Basic.ModChannel, "Error removing "+role.Show(info)+" from <@"+dat[0]+">: "+err.Error())
				} else {
					info.SendMessage(info.Config.Basic.ModChannel, "Removed "+role.Show(info)+" from <@"+dat[0]+">")
				}
			}
		}

		info.Bot.DB.RemoveSchedule(v.ID)
	}
}

type scheduleCommand struct {
}

func (c *scheduleCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "Schedule",
		Usage: "Gets a list of upcoming scheduled events.",
	}
}
func (c *scheduleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	timestamp := bot.GetTimestamp(msg)
	maxresults := 5
	var ty uint8
	ty = 255
	if len(args) > 1 {
		ty = getScheduleType(args[0])
		if ty == 255 {
			return "```\nUnknown schedule type.```", false, nil
		}
		maxresults, _ = strconv.Atoi(args[1])
	} else if len(args) > 0 {
		var err error
		maxresults, err = strconv.Atoi(args[0])
		if err != nil {
			maxresults = 5
			ty = getScheduleType(args[0])
			if ty == 255 {
				return "```\nUnknown schedule type.```", false, nil
			}
		}
	}
	if maxresults > 20 {
		maxresults = 20
	}
	if maxresults < 1 {
		maxresults = 1
	}
	if !info.UserIsMod(bot.DiscordUser(msg.Author.ID)) && !info.UserIsAdmin(bot.DiscordUser(msg.Author.ID)) && (ty == 0 || ty == 4 || ty == 8) {
		return "```\nYou aren't allowed to view those events.```", false, nil
	}
	var events []bot.ScheduleEvent
	if ty == 255 {
		events = info.Bot.DB.GetEvents(bot.SBatoi(info.ID), maxresults)
	} else if ty == 6 {
		events = info.Bot.DB.GetReminders(bot.SBatoi(info.ID), msg.Author.ID, maxresults)
	} else {
		events = info.Bot.DB.GetEventsByType(bot.SBatoi(info.ID), ty, maxresults)
	}
	if len(events) == 0 {
		return "There are no upcoming events.", false, nil
	}
	lines := make([]string, len(events)+1, len(events)+1)
	lines[0] = "Upcoming Events:"
	for k, v := range events {
		t := ""
		if v.Date.Year() == timestamp.Year() {
			t = info.ApplyTimezone(v.Date, bot.DiscordUser(msg.Author.ID)).Format("Jan 2 3:04pm")
		} else {
			t = info.ApplyTimezone(v.Date, bot.DiscordUser(msg.Author.ID)).Format("Jan 2 2006 3:04pm")
		}
		data := v.Data
		mt := "UNKNOWN"
		switch v.Type {
		case typeEventBan:
			mt = "UNBAN"
			data = "<@" + data + ">"
		case typeEventBirthday:
			mt = "BIRTHDAY"
			data = "<@" + data + ">"
		case typeEventMessage:
			mt = "MESSAGE"
		case typeEventEpisode:
			mt = "EPISODE"
		case typeEvent:
			mt = "EVENT"
		case typeEventReminder:
			mt = "REMINDER"
			data = strings.SplitN(data, "|", 2)[1]
		case typeEventRole:
			datas := strings.SplitN(data, "|", 2)
			mt = "ROLE:" + bot.ReplaceAllRolePings(datas[0], info)
			data = datas[1]
		case typeEventRemoveRole:
			datas := strings.SplitN(data, "|", 2)
			mt = "REMOVAL:" + bot.DiscordRole(datas[1]).Show(info)
			data = "<@" + datas[0] + ">"
		}
		lines[k+1] = fmt.Sprintf("#%v **%s** [%s] %s", bot.SBitoa(v.ID), t, mt, info.Sanitize(data, bot.CleanMentions|bot.CleanPings|bot.CleanEmotes))
	}

	return strings.Join(lines, "\n"), len(lines) > 6, nil
}
func (c *scheduleCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Lists up to `maxresults` upcoming events from the schedule. If the first argument is specified, lists only events of that type. Some event types can only be viewed by moderators. Max results: 20",
		Params: []bot.CommandUsageParam{
			{Name: "type", Desc: "Can be one of: bans, birthdays, messages, episodes, events, roles, reminders.", Optional: true},
			{Name: "maxresults", Desc: "Defaults to 5.", Optional: true},
		},
	}
}

func getScheduleType(s string) uint8 {
	switch strings.ToLower(s) {
	case "bans", "ban":
		return typeEventBan
	case "birthdays", "birthday":
		return typeEventBirthday
	case "messages", "message":
		return typeEventMessage
	case "episodes", "episode":
		return typeEventEpisode
	case "events", "event":
		return typeEvent
	case "reminders", "reminder":
		return typeEventReminder
	case "roles", "role":
		return typeEventRole
	case "silences", "silence":
		return typeEventSilence
	case "removals", "removal":
		return typeEventRemoveRole
	}
	return 255
}

type nextCommand struct {
}

func (c *nextCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "Next",
		Usage: "Gets time until next event.",
	}
}
func (c *nextCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou must specify an event type.```", false, nil
	}
	ty := getScheduleType(args[0])
	if ty == 255 {
		return "```\nError: Invalid type specified.```", false, nil
	}
	timestamp := bot.GetTimestamp(msg)
	event := info.Bot.DB.GetNextEvent(bot.SBatoi(info.ID), ty)
	if event.Type > 0 && event.Date.Before(timestamp) {
		return "```\n" + info.GetBotName() + " will announce this event in just a moment!```", false, nil
	}
	diff := bot.TimeDiff(event.Date.Sub(timestamp))
	switch event.Type {
	case typeEventBirthday:
		return info.Sanitize("```\nIt'll be <@"+event.Data+">'s birthday in "+diff+"```", bot.CleanMentions|bot.CleanPings|bot.CleanEmotes), false, nil
	case typeEventMessage:
		return "```\n" + info.GetBotName() + " is scheduled to send a message in " + diff + "```", false, nil
	case typeEventEpisode:
		return "```\n" + event.Data + " airs in " + diff + "```", false, nil
	case typeEvent:
		return "```\n" + event.Data + " starts in " + diff + "```", false, nil
	case typeEventRole:
		return "```\n" + info.GetBotName() + " is scheduled to send a message to " + bot.ReplaceAllRolePings(strings.SplitN(event.Data, "|", 2)[0], info) + " in " + diff + "```", false, nil
	default:
		return "```\nThere are no upcoming events of that type (or you aren't allowed to view them).```", false, nil
	}
}
func (c *nextCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Gets the time until the next event of the given type.",
		Params: []bot.CommandUsageParam{
			{Name: "type", Desc: "Can be one of: bans, birthdays, messages, episodes, events, reminders.", Optional: true},
		},
	}
}

type addEventCommand struct {
}

func (c *addEventCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "AddEvent",
		Usage:     "Adds an event to the schedule.",
		Sensitive: true,
	}
}

func (c *addEventCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 2 {
		return "```\nAt least a type and a date must be specified!```", false, nil
	}
	ty := getScheduleType(args[0])
	if ty == 255 {
		return "```\nError: Invalid type specified.```", false, nil
	}
	if ty == typeEventReminder {
		return "```\nError: You cannot add a reminder event this way. Use " + info.Config.Basic.CommandPrefix + "remindme instead.```", false, nil
	}
	data := ""
	if ty == typeEventRole {
		data = strings.ToLower(args[1])
		data += "|"
		args = append(args[:1], args[2:]...)
		indices = append(indices[:1], indices[2:]...)
	}
	timestamp := bot.GetTimestamp(msg)
	t, err := info.ParseCommonTime(args[1], bot.DiscordUser(msg.Author.ID), timestamp)
	if err != nil {
		return "```\nError: Could not parse time! Make sure it's in the format \"2 January 2006 3:04pm -0700\" (or something similar, year, time and timezone are optional)```", false, nil
	}
	t = t.UTC()
	if t.Before(timestamp) {
		return "```\nError: Cannot specify an event in the past!```", false, nil
	}

	if len(args) > 2 && repeatregex.MatchString(strings.ToLower(args[2])) {
		repeats := strings.Split(args[2], " ")
		repeat, err := strconv.Atoi(repeats[1])
		if err != nil {
			return "```\nError: Repeat number was not an integer.```", false, nil
		}

		repeatinterval := bot.ParseRepeatInterval(repeats[2])
		if repeatinterval == 255 {
			return "```\nError: unrecognized interval.```", false, nil
		}

		if len(args) > 3 {
			data += msg.Content[indices[3]:]
		}
		if err := info.Bot.DB.AddScheduleRepeat(bot.SBatoi(info.ID), t, repeatinterval, repeat, ty, data); err != nil {
			return bot.ReturnError(err)
		}
	} else {
		if len(args) > 2 {
			data += msg.Content[indices[2]:]
		}

		if err := info.Bot.DB.AddSchedule(bot.SBatoi(info.ID), t, ty, data); err != nil {
			return bot.ReturnError(err)
		}
	}

	return "```\nAdded event to schedule.```", false, nil
}
func (c *addEventCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Adds an arbitrary event to the schedule table. For example: `" + info.Config.Basic.CommandPrefix + "addevent message \"12 Jun 16\" \"REPEAT 1 YEAR\" happy birthday!`, or `" + info.Config.Basic.CommandPrefix + "addevent episode \"9 Dec 15\" Slice of Life`. ",
		Params: []bot.CommandUsageParam{
			{Name: "type", Desc: "Can be one of: ban, message, episode, event, role.", Optional: false},
			{Name: "role", Desc: "A ping of the role that should be notified. Only include this when using the role event type.", Optional: true},
			{Name: "date", Desc: "A date in the format `12 Jun 16 2:10pm`, in quotes. The time, year, and timezone are all optional.", Optional: false},
			{Name: "REPEAT N INTERVAL", Desc: "INTERVAL can be one of SECONDS/MINUTES/HOURS/DAYS/WEEKS/MONTHS/YEARS. This parameter MUST be surrounded by quotes!", Optional: true},
		},
	}
}

func userOwnsEvent(e *bot.ScheduleEvent, u *discordgo.User) bool {
	if e.Type == 6 {
		dat := strings.SplitN(e.Data, "|", 2)
		if dat[0] == u.ID {
			return true
		}
	}
	return false
}

type removeEventCommand struct {
}

func (c *removeEventCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "RemoveEvent",
		Usage: "Remove an event from the schedule.",
	}
}

func (c *removeEventCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou must specify an event ID.```", false, nil
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return "```\nCould not parse event ID. Make sure you only specify the number itself.```", false, nil
	}

	e := info.Bot.DB.GetEvent(bot.DiscordGuild(info.ID).Convert(), id)
	if e == nil {
		return "```\nError: Event does not exist.```", false, nil
	}
	if !info.UserIsMod(bot.DiscordUser(msg.Author.ID)) && !info.UserIsAdmin(bot.DiscordUser(msg.Author.ID)) && !userOwnsEvent(e, msg.Author) {
		return "```\nError: You do not have permission to delete that event.```", false, nil
	}

	info.Bot.DB.DeleteSchedule(id)
	return "```\nRemoved Event #" + bot.SBitoa(id) + " from schedule.```", false, nil
}
func (c *removeEventCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Removes an event with the given ID from the schedule. ",
		Params: []bot.CommandUsageParam{
			{Name: "ID", Desc: "The event ID as gotten from a `" + info.Config.Basic.CommandPrefix + "schedule` command.", Optional: false},
		},
	}
}

type remindMeCommand struct {
}

func (c *remindMeCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "RemindMe",
		Usage: "Sets a reminder for you in the future.",
	}
}

func (c *remindMeCommand) Name() string {
	return "RemindMe"
}
func (c *remindMeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 3 {
		return "```\nYou must start your message with 'in' or 'on', followed by a date (in quotes!) or duration, followed by a message.```", false, nil
	}

	var t time.Time
	arg := ""
	timestamp := bot.GetTimestamp(msg)
	switch strings.ToLower(args[0]) {
	default: // You're supposed to use "in" here, but if we don't know what to do we just do this by default anyway.
		t = timestamp
		d, err := strconv.Atoi(args[1])
		if err != nil {
			return "```\nDuration is not numeric! Make sure it's in the format 'in 99 days', and DON'T put quotes around it.```", false, nil
		}
		if d <= 0 {
			return "```\nThat was " + bot.TimeDiff(timestamp.Sub(t)) + " ago, you idiot! Do you think I have a time machine or something?```", false, nil
		}
		switch bot.ParseRepeatInterval(args[2]) {
		case 1:
			t = t.Add(time.Duration(d) * time.Second)
		case 2:
			t = t.Add(time.Duration(d) * time.Minute)
		case 3:
			t = t.Add(time.Duration(d) * time.Hour)
		case 4:
			t = t.AddDate(0, 0, d)
		case 5:
			t = t.AddDate(0, 0, d*7)
		case 6:
			t = t.AddDate(0, d, 0)
		case 8:
			t = t.AddDate(d, 0, 0)
		default:
			return "```\nUnknown duration type! Acceptable types are seconds, minutes, hours, days, weeks, months, and years.```", false, nil
		}
		if len(indices) < 4 {
			return "```\nYou have to tell me what to say!```", false, nil
		}
		arg = msg.Content[indices[3]:]
	case "on":
		var err error
		t, err = info.ParseCommonTime(strings.ToLower(args[1]), bot.DiscordUser(msg.Author.ID), timestamp)
		if err != nil {
			return "```\nCould not parse time! Make sure its in the format \"2 January 2006 3:04pm -0700\" (or something similar, time, year, and timezone are optional). Make sure you surround it with quotes!```", false, nil
		}
		t = t.UTC()
		if t.Before(timestamp) {
			return "```\nThat was " + bot.TimeDiff(timestamp.Sub(t)) + " ago, dumbass! You have to give me a time that's in the FUTURE!```", false, nil
		}
		arg = msg.Content[indices[2]:]
	}

	if len(arg) == 0 {
		return "```\nWhat am I reminding you about? I can't send you a blank message!```", false, nil
	}
	if err := info.Bot.DB.AddSchedule(bot.SBatoi(info.ID), t, 6, msg.Author.ID+"|"+arg); err != nil {
		return bot.ReturnError(err)
	}
	return "Reminder set for " + bot.TimeDiff(t.Sub(timestamp)) + " from now.", false, nil
}
func (c *remindMeCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Tells " + info.GetBotName() + " to remind you about something in the future. ",
		Params: []bot.CommandUsageParam{
			{Name: "in N seconds/minutes/hours/etc.", Desc: "represents a time `N` units from the current time. The available units are: seconds, minutes, hours, days, weeks, months, years.", Optional: true},
			{Name: "on \"2 January 2006 3:04pm -0700\"", Desc: "represents an absolute date and time, which must be in quotes. You must choose the `in` syntax OR the `on` syntax to specify your time, not both.", Optional: true},
			{Name: "message", Desc: "An arbitrary string that will be sent to you at the appropriate time.", Optional: false},
		},
	}
}

type addBirthdayCommand struct {
}

func (c *addBirthdayCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "AddBirthday",
		Usage:     "Adds a birthday to the schedule.",
		Sensitive: true,
	}
}
func (c *addBirthdayCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 3 {
		return "```\nPut the date first (without quotes) and then the username, like so: !addbirthday 2 Jan Some Guy's Username```", false, nil
	}
	timestamp := bot.GetTimestamp(msg)
	arg := strings.Join(args[:2], " ") + " " + strconv.Itoa(timestamp.Year())
	user, err := bot.ParseUser(msg.Content[indices[2]:], info)
	if err != nil {
		return bot.ReturnError(err)
	}

	t, err := time.ParseInLocation("_2 Jan 2006", arg, info.GetTimezone(user))
	if err != nil {
		t, err = time.ParseInLocation("Jan _2 2006", arg, info.GetTimezone(user))
	}
	if err != nil {
		t, err = time.ParseInLocation("January _2 2006", arg, info.GetTimezone(user))
	}
	if err != nil {
		t, err = time.ParseInLocation("_2 January 2006", arg, info.GetTimezone(user))
	}
	t = t.UTC()
	if err != nil {
		return "```\nError: Could not parse time! Make sure it's in the format '2 Jan' or 'Jan 2'```", false, nil
	}
	for t.Before(timestamp.AddDate(0, 0, -1).UTC()) {
		t = t.AddDate(1, 0, 0)
	}

	if err := info.Bot.DB.AddScheduleRepeat(bot.SBatoi(info.ID), t, 8, 1, typeEventBirthday, user.String()); err != nil { // Create the normal birthday event at 12 AM on this server's timezone
		return bot.ReturnError(err)
	}
	if err := info.Bot.DB.AddScheduleRepeat(bot.SBatoi(info.ID), t.AddDate(0, 0, 1), 8, 1, typeEventUnbirthday, user.String()); err != nil { // Create the hidden "remove birthday role" event 24 hours later.
		return bot.ReturnError(err)
	}
	return info.Sanitize("```Added a birthday for "+user.Display()+"```", bot.CleanMentions|bot.CleanPings), false, nil
}
func (c *addBirthdayCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Adds member's birthday to the schedule.",
		Params: []bot.CommandUsageParam{
			{Name: "date", Desc: "The date in the form `Jan 2` or `2 Jan` - **do not** include the year!", Optional: false},
			{Name: "member", Desc: "A username or ping.", Optional: false},
		},
	}
}
