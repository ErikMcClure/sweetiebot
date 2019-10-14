package filtermodule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"../spammodule"
	bot "../sweetiebot"
	"github.com/blackhole12/discordgo"
)

// FilterModule implements word filters that allow you to look for spoilers or profanity uses regex matching.
type FilterModule struct {
	spam    *spammodule.SpamModule
	filters map[string]*regexp.Regexp
	lastmsg int64 // Universal saturation limit on all filter responses
}

// New instance of FilterModule
func New(info *bot.GuildInfo, s *spammodule.SpamModule) *FilterModule {
	w := &FilterModule{
		filters: make(map[string]*regexp.Regexp),
		lastmsg: 0,
		spam:    s,
	}
	for k := range info.Config.Filter.Filters {
		w.UpdateRegex(k, info)
	}
	return w
}

// Name of the module
func (w *FilterModule) Name() string {
	return "Filter"
}

// Commands in the module
func (w *FilterModule) Commands() []bot.Command {
	return []bot.Command{
		&setFilterCommand{},
		&addFilterCommand{w},
		&removeFilterCommand{w},
		&deleteFilterCommand{w},
		&searchFilterCommand{},
	}
}

// Description of the module
func (w *FilterModule) Description(info *bot.GuildInfo) string {
	return "Implements customizable filters that search for forbiddan words or phrases and removes them with a customizable response and excludable channels. Optionally also adds pressure to the user for triggering a filter, and if the response is set to !, doesn't remove the message at all, only adding pressure.\n\nIf you just want a basic case-insensitive word filter that respects spaces, use `!setconfig filter.templates` with your filter name and this template: `(?i)(^| )%%($| )`. \n\nExample usage: \n```!setfilter badwords \"This is a christian server, no swearing allowed.\"\n!setconfig filter.templates badwords (?i)(^| )%%($| )\n!addfilter badwords hell\n!addfilter badwords \"jesus christ\"```"
}

func (w *FilterModule) matchFilter(info *bot.GuildInfo, m *discordgo.Message) bool {
	author := bot.DiscordUser(m.Author.ID)
	if info.UserIsMod(author) || info.UserIsAdmin(author) || m.Author.Bot {
		return false
	}

	timestamp := bot.GetTimestamp(m)
	for k, v := range w.filters {
		if v == nil { // skip empty regex
			continue
		}
		if c, ok := info.Config.Filter.Channels[k]; ok {
			if _, ok := c[bot.DiscordChannel(m.ChannelID)]; ok {
				continue // This channel is excluded from this filter so skip it
			}
		}
		if v.MatchString(m.Content) {
			ch, _ := info.Bot.DG.State.Channel(m.ChannelID)

			if len(info.Config.Filter.Pressure) > 0 && w.spam != nil {
				if p, ok := info.Config.Filter.Pressure[k]; ok && p > 0.0 {
					w.spam.AddPressure(info, m, w.spam.TrackUser(author, bot.GetTimestamp(m)), p, "triggering the "+k+" filter")
				}
			}

			if s, _ := info.Config.Filter.Responses[k]; len(s) != 1 || s[0] != '!' {
				time.Sleep(bot.DelayTime)
				info.ChannelMessageDelete(ch, m.ID)

				if len(s) > 0 && bot.RateLimit(&w.lastmsg, 5, timestamp.Unix()) {
					info.SendMessage(bot.DiscordChannel(m.ChannelID), s)
				}
			}
			return true
		}
	}
	return false
}

// OnMessageCreate discord hook
func (w *FilterModule) OnMessageCreate(info *bot.GuildInfo, m *discordgo.Message) {
	w.matchFilter(info, m)
}

// OnMessageUpdate discord hook
func (w *FilterModule) OnMessageUpdate(info *bot.GuildInfo, m *discordgo.Message) {
	w.matchFilter(info, m)
}

// OnCommand discord hook
func (w *FilterModule) OnCommand(info *bot.GuildInfo, m *discordgo.Message) bool {
	return w.matchFilter(info, m)
}

var templateregex = regexp.MustCompile("%%")

// UpdateRegex updates all filter regexes
func (w *FilterModule) UpdateRegex(filter string, info *bot.GuildInfo) (err error) {
	combine := ""
	w.filters[filter] = nil
	if len(info.Config.Filter.Filters[filter]) > 0 {
		combine = "(" + strings.Join(bot.MapToSlice(info.Config.Filter.Filters[filter]), "|") + ")"
	}
	if len(info.Config.Filter.Templates[filter]) > 0 {
		combine = templateregex.ReplaceAllLiteralString(info.Config.Filter.Templates[filter], combine)
	}
	if len(combine) > 0 {
		w.filters[filter], err = regexp.Compile(combine)
	}
	return
}

func getAllFilters(info *bot.GuildInfo) []string {
	filters := []string{}
	for k := range info.Config.Filter.Filters {
		filters = append(filters, k)
	}
	return filters
}

type setFilterCommand struct{}

func (c *setFilterCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "SetFilter",
		Usage:     "Creates a new filter or sets the response and excluded channel list for an existing filter.",
		Sensitive: true,
	}
}
func (c *setFilterCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nNo filter given. All filters: " + strings.Join(getAllFilters(info), ", ") + "```", false, nil
	}

	filter := args[0]
	add := "%s response has been set and channels excluded."
	info.ConfigLock.Lock()
	m, _ := info.Config.Filter.Filters[filter]
	if len(m) == 0 {
		if len(info.Config.Filter.Filters) == 0 {
			info.Config.Filter.Filters = make(map[string]map[string]bool)
		}
		info.Config.Filter.Filters[filter] = make(map[string]bool)
		add = "%s has been created, the response set, and excluded channels configured."
	}
	if len(args) > 1 {
		if len(info.Config.Filter.Responses) == 0 {
			info.Config.Filter.Responses = make(map[string]string)
		}
		info.Config.Filter.Responses[filter] = args[1]
	}
	if len(info.Config.Filter.Channels) == 0 {
		info.Config.Filter.Channels = make(map[string]map[bot.DiscordChannel]bool)
	}
	info.Config.Filter.Channels[filter] = make(map[bot.DiscordChannel]bool)
	g, _ := info.GetGuild()
	for i := 2; i < len(args); i++ {
		if ch, err := bot.ParseChannel(args[i], g); err == nil {
			info.Config.Filter.Channels[filter][ch] = true
		}
	}
	info.ConfigLock.Unlock()

	info.SaveConfig()
	return fmt.Sprintf("```\n"+add+"```", info.Sanitize(filter, bot.CleanCodeBlock)), false, nil
}
func (c *setFilterCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Sets the [filter] response to [response] and it's excluded channel list to [channels]. Creates the filter if it doesn't exist.",
		Params: []bot.CommandUsageParam{
			{Name: "filter", Desc: "The name of a filter.", Optional: false},
			{Name: "response", Desc: "The message that will be sent when a message is deleted. Can be left blank, but quotes are mandatory.", Optional: true},
			{Name: "channels", Desc: "All additional arguments should be channels to exclude the filter from.", Optional: true},
		},
	}
}

type addFilterCommand struct {
	m *FilterModule
}

func (c *addFilterCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "AddFilter",
		Usage:     "Adds a string to a filter.",
		Sensitive: true,
	}
}

func (c *addFilterCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nNo filter given. All filters: " + strings.Join(getAllFilters(info), ", ") + "```", false, nil
	}
	if len(args) < 2 {
		return "```\nCan't add empty string!```", false, nil
	}

	info.ConfigLock.Lock()
	filter := args[0]
	m, ok := info.Config.Filter.Filters[filter]
	if !ok {
		info.ConfigLock.Unlock()
		return fmt.Sprintf("```\nThe %s filter does not exist!```", filter), false, nil
	}
	if len(m) == 0 {
		info.Config.Filter.Filters[filter] = make(map[string]bool)
	}

	add := "Added %s to %s."
	arg := msg.Content[indices[1]:]
	info.Config.Filter.Filters[filter][arg] = true
	if c.m.UpdateRegex(filter, info) != nil {
		delete(info.Config.Filter.Filters[filter], arg)
		add = "Failed to add %s to %s because regex compilation failed."
		c.m.UpdateRegex(filter, info)
	}
	info.ConfigLock.Unlock()

	info.SaveConfig()
	filter = info.Sanitize(filter, bot.CleanCodeBlock)
	return fmt.Sprintf("```\n"+add+" Length of %s: %v```", info.Sanitize(arg, bot.CleanCodeBlock), filter, filter, strconv.Itoa(len(info.Config.Filter.Filters[filter]))), false, nil
}
func (c *addFilterCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Adds [arbitrary string] to [filter] and recompiles the filter regex.",
		Params: []bot.CommandUsageParam{
			{Name: "filter", Desc: "The name of a filter. The filter must exist. Create a new filter by using !setfilter.", Optional: false},
			{Name: "arbitrary string", Desc: "Arbitrary string to add to the filter. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}

type removeFilterCommand struct {
	m *FilterModule
}

func (c *removeFilterCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "RemoveFilter",
		Usage:     "Removes a term from a filter.",
		Sensitive: true,
	}
}
func (c *removeFilterCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nNo filter given. All filters: " + strings.Join(getAllFilters(info), ", ") + "```", false, nil
	}
	if len(args) < 2 {
		return "```\nCan't remove an empty string!```", false, nil
	}

	filter := args[0]
	cmap, ok := info.Config.Filter.Filters[filter]
	if !ok {
		return "```\nThat filter does not exist!```", false, nil
	}

	arg := msg.Content[indices[1]:]
	_, ok = cmap[arg]
	if !ok {
		return "```\nCould not find " + arg + "!```", false, nil
	}
	delete(info.Config.Filter.Filters[filter], arg)
	c.m.UpdateRegex(filter, info)

	filter = info.Sanitize(filter, bot.CleanCodeBlock)
	retval := fmt.Sprintf("```\nRemoved %s from %s. Length of %s: %v```", info.Sanitize(arg, bot.CleanCodeBlock), filter, filter, len(info.Config.Filter.Filters[filter]))
	info.SaveConfig()
	return retval, false, nil
}
func (c *removeFilterCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Removes [arbitrary string] from [filter], then recompiles the regex.",
		Params: []bot.CommandUsageParam{
			{Name: "filter", Desc: "The name of a filter. The filter must exist. Create a new filter by using !setfilter.", Optional: false},
			{Name: "arbitrary string", Desc: "Arbitrary string to remove from the filter. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}

type deleteFilterCommand struct {
	m *FilterModule
}

func (c *deleteFilterCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "DeleteFilter",
		Usage:     "Deletes an entire filter.",
		Sensitive: true,
	}
}
func (c *deleteFilterCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nNo filter given. All filters: " + strings.Join(getAllFilters(info), ", ") + "```", false, nil
	}
	if len(args) > 1 {
		return "```\nYou specified more than one argument. This command completely removes an entire filter, use " + info.Config.Basic.CommandPrefix + "removefilter to remove a single item.```", false, nil
	}

	filter := args[0]
	_, ok := info.Config.Filter.Filters[filter]
	if !ok {
		return "```\nThat filter does not exist!```", false, nil
	}

	delete(info.Config.Filter.Filters, filter)
	delete(info.Config.Filter.Channels, filter)
	delete(info.Config.Filter.Responses, filter)
	delete(info.Config.Filter.Templates, filter)
	delete(c.m.filters, filter)
	c.m.UpdateRegex(filter, info)

	filter = info.Sanitize(filter, bot.CleanCodeBlock)
	info.SaveConfig()
	return fmt.Sprintf("```\nDeleted the %s filter```", filter), false, nil
}
func (c *deleteFilterCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Deletes a filter and all of its settings.",
		Params: []bot.CommandUsageParam{
			{Name: "filter", Desc: "The name of a filter. The filter must exist. Create a new filter by using !setfilter.", Optional: false},
		},
	}
}

type searchFilterCommand struct {
}

func (c *searchFilterCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "SearchFilter",
		Usage:     "Searches a filter.",
		Sensitive: true,
	}
}
func (c *searchFilterCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nNo filter given. All filters: " + strings.Join(getAllFilters(info), ", ") + "```", false, nil
	}

	filter := strings.ToLower(args[0])
	cmap, ok := info.Config.Filter.Filters[filter]
	if !ok {
		return "```\nThat filter doesn't exist!```", false, nil
	}
	results := []string{}
	if len(args) < 2 {
		results = bot.MapToSlice(cmap)
	} else {
		arg := msg.Content[indices[1]:]
		for k := range cmap {
			if strings.Contains(k, arg) {
				results = append(results, k)
			}
		}
	}

	if len(results) > 0 {
		return "```\nThe following entries match your query:\n" + info.Sanitize(strings.Join(results, "\n"), bot.CleanCodeBlock) + "```", len(results) > 6, nil
	}
	return "```\nNo results found in the " + filter + " filter.```", false, nil
}
func (c *searchFilterCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Returns all terms of the given filter that contain the given string.",
		Params: []bot.CommandUsageParam{
			{Name: "filter", Desc: "The name of the filter. The filter must exist. Create a new filter by using !setfilter.", Optional: false},
			{Name: "arbitrary string", Desc: "Arbitrary string to add to filter. If not provided, will simply return entire contents of the filter.", Optional: true},
		},
	}
}
