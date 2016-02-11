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
func (c *EchoCommand) Process(args []string, user *discordgo.User, channel string) (string, bool) {
  if len(args) == 0 {
    return "```You have to tell me to say something, silly!```", false
  }
  arg := args[0]
  if channelregex.Match([]byte(arg)) {
    if len(args) < 2 {
      return "```You have to tell me to say something, silly!```", false
    }
    sb.SendMessage(arg[2:len(arg)-1], "```" + strings.Join(args[1:], " ") + "```")
    return "", false 
  }
  return "```" + strings.Join(args, " ") + "```", false
}
func (c *EchoCommand) Usage() string { 
  return FormatUsage(c, "[#channel] [string]", "Makes Sweetie Bot say the given sentence in #channel, or in the current channel if no argument is provided.") 
}
func (c *EchoCommand) UsageShort() string { return "Makes Sweetie Bot say something in the given channel." }
func (c *EchoCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }

func SetCommandEnable(args []string, enable bool, success string) string {
  if len(args) == 0 {
    return "No module or command specified.\n\n" + GetActiveModules() + "\n\n" + GetActiveCommands()
  }
  name := strings.ToLower(args[0])
  for _, v := range sb.modules {
    if strings.ToLower(v.Name()) == name {
      v.Enable(enable)
      if v.IsEnabled() != enable {
        return "Could not enable/disable " + args[0] + " module/command. Is this a restricted module?"
      }
      return args[0] + success + "\n\n" + GetActiveModules() + "\n\n" + GetActiveCommands()
    }
  }
  for _, v := range sb.commands {
    str := v.c.Name()
    if strings.ToLower(str) == name {
      if enable {
        delete(sb.disablecommands, str)
      } else {
        sb.disablecommands[str] = true
      }
      return args[0] + success + "\n\n" + GetActiveModules() + "\n\n" + GetActiveCommands()
    }
  }
  return "The " + args[0] + " module/command does not exist.\n\n" + GetActiveModules() + "\n\n" + GetActiveCommands()
}

type DisableCommand struct {
}

func (c *DisableCommand) Name() string {
  return "Disable";  
}
func (c *DisableCommand) Process(args []string, user *discordgo.User, channel string) (string, bool) {
  return "```" + SetCommandEnable(args, false, " was disabled.") + "```", false
}
func (c *DisableCommand) Usage() string { 
  return FormatUsage(c, "[module|command]", "Disables the given module or command, if possible. If the module/command is already disabled, does nothing.") 
}
func (c *DisableCommand) UsageShort() string { return "Disables the given module/command, if possible." }
func (c *DisableCommand) Roles() []string { return []string{"Princesses", "Royal Guard"} }


type EnableCommand struct {
}

func (c *EnableCommand) Name() string {
  return "Enable";  
}
func (c *EnableCommand) Process(args []string, user *discordgo.User, channel string) (string, bool) {
  return "```" + SetCommandEnable(args, true, " was enabled.") + "```", false
}
func (c *EnableCommand) Usage() string { 
  return FormatUsage(c, "[module|command]", "Enables the given module or command. If the module/command is already enabled, does nothing.")
}
func (c *EnableCommand) UsageShort() string { return "Enables the given module/command." }
func (c *EnableCommand) Roles() []string { return []string{"Princesses", "Royal Guard"} }

type UpdateCommand struct {
}

func (c *UpdateCommand) Name() string {
  return "Update";  
}
func (c *UpdateCommand) Process(args []string, user *discordgo.User, channel string) (string, bool) {
  /*sb.log.Log("Update command called, current PID: ", os.Getpid())
  err := exec.Command("./update.sh", strconv.Itoa(os.Getpid())).Start()
  if err != nil {
    sb.log.Log("Command.Start() error: ", err.Error())
    return "```Could not start update script!```"
  }*/ 
  sb.quit = true // Instead of trying to call a batch script, we run the bot inside an infinite loop batch script and just shut it off when we want to update
  return "```Shutting down for update...```", false
}
func (c *UpdateCommand) Usage() string { 
  return FormatUsage(c, "", "Tells sweetiebot to shut down, calls an update script, rebuilds the code, and then restarts.")
}
func (c *UpdateCommand) UsageShort() string { return "Updates sweetiebot." }
func (c *UpdateCommand) Roles() []string { return []string{"Princesses"} }

type DumpTablesCommand struct {
}

func (c *DumpTablesCommand) Name() string {
  return "DumpTables";  
}
func (c *DumpTablesCommand) Process(args []string, user *discordgo.User, channel string) (string, bool) {
  return "```" + sb.db.GetTableCounts() + "```", false
}
func (c *DumpTablesCommand) Usage() string { 
  return FormatUsage(c, "", "Dumps table row counts.")
}
func (c *DumpTablesCommand) UsageShort() string { return "Dumps table row counts." }
func (c *DumpTablesCommand) Roles() []string { return []string{"Princesses"} }