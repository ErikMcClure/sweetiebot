package sweetiebot

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type ModuleHooks struct {
	OnEvent             []ModuleOnEvent
	OnTypingStart       []ModuleOnTypingStart
	OnMessageCreate     []ModuleOnMessageCreate
	OnMessageUpdate     []ModuleOnMessageUpdate
	OnMessageDelete     []ModuleOnMessageDelete
	OnMessageAck        []ModuleOnMessageAck
	OnPresenceUpdate    []ModuleOnPresenceUpdate
	OnVoiceStateUpdate  []ModuleOnVoiceStateUpdate
	OnGuildUpdate       []ModuleOnGuildUpdate
	OnGuildMemberAdd    []ModuleOnGuildMemberAdd
	OnGuildMemberRemove []ModuleOnGuildMemberRemove
	OnGuildMemberUpdate []ModuleOnGuildMemberUpdate
	OnGuildBanAdd       []ModuleOnGuildBanAdd
	OnGuildBanRemove    []ModuleOnGuildBanRemove
	OnCommand           []ModuleOnCommand
	OnIdle              []ModuleOnIdle
	OnTick              []ModuleOnTick
}

type BotConfig struct {
	Version               int                        `json:"version"`
	Debug                 bool                       `json:"debug"`
	Maxerror              int64                      `json:"maxerror"`
	Maxwit                int64                      `json:"maxwit"`
	Maxbored              int64                      `json:"maxbored"`
	DisableBored          int                        `json:"disablebored"`
	MaxPMlines            int                        `json:"maxpmlines"`
	Maxquotelines         int                        `json:"maxquotelines"`
	Maxsearchresults      int                        `json:"maxsearchresults"`
	Defaultmarkovlines    int                        `json:"defaultmarkovlines"`
	Commandperduration    int                        `json:"commandperduration"`
	Commandmaxduration    int64                      `json:"commandmaxduration"`
	StatusDelayTime       int                        `json:"statusdelaytime"`
	MaxRaidTime           int64                      `json:"maxraidtime"`
	RaidSize              int                        `json:"raidsize"`
	Witty                 map[string]string          `json:"witty"`
	Aliases               map[string]string          `json:"aliases"`
	MaxBucket             int                        `json:"maxbucket"`
	MaxBucketLength       int                        `json:"maxbucketlength"`
	MaxFightHP            int                        `json:"maxfighthp"`
	MaxFightDamage        int                        `json:"maxfightdamage"`
	MaxImageSpam          int                        `json:"maximagespam"`
	MaxAttachSpam         int                        `json:"maxattachspam"`
	MaxPingSpam           int                        `json:"maxpingspam"`
	MaxMessageSpam        map[int64]int              `json:"maxmessagespam"`
	IgnoreInvalidCommands bool                       `json:"ignoreinvalidcommands"`
	UseMemberNames        bool                       `json:"usemembernames"`
	Timezone              int                        `json:"timezone"`
	AutoSilence           int                        `json:"autosilence"`
	AlertRole             uint64                     `json:"alertrole"`
	SilentRole            uint64                     `json:"silentrole"`
	LogChannel            uint64                     `json:"logchannel"`
	ModChannel            uint64                     `json:"modchannel"`
	WelcomeChannel        uint64                     `json:"welcomechannel"`
	WelcomeMessage        string                     `json:"welcomemessage"`
	BirthdayRole          uint64                     `json:"birthdayrole"`
	SpoilChannels         []uint64                   `json:"spoilchannels"`
	FreeChannels          map[string]bool            `json:"freechannels"`
	Command_roles         map[string]map[string]bool `json:"command_roles"`
	Command_channels      map[string]map[string]bool `json:"command_channels"`
	Command_limits        map[string]int64           `json:command_limits`
	Command_disabled      map[string]bool            `json:command_disabled`
	Module_disabled       map[string]bool            `json:module_disabled`
	Module_channels       map[string]map[string]bool `json:module_channels`
	Collections           map[string]map[string]bool `json:"collections"`
	Groups                map[string]map[string]bool `json:"groups"`
	Quotes                map[uint64][]string        `json:"quotes"`
}

type GuildInfo struct {
	Guild        *discordgo.Guild
	log          *Log
	command_last map[string]map[string]int64
	commandlimit *SaturationLimit
	config       BotConfig
	emotemodule  *EmoteModule
	hooks        ModuleHooks
	modules      []Module
	commands     map[string]Command
}

type SweetieBot struct {
	db                 *BotDB
	dg                 *discordgo.Session
	version            string
	SelfID             string
	Owners             map[uint64]bool
	RestrictedCommands map[string]bool
	MainGuildID        uint64
	DebugChannels      map[string]string
	quit               bool
	guilds             map[string]*GuildInfo
	GuildChannels      map[string]*GuildInfo
	LastMessages       map[string]int64
	MaxConfigSize      int
}

var sb *SweetieBot
var channelregex = regexp.MustCompile("<#[0-9]+>")
var userregex = regexp.MustCompile("<@!?[0-9]+>")
var repeatregex = regexp.MustCompile("repeat -?[0-9]+ (second|minute|hour|day|week|month|quarter|year)s?")
var locUTC = time.FixedZone("UTC", 0)

func (sbot *SweetieBot) IsMainGuild(info *GuildInfo) bool {
	return SBatoi(info.Guild.ID) == sbot.MainGuildID
}
func (info *GuildInfo) AddCommand(c Command) {
	info.commands[strings.ToLower(c.Name())] = c
}

func (info *GuildInfo) SaveConfig() {
	data, err := json.Marshal(info.config)
	if err == nil {
		if len(data) > sb.MaxConfigSize {
			info.log.Log("Error saving config file: Config file is too large! Config files cannot exceed " + strconv.Itoa(sb.MaxConfigSize) + " bytes.")
		} else {
			ioutil.WriteFile(info.Guild.ID+".json", data, 0664)
		}
	} else {
		info.log.Log("Error writing json: ", err.Error())
	}
}

func DeleteFromMapReflect(f reflect.Value, k string) string {
	f.SetMapIndex(reflect.ValueOf(k), reflect.Value{})
	return "Deleted " + k
}

func (info *GuildInfo) SetConfig(name string, value string, extra ...string) (string, bool) {
	name = strings.ToLower(name)
	t := reflect.ValueOf(&info.config).Elem()
	n := t.NumField()
	for i := 0; i < n; i++ {
		if strings.ToLower(t.Type().Field(i).Name) == name {
			f := t.Field(i)
			switch t.Field(i).Interface().(type) {
			case string:
				f.SetString(value)
			case int, int8, int16, int32, int64:
				k, _ := strconv.ParseInt(value, 10, 64)
				f.SetInt(k)
			case uint, uint8, uint16, uint32:
				k, _ := strconv.ParseUint(value, 10, 64)
				f.SetUint(k)
			case uint64:
				f.SetUint(PingAtoi(value))
			case []uint64:
				f.Set(reflect.MakeSlice(reflect.TypeOf(f.Interface()), 0, 1+len(extra)))
				if len(value) > 0 {
					f.Set(reflect.Append(f, reflect.ValueOf(PingAtoi(value))))
					for _, k := range extra {
						f.Set(reflect.Append(f, reflect.ValueOf(PingAtoi(k))))
					}
				}
			case bool:
				f.SetBool(value == "true")
			case map[string]string:
				value = strings.ToLower(value)
				if len(extra) == 0 {
					return "No extra parameter given for " + name, false
				}
				if f.IsNil() {
					f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
				}
				if len(extra[0]) == 0 {
					return DeleteFromMapReflect(f, value), false
				}

				f.SetMapIndex(reflect.ValueOf(value), reflect.ValueOf(extra[0]))
				return value + ": " + extra[0], true
			case map[string]int64:
				value = strings.ToLower(value)
				if len(extra) == 0 {
					return "No extra parameter given for " + name, false
				}
				if f.IsNil() {
					f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
				}
				if len(extra[0]) == 0 {
					return DeleteFromMapReflect(f, value), false
				}

				k, _ := strconv.ParseInt(extra[0], 10, 64)
				f.SetMapIndex(reflect.ValueOf(value), reflect.ValueOf(k))
				return value + ": " + strconv.FormatInt(k, 10), true
			case map[int64]int:
				ivalue, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return value + " is not an integer.", false
				}
				if len(extra) == 0 {
					return "No extra parameter given for " + name, false
				}
				if f.IsNil() {
					f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
				}
				if len(extra[0]) == 0 {
					f.SetMapIndex(reflect.ValueOf(ivalue), reflect.Value{})
					return "Deleted " + value, false
				}

				k, _ := strconv.Atoi(extra[0])
				f.SetMapIndex(reflect.ValueOf(ivalue), reflect.ValueOf(k))
				return value + ": " + strconv.Itoa(k), true
			case map[string]bool:
				f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
				f.SetMapIndex(reflect.ValueOf(StripPing(value)), reflect.ValueOf(true))
				stripped := []string{StripPing(value)}
				for _, k := range extra {
					f.SetMapIndex(reflect.ValueOf(StripPing(k)), reflect.ValueOf(true))
					stripped = append(stripped, StripPing(k))
				}
				return "[" + strings.Join(stripped, ", ") + "]", true
			case map[string]map[string]bool:
				value = strings.ToLower(value)
				if f.IsNil() {
					f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
				}
				if len(extra) == 0 {
					return DeleteFromMapReflect(f, value), false
				}

				m := reflect.MakeMap(reflect.TypeOf(f.Interface()).Elem())
				stripped := []string{}
				for _, k := range extra {
					m.SetMapIndex(reflect.ValueOf(StripPing(k)), reflect.ValueOf(true))
					stripped = append(stripped, StripPing(k))
				}
				f.SetMapIndex(reflect.ValueOf(value), m)
				return value + ": [" + strings.Join(stripped, ", ") + "]", true
			default:
				info.log.Log(name + " is an unknown type " + t.Field(i).Type().Name())
				return "That config option has an unknown type!", false
			}
			return fmt.Sprint(t.Field(i).Interface()), true
		}
	}
	return "Could not find configuration parameter " + name + "!", false
}

func sbemotereplace(s string) string {
	return strings.Replace(s, "[](/", "[\u200B](/", -1)
}

func (info *GuildInfo) SanitizeOutput(message string) string {
	if info.emotemodule != nil {
		message = info.emotemodule.emoteban.ReplaceAllStringFunc(message, sbemotereplace)
	}
	return message
}

func ExtraSanitize(s string) string {
	s = strings.Replace(s, "`", "", -1)
	s = strings.Replace(s, "[](/", "[\u200B](/", -1)
	s = strings.Replace(s, "http://", "http\u200B://", -1)
	s = strings.Replace(s, "https://", "https\u200B://", -1)
	return ReplaceAllMentions(s)
}

func (info *GuildInfo) SendMessage(channelID string, message string) {
	sb.dg.ChannelMessageSend(channelID, info.SanitizeOutput(message))
}

func (info *GuildInfo) ProcessModule(channelID string, m Module) bool {
	_, disabled := info.config.Module_disabled[strings.ToLower(m.Name())]
	if disabled {
		return false
	}

	c := info.config.Module_channels[strings.ToLower(m.Name())]
	if len(channelID) > 0 && len(c) > 0 { // Only check for channels if we have a channel to check for, and the module actually has specific channels
		_, reverse := c["!"]
		_, ok := c[channelID]
		return ok != reverse
	}
	return true
}

func (info *GuildInfo) SwapStatusLoop() {
	if sb.IsMainGuild(info) {
		for !sb.quit {
			if len(info.config.Collections["status"]) > 0 {
				sb.dg.UpdateStatus(0, MapGetRandomItem(info.config.Collections["status"]))
			}
			time.Sleep(time.Duration(info.config.StatusDelayTime) * time.Second)
		}
	}
}

func ChangeBotName(s *discordgo.Session, name string, avatarfile string) {
	binary, _ := ioutil.ReadFile(avatarfile)
	email, _ := ioutil.ReadFile("email")
	password, _ := ioutil.ReadFile("passwd")
	avatar := base64.StdEncoding.EncodeToString(binary)

	_, err := s.UserUpdate(strings.TrimSpace(string(email)), strings.TrimSpace(string(password)), name, "data:image/jpeg;base64,"+avatar, "")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Changed username successfully")
	}
}

//func SBEvent(s *discordgo.Session, e *discordgo.Event) { ApplyFuncRange(len(info.hooks.OnEvent), func(i int) { if(ProcessModule("", info.hooks.OnEvent[i])) { info.hooks.OnEvent[i].OnEvent(s, e) } }) }
func SBReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("Ready message receieved, waiting for guilds...")
	sb.SelfID = r.User.ID

	// Only used to change sweetiebot's name or avatar
	//ChangeBotName(s, "Sweetie", "avatar.jpg")
}

func AttachToGuild(g *discordgo.Guild) {
	guild, exists := sb.guilds[g.ID]
	if exists {
		guild.ProcessGuild(g)
		return
	}

	fmt.Println("Initializing " + g.Name)

	guild = &GuildInfo{
		Guild:        g,
		command_last: make(map[string]map[string]int64),
		commandlimit: &SaturationLimit{[]int64{}, 0, AtomicFlag{0}},
		commands:     make(map[string]Command),
		emotemodule:  nil,
	}
	guild.log = &Log{0, guild}
	config, err := ioutil.ReadFile(g.ID + ".json")
	disableall := false
	if err != nil {
		config, _ = ioutil.ReadFile("default.json")
		disableall = true
	}
	err = json.Unmarshal(config, &guild.config)
	if err != nil {
		fmt.Println("Error reading config file for "+g.Name+": ", err.Error())
	}

	MigrateSettings(guild)
	guild.commandlimit.times = make([]int64, guild.config.Commandperduration*2, guild.config.Commandperduration*2)

	if len(guild.config.Witty) == 0 {
		guild.config.Witty = make(map[string]string)
	}
	if len(guild.config.Aliases) == 0 {
		guild.config.Aliases = make(map[string]string)
	}
	if len(guild.config.FreeChannels) == 0 {
		guild.config.FreeChannels = make(map[string]bool)
	}
	if len(guild.config.Command_roles) == 0 {
		guild.config.Command_roles = make(map[string]map[string]bool)
	}
	if len(guild.config.Command_channels) == 0 {
		guild.config.Command_channels = make(map[string]map[string]bool)
	}
	if len(guild.config.Command_limits) == 0 {
		guild.config.Command_limits = make(map[string]int64)
	}
	if len(guild.config.Command_disabled) == 0 {
		guild.config.Command_disabled = make(map[string]bool)
	}
	if len(guild.config.Module_disabled) == 0 {
		guild.config.Module_disabled = make(map[string]bool)
	}
	if len(guild.config.Module_channels) == 0 {
		guild.config.Module_channels = make(map[string]map[string]bool)
	}
	if len(guild.config.Groups) == 0 {
		guild.config.Groups = make(map[string]map[string]bool)
	}
	if len(guild.config.Collections) == 0 {
		guild.config.Collections = make(map[string]map[string]bool)
	}

	collections := []string{"emote", "bored", "cute", "status", "spoiler", "bucket"}
	for _, v := range collections {
		_, ok := guild.config.Collections[v]
		if !ok {
			guild.config.Collections[v] = make(map[string]bool)
		}
	}

	if sb.IsMainGuild(guild) {
		sb.db.log = guild.log
	}
	if guild.config.Debug { // The server does not necessarily tie a standard input to the program
		go WaitForInput()
	}

	sb.guilds[g.ID] = guild
	guild.ProcessGuild(g)

	episodegencommand := &EpisodeGenCommand{}
	guild.emotemodule = &EmoteModule{}
	spoilermodule := &SpoilerModule{}
	wittymodule := &WittyModule{}
	spammodule := &SpamModule{}
	guild.modules = make([]Module, 0, 6)
	guild.modules = append(guild.modules, spammodule)
	guild.modules = append(guild.modules, &PingModule{})
	guild.modules = append(guild.modules, guild.emotemodule)
	guild.modules = append(guild.modules, wittymodule)
	guild.modules = append(guild.modules, &BoredModule{Episodegen: episodegencommand})
	guild.modules = append(guild.modules, spoilermodule)
	guild.modules = append(guild.modules, &ScheduleModule{})

	for _, v := range guild.modules {
		v.Register(guild)
	}

	addfuncmap := map[string]func(string) string{
		"emote": func(arg string) string {
			r := guild.emotemodule.UpdateRegex(guild)
			if !r {
				delete(guild.config.Collections["emote"], arg)
				guild.emotemodule.UpdateRegex(guild)
				return "```Failed to ban " + arg + " because regex compilation failed.```"
			}
			return "```Banned " + arg + " and recompiled the emote regex.```"
		},
		"spoiler": func(arg string) string {
			r := spoilermodule.UpdateRegex(guild)
			if !r {
				delete(guild.config.Collections["spoiler"], arg)
				spoilermodule.UpdateRegex(guild)
				return "```Failed to ban " + arg + " because regex compilation failed.```"
			}
			return "```Banned " + arg + " and recompiled the spoiler regex.```"
		},
	}
	removefuncmap := map[string]func(string) string{
		"emote": func(arg string) string {
			guild.emotemodule.UpdateRegex(guild)
			return "```Unbanned " + arg + " and recompiled the emote regex.```"
		},
		"spoiler": func(arg string) string {
			spoilermodule.UpdateRegex(guild)
			return "```Unbanned " + arg + " and recompiled the spoiler regex.```"
		},
	}
	// We have to initialize commands and modules up here because they depend on the discord channel state
	guild.AddCommand(&AddCommand{addfuncmap})
	guild.AddCommand(&RemoveCommand{removefuncmap})
	guild.AddCommand(&CollectionsCommand{})
	guild.AddCommand(&EchoCommand{})
	guild.AddCommand(&HelpCommand{})
	guild.AddCommand(&NewUsersCommand{})
	guild.AddCommand(&EnableCommand{})
	guild.AddCommand(&DisableCommand{})
	guild.AddCommand(&UpdateCommand{})
	guild.AddCommand(&AKACommand{})
	guild.AddCommand(&AboutCommand{})
	guild.AddCommand(&LastPingCommand{})
	guild.AddCommand(&SetConfigCommand{})
	guild.AddCommand(&GetConfigCommand{})
	guild.AddCommand(&LastSeenCommand{})
	guild.AddCommand(&DumpTablesCommand{})
	guild.AddCommand(episodegencommand)
	guild.AddCommand(&EpisodeQuoteCommand{})
	guild.AddCommand(&ShipCommand{})
	guild.AddCommand(&AddWitCommand{wittymodule})
	guild.AddCommand(&RemoveWitCommand{wittymodule})
	guild.AddCommand(&SearchCommand{emotes: guild.emotemodule, statements: make(map[string][]*sql.Stmt)})
	guild.AddCommand(&SetStatusCommand{})
	guild.AddCommand(&AddGroupCommand{})
	guild.AddCommand(&JoinGroupCommand{})
	guild.AddCommand(&ListGroupCommand{})
	guild.AddCommand(&LeaveGroupCommand{})
	guild.AddCommand(&PingCommand{})
	guild.AddCommand(&PurgeGroupCommand{})
	guild.AddCommand(&BestPonyCommand{})
	guild.AddCommand(&BanCommand{})
	guild.AddCommand(&DropCommand{})
	guild.AddCommand(&GiveCommand{})
	guild.AddCommand(&ListCommand{})
	guild.AddCommand(&FightCommand{"", 0})
	guild.AddCommand(&PickCommand{})
	guild.AddCommand(&RollCommand{})
	guild.AddCommand(&ListGuildsCommand{})
	guild.AddCommand(&AnnounceCommand{})
	guild.AddCommand(&QuickConfigCommand{})
	guild.AddCommand(&ScheduleCommand{})
	guild.AddCommand(&NextCommand{})
	guild.AddCommand(&AddEventCommand{})
	guild.AddCommand(&RemoveEventCommand{})
	guild.AddCommand(&AddBirthdayCommand{})
	guild.AddCommand(&RemindMeCommand{})
	guild.AddCommand(&AutoSilenceCommand{spammodule})
	guild.AddCommand(&WipeWelcomeCommand{})
	guild.AddCommand(&SilenceCommand{})
	guild.AddCommand(&UnsilenceCommand{})
	guild.AddCommand(&TimeCommand{})
	guild.AddCommand(&SetTimeZoneCommand{})
	guild.AddCommand(&NewCommand{})
	guild.AddCommand(&SearchCollectionCommand{})
	guild.AddCommand(&QuoteCommand{})
	guild.AddCommand(&AddQuoteCommand{})
	guild.AddCommand(&RemoveQuoteCommand{})
	guild.AddCommand(&SearchQuoteCommand{})
	guild.AddCommand(&RemoveAliasCommand{})

	if disableall {
		for k, _ := range guild.commands {
			guild.config.Command_disabled[k] = true
		}
		for _, v := range guild.modules {
			guild.config.Module_disabled[strings.ToLower(v.Name())] = true
		}
		guild.SaveConfig()
	}
	go guild.IdleCheckLoop()
	go guild.SwapStatusLoop()

	debug := "."
	if guild.config.Debug {
		debug = ".\n[DEBUG BUILD]"
	}
	guild.log.Log("[](/sbload)\n Sweetiebot version ", sb.version, " successfully loaded on ", g.Name, debug)
}
func GetChannelGuild(id string) *GuildInfo {
	g, ok := sb.GuildChannels[id]
	if !ok {
		return sb.guilds[SBitoa(sb.MainGuildID)]
	}
	return g
}
func GetGuildFromID(id string) *GuildInfo {
	g, ok := sb.guilds[id]
	if !ok {
		return sb.guilds[SBitoa(sb.MainGuildID)]
	}
	return g
}
func (info *GuildInfo) IsDebug(channel string) bool {
	debugchannel, isdebug := sb.DebugChannels[info.Guild.ID]
	if isdebug {
		return channel == debugchannel
	}
	return false
}
func SBTypingStart(s *discordgo.Session, t *discordgo.TypingStart) {
	info := GetChannelGuild(t.ChannelID)
	ApplyFuncRange(len(info.hooks.OnTypingStart), func(i int) {
		if info.ProcessModule("", info.hooks.OnTypingStart[i]) {
			info.hooks.OnTypingStart[i].OnTypingStart(info, t)
		}
	})
}
func SBMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil { // This shouldn't ever happen but we check for it anyway
		return
	}

	t := time.Now().UTC().Unix()
	sb.LastMessages[m.ChannelID] = t

	ch, err := sb.dg.State.Channel(m.ChannelID)
	private := true
	if err == nil { // Because of the magic of web development, we can get a message BEFORE the "channel created" packet for the channel being used by that message.
		private = ch.IsPrivate
	} else {
		fmt.Println("Error retrieving channel "+m.ChannelID+": ", err.Error())
	}

	var info *GuildInfo
	ismainguild := true
	if !private {
		info = GetChannelGuild(m.ChannelID)
		ismainguild = SBatoi(ch.GuildID) == sb.MainGuildID
	} else {
		info = sb.guilds[SBitoa(sb.MainGuildID)]
	}
	cid := SBatoi(m.ChannelID)
	isdebug := info.IsDebug(m.ChannelID)
	if isdebug && !info.config.Debug {
		return // we do this up here so the release build doesn't log messages in bot-debug, but debug builds still log messages from the rest of the channels
	}

	if cid != info.config.LogChannel && !private && ismainguild { // Log this message if it was sent to the main guild only.
		sb.db.AddMessage(SBatoi(m.ID), SBatoi(m.Author.ID), m.ContentWithMentionsReplaced(), cid, m.MentionEveryone, SBatoi(ch.GuildID))
	}
	if m.Author.ID == sb.SelfID { // ALWAYS discard any of our own messages before analysis.
		SBAddPings(info, m.Message) // If we're discarding a message we still need to add any pings to the ping table
		return
	}

	if boolXOR(info.config.Debug, isdebug) { // debug builds only respond to the debug channel, and release builds ignore it
		return
	}

	// Check if this is a command. If it is, process it as a command, otherwise process it with our modules.
	if len(m.Content) > 1 && m.Content[0] == '!' && (len(m.Content) < 2 || m.Content[1] != '!') { // We check for > 1 here because a single character can't possibly be a valid command
		_, isfree := info.config.FreeChannels[m.ChannelID]
		_, isOwner := sb.Owners[SBatoi(m.Author.ID)]
		isOwner = isOwner || m.Author.ID == info.Guild.OwnerID
		ignore := false
		ApplyFuncRange(len(info.hooks.OnCommand), func(i int) {
			if info.ProcessModule(m.ChannelID, info.hooks.OnCommand[i]) {
				ignore = ignore || info.hooks.OnCommand[i].OnCommand(info, m.Message)
			}
		})
		if ignore && !isOwner { // if true, a module wants us to ignore this command
			return
		}

		args := ParseArguments(m.Content[1:])
		arg := strings.ToLower(args[0])
		alias, ok := info.config.Aliases[arg]
		if ok {
			nargs := ParseArguments(alias)
			arg = strings.ToLower(nargs[0])
			args = append(nargs[1:], args...)
		}
		c, ok := info.commands[arg]
		if ok {
			cmdname := strings.ToLower(c.Name())
			cch := info.config.Command_channels[cmdname]
			_, disabled := info.config.Command_disabled[cmdname]
			_, restricted := sb.RestrictedCommands[cmdname]
			if disabled && !isOwner {
				return
			}
			if restricted && !ismainguild {
				return
			}
			if !private && len(cch) > 0 {
				_, reverse := cch["!"]
				_, ok = cch[m.ChannelID]
				if ok == reverse {
					return
				}
			}
			if err != nil || (!private && !isdebug && !isfree) { // Private channels are not limited, nor is the debug channel
				if info.commandlimit.check(info.config.Commandperduration, info.config.Commandmaxduration, t) { // if we've hit the saturation limit, post an error (which itself will only post if the error saturation limit hasn't been hit)
					info.log.Error(m.ChannelID, "You can't input more than "+strconv.Itoa(info.config.Commandperduration)+" commands every "+TimeDiff(time.Duration(info.config.Commandmaxduration)*time.Second)+"!")
					return
				}
				info.commandlimit.append(t)
			}
			if !isOwner && !info.UserHasAnyRole(m.Author.ID, info.config.Command_roles[cmdname]) {
				info.log.Error(m.ChannelID, "You don't have permission to run this command! Allowed Roles: "+info.GetRoles(c))
				return
			}

			cmdlimit := info.config.Command_limits[cmdname]
			if !isfree && cmdlimit > 0 {
				lastcmd := info.command_last[m.ChannelID][cmdname]
				if !RateLimit(&lastcmd, cmdlimit) {
					info.log.Error(m.ChannelID, "You can only run that command once every "+TimeDiff(time.Duration(cmdlimit)*time.Second)+"!")
					return
				}
				if len(info.command_last[m.ChannelID]) == 0 {
					info.command_last[m.ChannelID] = make(map[string]int64)
				}
				info.command_last[m.ChannelID][cmdname] = t
			}

			result, usepm := c.Process(args[1:], m.Message, info)
			if len(result) > 0 {
				targetchannel := m.ChannelID
				if usepm && !private {
					channel, err := s.UserChannelCreate(m.Author.ID)
					info.log.LogError("Error opening private channel: ", err)
					if err == nil {
						targetchannel = channel.ID
						if rand.Float32() < 0.01 {
							info.SendMessage(m.ChannelID, "Check your ~~privilege~~ Private Messages for my reply!")
						} else {
							info.SendMessage(m.ChannelID, "```Check your Private Messages for my reply!```")
						}
					}
				}

				for len(result) > 1999 { // discord has a 2000 character limit
					if result[0:3] == "```" {
						index := strings.LastIndex(result[:1996], "\n")
						if index < 10 { // Ensure we process at least 10 characters to prevent an infinite loop
							index = 1996
						}
						info.SendMessage(targetchannel, result[:index]+"```")
						result = "```" + result[index:]
					} else {
						index := strings.LastIndex(result[:1999], "\n")
						if index < 10 {
							index = 1999
						}
						info.SendMessage(targetchannel, result[:index])
						result = result[index:]
					}
				}
				info.SendMessage(targetchannel, result)
			}
		} else {
			if !info.config.IgnoreInvalidCommands {
				info.log.Error(m.ChannelID, "Sorry, "+args[0]+" is not a valid command.\nFor a list of valid commands, type !help.")
			}
		}
	} else {
		ApplyFuncRange(len(info.hooks.OnMessageCreate), func(i int) {
			if info.ProcessModule(m.ChannelID, info.hooks.OnMessageCreate[i]) {
				info.hooks.OnMessageCreate[i].OnMessageCreate(info, m.Message)
			}
		})
	}
}

func SBMessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	info := GetChannelGuild(m.ChannelID)
	if boolXOR(info.config.Debug, info.IsDebug(m.ChannelID)) {
		return
	}
	if m.Author == nil { // Discord sends an update message with an empty author when certain media links are posted
		original, err := s.ChannelMessage(m.ChannelID, m.ID)
		if err != nil {
			info.log.LogError("Error processing MessageUpdate: ", err)
			return // Fuck it, we can't process this
		}
		m.Author = original.Author
	}

	ch, err := sb.dg.State.Channel(m.ChannelID)
	info.log.LogError("Error retrieving channel ID "+m.ChannelID+": ", err)
	private := true
	if err == nil {
		private = ch.IsPrivate
	}
	cid := SBatoi(m.ChannelID)
	if cid != info.config.LogChannel && !private && SBatoi(ch.GuildID) == sb.MainGuildID { // Always ignore messages from the log channel
		sb.db.AddMessage(SBatoi(m.ID), SBatoi(m.Author.ID), m.ContentWithMentionsReplaced(), cid, m.MentionEveryone, SBatoi(ch.GuildID))
	}
	ApplyFuncRange(len(info.hooks.OnMessageUpdate), func(i int) {
		if info.ProcessModule(m.ChannelID, info.hooks.OnMessageUpdate[i]) {
			info.hooks.OnMessageUpdate[i].OnMessageUpdate(info, m.Message)
		}
	})
}
func SBMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	info := GetChannelGuild(m.ChannelID)
	if boolXOR(info.config.Debug, info.IsDebug(m.ChannelID)) {
		return
	}
	ApplyFuncRange(len(info.hooks.OnMessageDelete), func(i int) {
		if info.ProcessModule(m.ChannelID, info.hooks.OnMessageDelete[i]) {
			info.hooks.OnMessageDelete[i].OnMessageDelete(info, m.Message)
		}
	})
}
func SBMessageAck(s *discordgo.Session, m *discordgo.MessageAck) {
	info := GetChannelGuild(m.ChannelID)
	ApplyFuncRange(len(info.hooks.OnMessageAck), func(i int) {
		if info.ProcessModule(m.ChannelID, info.hooks.OnMessageAck[i]) {
			info.hooks.OnMessageAck[i].OnMessageAck(info, m)
		}
	})
}
func SBUserUpdate(s *discordgo.Session, m *discordgo.UserUpdate) { ProcessUser(m.User) }
func SBPresenceUpdate(s *discordgo.Session, m *discordgo.PresenceUpdate) {
	info := GetGuildFromID(m.GuildID)
	ProcessUser(m.User)
	ApplyFuncRange(len(info.hooks.OnPresenceUpdate), func(i int) {
		if info.ProcessModule("", info.hooks.OnPresenceUpdate[i]) {
			info.hooks.OnPresenceUpdate[i].OnPresenceUpdate(info, m)
		}
	})
}
func SBVoiceStateUpdate(s *discordgo.Session, m *discordgo.VoiceStateUpdate) {
	info := GetGuildFromID(m.GuildID)
	ApplyFuncRange(len(info.hooks.OnVoiceStateUpdate), func(i int) {
		if info.ProcessModule("", info.hooks.OnVoiceStateUpdate[i]) {
			info.hooks.OnVoiceStateUpdate[i].OnVoiceStateUpdate(info, m.VoiceState)
		}
	})
}
func SBGuildUpdate(s *discordgo.Session, m *discordgo.GuildUpdate) {
	info := GetChannelGuild(m.ID)
	info.log.Log("Guild update detected, updating ", m.Name)
	info.ProcessGuild(m.Guild)
	ApplyFuncRange(len(info.hooks.OnGuildUpdate), func(i int) {
		if info.ProcessModule("", info.hooks.OnGuildUpdate[i]) {
			info.hooks.OnGuildUpdate[i].OnGuildUpdate(info, m.Guild)
		}
	})
}
func SBGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	info := GetGuildFromID(m.GuildID)
	info.ProcessMember(m.Member)
	ApplyFuncRange(len(info.hooks.OnGuildMemberAdd), func(i int) {
		if info.ProcessModule("", info.hooks.OnGuildMemberAdd[i]) {
			info.hooks.OnGuildMemberAdd[i].OnGuildMemberAdd(info, m.Member)
		}
	})
}
func SBGuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	info := GetGuildFromID(m.GuildID)
	ApplyFuncRange(len(info.hooks.OnGuildMemberRemove), func(i int) {
		if info.ProcessModule("", info.hooks.OnGuildMemberRemove[i]) {
			info.hooks.OnGuildMemberRemove[i].OnGuildMemberRemove(info, m.Member)
		}
	})
}
func SBGuildMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	info := GetGuildFromID(m.GuildID)
	info.ProcessMember(m.Member)
	ApplyFuncRange(len(info.hooks.OnGuildMemberUpdate), func(i int) {
		if info.ProcessModule("", info.hooks.OnGuildMemberUpdate[i]) {
			info.hooks.OnGuildMemberUpdate[i].OnGuildMemberUpdate(info, m.Member)
		}
	})
}
func SBGuildBanAdd(s *discordgo.Session, m *discordgo.GuildBanAdd) {
	info := GetGuildFromID(m.GuildID)
	ApplyFuncRange(len(info.hooks.OnGuildBanAdd), func(i int) {
		if info.ProcessModule("", info.hooks.OnGuildBanAdd[i]) {
			info.hooks.OnGuildBanAdd[i].OnGuildBanAdd(info, m.GuildBan)
		}
	})
}
func SBGuildBanRemove(s *discordgo.Session, m *discordgo.GuildBanRemove) {
	info := GetGuildFromID(m.GuildID)
	ApplyFuncRange(len(info.hooks.OnGuildBanRemove), func(i int) {
		if info.ProcessModule("", info.hooks.OnGuildBanRemove[i]) {
			info.hooks.OnGuildBanRemove[i].OnGuildBanRemove(info, m.GuildBan)
		}
	})
}
func SBGuildCreate(s *discordgo.Session, m *discordgo.GuildCreate) { ProcessGuildCreate(m.Guild) }
func SBChannelCreate(s *discordgo.Session, c *discordgo.ChannelCreate) {
	guild, ok := sb.guilds[c.GuildID]
	if ok {
		sb.GuildChannels[c.ID] = guild
	}
}
func SBChannelDelete(s *discordgo.Session, c *discordgo.ChannelDelete) {
	delete(sb.GuildChannels, c.ID)
}
func ProcessUser(u *discordgo.User) uint64 {
	id := SBatoi(u.ID)
	sb.db.AddUser(id, u.Email, u.Username, u.Avatar, u.Verified)
	return id
}

func (info *GuildInfo) ProcessMember(u *discordgo.Member) {
	ProcessUser(u.User)

	t := time.Now().UTC()
	if len(u.JoinedAt) > 0 { // Parse join date and update user table only if it is less than our current first seen date.
		var err error
		t, err = time.Parse(time.RFC3339Nano, u.JoinedAt)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}
	sb.db.AddMember(SBatoi(u.User.ID), SBatoi(info.Guild.ID), t, u.Nick)
}

func ProcessGuildCreate(g *discordgo.Guild) {
	AttachToGuild(g)
}

func (info *GuildInfo) ProcessGuild(g *discordgo.Guild) {
	if len(g.Members) == 0 || len(g.Channels) == 0 || len(g.Roles) == 0 { // If this is true we were given half a guild update
		info.log.Log("Got half a guild update for " + g.Name)
		info.Guild.Name = g.Name
		info.Guild.Icon = g.Icon
		info.Guild.Region = g.Region
		info.Guild.AfkChannelID = g.AfkChannelID
		info.Guild.EmbedChannelID = g.EmbedChannelID
		info.Guild.OwnerID = g.OwnerID
		info.Guild.JoinedAt = g.JoinedAt
		info.Guild.Splash = g.Splash
		info.Guild.AfkTimeout = g.AfkTimeout
		info.Guild.VerificationLevel = g.VerificationLevel
		info.Guild.EmbedEnabled = g.EmbedEnabled
		info.Guild.Large = g.Large
		info.Guild.DefaultMessageNotifications = g.DefaultMessageNotifications
	} else {
		info.Guild = g
		for _, v := range info.Guild.Channels {
			sb.GuildChannels[v.ID] = info
		}
		for _, v := range g.Members {
			info.ProcessMember(v)
		}
	}
}

func (info *GuildInfo) FindChannelID(name string) string {
	channels := info.Guild.Channels
	for _, v := range channels {
		if v.Name == name {
			return v.ID
		}
	}

	return ""
}

func ApplyFuncRange(length int, fn func(i int)) {
	for i := 0; i < length; i++ {
		fn(i)
	}
}

func (info *GuildInfo) IdleCheckLoop() {
	for !sb.quit {
		channels := info.Guild.Channels
		if info.config.Debug { // override this in debug mode
			c, err := sb.dg.State.Channel(sb.DebugChannels[info.Guild.ID])
			if err == nil {
				channels = []*discordgo.Channel{c}
			} else {
				channels = []*discordgo.Channel{}
			}
		}
		for _, ch := range channels {
			t, exists := sb.LastMessages[ch.ID]
			if exists {
				diff := time.Now().UTC().Sub(time.Unix(t, 0))
				ApplyFuncRange(len(info.hooks.OnIdle), func(i int) {
					if info.ProcessModule(ch.ID, info.hooks.OnIdle[i]) && diff >= (time.Duration(info.hooks.OnIdle[i].IdlePeriod(info))*time.Second) {
						info.hooks.OnIdle[i].OnIdle(info, ch)
					}
				})
			}
		}

		ApplyFuncRange(len(info.hooks.OnTick), func(i int) {
			if info.ProcessModule("", info.hooks.OnTick[i]) {
				info.hooks.OnTick[i].OnTick(info)
			}
		})
		time.Sleep(30 * time.Second)
	}
}

func WaitForInput() {
	var input string
	fmt.Scanln(&input)
	sb.quit = true
}

func Initialize(Token string) {
	dbauth, _ := ioutil.ReadFile("db.auth")
	sb = &SweetieBot{
		version:            "0.8.1",
		Owners:             map[uint64]bool{95585199324143616: true, 98605232707080192: true},
		RestrictedCommands: map[string]bool{"search": true, "lastping": true, "setstatus": true},
		MainGuildID:        98609319519453184,
		DebugChannels:      map[string]string{"98609319519453184": "141710126628339712", "105443346608095232": "200112394494541824"},
		GuildChannels:      make(map[string]*GuildInfo),
		quit:               false,
		guilds:             make(map[string]*GuildInfo),
		LastMessages:       make(map[string]int64),
		MaxConfigSize:      1000000,
	}

	rand.Intn(10)
	for i := 0; i < 20+rand.Intn(20); i++ {
		rand.Intn(50)
	}

	db, err := DB_Load(&Log{0, nil}, "mysql", strings.TrimSpace(string(dbauth)))
	if err != nil {
		fmt.Println("Error loading database", err.Error())
		return
	}

	sb.db = db
	sb.dg, err = discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating discord session", err.Error())
		return
	}

	sb.dg.AddHandler(SBReady)
	sb.dg.AddHandler(SBTypingStart)
	sb.dg.AddHandler(SBMessageCreate)
	sb.dg.AddHandler(SBMessageUpdate)
	sb.dg.AddHandler(SBMessageDelete)
	sb.dg.AddHandler(SBMessageAck)
	sb.dg.AddHandler(SBUserUpdate)
	sb.dg.AddHandler(SBPresenceUpdate)
	sb.dg.AddHandler(SBVoiceStateUpdate)
	sb.dg.AddHandler(SBGuildUpdate)
	sb.dg.AddHandler(SBGuildMemberAdd)
	sb.dg.AddHandler(SBGuildMemberRemove)
	sb.dg.AddHandler(SBGuildMemberUpdate)
	sb.dg.AddHandler(SBGuildBanAdd)
	sb.dg.AddHandler(SBGuildBanRemove)
	sb.dg.AddHandler(SBGuildCreate)
	sb.dg.AddHandler(SBChannelCreate)
	sb.dg.AddHandler(SBChannelDelete)

	sb.db.LoadStatements()
	fmt.Println("Finished loading database statements")

	//BuildMarkov(1, 1)
	//return
	err = sb.dg.Open()
	if err == nil {
		fmt.Println("Connection established")
		for !sb.quit {
			time.Sleep(400 * time.Millisecond)
		}
	} else {
		fmt.Println("Error opening websocket connection: ", err.Error())
	}

	fmt.Println("Sweetiebot quitting")
	sb.dg.Close()
	sb.db.Close()
}
