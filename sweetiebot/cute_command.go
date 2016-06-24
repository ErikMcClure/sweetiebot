package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
)

type CuteCommand struct {
  lastcute int64;
}

func (c *CuteCommand) Name() string {
  return "Cute";  
}
func (c *CuteCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(sb.config.Collections["cute"]) > 0 && RateLimit(&c.lastcute, sb.config.Maxcute) {
    return MapGetRandomItem(sb.config.Collections["cute"]), false
  }
  return "", false
}
func (c *CuteCommand) Usage() string { 
  return FormatUsage(c, "", "Returns a cute pony picture.") 
}
func (c *CuteCommand) UsageShort() string { return "Returns a cute pony picture." }