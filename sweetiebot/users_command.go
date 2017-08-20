package sweetiebot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blackhole12/discordgo"
)

type UsersModule struct {
}

func (w *UsersModule) Name() string {
	return "Users"
}

func (w *UsersModule) Commands() []Command {
	return []Command{
		&NewUsersCommand{},
		&AKACommand{},
		&BanCommand{},
		&BanNewcomersCommand{},
		&TimeCommand{},
		&SetTimeZoneCommand{},
		&UserInfoCommand{},
		&DefaultServerCommand{},
		&SilenceCommand{},
		&UnsilenceCommand{},
	}
}

func (w *UsersModule) Description() string {
	return "Contains commands for getting and setting user information."
}

type NewUsersCommand struct {
}

func (c *NewUsersCommand) Name() string {
	return "newusers"
}
func (c *NewUsersCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	maxresults := 5
	if len(args) > 0 {
		maxresults, _ = strconv.Atoi(args[0])
	}
	if maxresults < 1 {
		return "```How I return no results???```", false, nil
	}
	if maxresults > 30 {
		maxresults = 30
	}
	r := sb.db.GetNewestUsers(maxresults, SBatoi(info.ID))
	s := make([]string, 0, len(r))

	for _, v := range r {
		s = append(s, v.User.Username+"  (joined: "+ApplyTimezone(v.FirstSeen, info, msg.Author).Format(time.ANSIC)+") ["+v.User.ID+"]")
	}
	return "```\n" + strings.Join(s, "\n") + "```", true, nil
}
func (c *NewUsersCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists up to maxresults users, starting with the newest user to join the server.",
		Params: []CommandUsageParam{
			{Name: "maxresults", Desc: "Defaults to 5 results, returns a maximum of 30.", Optional: true},
		},
	}
}
func (c *NewUsersCommand) UsageShort() string {
	return "[PM Only] Gets a list of the most recent users to join the server."
}

type AKACommand struct {
}

func (c *AKACommand) Name() string {
	return "aka"
}
func (c *AKACommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
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

	r := sb.db.GetAliases(IDs[0])
	u, _, _ := sb.db.GetMember(IDs[0], SBatoi(info.ID))
	if u == nil {
		return "```Error: User does not exist!```", false, nil
	}
	nick := u.User.Username
	if len(u.User.Discriminator) > 0 {
		nick += "#" + u.User.Discriminator
	}
	if len(u.Nick) > 0 {
		nick = u.Nick + " (" + nick + ")"
	}
	return fmt.Sprintf("```All known aliases for %s [%s]\n  %s```", nick, u.User.ID, PartialSanitize(strings.Join(r, "\n  "))), false, nil
}
func (c *AKACommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists all known aliases of the user in question, up to a maximum of 10, with the names used the longest first.",
		Params: []CommandUsageParam{
			{Name: "user", Desc: "A ping of the user, or simply their name.", Optional: true},
		},
	}
}
func (c *AKACommand) UsageShort() string { return "Lists all known aliases of a user." }

func ProcessDurationAndReason(args []string, msg *discordgo.Message, indices []int, ty uint8, uID string, gID uint64) (string, string) {
	reason := ""
	if len(args) > 0 {
		if strings.ToLower(args[0]) == "for:" {
			if len(args) < 3 {
				return "", "```Error: Duration should be specified as 'for: 5 DAYS' or 'for: 72 HOURS'```"
			}
			duration, err := strconv.Atoi(args[1])
			if err != nil {
				return "", "```Error: Duration number was not an integer.```"
			}

			t := time.Now().UTC()
			switch parseRepeatInterval(args[2]) {
			case 1:
				t = t.Add(time.Duration(duration) * time.Second)
			case 2:
				t = t.Add(time.Duration(duration) * time.Minute)
			case 3:
				t = t.Add(time.Duration(duration) * time.Hour)
			case 4:
				t = t.AddDate(0, 0, duration)
			case 5:
				t = t.AddDate(0, 0, duration*7)
			case 6:
				t = t.AddDate(0, duration, 0)
			case 8:
				t = t.AddDate(duration, 0, 0)
			case 7, 255:
				return "", "```Error: unrecognized interval.```"
			}

			if !sb.db.AddSchedule(gID, t, ty, uID) {
				return "", "```Error: servers can't have more than 5000 events!```"
			}

			scheduleID := sb.db.FindEvent(uID, gID, ty)
			if scheduleID == nil {
				return "", "```Error: Could not find inserted event!```"
			}

			if len(args) > 3 {
				reason = msg.Content[indices[3]:]
			}
		} else {
			reason = msg.Content[indices[0]:]
		}
	}
	return reason, ""
}

// Ban command that tracks who banned someone, why, and optionally make the ban temporary
type BanCommand struct {
}

func (c *BanCommand) Name() string {
	return "ban"
}

func (c *BanCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	// make sure we passed a valid argument to the command
	if len(args) < 1 {
		return "```You didn't tell me who to zap with the friendship gun, silly.```", false, nil
	}
	// get the user ID and deal with Discord's alias bullshit
	arg := args[0]
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false, nil
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info, true), "\n") + "```", len(IDs) > 5, nil
	}

	gID := SBatoi(info.ID)
	u, _, _, _ := sb.db.GetUser(IDs[0])
	if u == nil {
		return "```Error: User does not exist!```", false, nil
	}
	uID := SBitoa(IDs[0])
	reason, e := ProcessDurationAndReason(args[1:], msg, indices[1:], 0, uID, gID)
	if len(e) > 0 {
		return e, false, nil
	}

	fmt.Printf("Banned %s because: %s\n", u.Username, reason)
	err := sb.dg.GuildBanCreate(info.ID, uID, 1) // Note that this will probably generate a SawBan event
	if err != nil {
		return "```Error: " + err.Error() + "```", false, nil
	}
	return "```Banned " + u.Username + " from the server. Harmony restored.```", false, nil
}
func (c *BanCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Bans the given user. Examples: `'" + info.config.Basic.CommandPrefix + "ban @CrystalFlash for: 5 MINUTES because he's a dunce` or `" + info.config.Basic.CommandPrefix + "ban \"Name With Spaces\" caught stealing cookies`",
		Params: []CommandUsageParam{
			{Name: "user", Desc: "A ping of the user, or simply their name. If the name has spaces, this argument must be put in quotes.", Optional: false},
			{Name: "for: duration", Desc: "If the keyword `for:` is used after the username, looks for a duration of the form `for: 50 MINUTES` and creates an unban event that will be fired after that much time has passed from now.", Optional: true},
			{Name: "reason", Desc: "The rest of the message is treated as a reason for the ban (currently not saved anywhere).", Optional: true},
		},
	}
}
func (c *BanCommand) UsageShort() string { return "Bans a user." }

// Bans everyone who has spoken their first message in the past N seconds, defaulting to 120
type BanNewcomersCommand struct {
}

func (c *BanNewcomersCommand) Name() string {
	return "bannewcomers"
}

func (c *BanNewcomersCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	duration := 120
	if len(args) > 0 {
		var err error
		duration, err = strconv.Atoi(args[0])
		if err != nil {
			return "```That's not a valid number of seconds!```", false, nil
		}
	}

	IDs := sb.db.GetNewcomers(duration, SBatoi(info.ID))
	if len(IDs) == 0 {
		return fmt.Sprintf("```No one has sent their first message in the past %v seconds!```", duration), false, nil
	}
	for _, id := range IDs {
		//var err error = nil
		err := sb.dg.GuildBanCreate(info.ID, SBitoa(id), 1)
		//sb.dg.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("Pretending to ban <@%v>", id))
		info.LogError("Error banning user: ", err)
	}

	return fmt.Sprintf("```Banned %v people from the server. Use discord's audit log if you need to reverse a ban.```", len(IDs)), false, nil
}
func (c *BanNewcomersCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Bans all users who have sent their first message in the past `duration` seconds.",
		Params: []CommandUsageParam{
			{Name: "duration", Desc: "The number of seconds to look back, defaults to 120 seconds (so anyone who sent their first message in the past 2 minutes would be banned).", Optional: true},
		},
	}
}
func (c *BanNewcomersCommand) UsageShort() string {
	return "Bans everyone who's recently spoken for the first time."
}

type TimeCommand struct {
}

func (c *TimeCommand) Name() string {
	return "time"
}

func (c *TimeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```This server's local time is: " + ApplyTimezone(time.Now().UTC(), info, nil).Format("Jan 2, 3:04pm```"), false, nil
	}

	arg := msg.Content[indices[0]:]
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false, nil
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info, true), "\n") + "```", len(IDs) > 5, nil
	}

	tz := sb.db.GetTimeZone(IDs[0])
	if tz == nil {
		return "```That user has not specified what their timezone is.```", false, nil
	}
	return "```That user's local time is: " + time.Now().In(tz).Format("Jan 2, 3:04pm```"), false, nil
}
func (c *TimeCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Gets the local time for the specified user, or simply gets the local time for this server.",
		Params: []CommandUsageParam{
			{Name: "user", Desc: "A ping of the user, or simply their name.", Optional: true},
		},
	}
}
func (c *TimeCommand) UsageShort() string { return "Gets a user's local time." }

type SetTimeZoneCommand struct {
}

func (c *SetTimeZoneCommand) Name() string {
	return "settimezone"
}

func (c *SetTimeZoneCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```You have to specify what your timezone is!```", false, nil
	}
	tz := []string{}
	if len(args) < 2 {
		tz = sb.db.FindTimeZone("%" + args[0] + "%")
	} else {
		offset, err := strconv.Atoi(args[1])
		if err != nil {
			return "```Could not parse offset. Note that timezones do not have spaces - use underscores (_) instead. The second argument should be your time difference from GMT in hours. For example, PDT is GMT-7, so you could search for \"America -7\".```", false, nil
		}
		tz = sb.db.FindTimeZoneOffset("%"+args[0]+"%", offset*60)
	}

	if len(tz) < 1 {
		if len(args) < 2 {
			return "```Could not find any timezone locations that match that string. Try broadening your search (for example, search for 'America' or 'Pacific').```", false, nil
		} else {
			return "```Could not find any timezone locations that match that string and offset combination. Try broadening your search, or leaving out the timezone offset parameter.```", false, nil
		}
	}
	if len(tz) > 1 {
		return "Could be any of the following timezones:\n" + strings.Join(tz, "\n"), len(tz) > 6, nil
	}

	loc, err := time.LoadLocation(tz[0])
	if err != nil {
		return "```Could not load location! Is the timezone data missing or corrupt? Error: " + err.Error() + "```", false, nil
	}

	if sb.db.SetTimeZone(SBatoi(msg.Author.ID), loc) != nil {
		return "```Error: could not set timezone!```", false, nil
	}
	return "```Set your timezone to " + loc.String() + "```", false, nil
}
func (c *SetTimeZoneCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Sets your timezone to the given location. Providing a partial timezone name, like \"America\", will return a list of all possible timezones that contain that string.",
		Params: []CommandUsageParam{
			{Name: "timezone", Desc: "A timezone location, such as `America/Los_Angeles`. Note that timezones do not have spaces.", Optional: true},
			{Name: "offset", Desc: "Your expected timezone offset in hours, used to narrow the search. For example, if you know you're in the PDT timezone, which is GMT-7, you could search for `America -7` to list all timezones in america with a standard or DST timezone offset of -7.", Optional: true},
		},
	}
}
func (c *SetTimeZoneCommand) UsageShort() string { return "Set your local timezone." }

type UserInfoCommand struct {
}

func (c *UserInfoCommand) Name() string {
	return "UserInfo"
}
func (c *UserInfoCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
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
	aliases := sb.db.GetAliases(IDs[0])
	dbuser, lastseen, tz, _ := sb.db.GetUser(IDs[0])
	dbmember, _, firstmessage := sb.db.GetMember(IDs[0], SBatoi(info.ID))

	localtime := ""
	if tz == nil {
		tz = time.FixedZone("[Not Set]", 0)
	} else {
		localtime = time.Now().In(tz).Format(time.RFC1123)
	}
	m, err := info.GetMember(SBitoa(IDs[0]))

	if err != nil {
		m = dbmember
		if m == nil {
			m = &discordgo.Member{Roles: []string{}}
		}
		u, err := sb.dg.User(SBitoa(IDs[0]))
		if err != nil {
			if dbuser == nil {
				return "```Error retrieving user information: " + err.Error() + "```", false, nil
			}
			u = dbuser
		}
		m.User = u
	}
	if dbmember != nil && len(dbmember.JoinedAt) > 0 {
		m.JoinedAt = dbmember.JoinedAt
	}
	authortz := getTimezone(info, msg.Author)
	joinedat, err := time.Parse(time.RFC3339, m.JoinedAt)
	joined := ""
	if err == nil {
		joined = TimeDiff(time.Now().UTC().Sub(joinedat.In(authortz))) + " ago (" + joinedat.In(authortz).Format(time.RFC822) + ")"
	}

	roles := make([]string, 0, len(m.Roles))
	for _, v := range m.Roles {
		role, err := sb.dg.State.Role(info.ID, v)
		if err == nil {
			roles = append(roles, role.Name)
		} else {
			roles = append(roles, "<@&"+v+">")
		}
	}
	created := snowflakeTime(IDs[0])
	fullusername := m.User.Username + "#" + m.User.Discriminator
	if m.User.Bot {
		fullusername += " [BOT]"
	}
	lastseenstring := "Never"
	if !lastseen.IsZero() {
		lastseenstring = fmt.Sprintf("%s ago (%v)", TimeDiff(time.Now().UTC().Sub(lastseen.In(authortz))), lastseen.In(authortz).Format(time.RFC822))
	}
	firstmessagestring := ""
	if firstmessage != nil {
		firstmessagestring = fmt.Sprintf("%s ago (%v)", TimeDiff(time.Now().UTC().Sub(firstmessage.In(authortz))), firstmessage.In(authortz).Format(time.RFC822))
	}
	s := fmt.Sprintf("        ID: %v\n  Username: %s\n  Nickname: %v\n   Aliases: %v\n     Roles: %v\n  Timezone: %v\nLocal Time: %v\n   Created: %s ago (%v)\n    Joined: %s\n Last Seen: %s\nFirst Msg: %s\n    Avatar: ",
		m.User.ID,
		fullusername,
		m.Nick,
		strings.Join(aliases, ", "),
		strings.Join(roles, ", "),
		tz,
		localtime,
		TimeDiff(time.Now().UTC().Sub(created)),
		created.In(authortz).Format(time.RFC822),
		joined,
		lastseenstring,
		firstmessagestring)
	return "```http\n" + PartialSanitize(s) + "```\n" + discordgo.EndpointUserAvatar(m.User.ID, m.User.Avatar), false, nil

	//s := fmt.Sprintf("**ID:** %v\n**Username:** %s\n**Nickname:** %v\n**Timezone:** %v\n**Local Time:** %v\n**Created:** %s ago (%v)\n **Joined:** %s\n**Roles:** %v\n**Last Seen:** %s ago (%v)\n**Aliases:** %v\n**Avatar:** %s", m.User.ID, fullusername, m.Nick, tz, localtime, TimeDiff(time.Now().UTC().Sub(created)), created.Format(time.RFC822), joined, strings.Join(roles, ", "), TimeDiff(time.Now().UTC().Sub(lastseen.In(authortz))), lastseen.In(authortz).Format(time.RFC822), strings.Join(aliases, ", "), discordgo.EndpointUserAvatar(m.User.ID, m.User.Avatar))
	//return SanitizeMentions(PartialSanitize(s)), false, nil
}
func (c *UserInfoCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists the ID, username, nickname, timezone, roles, avatar, join date, and other information about a given user.",
		Params: []CommandUsageParam{
			{Name: "user", Desc: "A ping of the user, or simply their name.", Optional: false},
		},
	}
}
func (c *UserInfoCommand) UsageShort() string { return "Lists information about a user." }

type DefaultServerCommand struct {
}

func (c *DefaultServerCommand) Name() string {
	return "DefaultServer"
}
func (c *DefaultServerCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	gIDs := sb.db.GetUserGuilds(SBatoi(msg.Author.ID))
	find := ""
	if len(args) > 0 {
		find = msg.Content[indices[0]:]
	}
	guilds := findServers(find, gIDs)
	names := make([]string, len(guilds), len(guilds))
	for k, v := range guilds {
		names[k] = v.Name
	}

	if len(args) < 1 {
		server := getDefaultServer(SBatoi(msg.Author.ID))
		if server != nil {
			return fmt.Sprintf("```Your default server is %s. You are on the following servers:\n%s```", server.Name, strings.Join(names, "\n")), false, nil
		}
		return fmt.Sprintf("```You have no default server. You are on the following servers:\n%s```", strings.Join(names, "\n")), false, nil
	}
	if len(guilds) > 1 {
		return "```Could be any of the following servers:\n" + strings.Join(names, "\n") + "```", false, nil
	}
	if len(guilds) < 1 {
		return "```No server matches that string (or you haven't joined that server).```", false, nil
	}

	target := SBatoi(guilds[0].ID)
	_, err := sb.dg.GuildMember(guilds[0].ID, msg.Author.ID) // Attempt to verify the user is actually in this guild.
	if err != nil {
		return fmt.Sprintf("```You aren't a member of %s (or discord blew up, in which case, try again).```", guilds[0].Name), false, nil
	}
	sb.db.SetDefaultServer(SBatoi(msg.Author.ID), target)
	return fmt.Sprintf("```Your default server was set to %s```", guilds[0].Name), false, nil
}
func (c *DefaultServerCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Sets the default server SB will run commands on that you PM to her.",
		Params: []CommandUsageParam{
			{Name: "server", Desc: "The exact name of your default server.", Optional: false},
		},
	}
}
func (c *DefaultServerCommand) UsageShort() string { return "Sets your default server." }

type SilenceCommand struct {
}

func (c *SilenceCommand) Name() string {
	return "Silence"
}
func (c *SilenceCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You must provide a user to silence.```", false, nil
	}
	index := len(args)
	for i := 1; i < len(args); i++ {
		if strings.ToLower(args[i]) == "for:" {
			index = i
			break
		}
	}
	arg := strings.Join(args[0:index], " ")
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false, nil
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info, true), "\n") + "```", len(IDs) > 5, nil
	}

	gID := SBatoi(info.ID)
	uID := SBitoa(IDs[0])
	reason, e := ProcessDurationAndReason(args[index:], msg, indices[index:], 8, uID, gID)
	if len(e) > 0 {
		return e, false, nil
	}

	code := SilenceMemberSimple(SBitoa(IDs[0]), info)
	if code < 0 {
		return "```Error occurred trying to silence " + IDsToUsernames(IDs, info, false)[0] + ".```", false, nil
	} else if code == 1 {
		var t *time.Time
		if sb.db.status.get() {
			t = sb.db.GetUnsilenceDate(gID, IDs[0])
		}
		if t == nil {
			return "```" + IDsToUsernames(IDs, info, false)[0] + " is already silenced!```", false, nil
		}
		return fmt.Sprintf("```%s is already silenced, and will be unsilenced in %s```", IDsToUsernames(IDs, info, false)[0], TimeDiff(t.Sub(time.Now().UTC()))), false, nil
	}
	if len(info.config.Spam.SilenceMessage) > 0 {
		sb.dg.ChannelMessageSend(SBitoa(info.config.Users.WelcomeChannel), "<@"+SBitoa(IDs[0])+"> "+info.config.Spam.SilenceMessage)
	}
	if len(reason) > 0 {
		reason = " because " + reason
	}
	return fmt.Sprintf("```Silenced %s%s.```", IDsToUsernames(IDs, info, false)[0], reason), false, nil
}
func (c *SilenceCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Silences the given user.",
		Params: []CommandUsageParam{
			{Name: "user", Desc: "A ping of the user, or simply their name.", Optional: false},
			{Name: "for: duration", Desc: "If the keyword `for:` is used after the username, looks for a duration of the form `for: 50 MINUTES` and creates an unsilence event that will be fired after that much time has passed from now.", Optional: true},
		},
	}
}
func (c *SilenceCommand) UsageShort() string { return "Silences a user." }

func UnsilenceMember(user uint64, info *GuildInfo) error {
	m, err := info.GetMember(SBitoa(user))
	if err == nil {
		sb.dg.State.Lock()
		RemoveSliceString(&m.Roles, SBitoa(info.config.Spam.SilentRole))
		sb.dg.State.Unlock()
	}

	return sb.dg.GuildMemberRoleRemove(info.ID, SBitoa(user), SBitoa(info.config.Spam.SilentRole))
}

type UnsilenceCommand struct {
}

func (c *UnsilenceCommand) Name() string {
	return "Unsilence"
}
func (c *UnsilenceCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You must provide a user to unsilence.```", false, nil
	}
	arg := msg.Content[indices[0]:]
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false, nil
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info, true), "\n") + "```", len(IDs) > 5, nil
	}

	err := UnsilenceMember(IDs[0], info)
	if err != nil {
		return "```Error unsilencing member: " + err.Error() + "```", false, nil
	}
	return "```Unsilenced " + IDsToUsernames(IDs, info, false)[0] + ".```", false, nil
}
func (c *UnsilenceCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Unsilences the given user.",
		Params: []CommandUsageParam{
			{Name: "user", Desc: "A ping of the user, or simply their name.", Optional: false},
		},
	}
}
func (c *UnsilenceCommand) UsageShort() string { return "Unsilences a user." }
