package sweetiebot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type NewUsersCommand struct {
}

func (c *NewUsersCommand) Name() string {
	return "newusers"
}
func (c *NewUsersCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	maxresults := 5
	if len(args) > 0 {
		maxresults, _ = strconv.Atoi(args[0])
	}
	if maxresults < 1 {
		return "```How I return no results???```", false
	}
	if maxresults > 30 {
		maxresults = 30
	}
	r := sb.db.GetNewestUsers(maxresults, SBatoi(info.Guild.ID))
	s := make([]string, 0, len(r))

	for _, v := range r {
		s = append(s, v.User.Username+"  (joined: "+ApplyTimezone(v.FirstSeen, info, msg.Author).Format(time.ANSIC)+") ["+v.User.ID+"]")
	}
	return "```" + strings.Join(s, "\n") + "```", true
}
func (c *NewUsersCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[maxresults]", "Lists up to maxresults users, starting with the newest user to join the server. Defaults to 5 results, returns a maximum of 30.")
}
func (c *NewUsersCommand) UsageShort() string {
	return "[PM Only] Gets a list of the most recent users to join the server."
}

type AKACommand struct {
}

func (c *AKACommand) Name() string {
	return "aka"
}
func (c *AKACommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You must provide a user to search for.```", false
	}
	arg := strings.Join(args, " ")
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	r := sb.db.GetAliases(IDs[0])
	u, _ := sb.db.GetMember(IDs[0], SBatoi(info.Guild.ID))
	if u == nil {
		return "```Error: User does not exist!```", false
	}
	nick := u.User.Username
	if len(u.Nick) > 0 {
		nick = u.Nick
	}
	return "```All known aliases for " + nick + " [" + u.User.ID + "]\n  " + strings.Join(r, "\n  ") + "```", false
}
func (c *AKACommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[@user]", "Lists all known aliases of the user in question, up to a maximum of 10, with the names used the longest first.")
}
func (c *AKACommand) UsageShort() string { return "Lists all known aliases of a user." }

func ProcessDurationAndReason(args []string, ty uint8, uID string, gID uint64) (string, string) {
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
			case 7:
				fallthrough
			case 255:
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
				reason = strings.Join(args[3:], " ")
			}
		} else {
			reason = strings.Join(args, " ")
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

func (c *BanCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	// make sure we passed a valid argument to the command
	if len(args) < 1 {
		return "```You didn't tell me who to zap with the friendship gun, silly.```", false
	}
	// get the user ID and deal with Discord's alias bullshit
	arg := args[0]
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	gID := SBatoi(info.Guild.ID)
	u, _, _, _ := sb.db.GetUser(IDs[0])
	if u == nil {
		return "```Error: User does not exist!```", false
	}
	uID := SBitoa(IDs[0])
	reason, e := ProcessDurationAndReason(args[1:], 0, uID, gID)
	if len(e) > 0 {
		return e, false
	}

	fmt.Printf("Banned %s because: %s\n", u.Username, reason)
	err := sb.dg.GuildBanCreate(info.Guild.ID, uID, 1) // Note that this will probably generate a SawBan event
	if err != nil {
		return "```Error: " + err.Error() + "```", false
	}
	return "```Banned " + u.Username + " from the server. Harmony restored.```", false
}
func (c *BanCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[@user] [for: duration] [reason]", "Bans the given user. The username must be a single argument, so if it has spaces, it must be put in quotes, like \"User Name\". If the keyword 'for:' is used after the username, looks for a duration of the form 'for: 50 MINUTES' and creates an unban event that will be fired after that much time has passed from now. The rest of the message is treated as a reason for the ban. Examples: '!ban @CrystalFlash for: 5 MINUTES because he's a dunce' or '!ban \"Name With Spaces\" caught stealing cookies'.")
}
func (c *BanCommand) UsageShort() string { return "Bans a user." }

type TimeCommand struct {
}

func (c *TimeCommand) Name() string {
	return "time"
}

func (c *TimeCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```This server's local time is: " + ApplyTimezone(time.Now().UTC(), info, nil).Format("Jan 2, 3:04pm```"), false
	}

	arg := strings.Join(args, " ")
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	tz := sb.db.GetTimeZone(IDs[0])
	if tz == nil {
		return "```That user has not specified what their timezone is.```", false
	}
	return "```That user's local time is: " + time.Now().In(tz).Format("Jan 2, 3:04pm```"), false
}
func (c *TimeCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[user]", "Gets the local time for the specified user, or simply gets the local time for this server.")
}
func (c *TimeCommand) UsageShort() string { return "Gets a user's local time." }

type SetTimeZoneCommand struct {
}

func (c *SetTimeZoneCommand) Name() string {
	return "settimezone"
}

func (c *SetTimeZoneCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You have to specify what your timezone is!```", false
	}
	tz := []string{}
	if len(args) < 2 {
		tz = sb.db.FindTimeZone("%" + args[0] + "%")
	} else {
		offset, err := strconv.Atoi(args[1])
		if err != nil {
			return "```Could not parse offset. Note that timezones do not have spaces - use underscores (_) instead. The second argument should be your time difference from GMT in hours. For example, PDT is GMT-7, so you could search for \"America -7\".```", false
		}
		tz = sb.db.FindTimeZoneOffset("%"+args[0]+"%", offset*60)
	}

	if len(tz) < 1 {
		if len(args) < 2 {
			return "```Could not find any timezone locations that match that string. Try broadening your search (for example, search for 'America' or 'Pacific').```", false
		} else {
			return "```Could not find any timezone locations that match that string and offset combination. Try broadening your search, or leaving out the timezone offset parameter.```", false
		}
	}
	if len(tz) > 1 {
		return "Could be any of the following timezones:\n" + strings.Join(tz, "\n"), len(tz) > 6
	}

	loc, err := time.LoadLocation(tz[0])
	if err != nil {
		return "```Could not load location! Is the timezone data missing or corrupt? Error: " + err.Error() + "```", false
	}

	if sb.db.SetTimeZone(SBatoi(msg.Author.ID), loc) != nil {
		return "```Error: could not set timezone!```", false
	}
	return "```Set your timezone to " + loc.String() + "```", false
}
func (c *SetTimeZoneCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[timezone] [offset]", "Sets your timezone to the given location, such as \"America/Los_Angeles\". Providing a partial timezone name, like \"America\", will return a list of all possible timezones that contain that string. Timezones do not have spaces. You can also specify your expected timezone offset in hours to narrow the search. For example, if you know you're in the PDT timezone, which is GMT-7, you could search for \"America -7\" to list all timezones in america with a standard or DST timezone offset of -7.")
}
func (c *SetTimeZoneCommand) UsageShort() string { return "Set your local timezone." }

type UserInfoCommand struct {
}

func (c *UserInfoCommand) Name() string {
	return "UserInfo"
}
func (c *UserInfoCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You must provide a user to search for.```", false
	}
	arg := strings.Join(args, " ")
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	aliases := sb.db.GetAliases(IDs[0])
	dbuser, lastseen, tz, _ := sb.db.GetUser(IDs[0])
	localtime := ""
	if tz == nil {
		tz = time.FixedZone("[Not Set]", 0)
	} else {
		localtime = time.Now().In(tz).Format(time.RFC1123)
	}
	m, err := sb.dg.GuildMember(info.Guild.ID, SBitoa(IDs[0]))
	if err != nil {
		m = &discordgo.Member{Roles: []string{}}
		u, err := sb.dg.User(SBitoa(IDs[0]))
		if err != nil {
			if dbuser == nil {
				return "```Error retrieving user information: " + err.Error() + "```", false
			}
			u = dbuser
		}
		m.User = u
	}
	authortz := getTimezone(info, msg.Author)
	joinedat, err := time.Parse(time.RFC3339Nano, m.JoinedAt)
	joined := ""
	if err == nil {
		joined = joinedat.In(authortz).Format(time.RFC1123)
	}
	guildroles, err := sb.dg.GuildRoles(info.Guild.ID)
	if err != nil {
		guildroles = info.Guild.Roles
	}

	roles := make([]string, 0, len(m.Roles))
	for _, v := range m.Roles {
		if err == nil {
			for _, role := range guildroles {
				if role.ID == v {
					roles = append(roles, role.Name)
					break
				}
			}
		} else {
			roles = append(roles, "<@&"+v+">")
		}
	}

	return ExtraSanitize(fmt.Sprintf("**ID:** %v\n**Username:** %v#%v\n**Nickname:** %v\n**Timezone:** %v\n**Local Time:** %v\n**Joined:** %v\n**Roles:** %v\n**Bot:** %v\n**Last Seen:** %v\n**Aliases:** %v\n**Avatar:** ", m.User.ID, m.User.Username, m.User.Discriminator, m.Nick, tz, localtime, joined, strings.Join(roles, ", "), m.User.Bot, lastseen.In(authortz).Format(time.RFC1123), strings.Join(aliases, ", "))) + discordgo.EndpointUserAvatar(m.User.ID, m.User.Avatar), false
}
func (c *UserInfoCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[@user]", "Lists the ID, username, nickname, timezone, roles, avatar, join date, and other information about a given user.")
}
func (c *UserInfoCommand) UsageShort() string { return "Lists information about a user." }

type DefaultServerCommand struct {
}

func (c *DefaultServerCommand) Name() string {
	return "DefaultServer"
}
func (c *DefaultServerCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	gIDs := sb.db.GetUserGuilds(SBatoi(msg.Author.ID))
	guilds := findServers(strings.Join(args, " "), gIDs)
	if len(guilds) > 1 {
		names := make([]string, len(guilds), len(guilds))
		for k, v := range guilds {
			names[k] = v.Guild.Name
		}

		if len(args) < 1 {
			server := getDefaultServer(SBatoi(msg.Author.ID))
			if server != nil {
				return fmt.Sprintf("```Your default server is %s. You are on the following servers:\n%s```", server.Guild.Name, strings.Join(names, "\n")), false
			}
			return fmt.Sprintf("```You have no default server. You are on the following servers:\n%s```", strings.Join(names, "\n")), false
		}
		return "```Could be any of the following servers:\n" + strings.Join(names, "\n") + "```", false
	}
	if len(guilds) < 1 {
		return "```No server matches that string!```", false
	}

	sb.db.SetDefaultServer(SBatoi(msg.Author.ID), SBatoi(guilds[0].Guild.ID))
	return fmt.Sprintf("```Your default server was set to %s```", guilds[0].Guild.Name), false
}
func (c *DefaultServerCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[server]", "Sets the default server SB will run commands on that you PM to her.")
}
func (c *DefaultServerCommand) UsageShort() string { return "Sets your default server." }

type SilenceCommand struct {
}

func (c *SilenceCommand) Name() string {
	return "Silence"
}
func (c *SilenceCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You must provide a user to silence.```", false
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
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	gID := SBatoi(info.Guild.ID)
	uID := SBitoa(IDs[0])
	reason, e := ProcessDurationAndReason(args[index:], 8, uID, gID)
	if len(e) > 0 {
		return e, false
	}

	if SilenceMember(SBitoa(IDs[0]), info) < 0 {
		return "```Error occured trying to silence " + IDsToUsernames(IDs, info)[0] + ".```", false
	}
	if len(info.config.SilenceMessage) > 0 {
		sb.dg.ChannelMessageSend(SBitoa(info.config.WelcomeChannel), "<@"+SBitoa(IDs[0])+"> "+info.config.SilenceMessage)
	}
	if len(reason) > 0 {
		reason = " because " + reason
	}
	return fmt.Sprintf("```Silenced %s%s.```", IDsToUsernames(IDs, info)[0], reason), false
}
func (c *SilenceCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[user]", "Silences the given user.")
}
func (c *SilenceCommand) UsageShort() string { return "Silences a user." }

func UnsilenceMember(user uint64, info *GuildInfo) (int8, error) {
	srole := SBitoa(info.config.SilentRole)
	userID := SBitoa(user)
	m, err := sb.dg.GuildMember(info.Guild.ID, userID)
	if err != nil {
		return -1, err
	}
	for i := 0; i < len(m.Roles); i++ {
		if m.Roles[i] == srole {
			m.Roles = append(m.Roles[:i], m.Roles[i+1:]...)
			sb.dg.GuildMemberEdit(info.Guild.ID, userID, m.Roles)
			return 0, nil
		}
	}
	return 1, nil
}

type UnsilenceCommand struct {
}

func (c *UnsilenceCommand) Name() string {
	return "Unsilence"
}
func (c *UnsilenceCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You must provide a user to unsilence.```", false
	}
	arg := strings.Join(args, " ")
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	e, err := UnsilenceMember(IDs[0], info)
	if e == -1 {
		return "```Could not get member: " + err.Error() + "```", false
	} else if e == 1 {
		return "```" + IDsToUsernames(IDs, info)[0] + " wasn't silenced in the first place!```", false
	}
	return "```Unsilenced " + IDsToUsernames(IDs, info)[0] + ".```", false
}
func (c *UnsilenceCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[user]", "Unsilences the given user.")
}
func (c *UnsilenceCommand) UsageShort() string { return "Unsilences a user." }
