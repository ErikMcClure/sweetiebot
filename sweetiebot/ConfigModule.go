package sweetiebot

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/blackhole12/discordgo"
)

// ConfigModule manages the configuration file
type ConfigModule struct {
}

// Name of the module
func (w *ConfigModule) Name() string {
	return "Configuration"
}

// Commands in the module
func (w *ConfigModule) Commands() []Command {
	return []Command{
		&setConfigCommand{},
		&getConfigCommand{},
		&setupCommand{},
		&setMemberRole{},
	}
}

// Description of the module
func (w *ConfigModule) Description(info *GuildInfo) string { return "Manages the configuration file." }

type setConfigCommand struct {
}

func (c *setConfigCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:      "SetConfig",
		Usage:     "Sets a config value and saves the new configuration.",
		Sensitive: true,
	}
}
func (c *setConfigCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nNo configuration parameter to look for!```", false, nil
	}
	if len(args) < 2 {
		return "```\nNo value to set!```", false, nil
	}
	var err error
	args[0], err = FixRequest(args[0], reflect.ValueOf(&info.Config).Elem())
	if err != nil {
		return ReturnError(err)
	}
	n, ok := info.Config.SetConfig(info, args, indices, msg.Content)
	info.SaveConfig()
	if ok {
		return "```\nSuccessfully set " + args[0] + " to " + n + ".```", false, nil
	}
	return "```\n" + n + "```", false, nil
}
func (c *setConfigCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Sets a configuration value of the format `Collection.Parameter`, possibly involving a key or multiple values, depending on the type of configuration parameter. Will only save the new configuration if it succeeds, and returns the new value upon success. To set lists of values with spaces in them, surround them with quotes, \"like this\" \"and this\". Don't use quotes on single values or key/value parameters, because the quotes won't be removed. To delete a value, set it to nothing.",
		Params: []CommandUsageParam{
			{Name: "[parameter] [value]", Desc: "Attempts to set the configuration value matching [parameter] (not case-sensitive) to [value]. Don't use quotes for these parameters, even if they have spaces.", Optional: true},
			{Name: "[list parameter] [value]", Desc: "If the parameter is a list, it will accept multiple new values. For these parameters, you must use quotes on values that have spaces.", Optional: true, Variadic: true},
			{Name: "[map parameter] [key] [value]", Desc: "If the parameter is a map, it will accept two values: the first is the key, and the second is the value of that key. The key, if it has spaces, must have quotes. The value should not have quotes, even if it has spaces.", Optional: true},
			{Name: "[maplist parameter] [key] [value]", Desc: "If the parameter is a maplist, the first value is the key, and all other values make up the list of values that key is set to. The key and all the values must have quotes if they have spaces in them.", Optional: true, Variadic: true},
		},
	}
}

type getConfigCommand struct {
}

func (c *getConfigCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:      "GetConfig",
		Usage:     "Returns the current configuration, or a specific option.",
		Sensitive: true,
	}
}

func (c *getConfigCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	t := reflect.ValueOf(&info.Config).Elem()
	n := t.NumField()
	if len(args) < 1 {
		fields := make([]*discordgo.MessageEmbedField, 0, n)
		for i := 0; i < n; i++ {
			switch t.Field(i).Kind() {
			case reflect.Struct:
				f := t.Field(i)
				s := make([]string, 0, f.NumField())
				for j := 0; j < f.NumField(); j++ {
					str := f.Type().Field(j).Name
					switch f.Field(j).Kind() {
					case reflect.Slice:
						str += " [list]"
					case reflect.Map:
						if f.Field(j).Type().Elem() == reflect.TypeOf(true) {
							str += " [list]"
						} else {
							switch f.Field(j).Type().Elem().Kind() {
							case reflect.Slice:
								fallthrough
							case reflect.Map:
								str += " [maplist]"
							default:
								str += " [map]"
							}
						}
					}
					s = append(s, str)
				}
				fields = append(fields, &discordgo.MessageEmbedField{Name: "**" + t.Type().Field(i).Name + "**", Value: strings.Join(s, "\n"), Inline: true})
			}
		}
		embed := &discordgo.MessageEmbed{
			Type: "rich",
			Author: &discordgo.MessageEmbedAuthor{
				URL:     "https://sweetiebot.io/help/",
				Name:    info.Bot.AppName + " Config Options",
				IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", info.Bot.SelfID, info.Bot.SelfAvatar),
			},
			Color:  0x3e92e5,
			Fields: fields,
		}
		info.SendEmbed(DiscordChannel(msg.ChannelID), embed)
		return "", false, nil
	}
	var err error
	args[0], err = FixRequest(args[0], t)
	if err != nil {
		return ReturnError(err)
	}
	arg := strings.SplitN(strings.ToLower(args[0]), ".", 3)
	if len(args) > 1 {
		arg = append(arg, args[1])
	}

	for i := 0; i < n; i++ {
		if strings.ToLower(t.Type().Field(i).Name) == arg[0] {
			switch t.Field(i).Kind() {
			case reflect.Struct:
				f := t.Field(i)
				if len(arg) > 1 {
					for j := 0; j < f.NumField(); j++ {
						if strings.ToLower(f.Type().Field(j).Name) == arg[1] {
							lines := getSubStruct(arg, f, j, info)
							if len(lines) == 0 {
								return fmt.Sprintf("```\n%s.%s: [empty]```", arg[0], arg[1]), false, nil
							} else if len(lines) == 1 {
								return fmt.Sprintf("```\n%s.%s: %s```", arg[0], arg[1], info.Sanitize(lines[0], CleanCodeBlock)), false, nil
							}
							return fmt.Sprintf("```\n--- %s.%s ---\n%s```", arg[0], arg[1], info.Sanitize(strings.Join(lines, "\n"), CleanCodeBlock)), false, nil
						}
					}
				} else {
					fields := make([]*discordgo.MessageEmbedField, 0, f.NumField())
					dump := []string{}
					for j := 0; j < f.NumField(); j++ {
						desc, ok := getConfigHelp(t.Type().Field(i).Name, f.Type().Field(j).Name)
						if !ok {
							desc = "\u200b"
						}
						fields = append(fields, &discordgo.MessageEmbedField{Name: "**" + f.Type().Field(j).Name + "**", Value: desc, Inline: false})

						lines := getSubStruct(arg, f, j, info)
						if len(lines) == 0 {
							dump = append(dump, fmt.Sprintf("%s: [empty]", f.Type().Field(j).Name))
						} else if len(lines) == 1 {
							dump = append(dump, fmt.Sprintf("%s: %s", f.Type().Field(j).Name, info.Sanitize(lines[0], CleanCodeBlock)))
						} else {
							dump = append(dump, fmt.Sprintf("%s: [%v items]", f.Type().Field(j).Name, len(lines)))
						}
					}
					embed := &discordgo.MessageEmbed{
						Type: "rich",
						Author: &discordgo.MessageEmbedAuthor{
							URL:     "https://sweetiebot.io/help/" + strings.ToLower(t.Type().Field(i).Name),
							Name:    t.Type().Field(i).Name + " Config Category",
							IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", info.Bot.SelfID, info.Bot.SelfAvatar),
						},
						Description: "```\n" + strings.Join(dump, "\n") + "```",
						Color:       0x3e92e5,
						Fields:      fields,
					}
					return "", false, embed
				}
			}
		}
	}

	return "```\nThat's not a recognized config option! Type " + info.Config.Basic.CommandPrefix + "getconfig without any arguments to list all possible config options. Use \".\" to specify which category of options you want - for example, \"Basic.ModChannel\". If the option is a map, you can specify the key as well: \"Help.Rules 1\". Using " + info.Config.Basic.CommandPrefix + "getconfig with just a category will list help for that category, e.g. \"" + info.Config.Basic.CommandPrefix + "getconfig Basic\".```", false, nil
}
func (c *getConfigCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Displays a list of available configuration options or their values.",
		Params: []CommandUsageParam{
			{Name: "option", Desc: "The configuration option to display. Use `Help.Rules` to specify a config option in a category. If this is just a category, like `Basic`, lists help information for all config options in that category.", Optional: true},
			{Name: "map key", Desc: "If the option is a map, this determines the particular key to display. For example: `" + info.Config.Basic.CommandPrefix + "getconfig Help.Rules 1` will return rule 1 in the rules map.", Optional: true},
		},
	}
}

func (c *setupCommand) DisableModule(info *GuildInfo, module string) {
	for _, v := range info.Modules {
		if strings.ToLower(v.Name()) == module {
			cmds := v.Commands()
			for _, v := range cmds {
				str := strings.ToLower(v.Info().Name)
				CheckMapNilBool(&info.Config.Modules.CommandDisabled)
				info.Config.Modules.CommandDisabled[CommandID(str)] = true
			}
		}
	}

	info.Config.Modules.Disabled[ModuleID(module)] = true
}

type setupCommand struct {
}

func (c *setupCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:  "Setup",
		Usage: "Performs first-time initialization on this server.",
	}
}
func (c *setupCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	guild, err := info.GetGuild()
	if err != nil || guild == nil {
		return "```\nCan't find guild in state object?!?", false, nil
	}
	perms, _ := info.Bot.DG.UserPermissions(DiscordUser(msg.Author.ID), info.ID)
	if perms&discordgo.PermissionAdministrator == 0 {
		return "```\nOnly administrators can use this command!```", false, nil
	}
	if len(args) < 2 {
		return "```\nYou must provide at least the Moderator Role and Mod Channel arguments to this function.```", false, nil
	}
	if info.Config.SetupDone {
		if strings.ToLower(args[0]) != "override" {
			return "```\nWARNING: This server has already been configured. If you run " + info.Config.Basic.CommandPrefix + "setup again, it will reset ALL CONFIGURATION DATA to defaults! If you wish to proceed, use " + info.Config.Basic.CommandPrefix + "setup OVERRIDE <your arguments>```", false, nil
		}
		args = args[1:]
		indices = indices[1:]
		info.Config = *DefaultConfig()
	}
	if len(args) < 2 {
		return "```\nYou must provide at least the Moderator Role and Mod Channel arguments to this function.```", false, nil
	}
	if len(args) > 4 {
		return fmt.Sprintf("```\nThis function only accepts 4 arguments, but you put in %v! Are you actually using @Role for the mod role and #channel for the channels? Alternatively, put your moderator role in \"quotes\".```", len(args)), false, nil
	}

	info.Config.Basic.ModRole, err = ParseRole(args[0], guild)
	if err != nil || info.Config.Basic.ModRole == RoleEmpty || info.Config.Basic.ModRole == RoleExclusion {
		return args[0] + " is not a valid role!", false, nil
	}
	info.Config.Basic.ModChannel, err = ParseChannel(args[1], guild)
	if err != nil || info.Config.Basic.ModChannel == ChannelEmpty || info.Config.Basic.ModChannel == ChannelExclusion {
		return args[1] + " is not a valid channel!", false, nil
	}

	if len(args) > 2 {
		info.Config.Log.Channel, err = ParseChannel(args[2], guild)
		if err != nil || info.Config.Log.Channel == ChannelEmpty || info.Config.Log.Channel == ChannelExclusion {
			return args[2] + " is not a valid channel!", false, nil
		}
	}

	if len(args) > 3 {
		info.Config.Basic.MemberRole, err = ParseRole(args[3], guild)
		if err != nil || info.Config.Basic.MemberRole == RoleEmpty {
			return args[3] + " is not a valid role!", false, nil
		}
	}

	if info.Config.Basic.MemberRole != RoleExclusion {
		if e := setupMemberRole(info.Config.Basic.MemberRole, DiscordChannel(msg.ChannelID), info); e != "" {
			return e, false, nil
		}
	} else {
		info.Config.Basic.MemberRole = RoleEmpty
	}

	silent, err := info.Bot.DG.GuildRoleCreate(info.ID)
	if err != nil {
		return fmt.Sprintf("```\nFailed to create the silent role! %s```", err.Error()), false, nil
	}
	_, err = info.Bot.DG.GuildRoleEdit(info.ID, silent.ID, "Silence", 0, false, discordgo.PermissionReadMessages, false)
	if err != nil {
		info.Bot.DG.GuildRoleDelete(info.ID, silent.ID)
		return fmt.Sprintf("```\nFailed to set up the silent role! %s```", err.Error()), false, nil
	}

	info.Config.Basic.SilenceRole, _ = ParseRole(silent.ID, nil)
	info.Config.Basic.Aliases = make(map[string]string)
	info.Config.Basic.Aliases["calc"] = "roll"
	info.Config.Basic.Aliases["calculate"] = "roll"
	info.Config.Modules.CommandRoles = make(map[CommandID]map[DiscordRole]bool)

	for k, v := range info.commands {
		if v.Info().Sensitive {
			info.Config.Modules.CommandRoles[k] = make(map[DiscordRole]bool)
			info.Config.Modules.CommandRoles[k][info.Config.Basic.ModRole] = true
		}
	}

	info.Config.Modules.CommandDisabled = make(map[CommandID]bool)
	info.Config.Modules.Disabled = make(map[ModuleID]bool)

	c.DisableModule(info, "bucket")
	c.DisableModule(info, "bored")
	c.DisableModule(info, "markov")
	c.DisableModule(info, "witty")
	c.DisableModule(info, "poll")
	c.DisableModule(info, "misc")

	modname := info.Config.Basic.ModRole.Show(info)
	modchannel := info.Config.Basic.ModChannel.Show(info)
	logchannel := info.Config.Log.Channel.Show(info)
	membername := "[not used]"

	if info.Config.Basic.MemberRole != RoleEmpty {
		membername = info.Config.Basic.MemberRole.Show(info)
	}

	info.setupSilenceRole()
	info.Config.SetupDone = true
	info.SaveConfig()
	return fmt.Sprintf("```\nServer configured!\nModerator Role: %v\nMod Channel: %v\nLog Channel: %v\nMember Role: %v```\nNow that you've done basic configuration on %s, here are some additional features you can enable. For additional help, type `"+info.Config.Basic.CommandPrefix+"help` for a list of commands and modules, or `"+info.Config.Basic.CommandPrefix+"getconfig` with no arguments for a list of configuration options. Using `"+info.Config.Basic.CommandPrefix+"help <module>` will display detailed help for that module and all its commands. Using `"+info.Config.Basic.CommandPrefix+"getconfig <group>` will display detailed help for all the configuration options in that configuration group. If you're still confused, please check the website: https://sweetiebot.io/\n\n**Bucket**\nIf you'd like to enable the bucket, use the command `"+info.Config.Basic.CommandPrefix+"enable Bucket`. It defaults to carrying a maximum of 10 items, but you can change this via the `Bucket.MaxItems` option.\n\n**Bored Module**\nIf you'd like "+info.GetBotName()+" to perform actions when the chat in a certain channel hasn't been active for a period of time, use `"+info.Config.Basic.CommandPrefix+"enable bored` followed by `"+info.Config.Basic.CommandPrefix+"setconfig modules.channels bored #yourchannel`, where `#yourchannel` is your general chat channel. The commands picked from are stored in `bored.commands`. By default, it will quote someone or attempt to throw an item out of the bucket.\n\n**Free Channels**\nIf you like, you can designate a channel to be free from command restrictions, so people can spam silly bot commands to their hearts content. If you had a channel called `#bot` for this, you can disable all command restrictions by using the command ```"+info.Config.Basic.CommandPrefix+"setconfig basic.freechannels #bot```.",
		modname,
		modchannel,
		logchannel,
		membername,
		info.GetBotName()), false, nil
}
func (c *setupCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Sets up " + info.GetBotName() + " on this server and restricts all sensitive commands to `Moderator Role`.",
		Params: []CommandUsageParam{
			{Name: "Moderator Role", Desc: "A role shared by all moderators. It is used to alert moderators and also allows the moderators to bypass command restrictions imposed by certain modules.", Optional: false},
			{Name: "Mod Channel", Desc: "Whatever channel the moderators would like to receive notifications on, such as potential raids, spammers being silenced, etc.", Optional: false},
			{Name: "Log Channel", Desc: "An optional channel that receives log messages about errors and initialization. Usually this channel is only visible to the bot and the moderators.", Optional: true},
			{Name: "Member Role", Desc: "If you have an existing role that all users are assigned to, provide it here. Otherwise, the bot will create a new role called \"Member\" that it will assign to all users. If you don't want to use this feature, provide `!` here.", Optional: true},
		},
	}
}

type setMemberRole struct {
}

func (c *setMemberRole) Info() *CommandInfo {
	return &CommandInfo{
		Name:      "SetMemberRole",
		Usage:     "Sets the member role and migrates the server.",
		Sensitive: true,
	}
}
func (c *setMemberRole) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) > 0 {
		g, err := info.GetGuild()
		var role DiscordRole
		if role, err = ParseRole(msg.Content[indices[0]:], g); err != nil {
			return fmt.Sprintf("```%s is not a valid role!```", msg.Content[indices[0]:]), false, nil
		}
		if err := setupMemberRole(role, DiscordChannel(msg.ChannelID), info); err != "" {
			return err, false, nil
		}
	} else {
		if err := setupMemberRole(RoleEmpty, DiscordChannel(msg.ChannelID), info); err != "" {
			return err, false, nil
		}
	}

	return "Added a member role to all users and removed all permissions from everyone role. If you haven't changed it, JailChannel will have been set to your old WelcomeChannel. Silenced users go to JailChannel, whereas users without Member will go to WelcomeChannel. Feel free to make them seperate channels or keep them as the same channel.", false, nil
}
func (c *setMemberRole) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Sets the member role and adds it to all the users before removing all permissions from the everyone role. If an existing role is specified, skips over members that already have it.",
		Params: []CommandUsageParam{
			{Name: "member role", Desc: "An existing role to use as the \"member role\". If not specified, the bot generates a role called \"Member\".", Optional: true},
		},
	}
}

func setupMemberRole(memberRole DiscordRole, channel DiscordChannel, info *GuildInfo) string {
	if memberRole == RoleEmpty {
		role, err := info.Bot.DG.GuildRoleCreate(info.ID)
		if err != nil {
			return fmt.Sprintf("```\nFailed to create the member role! %s```", err.Error())
		}
		everyone, err := info.Bot.DG.State.Role(info.ID, info.ID)
		if err == nil {
			_, err = info.Bot.DG.GuildRoleEdit(info.ID, role.ID, "Member", 0, false, everyone.Permissions, false)
		}
		if err != nil {
			info.Bot.DG.GuildRoleDelete(info.ID, role.ID)
			return fmt.Sprintf("```\nFailed to set up the member role! %s```", err.Error())
		}
		memberRole = DiscordRole(role.ID)
	}

	if memberRole == RoleEmpty {
		return "```memberRole is somehow still empty, aborting!```"
	}

	info.Config.Basic.MemberRole = memberRole

	if g, err := info.GetGuild(); err == nil {
		count := g.MemberCount
		if count < len(g.Members) {
			count = len(g.Members)
		}
		info.SendMessage(channel, fmt.Sprintf("Adding %v to %v users. Given discord's average rate limit of 10 roles per 10 seconds, this will take approximately %v minutes. If this process is interrupted, you can restart by running this command with the existing Member role as an argument, and it will skip over members that already have it. You should get status updates every minute, or about every 60 users.", memberRole.Show(info), count, count/60))
	} else {
		return "```Error getting guild information, aborting!```"
	}

	count := 0
	lastid := ""
	for {
		m, err := info.Bot.DG.GuildMembers(info.ID, lastid, 60)
		if err != nil || len(m) == 0 {
			break
		}
		for _, v := range m {
			if !MemberHasRole(v, memberRole) {
				info.Bot.DG.GuildMemberRoleAdd(info.ID, v.User.ID, memberRole.String())
				count++
			}
		}

		if count > 0 {
			info.SendMessage(channel, fmt.Sprintf("%v users processed...", count))
		}

		lastid = m[len(m)-1].User.ID
	}

	if _, err := info.Bot.DG.GuildRoleEdit(info.ID, info.ID, "", 0, false, 0, false); err != nil {
		return "```Failed to remove permissions from everyone role!```"
	}

	return ""
}
