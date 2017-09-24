package sweetiebot

// BotConfig lists all bot configuration options, grouped into structs
type BotConfig struct {
	Version     int                        `json:"version"`
	LastVersion int                        `json:"lastversion"`
	SetupDone   bool                       `json:"setupdone"`
	Collections map[string]map[string]bool `json:"collections"`
	Basic       struct {
		IgnoreInvalidCommands bool              `json:"ignoreinvalidcommands"`
		Importable            bool              `json:"importable"`
		AlertRole             uint64            `json:"alertrole"`
		ModChannel            uint64            `json:"modchannel"`
		FreeChannels          map[string]bool   `json:"freechannels"`
		BotChannel            uint64            `json:"botchannel"`
		Aliases               map[string]string `json:"aliases"`
		ListenToBots          bool              `json:"listentobots"`
		CommandPrefix         string            `json:"commandprefix"`
		TrackUserLeft         bool              `json:"trackuserleft"`
	} `json:"basic"`
	Modules struct {
		Channels           map[string]map[string]bool `json:"modulechannels"`
		Disabled           map[string]bool            `json:"moduledisabled"`
		CommandRoles       map[string]map[string]bool `json:"commandroles"`
		CommandChannels    map[string]map[string]bool `json:"commandchannels"`
		CommandLimits      map[string]int64           `json:"Commandlimits"`
		CommandDisabled    map[string]bool            `json:"commanddisabled"`
		CommandPerDuration int                        `json:"commandperduration"`
		CommandMaxDuration int64                      `json:"commandmaxduration"`
	} `json:"modules"`
	Spam struct {
		ImagePressure      float32            `json:"imagepressure"`
		PingPressure       float32            `json:"pingpressure"`
		LengthPressure     float32            `json:"lengthpressure"`
		RepeatPressure     float32            `json:"repeatpressure"`
		LinePressure       float32            `json:"linepressure"`
		BasePressure       float32            `json:"basepressure"`
		PressureDecay      float32            `json:"pressuredecay"`
		MaxPressure        float32            `json:"maxpressure"`
		MaxChannelPressure map[uint64]float32 `json:"maxchannelpressure"`
		MaxRemoveLookback  int                `json:"MaxSpamRemoveLookback"`
		SilentRole         uint64             `json:"silentrole"`
		IgnoreRole         uint64             `json:"ignorerole"`
		RaidTime           int64              `json:"maxraidtime"`
		RaidSize           int                `json:"raidsize"`
		SilenceMessage     string             `json:"silencemessage"`
		AutoSilence        int                `json:"autosilence"`
		LockdownDuration   int                `json:"lockdownduration"`
	} `json:"spam"`
	Bucket struct {
		MaxItems       int `json:"maxbucket"`
		MaxItemLength  int `json:"maxbucketlength"`
		MaxFightHP     int `json:"maxfighthp"`
		MaxFightDamage int `json:"maxfightdamage"`
	} `json:"bucket"`
	Markov struct {
		MaxPMlines     int  `json:"maxpmlines"`
		MaxLines       int  `json:"maxquotelines"`
		DefaultLines   int  `json:"defaultmarkovlines"`
		UseMemberNames bool `json:"usemembernames"`
	} `json:"markov"`
	Users struct {
		TimezoneLocation string          `json:"timezonelocation"`
		WelcomeChannel   uint64          `json:"welcomechannel"`
		WelcomeMessage   string          `json:"welcomemessage"`
		Roles            map[uint64]bool `json:"userroles"`
	} `json:"users"`
	Bored struct {
		Cooldown int64           `json:"maxbored"`
		Commands map[string]bool `json:"boredcommands"`
	}
	Help struct {
		Rules             map[int]string `json:"rules"`
		HideNegativeRules bool           `json:"hidenegativerules"`
	} `json:"help"`
	Log struct {
		Cooldown int64  `json:"maxerror"`
		Channel  uint64 `json:"logchannel"`
	} `json:"log"`
	Witty struct {
		Responses map[string]string `json:"witty"`
		Cooldown  int64             `json:"maxwit"`
	} `json:"Wit"`
	Schedule struct {
		BirthdayRole uint64 `json:"birthdayrole"`
	} `json:"schedule"`
	Search struct {
		MaxResults int `json:"maxsearchresults"`
	} `json:"search"`
	Spoiler struct {
		Channels []uint64 `json:"spoilchannels"`
	} `json:"spoiler"`
	Status struct {
		Cooldown int `json:"statusdelaytime"`
	} `json:"status"`
	Quote struct {
		Quotes map[uint64][]string `json:"quotes"`
	} `json:"quote"`
}

// ConfigHelp is a map of help strings for the configuration options above
var ConfigHelp map[string]string = map[string]string{
	"basic.ignoreinvalidcommands": "If true, Sweetie Bot won't display an error if a nonsensical command is used. This helps her co-exist with other bots that also use the `!` prefix.",
	"basic.importable":            "If true, the collections on this server will be importable into another server where sweetie is.",
	"basic.alertrole":             "This is intended to point at a moderator role shared by all admins and moderators of the server for notification purposes.",
	"basic.modchannel":            "This should point at the hidden moderator channel, or whatever channel moderates want to be notified on.",
	"basic.freechannels":          "This is a list of all channels that are exempt from rate limiting. Usually set to the dedicated `#botabuse` channel in a server.",
	"basic.botchannel":            "This allows you to designate a particular channel for sweetie bot to point users to if they are trying to run too many commands at once. Usually this channel will also be included in `basic.freechannels`",
	"basic.aliases":               "Can be used to redirect commands, such as making `!listgroup` call the `!listgroups` command. Useful for making shortcuts.\n\nExample: `!setconfig basic.aliases kawaii \"pick cute\"` sets an alias mapping `!kawaii arg1...` to `!pick cute arg1...`, preserving all arguments that are passed to the alias.",
	"basic.listentobots":          "If true, sweetiebot will process bot messages and allow them to run commands. Bots can never trigger anti-spam. Defaults to false.",
	"basic.commandprefix":         "Determines the SINGLE ASCII CHARACTER prefix used to denote sweetiebot commands. You can't set it to an emoji or any weird foreign character. The default is `!`. If this is set to an invalid value, Sweetiebot will default to using `!`.",
	"basic.trackuserleft":         "If true, sweetiebot will also track users that leave the server if autosilence is set to alert or log. Defaults to false.",
	"modules.commandroles":        "A map of which roles are allowed to run which command. If no mapping exists, everyone can run the command.",
	"modules.commandchannels":     "A map of which channels commands are allowed to run on. No entry means a command can be run anywhere. If \"!\" is included as a channel, it switches from a whitelist to a blacklist, enabling you to exclude certain channels instead of allow certain channels.",
	"modules.commandlimits":       "A map of timeouts for commands. A value of 30 means the command can't be used more than once every 30 seconds.",
	"modules.commanddisabled":     "A list of disabled commands.",
	"modules.commandperduration":  "Maximum number of commands that can be run within `commandmaxduration` seconds. Default: 3",
	"modules.commandmaxduration":  "Default: 20. This means that by default, at most 3 commands can be run every 20 seconds.",
	"modules.disabled":            "A list of disabled modules.",
	"modules.channels":            "A mapping of what channels a given module can operate on. If no mapping is given, a module operates on all channels. If \"!\" is included as a channel, it switches from a whitelist to a blacklist, enabling you to exclude certain channels instead of allow certain channels.",
	"spam.imagepressure":          "Additional pressure generated by each image, link or attachment in a message. Defaults to (MaxPressure - BasePressure) / 6, instantly silencing anyone posting 6 or more links at once.",
	"spam.repeatpressure":         "Additional pressure generated by a message that is identical to the previous message sent (ignores case). Defaults to BasePressure, effectively doubling the pressure penalty for repeated messages.",
	"spam.pingpressure":           "Additional pressure generated by each unique ping in a message. Defaults to (MaxPressure - BasePressure) / 20, instantly silencing anyone pinging 20 or more people at once.",
	"spam.lengthpressure":         "Additional pressure generated by each individual character in the message. Discord allows messages up to 2000 characters in length. Defaults to (MaxPressure - BasePressure) / 8000, silencing anyone posting 3 huge messages at the same time.",
	"spam.linepressure":           "Additional pressure generated by each newline in the message. Defaults to (MaxPressure - BasePressure) / 70, silencing anyone posting more than 70 newlines in a single message",
	"spam.basepressure":           "The base pressure generated by sending a message, regardless of length or content. Defaults to 10",
	"spam.maxpressure":            "The maximum pressure allowed. If a user's pressure exceeds this amount, they will be silenced. Defaults to 60, which is intended to ban after a maximum of 6 short messages sent in rapid succession.",
	"spam.maxchannelpressure":     "Per-channel pressure override. If a channel's pressure is specified in this map, it will override the global maxpressure setting.",
	"spam.pressuredecay":          "The number of seconds it takes for a user to lose Spam.BasePressure from their pressure amount. Defaults to 2.5, so after sending 3 messages, it will take 7.5 seconds for their pressure to return to 0.",
	"spam.maxremovelookback":      "Number of seconds back the bot should delete messages of a silenced user on the channel they spammed on. If set to 0, the bot will only delete the message that caused the user to be silenced. If less than 0, the bot won't delete any messages.",
	"spam.ignorerole":             "If set, the bot will exclude anyone with this role from spam detection. Use with caution.",
	"spam.silentrole":             "This should be a role with no permissions, so the bot can quarantine potential spammers without banning them.",
	"spam.raidtime":               "In order to trigger a raid alarm, at least `spam.raidsize` people must join the chat within this many seconds of each other.",
	"spam.raidsize":               "Specifies how many people must have joined the server within the `spam.raidtime` period to qualify as a raid.",
	"spam.silencemessage":         "This message will be sent to users that have been silenced by the `!silence` command.",
	"spam.autosilence":            "Gets the current autosilence state. Use the `!autosilence` command to set this.",
	"spam.lockdownduration":       "Determines how long the server's verification mode will temporarily be increased to tableflip levels after a raid is detected. If set to 0, disables lockdown entirely.",
	"bucket.maxitems":             "Determines the maximum number of items sweetiebot can carry in her bucket. If set to 0, her bucket is disabled.",
	"bucket.maxitemlength":        "Determines the maximum length of a string that can be added to her bucket.",
	"bucket.maxfighthp":           "Maximum HP of the randomly generated enemy for the `!fight` command.",
	"bucket.maxfightdamage":       "Maximum amount of damage a randomly generated weapon can deal for the `!fight` command.",
	"markov.maxpmlines":           "This is the maximum number of lines a response can be before sweetiebot automatically sends it as a PM to avoid cluttering the chat. Default: 5",
	"markov.maxlines":             "Maximum number of lines the `!episodequote` command can be given.",
	"markov.defaultlines":         "Number of lines for the markov chain to spawn when not given a line count.",
	"markov.usemembernames":       "Use member names instead of random pony names.",
	"users.timezonelocation":      "Sets the timezone location of the server itself. When no user timezone is available, the bot will use this.",
	"users.welcomechannel":        "If set to a channel ID, the bot will treat this channel as a \"quarantine zone\" for silenced members. If autosilence is enabled, new users will be sent to this channel.",
	"users.welcomemessage":        "If autosilence is enabled, this message will be sent to a new user upon joining.",
	"users.roles":                 "A list of all user-assignable roles. Manage it via !addrole and !removerole",
	"bored.cooldown":              "The bored cooldown timer, in seconds. This is the length of time a channel must be inactive for sweetiebot to post a bored message in it.",
	"bored.commands":              "This determines what commands sweetie will run when she gets bored. She will choose one command from this list at random.\n\nExample: `!setconfig bored.commands !drop \"!pick bored\"`",
	"help.rules":                  "Contains a list of numbered rules. The numbers do not need to be contiguous, and can be negative.",
	"help.hidenegativerules":      "If true, `!rules -1` will display a rule at index -1, but `!rules` will not. This is useful for joke rules or additional rules that newcomers don't need to know about.",
	"log.channel":                 "This is the channel where sweetiebot logs her output.",
	"log.cooldown":                "The cooldown time for sweetiebot to display an error message, in seconds, intended to prevent the bot from spamming itself. Default: 4",
	"witty.responses":             "Stores the replies used by the Witty module and must be configured using `!addwit` or `!removewit`",
	"witty.cooldown":              "The cooldown time for the witty module. At least this many seconds must have passed before the bot will make another witty reply.",
	"schedule.birthdayrole":       " This is the role given to members on their birthday.",
	"search.maxresults":           "Maximum number of search results that can be requested at once.",
	"spoiler.channels":            "A list of channels that are exempt from the spoiler rules.",
	"status.cooldown":             "Number of seconds sweetiebot waits before changing her status to a string picked randomly from the `status` collection.",
	"quote.quotes":                "This is a map of quotes, which should be managed via `!addquote` and `!removequote`.",
}
