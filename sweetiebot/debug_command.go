package sweetiebot

import (
  "strings"
)

type EchoCommand struct {
}

func (c *EchoCommand) Name() string {
  return "Echo";  
}
func (c *EchoCommand) Process(args []string) string {
  if len(args) == 0 {
    return "```You have to tell me to say something, silly!```"
  }
  return "```" + strings.Join(args, " ") + "```";
}
func (c *EchoCommand) Usage() string { 
  return "[#channel] [string]\n+" + strings.Join(c.Roles(), ", +") + "\n\nMakes Sweetie Bot say the given sentence in #channel, or in the current channel if no argument is provided." 
}
func (c *EchoCommand) UsageShort() string { return "Makes Sweetie Bot say something in the given channel." }
func (c *EchoCommand) Roles() []string { return []string{"Princesses"} }