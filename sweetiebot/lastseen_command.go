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
	var id uint64
	if userregex.MatchString(arg) {
		id = SBatoi(arg[2 : len(arg)-1])
	} else {
		IDs := sb.db.FindUsers("%"+arg+"%", 20, 0)
		if len(IDs) == 0 { // no matches!
			return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
		}
		if len(IDs) > 1 {
			s := []string{}

			for _, v := range IDs {
				u, _ := sb.db.GetUser(v)
				s = append(s, u.Username)
			}

			return "```Could be any of the following users or their aliases:\n" + strings.Join(s, "\n") + "```", len(s) > 5
		}
		id = IDs[0]
	}

	u, lastseen := sb.db.GetUser(id)
	return "```" + u.Username + " last seen " + TimeDiff(SinceUTC(lastseen)) + " ago.```", false
}
func (c *LastSeenCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[@user]", "Returns when a user was last seen on discord, which is usually their last status change.")
}
func (c *LastSeenCommand) UsageShort() string { return "Returns when a user was last seen." }
