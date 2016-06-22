package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "strconv"
)

type AddCommand struct {
  funcmap map[string]func(string)string
}

func (c *AddCommand) Name() string {
  return "Add";  
}
func (c *AddCommand) Process(args []string, msg *discordgo.Message) (string, bool) {   
  if len(args) < 1 {
    return "```No collection given```", false
  }
  if len(args) < 2 {
    return "```Can't add empty string!```", false
  }
  
  collection := args[0]
  _, ok := sb.config.Collections[collection];
  if !ok {
    return "```That collection does not exist!```", false
  }

  arg := strings.Join(args[1:], " ")
  sb.config.Collections[collection][arg] = true
  fn, ok := c.funcmap[collection];
  retval := "```Added " + arg + " to " + collection + ". Length of " + collection + ": " + strconv.Itoa(len(sb.config.Collections[collection])) + "```"
  if ok {
    retval = fn(arg)
  }
  sb.SaveConfig()
  return ExtraSanitize(retval), false
}
func (c *AddCommand) Usage() string { 
  return FormatUsage(c, "[collection] [arbitrary string]", "Adds [arbitrary string] to [collection] (no quotes are required), then calls a handler function for that specific collection.") 
}
func (c *AddCommand) UsageShort() string { return "Adds a line to a collection." }
func (c *AddCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *AddCommand) Channels() []string { return []string{} }

type RemoveCommand struct {
  funcmap map[string]func(string)string
}

func (c *RemoveCommand) Name() string {
  return "Remove";  
}
func (c *RemoveCommand) Process(args []string, msg *discordgo.Message) (string, bool) {  
  if len(args) < 1 {
    return "```No collection given```", false
  }
  if len(args) < 2 {
    return "```Can't remove an empty string!```", false
  }

  collection := args[0]
  cmap, ok := sb.config.Collections[collection];
  if !ok {
    return "```That collection does not exist!```", false
  }
  
  arg := strings.Join(args[1:], " ")
  _, ok = cmap[arg]
  if !ok {
    return "```Could not find " + arg + "!```", false
  }
  delete(sb.config.Collections[collection], arg)
  fn, ok := c.funcmap[collection];
  retval := "```Removed " + arg + " from " + collection + ". Length of " + collection + ": " + strconv.Itoa(len(sb.config.Collections[collection])) + "```"
  if ok {
    retval = fn(arg)
  }

  sb.SaveConfig()
  return ExtraSanitize(retval), false
}
func (c *RemoveCommand) Usage() string { 
  return FormatUsage(c, "[collection] [arbitrary string]", "Removes [arbitrary string] from [collection] (no quotes are required) and calls a handler function for that collection.") 
}
func (c *RemoveCommand) UsageShort() string { return "Removes a line from a collection." }
func (c *RemoveCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *RemoveCommand) Channels() []string { return []string{} }


type CollectionsCommand struct {
}

func (c *CollectionsCommand) Name() string {
  return "Collections";  
}
func (c *CollectionsCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    s := make([]string, 0, len(sb.config.Collections))
    for k, _ := range sb.config.Collections {
      s = append(s, k)
    }
    
    return "```No collection specified. All collections:\n" + ExtraSanitize(strings.Join(s, "\n")) + "```", false
  }

  arg := args[0]
  cmap, ok := sb.config.Collections[arg];
  if !ok {
    return "```That collection doesn't exist! Use this command with no arguments to see a list of all collections.```", false
  }

  return "```" + ExtraSanitize(arg + " contains:\n" + strings.Join(MapToSlice(cmap), "\n")) + "```", false
}
func (c *CollectionsCommand) Usage() string { 
  return FormatUsage(c, "", "Lists all the collections that sweetiebot is using.") 
}
func (c *CollectionsCommand) UsageShort() string { return "Lists all collections." }
func (c *CollectionsCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *CollectionsCommand) Channels() []string { return []string{} }