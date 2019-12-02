package sweetiebot

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"4d63.com/tz"
	"github.com/erikmcclure/discordgo"
)

type ModuleID string
type CommandID string
type TimeLocation string

// BotConfig lists all bot configuration options, grouped into structs
type BotConfig struct {
	Version     int   `json:"version"`
	LastVersion int   `json:"lastversion"`
	SetupDone   bool  `json:"setupdone"`
	Expires     int64 `json:"expires"`
	Basic       struct {
		IgnoreInvalidCommands bool                    `json:"ignoreinvalidcommands"`
		Importable            bool                    `json:"importable"`
		ModRole               DiscordRole             `json:"modrole"`
		ModChannel            DiscordChannel          `json:"modchannel"`
		FreeChannels          map[DiscordChannel]bool `json:"freechannels"`
		BotChannel            DiscordChannel          `json:"botchannel"`
		Aliases               map[string]string       `json:"aliases"`
		ListenToBots          bool                    `json:"listentobots"`
		CommandPrefix         string                  `json:"commandprefix"`
		SilenceRole           DiscordRole             `json:"silencerole"`
		MemberRole            DiscordRole             `json:"memberrole"`
	} `json:"basic"`
	Modules struct {
		Channels           map[ModuleID]map[DiscordChannel]bool  `json:"modulechannels"`
		Disabled           map[ModuleID]bool                     `json:"moduledisabled"`
		CommandRoles       map[CommandID]map[DiscordRole]bool    `json:"commandroles"`
		CommandChannels    map[CommandID]map[DiscordChannel]bool `json:"commandchannels"`
		CommandLimits      map[CommandID]int64                   `json:"Commandlimits"`
		CommandDisabled    map[CommandID]bool                    `json:"commanddisabled"`
		CommandPerDuration int                                   `json:"commandperduration"`
		CommandMaxDuration int64                                 `json:"commandmaxduration"`
	} `json:"modules"`
	Spam struct {
		ImagePressure      float32                    `json:"imagepressure"`
		PingPressure       float32                    `json:"pingpressure"`
		LengthPressure     float32                    `json:"lengthpressure"`
		RepeatPressure     float32                    `json:"repeatpressure"`
		LinePressure       float32                    `json:"linepressure"`
		BasePressure       float32                    `json:"basepressure"`
		PressureDecay      float32                    `json:"pressuredecay"`
		MaxPressure        float32                    `json:"maxpressure"`
		MaxChannelPressure map[DiscordChannel]float32 `json:"maxchannelpressure"`
		MaxRemoveLookback  int                        `json:"MaxSpamRemoveLookback"`
		IgnoreRole         DiscordRole                `json:"ignorerole"`
		RaidTime           int64                      `json:"maxraidtime"`
		RaidSize           int                        `json:"raidsize"`
		RaidSilence        int                        `json:"raidsilence"`
		LockdownDuration   int                        `json:"lockdownduration"`
		SilenceTimeout     int64                      `json:"silencetimeout"`
	} `json:"spam"`
	Users struct {
		TimezoneLocation TimeLocation         `json:"timezonelocation"`
		WelcomeChannel   DiscordChannel       `json:"welcomechannel"`
		JailChannel      DiscordChannel       `json:"jailchannel"`
		WelcomeMessage   string               `json:"welcomemessage"`
		SilenceMessage   string               `json:"silencemessage"`
		Roles            map[DiscordRole]bool `json:"userroles"`
		NotifyChannel    DiscordChannel       `json:"joinchannel"`
		TrackUserLeft    bool                 `json:"trackuserleft"`
		NewUserRole      DiscordRole          `json:"newuserrole"`
		NewUserDuration  int64                `json:"newuserduration"`
	} `json:"users"`
	Bucket struct {
		MaxItems       int             `json:"maxbucket"`
		MaxItemLength  int             `json:"maxbucketlength"`
		MaxFightHP     int             `json:"maxfighthp"`
		MaxFightDamage int             `json:"maxfightdamage"`
		Items          map[string]bool `json:"items"`
	} `json:"bucket"`
	Markov struct {
		MaxPMlines     int  `json:"maxpmlines"`
		MaxLines       int  `json:"maxquotelines"`
		DefaultLines   int  `json:"defaultmarkovlines"`
		UseMemberNames bool `json:"usemembernames"`
	} `json:"markov"`
	Filter struct {
		Filters   map[string]map[string]bool         `json:"filters"`
		Channels  map[string]map[DiscordChannel]bool `json:"channels"`
		Responses map[string]string                  `json:"responses"`
		Templates map[string]string                  `json:"templates"`
		Pressure  map[string]float32                 `json:"pressure"`
	} `json:"filter"`
	Bored struct {
		Cooldown int64           `json:"maxbored"`
		Exponent float64         `json:"exponent"`
		Commands map[string]bool `json:"boredcommands"`
	}
	Information struct {
		Rules             map[int]string `json:"rules"`
		HideNegativeRules bool           `json:"hidenegativerules"`
	} `json:"help"`
	Log struct {
		Cooldown int64          `json:"maxerror"`
		Channel  DiscordChannel `json:"logchannel"`
	} `json:"log"`
	Witty struct {
		Responses map[string]string `json:"witty"`
		Cooldown  int64             `json:"maxwit"`
	} `json:"Wit"`
	Scheduler struct {
		BirthdayRole DiscordRole `json:"birthdayrole"`
	} `json:"scheduler"`
	Miscellaneous struct {
		MaxSearchResults int `json:"maxsearchresults"`
	} `json:"misc"`
	Status struct {
		Cooldown int             `json:"statusdelaytime"`
		Lines    map[string]bool `json:"lines"`
	} `json:"status"`
	Quote struct {
		Quotes map[DiscordUser][]string `json:"quotes"`
	} `json:"quote"`
	Counters struct {
		Map          map[string]int64  `json:"map"`
		Descriptions map[string]string `json:"counterdescriptions"`
	} `json:"counters"`
}

// ConfigHelp is a map of help strings for the configuration options above
var ConfigHelp = map[string]map[string]string{
	"basic": {
		"ignoreinvalidcommands": "If true, the bot won't display an error if a nonsensical command is used. This helps reduce confusion with other bots that use the same prefix.",
		"importable":            "If true, the collections on this server will be importable into another server.",
		"modrole":               "This is intended to point at a moderator role shared by all admins and moderators of the server for notification purposes.",
		"modchannel":            "This should point at the hidden moderator channel, or whatever channel moderates want to be notified on.",
		"freechannels":          "This is a list of all channels that are exempt from bot command rate limiting. Usually set to the dedicated `#botabuse` channel in a server. Does NOT affect anti-spam! To exclude anti-spam from a channel, use `!setconfig modules.channels spam ! #yourchannel`.",
		"botchannel":            "This allows you to designate a particular channel to point users if they are trying to run too many commands at once. Usually this channel will also be included in `basic.freechannels`. Again, this is for bot commands, not general spamming!",
		"aliases":               "Can be used to redirect commands, such as making `!listgroup` call the `!listgroups` command. Useful for making shortcuts.\n\nExample: `!setconfig basic.aliases kawaii pick cute` sets an alias mapping `!kawaii arg1...` to `!pick cute arg1...`, preserving all arguments that are passed to the alias.",
		"listentobots":          "If true, processes messages from other bots and allows them to run commands. Bots can never trigger anti-spam. Defaults to false.",
		"commandprefix":         "Determines the SINGLE ASCII CHARACTER prefix used to denote bot commands. You can't set it to an emoji or any weird foreign character. The default is `!`. If this is set to an invalid value, it defaults to `!`.",
		"silencerole":           "This should be a role with no permissions, so the bot can quarantine potential spammers without banning them. The bot usually manages this role for you, so you should almost never touch this value.",
		"memberrole":            "This should be a role with all permissions that the everyone role would normally have. You shouldn't touch this value, use the !SetMemberRole command to manage it instead.",
	},
	"modules": {
		"commandroles":       "A map of which roles are allowed to run which command. If no mapping exists, everyone can run the command.",
		"commandchannels":    "A map of which channels commands are allowed to run on. No entry means a command can be run anywhere. If `!` is included as a channel, it switches from a whitelist to a blacklist, enabling you to exclude certain channels instead of allow certain channels.",
		"commandlimits":      "A map of timeouts for commands. A value of 30 means the command can't be used more than once every 30 seconds.",
		"commanddisabled":    "A list of disabled commands. Disabled commands can still be run by administrators, but can't be run by the bot and will not function as a bored command.",
		"commandperduration": "Maximum number of commands that can be run within `commandmaxduration` seconds. Default: 3",
		"commandmaxduration": "Default: 20. This means that by default, at most 3 commands can be run every 20 seconds.",
		"disabled":           "A list of disabled modules. This disables any hooks the modules normally process, and also disables all commands inside that module (although commands can be selectively re-enabled without enabling the module).",
		"channels":           "A mapping of what channels a given module can operate on. If no mapping is given, a module operates on all channels. If `!` is included as a channel, it switches from a whitelist to a blacklist, enabling you to exclude certain channels instead of allow certain channels. Restricting a module to a channel DOES NOT restrict its commands to that channel.",
	},
	"spam": {
		"imagepressure":      "Additional pressure generated by each image, link or attachment in a message. Defaults to (MaxPressure - BasePressure) / 6 = 8.3, instantly silencing anyone posting 6 or more links at once.",
		"repeatpressure":     "Additional pressure generated by a message that is identical to the previous message sent (ignores case). Defaults to BasePressure, effectively doubling the pressure penalty for repeated messages.",
		"pingpressure":       "Additional pressure generated by each unique ping in a message. Defaults to (MaxPressure - BasePressure) / 20 = 2.5, instantly silencing anyone pinging 20 or more people at once.",
		"lengthpressure":     "Additional pressure generated by each individual character in the message. Discord allows messages up to 2000 characters in length. Defaults to (MaxPressure - BasePressure) / 8000 = 0.00625, silencing anyone posting 3 huge messages at the same time.",
		"linepressure":       "Additional pressure generated by each newline in the message. Defaults to (MaxPressure - BasePressure) / 70 = 0.714, silencing anyone posting more than 70 newlines in a single message",
		"basepressure":       "The base pressure generated by sending a message, regardless of length or content. Defaults to 10",
		"maxpressure":        "The maximum pressure allowed. If a user's pressure exceeds this amount, they will be silenced. Defaults to 60, which is intended to ban after a maximum of 6 short messages sent in rapid succession.",
		"maxchannelpressure": "Per-channel pressure override. If a channel's pressure is specified in this map, it will override the global maxpressure setting.",
		"pressuredecay":      "The number of seconds it takes for a user to lose Spam.BasePressure from their pressure amount. Defaults to 2.5, so after sending 3 messages, it will take 7.5 seconds for their pressure to return to 0.",
		"maxremovelookback":  "Number of seconds back the bot should delete messages of a silenced user on the channel they spammed on. If set to 0, the bot will only delete the message that caused the user to be silenced. If less than 0, the bot won't delete any messages.",
		"ignorerole":         "If set, the bot will exclude anyone with this role from spam detection. Use with caution. Does NOT prevent people from sending the bot commands.",
		"raidtime":           "In order to trigger a raid alarm, at least `spam.raidsize` people must join the chat within this many seconds of each other.",
		"raidsize":           "Specifies how many people must have joined the server within the `spam.raidtime` period to qualify as a raid.",
		"raidsilence":        "Gets the current raidsilence state. Use the `!RaidSilence` command to set this.",
		"lockdownduration":   "Determines how long the server's verification mode will temporarily be increased to tableflip levels after a raid is detected. If set to 0, disables lockdown entirely.",
		"silencetimeout":     "If greater than 0, any members silenced by the bot (not by the `!silence` command) will be automatically unsilenced after this many seconds. This includes anyone silenced during a raid.",
	},
	"bucket": {
		"maxitems":       "Determines the maximum number of items that can be carried in the bucket. If set to 0, the bucket is disabled.",
		"maxitemlength":  "Determines the maximum length of a string that can be added to the bucket.",
		"maxfighthp":     "Maximum HP of the randomly generated enemy for the `!fight` command.",
		"maxfightdamage": "Maximum amount of damage a randomly generated weapon can deal for the `!fight` command.",
		"items":          "List of items in the bucket.",
	},
	"markov": {
		"maxpmlines":     "This is the maximum number of lines a response can be before its automatically sent as a PM to avoid cluttering the chat. Default: 5",
		"maxlines":       "Maximum number of lines the `!episodequote` command can be given.",
		"defaultlines":   "Number of lines for the markov chain to spawn when not given a line count.",
		"usemembernames": "Use member names instead of random pony names.",
	},
	"users": {
		"timezonelocation": "Sets the timezone location of the server itself. When no user timezone is available, the bot will use this.",
		"welcomechannel":   "If set to a channel ID, the bot will treat this channel as a \"quarantine zone\" for new members that haven't had their Member role set. If RaidSilence is enabled, new users will be sent to this channel.",
		"jailchannel":      "If set to a channel ID, the bot will treat this channel as a \"quarantine zone\" for silenced members. This can be the same as the welcome channel, or it can be a different one.",
		"welcomemessage":   "If RaidSilence is enabled, this message will be sent to a new user upon joining.",
		"silencemessage":   "This message will be sent to users that have been silenced by the `!silence` command.",
		"roles":            "A list of all user-assignable roles. Manage it via !addrole and !removerole",
		"notifychannel":    "If set to a channel ID other than zero, sends a message to that channel whenever a new user joins the server.",
		"trackuserleft":    "If true, tracks users that leave the server if notifychannel is set.",
		"newuserrole":      "If this is set and `newuserduration` is nonzero, this role is given to any new user that joins, regardless of raid settings. It is automatically removed after `newuserduration` seconds.",
		"newuserduration":  "The number of seconds a new user will have the `newuserrole` role. If zero, `newuserrole` won't be given to any new user.",
	},
	"filter": {
		"filters":   "A collection of word lists for each filter. These are combined into a single regex of the form `(word1|word2|etc...)`, depending on the filter template.",
		"channels":  "A collection of channel exclusions for each filter.",
		"responses": "The response message sent by each filter when triggered. If this is set to `!`, the bot won't respond AND she won't delete the message, only the pressure will be added.",
		"templates": "The template used to construct the regex. `%%` is replaced with `(word1|word2|etc...)` using the filter's word list. Example: `\\[\\]\\(\\/r?%%[-) \"]` is transformed into `\\[\\]\\(\\/r?(word1|word2)[-) \"]`",
		"pressure":  "The amount of pressure added to the user when the filter is triggered (defaults to 0).",
	},
	"bored": {
		"cooldown": "The bored cooldown timer, in seconds. This is the length of time a channel must be inactive before a bored message is posted.",
		"exponent": "The exponential increase in time between bored posts if no one other than the bot is posting. A value of about 1.41 doubles the amount of time between bored posts until someone else says something. Defaults to 1, which means no increase. The actual formula is (2^n + n).",
		"commands": "This determines what commands will be run when nothing has been said in a channel for a while. One command will be chosen from this list at random.\n\nExample: `!setconfig bored.commands !drop \"!pick bored\"`",
	},
	"information": {
		"rules":             "Contains a list of numbered rules. The numbers do not need to be contiguous, and can be negative.",
		"hidenegativerules": "If true, `!rules -1` will display a rule at index -1, but `!rules` will not. This is useful for joke rules or additional rules that newcomers don't need to know about.",
	},
	"log": {
		"channel":  "This is the channel where log output is sent.",
		"cooldown": "The cooldown time to display an error message, in seconds, intended to prevent the bot from spamming itself. Default: 4",
	},
	"witty": {
		"responses": "Stores the replies used by the Witty module and must be configured using `!addwit` or `!removewit`",
		"cooldown":  "The cooldown time for the witty module. At least this many seconds must have passed before the bot will make another witty reply.",
	},
	"scheduler": {
		"birthdayrole": " This is the role given to members on their birthday.",
	},
	"miscellaneous": {
		"maxsearchresults": "Maximum number of search results that can be requested at once.",
	},
	"spoiler": {
		"channels": "A list of channels that are exempt from the spoiler rules.",
	},
	"status": {
		"cooldown": "Number of seconds the bot waits before changing its status to a string picked randomly from the `status` collection.",
		"lines":    "List of possible status messages that the bot can have.",
	},
	"quote": {
		"quotes": "This is a map of quotes, which should be managed via `!addquote` and `!removequote`.",
	},
	"counters": {
		"map":          "This is a map of counters, which should be managed via `!addcounter` and `!removecounter`.",
		"descriptions": "These are descriptions for each counter in map, which should be managed via `!addcounter` and `!removecounter`.",
	},
}

func getConfigHelp(module string, option string) (string, bool) {
	x, ok := ConfigHelp[strings.ToLower(module)]
	if !ok {
		return "", false
	}
	s, b := x[strings.ToLower(option)]
	return s, b
}

// ConfigVersion is the latest version of the config file
var ConfigVersion = 30

// DefaultConfig returns a default BotConfig struct. We can't define this as a variable because you can't initialize nested structs in a sane way in Go
func DefaultConfig() *BotConfig {
	config := &BotConfig{
		Version:     ConfigVersion,
		LastVersion: BotVersion.Integer(),
		SetupDone:   false,
	}
	config.Basic.IgnoreInvalidCommands = false
	config.Basic.Importable = false
	config.Basic.CommandPrefix = "!"
	config.Modules.CommandPerDuration = 3
	config.Modules.CommandMaxDuration = 15
	config.Spam.MaxPressure = 60
	config.Spam.BasePressure = 10
	config.Spam.ImagePressure = (config.Spam.MaxPressure - config.Spam.BasePressure) / 6
	config.Spam.PingPressure = (config.Spam.MaxPressure - config.Spam.BasePressure) / 20
	config.Spam.LengthPressure = (config.Spam.MaxPressure - config.Spam.BasePressure) / 8000
	config.Spam.RepeatPressure = config.Spam.BasePressure
	config.Spam.LinePressure = (config.Spam.MaxPressure - config.Spam.BasePressure) / 70
	config.Spam.PressureDecay = 2.5
	config.Spam.MaxRemoveLookback = 4
	config.Spam.RaidTime = 240
	config.Spam.RaidSize = 4
	config.Spam.RaidSilence = 1 // Default to raid mode
	config.Spam.LockdownDuration = 120
	config.Bucket.MaxItems = 10
	config.Bucket.MaxItemLength = 100
	config.Bucket.MaxFightHP = 300
	config.Bucket.MaxFightDamage = 60
	config.Markov.MaxPMlines = 5
	config.Markov.MaxLines = 30
	config.Markov.DefaultLines = 5
	config.Markov.UseMemberNames = true
	config.Bored.Cooldown = 500
	config.Bored.Exponent = 1
	config.Log.Cooldown = 4
	config.Witty.Cooldown = 180
	config.Miscellaneous.MaxSearchResults = 10
	config.Status.Cooldown = 3600

	return config
}

// FixRequest takes a request that is not fully qualified and attempts to find a fully qualified version
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
	for k := range list {
		list[k] += "." + args[0]
	}
	return "", errors.New("Could be any of the following:\n" + strings.Join(list, "\n"))
}

func setConfigValue(f reflect.Value, value string, info *GuildInfo) error {
	switch f.Interface().(type) {
	case string:
		if value == "\"\"" {
			f.SetString("")
		} else {
			f.SetString(value)
		}
	case TimeLocation:
		value = strings.TrimSpace(value)
		_, err := tz.LoadLocation(value)
		if err != nil {
			return fmt.Errorf("%s is not a valid timezone location! The location is CASE-SENSITIVE, use !settimezone to find the exact string you want.", value)
		}
		f.SetString(value)
	case DiscordRole:
		g, _ := info.GetGuild()
		s, err := ParseRole(value, g)
		if err != nil {
			return err
		}
		f.SetString(s.String())
	case DiscordChannel:
		g, _ := info.GetGuild()
		s, err := ParseChannel(value, g)
		if err != nil {
			return err
		}
		f.SetString(s.String())
	case DiscordUser:
		s, err := ParseUser(value, info)
		if err != nil {
			return err
		}
		f.SetString(s.String())
	case ModuleID:
		value = strings.ToLower(value)
		for _, v := range info.Modules {
			if value == strings.ToLower(v.Name()) {
				f.SetString(value)
				return nil
			}
		}
		return fmt.Errorf("%s is not a module name!", value)
	case CommandID:
		value = strings.ToLower(value)
		if _, ok := info.commands[CommandID(value)]; !ok {
			return fmt.Errorf("%s is not a command name!", value)
		}
		f.SetString(value)
	case int, int8, int16, int32, int64:
		k, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		f.SetInt(k)
	case uint, uint8, uint16, uint32, uint64:
		k, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		f.SetUint(k)
	case float32, float64:
		k, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return err
		}
		f.SetFloat(k)
	}
	return nil
}

func deleteFromMapReflect(f reflect.Value, k reflect.Value) string {
	if (f.MapIndex(k) == reflect.Value{}) {
		return fmt.Sprint(k.Interface()) + " does not exist."
	}
	f.SetMapIndex(k, reflect.Value{})
	return "Deleted " + fmt.Sprint(k.Interface())
}

func setConfigKeyValue(f reflect.Value, key string, value string, info *GuildInfo) (string, bool) {
	k := reflect.New(f.Type().Key()).Elem()
	if err := setConfigValue(k, key, info); err != nil {
		return "Key error: " + err.Error(), false
	}
	if f.IsNil() {
		f.Set(reflect.MakeMap(f.Type()))
	}
	if len(value) == 0 {
		return deleteFromMapReflect(f, k), false
	}
	v := reflect.New(f.Type().Elem()).Elem()
	if err := setConfigValue(v, value, info); err != nil {
		return "Value error: " + err.Error(), false
	}

	f.SetMapIndex(k, v)
	return fmt.Sprintf("%v: %v", k.Interface(), v.Interface()), true
}

func setConfigList(f reflect.Value, values []string, info *GuildInfo) (string, bool) {
	switch f.Kind() {
	case reflect.Slice:
		f.Set(reflect.MakeSlice(f.Type(), 0, len(values)))
		if len(values) > 0 && len(values[0]) > 0 {
			for _, value := range values {
				v := reflect.New(f.Type().Elem()).Elem()
				if err := setConfigValue(v, value, info); err != nil {
					return "Value error: " + err.Error(), false
				}
				f.Set(reflect.Append(f, v))
			}
		}
		return fmt.Sprint(f.Interface()), true
	case reflect.Map:
		if f.Type().Elem() != reflect.TypeOf(true) {
			return "Map sent into list function!", false
		}
		f.Set(reflect.MakeMap(f.Type()))
		stripped := []string{}
		if len(values) > 0 && len(values[0]) > 0 {
			for _, value := range values {
				v := reflect.New(f.Type().Key()).Elem()
				if err := setConfigValue(v, value, info); err != nil {
					return "Value error: " + err.Error(), false
				}
				f.SetMapIndex(v, reflect.ValueOf(true))
				stripped = append(stripped, fmt.Sprint(v.Interface()))
			}
		}
		return "[" + strings.Join(stripped, ", ") + "]", true
	}
	return "Unknown list type!", false
}

func setConfigMapList(f reflect.Value, key string, values []string, info *GuildInfo) (string, bool) {
	if f.IsNil() {
		f.Set(reflect.MakeMap(f.Type()))
	}
	if len(key) == 0 {
		return "No key specified", false
	}
	k := reflect.New(f.Type().Key()).Elem()
	if err := setConfigValue(k, key, info); err != nil {
		return "Key error: " + err.Error(), false
	}
	if len(values) == 0 {
		return deleteFromMapReflect(f, k), false
	}

	v := reflect.New(f.Type().Elem()).Elem()
	s, ok := setConfigList(v, values, info)
	if !ok {
		return s, false
	}
	f.SetMapIndex(k, v)
	return fmt.Sprintf("%v: %s", k, s), true
}

// SetConfig sets the given config option with the given value along with any extra parameters
func (config *BotConfig) SetConfig(info *GuildInfo, args []string, indices []int, message string) (string, bool) {
	name := args[0]
	names := strings.SplitN(strings.ToLower(name), ".", 3)
	t := reflect.ValueOf(config).Elem()
	for i := 0; i < t.NumField(); i++ {
		if strings.ToLower(t.Type().Field(i).Name) == names[0] {
			if len(names) < 2 {
				return "Can't set a configuration category! Use \"Category.Option\" to set a specific option.", false
			}
			switch t.Field(i).Kind() {
			case reflect.Struct:
				for j := 0; j < t.Field(i).NumField(); j++ {
					if strings.ToLower(t.Field(i).Type().Field(j).Name) == names[1] {
						f := t.Field(i).Field(j)
						switch f.Interface().(type) {
						case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, float32, float64, uint64, DiscordChannel, DiscordRole, DiscordUser, TimeLocation:
							value := ""
							if len(indices) > 1 {
								value = message[indices[1]:]
							}
							if err := setConfigValue(f, value, info); err != nil {
								return "Error: " + err.Error(), false
							}
						case map[DiscordChannel]bool, map[string]bool, map[DiscordRole]bool, map[CommandID]bool, map[ModuleID]bool:
							return setConfigList(f, args[1:], info)
						case bool:
							if len(indices) < 2 {
								return "No value parameter given", false
							}
							switch strings.ToLower(message[indices[1]:]) {
							case "true":
								f.SetBool(true)
							case "false":
								f.SetBool(false)
							default:
								return name + " must be set to either 'true' or 'false'", false
							}
						case map[string]string, map[CommandID]int64, map[DiscordChannel]float32, map[int]string, map[string]float32, map[string]int64:
							if len(indices) < 2 {
								return "No key parameter given", false
							}
							value := ""
							if len(indices) > 2 {
								value = message[indices[2]:]
							}
							return setConfigKeyValue(f, strings.ToLower(args[1]), value, info)
						case map[string]map[DiscordChannel]bool, map[CommandID]map[DiscordRole]bool, map[string]map[string]bool, map[DiscordUser][]string, map[CommandID]map[DiscordChannel]bool, map[ModuleID]map[DiscordChannel]bool:
							if len(indices) < 2 {
								return "No key parameter given", false
							}
							return setConfigMapList(f, strings.ToLower(args[1]), args[2:], info)
						default:
							return "That config option has an unknown type!", false
						}
						return fmt.Sprint(f.Interface()), true
					}
				}
			default:
				return "Not a configuration category!", false
			}
		}
	}
	return "Could not find configuration parameter " + name + "!", false
}

func getConfigValue(f reflect.Value, state *discordgo.State, guild string) string {
	switch f.Interface().(type) {
	case DiscordRole:
		if r, err := state.Role(guild, f.String()); err == nil {
			return "@" + r.Name
		}
	case DiscordChannel:
		if ch, err := state.Channel(f.String()); err == nil {
			return "#" + ch.Name
		}
	case DiscordUser:
		if m, err := state.Member(guild, f.String()); err == nil {
			if len(m.Nick) > 0 {
				return m.Nick
			}
			return m.User.Username
		}
		//case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
	}
	return fmt.Sprint(f.Interface())
}

type valueArray []reflect.Value

func (f valueArray) Len() int {
	return len(f)
}

func (f valueArray) Less(i, j int) bool {
	return strings.Compare(fmt.Sprint(f[i].Interface()), fmt.Sprint(f[j].Interface())) < 0
}

func (f valueArray) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func getConfigList(f reflect.Value, state *discordgo.State, guild string) (s []string) {
	switch f.Kind() {
	case reflect.Slice:
		for i := 0; i < f.Len(); i++ {
			s = append(s, getConfigValue(f.Index(i), state, guild))
		}
		sort.Strings(s)
	case reflect.Map:
		keys := f.MapKeys()
		if f.Type().Elem() == reflect.TypeOf(true) {
			for _, key := range keys {
				s = append(s, getConfigValue(key, state, guild))
			}
			sort.Strings(s)
		} else {
			sort.Sort(valueArray(keys))
			for _, key := range keys {
				s = append(s, "\""+getConfigValue(key, state, guild)+"\": "+getConfigValue(f.MapIndex(key), state, guild))
			}
		}
	}
	return
}

func getConfigMapList(f reflect.Value, state *discordgo.State, guild string) (s []string) {
	keys := f.MapKeys()
	sort.Sort(valueArray(keys))
	for _, key := range keys {
		v := f.MapIndex(key)
		k := getConfigValue(key, state, guild)

		if v.Len() == 1 {
			s = append(s, fmt.Sprintf("\"%s\": %s", k, strings.Join(getConfigList(v, state, guild), ", ")))
		} else {
			s = append(s, fmt.Sprintf("\"%s\": [%v items]", k, v.Len()))
		}
	}
	return
}

func (config *BotConfig) GetConfig(f reflect.Value, state *discordgo.State, guild string) (s []string) {
	switch f.Interface().(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, float32, float64, uint64, DiscordChannel, DiscordRole, DiscordUser, ModuleID, CommandID, bool, TimeLocation:
		s = append(s, getConfigValue(f, state, guild))
	case map[DiscordChannel]bool, map[string]bool, map[DiscordRole]bool, map[string]string, map[CommandID]int64, map[DiscordChannel]float32, map[int]string, map[CommandID]bool, map[ModuleID]bool, map[string]float32, map[string]int64:
		s = getConfigList(f, state, guild)
	case map[string]map[DiscordChannel]bool, map[CommandID]map[DiscordRole]bool, map[string]map[string]bool, map[DiscordUser][]string, map[CommandID]map[DiscordChannel]bool, map[ModuleID]map[DiscordChannel]bool:
		s = getConfigMapList(f, state, guild)
	default:
		data, err := json.Marshal(f.Interface())
		if err != nil {
			s = append(s, "[JSON Error]")
		} else {
			s = append(s, string(data))
		}
	}
	return
}

// IsModuleDisabled returns a string if a module is disabled
func (config *BotConfig) IsModuleDisabled(module Module) string {
	_, ok := config.Modules.Disabled[ModuleID(strings.ToLower(module.Name()))]
	if ok {
		return " [disabled]"
	}
	return ""
}

// IsCommandDisabled returns a string if a command is disabled
func (config *BotConfig) IsCommandDisabled(command Command) (str string) {
	_, disabled := config.Modules.CommandDisabled[CommandID(strings.ToLower(command.Info().Name))]
	if disabled {
		str = " [disabled]"
	}
	return
}

// FillConfig ensures root maps are not nil
func (config *BotConfig) FillConfig() {

	t := reflect.ValueOf(config).Elem()
	n := t.NumField()

	for i := 0; i < n; i++ {
		switch t.Field(i).Kind() {
		case reflect.Struct:
			f := t.Field(i)
			for j := 0; j < f.NumField(); j++ {
				switch f.Field(j).Kind() {
				case reflect.Map:
					if f.Field(j).Len() == 0 {
						f.Field(j).Set(reflect.MakeMap(f.Field(j).Type()))
					}
				}
			}
		}
	}
}

type legacyBotConfigV12 struct {
	Spam struct {
		MaxImages int `json:"maximagespam"`
		MaxPings  int `json:"maxpingspam"`
	} `json:"spam"`
}

type legacyBotConfigV13 struct {
	Basic struct {
		Groups map[string]map[string]bool `json:"groups"`
	} `json:"basic"`
}

type legacyBotConfigV19 struct {
	Basic struct {
		Collections map[string]map[string]bool `json:"collections"`
	} `json:"basic"`
}

type legacyBotConfigV20 struct {
	Collections map[string]map[string]bool `json:"collections"`
	Spam        struct {
		SilentRole     DiscordRole `json:"silentrole"`
		SilenceMessage string      `json:"silencemessage"`
	} `json:"spam"`
	Basic struct {
		AlertRole     DiscordRole `json:"alertrole"`
		TrackUserLeft bool        `json:"trackuserleft"`
	} `json:"basic"`
	Search struct {
		MaxResults int `json:"maxsearchresults"`
	} `json:"search"`
	Spoiler struct {
		Channels []DiscordChannel `json:"spoilchannels"`
	} `json:"spoiler"`
	Schedule struct {
		BirthdayRole DiscordRole `json:"birthdayrole"`
	} `json:"schedule"`
}

type legacyBotConfigV26 struct {
	Spam struct {
		AutoSilence int `json:"autosilence"`
	} `json:"spam"`
}

func restrictCommand(v string, roles map[CommandID]map[DiscordRole]bool, modrole DiscordRole) {
	id := CommandID(v)
	_, ok := roles[id]
	if !ok && modrole != "" {
		roles[id] = make(map[DiscordRole]bool)
		roles[id][modrole] = true
	}
}

func (guild *GuildInfo) renameCommand(old CommandID, new CommandID) {
	if val, ok := guild.Config.Modules.CommandRoles[old]; ok {
		guild.Config.Modules.CommandRoles[new] = val
		delete(guild.Config.Modules.CommandRoles, old)
	}

	if val, ok := guild.Config.Modules.CommandChannels[old]; ok {
		guild.Config.Modules.CommandChannels[new] = val
		delete(guild.Config.Modules.CommandChannels, old)
	}

	if val, ok := guild.Config.Modules.CommandLimits[old]; ok {
		guild.Config.Modules.CommandLimits[new] = val
		delete(guild.Config.Modules.CommandLimits, old)
	}

	if val, ok := guild.Config.Modules.CommandDisabled[old]; ok {
		guild.Config.Modules.CommandDisabled[new] = val
		delete(guild.Config.Modules.CommandDisabled, old)
	}

	// Migrate aliases by substituting old command name for new command name
	for k, v := range guild.Config.Basic.Aliases {
		target := strings.SplitN(v, " ", 2)
		if strings.ToLower(target[0]) == strings.ToLower(string(old)) {
			guild.Config.Basic.Aliases[k] = strings.ToLower(string(new)) + " " + target[1]
		}
	}
}

// MigrateSettings from earlier config version
func (guild *GuildInfo) MigrateSettings(config []byte) error {
	err := json.Unmarshal(config, &guild.Config)
	if err != nil {
		return err
	}

	if guild.Config.Version <= 11 {
		restrictCommand("getaudit", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
	}

	if guild.Config.Version <= 12 {
		guild.Config.Spam.BasePressure = 10.0
		guild.Config.Spam.MaxPressure = 60.0
		guild.Config.Spam.ImagePressure = ((guild.Config.Spam.MaxPressure - guild.Config.Spam.BasePressure) / 6.0)
		guild.Config.Spam.PingPressure = ((guild.Config.Spam.MaxPressure - guild.Config.Spam.BasePressure) / 24.0)
		guild.Config.Spam.LengthPressure = ((guild.Config.Spam.MaxPressure - guild.Config.Spam.BasePressure) / (2000.0 * 4))
		guild.Config.Spam.RepeatPressure = guild.Config.Spam.BasePressure
		guild.Config.Spam.PressureDecay = 2.5

		legacy := legacyBotConfigV12{}
		err := json.Unmarshal(config, &legacy)
		if err == nil {
			if legacy.Spam.MaxImages > 0 {
				guild.Config.Spam.ImagePressure = ((guild.Config.Spam.MaxPressure - guild.Config.Spam.BasePressure) / float32(legacy.Spam.MaxImages+1))
			} else {
				guild.Config.Spam.ImagePressure = 0
			}
			if legacy.Spam.MaxPings > 0 {
				guild.Config.Spam.PingPressure = ((guild.Config.Spam.MaxPressure - guild.Config.Spam.BasePressure) / float32(legacy.Spam.MaxPings+1))
			} else {
				guild.Config.Spam.PingPressure = 0
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	if guild.Config.Version <= 13 {
		legacy := legacyBotConfigV13{}
		err := json.Unmarshal(config, &legacy)
		if err == nil {
			guild.Config.Users.Roles = make(map[DiscordRole]bool, len(legacy.Basic.Groups))
			idmap := make(map[string]string, len(legacy.Basic.Groups)) // Map initial group name to new role ID

			for k, v := range legacy.Basic.Groups {
				role := k
				check, err := GetRoleByName(role, guild)
				if check != nil {
					role = "sb-" + role
				}
				r, err := guild.Bot.DG.GuildRoleCreate(guild.ID)
				if err == nil {
					r, err = guild.Bot.DG.GuildRoleEdit(guild.ID, r.ID, role, 0, false, 0, true)
				}
				if err == nil {
					idmap[strings.ToLower(k)] = r.ID
					if id, err := ParseRole(r.ID, nil); err == nil {
						guild.Config.Users.Roles[id] = true
					}

					for u := range v {
						err = guild.Bot.DG.GuildMemberRoleAdd(guild.ID, u, r.ID)
						if err != nil {
							fmt.Println(err)
						}
					}
				} else {
					fmt.Println(err)
				}
			}

			stmt, err := guild.Bot.DB.Prepare("SELECT ID, Data FROM schedule WHERE Guild = ? AND Type = 7")
			stmt2, err := guild.Bot.DB.Prepare("UPDATE schedule SET Data = ? WHERE ID = ?")
			if err != nil {
				fmt.Println(err)
			} else {
				q, err := stmt.Query(SBatoi(guild.ID))
				if err != nil {
					fmt.Println(err)
				} else {
					defer q.Close()
					for q.Next() {
						var id uint64
						var dat string
						if err := q.Scan(&id, &dat); err == nil {
							datas := strings.SplitN(dat, "|", 2)
							groups := strings.Split(datas[0], "+")
							for i := range groups {
								rid, ok := idmap[strings.ToLower(groups[i])]
								if ok {
									groups[i] = "<@&" + rid + ">"
								}
							}
							_, err = stmt2.Exec(strings.Join(groups, " ")+"|"+datas[1], id)
							if err != nil {
								fmt.Println(err)
							}
						}
					}
				}
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	if guild.Config.Version <= 14 {
		restrictCommand("addrole", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("removerole", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("deleterole", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
	}

	if guild.Config.Version <= 15 {
		restrictCommand("bannewcomers", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		guild.Config.Spam.LockdownDuration = 120
	}

	if guild.Config.Version <= 16 {
		guild.Config.Basic.CommandPrefix = "!"
	}

	if guild.Config.Version <= 17 {
		guild.Config.SetupDone = true
	}

	if guild.Config.Version <= 18 {
		restrictCommand("banraid", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("getraid", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("wipe", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("bannewcomers", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("getpressure", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		guild.Config.Spam.LinePressure = (guild.Config.Spam.MaxPressure - guild.Config.Spam.BasePressure) / 70.0
	}

	if guild.Config.Version <= 19 {
		guild.Bot.GuildsLock.Lock()
		if len(guild.Config.Filter.Filters) == 0 {
			guild.Config.Filter.Filters = make(map[string]map[string]bool)
		}
		legacy := legacyBotConfigV19{}
		err := json.Unmarshal(config, &legacy)
		if err == nil {
			guild.Config.Bucket.Items = legacy.Basic.Collections["bucket"]
			guild.Config.Filter.Filters["emote"] = legacy.Basic.Collections["emote"]
			guild.Config.Status.Lines = legacy.Basic.Collections["status"]
			guild.Config.Filter.Filters["spoiler"] = legacy.Basic.Collections["spoiler"]
			delete(legacy.Basic.Collections, "bucket")
			delete(legacy.Basic.Collections, "emote")
			delete(legacy.Basic.Collections, "status")
			delete(legacy.Basic.Collections, "spoiler")

			gID := SBatoi(guild.ID)
			for k, v := range legacy.Basic.Collections {
				if len(v) > 0 {
					fmt.Println("Importing:", k)
					guild.Bot.DB.CreateTag(k, gID)
					tag, err := guild.Bot.DB.GetTag(k, gID)
					if err == nil {
						for item := range v {
							id, err := guild.Bot.DB.AddItem(item)
							if err == nil || err != ErrDuplicateEntry {
								guild.Bot.DB.AddTag(id, tag)
							}
						}
					}
				} else {
					fmt.Println("Skipping empty collection:", k)
				}
			}
		} else {
			fmt.Println(err.Error())
		}
		guild.Bot.GuildsLock.Unlock()
		restrictCommand("addset", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("removeset", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("searchset", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
	}

	if guild.Config.Version <= 20 {
		legacy := legacyBotConfigV20{}
		err := json.Unmarshal(config, &legacy)
		if err == nil {
			guild.Config.Basic.ModRole = legacy.Basic.AlertRole
			guild.Config.Miscellaneous.MaxSearchResults = legacy.Search.MaxResults
			guild.Config.Scheduler.BirthdayRole = legacy.Schedule.BirthdayRole
			guild.Config.Filter.Filters = make(map[string]map[string]bool)
			guild.Config.Filter.Channels = make(map[string]map[DiscordChannel]bool)
			guild.Config.Filter.Responses = make(map[string]string)
			guild.Config.Filter.Templates = make(map[string]string)
			guild.Config.Bucket.Items = make(map[string]bool)
			guild.Config.Status.Lines = make(map[string]bool)
			guild.Config.Users.TrackUserLeft = legacy.Basic.TrackUserLeft
			guild.Config.Users.SilenceMessage = legacy.Spam.SilenceMessage
			guild.Config.Basic.SilenceRole = legacy.Spam.SilentRole

			if bucket, ok := legacy.Collections["bucket"]; ok {
				for k, v := range bucket {
					guild.Config.Bucket.Items[k] = v
				}
			}

			if status, ok := legacy.Collections["status"]; ok {
				for k, v := range status {
					guild.Config.Status.Lines[k] = v
				}
			}

			if guild.Config.Spam.RaidSilence == -2 {
				guild.Config.Users.NotifyChannel = guild.Config.Log.Channel
			} else if guild.Config.Spam.RaidSilence != 0 {
				guild.Config.Users.NotifyChannel = guild.Config.Basic.ModChannel
			}
			if guild.Config.Spam.RaidSilence < 0 {
				guild.Config.Spam.RaidSilence = 0
			}

			if spoilers, ok := legacy.Collections["spoiler"]; (ok && len(spoilers) > 0) || len(legacy.Spoiler.Channels) > 0 {
				guild.Config.Filter.Filters["spoiler"] = make(map[string]bool)
				if ok {
					for k, v := range spoilers {
						guild.Config.Filter.Filters["spoiler"][k] = v
					}
				}
				guild.Config.Filter.Channels["spoiler"] = make(map[DiscordChannel]bool)
				for _, v := range legacy.Spoiler.Channels {
					guild.Config.Filter.Channels["spoiler"][v] = true
				}
				guild.Config.Filter.Responses["spoiler"] = "[](/nospoilers) ```\nNO SPOILERS! Posting spoilers is a bannable offense. All discussion about new and future content MUST be in #mylittlespoilers.```"
			}

			if emotes, ok := legacy.Collections["emote"]; ok && len(emotes) > 0 {
				guild.Config.Filter.Filters["emote"] = make(map[string]bool)
				for k, v := range emotes {
					guild.Config.Filter.Filters["emote"][k] = v
				}
				guild.Config.Filter.Channels["emote"] = make(map[DiscordChannel]bool)
				guild.Config.Filter.Responses["emote"] = "```\nThat emote isn't allowed here! Try to avoid using large or disturbing emotes, as they can be problematic.```"
				guild.Config.Filter.Templates["emote"] = "\\[\\]\\(\\/r?%%[-) \"]"
			}
		}

		if guild.Config.Basic.ModRole == "0" {
			guild.Config.Basic.ModRole = ""
		}
		if guild.Config.Basic.ModChannel == "0" {
			guild.Config.Basic.ModChannel = ""
		}
		if guild.Config.Basic.SilenceRole == "0" {
			guild.Config.Basic.SilenceRole = ""
		}
		if guild.Config.Spam.IgnoreRole == "0" {
			guild.Config.Spam.IgnoreRole = ""
		}
		if guild.Config.Users.WelcomeChannel == "0" {
			guild.Config.Users.WelcomeChannel = ""
		}
		if guild.Config.Users.NotifyChannel == "0" {
			guild.Config.Users.NotifyChannel = ""
		}
		if guild.Config.Log.Channel == "0" {
			guild.Config.Log.Channel = ""
		}
		if guild.Config.Scheduler.BirthdayRole == "0" {
			guild.Config.Scheduler.BirthdayRole = ""
		}

		for k := range guild.Config.Modules.Channels {
			switch k {
			case "schedule":
				guild.Config.Modules.Channels["scheduler"] = guild.Config.Modules.Channels[k]
				delete(guild.Config.Modules.Channels, k)
			case "anti-spam":
				guild.Config.Modules.Channels["spam"] = guild.Config.Modules.Channels[k]
				delete(guild.Config.Modules.Channels, k)
			case "help/about":
				guild.Config.Modules.Channels["information"] = guild.Config.Modules.Channels[k]
				delete(guild.Config.Modules.Channels, k)
			}
		}

		for k := range guild.Config.Modules.Disabled {
			switch k {
			case "schedule":
				guild.Config.Modules.Channels["scheduler"] = guild.Config.Modules.Channels[k]
				delete(guild.Config.Modules.Channels, k)
			case "anti-spam":
				guild.Config.Modules.Channels["spam"] = guild.Config.Modules.Channels[k]
				delete(guild.Config.Modules.Channels, k)
			case "help/about":
				guild.Config.Modules.Channels["information"] = guild.Config.Modules.Channels[k]
				delete(guild.Config.Modules.Channels, k)
			}
		}
	}

	if guild.Config.Version <= 21 {
		restrictCommand("assignrole", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
	}

	if guild.Config.Version <= 22 {
		restrictCommand("setfilter", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("addfilter", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("removefilter", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("searchfilter", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("addstatus", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("removestatus", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("setstatus", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
	}

	if guild.Config.Version <= 23 {
		restrictCommand("createrole", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
	}

	if guild.Config.Version <= 24 {
		guild.setupSilenceRole()
	}

	if guild.Config.Version <= 25 {
		restrictCommand("echoembed", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
	}

	if guild.Config.Version <= 26 {
		legacy := legacyBotConfigV26{}
		err := json.Unmarshal(config, &legacy)
		if err == nil {
			guild.Config.Spam.RaidSilence = legacy.Spam.AutoSilence
		}
		guild.renameCommand("autosilence", "raidsilence")
		restrictCommand("increment", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("addcounter", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
		restrictCommand("removecounter", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
	}

	if guild.Config.Version <= 27 {
		guild.Config.Bored.Exponent = 1
	}

	if guild.Config.Version <= 28 {
		restrictCommand("import", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
	}

	if guild.Config.Version <= 29 {
		guild.Config.Users.JailChannel = guild.Config.Users.WelcomeChannel
		restrictCommand("setmemberrole", guild.Config.Modules.CommandRoles, guild.Config.Basic.ModRole)
	}

	if guild.Config.Version != ConfigVersion {
		guild.Config.Version = ConfigVersion // set version to most recent config version
		guild.SaveConfig()
	}
	return nil
}

func getSubStruct(arg []string, f reflect.Value, j int, info *GuildInfo) []string {
	val := f.Field(j)
	if len(arg) > 2 {
		switch f.Field(j).Interface().(type) {
		case map[string]bool, map[string]string, map[string]int64, map[string]map[DiscordChannel]bool, map[string]map[string]bool, map[string]float32:
			val = f.Field(j).MapIndex(reflect.ValueOf(arg[2]))
		case map[DiscordChannel]bool, map[DiscordChannel]float32:
			val = f.Field(j).MapIndex(reflect.ValueOf(DiscordChannel(arg[2])))
		case map[DiscordRole]bool:
			val = f.Field(j).MapIndex(reflect.ValueOf(DiscordRole(arg[2])))
		case map[DiscordUser][]string:
			val = f.Field(j).MapIndex(reflect.ValueOf(DiscordUser(arg[2])))
		case map[int]string:
			ival, _ := strconv.Atoi(arg[2])
			val = f.Field(j).MapIndex(reflect.ValueOf(ival))
		case map[CommandID]bool, map[CommandID]int64, map[CommandID]map[DiscordRole]bool, map[CommandID]map[DiscordChannel]bool:
			val = f.Field(j).MapIndex(reflect.ValueOf(CommandID(arg[2])))
		case map[ModuleID]bool, map[ModuleID]map[DiscordChannel]bool:
			val = f.Field(j).MapIndex(reflect.ValueOf(ModuleID(arg[2])))
		default:
			return []string{"is not a map"}
		}
		if !val.IsValid() || val == reflect.Zero(val.Type()) {
			return []string{fmt.Sprintf("can't find %v", arg[2])}
		}
	}
	return info.Config.GetConfig(val, info.Bot.DG.State, info.ID)
}
