package sweetiebot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blackhole12/discordgo"
)

// ChannelRegex matches any valid discord channel ping
var ChannelRegex = regexp.MustCompile("<#([0-9]+)>")

// UserRegex matches any valid user or nickname ping
var UserRegex = regexp.MustCompile("<\\\\?@!?([0-9]+)>")

// RoleRegex matches any valid role ping
var RoleRegex = regexp.MustCompile("<\\\\?@&([0-9]+)>")

var mentionRegex = regexp.MustCompile("<@(!|&)?[0-9]+>")
var discriminantregex = regexp.MustCompile(".*#[0-9][0-9][0-9]+")
var urlregex = regexp.MustCompile("https?:\\/\\/(www\\.)?[-a-zA-Z0-9@:%._\\+~#=]{1,256}\\.[a-z]{1,6}([-a-zA-Z0-9@:%_\\+.~#?&//=]*)")
var guildfileregex = regexp.MustCompile("^([0-9]+)[.]json$")

// DiscordEpoch is used to figure out snowflake creation times
const DiscordEpoch uint64 = 1420070400000

// BotVersion stores the current version of sweetiebot
var BotVersion = Version{1, 0, 0, 2}

const (
	MaxPublicLines    = 12
	maxPublicRules    = 15
	SilverServerID    = "105443346608095232"
	PatreonURL        = "https://www.patreon.com/erikmcclure"
	QuitNone          = 0
	QuitNow           = 1
	QuitRaid          = 2
	UpdateGrace       = 120
	MaxUpdateGrace    = 600
	UpdateInterval    = 300
	CleanInterval     = 3600
	ExpireTime        = 3600 * 72
	MaxScheduleRows   = 5000
	DelayTime         = time.Duration(200 * time.Millisecond)
	heartbeatInterval = time.Duration(20 * time.Second)
)

type deferPair struct {
	data interface{}
	info *GuildInfo
}

// SweetieBot is the primary bot object containing the bot state
type SweetieBot struct {
	DB              *BotDB
	DG              *DiscordGoSession
	Debug           bool `json:"debug"`
	changelog       map[int]string
	SelfID          DiscordUser
	SelfAvatar      string
	SelfName        string
	AppID           uint64
	AppName         string
	Owner           DiscordUser
	Token           string                          `json:"token"`
	DBAuth          string                          `json:"dbauth"`
	MainGuildID     DiscordGuild                    `json:"mainguildid"`
	DebugChannels   map[DiscordGuild]DiscordChannel `json:"debugchannels"`
	quit            uint32                          // QuitNone means to keep running. QuitNow means to quit immediately. QuitRaid means to wait until no raids have occurred before quitting
	Guilds          map[DiscordGuild]*GuildInfo
	GuildsLock      sync.RWMutex
	LastMessages    map[DiscordChannel]int64
	LastMessageLock sync.RWMutex
	MaxConfigSize   int    `json:"maxconfigsize"`
	MaxUniqueItems  uint64 `json:"maxuniqueitems"`
	StartTime       int64
	MessageCount    uint32 // 32-bit so we can do atomic ops on a 32-bit platform
	heartbeat       uint32 // perpetually incrementing heartbeat counter to detect deadlock
	locknumber      uint32
	loader          func(*GuildInfo) []Module
	memberChan      chan *GuildInfo
	deferChan       chan deferPair
	Selfhoster      *Selfhost
	IsUserMode      bool       `json:"runasuser"` // True if running as a user for some godawful reason
	WebSecure       bool       `json:"websecure"`
	WebDomain       string     `json:"webdomain"`
	WebPort         string     `json:"webport"`
	EmptyGuild      *GuildInfo // Holds an empty GuildInfo for running server independent commands
	UpdateLock      AtomicFlag
	Markov          *markovChain
}

type markovChain struct {
	Speakers []string                  `json:"speakers"`
	Phrases  []string                  `json:"phrases"`
	Mapping  []uint64                  `json:"mapping"` // speakerid in top 32 bits, phraseid in bottom 32 bits
	Chain    map[uint64]map[uint32]int `json:"chain"`   // prev in top 32 bits, prev2 in bottom 32 bits
}

// IsMainGuild returns true if that guild is considered the main (default) guild
func (sb *SweetieBot) IsMainGuild(info *GuildInfo) bool {
	return sb.MainGuildID.Equals(info.ID)
}

// ChannelIsPrivate returns true if channel should be considered private, false otherwise.
func (sb *SweetieBot) ChannelIsPrivate(channelID DiscordChannel) (*discordgo.Channel, bool) {
	if channelID == "heartbeat" {
		return nil, true
	}
	ch, err := sb.DG.State.Channel(channelID.String())
	if err == nil { // Because of the magic of web development, we can get a message BEFORE the "channel created" packet for the channel being used by that message.
		return ch, typeIsPrivate(ch.Type)
	}
	// Bots aren't supposed to be in Group DMs but can be grandfathered into them, and these channels will always fail to exist, so we simply ignore this error as harmless.
	return nil, true
}

//func (sb *SweetieBot) OnEvent(s *discordgo.Session, e *discordgo.Event) { ApplyFuncRange(len(info.hooks.OnEvent), func(i int) { if(ProcessModule("", info.hooks.OnEvent[i])) { info.hooks.OnEvent[i].OnEvent(s, e) } }) }

// OnReady discord hook
func (sb *SweetieBot) OnReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("Ready message receieved, waiting for guilds...")
	sb.SelfID = DiscordUser(r.User.ID)
	sb.SelfAvatar = r.User.Avatar
	sb.SelfName = r.User.Username
	sb.AppName = sb.SelfName
	if r.Guilds != nil && sb.IsUserMode {
		for _, G := range r.Guilds {
			sb.AttachToGuild(G)
		}
	}
	app, err := s.Application("@me")
	if err == nil {
		sb.Owner = DiscordUser(app.Owner.ID)
		sb.AppID = SBatoi(app.ID)
		sb.AppName = app.Name
	}

	sb.Selfhoster.SelfUpdate(sb.Owner)
}

type moduleArray []Module

func (f moduleArray) Len() int {
	return len(f)
}

func (f moduleArray) Less(i, j int) bool {
	return len(f[i].Commands()) > len(f[j].Commands())
}

func (f moduleArray) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// AttachToGuild adds a guild to sweetiebot's state tracking
func (sb *SweetieBot) AttachToGuild(g *discordgo.Guild) {
	sb.GuildsLock.RLock()
	guild, exists := sb.Guilds[DiscordGuild(g.ID)]
	sb.GuildsLock.RUnlock()
	if exists {
		sb.Selfhoster.CheckGuilds(map[DiscordGuild]*GuildInfo{DiscordGuild(g.ID): guild})
		guild.ProcessGuild(g)
		return
	}
	if sb.Debug {
		_, ok := sb.DebugChannels[DiscordGuild(g.ID)]
		if !ok {
			/*guild = NewGuildInfo(sb, g)
			sb.GuildsLock.Lock()
			sb.Guilds[DiscordGuild(g.ID)] = guild
			guild.ProcessGuild(g)
			sb.GuildsLock.Unlock()
			sb.memberChan <- guild
			//fmt.Println("Processed", g.Name)*/
			return
		}
	}

	fmt.Println("Initializing " + g.Name)
	guild = NewGuildInfo(sb, g)
	for _, m := range g.Members {
		if sb.SelfID.Equals(m.User.ID) {
			guild.BotNick = m.Nick
		}
	}
	config, err := ioutil.ReadFile(g.ID + ".json")
	disableall := false
	if err != nil {
		fmt.Println("New Guild Detected: " + g.Name)

		ch, e := sb.DG.UserChannelCreate(g.OwnerID)
		if e == nil {
			if sb.DB.Status.Get() {
				sb.DB.SetDefaultServer(SBatoi(g.OwnerID), SBatoi(g.ID)) // This ensures no one blows up another server by accident
			}
			perms, _ := guild.Bot.DG.UserPermissions(sb.SelfID, guild.ID)
			warning := ""
			if perms&discordgo.PermissionAdministrator != 0 {
				warning = "\nWARNING: You have given " + guild.GetBotName() + " the Administrator role, which implicitly gives it all roles! " + guild.GetBotName() + " only needs Ban Members, Manage Roles and Manage Messages in order to function correctly." + warning
			}
			if perms&discordgo.PermissionMentionEveryone != 0 {
				warning = "\nWARNING: You have given " + guild.GetBotName() + " the Mention Everyone role, which means users will be able to abuse it to ping everyone on the server! " + guild.GetBotName() + " does NOT attempt to filter @\u200Beveryone from it messages!" + warning
			}
			if perms&discordgo.PermissionBanMembers == 0 {
				warning = "\nWARNING: " + guild.GetBotName() + " cannot ban members spamming the welcome channel without the Ban Members role! (If you do not use this feature, it is safe to ignore this warning)." + warning
			}
			if perms&discordgo.PermissionManageRoles == 0 {
				warning = "\nWARNING: " + guild.GetBotName() + " cannot silence members or give birthday roles without the Manage Roles role!" + warning
			}
			if perms&discordgo.PermissionManageMessages == 0 {
				warning = "\nWARNING: " + guild.GetBotName() + " cannot delete messages without the Manage Messages role!" + warning
			}
			if perms&discordgo.PermissionManageServer == 0 {
				warning = "\nWARNING: " + guild.GetBotName() + " cannot engage lockdown mode without the Manage Server role!" + warning
			}
			sb.DG.ChannelMessageSend(ch.ID, "You've successfully added "+guild.GetBotName()+" to your server! To finish setting it up, run the `setup` command. Here is an explanation of the command and an example:\n```!setup <Mod Role> <Mod Channel> [Log Channel] [Member Role]```\n**> Mod Role**\nThis is a role shared by all the moderators and admins of your server. "+guild.GetBotName()+" will ping this role to alert you about potential raids or silenced users, and sensitive commands will be restricted so only users with the moderator role can use them. As the server owner, you will ALWAYS be able to run any command, no matter what. \n\n**> Mod Channel**\nThis is the channel "+guild.GetBotName()+" will post alerts on. Usually, this is your private moderation channel, but you can make it whatever channel you want. Just make sure you use the format `#channel`, and ensure the bot actually has permission to post messages on the channel.\n\n**> Log Channel**\nAn optional channel where "+guild.GetBotName()+" will post errors and update notifications. Usually, this is only visible to server admins and the bot. Ensure the bot has permission to post messages on the log channel, or you won't get any output. Providing a log channel is highly recommended, because it's often "+guild.GetBotName()+"'s last resort for notifying you about potential errors.\n\n**> Member Role**\nIf you already have a role that you assign to all members of your server, mention it here. Otherwise, leave this argument blank, and Sweetie will generate a new \"Member\" role and assign it to all your users.\n\nThat's it! Here is an example: ```!setup @Mods #staff-chat #bot-log @Member```")
			if len(warning) > 0 {
				sb.DG.ChannelMessageSend(ch.ID, warning)
			}
		} else {
			fmt.Println("Error sending introductory PM: ", e)
		}
		disableall = true
	} else if err := guild.MigrateSettings(config); err != nil {
		fmt.Println("Error reading config file for "+g.Name+": ", err.Error())
	}

	guild.Config.FillConfig()
	if sb.MainGuildID.Equals(g.ID) {
		guild.Silver.Set(true)
	}

	sb.GuildsLock.Lock()
	sb.Guilds[DiscordGuild(g.ID)] = guild
	guild.ProcessGuild(g) // This can be done outside of the guild lock, but it puts a lot of pressure on the database
	sb.GuildsLock.Unlock()
	sb.Selfhoster.CheckGuilds(map[DiscordGuild]*GuildInfo{DiscordGuild(g.ID): guild})

	if atomic.LoadUint32(&sb.quit) == QuitNone { // We can't check this inside the guild lock because it can deadlock if we run out of channel buffer, so we just run the risk of crashing while closing instead of closing nicely
		sb.memberChan <- guild // Do this concurrently because it just has to happen eventually.
	}

	guild.Modules = sb.loader(guild)
	sort.Sort(moduleArray(guild.Modules))

	for _, v := range guild.Modules {
		guild.RegisterModule(v)
		cmds := v.Commands()
		for _, command := range cmds {
			guild.AddCommand(command, v)
		}
	}

	guild.Clean()
	if sb.Debug {
		for _, v := range guild.Modules {
			_, ok := guild.commands[CommandID(strings.ToLower(v.Name()))]
			if ok {
				fmt.Println("WARNING: Ambiguous module/command name ", v.Name())
			}
		}
	}
	if disableall {
		for k := range guild.commands {
			guild.Config.Modules.CommandDisabled[k] = true
		}
		for _, v := range guild.Modules {
			guild.Config.Modules.Disabled[ModuleID(strings.ToLower(v.Name()))] = true
		}
		delete(guild.Config.Modules.CommandDisabled, "setup")
		delete(guild.Config.Modules.CommandDisabled, "about")
		guild.SaveConfig()
	}
	if sb.IsMainGuild(guild) {
		sb.DB.log = guild
	}

	debug := "."
	if sb.Debug {
		debug = ".\n[DEBUG BUILD]"
	}
	changes := ""
	if guild.Config.LastVersion != BotVersion.Integer() {
		guild.Config.LastVersion = BotVersion.Integer()
		guild.SaveConfig()
		var ok bool
		changes, ok = sb.changelog[BotVersion.Integer()]
		if ok {
			changes = "\nChangelog:\n" + changes
		}
		if guild.Silver.Get() {
			changes += "\n\nThank you for your support!"
		} else {
			changes += "\n\nPlease consider donating $1 to help pay for hosting costs: " + PatreonURL
		}
	}
	guild.Log(sb.AppName+" version ", BotVersion.String(), " successfully loaded on ", g.Name, debug, changes)
}
func (sb *SweetieBot) getChannelGuild(id string) *GuildInfo {
	c, err := sb.DG.State.Channel(id)
	if err != nil {
		fmt.Println("Failed to get channel " + id)
		return nil
	}
	return sb.getGuildFromID(c.GuildID)
}
func (sb *SweetieBot) getGuildFromID(id string) *GuildInfo {
	sb.GuildsLock.RLock()
	g, ok := sb.Guilds[DiscordGuild(id)]
	sb.GuildsLock.RUnlock()
	if !ok {
		return nil
	}
	return g
}
func (sb *SweetieBot) getAddMsg(info *GuildInfo) string {
	if info.Config.Basic.BotChannel != ChannelEmpty {
		addch, adderr := sb.DG.State.Channel(info.Config.Basic.BotChannel.String())
		if adderr == nil {
			return fmt.Sprintf(" Try going to #%s instead.", addch.Name)
		}
	}
	return ""
}

func (sb *SweetieBot) GetLastMessage(id DiscordChannel) (int64, bool) {
	sb.LastMessageLock.RLock()
	t, exists := sb.LastMessages[id]
	sb.LastMessageLock.RUnlock()
	return t, exists
}

// ProcessCommand processes a command given to sweetiebot in the form "!command"
func (sb *SweetieBot) ProcessCommand(m *discordgo.Message, info *GuildInfo, t int64, isdebug bool, private bool) {
	var prefix byte = '!'
	if info != nil && len(info.Config.Basic.CommandPrefix) == 1 {
		prefix = info.Config.Basic.CommandPrefix[0]
	}

	// Check if this is a command. If it is, process it as a command, otherwise process it with our modules.
	if len(m.Content) > 1 && m.Content[0] == prefix && (len(m.Content) < 2 || m.Content[1] != prefix) { // We check for > 1 here because a single character can't possibly be a valid command
		isfree := private
		authorid := SBatoi(m.Author.ID)
		channelID := DiscordChannel(m.ChannelID)
		if info != nil {
			_, isfree = info.Config.Basic.FreeChannels[channelID]
		}

		// command := strings.ToLower(strings.SplitN(m.Content[1:], " ", 2)[0])
		args, indices := ParseArguments(m.Content[1:])
		arg := CommandID(strings.ToLower(args[0]))
		if info == nil {
			info = sb.GetDefaultServer(authorid)
		}
		if info == nil {
			gIDs := []uint64{}
			if _, independent := sb.EmptyGuild.commands[arg]; !independent {
				if !sb.DB.Status.Get() {
					sb.DG.ChannelMessageSend(m.ChannelID, StringMap[STRING_DATABASE_ERROR])
					return
				}
				gIDs = sb.DB.GetUserGuilds(authorid)
				if len(gIDs) != 1 {
					sb.DG.ChannelMessageSend(m.ChannelID, StringMap[STRING_NO_SERVER])
					return
				}
			} else if sb.DB.Status.Get() {
				gIDs = sb.DB.GetUserGuilds(authorid)
			}

			if len(gIDs) == 1 {
				sb.GuildsLock.RLock()
				info = sb.Guilds[NewDiscordGuild(gIDs[0])]
				sb.GuildsLock.RUnlock()
			}

			if info == nil {
				info = sb.EmptyGuild
			}
		}

		c, ok := info.commands[arg] // First, we check if this matches an existing command so you can't alias yourself into a hole
		if !ok {
			if alias, aliasok := info.Config.Basic.Aliases[string(arg)]; aliasok {
				if len(indices) > 1 {
					m.Content = info.Config.Basic.CommandPrefix + alias + " " + m.Content[indices[1]:]
				} else {
					m.Content = info.Config.Basic.CommandPrefix + alias
				}
				args, indices = ParseArguments(m.Content[1:])
				if m.ChannelID != "heartbeat" && len(args) < 1 {
					info.SendError(channelID, "The "+string(arg)+" alias resolves to a blank command! Don't you know how dangerous that is?! That kind of abuse crashes bots! Go to your room and don't come back down until you've fixed that alias using '"+info.Config.Basic.CommandPrefix+"setconfig basic.aliases "+string(arg)+" [something else]', or leave out the fourth argument entirely if you want to delete it!", t)
					return
				}
				arg = CommandID(strings.ToLower(args[0]))
				c, ok = info.commands[arg]
			}
		}
		if ok {
			if sb.DB.Status.Get() && !sb.SelfID.Equals(m.Author.ID) {
				sb.DB.Audit(AuditTypeCommand, m.Author, m.Content, SBatoi(info.ID))
			}
			cmdname := CommandID(strings.ToLower(c.Info().Name))
			if m.ChannelID != "heartbeat" && !info.Config.SetupDone && cmdname != CommandID("setup") {
				info.SendError(channelID, "You haven't set up the bot yet! Run the !setup command first and follow the instructions.", t)
				return
			}

			ignore := false
			if !private {
				ignore = info.checkOnCommand(m)
			}

			cch := info.Config.Modules.CommandChannels[cmdname]
			if !private && len(cch) > 0 {
				_, reverse := cch["!"]
				_, ok = cch[channelID]
				ignore = ignore || ok == reverse
			}

			bypass, err := info.UserCanUseCommand(DiscordUser(m.Author.ID), c, ignore) // Bypass is true for administrators, mods, and the bot owner
			if m.ChannelID == "heartbeat" {                                            // The heartbeat can never be ignored or disabled
				bypass = true
				err = nil
			} else if err == errDisabled || err == errIgnored || err == errSilenced || err == errMainGuild {
				return
			}

			if !isdebug && !isfree && !bypass && info.Config.Modules.CommandPerDuration > 0 { // debug channels aren't limited
				if len(info.commandlimit.times) < info.Config.Modules.CommandPerDuration*2 { // Check if we need to re-allocate the array because the configuration changed
					info.commandlimit.times = make([]int64, info.Config.Modules.CommandPerDuration*2, info.Config.Modules.CommandPerDuration*2)
				}
				if info.commandlimit.check(info.Config.Modules.CommandPerDuration, info.Config.Modules.CommandMaxDuration, t) { // if we've hit the saturation limit, post an error (which itself will only post if the error saturation limit hasn't been hit)
					info.SendError(channelID, fmt.Sprintf(StringMap[STRING_COMMANDS_LIMIT], info.Config.Modules.CommandPerDuration, TimeDiff(time.Duration(info.Config.Modules.CommandMaxDuration)*time.Second), sb.getAddMsg(info)), t)
					return
				}
				info.commandlimit.append(t)
			}
			if err != nil {
				info.SendError(channelID, err.Error(), t)
				return
			}

			if c.Info().Silver && !info.Silver.Get() {
				info.SendError(channelID, "That command is for Silver supporters only. Server owners can donate $1 a month to gain access: "+PatreonURL+". Visit the support channel for help if you already donated.", t)
				return
			}

			cmdlimit := info.Config.Modules.CommandLimits[cmdname]
			if !isfree && cmdlimit > 0 && !bypass {
				cmdhash := channelID.String() + string(cmdname)
				info.commandLock.RLock()
				lastcmd := info.commandLast[cmdhash]
				info.commandLock.RUnlock()
				if !RateLimit(&lastcmd, cmdlimit, t) {
					info.SendError(channelID, fmt.Sprintf(StringMap[STRING_COMMAND_LIMIT], TimeDiff(time.Duration(cmdlimit)*time.Second), sb.getAddMsg(info)), t)
					return
				}
				info.commandLock.Lock()
				info.commandLast[cmdhash] = t
				info.commandLock.Unlock()
			}

			result, usepm, resultembed := c.Process(args[1:], m, indices[1:], info)
			if len(result) > 0 || resultembed != nil {
				targetchannel := channelID
				if usepm && !private {
					channel, err := sb.DG.UserChannelCreate(m.Author.ID)
					if err == nil {
						targetchannel = DiscordChannel(channel.ID)
						private = true
						if rand.Float32() < 0.01 {
							info.SendMessage(channelID, "Check your ~~privilege~~ Private Messages for my reply!")
						} else {
							info.SendMessage(channelID, StringMap[STRING_CHECK_PM])
						}
					} else {
						info.SendError(channelID, StringMap[STRING_PM_FAILURE], t)
					}
				}

				if resultembed != nil {
					if err := info.SendEmbed(targetchannel, resultembed); err != nil {
						fmt.Println(err)
					}
				} else if err := info.SendMessage(targetchannel, result); err != nil {
					fmt.Println(err)
				}
			}
		} else if !info.Config.Basic.IgnoreInvalidCommands {
			if private || !info.checkOnCommand(m) {
				info.SendError(channelID, fmt.Sprintf(StringMap[STRING_INVALID_COMMAND], args[0]), t)
			}
		}
	} else if info != nil { // If info is nil this was sent through a private message so just ignore it completely
		for _, h := range info.hooks.OnMessageCreate {
			if info.ProcessModule(DiscordChannel(m.ChannelID), h) {
				h.OnMessageCreate(info, m)
			}
		}
	}
}

// MessageCreate discord hook
func (sb *SweetieBot) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	atomic.AddUint32(&sb.MessageCount, 1)

	if m.Type != discordgo.MessageTypeDefault || m.Author == nil { // Ignore all messages that are of a special type or that have no author
		return
	}

	channelID := DiscordChannel(m.ChannelID)
	t := GetTimestamp(m.Message).Unix()
	if m.Author.ID != sb.SelfID.String() {
		sb.LastMessageLock.Lock()
		sb.LastMessages[channelID] = t
		sb.LastMessageLock.Unlock()
	}

	_, private := sb.ChannelIsPrivate(channelID)
	var info *GuildInfo
	isdebug := false
	if !private {
		info = sb.getChannelGuild(m.ChannelID)
		if info == nil {
			return
		}
		isdebug = info.IsDebug(channelID)
	}

	if isdebug && !sb.Debug {
		return // we do this up here so the release build doesn't log messages in bot-debug, but debug builds still log messages from the rest of the channels
	}
	if m.ChannelID != "heartbeat" {
		if info != nil && info.Silver.Get() && sb.DB.CheckStatus() { // Log message on silver guilds
			if channelID != info.Config.Log.Channel {
				sb.deferChan <- deferPair{m, info}
			}
		}
		if info != nil {
			sb.deferChan <- deferPair{m.Author, info}
		}
		if sb.SelfID.Equals(m.Author.ID) { // discard all our own messages (unless this is a heartbeat message)
			return
		}
		if info != nil && !info.Config.Basic.ListenToBots && m.Author.Bot { // If we aren't supposed to listen to bot messages, discard them.
			return
		}
		if boolXOR(sb.Debug, isdebug) { // debug builds only respond to the debug channel, and release builds ignore it
			return
		}
	} else {
		sb.GuildsLock.RLock()
		info, _ = sb.Guilds[sb.MainGuildID]
		sb.GuildsLock.RUnlock()
		if info == nil {
			fmt.Println("Failed to get main guild during heartbeat test!")
		}
	}

	sb.ProcessCommand(m.Message, info, t, isdebug, private)
}

// MessageUpdate discord hook
func (sb *SweetieBot) MessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	info := sb.getChannelGuild(m.ChannelID)
	if info == nil {
		return
	}
	channelID := DiscordChannel(m.ChannelID)
	if boolXOR(sb.Debug, info.IsDebug(channelID)) {
		return
	}
	if m.Type != discordgo.MessageTypeDefault || m.Author == nil { // Discord sends an update message with an empty author when certain media links are posted
		return // Because this only happens for media links and causes all sorts of problems, we just don't process messages without an author.
	}

	ch, err := sb.DG.State.Channel(m.ChannelID)
	info.LogError("Error retrieving channel ID "+m.ChannelID+": ", err)
	private := true
	if err == nil {
		private = typeIsPrivate(ch.Type)
	}
	if channelID != info.Config.Log.Channel && !private && info.Silver.Get() && sb.DB.CheckStatus() { // Always ignore messages from the log channel
		sb.DB.AddMessage(SBatoi(m.ID), m.Author, info.Sanitize(m.Content, CleanMentions|CleanPings), channelID.Convert(), SBatoi(ch.GuildID))
	}
	if sb.SelfID.Equals(m.Author.ID) {
		return
	}
	for _, h := range info.hooks.OnMessageUpdate {
		if info.ProcessModule(channelID, h) {
			h.OnMessageUpdate(info, m.Message)
		}
	}
}

// MessageDelete discord hook
func (sb *SweetieBot) MessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	info := sb.getChannelGuild(m.ChannelID)
	if info == nil {
		return
	}
	channelID := DiscordChannel(m.ChannelID)
	if boolXOR(sb.Debug, info.IsDebug(channelID)) {
		return
	}
	for _, h := range info.hooks.OnMessageDelete {
		if info.ProcessModule(channelID, h) {
			h.OnMessageDelete(info, m.Message)
		}
	}
}

// UserUpdate discord hook
func (sb *SweetieBot) UserUpdate(s *discordgo.Session, u *discordgo.UserUpdate) {
	sb.deferChan <- deferPair{u, nil}
}

// GuildUpdate discord hook
func (sb *SweetieBot) GuildUpdate(s *discordgo.Session, m *discordgo.GuildUpdate) {
	info := sb.getChannelGuild(m.ID)
	if info == nil {
		return
	}
	fmt.Println("Guild update detected, updating", m.Name)
	sb.Selfhoster.CheckGuilds(map[DiscordGuild]*GuildInfo{DiscordGuild(info.ID): info})
	info.ProcessGuild(m.Guild)

	for _, h := range info.hooks.OnGuildUpdate {
		if info.ProcessModule("", h) {
			h.OnGuildUpdate(info, m.Guild)
		}
	}
}

// GuildMemberAdd discord hook
func (sb *SweetieBot) GuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	info := sb.getGuildFromID(m.GuildID)
	if info == nil {
		return
	}
	info.ProcessMember(m.Member)

	if info.ID == SilverServerID && sb.Selfhoster.CheckDonor(m.Member) {
		sb.GuildsLock.RLock()
		sb.Selfhoster.CheckGuilds(sb.Guilds)
		sb.GuildsLock.RUnlock()
	}

	for _, h := range info.hooks.OnGuildMemberAdd {
		if info.ProcessModule("", h) {
			h.OnGuildMemberAdd(info, m.Member, time.Now().UTC())
		}
	}
}

// GuildMemberRemove discord hook
func (sb *SweetieBot) GuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	info := sb.getGuildFromID(m.GuildID)
	if info == nil {
		return
	}
	userID := DiscordUser(m.User.ID)
	if sb.DB.CheckStatus() {
		sb.DB.RemoveMember(userID.Convert(), SBatoi(info.ID))
	}

	if info.ID == SilverServerID {
		if _, check := sb.Selfhoster.Donors.Load(m.User.ID); check {
			sb.Selfhoster.Donors.Delete(m.User.ID)
			sb.GuildsLock.RLock()
			sb.Selfhoster.CheckGuilds(sb.Guilds)
			sb.GuildsLock.RUnlock()
		}
	}

	for _, h := range info.hooks.OnGuildMemberRemove {
		if info.ProcessModule("", h) {
			h.OnGuildMemberRemove(info, m.Member, time.Now().UTC())
		}
	}

	if userID == sb.SelfID {
		fmt.Println("Sweetie was removed from", info.Name)
		sb.GuildsLock.Lock()
		delete(sb.Guilds, DiscordGuild(info.ID))
		sb.GuildsLock.Unlock()
	}
}

// GuildMemberUpdate discord hook
func (sb *SweetieBot) GuildMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	info := sb.getGuildFromID(m.GuildID)
	if info == nil {
		return
	}

	if sb.SelfID.Equals(m.User.ID) && len(m.Nick) > 0 {
		info.BotNick = m.Nick
	}

	sb.deferChan <- deferPair{m, info}
	if info.ID == SilverServerID && sb.Selfhoster.CheckDonor(m.Member) {
		sb.GuildsLock.RLock()
		sb.Selfhoster.CheckGuilds(sb.Guilds)
		sb.GuildsLock.RUnlock()
	}

	for _, h := range info.hooks.OnGuildMemberUpdate {
		if info.ProcessModule("", h) {
			h.OnGuildMemberUpdate(info, m.Member, time.Now().UTC())
		}
	}
}

// GuildBanAdd discord hook
func (sb *SweetieBot) GuildBanAdd(s *discordgo.Session, m *discordgo.GuildBanAdd) {
	info := sb.getGuildFromID(m.GuildID) // We don't actually need to resolve this to get the guildID for SawBan, but we want to ignore any guilds we get messages from that we aren't currently attached to.
	if info == nil {
		return
	}

	for _, h := range info.hooks.OnGuildBanAdd {
		if info.ProcessModule("", h) {
			h.OnGuildBanAdd(info, m)
		}
	}
}

// GuildBanRemove discord hook
func (sb *SweetieBot) GuildBanRemove(s *discordgo.Session, m *discordgo.GuildBanRemove) {
	info := sb.getGuildFromID(m.GuildID)
	if info == nil {
		return
	}

	for _, h := range info.hooks.OnGuildBanRemove {
		if info.ProcessModule("", h) {
			h.OnGuildBanRemove(info, m)
		}
	}
}

// GuildRoleDelete discord hook
func (sb *SweetieBot) GuildRoleDelete(s *discordgo.Session, m *discordgo.GuildRoleDelete) {
	info := sb.getGuildFromID(m.GuildID)
	if info == nil {
		return
	}

	for _, h := range info.hooks.OnGuildRoleDelete {
		if info.ProcessModule("", h) {
			h.OnGuildRoleDelete(info, m)
		}
	}
}

// GuildCreate discord hook
func (sb *SweetieBot) GuildCreate(s *discordgo.Session, m *discordgo.GuildCreate) {
	sb.AttachToGuild(m.Guild)
}

// GuildDelete discord hook
func (sb *SweetieBot) GuildDelete(s *discordgo.Session, m *discordgo.GuildDelete) {
	if !m.Unavailable {
		fmt.Println("Sweetie was deleted from", m.Guild.Name)
		sb.GuildsLock.Lock()
		delete(sb.Guilds, DiscordGuild(m.Guild.ID))
		sb.GuildsLock.Unlock()
	}
}

// ChannelCreate discord hook
func (sb *SweetieBot) ChannelCreate(s *discordgo.Session, c *discordgo.ChannelCreate) {
	info := sb.getGuildFromID(c.GuildID)
	if info == nil {
		return
	}

	info.setupSilenceRole()
}

// FindServers matches server names against a string
func (sb *SweetieBot) FindServers(name string, guilds []uint64) []*GuildInfo {
	name = strings.ToLower(name)
	info := make([]*GuildInfo, 0, len(guilds))
	for _, g := range guilds {
		sb.GuildsLock.RLock()
		guild, ok := sb.Guilds[NewDiscordGuild(g)]
		sb.GuildsLock.RUnlock()
		if ok {
			n := strings.ToLower(guild.Name)
			if len(n) > 0 {
				if n == name { // if these are an EXACT match, throw away the other results and just return this
					return []*GuildInfo{guild}
				}
				if strings.Contains(n, name) {
					info = append(info, guild)
				}
			} else {
				info = append(info, guild)
			}
		}
	}
	return info
}

// GetDefaultServer attempts to find the default server for a user
func (sb *SweetieBot) GetDefaultServer(user uint64) *GuildInfo {
	if !sb.DB.Status.Get() {
		return nil
	}
	_, _, _, server := sb.DB.GetUser(user)
	if server == nil {
		return nil
	}
	sb.GuildsLock.RLock()
	defer sb.GuildsLock.RUnlock()
	info, ok := sb.Guilds[NewDiscordGuild(*server)]
	if !ok {
		return nil
	}
	return info
}

// ProcessUser adds a user to the database
func (sb *SweetieBot) ProcessUser(u *discordgo.User) uint64 {
	id := SBatoi(u.ID)
	discriminator, _ := strconv.Atoi(u.Discriminator)
	if sb.DB.CheckStatus() {
		sb.DB.AddUser(id, u.Username, discriminator, false)
	}
	return id
}

func (sb *SweetieBot) memberIngestionLoop() {
	for atomic.LoadUint32(&sb.quit) != QuitNow {
		guild, more := <-sb.memberChan
		if !more {
			return
		}
		//fmt.Println("Member processing for: " + guild.Name)
		members := []*discordgo.Member{}
		lastid := ""
		for {
			m, err := sb.DG.GuildMembers(guild.ID, lastid, 999)
			if err != nil || len(m) == 0 {
				break
			}
			members = append(members, m...)
			lastid = m[len(m)-1].User.ID
		}
		if guild.ID == SilverServerID {
			for _, m := range members {
				sb.Selfhoster.CheckDonor(m)
			}
			sb.GuildsLock.RLock()
			sb.Selfhoster.CheckGuilds(sb.Guilds)
			sb.GuildsLock.RUnlock()
		}
		for i := range members { // Put the guildID back in because discord is stupid
			members[i].GuildID = guild.ID
			sb.DG.State.MemberAdd(members[i])
		}
		sb.Selfhoster.CheckGuilds(map[DiscordGuild]*GuildInfo{DiscordGuild(guild.ID): guild})
	}
}

func (sb *SweetieBot) deferProcessing() {
	for atomic.LoadUint32(&sb.quit) != QuitNow {
		pair := <-sb.deferChan
		if sb.DB.Status.Get() {
			switch v := pair.data.(type) {
			case *discordgo.MessageCreate:
				sb.DB.AddMessage(SBatoi(v.ID), v.Author, pair.info.Sanitize(v.Content, CleanMentions|CleanPings), SBatoi(v.ChannelID), SBatoi(pair.info.ID))
			case *discordgo.User:
				sb.DB.SentMessage(SBatoi(v.ID), SBatoi(pair.info.ID))
				sb.DB.SawUser(SBatoi(v.ID), v.Username)
			case *discordgo.UserUpdate:
				sb.ProcessUser(v.User)
			case *discordgo.GuildMemberUpdate:
				pair.info.ProcessMember(v.Member)
			}
		}
	}
}

func (sb *SweetieBot) idleCheckLoop() {
	for atomic.LoadUint32(&sb.quit) != QuitNow {
		sb.DB.CheckStatus()
		sb.GuildsLock.RLock()
		infos := make([]*GuildInfo, 0, len(sb.Guilds))
		for _, v := range sb.Guilds {
			infos = append(infos, v)
		}
		sb.GuildsLock.RUnlock()
		tm := time.Now()

		for _, info := range infos {
			for _, h := range info.hooks.OnTick {
				if info.ProcessModule("", h) {
					h.OnTick(info, tm)
				}
			}
		}

		fmt.Println("Idle Check: ", tm)
		time.Sleep(20 * time.Second)
	}
}

func (sb *SweetieBot) deadlockTestFunc(s *discordgo.Session, m *discordgo.MessageCreate) {
	sb.DG.State.RLock()
	sb.DG.State.RUnlock()
	sb.locknumber++
	sb.DG.RLock()
	sb.DG.RUnlock()
	sb.locknumber++
	sb.GuildsLock.RLock()
	sb.GuildsLock.RUnlock()
	sb.locknumber++
	sb.MessageCreate(&sb.DG.Session, m)
}

func (sb *SweetieBot) deadlockDetector() {
	var counter = sb.heartbeat
	var missed = 0
	var info *GuildInfo
	time.Sleep(heartbeatInterval) // Give sweetie time to load everything first before initiating heartbeats

	for {
		sb.GuildsLock.RLock()
		info, _ = sb.Guilds[sb.MainGuildID]
		sb.GuildsLock.RUnlock()

		if info != nil {
			break
		}

		fmt.Println(sb.MainGuildID, "MAIN GUILD CANNOT BE FOUND! Deadlock detector is nonfunctional until this is addressed.")
		time.Sleep(heartbeatInterval)
	}

	for atomic.LoadUint32(&sb.quit) != QuitNow {
		m := discordgo.MessageCreate{
			&discordgo.Message{ChannelID: "heartbeat", Content: info.Config.Basic.CommandPrefix + "about",
				Author: &discordgo.User{
					ID:       sb.SelfID.String(),
					Verified: true,
					Bot:      true,
				},
				Timestamp: discordgo.Timestamp(time.Now().UTC().Format(time.RFC3339Nano)),
			},
		}
		sb.locknumber = 0
		go sb.deadlockTestFunc(&sb.DG.Session, &m) // Do this in another thread so the deadlock detector doesn't deadlock
		time.Sleep(heartbeatInterval)
		if atomic.LoadUint32(&sb.heartbeat) == counter+1 {
			counter++
			missed = 0
		} else {
			missed++
			fmt.Println("MISSED HEARTBEAT SIGNAL ", missed, " TIMES IN A ROW")
			counter = atomic.LoadUint32(&sb.heartbeat)
		}
		if missed >= 5 {
			fmt.Println("FATAL ERROR: DEADLOCK DETECTED! (", sb.locknumber, ") TERMINATING PROGRAM...")
			name := fmt.Sprintf("stacktrace_%v.txt", time.Now().UTC().Unix())
			if f, err := os.Create(name); err == nil {
				pprof.Lookup("goroutine").WriteTo(f, 1)
				f.Close()
			}

			if ch, e := sb.DG.UserChannelCreate(sb.Owner.String()); e == nil {
				go sb.DG.ChannelMessageSend(ch.ID, "Deadlock occured, stacktrace written to "+name)
			}
			time.Sleep(1 * time.Second)

			os.Exit(-1)
		}
	}
}

// New creates and initializes a new instance of Sweetiebot that's ready to connect. Returns nil on error.
func New(token string, loader func(*GuildInfo) []Module) *SweetieBot {
	path, _ := GetCurrentDir()
	selfhoster := &Selfhost{SelfhostBase{BotVersion.Integer()}, AtomicBool{0}, sync.Map{}}
	rand.Seed(time.Now().UTC().Unix())

	hostfile, gerr := ioutil.ReadFile("selfhost.json")
	if gerr != nil {

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Fatal error, press enter to exit: ", r)
				input := ""
				fmt.Scanln(&input)
				os.Exit(1)
			}
		}()
		Install(path, selfhoster)
	}

	sb := &SweetieBot{
		Token:          token,
		SelfName:       "Sweetie Bot",
		AppName:        "Sweetie Bot",
		DebugChannels:  make(map[DiscordGuild]DiscordChannel),
		Guilds:         make(map[DiscordGuild]*GuildInfo),
		LastMessages:   make(map[DiscordChannel]int64),
		MaxConfigSize:  1000000,
		MaxUniqueItems: 25000,
		StartTime:      time.Now().UTC().Unix(),
		heartbeat:      4294967290,
		loader:         loader,
		memberChan:     make(chan *GuildInfo, 2500),
		deferChan:      make(chan deferPair, 2000),
		Selfhoster:     selfhoster,
		WebSecure:      false,
		WebDomain:      "localhost",
		WebPort:        ":80",
		changelog: map[int]string{
			AssembleVersion(1, 0, 0, 2):  "- Removed selfhosting support until further notice\n- Sweetiebot now builds static version of the website.",
			AssembleVersion(1, 0, 0, 1):  "- You can no longer !ban or !silence mods or admins.\n- !import now accepts server IDs instead of just names.\n- Using Member Role silencing is now optional when setting up a new server (but still highly recommended).",
			AssembleVersion(1, 0, 0, 0):  "- Fixed hidden newuserrole dependency.\n- Introduced Member role silencing, which solves rate limiting problems during raids. To enable this, use !help SetMemberRole for more information.",
			AssembleVersion(0, 9, 9, 34): "- Fixed resilencer race condition\n- Fixed alias crash bug\n- Improved documentation.",
			AssembleVersion(0, 9, 9, 33): "- Changed how language override works",
			AssembleVersion(0, 9, 9, 32): "- Security hotfix",
			AssembleVersion(0, 9, 9, 31): "- Fixed crash bug\n- Added language override file\n- No longer allow leaving and rejoining a server to clear silence status",
			AssembleVersion(0, 9, 9, 30): "- Fixed permissions error on import",
			AssembleVersion(0, 9, 9, 29): "- Moved markov chain to in-memory representation.\n- Cleaned up database\n- Add exponential backoff option to bored module.\n- Detects if it can't send an embed to a channel and sends a warning instead.\n- All edits show up in message log search\n- Changed how !wipe parses it's arguments. Please check !help wipe\n- Added season 8 transcripts\n- Any administrator can now grant server silver.",
			AssembleVersion(0, 9, 9, 28): "- Increase max rules\n- add mismatched parentheses check.",
			AssembleVersion(0, 9, 9, 27): "- Update documentation\n- Updated the server and dependencies\n- Optimized tag queries with lots of OR statements\n- Added 'new member' role to the users module, a role that is added to all new members for a limited time after joining.",
			AssembleVersion(0, 9, 9, 26): "- Fixed crash in !search.\n- Fixed installer SQL script.\n- !setconfig timezonelocation now verifies the value and warns you if it is invalid.\n- Omitting the channel in !wipe now simply defaults to the current channel.",
			AssembleVersion(0, 9, 9, 25): "- Changed !autosilence command to !raidsilence and migrated any existing aliases.\n- The bot now tells the user if a PM failed to be sent.\n- The bot now yells at you if you haven't set it up on the server yet.\n- Added a silence timeout even though this is a bad idea becuase you all wanted it so damn bad.\n- Added a counter module for all your counting needs.\n- Setting a config string value to \"\" will now actually delete the string value.",
			AssembleVersion(0, 9, 9, 24): "- Fix updater issue on linux\n- provide zip files instead of raw files for downloads\n- Fix timezones on windows without go installations\n- more idiotproofing",
			AssembleVersion(0, 9, 9, 23): "- Fixed crash in RolesModule",
			AssembleVersion(0, 9, 9, 22): "- Fixed crash in RolesModule and FilterModule",
			AssembleVersion(0, 9, 9, 21): "- Put lastmessages back on a lock for better performance\n- Fix docker-specific bugs, add docker self-hosting image, because docker is cool now.",
			AssembleVersion(0, 9, 9, 20): "- Improve locking situation, begin concentrated effort to find and eliminate deadlocks via stacktraces\n- The bot now yells at you if you try to set your timezone to Etc/GMT±00",
			AssembleVersion(0, 9, 9, 19): "- Fix installer and silver permissions handling\n- Removed polls module, replaced with !poll command in the Misc module that analyzes emoji reaction polls instead.",
			AssembleVersion(0, 9, 9, 18): "- Fix database cleanup to preserve banned user comments.",
			AssembleVersion(0, 9, 9, 17): "- Fix potential detection failure in deadlock detector.",
			AssembleVersion(0, 9, 9, 16): "- Fix installer on linux\n- Upgrade transcripts in database\n- Better 64-bit support",
			AssembleVersion(0, 9, 9, 15): "- No longer attempts to track embed message updates\n- Ignores new member join messages and other special messages\n- Re-added echoembed command\n- Autosilencing now include a reason for the silence\n- Filters can now add pressure when triggered, and can be configured to not remove the message at all. Check the documentation for details\n- Filters are no longer applied to bots/mods/admins.\n- Ownership changes are properly tracked\n- RemoveEvent now works on repeating events\n- Sweetiebot now accepts escaped user pings and role mentions in the form <\\@12345> or <\\@&12345>. This won't ping the role/user, but still allows you to specify an exact ID.\n- Now has a 200ms delay before deleting messages to prevent ghost messages.\n- Ensure any user who sends a message will always have their username as an alias, even if it was missed before.",
			AssembleVersion(0, 9, 9, 14): "- Fuck Daylight Savings\n- Also, fuck timezones\n- Prevent silenced members from using emoji reactions.\n- Removed main instance status loop (still available on selfhost instances)\n- Can no longer search for a user that is not in your server. If you need to search for a banned user, ping them using the ID or specify username#1234. This makes searches much faster.",
			AssembleVersion(0, 9, 9, 13): "- Made some error messages more clear\n- Fixed database cleanup functions\n- Sweetiebot now deletes all information about guilds she hasn't been on for 3 days.",
			AssembleVersion(0, 9, 9, 12): "- Fix crash on !setfilter",
			AssembleVersion(0, 9, 9, 11): "- Merged !showroll command\n- Prevented setting your default server to one that isn't set up.",
			AssembleVersion(0, 9, 9, 10): "- Sweetiebot no longer inserts !quote and !drop into the bored commands after restarting, unless the bored commands are empty; if you need to disable bored, disable the module instead.\n- Exorcised demons from three servers with corrupted channel information.\n- Filters now applied to invalid commands.",
			AssembleVersion(0, 9, 9, 9):  "- Fix lastseen values\n- Fix missing access error message when sweetie doesn't have read message history permissions.",
			AssembleVersion(0, 9, 9, 8):  "- Restore old functionality of !echo\n- say whether a user was autosilenced upon joining.",
			AssembleVersion(0, 9, 9, 7):  "- Added !createroll\n- !setconfig now accepts arbitrary strings, without quotes, in basic and [map] settings. Quotes are still required for [list] and [maplist] settings. Deletion NO LONGER USES \"\" in [map] settings. Simply pass nothing to delete a key.\n- Fixed display problem in !getconfig, which now displays lists in alphabetical order.",
			AssembleVersion(0, 9, 9, 6):  "- Update to go v1.9.3\n- Improve database error handling.\n- Fix chatlog race condition.",
			AssembleVersion(0, 9, 9, 5):  "- Message logging is now deferred to a single thread to prevent database deadlocking.\n- Username lookup now does fuzzy lookups on all aliases\n- Only retains 10 most used aliases.",
			AssembleVersion(0, 9, 9, 4):  "- Fix crash",
			AssembleVersion(0, 9, 9, 3):  "- Sweetie Bot no longer tracks presence updates, because they were the cause of the database slowdowns. This means !lastseen will only operate on last message sent.\n- Fixed !search.\n- Added !assignrole",
			AssembleVersion(0, 9, 9, 2):  "- Attempt #2 at fixing the database :U",
			AssembleVersion(0, 9, 9, 1):  "- Database restructuring and optimizations",
			AssembleVersion(0, 9, 9, 0):  "- Sweetie Bot now supports selfhosting and gives all patreon supporters access to paid features (chat logs and higher database limits). To get the new features, make sure you've linked your Patreon and Discord accounts. Check the GitHub readme for more instructions.\n- Help now hides disabled or restricted commands from users.\n- The bot name is no longer hardcoded: it will use whatever nickname it has on the server or the bot name given to the selfhost instance.\n- Removed echoembed command\n- Made listguilds and dumptables restricted commands\n- renamed AlertRole to ModRole and Search.MaxResults to Miscellaneous.MaxSearchResults, moved TrackUserLeft to Users and added NotifyChannel to track users instead of using !autosilence.\n- Spoiler module and Emote module have been replaced by a Filter module. If you were using these modules, your existing configuration was migrated to the new module. Anti-Spam was also renamed to Spam and Help/About renamed to Information.\n- Removed !addset/!removeset/!searchset. Use !addstatus/!removestatus/etc. or !addfilter/!removefilter/etc. instead.\n- Scheduler module no longer hides episode names outside of spoiler channels.\n- Server owners and admins can now use any command in any channel.\n- getconfig/setconfig now accept/display role names and channel names in addition to pings, and enforce valid types on all inputs. Using !getconfig on a category now displays all options in that category. All commands now accept simply writing out the role name, channel name, or user name.\n- !addbirthday was changed to be easier to use, and now adds the birthday using the user's timezone.\n- This is a total rewrite of sweetiebot, so if you find any bugs, or you think you should have paid features but you don't, please visit sweetiebot's support channel: https://discord.gg/t2gVQvN",
			AssembleVersion(0, 9, 8, 23): "- Remove !getpressure restriction.\n- Limit results of !searchtags",
			AssembleVersion(0, 9, 8, 22): "- Prevent race-condition crashing set management.\n- Force boolean configuration values to take only true or false.",
			AssembleVersion(0, 9, 8, 21): "- Made !userinfo more persistent at trying to find a match.",
			AssembleVersion(0, 9, 8, 20): "- Changed !searchtag to !searchtags because it's more consistent. Feel free to alias it back.",
			AssembleVersion(0, 9, 8, 19): "- Change how !remove works, use !remove * <item> to remove something with spaces from all tags.\n- !pick now requires tags to be one argument, but supports * to pick from all tags.\n-!searchtag can now take * in the tag argument to search all tags",
			AssembleVersion(0, 9, 8, 18): "- Fix specific tag search allowing tags from other servers to leak",
			AssembleVersion(0, 9, 8, 17): "- !tags now truncates output to 50 items unless user is a moderator\n- More information added to !add and !tags\n- Fixed bug with !remove\n- Allow more lines to be returned before switching to private message",
			AssembleVersion(0, 9, 8, 16): "- All servers now have audit logs.\n- Collections are now tags in the database, supporting complex tag searching. Use !tags and !searchtags to explore tags. Built-in collections are now managed through !addset, !removeset, and !searchset.\n- Ignore LockWaitTimeout errors",
			AssembleVersion(0, 9, 8, 15): "- Return all possible !wipe errors",
			AssembleVersion(0, 9, 8, 14): "- Reduce database pressure on startup",
			AssembleVersion(0, 9, 8, 13): "- Fix crash on startup.\n- Did more code refactoring, fixed several spelling errors.",
			AssembleVersion(0, 9, 8, 12): "- Do bulk member insertions in single batch to reduce database pressure.\n- Removed bestpony command\n- Did large internal code refactor",
			AssembleVersion(0, 9, 8, 11): "- User left now lists username+discriminator instead of pinging them to avoid @invalid-user problems.\n- Add ToS to !about\n- Bot now detects when it's about to be rate limited and combines short messages into a single large message. Helps keep bot responsive during huge raids.\n- Fixed race condition in spam module.",
			AssembleVersion(0, 9, 8, 10): "- !setup can now be run by any user with the administrator role.\n- Sweetie splits up embed messages if they have more than 25 fields.\n- Added !getraid and !banraid commands\n- Replaced !wipewelcome with generic !wipe command\n- Added LinePressure, which adds pressure for each newline in a message\n- Added TrackUserLeft, which will send a message when a user leaves in addition to when they join.",
			AssembleVersion(0, 9, 8, 9):  "- Moved several options to outside files to make self-hosting simpler to set up",
			AssembleVersion(0, 9, 8, 8):  "- !roll returns errors now.\n- You can now change the command prefix to a different ascii character - no, you can't set it to an emoji. Don't try.",
			AssembleVersion(0, 9, 8, 7):  "- Account creation time included on join message.\n- Specifying the config category is now optional. For example, !setconfig rules 3 \"blah\" works.",
			AssembleVersion(0, 9, 8, 6):  "- Support a lot more time formats and make time format more obvious.",
			AssembleVersion(0, 9, 8, 5):  "- Augment discordgo with maps instead of slices, and switch to using standard discordgo functions.",
			AssembleVersion(0, 9, 8, 4):  "- Update discordgo.",
			AssembleVersion(0, 9, 8, 3):  "- Allow deadlock detector to respond to deadlocks in the underlying discordgo library.\n- Fixed guild user count.",
			AssembleVersion(0, 9, 8, 2):  "- Simplify sweetiebot setup\n- Setting autosilence now resets the lockdown timer\n- Sweetiebot won't restore the verification level if it was manually changed by an administrator.",
			AssembleVersion(0, 9, 8, 1):  "- Switch to fork of discordgo to fix serious connection error handling issues.",
			AssembleVersion(0, 9, 8, 0):  "- Attempts to register if she is removed from a server.\n- Silencing has been redone to minimize rate-limiting problems.\n- Sweetie now tracks the first time someone posts a message, used in the \"bannewcomers\" command, which bans everyone who sent their first message in the past two minutes (configurable).\n- Sweetie now attempts to engage a lockdown when a raid is detected by temporarily increasing the server verification level. YOU MUST GIVE THE BOT \"MANAGE SERVER\" PERMISSIONS FOR THIS TO WORK! This can be disabled by setting Spam.LockdownDuration to 0.",
			AssembleVersion(0, 9, 7, 9):  "- Discard Group DM errors from legacy conversations.",
			AssembleVersion(0, 9, 7, 8):  "- Correctly deal with rare edge-case on !userinfo queries.",
			AssembleVersion(0, 9, 7, 7):  "- Sweetiebot sends an autosilence change message before she starts silencing raiders, to ensure admins get immediate feedback even if discord is being slow.",
			AssembleVersion(0, 9, 7, 6):  "- Sweetiebot now ignores other bots by default. To revert this, run '!setconfig basic.listentobots true' and she will listen to them again, but will never attempt to silence them.\n- Removed legacy timezones\n- Spam messages are limited to 300 characters in the log.",
			AssembleVersion(0, 9, 7, 5):  "- Compensate for discordgo being braindead and forgetting JoinedAt dates.",
			AssembleVersion(0, 9, 7, 4):  "- Update discordgo API.",
			AssembleVersion(0, 9, 7, 3):  "- Fix permissions issue.",
			AssembleVersion(0, 9, 7, 2):  "- Fix ignoring admins in anti-spam.",
			AssembleVersion(0, 9, 7, 1):  "- Fixed an issue with out-of-date guild objects not including all server members.",
			AssembleVersion(0, 9, 7, 0):  "- Groups have been removed and replaced with user-assignable roles. All your groups have automatically been migrated to roles. If there was a name-collision with an existing role, your group name will be prefixed with 'sb-', which you can then resolve yourself. Use '!help roles' to get usage information about the new commands.",
			AssembleVersion(0, 9, 6, 9):  "- Sweetiebot no longer logs her own actions in the audit log",
			AssembleVersion(0, 9, 6, 8):  "- Sweetiebot now has a deadlock detector and will auto-restart if she detects that she is not responding to !about\n- Appending @ to the end of a name or server is no longer necessary. If sweetie finds an exact match to your query, she will always use that.",
			AssembleVersion(0, 9, 6, 7):  "- Sweetiebot no longer attempts to track edited messages for spam detection. This also fixes a timestamp bug with pinned messages.",
			AssembleVersion(0, 9, 6, 6):  "- Sweetiebot now automatically sets Silence permissions on newly created channels. If you have a channel that silenced members should be allowed to speak in, make sure you've set it as the welcome channel via !setconfig users.welcomechannel #yourchannel",
			AssembleVersion(0, 9, 6, 5):  "- Fix spam detection error for edited messages.",
			AssembleVersion(0, 9, 6, 4):  "- Enforce max DB connections to try to mitigate connection problems",
			AssembleVersion(0, 9, 6, 3):  "- Extreme spam could flood SB with user updates, crashing the database. She now throttles user updates to help prevent this.\n- Anti-spam now uses discord's message timestamp, which should prevent false positives from network problems\n- Sweetie will no longer silence mods for spamming under any circumstance.",
			AssembleVersion(0, 9, 6, 2):  "- Renamed !quickconfig to !setup, added a friendly PM to new servers to make initial setup easier.",
			AssembleVersion(0, 9, 6, 1):  "- Fix !bestpony crash",
			AssembleVersion(0, 9, 6, 0):  "- Sweetiebot is now self-repairing and can function without a database, although her functionality is EXTREMELY limited in this state.",
			AssembleVersion(0, 9, 5, 9):  "- MaxRemoveLookback no longer relies on the database and can now be used in any server. However, it only deletes messages from the channel that was spammed in.",
			AssembleVersion(0, 9, 5, 8):  "- You can now specify per-channel pressure overrides via '!setconfig spam.maxchannelpressure <channel> <pressure>'.",
			AssembleVersion(0, 9, 5, 7):  "- You can now do '!pick collection1+collection2' to pick a random item from multiple collections.\n- !fight <monster> is now sanitized.\n- !silence now tells you when someone already silenced will be unsilenced, if ever.",
			AssembleVersion(0, 9, 5, 6):  "- Prevent idiots from setting status.cooldown to 0 and breaking everything.",
			AssembleVersion(0, 9, 5, 5):  "- Fix crash on invalid command limits.",
			AssembleVersion(0, 9, 5, 4):  "- Added ignorerole for excluding certain users from spam detection.\n- Adjusted unsilence to force bot to assume user is unsilenced so it can be used to fix race conditions.",
			AssembleVersion(0, 9, 5, 3):  "- Prevent users from aliasing existing commands.",
			AssembleVersion(0, 9, 5, 2):  "- Show user account creation date in userinfo\n- Added !SnowflakeTime command",
			AssembleVersion(0, 9, 5, 1):  "- Allow !setconfig to edit float values",
			AssembleVersion(0, 9, 5, 0):  "- Completely overhauled Anti-Spam module. Sweetie now analyzes message content and tracks text pressure users exert on the chat. See !help anti-spam for details, or !getconfig spam for descriptions of the new configuration options. Your old MaxImages and MaxPings settings were migrated over to ImagePressure and PingPressure, respectively.",
			AssembleVersion(0, 9, 4, 5):  "- Escape nicknames correctly\n- Sweetiebot no longer tracks per-server nickname changes, only username changes.\n- You can now use the format username#1234 in user arguments.",
			AssembleVersion(0, 9, 4, 4):  "- Fix locks, update endpoint calls, improve antispam response.",
			AssembleVersion(0, 9, 4, 3):  "- Emergency revert of last changes",
			AssembleVersion(0, 9, 4, 2):  "- Spammer killing is now asynchronous and should have fewer duplicate alerts.",
			AssembleVersion(0, 9, 4, 1):  "- Attempt to make sweetiebot more threadsafe.",
			AssembleVersion(0, 9, 4, 0):  "- Reduced number of goroutines, made updating faster.",
			AssembleVersion(0, 9, 3, 9):  "- Added !getaudit command for server admins.\n- Updated documentation for consistency.",
			AssembleVersion(0, 9, 3, 8):  "- Removed arbitrary limit on spam message detection, replaced with sanity limit of 600.\n- Sweetiebot now automatically detects invalid spam.maxmessage settings and removes them instead of breaking your server.\n- Replaced a GuildMember call with an initial state check to eliminate lag and some race conditions.",
			AssembleVersion(0, 9, 3, 7):  "- If a collection only has one item, just display the item.\n- If you put \"!\" into CommandRoles[<command>], it will now allow any role EXCEPT the roles specified to use <command>. This behaves the same as the channel blacklist function.",
			AssembleVersion(0, 9, 3, 6):  "- Add log option to autosilence.\n- Ensure you actually belong to the server you set as your default.",
			AssembleVersion(0, 9, 3, 5):  "- Improve help messages.",
			AssembleVersion(0, 9, 3, 4):  "- Prevent cross-server message sending exploit, without destroying all private messages this time.",
			AssembleVersion(0, 9, 3, 3):  "- Emergency revert change.",
			AssembleVersion(0, 9, 3, 2):  "- Prevent cross-server message sending exploit.",
			AssembleVersion(0, 9, 3, 1):  "- Allow sweetiebot to be executed as a user bot.",
			AssembleVersion(0, 9, 3, 0):  "- Make argument parsing more consistent\n- All commands that accepted a trailing argument without quotes no longer strip quotes out. The quotes will now be included in the query, so don't put them in if you don't want them!\n- You can now escape '\"' inside an argument via '\\\"', which will work even if discord does not show the \\ character.",
			AssembleVersion(0, 9, 2, 3):  "- Fix echoembed crash when putting in invalid parameters.",
			AssembleVersion(0, 9, 2, 2):  "- Update help text.",
			AssembleVersion(0, 9, 2, 1):  "- Add !joingroup warning to deal with breathtaking stupidity of zootopia users.",
			AssembleVersion(0, 9, 2, 0):  "- Remove !lastping\n- Help now lists modules with no commands",
			AssembleVersion(0, 9, 1, 1):  "- Fix crash in !getconfig",
			AssembleVersion(0, 9, 1, 0):  "- Renamed config options\n- Made things more clear for new users\n- Fixed legacy importable problem\n- Fixed command saturation\n- Added botchannel notification\n- Changed getconfig behavior for maps",
			AssembleVersion(0, 9, 0, 4):  "- To protect privacy, !listguilds no longer lists servers that do not have Basic.Importable set to true.\n- Remove some more unnecessary sanitization",
			AssembleVersion(0, 9, 0, 3):  "- Don't sanitize links already in code blocks",
			AssembleVersion(0, 9, 0, 2):  "- Alphabetize collections because Tawmy is OCD",
			AssembleVersion(0, 9, 0, 1):  "- Update documentation\n- Simplify !collections output",
			AssembleVersion(0, 9, 0, 0):  "- Completely restructured Sweetie Bot into a module-based architecture\n- Disabling/Enabling a module now disables/enables all its commands\n- Help now includes information about modules\n- Collections command is now pretty",
			AssembleVersion(0, 8, 17, 2): "- Added ability to hide negative rules because Tawmy is weird",
			AssembleVersion(0, 8, 17, 1): "- Added echoembed command",
			AssembleVersion(0, 8, 17, 0): "- Sweetiebot can now send embeds\n- Made about message pretty",
			AssembleVersion(0, 8, 16, 3): "- Update discordgo structs to account for breaking API change.",
			AssembleVersion(0, 8, 16, 2): "- Enable sweetiebot to tell dumbasses that they are dumbasses.",
			AssembleVersion(0, 8, 16, 1): "- !add can now add to multiple collections at the same time.",
			AssembleVersion(0, 8, 16, 0): "- Alphabetized the command list",
			AssembleVersion(0, 8, 15, 4): "- ReplaceMentions now breaks role pings (but does not resolve them)",
			AssembleVersion(0, 8, 15, 3): "- Use database to resolve users to improve responsiveness",
			AssembleVersion(0, 8, 15, 2): "- Improved !vote error messages",
			AssembleVersion(0, 8, 15, 1): "- Quickconfig actually sets silentrole now",
			AssembleVersion(0, 8, 15, 0): "- Use 64-bit integer conversion",
			AssembleVersion(0, 8, 14, 6): "- Allow adding birthdays on current day\n-Update avatar change function",
			AssembleVersion(0, 8, 14, 5): "- Allow exact string matching on !import",
			AssembleVersion(0, 8, 14, 4): "- Added !import\n- Added Importable option\n- Make !collections more useful",
			AssembleVersion(0, 8, 14, 3): "- Allow pinging multiple groups via group1+group2",
			AssembleVersion(0, 8, 14, 2): "- Fix !createpoll unique option key\n- Add !addoption",
			AssembleVersion(0, 8, 14, 1): "- Clean up !poll",
			AssembleVersion(0, 8, 14, 0): "- Added !poll, !vote, !createpoll, !deletepoll and !results commands",
			AssembleVersion(0, 8, 13, 1): "- Fixed !setconfig rules",
			AssembleVersion(0, 8, 13, 0): "- Added changelog\n- Added !rules command",
			AssembleVersion(0, 8, 12, 0): "- Added temporary silences",
			AssembleVersion(0, 8, 11, 5): "- Added \"dumbass\" to Sweetie Bot's vocabulary",
			AssembleVersion(0, 8, 11, 4): "- Display channels in help for commands",
			AssembleVersion(0, 8, 11, 3): "- Make defaultserver an independent command",
			AssembleVersion(0, 8, 11, 2): "- Add !defaultserver command",
			AssembleVersion(0, 8, 11, 1): "- Fix !autosilence behavior",
			AssembleVersion(0, 8, 11, 0): "- Replace mentions in !search\n- Add temporary ban to !ban command",
			AssembleVersion(0, 8, 10, 0): "- !ping now accepts newlines\n- Added build version to make moonwolf happy",
			AssembleVersion(0, 8, 9, 0):  "- Add silence message for Tawmy\n- Make silence message ping user\n- Fix #27 (Sweetie Bot explodes if you search nothing)\n- Make !lastseen more reliable",
			AssembleVersion(0, 8, 8, 0):  "- Log all commands sent to SB in DB-enabled servers",
			AssembleVersion(0, 8, 7, 0):  "- Default to main server for PMs if it exists\n- Restrict PM commands to the server you belong in (fix #26)\n- Make spam deletion lookback configurable\n- Make !quickconfig complain if permissions are wrong\n- Add giant warning label for Tawmy\n- Prevent parse time crash\n- Make readme more clear on how things work\n- Sort !listguild by user count\n- Fallback to search all users if SB can't find one in the current server",
			AssembleVersion(0, 8, 6, 0):  "- Add full timezone support\n- Deal with discord's broken permissions\n- Improve timezone help messages",
			AssembleVersion(0, 8, 5, 0):  "- Add !userinfo\n- Fix #15 (Lock down !removeevent)\n- Fix guildmember query\n- Use nicknames in more places",
			AssembleVersion(0, 8, 4, 0):  "- Update readme, remove disablebored\n- Add delete command",
			AssembleVersion(0, 8, 3, 0):  "- Actually seed random number generator because Cloud is a FUCKING IDIOT\n- Allow newlines in commands\n- Bored module is now fully programmable\n- Display user ID in !aka\n- Hopefully stop sweetie from being an emo teenager\n- Add additional stupid proofing\n- Have bored commands override all restrictions",
			AssembleVersion(0, 8, 2, 0):  "- Enable multi-server message logging\n- Extend !searchquote\n- Attach !lastping to current server\n- Actually make aliases work with commands",
			AssembleVersion(0, 8, 1, 0):  "- Add dynamic collections\n- Add quotes\n- Prevent !aka command from spawning evil twins\n- Add !removealias\n- Use nicknames where possible\n- Fix off by one error\n- Sanitize !search output ",
			AssembleVersion(0, 8, 0, 0):  "- Appease the dark gods of discord's API\n- Allow sweetiebot to track nicknames\n- update help\n- Include nickname in searches",
		},
	}

	json.Unmarshal(hostfile, sb)
	sb.Token = strings.TrimSpace(sb.Token)
	sb.EmptyGuild = NewGuildInfo(sb, &discordgo.Guild{})

	sb.EmptyGuild.Config.FillConfig()
	sb.EmptyGuild.Config.SetupDone = true
	sb.EmptyGuild.Modules = sb.loader(sb.EmptyGuild)
	sort.Sort(moduleArray(sb.EmptyGuild.Modules))

	for _, v := range sb.EmptyGuild.Modules {
		sb.EmptyGuild.RegisterModule(v)
		for _, command := range v.Commands() {
			if command.Info().ServerIndependent {
				sb.EmptyGuild.AddCommand(command, v)
			}
		}
	}

	// Load language override
	if configHelpFile, err := ioutil.ReadFile("confighelp.json"); err == nil {
		if err = json.Unmarshal(configHelpFile, &ConfigHelp); err != nil {
			fmt.Println("Error loading config help replacement file: ", err)
		}
	}

	if stringsFile, err := ioutil.ReadFile("strings.json"); err == nil {
		if err = json.Unmarshal(stringsFile, &StringMap); err != nil {
			fmt.Println("Error loading strings replacement file: ", err)
		}
	}

	db, err := dbLoad(&emptyLog{}, "mysql", strings.TrimSpace(sb.DBAuth))
	sb.DB = db
	if !db.Status.Get() {
		fmt.Println("Database connection failure - running in No Database mode: ", err.Error())
	} else {
		err = sb.DB.LoadStatements()
		if err == nil {
			fmt.Println("Finished loading database statements")
		} else {
			fmt.Println("Loading database statements failed: ", err)
			fmt.Println("DATABASE IS BADLY FORMATTED OR CORRUPT - TERMINATING SWEETIE BOT!")
			return nil
		}
	}

	var dg *discordgo.Session
	if sb.IsUserMode {
		dg, err = discordgo.New(sb.Token)
		fmt.Println("Started SweetieBot on a user account.")
	} else {
		dg, err = discordgo.New("Bot " + sb.Token)
	}
	sb.DG = &DiscordGoSession{*dg}

	if err != nil {
		fmt.Println("Error creating discord session", err.Error())
		return nil
	}
	sb.DG.LogLevel = discordgo.LogWarning

	sb.DG.AddHandler(sb.OnReady)
	sb.DG.AddHandler(sb.MessageCreate)
	sb.DG.AddHandler(sb.MessageUpdate)
	sb.DG.AddHandler(sb.MessageDelete)
	sb.DG.AddHandler(sb.UserUpdate)
	sb.DG.AddHandler(sb.GuildUpdate)
	sb.DG.AddHandler(sb.GuildMemberAdd)
	sb.DG.AddHandler(sb.GuildMemberRemove)
	sb.DG.AddHandler(sb.GuildMemberUpdate)
	sb.DG.AddHandler(sb.GuildBanAdd)
	sb.DG.AddHandler(sb.GuildBanRemove)
	sb.DG.AddHandler(sb.GuildRoleDelete)
	sb.DG.AddHandler(sb.GuildCreate)
	sb.DG.AddHandler(sb.ChannelCreate)
	return sb
}

// Connect opens a websocket connection to discord. Only returns after disconnecting.
func (sb *SweetieBot) Connect() int {
	if sb.Debug { // The server does not necessarily tie a standard input to the program
		go func() {
			var input string
			fmt.Scanln(&input)
			atomic.StoreUint32(&sb.quit, QuitNow)
		}()
	}

	go sb.deferProcessing()
	go sb.idleCheckLoop()
	go sb.deadlockDetector()
	go sb.memberIngestionLoop()
	go sb.ServeWeb()
	go sb.buildMarkov()

	err := sb.DG.Open()
	if err == nil {
		fmt.Println("Connection established")
		for atomic.LoadUint32(&sb.quit) == QuitNone {
			time.Sleep(800 * time.Millisecond)
		}
		begin := time.Now().UTC().Unix()
		for cur := time.Now().UTC().Unix(); (cur-begin) < MaxUpdateGrace && atomic.LoadUint32(&sb.quit) == QuitRaid; cur = time.Now().UTC().Unix() {
			quit := true
			for _, g := range sb.Guilds {
				if cur-g.LastRaid < UpdateGrace {
					quit = false
					break
				}
			}
			if quit {
				atomic.StoreUint32(&sb.quit, QuitNow)
			} else {
				time.Sleep(2400 * time.Millisecond)
			}
		}
	} else {
		fmt.Println("Error opening websocket connection: ", err.Error())
	}

	/*if q, err := sb.DB.db.Query("SELECT DISTINCT Guild FROM members"); err == nil {
		f, _ := os.Create("out.csv")
		defer f.Close()
		for q.Next() {
			var p uint64
			if err := q.Scan(&p); err == nil {
				if _, ok := sb.Guilds[NewDiscordGuild(p)]; !ok {
					fmt.Fprintf(f, "%v,", p)
				}
			}
		}
	}*/

	fmt.Println("Sweetiebot quitting")
	sb.DG.Close()
	sb.DB.Close()
	sb.GuildsLock.Lock() // Prevents a race condition from sending a value to a closed channel
	close(sb.memberChan)
	sb.GuildsLock.Unlock()
	return BotVersion.Integer()
}
