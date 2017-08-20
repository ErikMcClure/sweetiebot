package sweetiebot

import (
	"fmt"
	"sort"
	"strings"

	"strconv"

	"github.com/blackhole12/discordgo"
)

type DebugModule struct {
}

func (w *DebugModule) Name() string {
	return "Debug"
}

func (w *DebugModule) Commands() []Command {
	return []Command{
		&EchoCommand{},
		&EchoEmbedCommand{},
		&DisableCommand{},
		&EnableCommand{},
		&UpdateCommand{},
		&DumpTablesCommand{},
		&ListGuildsCommand{},
		&AnnounceCommand{},
		&RemoveAliasCommand{},
		&GetAuditCommand{},
	}
}

func (w *DebugModule) Description() string {
	return "Contains various debugging commands. Some of these commands can only be run by the bot owner."
}

type EchoCommand struct {
}

func (c *EchoCommand) Name() string {
	return "Echo"
}
func (c *EchoCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) == 0 {
		return "```You have to tell me to say something, silly!```", false, nil
	}
	arg := args[0]
	if channelregex.MatchString(arg) {
		if len(args) < 2 {
			return "```You have to tell me to say something, silly!```", false, nil
		}
		info.SendMessage(arg[2:len(arg)-1], msg.Content[indices[1]:])
		return "", false, nil
	}
	return msg.Content[indices[0]:], false, nil
}
func (c *EchoCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Makes Sweetie Bot say the given sentence in `#channel`, or in the current channel if no channel is provided.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "#channel", Desc: "The channel to echo the message in. If omitted, message is sent to this channel.", Optional: true},
			CommandUsageParam{Name: "arbitrary string", Desc: "An arbitrary string for Sweetie Bot to say.", Optional: false},
		},
	}
}
func (c *EchoCommand) UsageShort() string {
	return "Makes Sweetie Bot say something in the given channel."
}

type EchoEmbedCommand struct {
}

func (c *EchoEmbedCommand) Name() string {
	return "EchoEmbed"
}
func (c *EchoEmbedCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) == 0 {
		return "```You have to tell me to say something, silly!```", false, nil
	}
	arg := args[0]
	channel := msg.ChannelID
	i := 0
	if channelregex.MatchString(arg) {
		if len(args) < 2 {
			return "```You have to tell me to say something, silly!```", false, nil
		}
		channel = arg[2 : len(arg)-1]
		i++
	}
	if i >= len(args) {
		return "```A URL is mandatory or discord won't send the embed message for some stupid reason.```", false, nil
	}
	url := args[i]
	i++
	var color uint64 = 0xFFFFFFFF
	if i < len(args) {
		if colorregex.MatchString(args[i]) {
			if len(args) < i+2 {
				return "```You have to tell me to say something, silly!```", false, nil
			}
			color, _ = strconv.ParseUint(args[i][2:], 16, 64)
			i++
		}
	}
	fields := make([]*discordgo.MessageEmbedField, 0, len(args)-i)
	for i < len(args) {
		s := strings.SplitN(args[i], ":", 2)
		if len(s) < 2 {
			return "```Malformed key:value pair. If your key value pair has a space in it, remember to put it in paranthesis!```", false, nil
		}
		fields = append(fields, &discordgo.MessageEmbedField{Name: s[0], Value: s[1], Inline: true})
		i++
	}
	embed := &discordgo.MessageEmbed{
		Type: "rich",
		Author: &discordgo.MessageEmbedAuthor{
			URL:     url,
			Name:    msg.Author.Username + "#" + msg.Author.Discriminator,
			IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.jpg", msg.Author.ID, msg.Author.Avatar),
		},
		Color:  int(color),
		Fields: fields,
	}
	info.SendEmbed(channel, embed)
	return "", false, nil
}
func (c *EchoEmbedCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Makes Sweetie Bot assemble a rich text embed and echo it in the given channel",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "#channel", Desc: "The channel to echo the message in. If omitted, message is sent to this channel.", Optional: true},
			CommandUsageParam{Name: "URL", Desc: "URL for the author to link to.", Optional: false},
			CommandUsageParam{Name: "0xC0L0R", Desc: "Color of the embed box.", Optional: true},
			CommandUsageParam{Name: "key:value", Desc: "A key:value pair of fields to display in the embed. Remember to use quotes around the *entire* key:value pair if either the key or the value have spaces.", Optional: true, Variadic: true},
		},
	}
}
func (c *EchoEmbedCommand) UsageShort() string {
	return "Makes Sweetie Bot echo a rich text embed in a given channel."
}

func SetCommandEnable(args []string, enable bool, success string, info *GuildInfo, channelID string) (string, bool, *discordgo.MessageEmbed) {
	if len(args) == 0 {
		return "```No module or command specified.Use " + info.config.Basic.CommandPrefix + "help with no arguments to list all modules and commands.```", false, nil
	}
	name := strings.ToLower(args[0])
	for _, v := range info.modules {
		if strings.ToLower(v.Name()) == name {
			cmds := v.Commands()
			for _, v := range cmds {
				str := strings.ToLower(v.Name())
				if enable {
					delete(info.config.Modules.CommandDisabled, str)
				} else {
					CheckMapNilBool(&info.config.Modules.CommandDisabled)
					info.config.Modules.CommandDisabled[str] = true
				}
			}

			if enable {
				delete(info.config.Modules.Disabled, name)
			} else {
				CheckMapNilBool(&info.config.Modules.Disabled)
				info.config.Modules.Disabled[name] = true
			}
			info.SaveConfig()
			return "", false, DumpCommandsModules(channelID, info, "", "**Success!** "+args[0]+success)
		}
	}
	for _, v := range info.commands {
		str := strings.ToLower(v.Name())
		if str == name {
			if enable {
				delete(info.config.Modules.CommandDisabled, str)
			} else {
				CheckMapNilBool(&info.config.Modules.CommandDisabled)
				info.config.Modules.CommandDisabled[str] = true
			}
			info.SaveConfig()
			return "", false, DumpCommandsModules(channelID, info, "", "**Success!** "+args[0]+success)
		}
	}
	return "```The " + args[0] + " module/command does not exist. Use " + info.config.Basic.CommandPrefix + "help with no arguments to list all modules and commands.```", false, nil
}

type DisableCommand struct {
}

func (c *DisableCommand) Name() string {
	return "Disable"
}
func (c *DisableCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	return SetCommandEnable(args, false, " was disabled.", info, msg.ChannelID)
}
func (c *DisableCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Disables the given module or command, if possible. If the module/command is already disabled, does nothing.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "module|command", Desc: "The module or command to disable. You do not need to specify the parent module of a command, only the command name itself.", Optional: false},
		},
	}
}
func (c *DisableCommand) UsageShort() string { return "Disables the given module/command, if possible." }

type EnableCommand struct {
}

func (c *EnableCommand) Name() string {
	return "Enable"
}
func (c *EnableCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	return SetCommandEnable(args, true, " was enabled.", info, msg.ChannelID)
}
func (c *EnableCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Enables the given module or command, if possible. If the module/command is already enabled, does nothing.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "module|command", Desc: "The module or command to enable. You do not need to specify the parent module of a command, only the command name itself.", Optional: false},
		},
	}
}
func (c *EnableCommand) UsageShort() string { return "Enables the given module/command." }
func (c *EnableCommand) Roles() []string    { return []string{"Princesses", "Royal Guard"} }
func (c *EnableCommand) Channels() []string { return []string{} }

type UpdateCommand struct {
}

func (c *UpdateCommand) Name() string {
	return "Update"
}
func (c *UpdateCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	_, isOwner := sb.Owners[SBatoi(msg.Author.ID)]
	if !isOwner {
		return "```Only the owner of the bot itself can call this!```", false, nil
	}
	/*sb.log.Log("Update command called, current PID: ", os.Getpid())
	  err := exec.Command("./update.sh", strconv.Itoa(os.Getpid())).Start()
	  if err != nil {
	    sb.log.Log("Command.Start() error: ", err.Error())
	    return "```Could not start update script!```"
	  }*/

	sb.guildsLock.RLock()
	defer sb.guildsLock.RUnlock()
	for _, v := range sb.guilds {
		if v.config.Log.Channel > 0 {
			v.SendMessage(SBitoa(v.config.Log.Channel), "```Shutting down for update...```")
		}
	}

	sb.quit.set(true) // Instead of trying to call a batch script, we run the bot inside an infinite loop batch script and just shut it off when we want to update
	return "```Shutting down for update...```", false, nil
}
func (c *UpdateCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{Desc: "Tells sweetiebot to shut down, calls an update script, rebuilds the code, and then restarts."}
}
func (c *UpdateCommand) UsageShort() string { return "[RESTRICTED] Updates sweetiebot." }
func (c *UpdateCommand) Roles() []string    { return []string{"Princesses"} }
func (c *UpdateCommand) Channels() []string { return []string{} }

type DumpTablesCommand struct {
}

func (c *DumpTablesCommand) Name() string {
	return "DumpTables"
}
func (c *DumpTablesCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	return "```\n" + sb.db.GetTableCounts() + "```", false, nil
}
func (c *DumpTablesCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{Desc: "Dumps table row counts."}
}
func (c *DumpTablesCommand) UsageShort() string { return "Dumps table row counts." }

type GuildSlice []*discordgo.Guild

func (s GuildSlice) Len() int {
	return len(s)
}
func (s GuildSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s GuildSlice) Less(i, j int) bool {
	if s[i].MemberCount > len(s[i].Members) {
		i = s[i].MemberCount
	} else {
		i = len(s[i].Members)
	}
	if s[j].MemberCount > len(s[j].Members) {
		j = s[j].MemberCount
	} else {
		j = len(s[j].Members)
	}
	return i > j
}

type ListGuildsCommand struct {
}

func (c *ListGuildsCommand) Name() string {
	return "ListGuilds"
}
func (c *ListGuildsCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	_, isOwner := sb.Owners[SBatoi(msg.Author.ID)]
	sb.dg.State.RLock()
	guilds := append([]*discordgo.Guild{}, sb.dg.State.Guilds...)
	sb.dg.State.RUnlock()
	sort.Sort(GuildSlice(guilds))
	s := make([]string, 0, len(guilds))
	private := 0
	for _, v := range guilds {
		if !isOwner {
			sb.guildsLock.RLock()
			g, ok := sb.guilds[SBatoi(v.ID)]
			sb.guildsLock.RUnlock()
			if ok && g.config.Basic.Importable {
				s = append(s, PartialSanitize(v.Name))
			} else {
				private++
			}
		} else {
			username := "<@" + v.OwnerID + ">"
			if sb.db.status.get() {
				m, _, _, _ := sb.db.GetUser(SBatoi(v.OwnerID))
				if m != nil {
					username = m.Username + "#" + m.Discriminator
				}
			}
			count := v.MemberCount
			if count < len(v.Members) {
				count = len(v.Members)
			}
			s = append(s, PartialSanitize(fmt.Sprintf("%v (%v) - %v", v.Name, count, username)))
		}
	}
	return fmt.Sprintf("```Sweetie has joined these servers:\n%s\n\n+ %v private servers (Basic.Importable is false)```", strings.Join(s, "\n"), private), len(s) > 8, nil
}
func (c *ListGuildsCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{Desc: "Lists the servers that sweetiebot has joined."}
}
func (c *ListGuildsCommand) UsageShort() string { return "Lists servers." }

type AnnounceCommand struct {
}

func (c *AnnounceCommand) Name() string {
	return "Announce"
}
func (c *AnnounceCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	_, isOwner := sb.Owners[SBatoi(msg.Author.ID)]
	if !isOwner {
		return "```Only the owner of the bot itself can call this!```", false, nil
	}

	arg := msg.Content[indices[0]:]
	sb.guildsLock.RLock()
	defer sb.guildsLock.RUnlock()
	for _, v := range sb.guilds {
		if v.config.Log.Channel > 0 {
			v.SendMessage(SBitoa(v.config.Log.Channel), "<@&"+SBitoa(v.config.Basic.AlertRole)+"> "+arg)
		}
	}

	return "", false, nil
}
func (c *AnnounceCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Restricted command that announces a message to all the log channels of all servers.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "arbitrary string", Desc: "An arbitrary string for Sweetie Bot to say.", Optional: false},
		},
	}
}
func (c *AnnounceCommand) UsageShort() string { return "[RESTRICTED] Announcement command." }

type RemoveAliasCommand struct {
}

func (c *RemoveAliasCommand) Name() string {
	return "RemoveAlias"
}
func (c *RemoveAliasCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	_, isOwner := sb.Owners[SBatoi(msg.Author.ID)]
	if !isOwner {
		return "```Only the owner of the bot itself can call this!```", false, nil
	}
	if len(args) < 1 {
		return "```You must PING the user you want to remove an alias from.```", false, nil
	}
	if len(args) < 2 {
		return "```You must provide an alias to remove.```", false, nil
	}
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	sb.db.RemoveAlias(PingAtoi(args[0]), msg.Content[indices[1]:])
	return "```Attempted to remove the alias. Use " + info.config.Basic.CommandPrefix + "aka to check if it worked.```", false, nil
}
func (c *RemoveAliasCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Restricted command that removes the alias for a given user. The user must be pinged, and the alias must match precisely.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "user", Desc: "A ping to a specific user in the format @User.", Optional: false},
			CommandUsageParam{Name: "alias", Desc: "The *exact* name of the alias to remove.", Optional: false},
		},
	}
}
func (c *RemoveAliasCommand) UsageShort() string { return "[RESTRICTED] Removes an alias." }

type GetAuditCommand struct {
}

func (c *GetAuditCommand) Name() string {
	return "GetAudit"
}
func (c *GetAuditCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	var low uint64 = 0
	var high uint64 = 10
	var user *uint64 = nil
	var search string = ""

	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}

	for i := 0; i < len(args); i++ {
		if len(args[i]) > 0 {
			switch args[i][0] {
			case '<', '@':
				if args[i][0] == '@' || (len(args[i]) > 1 && args[i][1] == '@') {
					var IDs []uint64
					if args[i][0] == '@' {
						IDs = FindUsername(args[i][1:], info)
					} else {
						IDs = []uint64{SBatoi(StripPing(args[i]))}
					}
					if len(IDs) == 0 { // no matches!
						return "```Error: Could not find any usernames or aliases matching " + args[i] + "!```", false, nil
					}
					if len(IDs) > 1 {
						return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info, true), "\n") + "```", len(IDs) > 5, nil
					}
					user = &IDs[0]
					break
				}
				fallthrough
			case '$', '!':
				if args[i][0] != '!' {
					search = "%"
				}
				if args[i][0] == '$' {
					search += msg.Content[indices[i]+1:] + "%"
				} else {
					search += msg.Content[indices[i]:] + "%"
				}
				i = len(args)
			default:
				s := strings.SplitN(args[i], "-", 2)
				if len(s) == 1 {
					high = SBatoi(s[0])
				} else if len(s) > 1 {
					low = SBatoi(s[0]) - 1
					high = SBatoi(s[1])
				}
			}
		}
	}

	r := sb.db.GetAuditRows(low, high, user, search, SBatoi(info.ID))
	ret := []string{"```Matching Audit Log entries:```"}

	for _, v := range r {
		ret = append(ret, fmt.Sprintf("[%s] %s: %s", ApplyTimezone(v.Timestamp, info, msg.Author).Format("1/2 3:04:05PM"), v.Author, v.Message))
	}

	return strings.Join(ret, "\n"), len(ret) > 12, nil
}
func (c *GetAuditCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Allows admins to inspect the audit log.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "range", Desc: "If this is a single number, the number of results to return. If it's a range in the form 999-9999, returns the given range of audit log entries, up to a maximum of 50 in one call. Defaults to displaying 1-10.", Optional: true},
			CommandUsageParam{Name: "user", Desc: "Must be in the form of @user, either as an actual ping or just part of the users name. If included, filters results to just that user. If there are spaces in the username, you must use quotes.", Optional: true},
			CommandUsageParam{Name: "arbitrary string", Desc: "An arbitrary string starting with either `!` or `$`. `!` will search for an exact command (regardless of what your command prefix has been set to), whereas `$` will simply search for the string anywhere in the audit log. This will eat up all remaining arguments, so put the user and the range BEFORE specifying the search string, and don't use quotes!", Optional: true},
		},
	}
}
func (c *GetAuditCommand) UsageShort() string { return "Inspects the audit log." }
