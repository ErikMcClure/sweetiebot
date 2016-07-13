package sweetiebot

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type AddCommand struct {
	funcmap map[string]func(string) string
}

func (c *AddCommand) Name() string {
	return "Add"
}
func (c *AddCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```No collection given```", false
	}
	if len(args) < 2 {
		return "```Can't add empty string!```", false
	}

	collection := args[0]
	_, ok := info.config.Collections[collection]
	if !ok {
		return "```That collection does not exist!```", false
	}

	arg := strings.Join(args[1:], " ")
	info.config.Collections[collection][arg] = true
	fn, ok := c.funcmap[collection]
	retval := "```Added " + arg + " to " + collection + ". Length of " + collection + ": " + strconv.Itoa(len(info.config.Collections[collection])) + "```"
	if ok {
		retval = fn(arg)
	}
	info.SaveConfig()
	return ExtraSanitize(retval), false
}
func (c *AddCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[collection] [arbitrary string]", "Adds [arbitrary string] to [collection] (no quotes are required), then calls a handler function for that specific collection.")
}
func (c *AddCommand) UsageShort() string { return "Adds a line to a collection." }

type RemoveCommand struct {
	funcmap map[string]func(string) string
}

func (c *RemoveCommand) Name() string {
	return "Remove"
}
func (c *RemoveCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```No collection given```", false
	}
	if len(args) < 2 {
		return "```Can't remove an empty string!```", false
	}

	collection := args[0]
	cmap, ok := info.config.Collections[collection]
	if !ok {
		return "```That collection does not exist!```", false
	}

	arg := strings.Join(args[1:], " ")
	_, ok = cmap[arg]
	if !ok {
		return "```Could not find " + arg + "!```", false
	}
	delete(info.config.Collections[collection], arg)
	fn, ok := c.funcmap[collection]
	retval := "```Removed " + arg + " from " + collection + ". Length of " + collection + ": " + strconv.Itoa(len(info.config.Collections[collection])) + "```"
	if ok {
		retval = fn(arg)
	}

	info.SaveConfig()
	return ExtraSanitize(retval), false
}
func (c *RemoveCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[collection] [arbitrary string]", "Removes [arbitrary string] from [collection] (no quotes are required) and calls a handler function for that collection.")
}
func (c *RemoveCommand) UsageShort() string { return "Removes a line from a collection." }

type CollectionsCommand struct {
}

func (c *CollectionsCommand) Name() string {
	return "Collections"
}
func (c *CollectionsCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		s := make([]string, 0, len(info.config.Collections))
		for k, _ := range info.config.Collections {
			s = append(s, k)
		}

		return "```No collection specified. All collections:\n" + ExtraSanitize(strings.Join(s, "\n")) + "```", false
	}

	arg := args[0]
	cmap, ok := info.config.Collections[arg]
	if !ok {
		return "```That collection doesn't exist! Use this command with no arguments to see a list of all collections.```", false
	}

	return "```" + ExtraSanitize(arg+" contains:\n"+strings.Join(MapToSlice(cmap), "\n")) + "```", false
}
func (c *CollectionsCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "", "Lists all the collections that sweetiebot is using.")
}
func (c *CollectionsCommand) UsageShort() string { return "Lists all collections." }
