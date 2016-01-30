package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "encoding/json"
)

type SetConfigCommand struct {
}

func (c *SetConfigCommand) Name() string {
  return "SetConfig";  
}
func (c *SetConfigCommand) Process(args []string, user *discordgo.User) (string, bool) {
  if len(args) < 1 {
    return "```No configuration parameter to look for!```", false
  }
  if len(args) < 2 {
    return "```No value to set!```", false
  }
  n, ok := sb.SetConfig(args[0], args[1])
  if ok {
    return "```Successfully set " + args[0] + " to " + n + ".```", false
  }
  return "```Could not find configuration parameter " + args[0] + "!```", false
}
func (c *SetConfigCommand) Usage() string { 
  return FormatUsage(c, "[config parameter] [value]", "Attempts to set the configuration value matching [config parameter] (not case-sensitive) to [value]. Will only save the new configuration if it succeeds, and returns the new value upon success.") 
}
func (c *SetConfigCommand) UsageShort() string { return "Sets a config value and saves the new configuration." }
func (c *SetConfigCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }

type GetConfigCommand struct {
}

func (c *GetConfigCommand) Name() string {
  return "GetConfig";  
}
func (c *GetConfigCommand) Process(args []string, user *discordgo.User) (string, bool) {
  data, err := json.Marshal(sb.config)
  if err == nil {
    return "```" + string(data) + "```", false
  }
  sb.log.Log("JSON error: ", err.Error())
  return "```Failed to marshal JSON :C```", false
}
func (c *GetConfigCommand) Usage() string { 
  return FormatUsage(c, "", "Returns the current configuration as a JSON string.") 
}
func (c *GetConfigCommand) UsageShort() string { return "Returns the current configuration." }
func (c *GetConfigCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }