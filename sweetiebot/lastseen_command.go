package sweetiebot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

type LastSeenCommand struct {
}

func (c *LastSeenCommand) Name() string {
	return "LastSeen"
}
func (c *LastSeenCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	arg := strings.Join(args, " ")
	IDs := FindUsername(arg)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs), "\n") + "```", len(IDs) > 5
	}

	u, lastseen := sb.db.GetUser(IDs[0])
	return "```" + u.Username + " last seen " + TimeDiff(SinceUTC(lastseen)) + " ago.```", false
}
func (c *LastSeenCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[@user]", "Returns when a user was last seen on discord, which is usually their last status change.")
}
func (c *LastSeenCommand) UsageShort() string { return "Returns when a user was last seen." }
