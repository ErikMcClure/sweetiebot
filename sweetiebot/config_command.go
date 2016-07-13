package sweetiebot

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type SetConfigCommand struct {
}

func (c *SetConfigCommand) Name() string {
	return "SetConfig"
}
func (c *SetConfigCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```No configuration parameter to look for!```", false
	}
	if len(args) < 2 {
		return "```No value to set!```", false
	}
	n, ok := info.SetConfig(args[0], args[1], args[2:]...)
	info.SaveConfig()
	if ok {
		return "```Successfully set " + args[0] + " to " + n + ".```", false
	}
	return "```" + n + "```", false
}
func (c *SetConfigCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[parameter] [value]\n!SetConfig [list parameter] [value] [value 2] [etc...]\n!SetConfig [map parameter] [key] [value]", "Attempts to set the configuration value matching [parameter] (not case-sensitive) to [value]. Will only save the new configuration if it succeeds, and returns the new value upon success. If the parameter is a list, it will accept multiple new values. To set a value with a space in it, surround it with quotes, \"like so\". If the parameter is a map, it will accept two values: the first is the key, and the second is the value of that key. If the parameter is a maplist, the first value is the key, and all other values make up the list of values that key is set to. For more information, see: https://github.com/blackhole12/sweetiebot#configuration")
}
func (c *SetConfigCommand) UsageShort() string {
	return "Sets a config value and saves the new configuration."
}

type GetConfigCommand struct {
}

func (c *GetConfigCommand) Name() string {
	return "GetConfig"
}
func (c *GetConfigCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	t := reflect.ValueOf(&info.config).Elem()
	n := t.NumField()
	if len(args) < 1 {
		s := make([]string, 0, n)
		for i := 0; i < n; i++ {
			str := strings.ToLower(t.Type().Field(i).Name)
			switch t.Field(i).Interface().(type) {
			case []uint64:
				str += " [list]"
			case map[string]string:
				str += " [map]"
			case map[string]int64:
				str += " [map]"
			case map[string]bool:
				str += " [list]"
			case map[string]map[string]bool:
				str += " [maplist]"
			}
			s = append(s, str)
		}
		return "```Choose a config option to display:\n" + strings.Join(s, "\n") + "```", false
	}
	arg := args[0]
	for i := 0; i < n; i++ {
		if strings.ToLower(t.Type().Field(i).Name) == arg {
			data, err := json.Marshal(t.Field(i).Interface())
			s := string(data)
			s = strings.Replace(s, "`", "", -1)
			s = strings.Replace(s, "[](/", "[\u200B](/", -1)
			s = strings.Replace(s, "http://", "http\u200B://", -1)
			s = strings.Replace(s, "https://", "https\u200B://", -1)
			if err == nil {
				return "```" + s + "```", false
			}
			info.log.Log("JSON error: ", err.Error())
			return "```Failed to marshal JSON :C```", false
		}
	}

	return "```That's not a recognized config option! Type !getconfig without any arguments to list all possible config options```", false
}
func (c *GetConfigCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "", "Returns the current configuration as a JSON string.")
}
func (c *GetConfigCommand) UsageShort() string { return "Returns the current configuration." }

type QuickConfigCommand struct {
}

func (c *QuickConfigCommand) Name() string {
	return "QuickConfig"
}
func (c *QuickConfigCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if msg.Author.ID != info.Guild.OwnerID {
		return "```Only the owner of this server can use this command!```", false
	}
	if len(args) < 6 {
		return "```You must provide all 6 parameters to this function. Use !help quickconfig and carefully review each one to make sure it is accurate.```", false
	}

	log := StripPing(args[0])
	mod := StripPing(args[1])
	modchannel := StripPing(args[2])
	free := StripPing(args[3])
	silent := StripPing(args[4])
	boredchannel := StripPing(args[5])

	info.config.LogChannel = SBatoi(log)
	info.config.AlertRole = SBatoi(mod)
	info.config.ModChannel = SBatoi(modchannel)
	info.config.FreeChannels = make(map[string]bool)
	info.config.FreeChannels[strconv.FormatUint(SBatoi(free), 10)] = true

	sensitive := []string{"add", "addgroup", "addwit", "ban", "disable", "dumptables", "echo", "enable", "getconfig", "purgegroup", "remove", "removewit", "setconfig", "setstatus", "update", "announce"}
	modint := strconv.FormatUint(info.config.AlertRole, 10)

	for _, v := range sensitive {
		info.config.Command_roles[v] = make(map[string]bool)
		info.config.Command_roles[v][modint] = true
	}

	info.config.Command_disabled = make(map[string]bool)
	info.config.Module_disabled = make(map[string]bool)

	boredint := SBatoi(boredchannel)
	if boredint > 0 {
		info.config.Module_channels["bored"] = map[string]bool{strconv.FormatUint(boredint, 10): true}
	} else {
		info.config.Module_disabled["bored"] = true
	}

	info.SaveConfig()
	return "```Server configured! \nLog Channel: " + log + "\nModerator Role: " + mod + "\nMod Channel: " + modchannel + "\nFree Channel: " + free + "\nSilent Role: " + silent + "\nBored Channel: " + boredchannel + "```", false
}
func (c *QuickConfigCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[Log Channel] [Moderator Role] [Mod Channel] [Free Channel] [Silent Role] [Bored Channel]", "Quickly performs basic configuration on the server and restricts all sensitive commands to [Moderator Role], then enables all commands and all modules. If [bored channel] is not zero, it restricts the bored module to that channel. Otherwise it disables the bored module to prevent the bot from spamming inactive channels.")
}
func (c *QuickConfigCommand) UsageShort() string { return "Quickly performs basic configuration." }
