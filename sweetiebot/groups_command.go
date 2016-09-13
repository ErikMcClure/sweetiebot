package sweetiebot

import (
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type AddGroupCommand struct {
}

func (c *AddGroupCommand) Name() string {
	return "AddGroup"
}

var nameargregex = regexp.MustCompile("[a-zA-Z0-9]+")

func (c *AddGroupCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You have to name the group!```", false
	}
	arg := strings.ToLower(args[0])
	if !nameargregex.MatchString(arg) {
		return "```A group name must be alphanumeric, no special characters.```", false
	}
	_, ok := info.config.Groups[arg]
	if ok {
		return "```That group already exists!```", false
	}

	if len(info.config.Groups) <= 0 {
		info.config.Groups = make(map[string]map[string]bool)
	}
	group := make(map[string]bool)
	group[msg.Author.ID] = true
	info.config.Groups[arg] = group
	info.SaveConfig()

	return "```Successfully created the " + arg + " group! Join it using !joingroup " + arg + " and ping it using !ping " + arg + "```", false
}
func (c *AddGroupCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[name]", "Creates a new group and automatically adds you to it. Groups are automatically destroyed when everyone in the group leaves.")
}
func (c *AddGroupCommand) UsageShort() string { return "Creates a new group." }

type JoinGroupCommand struct {
}

func (c *JoinGroupCommand) Name() string {
	return "JoinGroup"
}

func (c *JoinGroupCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You have to provide a group name!```", false
	}
	arg := strings.ToLower(args[0])
	_, ok := info.config.Groups[arg]
	if !ok {
		return "```That group doesn't exist! Use !listgroup to list existing groups.```", false
	}

	info.config.Groups[arg][msg.Author.ID] = true
	info.SaveConfig()

	return "```Successfully joined the " + arg + " group! Ping it using !ping " + arg + " or leave it using !leavegroup " + arg + "```", false
}
func (c *JoinGroupCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[group]", "Joins an existing group.")
}
func (c *JoinGroupCommand) UsageShort() string { return "Joins an existing group." }

type ListGroupCommand struct {
}

func (c *ListGroupCommand) Name() string {
	return "ListGroup"
}

func (c *ListGroupCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		if len(info.config.Groups) <= 0 {
			return "```No groups to list!```", false
		}
		keys := make([]string, len(info.config.Groups))

		i := 0
		for k := range info.config.Groups {
			keys[i] = k
			i++
		}

		return "```" + strings.Join(keys, ", ") + "```", false
	}

	arg := strings.ToLower(args[0])
	_, ok := info.config.Groups[arg]
	if !ok {
		return "```That group doesn't exist! Use !listgroup with no arguments to list existing groups.```", false
	}

	pings := make([]string, len(info.config.Groups[arg]))

	i := 0
	for k := range info.config.Groups[arg] {
		m, _ := sb.db.GetUser(SBatoi(k))
		if m != nil {
			pings[i] = m.Username
		}
		i++
	}

	return "```" + strings.Join(pings, ", ") + "```", false
}
func (c *ListGroupCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[group]", "If no argument is given, lists all the current groups. If a group name is given, lists all the members of that group.")
}
func (c *ListGroupCommand) UsageShort() string { return "Lists all groups." }

type LeaveGroupCommand struct {
}

func (c *LeaveGroupCommand) Name() string {
	return "LeaveGroup"
}

func (c *LeaveGroupCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You have to provide a group name!```", false
	}
	arg := strings.ToLower(args[0])
	_, ok := info.config.Groups[arg]
	if !ok {
		return "```That group doesn't exist! Use !listgroup to list existing groups.```", false
	}

	_, ok = info.config.Groups[arg][msg.Author.ID]
	if !ok {
		return "```You aren't in that group!```", false
	}

	delete(info.config.Groups[arg], msg.Author.ID)

	if len(info.config.Groups[arg]) <= 0 {
		delete(info.config.Groups, arg)
	}

	info.SaveConfig()

	return "```You have been removed from " + arg + "```", false
}
func (c *LeaveGroupCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[group]", "Removes you from the given group, if you are a member of it.")
}
func (c *LeaveGroupCommand) UsageShort() string { return "Removes you from a group." }

func getGroupPings(group string, info *GuildInfo) string {
	pings := make([]string, len(info.config.Groups[group]))

	i := 0
	for k := range info.config.Groups[group] {
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

func (c *PingCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You have to provide a group name!```", false
	}
	arg := strings.ToLower(args[0])
	_, ok := info.config.Groups[arg]
	if !ok {
		return "```That group doesn't exist! Use !listgroup to list existing groups.```", false
	}

	_, ok = info.config.Groups[arg][msg.Author.ID]
	if !ok {
		return "```You can only ping groups you are a member of.```", false
	}

	sb.dg.ChannelMessageSend(msg.ChannelID, arg+": "+getGroupPings(arg, info)+" "+info.SanitizeOutput(strings.Join(args[1:], " ")))
	return "", false
}
func (c *PingCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[group] [arbitrary string]", "Pings everyone in a group with the given message, but only if you are a member of the group.")
}
func (c *PingCommand) UsageShort() string { return "Pings a group." }

type PurgeGroupCommand struct {
}

func (c *PurgeGroupCommand) Name() string {
	return "PurgeGroup"
}

func (c *PurgeGroupCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You have to provide a group name!```", false
	}
	arg := strings.ToLower(args[0])
	_, ok := info.config.Groups[arg]
	if !ok {
		return "```That group doesn't exist! Use !listgroup to list existing groups.```", false
	}

	delete(info.config.Groups, arg)
	info.SaveConfig()

	return "```Deleted " + arg + "```", false
}
func (c *PurgeGroupCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[group]", "Deletes the group, if it exists.")
}
func (c *PurgeGroupCommand) UsageShort() string { return "Deletes a group." }
