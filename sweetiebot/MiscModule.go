package sweetiebot

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blackhole12/discordgo"
)

type MiscModule struct {
	emotes  *EmoteModule
	spoiler *SpoilerModule
}

// Name of the module
func (w *MiscModule) Name() string {
	return "Miscellaneous"
}

// Commands in the module
func (w *MiscModule) Commands() []Command {
	return []Command{
		&LastSeenCommand{},
		&searchCommand{emotes: w.emotes, statements: make(map[string][]*sql.Stmt)},
		&rollCommand{},
		&SnowflakeTimeCommand{},
		&addSetCommand{w},
		&removeSetCommand{w},
		&searchSetCommand{},
	}
}

// Description of the module
func (w *MiscModule) Description() string {
	return "A collection of miscellaneous commands that don't belong to a module."
}

type LastSeenCommand struct {
}

func (c *LastSeenCommand) Name() string {
	return "LastSeen"
}
func (c *LastSeenCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.DB.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(indices) < 1 {
		return "```You have to give me someone to look for!```", false, nil
	}
	arg := msg.Content[indices[0]:]
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false, nil
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info, true), "\n") + "```", len(IDs) > 5, nil
	}

	u, lastseen, _ := sb.DB.GetMember(IDs[0], SBatoi(info.ID))
	if u == nil {
		return "```Error: User does not exist!```", false, nil
	}
	nick := u.User.Username
	if len(u.Nick) > 0 {
		nick = u.Nick
	}
	return "```" + nick + " last seen " + TimeDiff(SinceUTC(lastseen)) + " ago.```", false, nil
}
func (c *LastSeenCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Returns when a user was last seen on discord, which is usually their last status change.",
		Params: []CommandUsageParam{
			{Name: "@user", Desc: "Either a ping for the user, their username, or their nickname.", Optional: false},
		},
	}
}
func (c *LastSeenCommand) UsageShort() string { return "Returns when a user was last seen." }

type SnowflakeTimeCommand struct {
}

func (c *SnowflakeTimeCommand) Name() string {
	return "SnowflakeTime"
}
func (c *SnowflakeTimeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You have to give me an ID!```", false, nil
	}
	ID := SBatoi(StripPing(args[0]))
	t := snowflakeTime(ID)
	tz := getTimezone(info, msg.Author)
	return t.In(tz).Format(time.RFC1123), false, nil
}
func (c *SnowflakeTimeCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Given a discord snowflake ID, returns when that ID was created.",
		Params: []CommandUsageParam{
			{Name: "ID", Desc: "Any unique ID used by discord (these are called snowflake IDs)", Optional: false},
		},
	}
}
func (c *SnowflakeTimeCommand) UsageShort() string { return "Returns when a snowflake ID was created." }

type addSetCommand struct {
	m *MiscModule
}

func GetAllSets(info *GuildInfo) []string {
	sets := []string{}
	for k := range info.config.Collections {
		sets = append(sets, k)
	}
	return sets
}

func (c *addSetCommand) Name() string {
	return "AddSet"
}
func (c *addSetCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```No set given. All sets: " + strings.Join(GetAllSets(info), ", ") + "```", false, nil
	}
	if len(args) < 2 {
		return "```Can't add empty string!```", false, nil
	}

	set := args[0]
	_, ok := info.config.Collections[set]
	if !ok {
		return fmt.Sprintf("```The %s set does not exist!```", set), false, nil
	}

	add := ""
	arg := msg.Content[indices[1]:]
	info.config.Collections[set][arg] = true

	switch set {
	case "emote":
		r := c.m.emotes.UpdateRegex(info)
		if !r {
			delete(info.config.Collections["emote"], arg)
			c.m.emotes.UpdateRegex(info)
			add = ". Failed to ban " + arg + " because regex compilation failed"
		}
		add = "and recompiled the emote regex"
	case "spoiler":
		r := c.m.spoiler.UpdateRegex(info)
		if !r {
			delete(info.config.Collections["spoiler"], arg)
			c.m.spoiler.UpdateRegex(info)
			add = ". Failed to ban " + arg + " because regex compilation failed"
		}
		add = "and recompiled the spoiler regex"
	}

	info.SaveConfig()
	return fmt.Sprintf("```Added %s to %s%s. Length of %s: %v```", PartialSanitize(arg), PartialSanitize(set), add, PartialSanitize(set), strconv.Itoa(len(info.config.Collections[set]))), false, nil
}
func (c *addSetCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Adds [arbitrary string] to [set], then calls a handler function for that specific set.",
		Params: []CommandUsageParam{
			{Name: "set", Desc: "The name of a set.", Optional: false},
			{Name: "arbitrary string", Desc: "Arbitrary string to add to set. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}
func (c *addSetCommand) UsageShort() string { return "Adds a line to a set." }

type removeSetCommand struct {
	m *MiscModule
}

func (c *removeSetCommand) Name() string {
	return "RemoveSet"
}
func (c *removeSetCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```No set given. All sets: " + strings.Join(GetAllSets(info), ", ") + "```", false, nil
	}
	if len(args) < 2 {
		return "```Can't remove an empty string!```", false, nil
	}

	set := args[0]
	cmap, ok := info.config.Collections[set]
	if !ok {
		return "```That set does not exist!```", false, nil
	}

	arg := msg.Content[indices[1]:]
	_, ok = cmap[arg]
	if !ok {
		return "```Could not find " + arg + "!```", false, nil
	}
	delete(info.config.Collections[set], arg)
	add := ""

	switch set {
	case "emote":
		c.m.emotes.UpdateRegex(info)
		add = "Unbanned " + arg + " and recompiled the emote regex"
	case "spoiler":
		c.m.spoiler.UpdateRegex(info)
		add = "Unbanned " + arg + " and recompiled the spoiler regex"
	}

	retval := fmt.Sprintf("```Removed %s from %s%s. Length of %s: %v```", PartialSanitize(arg), PartialSanitize(set), add, PartialSanitize(set), len(info.config.Collections[set]))
	info.SaveConfig()
	return retval, false, nil
}
func (c *removeSetCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Removes [arbitrary string] from [set], then calls a handler function for that specific set.",
		Params: []CommandUsageParam{
			{Name: "set", Desc: "The name of a set.", Optional: false},
			{Name: "arbitrary string", Desc: "Arbitrary string to remove from set. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}
func (c *removeSetCommand) UsageShort() string { return "Removes a line from an internal set." }

type searchSetCommand struct {
}

func (c *searchSetCommand) Name() string {
	return "SearchSet"
}
func (c *searchSetCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```No set given. All sets: " + strings.Join(GetAllSets(info), ", ") + "```", false, nil
	}

	set := strings.ToLower(args[0])
	cmap, ok := info.config.Collections[set]
	if !ok {
		return "```That set doesn't exist!```", false, nil
	}
	results := []string{}
	if len(args) < 2 {
		results = MapToSlice(cmap)
	} else {
		arg := msg.Content[indices[1]:]
		for k := range cmap {
			if strings.Contains(k, arg) {
				results = append(results, k)
			}
		}
	}

	if len(results) > 0 {
		return "```The following entries match your query:\n" + PartialSanitize(strings.Join(results, "\n")) + "```", len(results) > 6, nil
	}
	return "```No results found in the " + set + " set.```", false, nil
}
func (c *searchSetCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Returns all members of the given set that contain the given string.",
		Params: []CommandUsageParam{
			{Name: "set", Desc: "The name of the set.", Optional: false},
			{Name: "arbitrary string", Desc: "Arbitrary string to add to set. If not provided, will simply return entire contents of the set.", Optional: true},
		},
	}
}
func (c *searchSetCommand) UsageShort() string { return "Searches a set." }
