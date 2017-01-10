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
func (c *LastSeenCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(indices) < 1 {
		return "```You have to give me someone to look for!```", false, nil
	}
	arg := msg.Content[indices[0]:]
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false, nil
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5, nil
	}

	u, lastseen := sb.db.GetMember(IDs[0], SBatoi(info.Guild.ID))
	if u == nil {
		return "```Error: User does not exist!```", false, nil
	}
	nick := u.User.Username
	if len(u.Nick) > 0 {
		nick = u.Nick
	}
	return "```" + nick + " last seen " + TimeDiff(SinceUTC(lastseen)) + " ago.```", false, nil
}
func (c *LastSeenCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Returns when a user was last seen on discord, which is usually their last status change.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "@user", Desc: "Either a ping for the user, their username, or their nickname.", Optional: false},
		},
	}
}
func (c *LastSeenCommand) UsageShort() string { return "Returns when a user was last seen." }
