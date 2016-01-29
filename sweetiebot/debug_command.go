package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "regexp"
)

var channelregex = regexp.MustCompile("<#[0-9]*>")

type EchoCommand struct {
}

func (c *EchoCommand) Name() string {
  return "Echo";  
}
func (c *EchoCommand) Process(args []string, user *discordgo.User) string {
  if len(args) == 0 {
    return "```You have to tell me to say something, silly!```"
  }
  arg := args[0]
  if channelregex.Match([]byte(arg)) {
    sb.dg.ChannelMessageSend(arg[2:len(arg)-1], "```" + strings.Join(args[1:], " ") + "```")
    return "";  
  }
  return "```" + strings.Join(args, " ") + "```";
}
func (c *EchoCommand) Usage() string { 
  return FormatUsage(c, "[#channel] [string]", "Makes Sweetie Bot say the given sentence in #channel, or in the current channel if no argument is provided.") 
}
func (c *EchoCommand) UsageShort() string { return "Makes Sweetie Bot say something in the given channel." }
func (c *EchoCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *EchoCommand) UsePM() bool { return false }

func SetCommandEnable(args []string, enable bool, success string) string {
  if len(args) == 0 {
    return "No module specified.\n\n" + GetActiveModules()
  }
  name := strings.ToLower(args[0])
  for _, v := range sb.modules {
    if strings.ToLower(v.Name()) == name {
      v.Enable(enable)
      if v.IsEnabled() != enable {
        return "Could not enable/disable " + args[0] + " module. Is this a restricted module?"
      }
      return args[0] + success + "\n\n" + GetActiveModules()
    }
  }
  return "The " + args[0] + " module does not exist.\n\n" + GetActiveModules()
}

type DisableCommand struct {
}

func (c *DisableCommand) Name() string {
  return "Disable";  
}
func (c *DisableCommand) Process(args []string, user *discordgo.User) string {
  return "```" + SetCommandEnable(args, false, " was disabled.") + "```"
}
func (c *DisableCommand) Usage() string { 
  return FormatUsage(c, "[module]", "Disables the given module, if possible. If the module is already disabled, does nothing.") 
}
func (c *DisableCommand) UsageShort() string { return "Disables the given module, if possible." }
func (c *DisableCommand) Roles() []string { return []string{"Princesses", "Royal Guard"} }
func (c *DisableCommand) UsePM() bool { return false }


type EnableCommand struct {
}

func (c *EnableCommand) Name() string {
  return "Enable";  
}
func (c *EnableCommand) Process(args []string, user *discordgo.User) string {
  return "```" + SetCommandEnable(args, true, " was enabled.") + "```"
}
func (c *EnableCommand) Usage() string { 
  return FormatUsage(c, "[module]", "Disables the given module. If the module is already enabled, does nothing.")
}
func (c *EnableCommand) UsageShort() string { return "Enables the given module." }
func (c *EnableCommand) Roles() []string { return []string{"Princesses", "Royal Guard"} }
func (c *EnableCommand) UsePM() bool { return false }

type UpdateCommand struct {
}

func (c *UpdateCommand) Name() string {
  return "Update";  
}
func (c *UpdateCommand) Process(args []string, user *discordgo.User) string {
  /*sb.log.Log("Update command called, current PID: ", os.Getpid())
  err := exec.Command("./update.sh", strconv.Itoa(os.Getpid())).Start()
  if err != nil {
    sb.log.Log("Command.Start() error: ", err.Error())
    return "```Could not start update script!```"
  }*/ 
  sb.quit = true // Instead of trying to call a batch script, we run the bot inside an infinite loop batch script and just shut it off when we want to update
  return "```Shutting down for update...```"
}
func (c *UpdateCommand) Usage() string { 
  return FormatUsage(c, "", "Tells sweetiebot to shut down, calls an update script, rebuilds the code, and then restarts.")
}
func (c *UpdateCommand) UsageShort() string { return "Updates sweetiebot." }
func (c *UpdateCommand) Roles() []string { return []string{"Princesses"} }
func (c *UpdateCommand) UsePM() bool { return false }