package sweetiebot

import (
	"database/sql"
	"strings"
	"time"

	"github.com/blackhole12/discordgo"
)

type MiscModule struct {
	emotes *EmoteModule
}

// Name of the module
func (w *MiscModule) Name() string {
	return "Miscellaneous"
}

// Commands in the module
func (w *MiscModule) Commands() []Command {
	return []Command{
		&LastSeenCommand{},
		&searchCommand{emotes: w.emotes, statements: make(map[string][]*sql.Stmt)},
		&rollCommand{},
		&SnowflakeTimeCommand{},
	}
}

// Description of the module
func (w *MiscModule) Description() string {
	return "A collection of miscellaneous commands that don't belong to a module."
}

type LastSeenCommand struct {
}

func (c *LastSeenCommand) Name() string {
	return "LastSeen"
}
func (c *LastSeenCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(indices) < 1 {
		return "```You have to give me someone to look for!```", false, nil
	}
	arg := msg.Content[indices[0]:]
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false, nil
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info, true), "\n") + "```", len(IDs) > 5, nil
	}

	u, lastseen, _ := sb.db.GetMember(IDs[0], SBatoi(info.ID))
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
			{Name: "@user", Desc: "Either a ping for the user, their username, or their nickname.", Optional: false},
		},
	}
}
func (c *LastSeenCommand) UsageShort() string { return "Returns when a user was last seen." }

type SnowflakeTimeCommand struct {
}

func (c *SnowflakeTimeCommand) Name() string {
	return "SnowflakeTime"
}
func (c *SnowflakeTimeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You have to give me an ID!```", false, nil
	}
	ID := SBatoi(StripPing(args[0]))
	t := snowflakeTime(ID)
	tz := getTimezone(info, msg.Author)
	return t.In(tz).Format(time.RFC1123), false, nil
}
func (c *SnowflakeTimeCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Given a discord snowflake ID, returns when that ID was created.",
		Params: []CommandUsageParam{
			{Name: "ID", Desc: "Any unique ID used by discord (these are called snowflake IDs)", Optional: false},
		},
	}
}
func (c *SnowflakeTimeCommand) UsageShort() string { return "Returns when a snowflake ID was created." }
