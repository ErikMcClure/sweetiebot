package sweetiebot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

type HelpCommand struct {
}

func (c *HelpCommand) Name() string {
	return "Help"
}
func (c *HelpCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) == 0 {
		s := []string{"Sweetie Bot knows the following commands. For more information on a specific command, type !help [command].\n"}
		for k, v := range info.commands {
			s = append(s, k+": "+v.UsageShort())
		}

		return "```" + strings.Join(s, "\n") + "```", true
	}
	v, ok := info.commands[strings.ToLower(args[0])]
	if !ok {
		return "``` Sweetie Bot doesn't recognize that command. You can check what commands Sweetie Bot knows by typing !help.```", false
	}
	return "```> !" + v.Name() + " " + v.Usage(info) + "```", true
}
func (c *HelpCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[command]", "Lists all available commands Sweetie Bot knows, or gives information about the given command. Of course, you should have figured this out by now, since you just typed !help help for some reason.")
}
func (c *HelpCommand) UsageShort() string {
	return "[PM Only] Generates the list you are looking at right now."
}

type AboutCommand struct {
}

func (c *AboutCommand) Name() string {
	return "About"
}
func (c *AboutCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	s := "```Sweetie Bot version " + sb.version
	if sb.Debug {
		return s + " [debug]```", false
	}
	return s + " [release]```", false
}
func (c *AboutCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "", "Displays information about Sweetie Bot. What, did you think it would do something else?")
}
func (c *AboutCommand) UsageShort() string { return "Displays information about Sweetie Bot." }
