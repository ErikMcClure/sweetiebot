package statusmodule

import (
	"time"

	bot "../sweetiebot"
	"github.com/erikmcclure/discordgo"
)

// StatusModule manages the status message
type StatusModule struct {
	lastchange time.Time
}

// New StatusModule
func New() *StatusModule {
	return &StatusModule{}
}

// Name of the module
func (w *StatusModule) Name() string {
	return "Status"
}

// Commands in the module
func (w *StatusModule) Commands() []bot.Command {
	return []bot.Command{
		&setStatusCommand{},
		&addStatusCommand{},
		&removeStatusCommand{},
	}
}

// Description of the module
func (w *StatusModule) Description(info *bot.GuildInfo) string { return "Manages the status message." }

// OnTick discord hook
func (w *StatusModule) OnTick(info *bot.GuildInfo, t time.Time) {
	if info.Bot.IsMainGuild(info) {
		if w.lastchange.Add(time.Duration(info.Config.Status.Cooldown) * time.Second).Before(t) {
			w.lastchange = t
			if len(info.Config.Status.Lines) > 0 {
				info.Bot.DG.UpdateStatus(0, bot.MapGetRandomItem(info.Config.Status.Lines))
			}
		}
	}
}

type setStatusCommand struct {
}

func (c *setStatusCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:         "SetStatus",
		Usage:        "Sets the status message.",
		Sensitive:    true,
		MainInstance: true,
	}
}
func (c *setStatusCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.MainGuildID.Equals(info.ID) {
		return "```\nYou can only do this from the main server!```", false, nil
	}
	if len(args) < 1 {
		info.Bot.DG.UpdateStatus(0, "")
		return "```\nRemoved status```", false, nil
	}
	arg := msg.Content[indices[0]:]
	info.Bot.DG.UpdateStatus(0, arg)
	return "```\nStatus was set to " + arg + "```", false, nil
}
func (c *setStatusCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Sets the status message to the given string, at least until it's automatically changed again. Only works from the main guild.",
		Params: []bot.CommandUsageParam{
			{Name: "arbitrary string", Desc: "String to set the status to. Be careful that it's a valid Discord status.", Optional: false},
		},
	}
}

type addStatusCommand struct {
}

func (c *addStatusCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:         "AddStatus",
		Usage:        "Adds a status to the rotation",
		Sensitive:    true,
		MainInstance: true,
	}
}
func (c *addStatusCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.MainGuildID.Equals(info.ID) {
		return "```\nYou can only do this from the main server!```", false, nil
	}
	if len(args) < 1 {
		return "```\nNo status given.```", false, nil
	}
	arg := msg.Content[indices[0]:]
	_, ok := info.Config.Status.Lines[arg]
	if ok {
		return "```\n" + arg + " is already in the status rotation!```", false, nil
	}
	info.Config.Status.Lines[arg] = true
	info.SaveConfig()
	return "```\nAdded " + arg + " to the status rotation.```", false, nil
}
func (c *addStatusCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Adds a string to the discord status rotation.",
		Params: []bot.CommandUsageParam{
			{Name: "arbitrary string", Desc: "Status string. Be careful that it's a valid Discord status.", Optional: false},
		},
	}
}

type removeStatusCommand struct {
}

func (c *removeStatusCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:         "RemoveStatus",
		Usage:        "Removes a status message from the rotation.",
		Sensitive:    true,
		MainInstance: true,
	}
}
func (c *removeStatusCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.MainGuildID.Equals(info.ID) {
		return "```\nYou can only do this from the main server!```", false, nil
	}
	if len(args) < 1 {
		return "```\nNo status given.```", false, nil
	}
	arg := msg.Content[indices[0]:]
	_, ok := info.Config.Status.Lines[arg]
	if !ok {
		return "```\n" + arg + " is not in the status rotation!```", false, nil
	}
	delete(info.Config.Status.Lines, arg)
	info.SaveConfig()
	return "```\nRemoved " + arg + " from the status rotation.```", false, nil
}
func (c *removeStatusCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Removes a string to the discord status rotation. Use " + info.Config.Basic.CommandPrefix + "getconfig status.lines to get a list of all strings currently in rotation.",
		Params: []bot.CommandUsageParam{
			{Name: "arbitrary string", Desc: "Status string that must exactly match the one you want to remove.", Optional: false},
		},
	}
}
