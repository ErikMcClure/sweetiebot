package sweetiebot

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"strconv"

	"github.com/bwmarrin/discordgo"
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
		&QuickConfigCommand{},
	}
}

func (w *ConfigModule) Description() string { return "Manages Sweetie Bot's configuration file." }

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
			s = append(s, fmt.Sprintf("\"%v\": [%v items]", k, len(v)))
		}
	case map[string]map[string]bool:
		t, _ := f.Interface().(map[string]map[string]bool)
		for k, v := range t {
			s = append(s, fmt.Sprintf("\"%v\": [%v items]", k, len(v)))
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
				for i := 0; i < f.NumField(); i++ {
					str := f.Type().Field(i).Name
					switch f.Field(i).Interface().(type) {
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
								return c.GetOption(val, info, t.Field(i))
							}
							return c.GetOption(f.Field(j), info, t.Field(i))
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

	return "```That's not a recognized config option! Type !getconfig without any arguments to list all possible config options. Use \".\" to specify which category of options you want - for example, \"Basic.ModChannel\". If the option is a map, you can specify the key as well: \"Help.Rules 1\". Using !getconfig with just a category will list help for that category, e.g. \"!getconfig Basic\".```", false, nil
}
func (c *GetConfigCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Displays a list of available configuration options or their values.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "option", Desc: "The configuration option to display. Use `Help.Rules` to specify a config option in a category. If this is just a category, like `Basic`, lists help information for all config options in that category.", Optional: true},
			CommandUsageParam{Name: "map key", Desc: "If the option is a map, this determines the particular key to display. For example: `!getconfig Help.Rules 1` will return rule 1 in the rules map.", Optional: true},
		},
	}
}
func (c *GetConfigCommand) UsageShort() string {
	return "Returns the current configuration, or a specific option."
}

type QuickConfigCommand struct {
}

func (c *QuickConfigCommand) Name() string {
	return "QuickConfig"
}
func (c *QuickConfigCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if msg.Author.ID != info.Guild.OwnerID {
		return "```Only the owner of this server can use this command!```", false, nil
	}
	if len(args) < 6 {
		return "```You must provide all 6 parameters to this function. Use !help quickconfig and carefully review each one to make sure it is accurate.```", false, nil
	}

	log := StripPing(args[0])
	mod := StripPing(args[1])
	modchannel := StripPing(args[2])
	free := StripPing(args[3])
	silent := StripPing(args[4])
	boredchannel := StripPing(args[5])

	info.config.Log.Channel = SBatoi(log)
	info.config.Basic.AlertRole = SBatoi(mod)
	info.config.Basic.ModChannel = SBatoi(modchannel)
	info.config.Spam.SilentRole = SBatoi(silent)
	info.config.Basic.FreeChannels = make(map[string]bool)
	info.config.Basic.FreeChannels[SBitoa(SBatoi(free))] = true
	info.config.Basic.Aliases["cute"] = "pick cute"
	info.config.Basic.Aliases["calc"] = "roll"
	info.config.Basic.Aliases["calculate"] = "roll"

	sensitive := []string{"add", "addgroup", "addwit", "ban", "disable", "dumptables", "echo", "enable", "getconfig", "purgegroup", "remove", "removewit", "setconfig", "setstatus", "update", "announce", "collections", "addevent", "addbirthday", "autosilence", "silence", "unsilence", "wipewelcome", "new", "addquote", "removequote", "removealias", "delete", "createpoll", "deletepoll", "addoption", "echoembed"}
	modint := SBitoa(info.config.Basic.AlertRole)

	for _, v := range sensitive {
		info.config.Modules.CommandRoles[v] = make(map[string]bool)
		info.config.Modules.CommandRoles[v][modint] = true
	}

	info.config.Modules.CommandDisabled = make(map[string]bool)
	info.config.Modules.Disabled = make(map[string]bool)

	boredint := SBatoi(boredchannel)
	if boredint > 0 {
		info.config.Modules.Channels["bored"] = map[string]bool{SBitoa(boredint): true}
	} else {
		info.config.Modules.Disabled["bored"] = true
	}

	info.SaveConfig()
	warning := "```"
	perms, _ := getAllPerms(info, sb.SelfID)
	if perms&0x00000008 != 0 {
		warning = "\nWARNING: You have given sweetiebot the Administrator role, which implicitely gives her all roles! Sweetie Bot only needs Ban Members, Manage Roles and Manage Messages in order to function correctly." + warning
	}
	if perms&0x00020000 != 0 {
		warning = "\nWARNING: You have given sweetiebot the Mention Everyone role, which means users will be able to abuse her to ping everyone on the server! Sweetie Bot does NOT attempt to filter @\u200Beveryone from her messages!" + warning
	}
	if perms&0x00000004 == 0 {
		warning = "\nWARNING: Sweetiebot cannot ban members spamming the welcome channel without the Ban Members role! (If you do not use this feature, it is safe to ignore this warning)." + warning
	}
	if perms&0x10000000 == 0 {
		warning = "\nWARNING: Sweetiebot cannot silence members or give birthday roles without the Manage Roles role! (If you do not use these features, it is safe to ignore this warning)." + warning
	}
	if perms&0x00002000 == 0 {
		warning = "\nWARNING: Sweetiebot cannot delete messages without the Manage Messages role!" + warning
	}
	return "```Server configured! \nLog Channel: " + log + "\nModerator Role: " + mod + "\nMod Channel: " + modchannel + "\nFree Channel: " + free + "\nSilent Role: " + silent + "\nBored Channel: " + boredchannel + warning, false, nil
}
func (c *QuickConfigCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Quickly performs basic configuration on the server and restricts all sensitive commands to `Moderator Role`, then enables all commands and all modules. If `bored channel` is not zero, it restricts the bored module to that channel. Otherwise it disables the bored module to prevent the bot from spamming inactive channels. You must ping each role and channel via `@Role` or `#channel`, you cannot simply input the name of a role or channel.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "Log Channel", Desc: "The channel that recieves log messages about errors and initialization. Usually this channel is only visible to the bot and the moderators.", Optional: false},
			CommandUsageParam{Name: "Moderator Role", Desc: "A role shared by all moderators. It is used to alert moderators and also allows the moderators to bypass command restrictions imposed by certain modules.", Optional: false},
			CommandUsageParam{Name: "Mod Channel", Desc: "Whatever channel the moderators would like to recieve notifications on, such as potential raids, spammers being silenced, etc.", Optional: false},
			CommandUsageParam{Name: "Free Channel", Desc: "A list of channel IDs that are excluded from rate limiting. If you have a `#bot` channel for spamming the bot, add it here.", Optional: false},
			CommandUsageParam{Name: "Silent Role", Desc: "A role with all permissions disabled. This is the role assigned to spammers, which allows the moderation team to review what happened and ban them if necessary.", Optional: false},
			CommandUsageParam{Name: "Bored Channel", Desc: "Either the channel that sweetiebot will post bored messages on, or 0, which will disable the bored module. *This is not a real config option*, to manually set this option, use `!setconfig Modules.Channels bored #channelname`", Optional: false},
		},
	}
}
func (c *QuickConfigCommand) UsageShort() string { return "Quickly performs basic configuration." }
