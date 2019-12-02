package miscmodule

import (
	"database/sql"
	"strings"
	"time"

	bot "../sweetiebot"
	"github.com/erikmcclure/discordgo"
)

// MiscModule contains miscellaneous commands
type MiscModule struct {
}

// Name of the module
func (w *MiscModule) Name() string {
	return "Miscellaneous"
}

// New instance of MiscModule
func New() *MiscModule {
	return &MiscModule{}
}

// Commands in the module
func (w *MiscModule) Commands() []bot.Command {
	return []bot.Command{
		&lastSeenCommand{},
		&searchCommand{statements: make(map[string][]*sql.Stmt)},
		&rollCommand{},
		&showrollCommand{},
		&snowflakeTimeCommand{},
		&pollCommand{},
	}
}

// Description of the module
func (w *MiscModule) Description(info *bot.GuildInfo) string {
	return "A collection of miscellaneous commands that don't belong to a module. Review the help information on each command for more details."
}

type lastSeenCommand struct {
}

func (c *lastSeenCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "LastSeen",
		Usage: "Returns when a user was last seen.",
	}
}
func (c *lastSeenCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(indices) < 1 {
		return "```\nYou have to give me someone to look for!```", false, nil
	}
	arg := msg.Content[indices[0]:]
	IDs := info.FindUsername(arg)
	if len(IDs) == 0 { // no matches!
		return "```\nError: Could not find any usernames or aliases matching " + arg + "!```", false, nil
	}
	if len(IDs) > 1 {
		return "```\nCould be any of the following users or their aliases:\n" + strings.Join(info.IDsToUsernames(IDs, true), "\n") + "```", len(IDs) > 5, nil
	}

	u, lastseen, _ := info.Bot.DB.GetMember(IDs[0], bot.SBatoi(info.ID))
	if u == nil {
		return "```\nError: User does not exist!```", false, nil
	}
	nick := u.User.Username
	if len(u.Nick) > 0 {
		nick = u.Nick
	}
	return "```\n" + nick + " last seen " + bot.TimeDiff(bot.GetTimestamp(msg).Sub(lastseen)) + " ago.```", false, nil
}
func (c *lastSeenCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Returns when a user was last seen on discord, which is usually their last status change.",
		Params: []bot.CommandUsageParam{
			{Name: "@user", Desc: "Either a ping for the user, their username, or their nickname.", Optional: false},
		},
	}
}

type snowflakeTimeCommand struct {
}

func (c *snowflakeTimeCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "SnowflakeTime",
		Usage: "Returns when a snowflake ID was created.",
	}
}
func (c *snowflakeTimeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou have to give me an ID!```", false, nil
	}
	ID := bot.SBatoi(bot.StripPing(args[0]))
	if ID == 0 {
		return "```\nInvalid snowflake ID.```", false, nil
	}
	t := bot.SnowflakeTime(ID)
	tz := info.GetTimezone(bot.DiscordUser(msg.Author.ID))
	return t.In(tz).Format(time.RFC1123), false, nil
}
func (c *snowflakeTimeCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Given a discord snowflake ID, returns when that ID was created.",
		Params: []bot.CommandUsageParam{
			{Name: "ID", Desc: "Any unique ID used by discord (these are called snowflake IDs)", Optional: false},
		},
	}
}
func (c *snowflakeTimeCommand) UsageShort() string { return "Returns when a snowflake ID was created." }
