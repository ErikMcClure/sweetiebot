package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
)

type AddStatusCommand struct {
}

func (c *AddStatusCommand) Name() string {
  return "AddStatus";  
}
func (c *AddStatusCommand) Process(args []string, msg *discordgo.Message) (string, bool) {  
  if len(args) < 1 {
    return "```Nothing specified.```", false
  }
  if strings.ToLower(args[0]) == "remove" {
    arg := strings.Join(args[1:], " ")
    if !RemoveSliceString(&sb.config.Statuses, arg) {
      return "```Could not find " + arg + "!```", false
    }
    sb.SaveConfig()
    return "```Removed " + arg + " as a possible status.```", false
  }
  
  arg := strings.Join(args, " ")
  sb.config.Statuses = append(sb.config.Statuses, arg)
  sb.SaveConfig()
  return "```Added " + arg + " as a possible status.```", false
}
func (c *AddStatusCommand) Usage() string { 
  return FormatUsage(c, "[remove] [arbitrary string]", "Adds a possible status message that Sweetie bot can choose") 
}
func (c *AddStatusCommand) UsageShort() string { return "Adds a status message." }
func (c *AddStatusCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *AddStatusCommand) Channels() []string { return []string{} }


type SetStatusCommand struct {
}

func (c *SetStatusCommand) Name() string {
  return "SetStatus";  
}
func (c *SetStatusCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    sb.dg.UpdateStatus(0, "")
    return "```Removed status```", false
  }
  arg := strings.Join(args, " ")
  sb.dg.UpdateStatus(0, arg)
  return "```Set status to " + arg + "```", false
}
func (c *SetStatusCommand) Usage() string { 
  return FormatUsage(c, "[arbitrary string]", "Sets Sweetie Bot's status message to the given string, at least until she automatically changes it again.") 
}
func (c *SetStatusCommand) UsageShort() string { return "Sets the status message." }
func (c *SetStatusCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *SetStatusCommand) Channels() []string { return []string{} }