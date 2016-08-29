package sweetiebot

import (
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
		s = append(s, v.User.Username+"  (joined: "+ApplyTimezone(v.FirstSeen, info).Format(time.ANSIC)+") ["+v.User.ID+"]")
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
	IDs := FindUsername(arg)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs), "\n") + "```", len(IDs) > 5
	}

	r := sb.db.GetAliases(IDs[0])
	u, _ := sb.db.GetUser(IDs[0])
	return "```All known aliases for " + u.Username + "\n  " + strings.Join(r, "\n  ") + "```", !CheckShutup(msg.ChannelID)
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
	IDs := FindUsername(arg)
	if len(IDs) == 0 { // no matches
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs), "\n") + "```", len(IDs) > 5
	}
	// we're done with our checks
	// actually ban the user here and send the output. This is probably poorly done.
	gID := info.Guild.ID
	u, _ := sb.db.GetUser(IDs[0])
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
		return "```This server's local time is: " + ApplyTimezone(time.Now().UTC(), info).Format("Jan 2, 3:04pm```"), false
	}

	arg := strings.Join(args, " ")
	IDs := FindUsername(arg)
	if len(IDs) == 0 { // no matches
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs), "\n") + "```", len(IDs) > 5
	}

	tz := sb.db.GetTimeZone(IDs[0])
	if !tz.Valid {
		return "```That user has not specified what their timezone is.```", false
	}
	return "```That user's local time is: " + time.Now().UTC().Add(time.Duration(tz.Int64)*time.Hour).Format("Jan 2, 3:04pm```"), false
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
		return "You have to specify what your timezone is!", false
	}

	tz, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return "Could not parse timezone. The timezone must be the number of hours difference between you and GMT. For example, if your timezone is EST, that's GMT-5, so use '-5' as the argument. This does not account for daylight savings!", false
	}

	if sb.db.SetTimeZone(SBatoi(msg.Author.ID), tz) != nil {
		return "Error: could not set timezone!", false
	}
	return "Set your timezone to " + args[0], false
}
func (c *SetTimeZoneCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[timezone]", "Sets your timezone to the given GMT offset, in hours. For example, '-7' would set your timezone to GMT-7, or PDT. This does not account for daylight savings!")
}
func (c *SetTimeZoneCommand) UsageShort() string { return "Set your local timezone." }
