package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "strconv"
  "regexp"
)

type AddGroupCommand struct {
}

func (c *AddGroupCommand) Name() string {
  return "AddGroup";  
}

var nameargregex = regexp.MustCompile("[a-zA-Z0-9]+")

func (c *AddGroupCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    return "```You have to name the group!```", false
  }
  arg := strings.ToLower(args[0])
  if !nameargregex.MatchString(arg) {
    return "```A group name must be alphanumeric, no special characters.```", false
  }
  _, ok := sb.config.Groups[arg]
  if ok {
    return "```That group already exists!```", false
  }
  
  if len(sb.config.Groups) <= 0 {
    sb.config.Groups = make(map[string]map[string]bool)
  } 
  group := make(map[string]bool)
  group[msg.Author.ID] = true
  sb.config.Groups[arg] = group
  sb.SaveConfig()
  
  return "```Successfully created the " + arg + " group! Join it using !joingroup " + arg + " and ping it using !ping " + arg + "```", false
}
func (c *AddGroupCommand) Usage() string { 
  return FormatUsage(c, "[name]", "Creates a new group and automatically adds you to it. Groups are automatically destroyed when everyone in the group leaves.") 
}
func (c *AddGroupCommand) UsageShort() string { return "Creates a new group." }
func (c *AddGroupCommand) Roles() []string { return []string{} }
func (c *AddGroupCommand) Channels() []string { return []string{} }

type JoinGroupCommand struct {
}

func (c *JoinGroupCommand) Name() string {
  return "JoinGroup";  
}

func (c *JoinGroupCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    return "```You have to provide a group name!```", false
  }
  arg := strings.ToLower(args[0])
  _, ok := sb.config.Groups[arg]
  if !ok {
    return "```That group doesn't exist! Use !listgroups to list existing groups.```", false
  }
  
  sb.config.Groups[arg][msg.Author.ID] = true
  sb.SaveConfig()
  
  return "```Successfully joined the " + arg + " group! Ping it using !ping " + arg + " or leave it using !leavegroup " + arg + "```", false
}
func (c *JoinGroupCommand) Usage() string { 
  return FormatUsage(c, "[group]", "Joins an existing group.") 
}
func (c *JoinGroupCommand) UsageShort() string { return "Joins an existing group." }
func (c *JoinGroupCommand) Roles() []string { return []string{} }
func (c *JoinGroupCommand) Channels() []string { return []string{} }

type ListGroupsCommand struct {
}

func (c *ListGroupsCommand) Name() string {
  return "ListGroups";  
}

func (c *ListGroupsCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(sb.config.Groups) <= 0 {
    return "```No groups to list!```", false
  }
  keys := make([]string, len(sb.config.Groups))

  i := 0
  for k := range sb.config.Groups {
      keys[i] = k
      i++
  }
  
  return "```" + strings.Join(keys, ", ") + "```", false
}
func (c *ListGroupsCommand) Usage() string { 
  return FormatUsage(c, "", "Lists all groups in no particular order.") 
}
func (c *ListGroupsCommand) UsageShort() string { return "Lists all groups." }
func (c *ListGroupsCommand) Roles() []string { return []string{} }
func (c *ListGroupsCommand) Channels() []string { return []string{} }


type LeaveGroupCommand struct {
}

func (c *LeaveGroupCommand) Name() string {
  return "LeaveGroup";  
}

func (c *LeaveGroupCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    return "```You have to provide a group name!```", false
  }
  arg := strings.ToLower(args[0])
  _, ok := sb.config.Groups[arg]
  if !ok {
    return "```That group doesn't exist! Use !listgroups to list existing groups.```", false
  }
  
  _, ok = sb.config.Groups[arg][msg.Author.ID]
  if !ok {
    return "```You aren't in that group!```", false
  }
  
  delete(sb.config.Groups[arg], msg.Author.ID)
  
  if len(sb.config.Groups[arg]) <= 0 {
    delete(sb.config.Groups, arg)
  }
  
  sb.SaveConfig()
  
  return "```You have been removed from " + arg + "```", false
}
func (c *LeaveGroupCommand) Usage() string { 
  return FormatUsage(c, "[group]", "Removes you from the given group, if you are a member of it.") 
}
func (c *LeaveGroupCommand) UsageShort() string { return "Removes you from a group." }
func (c *LeaveGroupCommand) Roles() []string { return []string{} }
func (c *LeaveGroupCommand) Channels() []string { return []string{} }


type PingCommand struct {
}

func (c *PingCommand) Name() string {
  return "Ping";  
}

func (c *PingCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    return "```You have to provide a group name!```", false
  }
  arg := strings.ToLower(args[0])
  _, ok := sb.config.Groups[arg]
  if !ok {
    return "```That group doesn't exist! Use !listgroups to list existing groups.```", false
  }
  
  _, ok = sb.config.Groups[arg][msg.Author.ID]
  if !ok {
    return "```You can only ping groups you are a member of.```", false
  }
  
  pings := make([]string, len(sb.config.Groups[arg])) 

  i := 0
  for k := range sb.config.Groups[arg] {
      pings[i] = strconv.FormatUint(SBatoi(k), 10) // We convert to integers and then back to strings to prevent bloons from fucking with the bot
      i++
  }
  
  sb.dg.ChannelMessageSend(msg.ChannelID, "<@" + strings.Join(pings, "> <@") + "> " + SanitizeOutput(strings.Join(args[1:], " ")))
  return "", false;
}
func (c *PingCommand) Usage() string { 
  return FormatUsage(c, "[group] [arbitrary string]", "Pings everyone in a group with the given message, but only if you are a member of the group.") 
}
func (c *PingCommand) UsageShort() string { return "Pings a group." }
func (c *PingCommand) Roles() []string { return []string{} }
func (c *PingCommand) Channels() []string { return []string{} }