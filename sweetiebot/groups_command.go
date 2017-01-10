package sweetiebot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type GroupsModule struct {
}

func (w *GroupsModule) Name() string {
	return "Groups"
}

func (w *GroupsModule) Register(info *GuildInfo) {}

func (w *GroupsModule) Commands() []Command {
	return []Command{
		&AddGroupCommand{},
		&JoinGroupCommand{},
		&ListGroupCommand{},
		&LeaveGroupCommand{},
		&PingCommand{},
		&PurgeGroupCommand{},
	}
}

func (w *GroupsModule) Description() string {
	return "Contains commands for manipulating groups and pinging them."
}

type AddGroupCommand struct {
}

func (c *AddGroupCommand) Name() string {
	return "AddGroup"
}

func (c *AddGroupCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You have to name the group!```", false, nil
	}
	arg := strings.TrimSpace(strings.ToLower(args[0]))
	_, ok := info.config.Basic.Groups[arg]
	if ok {
		return "```That group already exists!```", false, nil
	}

	if len(info.config.Basic.Groups) <= 0 {
		info.config.Basic.Groups = make(map[string]map[string]bool)
	}
	group := make(map[string]bool)
	group[msg.Author.ID] = true
	info.config.Basic.Groups[arg] = group
	info.SaveConfig()

	return "```Successfully created the " + arg + " group! Join it using !joingroup " + arg + " and ping it using !ping " + arg + ".```", false, nil
}
func (c *AddGroupCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Creates a new group and automatically adds you to it. Groups are automatically destroyed when everyone in the group leaves.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "name", Desc: "Name of the new group. Should not contain spaces or anything other than letters and numbers.", Optional: false},
		},
	}
}
func (c *AddGroupCommand) UsageShort() string { return "Creates a new group." }

type JoinGroupCommand struct {
}

func (c *JoinGroupCommand) Name() string {
	return "JoinGroup"
}

func (c *JoinGroupCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You have to provide a group name!```", false, nil
	}
	arg := strings.TrimSpace(strings.ToLower(args[0]))
	_, ok := info.config.Basic.Groups[arg]
	if !ok {
		return "```That group doesn't exist! Use !listgroup to list existing groups.```", false, nil
	}

	info.config.Basic.Groups[arg][msg.Author.ID] = true
	info.SaveConfig()

	return "```Successfully joined the " + arg + " group! Ping it using !ping " + arg + " or leave it using !leavegroup " + arg + ". WARNING: Pinging a group will ping EVERYONE IN THE GROUP.```", false, nil
}
func (c *JoinGroupCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Joins an existing group.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "group", Desc: "Name of the group to join (case-insensitive).", Optional: false},
		},
	}
}
func (c *JoinGroupCommand) UsageShort() string { return "Joins an existing group." }

type ListGroupCommand struct {
}

func (c *ListGroupCommand) Name() string {
	return "ListGroup"
}

func (c *ListGroupCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		if len(info.config.Basic.Groups) <= 0 {
			return "```No groups to list!```", false, nil
		}
		keys := make([]string, len(info.config.Basic.Groups))

		i := 0
		for k := range info.config.Basic.Groups {
			keys[i] = k
			i++
		}

		return "```\n" + strings.Join(keys, ", ") + "```", false, nil
	}

	arg := strings.TrimSpace(strings.ToLower(args[0]))
	_, ok := info.config.Basic.Groups[arg]
	if !ok {
		return "```That group doesn't exist! Use !listgroup with no arguments to list existing groups.```", false, nil
	}

	pings := make([]string, len(info.config.Basic.Groups[arg]))

	i := 0
	for k := range info.config.Basic.Groups[arg] {
		m, _, _, _ := sb.db.GetUser(SBatoi(k))
		if m != nil {
			pings[i] = m.Username
		}
		i++
	}

	return "```\n" + strings.Join(pings, ", ") + "```", false, nil
}
func (c *ListGroupCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists all current groups, or lists all the members of a group.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "group", Desc: "Name of the group to display. If omitted, will display all groups instead.", Optional: true},
		},
	}
}
func (c *ListGroupCommand) UsageShort() string { return "Lists all groups." }

type LeaveGroupCommand struct {
}

func (c *LeaveGroupCommand) Name() string {
	return "LeaveGroup"
}

func (c *LeaveGroupCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You have to provide a group name!```", false, nil
	}
	arg := strings.TrimSpace(strings.ToLower(args[0]))
	_, ok := info.config.Basic.Groups[arg]
	if !ok {
		return "```That group doesn't exist! Use !listgroup to list existing groups.```", false, nil
	}

	_, ok = info.config.Basic.Groups[arg][msg.Author.ID]
	if !ok {
		return "```You aren't in that group!```", false, nil
	}

	delete(info.config.Basic.Groups[arg], msg.Author.ID)

	if len(info.config.Basic.Groups[arg]) <= 0 {
		delete(info.config.Basic.Groups, arg)
	}

	info.SaveConfig()

	return "```You have been removed from " + arg + "```", false, nil
}
func (c *LeaveGroupCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Removes you from the given group, if you are a member of it.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "group", Desc: "Name of the group to leave.", Optional: false},
		},
	}
}
func (c *LeaveGroupCommand) UsageShort() string { return "Removes you from a group." }

func getGroupPings(groups []string, info *GuildInfo) string {
	if len(groups) == 0 {
		return ""
	}
	union := make(map[string]bool)
	for _, group := range groups {
		for k, v := range info.config.Basic.Groups[group] {
			union[k] = v
		}
	}
	pings := make([]string, len(union), len(union))

	i := 0
	for k := range union {
		pings[i] = SBitoa(SBatoi(k)) // We convert to integers and then back to strings to prevent bloons from fucking with the bot
		i++
	}

	return "<@" + strings.Join(pings, "> <@") + ">"
}

type PingCommand struct {
}

func (c *PingCommand) Name() string {
	return "Ping"
}

func (c *PingCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You have to provide a group name!```", false, nil
	}
	nargs := strings.SplitN(args[0], "\n", 2)
	args = append(nargs, args[1:]...)
	arg := strings.TrimSpace(strings.ToLower(args[0]))
	_, ok := info.config.Basic.Groups[arg]
	m := ""
	if len(indices) > 1 {
		m = msg.Content[indices[1]:]
	}

	if !ok {
		groups := strings.Split(arg, "+")
		for _, v := range groups {
			_, ok = info.config.Basic.Groups[v]
			if !ok {
				return fmt.Sprintf("```The %s group doesn't exist! Use !listgroup to list existing groups.```", v), false, nil
			}
			_, ok = info.config.Basic.Groups[v][msg.Author.ID]
			if !ok {
				return fmt.Sprintf("```You aren't a member of %s. You can only ping groups you are a member of.```", v), false, nil
			}
		}
		sb.dg.ChannelMessageSend(msg.ChannelID, arg+": "+getGroupPings(groups, info)+" "+info.SanitizeOutput(m))

	} else {
		_, ok = info.config.Basic.Groups[arg][msg.Author.ID]
		if !ok {
			return "```You can only ping groups you are a member of.```", false, nil
		}
		sb.dg.ChannelMessageSend(msg.ChannelID, arg+": "+getGroupPings([]string{arg}, info)+" "+info.SanitizeOutput(m))
	}
	return "", false, nil
}
func (c *PingCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Pings everyone in a group with the given message, but only if you are a member of the group.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "group", Desc: "Name of the group to ping. You can ping multiple groups at the same time by using `group1+group2`", Optional: false},
			CommandUsageParam{Name: "arbitrary string", Desc: "String for Sweetiebot to echo to the group, no spaces required.", Optional: false},
		},
	}
}
func (c *PingCommand) UsageShort() string { return "Pings a group." }

type PurgeGroupCommand struct {
}

func (c *PurgeGroupCommand) Name() string {
	return "PurgeGroup"
}

func (c *PurgeGroupCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You have to provide a group name!```", false, nil
	}
	arg := strings.TrimSpace(strings.ToLower(args[0]))
	_, ok := info.config.Basic.Groups[arg]
	if !ok {
		return "```That group doesn't exist! Use !listgroup to list existing groups.```", false, nil
	}

	delete(info.config.Basic.Groups, arg)
	info.SaveConfig()

	return "```Deleted " + arg + "```", false, nil
}
func (c *PurgeGroupCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Deletes the group, if it exists.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "group", Desc: "Name of the group to delete.", Optional: false},
		},
	}
}
func (c *PurgeGroupCommand) UsageShort() string { return "Deletes a group." }
