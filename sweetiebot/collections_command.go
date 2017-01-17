package sweetiebot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type CollectionsModule struct {
	AddFuncMap    map[string]func(string) string
	RemoveFuncMap map[string]func(string) string
}

func (w *CollectionsModule) Name() string {
	return "Collection"
}

func (w *CollectionsModule) Register(info *GuildInfo) {}

func (w *CollectionsModule) Commands() []Command {
	return []Command{
		&AddCommand{w.AddFuncMap},
		&RemoveCommand{w.RemoveFuncMap},
		&CollectionsCommand{},
		&PickCommand{},
		&NewCommand{},
		&DeleteCommand{},
		&SearchCollectionCommand{},
		&ImportCommand{},
	}
}

func (w *CollectionsModule) Description() string {
	return "Contains commands for manipulating Sweetie Bot's collections."
}

type AddCommand struct {
	funcmap map[string]func(string) string
}

func (c *AddCommand) Name() string {
	return "Add"
}
func (c *AddCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```No collection given```", false, nil
	}
	if len(args) < 2 {
		return "```Can't add empty string!```", false, nil
	}

	collections := strings.Split(args[0], "+")
	for _, v := range collections {
		_, ok := info.config.Basic.Collections[v]
		if !ok {
			return fmt.Sprintf("```The %s collection does not exist!```", v), false, nil
		}
	}

	add := ""
	length := make([]string, len(collections), len(collections))
	arg := msg.Content[indices[1]:]
	for k, v := range collections {
		info.config.Basic.Collections[v][arg] = true
		fn, ok := c.funcmap[v]
		length[k] = fmt.Sprintf("Length of %s: %v", PartialSanitize(v), strconv.Itoa(len(info.config.Basic.Collections[v])))
		if ok {
			add += " " + fn(arg)
		}
	}
	info.SaveConfig()
	return fmt.Sprintf("```Added %s to %s%s. \n%s```", PartialSanitize(arg), PartialSanitize(strings.Join(collections, ", ")), add, strings.Join(length, "\n")), false, nil
}
func (c *AddCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Adds [arbitrary string] to [collection], then calls a handler function for that specific collection.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "collection(s)", Desc: "The name of a collection. Specify multiple collections with \"collection1+collection2\"", Optional: false},
			CommandUsageParam{Name: "arbitrary string", Desc: "Arbitrary string to add to collection. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}
func (c *AddCommand) UsageShort() string { return "Adds a line to a collection." }

type RemoveCommand struct {
	funcmap map[string]func(string) string
}

func (c *RemoveCommand) Name() string {
	return "Remove"
}
func (c *RemoveCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```No collection given```", false, nil
	}
	if len(args) < 2 {
		return "```Can't remove an empty string!```", false, nil
	}

	collection := args[0]
	cmap, ok := info.config.Basic.Collections[collection]
	if !ok {
		return "```That collection does not exist!```", false, nil
	}

	arg := msg.Content[indices[1]:]
	_, ok = cmap[arg]
	if !ok {
		return "```Could not find " + arg + "!```", false, nil
	}
	delete(info.config.Basic.Collections[collection], arg)
	fn, ok := c.funcmap[collection]
	retval := "```Removed " + PartialSanitize(arg) + " from " + PartialSanitize(collection) + ". Length of " + PartialSanitize(collection) + ": " + strconv.Itoa(len(info.config.Basic.Collections[collection])) + "```"
	if ok {
		retval = fn(arg)
	}

	info.SaveConfig()
	return retval, false, nil
}
func (c *RemoveCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Removes [arbitrary string] from [collection], then calls a handler function for that specific collection.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "collection(s)", Desc: "The name of a collection. Specifying multiple collections is not supported.", Optional: false},
			CommandUsageParam{Name: "arbitrary string", Desc: "Arbitrary string to remove from collection. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}
func (c *RemoveCommand) UsageShort() string { return "Removes a line from a collection." }

type MemberFields []*discordgo.MessageEmbedField

func (f MemberFields) Len() int {
	return len(f)
}

func (f MemberFields) Less(i, j int) bool {
	return strings.Compare(f[i].Name, f[j].Name) < 0
}

func (f MemberFields) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type CollectionsCommand struct {
}

func (c *CollectionsCommand) Name() string {
	return "Collections"
}
func ShowAllCollections(message string, info *GuildInfo) *discordgo.MessageEmbed {
	fields := make(MemberFields, 0, len(info.modules))

	for k, v := range info.config.Basic.Collections {
		fields = append(fields, &discordgo.MessageEmbedField{Name: k, Value: fmt.Sprintf("%v items", len(v)), Inline: true})
	}
	sort.Sort(fields)
	return &discordgo.MessageEmbed{
		Type: "rich",
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://github.com/blackhole12/sweetiebot",
			Name:    "Sweetie Bot Collections",
			IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", sb.SelfID, sb.SelfAvatar),
		},
		Description: message,
		Color:       0x3e92e5,
		Fields:      fields,
	}
}
func (c *CollectionsCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	const LINES int = 3
	const MAXLENGTH int = 24
	if len(args) < 1 {
		return "", false, ShowAllCollections("No collection specified.", info)
	}

	arg := args[0]
	cmap, ok := info.config.Basic.Collections[arg]
	if !ok {
		return "```That collection doesn't exist! Use this command with no arguments to see a list of all collections.```", false, nil
	}
	s := strings.Join(MapToSlice(cmap), "\n")
	s = strings.Replace(s, "```", "\\`\\`\\`", -1)
	s = strings.Replace(s, "[](/", "[\u200B](/", -1)
	return fmt.Sprintf("```\n%s contains:\n%s```", arg, s), false, nil
}
func (c *CollectionsCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists all the collections that sweetiebot is using, or the contents of a specific collection.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "collection", Desc: "The name of a collection. Specifying multiple collections is not supported.", Optional: true},
		},
	}
}
func (c *CollectionsCommand) UsageShort() string { return "Lists all collections." }

type PickCommand struct {
}

func (c *PickCommand) Name() string {
	return "Pick"
}
func (c *PickCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "", false, ShowAllCollections("No collection specified.", info)
	}

	arg := strings.ToLower(args[0])
	if arg == "spoiler" || arg == "emote" {
		return "```You cannot pick an item from that collection.```", false, nil
	}
	cmap, ok := info.config.Basic.Collections[arg]
	if !ok {
		return "```That collection doesn't exist! Use this command with no arguments to see a list of all collections.```", false, nil
	}
	if len(cmap) > 0 {
		return ReplaceAllMentions(MapGetRandomItem(cmap)), false, nil
	}
	return "```That collection is empty.```", false, nil
}
func (c *PickCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Picks a random item from the given collection and displays it.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "collection", Desc: "The name of a collection. Specifying multiple collections is not supported.", Optional: false},
		},
	}
}
func (c *PickCommand) UsageShort() string { return "Picks a random item." }

type NewCommand struct {
}

func (c *NewCommand) Name() string {
	return "New"
}
func (c *NewCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You have to provide a new collection name.```", false, nil
	}

	collection := strings.ToLower(args[0])
	if strings.ContainsAny(collection, "+") {
		return "```Don't make collection names with + in them, dumbass!```", false, nil
	}
	_, ok := info.config.Basic.Collections[collection]
	if ok {
		return "```That collection already exists!```", false, nil
	}
	info.config.Basic.Collections[collection] = make(map[string]bool)
	info.SaveConfig()

	return "```Created the " + collection + " collection.```", false, nil
}
func (c *NewCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Creates a new collection with the given name, provided the collection does not already exist.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "collection", Desc: "The name of the new collection. No spaces are allowed, should only use letters and numbers.", Optional: false},
		},
	}
}
func (c *NewCommand) UsageShort() string { return "Creates a new collection." }

type DeleteCommand struct {
}

func (c *DeleteCommand) Name() string {
	return "Delete"
}
func (c *DeleteCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You have to provide a collection name.```", false, nil
	}

	collection := strings.ToLower(args[0])
	_, ok := info.config.Basic.Collections[collection]
	if !ok {
		return "```That collection doesn't exist!```", false, nil
	}
	_, ok = map[string]bool{"emote": true, "bored": true, "status": true, "spoiler": true, "bucket": true}[collection]
	if ok {
		return "```You can't delete that collection!```", false, nil
	}
	delete(info.config.Basic.Collections, collection)
	info.SaveConfig()

	return "```Deleted the " + collection + " collection.```", false, nil
}
func (c *DeleteCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Deletes a collection with the given name, provided the collection is not protected.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "collection", Desc: "The name of the collection. Certain collections cannot be deleted.", Optional: false},
		},
	}
}
func (c *DeleteCommand) UsageShort() string { return "Deletes a collection." }

type SearchCollectionCommand struct {
}

func (c *SearchCollectionCommand) Name() string {
	return "SearchCollection"
}
func (c *SearchCollectionCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You have to provide a new collection name.```", false, nil
	}
	if len(args) < 2 {
		return "```You have to provide something to search for (use !collections to dump the contents of a collection).```", false, nil
	}

	collection := strings.ToLower(args[0])
	if collection == "spoiler" {
		return "```You can't search in that collection.```", false, nil
	}
	cmap, ok := info.config.Basic.Collections[collection]
	if !ok {
		return "```That collection doesn't exist! Use !collections without any arguments to list them.```", false, nil
	}
	results := []string{}
	arg := msg.Content[indices[1]:]
	for k, _ := range cmap {
		if strings.Contains(k, arg) {
			results = append(results, k)
		}
	}

	if len(results) > 0 {
		return "```The following collection entries match your query:\n" + PartialSanitize(strings.Join(results, "\n")) + "```", len(results) > 6, nil
	}
	return "```No results found in the " + collection + " collection.```", false, nil
}
func (c *SearchCollectionCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Returns all members of the given collection that contain the given string.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "collection", Desc: "The name of the collection. Specifying multiple collections is not supported.", Optional: false},
			CommandUsageParam{Name: "arbitrary string", Desc: "Arbitrary string to add to collection. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}
func (c *SearchCollectionCommand) UsageShort() string { return "Searches a collection." }

type ImportCommand struct {
}

func (c *ImportCommand) Name() string {
	return "Import"
}
func (c *ImportCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```No source server provided.```", false, nil
	}

	other := []*GuildInfo{}
	str := args[0]
	exact := false
	if str[len(str)-1] == '@' {
		str = str[:len(str)-1]
		exact = true
	}
	for _, v := range sb.guilds {
		if exact {
			if strings.Compare(strings.ToLower(v.Guild.Name), strings.ToLower(str)) == 0 {
				other = append(other, v)
			}
		} else {
			if strings.Contains(strings.ToLower(v.Guild.Name), strings.ToLower(str)) {
				other = append(other, v)
			}
		}
	}
	if len(other) > 1 {
		names := make([]string, len(other), len(other))
		for i := 0; i < len(other); i++ {
			names[i] = other[i].Guild.Name
		}
		return fmt.Sprintf("```Could be any of the following servers: \n%s```", PartialSanitize(strings.Join(names, "\n"))), len(names) > 8, nil
	}
	if len(other) < 1 {
		return fmt.Sprintf("```Could not find any server matching %s!```", args[0]), false, nil
	}
	if !other[0].config.Basic.Importable {
		return "```That server has not made their collections importable by other servers. If this is a public server, you can ask a moderator on that server to run \"!setconfig importable true\" if they wish to make their collections public.```", false, nil
	}

	if len(args) < 2 {
		return "```No source collection provided.```", false, nil
	}
	source := args[1]
	target := source
	if len(args) > 2 {
		target = args[2]
	}

	sourceCollection, ok := other[0].config.Basic.Collections[source]
	if !ok {
		return fmt.Sprintf("```The source collection (%s) does not exist on the source server (%s)!```", source, other[0].Guild.Name), false, nil
	}

	targetCollection, tok := info.config.Basic.Collections[target]
	if !tok {
		return fmt.Sprintf("```The target collection (%s) does not exist on this server! Please manually create this collection using !new if you actually intended this.```", target), false, nil
	}

	for k, v := range sourceCollection {
		targetCollection[k] = v
	}

	info.SaveConfig()
	return fmt.Sprintf("```Successfully merged \"%s\" from %s into \"%s\" on this server. New size: %v```", source, other[0].Guild.Name, target, len(targetCollection)), false, nil
}
func (c *ImportCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Adds all elements from the source collection on the source server to the target collection on this server. If no target is specified, attempts to copy all items into a collection of the same name as the source. Example: ```!import Manechat cool notcool```",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "source server", Desc: "The exact name of the source server to copy from.", Optional: false},
			CommandUsageParam{Name: "source collection", Desc: "Name of the collection to copy from on the source server.", Optional: false},
			CommandUsageParam{Name: "target collection", Desc: "The target collection to copy to on this server. If omitted, defaults to the source collection name.", Optional: true},
		},
	}
}
func (c *ImportCommand) UsageShort() string { return "Imports a collection from another server." }
