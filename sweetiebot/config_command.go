package sweetiebot

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"strconv"

	"github.com/blackhole12/discordgo"
)

type ConfigModule struct {
}

func (w *ConfigModule) Name() string {
	return "Configuration"
}

func (w *ConfigModule) Register(info *GuildInfo) {}

func (w *ConfigModule) Commands() []Command {
	return []Command{
		&SetConfigCommand{},
		&GetConfigCommand{},
		&SetupCommand{},
	}
}

func (w *ConfigModule) Description() string { return "Manages Sweetie Bot's configuration file." }

func FixRequest(arg string, t reflect.Value) (string, error) {
	args := strings.SplitN(strings.ToLower(arg), ".", 3)
	list := []string{}
	n := t.NumField()

	for i := 0; i < n; i++ {
		if strings.ToLower(t.Type().Field(i).Name) == args[0] {
			return arg, nil
		}
	}

	for i := 0; i < n; i++ {
		switch t.Field(i).Kind() {
		case reflect.Struct:
			f := t.Field(i)
			for j := 0; j < f.NumField(); j++ {
				if strings.ToLower(f.Type().Field(j).Name) == args[0] {
					list = append(list, t.Type().Field(i).Name)
				}
			}
		}
	}
	if len(list) < 1 {
		return arg, nil
	}
	if len(list) == 1 {
		return strings.ToLower(list[0]) + "." + arg, nil
	}
	for i := 0; i < len(list); i++ {
		list[i] += "." + args[0]
	}
	return "", errors.New("```Could be any of the following:\n" + strings.Join(list, "\n") + "```")
}

type SetConfigCommand struct {
}

func (c *SetConfigCommand) Name() string {
	return "SetConfig"
}
func (c *SetConfigCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```No configuration parameter to look for!```", false, nil
	}
	if len(args) < 2 {
		return "```No value to set!```", false, nil
	}
	var err error
	args[0], err = FixRequest(args[0], reflect.ValueOf(&info.config).Elem())
	if err != nil {
		return err.Error(), false, nil
	}
	n, ok := info.SetConfig(args[0], args[1], args[2:]...)
	info.SaveConfig()
	if ok {
		return "```Successfully set " + args[0] + " to " + n + ".```", false, nil
	}
	return "```" + n + "```", false, nil
}
func (c *SetConfigCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Sets a configuration value of the format `Collection.Parameter`, possibly involving a key or multiple values, depending on the type of configuration parameter. Will only save the new configuration if it succeeds, and returns the new value upon success.  To set a value with a space in it, surround it with quotes, \"like so\".",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "[parameter] [value]", Desc: "Attempts to set the configuration value matching [parameter] (not case-sensitive) to [value]", Optional: true},
			CommandUsageParam{Name: "[list parameter] [value]", Desc: "If the parameter is a list, it will accept multiple new values.", Optional: true, Variadic: true},
			CommandUsageParam{Name: "[map parameter] [key] [value]", Desc: "If the parameter is a map, it will accept two values: the first is the key, and the second is the value of that key.", Optional: true},
			CommandUsageParam{Name: "[maplist parameter] [key] [value]", Desc: " If the parameter is a maplist, the first value is the key, and all other values make up the list of values that key is set to.", Optional: true, Variadic: true},
		},
	}
}
func (c *SetConfigCommand) UsageShort() string {
	return "Sets a config value and saves the new configuration."
}

type GetConfigCommand struct {
}

func (c *GetConfigCommand) Name() string {
	return "GetConfig"
}
func (c *GetConfigCommand) GetOption(f reflect.Value, info *GuildInfo, t reflect.Value) (string, bool, *discordgo.MessageEmbed) {
	s := []string{}
	fields := make([]*discordgo.MessageEmbedField, 0, 0)
	switch f.Interface().(type) {
	case []string:
		s = f.Interface().([]string)
	case []uint64:
		t, _ := f.Interface().([]uint64)
		for _, v := range t {
			s = append(s, SBitoa(v))
		}
	case map[string]string:
		t, _ := f.Interface().(map[string]string)
		for k, v := range t {
			s = append(s, fmt.Sprintf("\"%v\": %v", k, v))
		}
	case map[int64]int:
		t, _ := f.Interface().(map[int64]int)
		for k, v := range t {
			s = append(s, fmt.Sprintf("\"%v\": %v", k, v))
		}
	case map[int]string:
		t, _ := f.Interface().(map[int]string)
		for k, v := range t {
			s = append(s, fmt.Sprintf("\"%v\": %v", k, v))
		}
	case map[string]int64:
		t, _ := f.Interface().(map[string]int64)
		for k, v := range t {
			//fields = append(fields, &discordgo.MessageEmbedField{Name: k, Value: strconv.Itoa(int(v)), Inline: true})
			s = append(s, fmt.Sprintf("\"%v\": %v", k, v))
		}
	case map[string]bool:
		t, _ := f.Interface().(map[string]bool)
		for k := range t {
			s = append(s, k)
		}
	case map[uint64][]string:
		t, _ := f.Interface().(map[uint64][]string)
		for k, v := range t {
			if len(v) == 1 {
				s = append(s, fmt.Sprintf("\"%v\": %s", k, v[0]))
			} else {
				s = append(s, fmt.Sprintf("\"%v\": [%v items]", k, len(v)))
			}
		}
	case map[string]map[string]bool:
		t, _ := f.Interface().(map[string]map[string]bool)
		for k, v := range t {
			if len(v) == 1 {
				for q := range v {
					s = append(s, fmt.Sprintf("\"%v\": %s", k, q))
				}
			} else {
				s = append(s, fmt.Sprintf("\"%v\": [%v items]", k, len(v)))
			}
		}
	default:
		data, err := json.Marshal(f.Interface())
		if err != nil {
			info.log.Log("JSON error: ", err.Error())
			s = append(s, "[JSON Error]")
		} else {
			s = append(s, ExtraSanitize(string(data)))
		}
	}
	if len(fields) > 0 {
		desc, _ := ConfigHelp[strings.ToLower(t.Type().Name()+"."+f.Type().Name())]
		return "", false, &discordgo.MessageEmbed{
			Type: "rich",
			Author: &discordgo.MessageEmbedAuthor{
				URL:     "https://github.com/blackhole12/sweetiebot#configuration",
				Name:    f.Type().Name(),
				IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", sb.SelfID, sb.SelfAvatar),
			},
			Description: desc,
			Color:       0x3e92e5,
			Fields:      fields,
		}
	}
	return "```\n" + strings.Join(s, "\n") + "```", false, nil
}

func (c *GetConfigCommand) GetSubStruct(arg []string, f reflect.Value, j int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(arg) > 2 {
		str := f.Type().Field(j).Name
		var val reflect.Value
		switch f.Field(j).Interface().(type) {
		case map[string]string, map[string]int64, map[string]map[string]bool:
			val = f.Field(j).MapIndex(reflect.ValueOf(arg[2]))
		case map[int64]int:
			ival, _ := strconv.ParseInt(arg[2], 10, 64)
			val = f.Field(j).MapIndex(reflect.ValueOf(ival))
		case map[int]string:
			ival, _ := strconv.Atoi(arg[2])
			val = f.Field(j).MapIndex(reflect.ValueOf(ival))
		case map[uint64][]string:
			val = f.Field(j).MapIndex(reflect.ValueOf(SBatoi(arg[2])))
		default:
			return fmt.Sprintf("```Error: %s is not a map.```", str), false, nil
		}
		if !val.IsValid() || val == reflect.Zero(val.Type()) {
			return fmt.Sprintf("```Error: Can't find %v in %s.```", arg[2], str), false, nil
		}
		return c.GetOption(val, info, f)
	}
	return c.GetOption(f.Field(j), info, f)
}

func (c *GetConfigCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	t := reflect.ValueOf(&info.config).Elem()
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
					switch f.Field(j).Interface().(type) {
					case []uint64, map[string]bool:
						str += " [list]"
					case map[string]string, map[int64]int, map[int]string, map[string]int64:
						str += " [map]"
					case map[uint64][]string, map[string]map[string]bool:
						str += " [maplist]"
					}
					s = append(s, str)
				}
				fields = append(fields, &discordgo.MessageEmbedField{Name: t.Type().Field(i).Name, Value: strings.Join(s, "\n"), Inline: true})
			}
		}
		embed := &discordgo.MessageEmbed{
			Type: "rich",
			Author: &discordgo.MessageEmbedAuthor{
				URL:     "https://github.com/blackhole12/sweetiebot#configuration",
				Name:    "Sweetie Bot Config Options",
				IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", sb.SelfID, sb.SelfAvatar),
			},
			Color:  0x3e92e5,
			Fields: fields,
		}
		info.SendEmbed(msg.ChannelID, embed)
		return "", false, nil
	}
	var err error
	args[0], err = FixRequest(args[0], t)
	if err != nil {
		return err.Error(), false, nil
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
							return c.GetSubStruct(arg, f, j, info)
						}
					}
				} else {
					fields := make([]*discordgo.MessageEmbedField, 0, f.NumField())
					for j := 0; j < f.NumField(); j++ {
						desc, ok := ConfigHelp[strings.ToLower(t.Type().Field(i).Name+"."+f.Type().Field(j).Name)]
						if !ok {
							desc = "\u200b"
						}
						fields = append(fields, &discordgo.MessageEmbedField{Name: f.Type().Field(j).Name, Value: desc, Inline: false})
					}
					embed := &discordgo.MessageEmbed{
						Type: "rich",
						Author: &discordgo.MessageEmbedAuthor{
							URL:     "https://github.com/blackhole12/sweetiebot#configuration",
							Name:    t.Type().Field(i).Name + " Config Category",
							IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", sb.SelfID, sb.SelfAvatar),
						},
						Color:  0x3e92e5,
						Fields: fields,
					}
					return "", false, embed
				}
			}
		}
	}

	return "```That's not a recognized config option! Type !getconfig without any arguments to list all possible config options. Use \".\" to specify which category of options you want - for example, \"Basic.ModChannel\". If the option is a map, you can specify the key as well: \"Help.Rules 1\". Using !getconfig with just a category will list help for that category, e.g. \"" + info.config.Basic.CommandPrefix + "getconfig Basic\".```", false, nil
}
func (c *GetConfigCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Displays a list of available configuration options or their values.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "option", Desc: "The configuration option to display. Use `Help.Rules` to specify a config option in a category. If this is just a category, like `Basic`, lists help information for all config options in that category.", Optional: true},
			CommandUsageParam{Name: "map key", Desc: "If the option is a map, this determines the particular key to display. For example: `" + info.config.Basic.CommandPrefix + "getconfig Help.Rules 1` will return rule 1 in the rules map.", Optional: true},
		},
	}
}
func (c *GetConfigCommand) UsageShort() string {
	return "Returns the current configuration, or a specific option."
}

func (c *SetupCommand) DisableModule(info *GuildInfo, module string) {
	for _, v := range info.modules {
		if strings.ToLower(v.Name()) == module {
			cmds := v.Commands()
			for _, v := range cmds {
				str := strings.ToLower(v.Name())
				CheckMapNilBool(&info.config.Modules.CommandDisabled)
				info.config.Modules.CommandDisabled[str] = true
			}
		}
	}

	info.config.Modules.Disabled[module] = true
}

type SetupCommand struct {
}

func (c *SetupCommand) Name() string {
	return "Setup"
}
func (c *SetupCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	guild, err := sb.dg.State.Guild(info.ID)
	if err != nil || guild == nil {
		return "```Can't find guild in state object?!?", false, nil
	}
	if msg.Author.ID != guild.OwnerID {
		return "```Only the owner of this server can use this command!```", false, nil
	}
	if len(args) < 2 {
		return "```You must provide at least the Moderator Role and Mod Channel arguments to this function.```", false, nil
	}
	if len(args) > 3 {
		return fmt.Sprintf("```This function only accepts 3 arguments, but you put in %v! Are you actually using @Role for the mod role and #channel for the channels?```", len(args)), false, nil
	}

	mod := StripPing(args[0])
	modchannel := StripPing(args[1])

	if SBatoi(modchannel) == 0 {
		if args[1][0] == '#' {
			args[1] = strings.ToLower(args[1][1:])
			for _, c := range guild.Channels {
				if strings.ToLower(c.Name) == args[1] {
					modchannel = c.ID
					break
				}
			}
		}
		if SBatoi(modchannel) == 0 {
			return fmt.Sprintf("```%s is not a valid channel ID! Remember to use #channel so discord actually sends the ID.```", modchannel), false, nil
		}
	}
	if SBatoi(mod) == 0 {
		if args[0][0] == '@' {
			args[0] = strings.ToLower(args[0][1:])
			for _, r := range guild.Roles {
				if strings.ToLower(r.Name) == args[0] {
					mod = r.ID
					break
				}
			}
		}
		if SBatoi(mod) == 0 {
			return fmt.Sprintf("```%s is not a valid role ID! Remember to use @role so discord actually sends the ID.```", mod), false, nil
		}
	}

	log := "[None]"
	if len(args) > 2 {
		log = StripPing(args[2])
		if SBatoi(log) == 0 {
			if args[2][0] == '#' {
				args[2] = strings.ToLower(args[2][1:])
				for _, c := range guild.Channels {
					if strings.ToLower(c.Name) == args[2] {
						log = c.ID
						break
					}
				}
			}
			if SBatoi(log) == 0 {
				return fmt.Sprintf("```%s is not a valid channel ID!```", log), false, nil
			}
		}
		info.config.Log.Channel = SBatoi(log)
	}

	silent, err := sb.dg.GuildRoleCreate(info.ID)
	if err != nil {
		return fmt.Sprintf("```Failed to create the silent role! %s```", err.Error()), false, nil
	}
	_, err = sb.dg.GuildRoleEdit(info.ID, silent.ID, "Silence", 0, false, 0x00000400, false)
	if err != nil {
		sb.dg.GuildRoleDelete(info.ID, silent.ID)
		return fmt.Sprintf("```Failed to set up the silent role! %s```", err.Error()), false, nil
	}

	info.config.Basic.AlertRole = SBatoi(mod)
	info.config.Basic.ModChannel = SBatoi(modchannel)
	info.config.Spam.SilentRole = SBatoi(silent.ID)
	info.config.Log.Channel = 0
	info.config.Basic.Aliases["calc"] = "roll"
	info.config.Basic.Aliases["calculate"] = "roll"

	sensitive := []string{"add", "addrole", "addwit", "ban", "disable", "dumptables", "echo", "enable", "getconfig", "deleterole", "removerole", "remove", "removewit", "setconfig", "setstatus", "update", "announce", "collections", "addevent", "addbirthday", "autosilence", "silence", "unsilence", "wipewelcome", "new", "addquote", "removequote", "removealias", "delete", "createpoll", "deletepoll", "addoption", "echoembed", "getaudit"}
	modint := SBitoa(info.config.Basic.AlertRole)

	for _, v := range sensitive {
		info.config.Modules.CommandRoles[v] = make(map[string]bool)
		info.config.Modules.CommandRoles[v][modint] = true
	}

	info.config.Modules.CommandDisabled = make(map[string]bool)
	info.config.Modules.Disabled = make(map[string]bool)

	c.DisableModule(info, "bucket")
	c.DisableModule(info, "bored")
	c.DisableModule(info, "markov")
	c.DisableModule(info, "witty")
	c.DisableModule(info, "emote")
	c.DisableModule(info, "spoiler")

	setupSilenceRole(info)
	info.SaveConfig()
	return fmt.Sprintf("```Server configured!\nModerator Role: %s\nMod Channel: %s\nLog Channel: %s```\nNow that you've done basic configuration on Sweetie Bot, here are some additional features you can enable. For additional help, type `"+info.config.Basic.CommandPrefix+"help` for a list of commands and modules, or `"+info.config.Basic.CommandPrefix+"getconfig` with no arguments for a list of configuration options. Using `"+info.config.Basic.CommandPrefix+"help <module>` will display detailed help for that module and all its commands. Using `"+info.config.Basic.CommandPrefix+"getconfig <group>` will display detailed help for all the configuration options in that configuration group. If you're still confused, please check out the readme: https://github.com/blackhole12/sweetiebot/blob/master/README.md \n\n**Bucket**\nIf you'd like to enable Sweetie Bot's bucket, use the command `"+info.config.Basic.CommandPrefix+"enable Bucket`. She defaults to carrying a maximum of 10 items, but you can change this via the `Bucket.MaxItems` option.\n\n**Bored Module**\nIf you'd like Sweetie Bot to perform actions when the chat in a certain channel hasn't been active for a period of time, use `"+info.config.Basic.CommandPrefix+"enable bored` followed by `"+info.config.Basic.CommandPrefix+"setconfig modules.channels bored #yourchannel`, where `#yourchannel` is your general chat channel. The commands she picks from are stored in `bored.commands`. By default, she will quote someone or attempt to throw an item out of her bucket.\n\n**Free Channels**\nIf you like, you can designate a channel to be free from command restrictions, so people can spam silly bot commands to their hearts content. If you had a channel called `#bot` for this, you can disable all command restrictions by using the command ```"+info.config.Basic.CommandPrefix+"setconfig basic.freechannels #bot```.", mod, modchannel, log), false, nil
}
func (c *SetupCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Sets up sweetie bot on this server and restricts all sensitive commands to `Moderator Role`. You must ping each role and channel via `@Role` or `#channel`, you cannot simply input the name of a role or channel. Go to Server Settings -> Roles and select your mod role, then make sure \"Allow anyone to @mention this role\" is checked.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "Moderator Role", Desc: "A role shared by all moderators. It is used to alert moderators and also allows the moderators to bypass command restrictions imposed by certain modules.", Optional: false},
			CommandUsageParam{Name: "Mod Channel", Desc: "Whatever channel the moderators would like to recieve notifications on, such as potential raids, spammers being silenced, etc.", Optional: false},
			CommandUsageParam{Name: "Log Channel", Desc: "An optional channel that recieves log messages about errors and initialization. Usually this channel is only visible to the bot and the moderators.", Optional: true},
		},
	}
}
func (c *SetupCommand) UsageShort() string {
	return "Performs first-time initialization for Sweetie Bot on this server."
}
