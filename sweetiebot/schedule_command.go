package sweetiebot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blackhole12/discordgo"
)

// ScheduleModule manages the scheduling system
type ScheduleModule struct {
}

// Name of the module
func (w *ScheduleModule) Name() string {
	return "Scheduler"
}

// Commands in the module
func (w *ScheduleModule) Commands() []Command {
	return []Command{
		&scheduleCommand{},
		&nextCommand{},
		&addEventCommand{},
		&removeEventCommand{},
		&remindMeCommand{},
		&addBirthdayCommand{},
	}
}

// Description of the module
func (w *ScheduleModule) Description() string {
	return "Manages the scheduling system, and periodically checks for events that need to be processed."
}

// OnTick discord hook
func (w *ScheduleModule) OnTick(info *GuildInfo) {
	if !sb.db.CheckStatus() {
		return
	}
	events := sb.db.GetSchedule(SBatoi(info.ID))
	channel := SBitoa(info.config.Basic.ModChannel)
	if len(info.config.Modules.Channels[strings.ToLower(w.Name())]) > 0 {
		for k := range info.config.Modules.Channels[strings.ToLower(w.Name())] {
			channel = k
			break
		}
	} else if len(info.config.Modules.Channels["bored"]) > 0 {
		for k := range info.config.Modules.Channels["bored"] {
			channel = k
			break
		}
	} else if len(info.config.Basic.FreeChannels) > 0 {
		for k := range info.config.Basic.FreeChannels {
			channel = k
			break
		}
	}

	if len(channel) == 0 {
		return
	}

	for _, v := range events {
		switch v.Type {
		case 0:
			err := sb.dg.GuildBanDelete(info.ID, v.Data)
			if err != nil {
				info.SendMessage(SBitoa(info.config.Basic.ModChannel), "Error unbanning <@"+v.Data+">: "+err.Error())
			} else {
				info.SendMessage(SBitoa(info.config.Basic.ModChannel), "Unbanned <@"+v.Data+">")
			}
		case 1:
			if info.config.Schedule.BirthdayRole == 0 {
				info.Log("No birthday role set!")
			} else {
				err := sb.dg.GuildMemberRoleAdd(info.ID, v.Data, SBitoa(info.config.Schedule.BirthdayRole))
				info.LogError("Failed to set birthday role: ", err)
			}
			info.SendMessage(channel, "Happy Birthday <@"+v.Data+">!")
		case 2:
			info.SendMessage(channel, v.Data)
		case 5, 3:
			info.SendMessage(channel, v.Data+" is starting now!")
		case 4:
			if info.config.Schedule.BirthdayRole == 0 {
				info.Log("No birthday role set!")
			} else {
				err := sb.dg.GuildMemberRoleRemove(info.ID, v.Data, SBitoa(info.config.Schedule.BirthdayRole))
				info.LogError("Failed to remove birthday role: ", err)
			}
		case 6:
			dat := strings.SplitN(v.Data, "|", 2)
			ch, err := sb.dg.UserChannelCreate(dat[0])
			info.LogError("Error opening private channel: ", err)
			if err == nil {
				info.SendMessage(ch.ID, dat[1])
			}
		case 7:
			dat := strings.SplitN(v.Data, "|", 2)
			info.SendMessage(channel, dat[0]+" "+dat[1])
		case 8:
			err := UnsilenceMember(SBatoi(v.Data), info)
			if err != nil {
				info.SendMessage(SBitoa(info.config.Basic.ModChannel), "Error unsilencing <@"+v.Data+">: "+err.Error())
			} else {
				info.SendMessage(SBitoa(info.config.Basic.ModChannel), "Unsilenced <@"+v.Data+">")
			}
		}

		sb.db.RemoveSchedule(v.ID)
	}
}

type scheduleCommand struct {
}

func (c *scheduleCommand) Name() string {
	return "Schedule"
}
func (c *scheduleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	maxresults := 5
	var ty uint8
	ty = 255
	if len(args) > 1 {
		ty = getScheduleType(args[0])
		if ty == 255 {
			return "```Unknown schedule type.```", false, nil
		}
		maxresults, _ = strconv.Atoi(args[1])
	} else if len(args) > 0 {
		var err error
		maxresults, err = strconv.Atoi(args[0])
		if err != nil {
			maxresults = 5
			ty = getScheduleType(args[0])
			if ty == 255 {
				return "```Unknown schedule type.```", false, nil
			}
		}
	}
	if maxresults > 20 {
		maxresults = 20
	}
	if maxresults < 1 {
		maxresults = 1
	}
	if !info.UserHasRole(msg.Author.ID, SBitoa(info.config.Basic.AlertRole)) && (ty == 0 || ty == 4 || ty == 8) {
		return "```You aren't allowed to view those events.```", false, nil
	}
	var events []ScheduleEvent
	if ty == 255 {
		events = sb.db.GetEvents(SBatoi(info.ID), maxresults)
	} else if ty == 6 {
		events = sb.db.GetReminders(SBatoi(info.ID), msg.Author.ID, maxresults)
	} else {
		events = sb.db.GetEventsByType(SBatoi(info.ID), ty, maxresults)
	}
	if len(events) == 0 {
		return "There are no upcoming events.", false, nil
	}
	lines := make([]string, len(events)+1, len(events)+1)
	lines[0] = "Upcoming Events:"
	for k, v := range events {
		t := ""
		if v.Date.Year() == time.Now().UTC().Year() {
			t = ApplyTimezone(v.Date, info, msg.Author).Format("Jan 2 3:04pm")
		} else {
			t = ApplyTimezone(v.Date, info, msg.Author).Format("Jan 2 2006 3:04pm")
		}
		data := v.Data
		mt := "UNKNOWN"
		switch v.Type {
		case 1:
			mt = "BIRTHDAY"
			data = "<@" + data + ">"
		case 2:
			mt = "MESSAGE"
		case 3:
			mt = "EPISODE"
			if len(info.config.Spoiler.Channels) > 0 && !FindIntSlice(SBatoi(msg.ChannelID), info.config.Spoiler.Channels) {
				data = "(title removed)"
			}
		case 5:
			mt = "EVENT"
		case 6:
			mt = "REMINDER"
			data = strings.SplitN(data, "|", 2)[1]
		case 7:
			datas := strings.SplitN(data, "|", 2)
			mt = "ROLE:" + ReplaceAllRolePings(datas[0], info)
			data = datas[1]
		}
		lines[k+1] = fmt.Sprintf("#%v **%s** [%s] %s", SBitoa(v.ID), t, mt, ReplaceAllMentions(data))
	}

	return strings.Join(lines, "\n"), len(lines) > 6, nil
}
func (c *scheduleCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists up to `maxresults` upcoming events from the schedule. If the first argument is specified, lists only events of that type. Some event types can only be viewed by moderators. Max results: 20",
		Params: []CommandUsageParam{
			{Name: "type", Desc: "Can be one of: bans, birthdays, messages, episodes, events, roles, reminders.", Optional: true},
			{Name: "maxresults", Desc: "Defaults to 5.", Optional: true},
		},
	}
}
func (c *scheduleCommand) UsageShort() string { return "Gets a list of upcoming scheduled events." }

func getScheduleType(s string) uint8 {
	switch strings.ToLower(s) {
	case "bans", "ban":
		return 0
	case "birthdays", "birthday":
		return 1
	case "messages", "message":
		return 2
	case "episodes", "episode":
		return 3
	case "events", "event":
		return 5
	case "reminders", "reminder":
		return 6
	case "roles", "role":
		return 7
	case "silences", "silence":
		return 8
	}
	return 255
}

type nextCommand struct {
}

func (c *nextCommand) Name() string {
	return "Next"
}
func (c *nextCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```You must specify an event type.```", false, nil
	}
	ty := getScheduleType(args[0])
	if ty == 255 {
		return "```Error: Invalid type specified.```", false, nil
	}

	event := sb.db.GetNextEvent(SBatoi(info.ID), ty)
	if event.Type > 0 && event.Date.Before(time.Now().UTC()) {
		return "```Sweetie will announce this event in just a moment!```", false, nil
	}
	diff := TimeDiff(event.Date.Sub(time.Now().UTC()))
	switch event.Type {
	case 1:
		return ReplaceAllMentions("```It'll be <@" + event.Data + ">'s birthday in " + diff + "```"), false, nil
	case 2:
		return "```Sweetie is scheduled to send a message in " + diff + "```", false, nil
	case 3:
		if len(info.config.Spoiler.Channels) > 0 && !FindIntSlice(SBatoi(msg.ChannelID), info.config.Spoiler.Channels) {
			return "```The next episode airs in " + diff + "```", false, nil
		}
		return "```" + event.Data + " airs in " + diff + "```", false, nil
	case 5:
		return "```" + event.Data + " starts in " + diff + "```", false, nil
	case 7:
		return "```Sweetie is scheduled to send a message to " + ReplaceAllRolePings(strings.SplitN(event.Data, "|", 2)[0], info) + " in " + diff + "```", false, nil
	default:
		return "```There are no upcoming events of that type (or you aren't allowed to view them).```", false, nil
	}
}
func (c *nextCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Gets the time until the next event of the given type.",
		Params: []CommandUsageParam{
			{Name: "type", Desc: "Can be one of: bans, birthdays, messages, episodes, events, reminders.", Optional: true},
		},
	}
}
func (c *nextCommand) UsageShort() string { return "Gets time until next event." }

type addEventCommand struct {
}

func parseRepeatInterval(s string) uint8 {
	switch strings.ToLower(s) {
	case "seconds", "second":
		return 1
	case "minutes", "minute":
		return 2
	case "hours", "hour":
		return 3
	case "days", "day":
		return 4
	case "weeks", "week":
		return 5
	case "months", "month":
		return 6
	case "quarters", "quarter":
		return 7
	case "years", "year":
		return 8
	}
	return 255
}

func (c *addEventCommand) Name() string {
	return "AddEvent"
}
func (c *addEventCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 2 {
		return "```At least a type and a date must be specified!```", false, nil
	}
	ty := getScheduleType(args[0])
	if ty == 255 {
		return "```Error: Invalid type specified.```", false, nil
	}
	data := ""
	if ty == 7 {
		data = strings.ToLower(args[1])
		data += "|"
		args = append(args[:1], args[2:]...)
		indices = append(indices[:1], indices[2:]...)
	}
	if ty == 6 {
		data = StripPing(args[1])
		_, err := sb.dg.GuildMember(info.ID, data)
		if err != nil {
			return "Error: user ID doesn't exist.", false, nil
		}
		data += "|"
		args = append(args[:1], args[2:]...)
		indices = append(indices[:1], indices[2:]...)
	}
	t, err := parseCommonTime(args[1], info, msg.Author)
	if err != nil {
		return "```Error: Could not parse time! Make sure it's in the format \"2 January 2006 3:04pm -0700\" (or something similar, year, time and timezone are optional)```", false, nil
	}
	t = t.UTC()
	if t.Before(time.Now().UTC()) {
		return "```Error: Cannot specify an event in the past!```", false, nil
	}

	if len(args) > 2 && repeatregex.MatchString(strings.ToLower(args[2])) {
		repeats := strings.Split(args[2], " ")
		repeat, err := strconv.Atoi(repeats[1])
		if err != nil {
			return "```Error: Repeat number was not an integer.```", false, nil
		}

		repeatinterval := parseRepeatInterval(repeats[2])
		if repeatinterval == 255 {
			return "```Error: unrecognized interval.```", false, nil
		}

		if len(args) > 3 {
			data += msg.Content[indices[3]:]
		}
		if !sb.db.AddScheduleRepeat(SBatoi(info.ID), t, repeatinterval, repeat, ty, data) {
			return "```Error: servers can't have more than 5000 events!```", false, nil
		}
	} else {
		if len(args) > 2 {
			data += msg.Content[indices[2]:]
		}

		if !sb.db.AddSchedule(SBatoi(info.ID), t, ty, data) {
			return "```Error: servers can't have more than 5000 events!```", false, nil
		}
	}

	return "```Added event to schedule.```", false, nil
}
func (c *addEventCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Adds an arbitrary event to the schedule table. For example: `" + info.config.Basic.CommandPrefix + "addevent message \"12 Jun 16\" \"REPEAT 1 YEAR\" happy birthday!`, or `" + info.config.Basic.CommandPrefix + "addevent episode \"9 Dec 15\" Slice of Life`. ",
		Params: []CommandUsageParam{
			{Name: "type", Desc: "Can be one of: ban, birthday, message, episode, event, reminder, role. You shouldn't add birthday or reminder events manually, though.", Optional: false},
			{Name: "role/user", Desc: "The target role or user to ping. Only include this if the type is role or reminder. If the type is \"role\", it must be an actual ping for the role, not just the name.", Optional: true},
			{Name: "date", Desc: "A date in the format 12 Jun 16 2:10pm, in quotes. The time, year, and timezone are all optional.", Optional: false},
			{Name: "REPEAT N INTERVAL", Desc: "INTERVAL can be one of SECONDS/MINUTES/HOURS/DAYS/WEEKS/MONTHS/YEARS. This parameter MUST be surrounded by quotes!", Optional: true},
		},
	}
}
func (c *addEventCommand) UsageShort() string { return "Adds an event to the schedule." }

func userOwnsEvent(e *ScheduleEvent, u *discordgo.User) bool {
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

func (c *removeEventCommand) Name() string {
	return "RemoveEvent"
}
func (c *removeEventCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```You must specify an event ID.```", false, nil
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return "```Could not parse event ID. Make sure you only specify the number itself.```", false, nil
	}

	e := sb.db.GetEvent(id)
	if e == nil {
		return "```Error: Event does not exist.```", false, nil
	}
	_, isOwner := sb.Owners[SBatoi(msg.Author.ID)]
	if !isOwner && !info.UserHasRole(msg.Author.ID, SBitoa(info.config.Basic.AlertRole)) && !userOwnsEvent(e, msg.Author) {
		return "```Error: You do not have permission to delete that event.```", false, nil
	}

	sb.db.RemoveSchedule(id)
	return "```Removed Event #" + SBitoa(id) + " from schedule.```", false, nil
}
func (c *removeEventCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Removes an event with the given ID from the schedule. ",
		Params: []CommandUsageParam{
			{Name: "ID", Desc: "The event ID as gotten from a `" + info.config.Basic.CommandPrefix + "schedule` command.", Optional: false},
		},
	}
}
func (c *removeEventCommand) UsageShort() string { return "Removes an event." }

type remindMeCommand struct {
}

func (c *remindMeCommand) Name() string {
	return "RemindMe"
}
func (c *remindMeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 3 {
		return "```You must start your message with 'in' or 'on', followed by a date (in quotes!) or duration, followed by a message.```", false, nil
	}

	var t time.Time
	arg := ""
	switch strings.ToLower(args[0]) {
	default: // You're supposed to use "in" here, but if we don't know what to do we just do this by default anyway.
		t = time.Now().UTC()
		d, err := strconv.Atoi(args[1])
		if err != nil {
			return "```Duration is not numeric! Make sure it's in the format 'in 99 days', and DON'T put quotes around it.```", false, nil
		}
		if d <= 0 {
			return "```That was " + TimeDiff(time.Now().UTC().Sub(t)) + " ago, you idiot! Do you think I have a time machine or something?```", false, nil
		}
		switch parseRepeatInterval(args[2]) {
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
			return "```Unknown duration type! Acceptable types are seconds, minutes, hours, days, weeks, months, and years.```", false, nil
		}
		if len(indices) < 4 {
			return "```You have to tell me what to say!```", false, nil
		}
		arg = msg.Content[indices[3]:]
	case "on":
		var err error
		t, err = parseCommonTime(strings.ToLower(args[1]), info, msg.Author)
		if err != nil {
			return "```Could not parse time! Make sure its in the format \"2 January 2006 3:04pm -0700\" (or something similar, time, year, and timezone are optional). Make sure you surround it with quotes!```", false, nil
		}
		t = t.UTC()
		if t.Before(time.Now().UTC()) {
			return "```That was " + TimeDiff(time.Now().UTC().Sub(t)) + " ago, dumbass! You have to give me a time that's in the FUTURE!```", false, nil
		}
		arg = msg.Content[indices[2]:]
	}

	if len(arg) == 0 {
		return "```What am I reminding you about? I can't send you a blank message!```", false, nil
	}
	if !sb.db.AddSchedule(SBatoi(info.ID), t, 6, msg.Author.ID+"|"+arg) {
		return "```Error: servers can't have more than 5000 events!```", false, nil
	}
	return "Reminder set for " + TimeDiff(t.Sub(time.Now().UTC())) + " from now.", false, nil
}
func (c *remindMeCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Tells sweetiebot to remind you about something in the future. ",
		Params: []CommandUsageParam{
			{Name: "in N seconds/minutes/hours/etc.", Desc: "represents a time `N` units from the current time. The available units are: seconds, minutes, hours, days, weeks, months, years.", Optional: true},
			{Name: "on \"2 January 2006 3:04pm -0700\"", Desc: "represents an absolute date and time, which must be in quotes. You must choose the `in` syntax OR the `on` syntax to specify your time, not both.", Optional: true},
			{Name: "message", Desc: "An arbitrary string that will be sent to you at the appropriate time.", Optional: false},
		},
	}
}
func (c *remindMeCommand) UsageShort() string {
	return "Tells sweetiebot to remind you about something."
}

type addBirthdayCommand struct {
}

func (c *addBirthdayCommand) Name() string {
	return "AddBirthday"
}
func (c *addBirthdayCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 2 {
		return "```You must first ping the member and then provide the date!```", false, nil
	}
	ping := StripPing(args[0])
	arg := strings.Join(args[1:], " ") + " " + strconv.Itoa(time.Now().Year())
	t, err := time.ParseInLocation("_2 Jan 2006", arg, getTimezone(info, nil)) // Deliberately do not include the user timezone here. We want this to operate on the server timezone.
	if err != nil {
		t, err = time.ParseInLocation("Jan _2 2006", arg, getTimezone(info, nil))
	}
	if err != nil {
		t, err = time.ParseInLocation("January _2 2006", arg, getTimezone(info, nil))
	}
	if err != nil {
		t, err = time.ParseInLocation("_2 January 2006", arg, getTimezone(info, nil))
	}
	t = t.UTC()
	if err != nil {
		return "```Error: Could not parse time! Make sure it's in the format \"2 Jan\"```", false, nil
	}
	for t.Before(time.Now().AddDate(0, 0, -1).UTC()) {
		t = t.AddDate(1, 0, 0)
	}
	_, err = strconv.ParseUint(ping, 10, 64)
	if len(ping) == 0 || err != nil {
		return "```Error: Invalid ping for member! Make sure you actually ping them via @MemberName, don't just type the name in.```", false, nil
	}

	sb.db.AddScheduleRepeat(SBatoi(info.ID), t, 8, 1, 1, ping)                        // Create the normal birthday event at 12 AM on this server's timezone
	if !sb.db.AddScheduleRepeat(SBatoi(info.ID), t.AddDate(0, 0, 1), 8, 1, 4, ping) { // Create the hidden "remove birthday role" event 24 hours later.
		return "```Error: servers can't have more than 5000 events!```", false, nil
	}
	return ReplaceAllMentions("```Added a birthday for <@" + ping + ">```"), false, nil
}
func (c *addBirthdayCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Adds member's birthday to the schedule.",
		Params: []CommandUsageParam{
			{Name: "member", Desc: "A user ping in the form @User.", Optional: false},
			{Name: "date", Desc: "The date in the form `Jan 2` or `2 Jan` - **do not** include the year!", Optional: false},
		},
	}
}
func (c *addBirthdayCommand) UsageShort() string { return "Adds a birthday to the schedule." }
