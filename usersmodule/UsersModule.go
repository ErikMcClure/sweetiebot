package usersmodule

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	bot "../sweetiebot"
	"4d63.com/tz"
	"github.com/blackhole12/discordgo"
)

// UsersModule contains commands for getting and setting user information
type UsersModule struct {
}

// New instance of UsersModule
func New() *UsersModule {
	return &UsersModule{}
}

// Name of the module
func (w *UsersModule) Name() string {
	return "Users"
}

// Commands in the module
func (w *UsersModule) Commands() []bot.Command {
	return []bot.Command{
		&newUsersCommand{},
		&akaCommand{},
		&banCommand{},
		&banNewcomersCommand{},
		&timeCommand{},
		&setTimeZoneCommand{},
		&userInfoCommand{},
		&defaultServerCommand{},
		&silenceCommand{},
		&unsilenceCommand{},
		&assignRoleCommand{},
	}
}

// Description of the module
func (w *UsersModule) Description() string {
	return "Contains commands for getting and setting user information, or manipulating user roles."
}

// OnGuildMemberAdd discord hook
func (w *UsersModule) OnGuildMemberAdd(info *bot.GuildInfo, m *discordgo.Member, t time.Time) {
	if info.Config.Users.NotifyChannel != bot.ChannelEmpty {
		created := "(Created " + bot.TimeDiff(t.Sub(bot.SnowflakeTime(bot.SBatoi(m.User.ID)))) + " ago) joined"
		if info.Config.Spam.RaidSilence >= 2 || (info.Config.Spam.RaidSilence >= 1 && ((info.LastRaid + info.Config.Spam.RaidTime*2) > t.Unix())) {
			created += " and was silenced"
		}
		info.SendMessage(info.Config.Users.NotifyChannel, "<@"+m.User.ID+"> "+created+".")
	}
	if info.Config.Users.NewUserRole != bot.RoleEmpty && info.Config.Users.NewUserDuration > 0 {
		assignRoleMember(info, bot.DiscordUser(m.User.ID), info.Config.Users.NewUserRole)

		gID := bot.SBatoi(info.ID)
		if err := info.Bot.DB.AddSchedule(gID, time.Now().Add(time.Second*time.Duration(info.Config.Users.NewUserDuration)), 9, m.User.ID+"|"+info.Config.Users.NewUserRole.String()); err != nil {
			info.LogError("Failed to add NewUserRole to schedule: ", err)
		}
	}
}

// OnGuildMemberRemove discord hook
func (w *UsersModule) OnGuildMemberRemove(info *bot.GuildInfo, m *discordgo.Member, t time.Time) {
	if info.Config.Users.TrackUserLeft && info.Config.Users.NotifyChannel != bot.ChannelEmpty {
		text := m.User.Username + "#" + m.User.Discriminator + " left."
		info.SendMessage(info.Config.Users.NotifyChannel, text)
	}
}

// assignRoleMember adds a role to a member that already exists
func assignRoleMember(info *bot.GuildInfo, userID bot.DiscordUser, roleID bot.DiscordRole) (int8, error) {
	m, merr := info.Bot.DG.GetMember(userID, info.ID)
	if merr == nil { // Manually set our internal state to say this role is set to prevent race conditions
		if bot.MemberHasRole(m, roleID) {
			return 1, info.ResolveRoleAddError(info.Bot.DG.GuildMemberRoleAdd(info.ID, userID.String(), roleID.String()))
		}
		nroles := make([]string, len(m.Roles)) // We set this to a new slice so we can atomically replace it on x86 architectures, avoiding a lock
		copy(nroles, m.Roles)
		m.Roles = append(nroles, roleID.String())
	}

	return 0, info.ResolveRoleAddError(info.Bot.DG.GuildMemberRoleAdd(info.ID, userID.String(), roleID.String()))
}

type newUsersCommand struct {
}

func (c *newUsersCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "NewUsers",
		Usage: "Gets a list of the most recent users to join the server.",
	}
}

func (c *newUsersCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	maxresults := 5
	if len(args) > 0 {
		maxresults, _ = strconv.Atoi(args[0])
	}
	if maxresults < 1 {
		return "```\nHow I return no results???```", false, nil
	}
	if maxresults > 40 {
		maxresults = 40
	}
	r := info.Bot.DB.GetNewestUsers(maxresults, bot.SBatoi(info.ID))
	s := make([]string, 0, len(r))

	for _, v := range r {
		s = append(s, v.User.Username+"  (joined: "+info.ApplyTimezone(v.FirstSeen, bot.DiscordUser(msg.Author.ID)).Format(time.ANSIC)+") ["+v.User.ID+"]")
	}
	return "```\n" + info.Sanitize(strings.Join(s, "\n"), bot.CleanCodeBlock) + "```", len(s) > bot.MaxPublicLines, nil
}
func (c *newUsersCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Lists up to maxresults users, starting with the newest user to join the server.",
		Params: []bot.CommandUsageParam{
			{Name: "maxresults", Desc: "Defaults to 5 results, returns a maximum of 40.", Optional: true},
		},
	}
}

type akaCommand struct {
}

func (c *akaCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "aka",
		Usage: "Lists all known aliases of a user.",
	}
}

func (c *akaCommand) Name() string {
	return "aka"
}
func (c *akaCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou must provide a user to search for.```", false, nil
	}
	user, err := bot.ParseUser(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(err)
	}

	r := info.Bot.DB.GetAliases(user.Convert())
	u, err := info.Bot.DG.GetMember(user, info.ID)
	if err != nil {
		return bot.ReturnError(err)
	}
	nick := u.User.Username
	if len(u.User.Discriminator) > 0 {
		nick += "#" + u.User.Discriminator
	}
	if len(u.Nick) > 0 {
		nick = u.Nick + " (" + nick + ")"
	}
	return fmt.Sprintf("```All known aliases for %s [%s]\n  %s```", nick, u.User.ID, info.Sanitize(strings.Join(r, "\n  "), bot.CleanCodeBlock)), false, nil
}
func (c *akaCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Lists all known aliases of the user in question, up to a maximum of 10, with the names used the longest first.",
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: "A ping of the user, or simply their name.", Optional: true},
		},
	}
}

func processDurationAndReason(args []string, msg *discordgo.Message, indices []int, ty uint8, data string, gID uint64, db *bot.BotDB) (string, error) {
	if ty == 6 {
		return "", errors.New("Illegal event type.")
	}
	reason := ""
	if len(args) > 0 {
		if strings.ToLower(args[0]) == "for:" {
			if len(args) < 3 {
				return "", errors.New("Duration should be specified as 'for: 5 DAYS' or 'for: 72 HOURS'")
			}
			duration, err := strconv.Atoi(args[1])
			if err != nil {
				return "", errors.New("Duration number was not an integer.")
			}

			t := bot.GetTimestamp(msg)
			switch bot.ParseRepeatInterval(args[2]) {
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
				return "", errors.New("unrecognized interval.")
			}

			if err := db.AddSchedule(gID, t, ty, data); err != nil {
				return "", err
			}

			scheduleID := db.FindEvent(data, gID, ty)
			if scheduleID == nil {
				return "", errors.New("Could not find inserted event!")
			}

			if len(args) > 3 {
				reason = msg.Content[indices[3]:]
			}
		} else {
			reason = msg.Content[indices[0]:]
		}
	}
	return reason, nil
}

// Ban command that tracks who banned someone, why, and optionally make the ban temporary
type banCommand struct {
}

func (c *banCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "ban",
		Usage:     "Bans a user.",
		Sensitive: true,
	}
}

func (c *banCommand) Name() string {
	return "ban"
}

func (c *banCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou must specify a username (in quotes, if it has spaces).```", false, nil
	}
	name, err := bot.ParseUser(args[0], info)
	if err != nil {
		return bot.ReturnError(err)
	}

	if info.UserIsMod(name) || info.UserIsAdmin(name) {
		return fmt.Sprintf(bot.StringMap[bot.STRING_USERS_BAN_MOD_ERROR], info.GetUserName(name)), false, nil
	}

	reason, err := processDurationAndReason(args[1:], msg, indices[1:], 0, name.String(), bot.SBatoi(info.ID), info.Bot.DB)
	if err != nil {
		return bot.ReturnError(err)
	}
	reason = fmt.Sprintf("Banned by %s#%s for %s", msg.Author.Username, msg.Author.Discriminator, reason)
	username := info.GetUserName(name)

	err = info.Bot.DG.GuildBanCreateWithReason(info.ID, name.String(), reason, 1) // Note that this will probably generate a SawBan event
	if err != nil {
		return bot.ReturnError(err)
	}
	return "```\nBanned " + username + " from the server.```", false, nil
}
func (c *banCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Bans the given user. Examples: `'" + info.Config.Basic.CommandPrefix + "ban @CrystalFlash for: 5 MINUTES because he's a dunce` or `" + info.Config.Basic.CommandPrefix + "ban \"Name With Spaces\" caught stealing cookies`",
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: "A ping of the user, or simply their name. If the name has spaces, this argument must be put in quotes.", Optional: false},
			{Name: "for: duration", Desc: "If the keyword `for:` is used after the username, looks for a duration of the form `for: 50 MINUTES` and creates an unban event that will be fired after that much time has passed from now.", Optional: true},
			{Name: "reason", Desc: "The rest of the message is treated as a reason for the ban.", Optional: true},
		},
	}
}

// Bans everyone who has spoken their first message in the past N seconds, defaulting to 120
type banNewcomersCommand struct {
}

func (c *banNewcomersCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "BanNewcomers",
		Usage:     "Bans everyone who has recently spoken for the first time.",
		Sensitive: true,
	}
}

func (c *banNewcomersCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	duration := 120
	if len(args) > 0 {
		var err error
		duration, err = strconv.Atoi(args[0])
		if err != nil {
			return "```\nThat's not a valid number of seconds!```", false, nil
		}
	}

	IDs := info.Bot.DB.GetNewcomers(duration, bot.SBatoi(info.ID))
	if len(IDs) == 0 {
		return fmt.Sprintf("```No one has sent their first message in the past %v seconds!```", duration), false, nil
	}
	reason := fmt.Sprintf("Banned by %s#%s via the !bannewcomers command", msg.Author.Username, msg.Author.Discriminator)
	for _, id := range IDs {
		err := info.Bot.DG.GuildBanCreateWithReason(info.ID, bot.SBitoa(id), reason, 1)
		info.LogError("Error banning user: ", err)
	}

	return fmt.Sprintf("```Banned %v people from the server. Use discord's audit log if you need to reverse a ban.```", len(IDs)), false, nil
}
func (c *banNewcomersCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Bans all users who have sent their first message in the past `duration` seconds.",
		Params: []bot.CommandUsageParam{
			{Name: "duration", Desc: "The number of seconds to look back, defaults to 120 seconds (so anyone who sent their first message in the past 2 minutes would be banned).", Optional: true},
		},
	}
}

type timeCommand struct {
}

func (c *timeCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "time",
		Usage: "Gets a user's local time.",
	}
}

func (c *timeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nThis server's local time is: " + info.ApplyTimezone(bot.GetTimestamp(msg), bot.UserEmpty).Format("Jan 2, 3:04pm```"), false, nil
	}

	user, err := bot.ParseUser(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(err)
	}
	tz := info.Bot.DB.GetTimeZone(user.Convert())
	if tz == nil {
		return "```\nThat user has not specified what their timezone is.```", false, nil
	}
	return "```\nThat user's local time is: " + bot.GetTimestamp(msg).In(tz).Format("Jan 2, 3:04pm```"), false, nil
}
func (c *timeCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Gets the local time for the specified user, or simply gets the local time for this server.",
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: "A ping of the user, or simply their name.", Optional: true},
		},
	}
}

type setTimeZoneCommand struct {
}

func (c *setTimeZoneCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "SetTimezone",
		Usage: "Set your local timezone.",
	}
}
func (c *setTimeZoneCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou have to specify what your timezone is!```", false, nil
	}
	zone := []string{}
	if len(args) < 2 {
		zone = info.Bot.DB.FindTimeZone("%" + args[0] + "%")
	} else {
		offset, err := strconv.Atoi(args[1])
		if err != nil {
			return "```\nCould not parse offset. Note that timezones do not have spaces - use underscores (_) instead. The second argument should be your time difference from GMT in hours. For example, PDT is GMT-7, so you could search for \"America -7\".```", false, nil
		}
		zone = info.Bot.DB.FindTimeZoneOffset("%"+args[0]+"%", offset*60)
	}
	if strings.Contains(strings.ToLower(args[0]), "gmt") || (len(zone) == 1 && strings.Contains(strings.ToLower(zone[0]), "gmt")) {
		return "```\nStop. Just stop. That's not going to work for daylight savings. You have to provide a timezone LOCATION, like 'America/Los_Angeles'. If you aren't sure what timezone location to use, check what your operating system is set to.```", false, nil
	}

	if len(zone) < 1 {
		if len(args) < 2 {
			return "```\nCould not find any timezone locations that match that string. Try broadening your search (for example, search for 'America' or 'Pacific').```", false, nil
		}
		return "```\nCould not find any timezone locations that match that string and offset combination. Try broadening your search, or leaving out the timezone offset parameter.```", false, nil
	}
	if len(zone) > 1 {
		return "Could be any of the following timezones:\n" + strings.Join(zone, "\n"), len(zone) > 6, nil
	}

	loc, err := tz.LoadLocation(zone[0])
	if err != nil {
		return "```\nCould not load location! Is the timezone data missing or corrupt? Error: " + err.Error() + "```", false, nil
	}

	if info.Bot.DB.SetTimeZone(bot.SBatoi(msg.Author.ID), loc) != nil {
		return "```\nError: could not set timezone!```", false, nil
	}
	return "```\nYour timezone was set to " + loc.String() + "```", false, nil
}
func (c *setTimeZoneCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Sets your timezone to the given location. Providing a partial timezone name, like \"America\", will return a list of all possible timezones that contain that string.",
		Params: []bot.CommandUsageParam{
			{Name: "timezone", Desc: "A timezone location, such as `America/Los_Angeles`. Note that timezones do not have spaces.", Optional: true},
			{Name: "offset", Desc: "Your expected timezone offset in hours, used to narrow the search. For example, if you know you're in the PDT timezone, which is GMT-7, you could search for `America -7` to list all timezones in america with a standard or DST timezone offset of -7.", Optional: true},
		},
	}
}

type userInfoCommand struct {
}

func (c *userInfoCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "UserInfo",
		Usage: "Lists information about a user.",
	}
}

func (c *userInfoCommand) Name() string {
	return "UserInfo"
}
func (c *userInfoCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou must provide a user to search for.```", false, nil
	}

	user, err := bot.ParseUser(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(err)
	}
	id := user.Convert()
	aliases := info.Bot.DB.GetAliases(id)
	dbuser, lastseen, tz, _ := info.Bot.DB.GetUser(id)
	dbmember, _, firstmessage := info.Bot.DB.GetMember(id, bot.SBatoi(info.ID))
	timestamp := bot.GetTimestamp(msg)

	localtime := ""
	if tz == nil {
		tz = time.FixedZone("[Not Set]", 0)
	} else {
		localtime = timestamp.In(tz).Format(time.RFC1123)
	}
	m, err := info.Bot.DG.GetMember(user, info.ID)

	if err != nil {
		m = dbmember
		if m == nil {
			m = &discordgo.Member{Roles: []string{}}
		}
		u, err := info.Bot.DG.User(user.String())
		if err != nil {
			if dbuser == nil {
				return "```\nError retrieving user information: " + err.Error() + "```", false, nil
			}
			u = dbuser
		}
		m.User = u
	}
	if dbmember != nil && len(dbmember.JoinedAt) > 0 {
		m.JoinedAt = dbmember.JoinedAt
	}
	authortz := info.GetTimezone(bot.DiscordUser(msg.Author.ID))
	joinedat, err := m.JoinedAt.Parse()
	joined := ""
	if err == nil {
		joined = bot.TimeDiff(timestamp.Sub(joinedat.In(authortz))) + " ago (" + joinedat.In(authortz).Format(time.RFC822) + ")"
	}

	roles := make([]string, 0, len(m.Roles))
	for _, v := range m.Roles {
		roles = append(roles, bot.DiscordRole(v).Show(info))
	}
	created := bot.SnowflakeTime(id)
	fullusername := m.User.Username + "#" + m.User.Discriminator
	if m.User.Bot {
		fullusername += " [BOT]"
	}
	lastseenstring := "Never"
	if !lastseen.IsZero() {
		lastseenstring = fmt.Sprintf("%s ago (%v)", bot.TimeDiff(timestamp.Sub(lastseen.In(authortz))), lastseen.In(authortz).Format(time.RFC822))
	}
	firstmessagestring := ""
	if firstmessage != nil {
		firstmessagestring = fmt.Sprintf("%s ago (%v)", bot.TimeDiff(timestamp.Sub(firstmessage.In(authortz))), firstmessage.In(authortz).Format(time.RFC822))
	}
	s := fmt.Sprintf("        ID: %v\n  Username: %s\n  Nickname: %v\n   Aliases: %v\n     Roles: %v\n  Timezone: %v\nLocal Time: %v\n   Created: %s ago (%v)\n    Joined: %s\n Last Seen: %s\n First Msg: %s\n    Avatar: ",
		m.User.ID,
		fullusername,
		m.Nick,
		strings.Join(aliases, ", "),
		strings.Join(roles, ", "),
		tz,
		localtime,
		bot.TimeDiff(timestamp.Sub(created)),
		created.In(authortz).Format(time.RFC822),
		joined,
		lastseenstring,
		firstmessagestring)
	return "```http\n" + info.Sanitize(s, bot.CleanCodeBlock) + "```\n" + discordgo.EndpointUserAvatar(m.User.ID, m.User.Avatar), false, nil
}
func (c *userInfoCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Lists the ID, username, nickname, timezone, roles, avatar, join date, and other information about a given user.",
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: "A ping of the user, or simply their name.", Optional: false},
		},
	}
}

type defaultServerCommand struct {
}

func (c *defaultServerCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:              "DefaultServer",
		Usage:             "Sets your default server.",
		ServerIndependent: true,
	}
}

func (c *defaultServerCommand) Name() string {
	return "DefaultServer"
}
func (c *defaultServerCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	gIDs := info.Bot.DB.GetUserGuilds(bot.SBatoi(msg.Author.ID))
	find := ""
	if len(args) > 0 {
		find = msg.Content[indices[0]:]
	}
	guilds := info.Bot.FindServers(find, gIDs)
	names := make([]string, len(guilds), len(guilds))
	for k, v := range guilds {
		names[k] = v.Name
	}

	if len(args) < 1 {
		server := info.Bot.GetDefaultServer(bot.SBatoi(msg.Author.ID))
		if server != nil {
			return fmt.Sprintf("```Your default server is %s. You are on the following servers:\n%s```", server.Name, strings.Join(names, "\n")), false, nil
		}
		return fmt.Sprintf("```You have no default server. You are on the following servers:\n%s```", strings.Join(names, "\n")), false, nil
	}
	if len(guilds) > 1 {
		return "```\nCould be any of the following servers:\n" + strings.Join(names, "\n") + "```", false, nil
	}
	if len(guilds) < 1 {
		return "```\nNo server matches that string (or you haven't joined that server).```", false, nil
	}

	if !guilds[0].Config.SetupDone {
		return fmt.Sprintf("```%s hasn't been set up yet! Someone needs to run !setup on that server first. Go here for instructions: https://sweetiebot.io```", guilds[0].Name), false, nil
	}

	target := bot.SBatoi(guilds[0].ID)
	_, err := info.Bot.DG.GuildMember(guilds[0].ID, msg.Author.ID) // Attempt to verify the user is actually in this guild.
	if err != nil {
		return fmt.Sprintf("```You aren't a member of %s (or discord blew up, in which case, try again).```", guilds[0].Name), false, nil
	}
	info.Bot.DB.SetDefaultServer(bot.SBatoi(msg.Author.ID), target)
	return fmt.Sprintf("```Your default server was set to %s```", guilds[0].Name), false, nil
}
func (c *defaultServerCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Sets the default server SB will run commands on that you PM to her.",
		Params: []bot.CommandUsageParam{
			{Name: "server", Desc: "The exact name of your default server.", Optional: false},
		},
	}
}

type silenceCommand struct {
}

func (c *silenceCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "Silence",
		Usage:     bot.StringMap[bot.STRING_USERS_SILENCE_USAGE],
		Sensitive: true,
	}
}

func (c *silenceCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return bot.StringMap[bot.STRING_USERS_SILENCE_ARG_ERROR], false, nil
	}
	index := len(args)
	for i := 1; i < len(args); i++ {
		if strings.ToLower(args[i]) == "for:" {
			index = i
			break
		}
	}

	user, err := bot.ParseUser(strings.Join(args[0:index], " "), info)
	if err != nil {
		return bot.ReturnError(err)
	}

	if info.UserIsMod(user) || info.UserIsAdmin(user) {
		return fmt.Sprintf(bot.StringMap[bot.STRING_USERS_SILENCE_MOD_ERROR], info.GetUserName(user)), false, nil
	}

	gID := bot.SBatoi(info.ID)
	reason, err := processDurationAndReason(args[index:], msg, indices[index:], 8, user.String(), gID, info.Bot.DB)
	if err != nil {
		return bot.ReturnError(err)
	}

	code, err := assignRoleMember(info, user, info.Config.Basic.SilenceRole)
	if code < 0 || err != nil {
		return fmt.Sprintf(bot.StringMap[bot.STRING_USERS_SILENCE_ERROR], info.GetUserName(user), info.ResolveRoleAddError(err).Error()), false, nil
	} else if code == 1 {
		var t *time.Time
		if info.Bot.DB.Status.Get() {
			t = info.Bot.DB.GetScheduleDate(gID, 8, user.String())
		}
		if t == nil {
			return fmt.Sprintf(bot.StringMap[bot.STRING_USERS_SILENCE_ALREADY_SILENCED], info.GetUserName(user)), false, nil
		}
		return fmt.Sprintf(bot.StringMap[bot.STRING_USERS_SILENCE_WILL_BE_UNSILENCED], info.GetUserName(user), bot.TimeDiff(t.Sub(bot.GetTimestamp(msg)))), false, nil
	}
	if len(info.Config.Users.SilenceMessage) > 0 {
		info.SendMessage(info.Config.Users.JailChannel, user.Display()+info.Config.Users.SilenceMessage)
	}
	if len(reason) > 0 {
		reason = fmt.Sprintf(bot.StringMap[bot.STRING_USERS_SILENCE_REASON], reason)
	}
	return fmt.Sprintf(bot.StringMap[bot.STRING_USERS_SILENCE], info.GetUserName(user), reason), false, nil
}
func (c *silenceCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: bot.StringMap[bot.STRING_USERS_SILENCE_DESCRIPTION],
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: bot.StringMap[bot.STRING_USERS_SILENCE_USER], Optional: false},
			{Name: "for: duration", Desc: bot.StringMap[bot.STRING_USERS_SILENCE_DURATION], Optional: true},
		},
	}
}

type unsilenceCommand struct {
}

func (c *unsilenceCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "Unsilence",
		Usage:     bot.StringMap[bot.STRING_USERS_UNSILENCE_USAGE],
		Sensitive: true,
	}
}

func (c *unsilenceCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return bot.StringMap[bot.STRING_USERS_UNSILENCE_ARG_ERROR], false, nil
	}
	user, err := bot.ParseUser(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(info.ResolveRoleAddError(err))
	}
	if info.UserIsMod(user) || info.UserIsAdmin(user) {
		return fmt.Sprintf(bot.StringMap[bot.STRING_USERS_UNSILENCE_MOD_ERROR], info.GetUserName(user)), false, nil
	}

	err = info.Bot.DG.RemoveRole(info.ID, user, info.Config.Basic.SilenceRole)
	if err != nil {
		return fmt.Sprintf(bot.StringMap[bot.STRING_USERS_UNSILENCE_ERROR], err.Error()), false, nil
	}
	return fmt.Sprintf(bot.StringMap[bot.STRING_USERS_UNSILENCE], info.GetUserName(user)), false, nil
}
func (c *unsilenceCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: bot.StringMap[bot.STRING_USERS_UNSILENCE_DESCRIPTION],
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: bot.StringMap[bot.STRING_USERS_UNSILENCE_USER], Optional: false},
		},
	}
}

type assignRoleCommand struct {
}

func (c *assignRoleCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "AssignRole",
		Usage:     "Assigns an arbitrary role to a user for an optional amount of time.",
		Sensitive: true,
	}
}

func (c *assignRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 2 {
		return "```\nYou must provide a role to assign and a user to assign it to.```", false, nil
	}
	index := len(args)
	for i := 1; i < len(args); i++ {
		if strings.ToLower(args[i]) == "for:" {
			index = i
			break
		}
	}

	g, err := info.GetGuild()
	role, err := bot.ParseRole(args[0], g)
	if err != nil {
		return bot.ReturnError(err)
	}

	user, err := bot.ParseUser(strings.Join(args[1:index], " "), info)
	if err != nil {
		return bot.ReturnError(err)
	}

	gID := bot.SBatoi(info.ID)
	reason, err := processDurationAndReason(args[index:], msg, indices[index:], 9, user.String()+"|"+role.String(), gID, info.Bot.DB)
	if err != nil {
		return bot.ReturnError(err)
	}

	code, err := assignRoleMember(info, user, role)
	if code < 0 || err != nil {
		return fmt.Sprintf("```\nError occurred trying to assign %s to  %s: %s```", role.Show(info), info.GetUserName(user), info.ResolveRoleAddError(err).Error()), false, nil
	} else if code == 1 {
		var t *time.Time
		if info.Bot.DB.Status.Get() {
			t = info.Bot.DB.GetScheduleDate(gID, 9, user.String()+"|"+role.String())
		}
		if t == nil {
			return "```\n" + info.GetUserName(user) + " already has that role!```", false, nil
		}
		return fmt.Sprintf("```\n%s already has that role, which will be removed in %s```", info.GetUserName(user), bot.TimeDiff(t.Sub(bot.GetTimestamp(msg)))), false, nil
	}
	if len(reason) > 0 {
		reason = " because " + reason
	}
	return fmt.Sprintf("```\nAssigned the %s role to %s%s.```", role.Show(info), info.GetUserName(user), reason), false, nil
}
func (c *assignRoleCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Assigns the role to the given user, and optionally adds an event to remove it in the future.",
		Params: []bot.CommandUsageParam{
			{Name: "role", Desc: "The role to add, either as a ping or as the name, but must be in quotes if it has spaces.", Optional: false},
			{Name: "user", Desc: "A ping of the user, or simply their name.", Optional: false},
			{Name: "for: duration", Desc: "If the keyword `for:` is used after the username, looks for a duration of the form `for: 50 MINUTES` and creates an event that will remove the role after that much time has passed from now.", Optional: true},
		},
	}
}
