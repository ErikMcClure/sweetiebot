package sweetiebot

import (
	"fmt"
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
		for k, v := range info.config.Collections {
			s = append(s, fmt.Sprintf("%s (%v items)", k, len(v)))
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

type PickCommand struct {
}

func (c *PickCommand) Name() string {
	return "Pick"
}
func (c *PickCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		s := make([]string, 0, len(info.config.Collections))
		for k, v := range info.config.Collections {
			s = append(s, fmt.Sprintf("%s (%v items)", k, len(v)))
		}

		return "```No collection specified. All collections:\n" + ExtraSanitize(strings.Join(s, "\n")) + "```", false
	}

	arg := strings.ToLower(args[0])
	if arg == "spoiler" || arg == "emote" {
		return "```You cannot pick an item from that collection.```", false
	}
	cmap, ok := info.config.Collections[arg]
	if !ok {
		return "```That collection doesn't exist! Use this command with no arguments to see a list of all collections.```", false
	}
	if len(cmap) > 0 {
		return ReplaceAllMentions(MapGetRandomItem(cmap)), false
	}
	return "```That collection is empty.```", false
}
func (c *PickCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[collection]", "Picks a random item from the given collection and returns it.")
}
func (c *PickCommand) UsageShort() string { return "Picks a random item." }

type NewCommand struct {
}

func (c *NewCommand) Name() string {
	return "New"
}
func (c *NewCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You have to provide a new collection name.```", false
	}

	collection := strings.ToLower(args[0])
	_, ok := info.config.Collections[collection]
	if ok {
		return "```That collection already exists!```", false
	}
	info.config.Collections[collection] = make(map[string]bool)
	info.SaveConfig()

	return "```Created the " + collection + " collection.```", false
}
func (c *NewCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[collection]", "Creates a new collection with the given name, provided the collection does not already exist.")
}
func (c *NewCommand) UsageShort() string { return "Creates a new collection." }

type DeleteCommand struct {
}

func (c *DeleteCommand) Name() string {
	return "Delete"
}
func (c *DeleteCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You have to provide a collection name.```", false
	}

	collection := strings.ToLower(args[0])
	_, ok := info.config.Collections[collection]
	if !ok {
		return "```That collection doesn't exist!```", false
	}
	_, ok = map[string]bool{"emote": true, "bored": true, "status": true, "spoiler": true, "bucket": true}[collection]
	if ok {
		return "```You can't delete that collection!```", false
	}
	delete(info.config.Collections, collection)
	info.SaveConfig()

	return "```Deleted the " + collection + " collection.```", false
}
func (c *DeleteCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[collection]", "Deletes the collection with the given name.")
}
func (c *DeleteCommand) UsageShort() string { return "Deletes a collection." }

type SearchCollectionCommand struct {
}

func (c *SearchCollectionCommand) Name() string {
	return "SearchCollection"
}
func (c *SearchCollectionCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You have to provide a new collection name.```", false
	}
	if len(args) < 2 {
		return "```You have to provide something to search for (use !collections to dump the contents of a collection).```", false
	}

	collection := strings.ToLower(args[0])
	if collection == "spoiler" {
		return "```You can't search in that collection.```", false
	}
	cmap, ok := info.config.Collections[collection]
	if !ok {
		return "```That collection doesn't exist! Use !collections without any arguments to list them.```", false
	}
	results := []string{}
	arg := strings.Join(args[1:], " ")
	for k, _ := range cmap {
		if strings.Contains(k, arg) {
			results = append(results, k)
		}
	}

	if len(results) > 0 {
		return "```The following collection entries match your query:\n" + ExtraSanitize(strings.Join(results, "\n")) + "```", len(results) > 6
	}
	return "```No results found in the " + collection + " collection.```", false
}
func (c *SearchCollectionCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[collection] [arbitrary string]", "Returns all members of the given collection that match the search query.")
}
func (c *SearchCollectionCommand) UsageShort() string { return "Searches a collection." }

type ImportCommand struct {
}

func (c *ImportCommand) Name() string {
	return "Import"
}
func (c *ImportCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```No source server provided.```", false
	}

	other := []*GuildInfo{}
	for _, v := range sb.guilds {
		if strings.Contains(strings.ToLower(v.Guild.Name), strings.ToLower(args[0])) {
			other = append(other, v)
		}
	}
	if len(other) > 1 {
		names := make([]string, len(other), len(other))
		for i := 0; i < len(other); i++ {
			names[i] = other[i].Guild.Name
		}
		return fmt.Sprintf("```Could be any of the following servers: \n%s```", ExtraSanitize(strings.Join(names, "\n"))), len(names) > 8
	}
	if len(other) < 1 {
		return fmt.Sprintf("```Could not find any server matching %s!```", args[0]), false
	}
	if !other[0].config.Importable {
		return "```That server has not made their collections importable by other servers. If this is a public server, you can ask a moderator on that server to run \"!setconfig importable true\" if they wish to make their collections public.```", false
	}

	if len(args) < 2 {
		return "```No source collection provided.```", false
	}
	source := args[1]
	target := source
	if len(args) > 2 {
		target = args[2]
	}

	sourceCollection, ok := other[0].config.Collections[source]
	if !ok {
		return fmt.Sprintf("```The source collection (%s) does not exist on the source server (%s)!```", source, other[0].Guild.Name), false
	}

	targetCollection, tok := info.config.Collections[target]
	if !tok {
		return fmt.Sprintf("```The target collection (%s) does not exist on this server! Please manually create this collection using !new if you actually intended this.```", target), false
	}

	for k, v := range sourceCollection {
		targetCollection[k] = v
	}

	info.SaveConfig()
	return fmt.Sprintf("```Successfully merged \"%s\" from %s into \"%s\" on this server. New size: %v```", source, other[0].Guild.Name, target, len(targetCollection)), false
}
func (c *ImportCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[source server] [source collection] [target collection]", "Adds all elements from the source collection on the source server to the target collection on this server. If no target is specified, attempts to copy all items into a collection of the same name as the source. Example: \"!import Manechat cool notcool\"")
}
func (c *ImportCommand) UsageShort() string { return "Imports a collection from another server." }
