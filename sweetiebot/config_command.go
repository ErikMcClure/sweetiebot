package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "encoding/json"
  "strings"
  "reflect"
)

type SetConfigCommand struct {
}

func (c *SetConfigCommand) Name() string {
  return "SetConfig";  
}
func (c *SetConfigCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    return "```No configuration parameter to look for!```", false
  }
  if len(args) < 2 {
    return "```No value to set!```", false
  }
  n, ok := sb.SetConfig(args[0], args[1], args[2:]...)
  if ok {
    return "```Successfully set " + args[0] + " to " + n + ".```", false
  }
  return "```" + n + "```", false
}
func (c *SetConfigCommand) Usage() string { 
  return FormatUsage(c, "[config parameter] [value]", "Attempts to set the configuration value matching [config parameter] (not case-sensitive) to [value]. Will only save the new configuration if it succeeds, and returns the new value upon success.") 
}
func (c *SetConfigCommand) UsageShort() string { return "Sets a config value and saves the new configuration." }

type GetConfigCommand struct {
}

func (c *GetConfigCommand) Name() string {
  return "GetConfig";  
}
func (c *GetConfigCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  t := reflect.ValueOf(&sb.config).Elem()
  n := t.NumField()
  if len(args) < 1 {
    s := make([]string, 0, n)
    for i := 0; i < n; i++ {
      s = append(s, strings.ToLower(t.Type().Field(i).Name))
    }
    return "```Choose a config option to display:\n" + strings.Join(s, "\n") + "```", false
  }
  arg := args[0]
  for i := 0; i < n; i++ {
    if strings.ToLower(t.Type().Field(i).Name) == arg {
      data, err := json.Marshal(t.Field(i).Interface())
      s := string(data);
      s = strings.Replace(s,"`","",-1)
      s = strings.Replace(s, "[](/", "[\u200B](/", -1)
      s = strings.Replace(s, "http://", "http\u200B://", -1)
      s = strings.Replace(s, "https://", "https\u200B://", -1)
      if err == nil {
        return "```" + s + "```", false
      }
      sb.log.Log("JSON error: ", err.Error())
      return "```Failed to marshal JSON :C```", false
    }
  }
  
  return "```That's not a recognized config option! Type !getconfig without any arguments to list all possible config options```", false
}
func (c *GetConfigCommand) Usage() string { 
  return FormatUsage(c, "", "Returns the current configuration as a JSON string.") 
}
func (c *GetConfigCommand) UsageShort() string { return "Returns the current configuration." }