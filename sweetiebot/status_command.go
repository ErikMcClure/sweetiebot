package sweetiebot

import (
	"github.com/bwmarrin/discordgo"
)

type StatusModule struct {
}

func (w *StatusModule) Name() string {
	return "Status"
}

func (w *StatusModule) Register(info *GuildInfo) {}

func (w *StatusModule) Commands() []Command {
	return []Command{
		&SetStatusCommand{},
	}
}

func (w *StatusModule) Description() string { return "Manages Sweetie Bot's status." }

type SetStatusCommand struct {
}

func (c *SetStatusCommand) Name() string {
	return "SetStatus"
}
func (c *SetStatusCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		sb.dg.UpdateStatus(0, "")
		return "```Removed status```", false, nil
	}
	arg := msg.Content[indices[0]:]
	sb.dg.UpdateStatus(0, arg)
	return "```Set status to " + arg + "```", false, nil
}
func (c *SetStatusCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Sets Sweetie Bot's status message to the given string, at least until she automatically changes it again.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "arbitrary string", Desc: "String to set the status to. Be careful that it's a valid Discord status.", Optional: false},
		},
	}
}
func (c *SetStatusCommand) UsageShort() string { return "Sets the status message." }
