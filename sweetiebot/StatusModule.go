package sweetiebot

import (
	"fmt"

	"github.com/blackhole12/discordgo"
)

// StatusModule manages Sweetie Bot's status
type StatusModule struct {
}

// Name of the module
func (w *StatusModule) Name() string {
	return "Status"
}

// Commands in the module
func (w *StatusModule) Commands() []Command {
	return []Command{
		&setStatusCommand{},
	}
}

// Description of the module
func (w *StatusModule) Description() string { return "Manages Sweetie Bot's status." }

type setStatusCommand struct {
}

func (c *setStatusCommand) Name() string {
	return "SetStatus"
}
func (c *setStatusCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		sb.dg.UpdateStatus(0, "")
		return "```Removed status```", false, nil
	}
	arg := msg.Content[indices[0]:]
	fmt.Printf(arg)
	sb.dg.UpdateStatus(0, arg)
	return "```Set status to " + arg + "```", false, nil
}
func (c *setStatusCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Sets Sweetie Bot's status message to the given string, at least until she automatically changes it again.",
		Params: []CommandUsageParam{
			{Name: "arbitrary string", Desc: "String to set the status to. Be careful that it's a valid Discord status.", Optional: false},
		},
	}
}
func (c *setStatusCommand) UsageShort() string { return "Sets the status message." }
