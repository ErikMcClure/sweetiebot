package sweetiebot

import (
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

type LastPingCommand struct {
}

func (c *LastPingCommand) Name() string {
	return "LastPing"
}
func (c *LastPingCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	index := 1
	maxrows := 2
	if len(args) > 0 {
		index, _ = strconv.Atoi(args[0])
	}
	if len(args) > 1 {
		maxrows, _ = strconv.Atoi(args[1])
	}
	if index < 1 {
		index = 1
	}
	if maxrows < 0 {
		maxrows = 0
	}
	if maxrows > 3 {
		maxrows = 3
	}
	id, channel := sb.db.GetPing(SBatoi(msg.Author.ID), index-1, info.config.Basic.ModChannel, SBatoi(info.Guild.ID))
	if id == 0 {
		return "```No recent pings in the chat log.```", false, nil
	}

	after := sb.db.GetPingContext(id, channel, maxrows+1)
	before := sb.db.GetPingContextBefore(id, channel, maxrows)
	s := "```Pinged " + TimeDiff(SinceUTC(after[0].Timestamp)) + " ago, on " + ApplyTimezone(after[0].Timestamp, info, msg.Author).Format(time.RFC822) + "```\n"

	for i := len(before) - 1; i >= 0; i-- {
		s += before[i].Author + ": " + before[i].Message + "\n"
	}
	s += "**" + after[0].Author + ": " + after[0].Message + "**\n"
	for i := 1; i < len(after); i++ {
		s += after[i].Author + ": " + after[i].Message + "\n"
	}
	return s, true, nil
}
func (c *LastPingCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Returns the `n`th most recent ping (where `n` is the ping index) in the chat, plus up to `max context rows` messages before and after it.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "ping index", Desc: "Latest ping index, counting from the most recent at index 1, to the second most recent at index 2, etc.", Optional: true},
			CommandUsageParam{Name: "max context rows", Desc: "Number of rows before and after the ping to display. Defaults to 2, goes to a maximum of 3.", Optional: true},
		},
	}
}
func (c *LastPingCommand) UsageShort() string {
	return "[PM Only] Returns the last message that pinged you."
}
