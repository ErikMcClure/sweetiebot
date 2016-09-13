package sweetiebot

import (
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type ScheduleModule struct {
}

func (w *ScheduleModule) Name() string {
	return "Schedule"
}

func (w *ScheduleModule) Register(info *GuildInfo) {
	info.hooks.OnTick = append(info.hooks.OnTick, w)
}

func (w *ScheduleModule) OnTick(info *GuildInfo) {
	events := sb.db.GetSchedule(SBatoi(info.Guild.ID))
	channel := SBitoa(info.config.ModChannel)
	if len(info.config.Module_channels[strings.ToLower(w.Name())]) > 0 {
		for k := range info.config.Module_channels[strings.ToLower(w.Name())] {
			channel = k
			break
		}
	} else if len(info.config.Module_channels["bored"]) > 0 {
		for k := range info.config.Module_channels["bored"] {
			channel = k
			break
		}
	} else if len(info.config.FreeChannels) > 0 {
		for k := range info.config.FreeChannels {
			channel = k
			break
		}
	}

	if len(channel) == 0 {
		//info.log.Error(SBitoa(info.config.LogChannel), "No channel available to process events on. No events processed. If you want to suppress this message, you should either disable the schedule module, or use '!setconfig module_channels schedule #channel'.")
		return
	}

	for _, v := range events {
		switch v.Type {
		case 0:
			sb.dg.GuildBanDelete(info.Guild.ID, v.Data)
			info.SendMessage(SBitoa(info.config.ModChannel), "Unbanned <@"+v.Data+">")
		case 1:
			m, err := sb.dg.GuildMember(info.Guild.ID, v.Data)
			if err != nil {
				info.log.LogError("Couldn't get <@"+v.Data+"> member data! ", err)
			} else if info.config.BirthdayRole == 0 {
				info.log.Log("No birthday role set!")
			} else {
				m.Roles = append(m.Roles, SBitoa(info.config.BirthdayRole))
				sb.dg.GuildMemberEdit(info.Guild.ID, v.Data, m.Roles)
			}
			info.SendMessage(channel, "Happy Birthday <@"+v.Data+">!")
		case 2:
			info.SendMessage(channel, v.Data)
		case 5:
			fallthrough
		case 3:
			info.SendMessage(channel, v.Data+" is starting now!")
		case 4:
			m, err := sb.dg.GuildMember(info.Guild.ID, v.Data)
			if err != nil {
				info.log.LogError("Couldn't get <@"+v.Data+"> member data! ", err)
			} else {
				RemoveSliceString(&m.Roles, SBitoa(info.config.BirthdayRole))
				sb.dg.GuildMemberEdit(info.Guild.ID, v.Data, m.Roles)
			}
		case 6:
			dat := strings.SplitN(v.Data, "|", 2)
			ch, err := sb.dg.UserChannelCreate(dat[0])
			info.log.LogError("Error opening private channel: ", err)
			if err == nil {
				info.SendMessage(ch.ID, dat[1])
			}
		case 7:
			dat := strings.SplitN(v.Data, "|", 2)
			info.SendMessage(channel, getGroupPings(dat[0], info)+" "+dat[1])
		}

		sb.db.RemoveSchedule(v.ID)
	}
}

type ScheduleCommand struct {
}

func (c *ScheduleCommand) Name() string {
	return "Schedule"
}
func (c *ScheduleCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	maxresults := 5
	var ty uint8
	ty = 255
	if len(args) > 1 {
		ty = getScheduleType(args[0])
		if ty == 255 {
			return "```Unknown schedule type.```", false
		}
		maxresults, _ = strconv.Atoi(args[1])
	} else if len(args) > 0 {
		var err error
		maxresults, err = strconv.Atoi(args[0])
		if err != nil {
			maxresults = 5
			ty = getScheduleType(args[0])
			if ty == 255 {
				return "```Unknown schedule type.```", false
			}
		}
	}
	if maxresults > 20 {
		maxresults = 20
	}
	if maxresults < 1 {
		maxresults = 1
	}
	if !info.UserHasRole(msg.Author.ID, SBitoa(info.config.AlertRole)) && (ty == 0 || ty == 4) {
		return "```You aren't allowed to view those events.```", false
	}
	var events []ScheduleEvent
	if ty == 255 {
		events = sb.db.GetEvents(SBatoi(info.Guild.ID), maxresults)
	} else if ty == 6 {
		events = sb.db.GetReminders(SBatoi(info.Guild.ID), msg.Author.ID, maxresults)
	} else {
		events = sb.db.GetEventsByType(SBatoi(info.Guild.ID), ty, maxresults)
	}
	if len(events) == 0 {
		return "There are no upcoming events.", false
	}
	lines := []string{"Upcoming Events:"}
	for _, v := range events {
		s := "#" + SBitoa(v.ID)
		if v.Date.Year() == time.Now().UTC().Year() {
			s += ApplyTimezone(v.Date, info, msg.Author).Format(" **Jan 2 3:04pm**")
		} else {
			s += ApplyTimezone(v.Date, info, msg.Author).Format(" **Jan 2 2006 3:04pm**")
		}
		data := v.Data
		switch v.Type {
		case 1:
			s += " [BIRTHDAY] "
			data = "<@" + data + ">"
		case 2:
			s += " [MESSAGE] "
		case 3:
			s += " [EPISODE] "
			if len(info.config.SpoilChannels) > 0 && !FindIntSlice(SBatoi(msg.ChannelID), info.config.SpoilChannels) {
				data = "(title removed)"
			}
		case 5:
			s += " [EVENT]"
		case 6:
			s += " [REMINDER] "
			data = strings.SplitN(data, "|", 2)[1]
		case 7:
			datas := strings.SplitN(data, "|", 2)
			s += " [GROUP:" + datas[0] + "] "
			data = datas[1]
		default:
			s += " [UNKNOWN] "
		}
		lines = append(lines, s+ReplaceAllMentions(data))
	}

	return strings.Join(lines, "\n"), len(lines) > 6
}
func (c *ScheduleCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[bans/birthdays/messages/episodes/events/reminders] [maxresults]", "Lists up to maxresults (default: 5) upcoming events from the schedule. If the first argument is specified, lists only events of that type. Some event types can only be viewed by moderators. Max results: 20")
}
func (c *ScheduleCommand) UsageShort() string { return "Gets a list of upcoming scheduled events." }

func getScheduleType(s string) uint8 {
	switch strings.ToLower(s) {
	case "bans":
		fallthrough
	case "ban":
		return 0
	case "birthdays":
		fallthrough
	case "birthday":
		return 1
	case "messages":
		fallthrough
	case "message":
		return 2
	case "episodes":
		fallthrough
	case "episode":
		return 3
	case "events":
		fallthrough
	case "event":
		return 5
	case "reminders":
		fallthrough
	case "reminder":
		return 6
	case "groups":
		fallthrough
	case "group":
		return 7
	}
	return 255
}

type NextCommand struct {
}

func (c *NextCommand) Name() string {
	return "Next"
}
func (c *NextCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You must specify an event type.```", false
	}
	ty := getScheduleType(args[0])
	if ty == 255 {
		return "```Error: Invalid type specified.```", false
	}

	event := sb.db.GetNextEvent(SBatoi(info.Guild.ID), ty)
	if event.Type > 0 && event.Date.Before(time.Now().UTC()) {
		return "```Sweetie will announce this event in just a moment!```", false
	}
	diff := TimeDiff(event.Date.Sub(time.Now().UTC()))
	switch event.Type {
	case 1:
		return ReplaceAllMentions("```It'll be <@" + event.Data + ">'s birthday in " + diff + "```"), false
	case 2:
		return "```Sweetie is scheduled to send a message in " + diff + "```", false
	case 3:
		if len(info.config.SpoilChannels) > 0 && !FindIntSlice(SBatoi(msg.ChannelID), info.config.SpoilChannels) {
			return "```The next episode airs in " + diff + "```", false
		}
		return "```" + event.Data + " airs in " + diff + "```", false
	case 5:
		return "```" + event.Data + " starts in " + diff + "```", false
	case 7:
		return "```Sweetie is scheduled to send a message to " + strings.SplitN(event.Data, "|", 2)[0] + " in " + diff + "```", false
	default:
		return "```There are no upcoming events of that type (or you aren't allowed to view them).```", false
	}
}
func (c *NextCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[ban/episode/birthday/message/event/reminder/group]", "Gets the time until the next event of the given type.")
}
func (c *NextCommand) UsageShort() string { return "Gets time until next event." }

type AddEventCommand struct {
}

func parseRepeatInterval(s string) uint8 {
	switch strings.ToLower(s) {
	case "seconds":
		fallthrough
	case "second":
		return 1
	case "minutes":
		fallthrough
	case "minute":
		return 2
	case "hours":
		fallthrough
	case "hour":
		return 3
	case "days":
		fallthrough
	case "day":
		return 4
	case "weeks":
		fallthrough
	case "week":
		return 5
	case "months":
		fallthrough
	case "month":
		return 6
	case "quarters":
		fallthrough
	case "quarter":
		return 7
	case "years":
		fallthrough
	case "year":
		return 8
	}
	return 255
}

func (c *AddEventCommand) Name() string {
	return "AddEvent"
}
func (c *AddEventCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 2 {
		return "```At least a type and a date must be specified!```", false
	}
	ty := getScheduleType(args[0])
	if ty == 255 {
		return "```Error: Invalid type specified.```", false
	}
	data := ""
	if ty == 7 {
		data = strings.ToLower(args[1])
		_, ok := info.config.Groups[data]
		if !ok {
			return "Error: That group doesn't exist.", false
		}
		data += "|"
		args = append(args[:1], args[2:]...)
	}
	if ty == 6 {
		data = StripPing(args[1])
		_, err := sb.dg.GuildMember(info.Guild.ID, data)
		if err != nil {
			return "Error: user ID doesn't exist.", false
		}
		data += "|"
		args = append(args[:1], args[2:]...)
	}
	t, err := parseCommonTime(args[1], info, msg.Author)
	if err != nil {
		return "```Error: Could not parse time! Make sure it's in the format \"2 Jan 06 3:04pm -0700\" (time and timezone are optional)```", false
	}
	t = t.UTC()
	if t.Before(time.Now().UTC()) {
		return "```Error: Cannot specify an event in the past!```", false
	}

	if len(args) > 2 && repeatregex.MatchString(strings.ToLower(args[2])) {
		repeats := strings.Split(args[2], " ")
		repeat, err := strconv.Atoi(repeats[1])
		if err != nil {
			return "```Error: Repeat number was not an integer.```", false
		}

		repeatinterval := parseRepeatInterval(repeats[2])
		if repeatinterval == 255 {
			return "```Error: unrecognized interval.```", false
		}

		if len(args) > 3 {
			data += strings.Join(args[3:], " ")
		}
		if !sb.db.AddScheduleRepeat(SBatoi(info.Guild.ID), t, repeatinterval, repeat, ty, data) {
			return "```Error: servers can't have more than 5000 events!```", false
		}
	} else {
		if len(args) > 2 {
			data += strings.Join(args[2:], " ")
		}

		if !sb.db.AddSchedule(SBatoi(info.Guild.ID), t, ty, data) {
			return "```Error: servers can't have more than 5000 events!```", false
		}
	}

	return "```Added event to schedule.```", false
}
func (c *AddEventCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[type] [group/user] [date] [REPEAT N SECONDS/MINUTES/HOURS/DAYS/WEEKS/MONTHS/YEARS] [data]", "Adds an arbitrary event to the schedule table. Only use the [group/user] parameter if type is 'group' or 'reminder'. The REPEAT parameter is optional, but MUST be surrounded by quotes, just like the time parameter. For example: '!addevent message \"12 Jun 16\" \"REPEAT 1 YEAR\" happy birthday!', or '!addevent episode \"9 Dec 15\" Slice of Life'. Available types of events: ban, birthday, message, episode, event, reminder, group. You shouldn't add birthday or reminder events manually - use !addbirthday or !remindme.")
}
func (c *AddEventCommand) UsageShort() string { return "Adds an event to the schedule." }

func userOwnsEvent(e *ScheduleEvent, u *discordgo.User) bool {
	if e.Type == 6 {
		dat := strings.SplitN(e.Data, "|", 2)
		if dat[0] == u.ID {
			return true
		}
	}
	return false
}

type RemoveEventCommand struct {
}

func (c *RemoveEventCommand) Name() string {
	return "RemoveEvent"
}
func (c *RemoveEventCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You must specify an event ID.```", false
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return "```Could not parse event ID. Make sure you only specify the number itself.```", false
	}

	e := sb.db.GetEvent(id)
	if e == nil {
		return "```Error: Event does not exist.```", false
	}
	if !info.UserHasRole(msg.Author.ID, SBitoa(info.config.AlertRole)) && !userOwnsEvent(e, msg.Author) {
		return "```Error: You do not have permission to delete that event.```", false
	}

	sb.db.RemoveSchedule(id)
	return "```Removed Event #" + SBitoa(id) + " from schedule.```", false
}
func (c *RemoveEventCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[ID]", "Removes an event with the given ID from the schedule.")
}
func (c *RemoveEventCommand) UsageShort() string { return "Removes an event." }

type RemindMeCommand struct {
}

func (c *RemindMeCommand) Name() string {
	return "RemindMe"
}
func (c *RemindMeCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 3 {
		return "```You must start your message with 'in' or 'on', followed by a time or duration, followed by a message.```", false
	}

	var t time.Time
	arg := ""
	switch strings.ToLower(args[0]) {
	default:
		fallthrough
	case "in":
		t = time.Now().UTC()
		d, err := strconv.Atoi(args[1])
		if err != nil {
			return "```Duration is not numeric! Make sure it's in the format 'in 99 days', and DON'T put quotes around it.```", false
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
			return "```Unknown duration type! Acceptable types are seconds, minutes, hours, days, weeks, months, and years.```", false
		}
		arg = strings.Join(args[3:], " ")
	case "on":
		var err error
		t, err = parseCommonTime(strings.ToLower(args[1]), info, msg.Author)
		if err != nil {
			return "```Could not parse time! Make sure its in the format \"2 Jan 06 3:04pm -0700\" (time and timezone are optional). Make sure you surround it with quotes!```", false
		}
		t = t.UTC()
		arg = strings.Join(args[2:], " ")
	}

	if len(arg) == 0 {
		return "```What am I reminding you about? I can't send you a blank message!```", false
	}
	if !sb.db.AddSchedule(SBatoi(info.Guild.ID), t, 6, msg.Author.ID+"|"+arg) {
		return "```Error: servers can't have more than 5000 events!```", false
	}
	return "Reminder set for " + TimeDiff(t.Sub(time.Now().UTC())) + " from now.", false
}
func (c *RemindMeCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[in N seconds/minutes/hours/etc.] OR [on \"2 Jan 06 3:04pm -0700\"] [message]", "Tells sweetiebot to remind you about something in the future.")
}
func (c *RemindMeCommand) UsageShort() string {
	return "Tells sweetiebot to remind you about something."
}

type AddBirthdayCommand struct {
}

func (c *AddBirthdayCommand) Name() string {
	return "AddBirthday"
}
func (c *AddBirthdayCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 2 {
		return "```You must first ping the member and then provide the date!```", false
	}
	ping := StripPing(args[0])
	arg := strings.Join(args[1:], " ") + " 16"
	t, err := time.ParseInLocation("_2 Jan 06", arg, getTimezone(info, nil)) // Deliberately do not include the user timezone here. We want this to operate on the server timezone.
	if err != nil {
		t, err = time.ParseInLocation("Jan _2 06", arg, getTimezone(info, nil))
	}
	t = t.UTC()
	if err != nil {
		return "```Error: Could not parse time! Make sure it's in the format \"2 Jan\"```", false
	}
	for t.Before(time.Now().UTC()) {
		t = t.AddDate(1, 0, 0)
	}
	if len(ping) == 0 {
		return "```Error: Invalid ping for member! Make sure you actually ping them via @MemberName, don't just type the name in.```", false
	}

	sb.db.AddScheduleRepeat(SBatoi(info.Guild.ID), t, 8, 1, 1, ping)                        // Create the normal birthday event at 12 AM on this server's timezone
	if !sb.db.AddScheduleRepeat(SBatoi(info.Guild.ID), t.AddDate(0, 0, 1), 8, 1, 4, ping) { // Create the hidden "remove birthday role" event 24 hours later.
		return "```Error: servers can't have more than 5000 events!```", false
	}
	return ReplaceAllMentions("```Added a birthday for <@" + ping + ">```"), false
}
func (c *AddBirthdayCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[member] [date]", "Adds member's birthday to the schedule - Be sure to ping the member, and DO NOT include the year in the date!")
}
func (c *AddBirthdayCommand) UsageShort() string { return "Adds a birthday to the schedule." }
