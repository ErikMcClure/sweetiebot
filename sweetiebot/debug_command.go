package sweetiebot

import (
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
		info.SendMessage(arg[2:len(arg)-1], strings.Join(args[1:], " "))
		return "", false
	}
	return strings.Join(args, " "), false
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
		v.SendMessage(SBitoa(v.config.LogChannel), "```Shutting down for update...```")
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
		v.SendMessage(SBitoa(v.config.LogChannel), "<@&"+SBitoa(v.config.AlertRole)+"> "+arg)
	}
	return "", false
}
func (c *AnnounceCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[arbitrary message]", "Restricted command that announces a message to all the log channels of all servers.")
}
func (c *AnnounceCommand) UsageShort() string { return "Restricted announcement command." }

type SilenceCommand struct {
}

func (c *SilenceCommand) Name() string {
	return "Silence"
}
func (c *SilenceCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You must provide a user to silence.```", false
	}
	arg := strings.Join(args, " ")
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	if SilenceMember(SBitoa(IDs[0]), info) < 0 {
		return "```Error occured trying to silence " + IDsToUsernames(IDs, info)[0] + ".```", false
	}
	return "```Silenced " + IDsToUsernames(IDs, info)[0] + ".```", false
}
func (c *SilenceCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[user]", "Silences the given user.")
}
func (c *SilenceCommand) UsageShort() string { return "Silences a user." }

type UnsilenceCommand struct {
}

func (c *UnsilenceCommand) Name() string {
	return "Unsilence"
}
func (c *UnsilenceCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You must provide a user to unsilence.```", false
	}
	arg := strings.Join(args, " ")
	IDs := FindUsername(arg, info)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	srole := SBitoa(info.config.SilentRole)
	userID := SBitoa(IDs[0])
	m, err := sb.dg.GuildMember(info.Guild.ID, userID)
	if err != nil {
		return "```Could not get member: " + err.Error() + "```", false
	}
	for i := 0; i < len(m.Roles); i++ {
		if m.Roles[i] == srole {
			m.Roles = append(m.Roles[:i], m.Roles[i+1:]...)
			sb.dg.GuildMemberEdit(info.Guild.ID, userID, m.Roles)
			return "```Unsilenced " + IDsToUsernames(IDs, info)[0] + ".```", false
		}
	}
	return "```" + IDsToUsernames(IDs, info)[0] + " wasn't silenced in the first place!```", false
}
func (c *UnsilenceCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[user]", "Unsilences the given user.")
}
func (c *UnsilenceCommand) UsageShort() string { return "Unsilences a user." }

type RemoveAliasCommand struct {
}

func (c *RemoveAliasCommand) Name() string {
	return "RemoveAlias"
}
func (c *RemoveAliasCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	_, isOwner := sb.Owners[SBatoi(msg.Author.ID)]
	if !isOwner {
		return "```Only the owner of the bot itself can call this!```", false
	}
	if len(args) < 1 {
		return "```You must PING the user you want to remove an alias from.```", false
	}
	if len(args) < 2 {
		return "```You must provide an alias to remove.```", false
	}
	sb.db.RemoveAlias(PingAtoi(args[0]), strings.Join(args[1:], " "))
	return "```Attempted to remove the alias. Use !aka to check if it worked.```", false
}
func (c *RemoveAliasCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[user] [alias]", "Removes the alias for the given user. The user must be pinged, and the alias must match precisely.")
}
func (c *RemoveAliasCommand) UsageShort() string { return "Removes an alias." }
