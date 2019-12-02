package sweetiebot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/erikmcclure/discordgo"
)

var colorregex = regexp.MustCompile("0x[0-9A-Fa-f]+")

// DebugModule contains various debugging commands
type DebugModule struct {
	lastcheck int64
	lastclean int64
}

// Name of the module
func (w *DebugModule) Name() string {
	return "Debug"
}

// Commands in the module
func (w *DebugModule) Commands() []Command {
	return []Command{
		&echoCommand{},
		&echoEmbedCommand{},
		&disableCommand{},
		&enableCommand{},
		&updateCommand{},
		&dumpTablesCommand{},
		&listGuildsCommand{},
		&announceCommand{},
		&removeAliasCommand{},
		&getAuditCommand{},
		&setProfileCommand{},
	}
}

// Description of the module
func (w *DebugModule) Description(info *GuildInfo) string {
	return "Contains various debugging commands and checks for updates. Some of these commands can only be run by the bot owner. If you are selfhosting sweetie bot, these commands can be used to debug the bot."
}

// OnTick event hook
func (w *DebugModule) OnTick(info *GuildInfo, t time.Time) {
	if info.Bot.IsMainGuild(info) && t.Unix()-w.lastcheck > UpdateInterval {
		w.lastcheck = t.Unix()
		m := &discordgo.Message{ChannelID: info.Config.Log.Channel.String(),
			Content:   "",
			Author:    &discordgo.User{ID: info.Bot.Owner.String()},
			Timestamp: discordgo.Timestamp(t.Format(time.RFC3339Nano)),
		}

		updater := &updateCommand{}
		updater.Process([]string{}, m, []int{}, info)
	}

	if info.Bot.IsMainGuild(info) && t.Unix()-w.lastclean > CleanInterval {
		go func() { // Getting the user list takes forever because of discord API limits, so we do this on a new thread
			fmt.Println("CLEANING GUILDS")
			w.lastclean = t.Unix()

			lastid := ""
			guilds := make(map[string]*discordgo.UserGuild)
			list, err := info.Bot.DG.UserGuilds(100, "", lastid)
			for err == nil && len(list) > 0 {
				for _, v := range list {
					guilds[v.ID] = v
				}
				lastid = list[len(list)-1].ID
				list, err = info.Bot.DG.UserGuilds(100, "", lastid)
			}
			info.LogError("Error cleaning guilds: ", err)

			if err == nil { // ONLY proceed if we have a complete listing of all our guilds with no errors
				cur, _ := GetCurrentDir()
				timeNow := time.Now().UTC().Unix()
				results, err := ioutil.ReadDir(cur)
				info.LogError("Error cleaning guilds: ", err)

				for _, f := range results {
					if !f.IsDir() {
						matches := guildfileregex.FindStringSubmatch(f.Name())
						if len(matches) > 1 {
							id := matches[1]
							_, ok := guilds[id]
							config := BotConfig{}
							cfgfile, err := ioutil.ReadFile(filepath.Join(cur, f.Name()))
							if err == nil {
								err = json.Unmarshal(cfgfile, &config)
							}
							if err != nil {
								info.LogError("Error loading config: ", err)
								continue
							}

							if ok || config.Expires == 0 {
								config.Expires = timeNow + ExpireTime
								info.Bot.GuildsLock.RLock()
								guild, ok := info.Bot.Guilds[DiscordGuild(id)]
								info.Bot.GuildsLock.RUnlock()

								if ok {
									guild.ConfigLock.Lock()
									guild.Config.Expires = config.Expires
									guild.ConfigLock.Unlock()
								}

								if data, err := json.Marshal(&config); err == nil {
									if err = ioutil.WriteFile(id+".json", data, 0664); err != nil {
										info.Log("Error saving config file: ", err.Error())
									}
								} else {
									info.Log("Error marshalling config file: ", err.Error())
								}
							} else if config.Expires < timeNow {
								fmt.Println("Server ID " + id + " has expired.")
								info.Bot.GuildsLock.Lock()
								delete(info.Bot.Guilds, DiscordGuild(id))
								info.Bot.GuildsLock.Unlock()
								if err := os.Remove(id + ".json"); err == nil {
									err = info.Bot.DB.RemoveGuild(SBatoi(id))
								}
								info.LogError("Error deleting guild: ", err)
							}
						}
					}
				}
			}
		}()
	}
}

type echoCommand struct {
}

func (c *echoCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:      "Echo",
		Usage:     "Says something in the given channel.",
		Sensitive: true,
	}
}
func (c *echoCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou have to tell me to say something, silly!```", false, nil
	}
	if ChannelRegex.MatchString(args[0]) || (len(args[0]) > 0 && args[0][0] == '#') {
		if len(args) < 2 {
			return "```\nYou have to tell me to say something, silly!```", false, nil
		}
		g, _ := info.GetGuild()
		ch, err := ParseChannel(args[0], g)
		if err != nil {
			return ReturnError(err)
		}
		if err = info.SendMessage(ch, msg.Content[indices[1]:]); err != nil {
			return ReturnError(err)
		}
		return "", false, nil
	}
	return msg.Content[indices[0]:], false, nil
}
func (c *echoCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Makes " + info.GetBotName() + " say the given sentence in `#channel`. If `#channel` is omitted, returns the string in the current channel.",
		Params: []CommandUsageParam{
			{Name: "#channel", Desc: "The channel to echo the message in. Must have the `#` prefix, but doesn't have to be a channel ping.", Optional: true},
			{Name: "arbitrary string", Desc: "An arbitrary string for " + info.GetBotName() + " to say.", Optional: false},
		},
	}
}

type echoEmbedCommand struct {
}

func (c *echoEmbedCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:      "EchoEmbed",
		Usage:     "Makes the bot echo a rich text embed in a given channel.",
		Sensitive: true,
	}
}
func (c *echoEmbedCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) == 0 {
		return "```\nYou have to tell me to say something, silly!```", false, nil
	}
	arg := args[0]
	channel := msg.ChannelID
	i := 0
	if ChannelRegex.MatchString(arg) {
		if len(args) < 2 {
			return "```\nYou have to tell me to say something, silly!```", false, nil
		}
		channel = arg[2 : len(arg)-1]
		i++
	}
	if i >= len(args) {
		return "```\nA URL is mandatory or discord won't send the embed message for some stupid reason.```", false, nil
	}
	url := args[i]
	i++
	var color uint64 = 0xFFFFFFFF
	if i < len(args) {
		if colorregex.MatchString(args[i]) {
			if len(args) < i+2 {
				return "```\nYou have to tell me to say something, silly!```", false, nil
			}
			color, _ = strconv.ParseUint(args[i][2:], 16, 64)
			i++
		}
	}
	fields := make([]*discordgo.MessageEmbedField, 0, len(args)-i)
	for i < len(args) {
		s := strings.SplitN(args[i], ":", 2)
		if len(s) < 2 {
			return "```\nMalformed key:value pair. If your key value pair has a space in it, remember to put it in parenthesis!```", false, nil
		}
		fields = append(fields, &discordgo.MessageEmbedField{Name: "**" + s[0] + "**", Value: s[1], Inline: true})
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
	info.SendEmbed(DiscordChannel(channel), embed)
	return "", false, nil
}
func (c *echoEmbedCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Makes the bot assemble a rich text embed and echo it in the given channel",
		Params: []CommandUsageParam{
			{Name: "#channel", Desc: "The channel to echo the message in. If omitted, message is sent to this channel.", Optional: true},
			{Name: "URL", Desc: "URL for the author to link to.", Optional: false},
			{Name: "0xC0L0R", Desc: "Color of the embed box.", Optional: false},
			{Name: "key:value", Desc: "A key:value pair of fields to display in the embed. Remember to use quotes around the *entire* key:value pair if either the key or the value have spaces.", Optional: true, Variadic: true},
		},
	}
}

func setCommandEnable(args []string, enable bool, success string, info *GuildInfo, msg *discordgo.Message) (string, bool, *discordgo.MessageEmbed) {
	if len(args) == 0 {
		return "```\nNo module or command specified.Use " + info.Config.Basic.CommandPrefix + "help with no arguments to list all modules and commands.```", false, nil
	}
	name := strings.ToLower(args[0])
	for _, v := range info.Modules {
		if strings.ToLower(v.Name()) == name {
			cmds := v.Commands()
			for _, v := range cmds {
				str := strings.ToLower(v.Info().Name)
				if enable {
					delete(info.Config.Modules.CommandDisabled, CommandID(str))
				} else {
					CheckMapNilBool(&info.Config.Modules.CommandDisabled)
					info.Config.Modules.CommandDisabled[CommandID(str)] = true
				}
			}

			if enable {
				delete(info.Config.Modules.Disabled, ModuleID(name))
			} else {
				if len(info.Config.Modules.Disabled) == 0 {
					info.Config.Modules.Disabled = make(map[ModuleID]bool)
				}
				info.Config.Modules.Disabled[ModuleID(name)] = true
			}
			info.SaveConfig()
			return "", false, DumpCommandsModules(info, "", "**Success!** "+args[0]+success, msg)
		}
	}
	for k := range info.commands {
		if string(k) == name {
			if enable {
				delete(info.Config.Modules.CommandDisabled, k)
			} else {
				CheckMapNilBool(&info.Config.Modules.CommandDisabled)
				info.Config.Modules.CommandDisabled[k] = true
			}
			info.SaveConfig()
			return "", false, DumpCommandsModules(info, "", "**Success!** "+args[0]+success, msg)
		}
	}
	return "```\nThe " + args[0] + " module/command does not exist. Use " + info.Config.Basic.CommandPrefix + "help with no arguments to list all modules and commands.```", false, nil
}

type disableCommand struct {
}

func (c *disableCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:      "Disable",
		Usage:     "Disables the given module/command, if possible.",
		Sensitive: true,
	}
}
func (c *disableCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	return setCommandEnable(args, false, " was disabled.", info, msg)
}
func (c *disableCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Disables the given module or command, if possible. If the module/command is already disabled, does nothing.",
		Params: []CommandUsageParam{
			{Name: "module|command", Desc: "The module or command to disable. You do not need to specify the parent module of a command, only the command name itself.", Optional: false},
		},
	}
}
func (c *disableCommand) UsageShort() string { return "Disables the given module/command, if possible." }

type enableCommand struct {
}

func (c *enableCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:      "Enable",
		Usage:     "Enables the given module/command.",
		Sensitive: true,
	}
}
func (c *enableCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	return setCommandEnable(args, true, " was enabled.", info, msg)
}
func (c *enableCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Enables the given module or command, if possible. If the module/command is already enabled, does nothing.",
		Params: []CommandUsageParam{
			{Name: "module|command", Desc: "The module or command to enable. You do not need to specify the parent module of a command, only the command name itself.", Optional: false},
		},
	}
}

type updateCommand struct {
}

func (c *updateCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:              "Update",
		Usage:             "Updates the bot.",
		Restricted:        true,
		Sensitive:         true,
		ServerIndependent: true,
	}
}
func (c *updateCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.Owner.Equals(msg.Author.ID) {
		return "```\nOnly the owner of the bot itself can call this!```", false, nil
	}
	if info.Bot.UpdateLock.TestAndSet() {
		return "```\nThe bot is already checking for or downloading an update.```", false, nil
	}
	defer info.Bot.UpdateLock.Clear()
	switch atomic.LoadUint32(&info.Bot.quit) {
	case QuitNow:
		return "```\nThe bot is shutting down.```", false, nil
	case QuitRaid:
		return "```\nAn update has already been scheduled, and will happen soon.```", false, nil
	}

	r, update := info.Bot.Selfhoster.CheckForUpdate(info.Bot.Owner, BotVersion.Integer())
	switch r {
	case -1:
		return buySelfhosting, false, nil
	case 0:
		return "```\n" + info.GetBotName() + " is currently up-to-date.```", false, nil
	case 1:
		info.SendMessage(DiscordChannel(msg.ChannelID), "```\nAn update to v"+VersionInt(update.Version).String()+" is available, downloading files now. The bot will restart when the download is complete (or after any active raids have subsided)```")
	}

	for _, file := range update.Files { // We ignore any errors here because the updater will re-attempt the downloads anyway
		DownloadFile(UpdateEndpoint(file, info.Bot.Owner, 0), "~"+file, false)
	}

	info.Bot.GuildsLock.RLock()
	defer info.Bot.GuildsLock.RUnlock()
	for _, v := range info.Bot.Guilds {
		if v.Config.Log.Channel != ChannelEmpty && v.Config.Log.Channel != ChannelExclusion && !v.Config.Log.Channel.Equals(msg.ChannelID) {
			v.SendMessage(v.Config.Log.Channel, "```\nShutting down for update...```")
		}
	}

	atomic.StoreUint32(&info.Bot.quit, QuitRaid) // Instead of trying to call a batch script, we run the bot inside an infinite loop batch script and just shut it off when we want to update
	return "```\nShutting down for update...```", false, nil
}
func (c *updateCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{Desc: "Shuts down, calls an update script, rebuilds the code, and then restarts."}
}

type dumpTablesCommand struct {
}

func (c *dumpTablesCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:              "DumpTables",
		Usage:             "Dumps table row counts.",
		Restricted:        true,
		Sensitive:         true,
		ServerIndependent: true,
	}
}
func (c *dumpTablesCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.Owner.Equals(msg.Author.ID) {
		return "```\nOnly the owner of the bot itself can call this!```", false, nil
	}
	return "```\n" + info.Bot.DB.GetTableCounts() + "```", false, nil
}
func (c *dumpTablesCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{Desc: "Dumps table row counts."}
}
func (c *dumpTablesCommand) UsageShort() string { return "Dumps table row counts." }

type guildSlice []*discordgo.Guild

func (s guildSlice) Len() int {
	return len(s)
}
func (s guildSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s guildSlice) Less(i, j int) bool {
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

type listGuildsCommand struct {
}

func (c *listGuildsCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:              "ListGuilds",
		Usage:             "Lists the servers the bot is on.",
		MainInstance:      true,
		Sensitive:         true,
		ServerIndependent: true,
	}
}
func (c *listGuildsCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.Owner.Equals(msg.Author.ID) {
		return "```\nOnly the owner of the bot itself can call this!```", false, nil
	}
	info.Bot.DG.State.RLock()
	guilds := append([]*discordgo.Guild{}, info.Bot.DG.State.Guilds...)
	info.Bot.DG.State.RUnlock()
	sort.Sort(guildSlice(guilds))
	s := make([]string, 0, len(guilds))
	private := 0
	for _, v := range guilds {
		username := "<@" + v.OwnerID + ">"
		m, _ := info.Bot.DG.GetMember(DiscordUser(v.OwnerID), v.ID)
		if m != nil {
			username = m.User.Username + "#" + m.User.Discriminator
		}
		count := v.MemberCount
		if count < len(v.Members) {
			count = len(v.Members)
		}
		if count > 200 {
			s = append(s, info.Sanitize(fmt.Sprintf("%v (%v) - %v", v.Name, count, username), CleanCodeBlock))
		} else {
			private++
		}
	}
	return fmt.Sprintf("```\n%s has joined these servers:\n%s\n\n+ %v private servers (Basic.Importable is false)```", info.GetBotName(), strings.Join(s, "\n"), private), len(s) > 8, nil
}
func (c *listGuildsCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{Desc: "Lists the servers the bot is on."}
}

type announceCommand struct {
}

func (c *announceCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:              "Announce",
		Usage:             "Announcement command.",
		Restricted:        true,
		Sensitive:         true,
		ServerIndependent: true,
	}
}
func (c *announceCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.Owner.Equals(msg.Author.ID) {
		return "```\nOnly the owner of the bot itself can call this!```", false, nil
	}

	arg := msg.Content[indices[0]:]
	info.Bot.GuildsLock.RLock()
	defer info.Bot.GuildsLock.RUnlock()
	for _, v := range info.Bot.Guilds {
		if v.Config.Log.Channel != ChannelEmpty {
			v.SendMessage(v.Config.Log.Channel, v.Config.Basic.ModRole.Display()+" "+arg)
		}
	}

	return "", false, nil
}
func (c *announceCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Restricted command that announces a message to all the log channels of all servers.",
		Params: []CommandUsageParam{
			{Name: "arbitrary string", Desc: "An arbitrary string for " + info.GetBotName() + " to say.", Optional: false},
		},
	}
}

type removeAliasCommand struct {
}

func (c *removeAliasCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:       "RemoveAlias",
		Usage:      "Removes an alias.",
		Restricted: true,
		Sensitive:  true,
	}
}
func (c *removeAliasCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.Owner.Equals(msg.Author.ID) {
		return "```\nOnly the owner of the bot itself can call this!```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou must PING the user you want to remove an alias from.```", false, nil
	}
	if len(args) < 2 {
		return "```\nYou must provide an alias to remove.```", false, nil
	}
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	info.Bot.DB.RemoveAlias(PingAtoi(args[0]), msg.Content[indices[1]:])
	return "```\nAttempted to remove the alias. Use " + info.Config.Basic.CommandPrefix + "aka to check if it worked.```", false, nil
}
func (c *removeAliasCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Restricted command that removes the alias for a given user. The user must be pinged, and the alias must match precisely.",
		Params: []CommandUsageParam{
			{Name: "user", Desc: "A ping to a specific user in the format @User.", Optional: false},
			{Name: "alias", Desc: "The *exact* name of the alias to remove.", Optional: false},
		},
	}
}

type getAuditCommand struct {
}

func (c *getAuditCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:      "GetAudit",
		Usage:     "Inspects the audit log.",
		Sensitive: true,
	}
}
func (c *getAuditCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	var low uint64
	var high uint64 = 10
	var user *uint64
	var search string

	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}

	for i := range args {
		if len(args[i]) > 0 {
			switch args[i][0] {
			case '<', '@':
				if args[i][0] == '@' || (len(args[i]) > 1 && args[i][1] == '@') {
					var IDs []uint64
					if args[i][0] == '@' {
						IDs = info.FindUsername(args[i][1:])
					} else {
						IDs = []uint64{SBatoi(StripPing(args[i]))}
					}
					if len(IDs) == 0 { // no matches!
						return "```\nError: Could not find any usernames or aliases matching " + args[i] + "!```", false, nil
					}
					if len(IDs) > 1 {
						return "```\nCould be any of the following users or their aliases:\n" + strings.Join(info.IDsToUsernames(IDs, true), "\n") + "```", len(IDs) > 5, nil
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

	r := info.Bot.DB.GetAuditRows(low, high, user, search, SBatoi(info.ID))
	ret := []string{"```\nMatching Audit Log entries:```"}

	for _, v := range r {
		ret = append(ret, fmt.Sprintf("[%s] %s: %s", info.ApplyTimezone(v.Timestamp, DiscordUser(msg.Author.ID)).Format("1/2 3:04:05PM"), v.Author, v.Message))
	}

	return info.Sanitize(strings.Join(ret, "\n"), CleanMost), len(ret) > 12, nil
}
func (c *getAuditCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Allows admins to inspect the audit log.",
		Params: []CommandUsageParam{
			{Name: "range", Desc: "If this is a single number, the number of results to return. If it's a range in the form 999-9999, returns the given range of audit log entries, up to a maximum of 50 in one call. Defaults to displaying 1-10.", Optional: true},
			{Name: "user", Desc: "Must be in the form of @user, either as an actual ping or just part of the users name. If included, filters results to just that user. If there are spaces in the username, you must use quotes.", Optional: true},
			{Name: "arbitrary string", Desc: "An arbitrary string starting with either `!` or `$`. `!` will search for an exact command (regardless of what your command prefix has been set to), whereas `$` will simply search for the string anywhere in the audit log. This will eat up all remaining arguments, so put the user and the range BEFORE specifying the search string, and don't use quotes!", Optional: true},
		},
	}
}

type setProfileCommand struct {
}

func (c *setProfileCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:              "SetProfile",
		Usage:             "Changes username and avatar of the bot.",
		Sensitive:         true,
		ServerIndependent: true,
		MainInstance:      true,
	}
}
func (c *setProfileCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.Owner.Equals(msg.Author.ID) {
		return "```\nOnly the owner of the bot itself can call this!```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou must include at least a username to change.```", false, nil
	}
	avatarfile := ""
	if len(args) > 1 {
		avatarfile = msg.Content[indices[1]:]
	}
	if err := info.Bot.DG.ChangeBotName(args[0], avatarfile); err != nil {
		return fmt.Sprintf("```\nError changing bot name or avatar: %s```", err.Error()), false, nil
	}
	if len(args) < 1 {
		return "```\nSuccessfully changed the bot name!```", false, nil
	}
	return "```\nSuccessfully changed the bot name and avatar!```", false, nil
}
func (c *setProfileCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Restricted command that changes the bot name and/or avatar.",
		Params: []CommandUsageParam{
			{Name: "username", Desc: "What the bot's username should be. If the name has spaces, you must put this argument in quotes!", Optional: false},
			{Name: "avatar", Desc: "A PNG, JPG or GIF file relative to the bot's executable that contains the avatar. If this parameter is omitted, the avatar is not changed. Quotes are optional.", Optional: true},
		},
	}
}
