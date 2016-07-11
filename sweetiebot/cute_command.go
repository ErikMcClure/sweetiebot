package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
)

type CuteCommand struct {
}

func (c *CuteCommand) Name() string {
  return "Cute";  
}
func (c *CuteCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
  if len(info.config.Collections["cute"]) > 0 {
    return MapGetRandomItem(info.config.Collections["cute"]), false
  }
  return "", false
}
func (c *CuteCommand) Usage(info *GuildInfo) string { 
  return info.FormatUsage(c, "", "Returns a cute pony picture.") 
}
func (c *CuteCommand) UsageShort() string { return "Returns a cute pony picture." }