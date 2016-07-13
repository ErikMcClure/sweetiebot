package sweetiebot

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type EchoCommand struct {
}

func (c *EchoCommand) Name() string {
	return "Echo"
}
func (c *EchoCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) == 0 {
		return "```You have to tell me to say something, silly!```", false
	}
	arg := args[0]
	if channelregex.MatchString(arg) {
		if len(args) < 2 {
			return "```You have to tell me to say something, silly!```", false
		}
		info.SendMessage(arg[2:len(arg)-1], "```"+strings.Join(args[1:], " ")+"```")
		return "", false
	}
	return "```" + strings.Join(args, " ") + "```", false
}
func (c *EchoCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[#channel] [string]", "Makes Sweetie Bot say the given sentence in #channel, or in the current channel if no argument is provided.")
}
func (c *EchoCommand) UsageShort() string {
	return "Makes Sweetie Bot say something in the given channel."
}

func SetCommandEnable(args []string, enable bool, success string, info *GuildInfo) string {
	if len(args) == 0 {
		return "No module or command specified.\n\n" + info.GetActiveModules() + "\n\n" + info.GetActiveCommands()
	}
	name := strings.ToLower(args[0])
	for _, v := range info.modules {
		if strings.ToLower(v.Name()) == name {
			if enable {
				delete(info.config.Module_disabled, name)
			} else {
				CheckMapNilBool(&info.config.Module_disabled)
				info.config.Module_disabled[name] = true
				info.SaveConfig()
			}
			return args[0] + success + "\n\n" + info.GetActiveModules() + "\n\n" + info.GetActiveCommands()
		}
	}
	for _, v := range info.commands {
		str := strings.ToLower(v.Name())
		if str == name {
			if enable {
				delete(info.config.Command_disabled, str)
			} else {
				CheckMapNilBool(&info.config.Command_disabled)
				info.config.Command_disabled[str] = true
				info.SaveConfig()
			}
			return args[0] + success + "\n\n" + info.GetActiveModules() + "\n\n" + info.GetActiveCommands()
		}
	}
	return "The " + args[0] + " module/command does not exist.\n\n" + info.GetActiveModules() + "\n\n" + info.GetActiveCommands()
}

type DisableCommand struct {
}

func (c *DisableCommand) Name() string {
	return "Disable"
}
func (c *DisableCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	return "```" + SetCommandEnable(args, false, " was disabled.", info) + "```", false
}
func (c *DisableCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[module|command]", "Disables the given module or command, if possible. If the module/command is already disabled, does nothing.")
}
func (c *DisableCommand) UsageShort() string { return "Disables the given module/command, if possible." }

type EnableCommand struct {
}

func (c *EnableCommand) Name() string {
	return "Enable"
}
func (c *EnableCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	return "```" + SetCommandEnable(args, true, " was enabled.", info) + "```", false
}
func (c *EnableCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[module|command]", "Enables the given module or command. If the module/command is already enabled, does nothing.")
}
func (c *EnableCommand) UsageShort() string { return "Enables the given module/command." }
func (c *EnableCommand) Roles() []string    { return []string{"Princesses", "Royal Guard"} }
func (c *EnableCommand) Channels() []string { return []string{} }

type UpdateCommand struct {
}

func (c *UpdateCommand) Name() string {
	return "Update"
}
func (c *UpdateCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	_, isOwner := sb.Owners[SBatoi(msg.Author.ID)]
	if !isOwner {
		return "```Only the owner of the bot itself can call this!```", false
	}
	/*sb.log.Log("Update command called, current PID: ", os.Getpid())
	  err := exec.Command("./update.sh", strconv.Itoa(os.Getpid())).Start()
	  if err != nil {
	    sb.log.Log("Command.Start() error: ", err.Error())
	    return "```Could not start update script!```"
	  }*/

	for _, v := range sb.guilds {
		v.SendMessage(strconv.FormatUint(v.config.LogChannel, 10), "```Shutting down for update...```")
	}

	sb.quit = true // Instead of trying to call a batch script, we run the bot inside an infinite loop batch script and just shut it off when we want to update
	return "```Shutting down for update...```", false
}
func (c *UpdateCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "", "Tells sweetiebot to shut down, calls an update script, rebuilds the code, and then restarts.")
}
func (c *UpdateCommand) UsageShort() string { return "Updates sweetiebot." }
func (c *UpdateCommand) Roles() []string    { return []string{"Princesses"} }
func (c *UpdateCommand) Channels() []string { return []string{} }

type DumpTablesCommand struct {
}

func (c *DumpTablesCommand) Name() string {
	return "DumpTables"
}
func (c *DumpTablesCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	return "```" + sb.db.GetTableCounts() + "```", false
}
func (c *DumpTablesCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "", "Dumps table row counts.")
}
func (c *DumpTablesCommand) UsageShort() string { return "Dumps table row counts." }

type ListGuildsCommand struct {
}

func (c *ListGuildsCommand) Name() string {
	return "ListGuilds"
}
func (c *ListGuildsCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	s := make([]string, 0, len(sb.guilds))
	for _, v := range sb.guilds {
		s = append(s, v.Guild.Name)
	}
	return "```Sweetie has joined these servers:\n" + strings.Join(s, "\n") + "```", len(s) > 8
}
func (c *ListGuildsCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "", "Lists the servers that sweetiebot has joined.")
}
func (c *ListGuildsCommand) UsageShort() string { return "Lists servers." }

type AnnounceCommand struct {
}

func (c *AnnounceCommand) Name() string {
	return "Announce"
}
func (c *AnnounceCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	_, isOwner := sb.Owners[SBatoi(msg.Author.ID)]
	if !isOwner {
		return "```Only the owner of the bot itself can call this!```", false
	}

	arg := strings.Join(args, " ")
	for _, v := range sb.guilds {
		v.SendMessage(strconv.FormatUint(v.config.LogChannel, 10), "<@&"+strconv.FormatUint(v.config.AlertRole, 10)+"> "+arg)
	}
	return "", false
}
func (c *AnnounceCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[arbitrary message]", "Restricted command that announces a message to all the log channels of all guilds.")
}
func (c *AnnounceCommand) UsageShort() string { return "Restricted announcement command." }
