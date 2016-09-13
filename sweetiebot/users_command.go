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

// experimental ban command for admins to ban users from the server with extreme prejudice
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
	arg := strings.Join(args, " ")
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}
	// we're done with our checks
	// actually ban the user here and send the output. This is probably poorly done.
	gID := info.Guild.ID
	u, _, _ := sb.db.GetUser(IDs[0])
	if u == nil {
		return "```Error: User does not exist!```", false
	}
	uID := SBitoa(IDs[0])
	sb.dg.GuildBanCreate(gID, uID, 1)

	return "```Banned " + u.Username + " from the server. Harmony restored.```", !CheckShutup(msg.ChannelID)
}
func (c *BanCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[@user]", "Commands Sweetie Bot to ban a given user.")
}
func (c *BanCommand) UsageShort() string { return "Commands Sweetie Bot to ban a given user." }

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
			return "```Could not parse offset. The second argument should be your time difference from GMT in hours. For example, PDT is GMT-7, so you'd put -7.```", false
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
	return info.FormatUsage(c, "[timezone] [offset]", "Sets your timezone to the given location, such as \"America/Los_Angeles\". Providing a partial timezone name, like \"America\", will return a list of all possible timezones that contain that string. You can also specify your expected timezone offset in hours to narrow the search. For example, if you know you're in the PDT timezone, which is GMT-7, you could search for \"America -7\" to list all timezones in america with a standard or DST timezone offset of -7.")
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
	dbuser, lastseen, tz := sb.db.GetUser(IDs[0])
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
