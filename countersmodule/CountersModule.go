package countersmodule

import (
	"fmt"
	"strconv"
	"strings"

	bot "../sweetiebot"
	"github.com/erikmcclure/discordgo"
)

// CountersModule manages incrementable counters
type CountersModule struct {
}

// New instance of CountersModule
func New() *CountersModule {
	return &CountersModule{}
}

// Name of the module
func (w *CountersModule) Name() string {
	return "Counters"
}

// Commands in the module
func (w *CountersModule) Commands() []bot.Command {
	return []bot.Command{
		&addCounterCommand{},
		&removeCounterCommand{},
		&counterCommand{},
		&incrementCommand{},
	}
}

// Description of the module
func (w *CountersModule) Description(info *bot.GuildInfo) string {
	return "Allows creating and managing incrementable counters."
}

type addCounterCommand struct {
}

func (c *addCounterCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "AddCounter",
		Usage:     "Creates an incrementable counter with an initial value and description.",
		Sensitive: true,
	}
}
func (c *addCounterCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou at least have to provide a name for your counter, but you should probably provide an initial value and description too.```", false, nil
	}

	name := info.Sanitize(args[0], bot.CleanMentions|bot.CleanPings|bot.CleanEmotes|bot.CleanCode)
	if _, ok := info.Config.Counters.Map[name]; ok {
		return "```\nThat counter already exists!```", false, nil
	}

	var init int64
	desc := name + " is at %%"

	if len(args) > 1 {
		var err error
		init, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return "```" + args[1] + " is not a number.```", false, nil
		}
	}

	if len(args) > 2 {
		desc = info.Sanitize(msg.Content[indices[2]:], bot.CleanMentions|bot.CleanPings|bot.CleanEmotes|bot.CleanCode)
	}

	info.Config.Counters.Map[name] = init
	info.Config.Counters.Descriptions[name] = desc
	info.SaveConfig()
	return fmt.Sprintf("```\nAdded the %v counter, starting at %v```", name, init), false, nil
}
func (c *addCounterCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Creates a new counter with an initial value and description that can be incremented with the `" + info.Config.Basic.CommandPrefix + "increment` command.",
		Params: []bot.CommandUsageParam{
			{Name: "name", Desc: "A short name for the counter. Quotes are required if it has spaces.", Optional: false},
			{Name: "initial value", Desc: "A number to start the counter at. If omitted, defaults to 0.", Optional: true},
			{Name: "description", Desc: "The string echoed by " + info.GetBotName() + " when the counter is incremented. Defaults to `[counter] is at %%`, where `%%` is replaced by the counter value. If the string doesn't contain `%%` it pastes the value of the counter on the end.", Optional: true},
		},
	}
}

type removeCounterCommand struct {
}

func (c *removeCounterCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "RemoveCounter",
		Usage:     "Deletes a counter.",
		Sensitive: true,
	}
}
func (c *removeCounterCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou must provide the name of the counter to delete.```", false, nil
	}
	arg := info.Sanitize(msg.Content[indices[0]:], bot.CleanMentions|bot.CleanPings|bot.CleanEmotes|bot.CleanCode)
	if _, ok := info.Config.Counters.Map[arg]; ok {
		delete(info.Config.Counters.Map, arg)
		delete(info.Config.Counters.Descriptions, arg)
		info.SaveConfig()
		return "```\nDeleted " + arg + ".```", false, nil
	}
	return "```\nThat counter doesn't exist!```", false, nil
}
func (c *removeCounterCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Deletes the counter with the given name.",
		Params: []bot.CommandUsageParam{
			{Name: "name", Desc: "Name of the counter to delete.", Optional: false},
		},
	}
}

func resolveDesc(counter int64, desc string) string {
	if strings.Contains(desc, "%%") {
		return strings.Replace(desc, "%%", strconv.FormatInt(counter, 10), -1)
	}
	return desc + " " + strconv.FormatInt(counter, 10)
}

type counterCommand struct {
}

func (c *counterCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "Counter",
		Usage: "Returns the value of a counter, or a list of counters if none is provided.",
	}
}
func (c *counterCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	response := []string{"All available counters:"}
	if len(args) > 0 {
		arg := info.Sanitize(msg.Content[indices[0]:], bot.CleanMentions|bot.CleanPings|bot.CleanEmotes|bot.CleanCode)
		if counter, ok := info.Config.Counters.Map[arg]; ok {
			desc, _ := info.Config.Counters.Descriptions[arg]
			return resolveDesc(counter, desc), false, nil
		}
		response[0] = "Could not find " + arg + "! " + response[0]
	}

	for k := range info.Config.Counters.Map {
		response = append(response, k)
	}

	return strings.Join(response, "\n"), false, nil
}
func (c *counterCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Returns the value of a counter, or a list of counters if no name is provided.",
		Params: []bot.CommandUsageParam{
			{Name: "name", Desc: "Name of the counter to display.", Optional: true},
		},
	}
}

type incrementCommand struct {
}

func (c *incrementCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "Increment",
		Usage:     "Increments a given counter by 1.",
		Sensitive: true,
	}
}
func (c *incrementCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You must provide a counter to increment!```", false, nil
	}

	arg := info.Sanitize(msg.Content[indices[0]:], bot.CleanMentions|bot.CleanPings|bot.CleanEmotes|bot.CleanCode)
	if counter, ok := info.Config.Counters.Map[arg]; ok {
		counter++
		info.Config.Counters.Map[arg] = counter
		desc, _ := info.Config.Counters.Descriptions[arg]
		info.SaveConfig()
		return resolveDesc(counter, desc), false, nil
	}
	return arg + " is not a counter!", false, nil
}
func (c *incrementCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Increments a counter by 1 and returns the new value.",
		Params: []bot.CommandUsageParam{
			{Name: "name", Desc: "Name of the counter to increment.", Optional: false},
		},
	}
}
