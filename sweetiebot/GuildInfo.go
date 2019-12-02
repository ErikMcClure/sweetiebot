package sweetiebot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"4d63.com/tz"
	"github.com/erikmcclure/discordgo"
)

// GuildInfo Stores state information about a guild
type GuildInfo struct {
	ID           string // Cache the ID because it doesn't change
	Name         string // Cache the name to reduce locking
	OwnerID      DiscordUser
	BotNick      string     // If not empty, the nickname assigned to the bot in this server
	Silver       AtomicBool // Has paid features (always true if selfhosting)
	lastlogerr   int64
	LastRaid     int64 // Last time a raid was recorded by the spam module (or any other module that records raids)
	commandLock  sync.RWMutex
	commandLast  map[string]int64
	commandlimit *SaturationLimit
	ConfigLock   sync.RWMutex
	Config       BotConfig
	hooks        moduleHooks
	Modules      []Module
	commands     map[CommandID]Command
	commandmap   map[CommandID]ModuleID // Exists entirely so the help command can match commands to their parent module
	Bot          *SweetieBot
}

var errOwnerExclusive = errors.New("Only the owner of the bot can run this command!")
var errMainGuild = errors.New("this command can only be run on the main server")
var errNoPermissions = errors.New("You don't have permission to run this command!")
var errIgnored = errors.New("a module is ignoring this command")
var errDisabled = errors.New("this command is disabled")
var errSilenced = errors.New("silenced users cannot use commands")
var errInvalidChannel = errors.New("Attempted to send message to channel on a different server.")
var errConfigFileTooLarge = errors.New("Error saving config file: Config file is too large!")

// NewGuildInfo spawns a new GuildInfo object with a default configuration
func NewGuildInfo(sb *SweetieBot, g *discordgo.Guild) *GuildInfo {
	return &GuildInfo{
		ID:           g.ID,
		Name:         g.Name,
		OwnerID:      DiscordUser(g.OwnerID),
		commandLast:  make(map[string]int64),
		commandlimit: &SaturationLimit{[]int64{}, 0, AtomicFlag{0}},
		commands:     make(map[CommandID]Command),
		commandmap:   make(map[CommandID]ModuleID),
		lastlogerr:   0,
		Bot:          sb,
		Config:       *DefaultConfig(),
	}
}

// AddCommand adds a command to the guild
func (info *GuildInfo) AddCommand(c Command, m Module) {
	name := CommandID(strings.ToLower(c.Info().Name))
	info.commands[name] = c
	info.commandmap[name] = ModuleID(strings.ToLower(m.Name()))
}

// SaveConfig saves the config file to disk
func (info *GuildInfo) SaveConfig() (err error) {
	data, err := json.Marshal(info.Config)
	if err == nil {
		if len(data) > info.Bot.MaxConfigSize {
			info.Log("Error saving config file: Config file is too large! Config files cannot exceed " + strconv.Itoa(info.Bot.MaxConfigSize) + " bytes.")
			err = errConfigFileTooLarge
		} else {
			if err = ioutil.WriteFile(info.ID+".json", data, 0664); err != nil {
				info.Log("Error saving config file: ", err.Error())
			}
		}
	} else {
		info.Log("Error writing json: ", err.Error())
	}
	return
}

// SendEmbed sends an embed message to the channel, splitting it into multiple messages if necessary
func (info *GuildInfo) SendEmbed(channelID DiscordChannel, embed *discordgo.MessageEmbed) error {
	if channelID == "heartbeat" {
		atomic.AddUint32(&info.Bot.heartbeat, 1)
		return nil
	}
	if ch, private := info.Bot.ChannelIsPrivate(channelID); !private && (ch == nil || ch.GuildID != info.ID) {
		return errInvalidChannel
	}

	if perms, err := info.Bot.DG.State.UserChannelPermissions(info.Bot.SelfID.String(), channelID.String()); err == nil && (perms&discordgo.PermissionEmbedLinks) == 0 {
		return info.SendMessage(channelID, "```\nFailed to send embedded message! Make sure the bot has EmbedLink permissions on this channel. Embed description: "+embed.Description+"```")
	}

	fields := embed.Fields
	for len(fields) > 25 {
		embed.Fields = fields[:25]
		fields = fields[25:]
		if _, err := info.Bot.DG.ChannelMessageSendEmbed(channelID.String(), embed); err != nil {
			return err
		}
	}
	embed.Fields = fields
	_, err := info.Bot.DG.ChannelMessageSendEmbed(channelID.String(), embed)
	return err
	//info.Bot.DG.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
	//	Content: "Test content",
	//	Embed:   embed,
	//})
}

// RequestPostWithBuffer uses a buffer and a buffer combination function to combine multiple messages if there are fewer than minRequests requests left in the current bucket
func (info *GuildInfo) RequestPostWithBuffer(urlStr string, data *discordgo.MessageSend, minRemaining int) (response []byte, err error) {
	b := info.Bot.DG.Ratelimiter.GetBucket(urlStr)
	b.Lock()
	if b.Userdata == nil {
		b.Userdata = &sbRequestBuffer{nil, 0}
	}
	buffer := b.Userdata.(*sbRequestBuffer)

	// data can be nil here, which tells the buffer to check if it's full
	remain := buffer.Append(data)
	softwait := info.Bot.DG.Ratelimiter.GetWaitTime(b, minRemaining)

	if remain == 0 && softwait > 0 {
		b.Release(nil)
		time.Sleep(softwait)
		b.Lock()
	}

	for {
		data, remain = buffer.Process()

		if data != nil {
			if wait := info.Bot.DG.Ratelimiter.GetWaitTime(b, 1); wait > 0 {
				//fmt.Printf("Hit rate limit in buffered request, sleeping for %v (%v remaining)\n", wait, remain)
				time.Sleep(wait)
			}

			b.Remaining--
			softwait = info.Bot.DG.Ratelimiter.GetWaitTime(b, minRemaining)
			var body []byte
			body, err = json.Marshal(data)
			if err == nil {
				response, err = info.Bot.DG.RequestWithLockedBucket("POST", urlStr, "application/json", body, b, 0)
			} else {
				b.Release(nil)
				break
			}
		} else {
			b.Release(nil)
			break
		}

		// If we have nothing left to do, bail out early to avoid extra work
		if remain == 0 {
			break
		}

		// If we ran out of breathing room on our bucket, sleep until the end of the soft limit
		if softwait > 0 {
			time.Sleep(softwait)
		}
		b.Lock() // Re-lock the bucket
	}

	return
}

func (info *GuildInfo) sendContent(channelID DiscordChannel, message string, minRequest int) {
	_, err := info.RequestPostWithBuffer(discordgo.EndpointChannelMessages(channelID.String()), &discordgo.MessageSend{
		Content: message,
	}, minRequest)
	if err != nil {
		fmt.Println("Failed to send message: ", err.Error())
	}
}

// SendMessage sends a message to the given channel, splitting it into multiple messages if necessary, and combining smaller messages if a rate limit is about to be hit
func (info *GuildInfo) SendMessage(channelID DiscordChannel, message string) error {
	if channelID == "heartbeat" {
		atomic.AddUint32(&info.Bot.heartbeat, 1)
		return nil
	}
	if ch, private := info.Bot.ChannelIsPrivate(channelID); !private && (ch == nil || ch.GuildID != info.ID || ch.Type == discordgo.ChannelTypeGuildVoice || ch.Type == discordgo.ChannelTypeGuildCategory) {
		return errInvalidChannel
	}

	for len(message) > 1999 { // discord has a 2000 character limit
		if message[0:3] == "```" && message[len(message)-3:] == "```" {
			index := strings.LastIndex(message[:1995], "\n")
			if index < 10 { // Ensure we process at least 10 characters to prevent an infinite loop
				index = 1995
			}
			info.sendContent(channelID, message[:index]+"```", 1)
			message = "```\n" + message[index:]
		} else {
			index := strings.LastIndex(message[:1999], "\n")
			if index < 10 {
				index = 1999
			}
			info.sendContent(channelID, message[:index], 1)
			message = message[index:]
		}
	}
	info.sendContent(channelID, message, 2)

	return nil
}

// ProcessModule returns true if a module should process events on this channel
func (info *GuildInfo) ProcessModule(channelID DiscordChannel, m Module) bool {
	id := ModuleID(strings.ToLower(m.Name()))
	if _, disabled := info.Config.Modules.Disabled[id]; disabled {
		return false
	}
	if channelID == ChannelEmpty {
		return true
	}

	collection := info.Config.Modules.Channels[id]
	item, err := ParseChannel(channelID.String(), nil)
	if err == nil && len(collection) > 0 {
		_, reverse := collection[ChannelExclusion]
		_, ok := collection[item]
		return ok != reverse
	}
	return err == nil
}

// IsDebug returns true if the channel is a debug channel
func (info *GuildInfo) IsDebug(channelID DiscordChannel) bool {
	debugchannel, isdebug := info.Bot.DebugChannels[DiscordGuild(info.ID)]
	if isdebug {
		return channelID == debugchannel
	}
	return false
}

// ProcessMember called ProcessUser and adds additional member information to the database
func (info *GuildInfo) ProcessMember(u *discordgo.Member) {
	info.Bot.ProcessUser(u.User)

	if info.Bot.DB.CheckStatus() {
		info.Bot.DB.AddMember(SBatoi(u.User.ID), SBatoi(info.ID), GetJoinedAt(u), u.Nick)
	}
}

func (info *GuildInfo) userBulkUpdate(members []*discordgo.Member) {
	valueArgs := make([]interface{}, 0, len(members)*6)
	valueStrings := make([]string, 0, len(members))

	for _, m := range members {
		valueStrings = append(valueStrings, "(?,?,?,UTC_TIMESTAMP(), UTC_TIMESTAMP())")
		discriminator, _ := strconv.Atoi(m.User.Discriminator)
		valueArgs = append(valueArgs, SBatoi(m.User.ID), m.User.Username, discriminator)
	}

	stmt := fmt.Sprintf("INSERT IGNORE INTO users (ID, Username, Discriminator, LastSeen, LastNameChange) VALUES %s", strings.Join(valueStrings, ","))
	_, err := info.Bot.DB.db.Exec(stmt, valueArgs...)
	info.LogError("Error in UserBulkUpdate", err)
}

func (info *GuildInfo) memberBulkUpdate(members []*discordgo.Member) {
	valueArgs := make([]interface{}, 0, len(members)*4)
	valueStrings := make([]string, 0, len(members))

	for _, m := range members {
		valueStrings = append(valueStrings, "(?,?,?,?)")
		valueArgs = append(valueArgs, SBatoi(m.User.ID), SBatoi(info.ID), GetJoinedAt(m), m.Nick)
	}
	stmt := fmt.Sprintf("INSERT IGNORE INTO members (ID, Guild, FirstSeen, Nickname) VALUES %s", strings.Join(valueStrings, ","))
	_, err := info.Bot.DB.db.Exec(stmt, valueArgs...)
	info.LogError("Error in MemberBulkUpdate", err)
}

// ProcessGuild updates guild information and adds the initial member list to the database
func (info *GuildInfo) ProcessGuild(g *discordgo.Guild) {
	info.Name = g.Name
	info.OwnerID = DiscordUser(g.OwnerID)
	const chunksize int = 1000

	if len(g.Members) > 0 && info.Bot.DB.CheckStatus() {
		// First process userdata
		i := chunksize
		for i < len(g.Members) {
			info.userBulkUpdate(g.Members[i-chunksize : i])
			i += chunksize
		}
		info.userBulkUpdate(g.Members[i-chunksize:])

		// Then process member data
		i = chunksize
		for i < len(g.Members) {
			info.memberBulkUpdate(g.Members[i-chunksize : i])
			i += chunksize
		}
		info.memberBulkUpdate(g.Members[i-chunksize:])
	}
}

// FindChannelID returns the ID of the first channel in this guild with a matching name
func (info *GuildInfo) FindChannelID(name string) string {
	guild, err := info.GetGuild()
	if err != nil {
		return ""
	}
	info.Bot.DG.State.RLock()
	defer info.Bot.DG.State.RUnlock()
	for _, v := range guild.Channels {
		if v.Name == name {
			return v.ID
		}
	}

	return ""
}

// Log the given arguments to the server and the command line
func (info *GuildInfo) Log(args ...interface{}) {
	s := fmt.Sprint(args...)
	fmt.Printf("[%s] %s\n", time.Now().Format(time.Stamp), s)
	if info != nil && info.Config.Log.Channel != ChannelEmpty {
		info.SendMessage(info.Config.Log.Channel, "```\n"+s+"```")
	}
}

// LogError logs an error only if it exists
func (info *GuildInfo) LogError(msg string, err error) {
	if err != nil {
		info.Log(msg, err.Error())
	}
}

// SendError prints an error message with a saturation limit
func (info *GuildInfo) SendError(channelID DiscordChannel, message string, t int64) {
	if info != nil && RateLimit(&info.lastlogerr, info.Config.Log.Cooldown, t) { // Don't print more than one error message every n seconds.
		info.SendMessage(channelID, "```\n"+message+"```")
	}
}

// UserHasRole returns true if the specified user ID has the given role ID (both in strings)
func (info *GuildInfo) UserHasRole(userID DiscordUser, role DiscordRole) bool {
	if m, err := info.Bot.DG.GetMember(userID, info.ID); err == nil {
		return MemberHasRole(m, role)
	}
	return false
}

// UserCanUseCommand returns nil if the user can use the command, or an error explaining why they can't.
// The boolean marks whether or not they bypass restrictions. Note that even moderators cannot use exclusive commands, only the owner of the bot can.
func (info *GuildInfo) UserCanUseCommand(userID DiscordUser, command Command, ignore bool) (bypass bool, err error) {
	if info.Bot.Owner == userID {
		bypass = true
		return
	}
	isAdmin := info.UserIsAdmin(userID)
	isMod := info.UserIsMod(userID)
	isSelf := userID == info.Bot.SelfID
	bypass = isAdmin || isSelf
	dat := command.Info()
	name := CommandID(strings.ToLower(dat.Name))
	if dat.Restricted {
		err = errOwnerExclusive
		return
	}
	if dat.MainInstance && !info.Bot.MainGuildID.Equals(info.ID) {
		err = errMainGuild
		return
	}
	if isAdmin { // Admins can run disabled commands
		return
	}
	if _, disabled := info.Config.Modules.CommandDisabled[name]; disabled {
		err = errDisabled
		return
	}
	if isSelf { // The bot can always run any command that isn't disabled or exclusive
		return
	}
	if info.Config.Basic.SilenceRole != RoleEmpty && info.UserHasRole(userID, info.Config.Basic.SilenceRole) {
		err = errSilenced // Silenced users can never run commands
		return
	}
	if info.Config.Basic.MemberRole != RoleEmpty && !info.UserHasRole(userID, info.Config.Basic.MemberRole) {
		err = errSilenced // Non-member users can never run commands
		return
	}
	if ignore && !isMod {
		err = errIgnored
		return
	}
	if info.Bot.DG.UserHasAnyRole(userID, info.ID, info.Config.Modules.CommandRoles[name]) {
		return
	}
	err = errors.New("You don't have permission to run this command! Allowed Roles: " + info.GetRoles(name))
	return
}

// UserIsAdmin returns true if the user is an admin or the owner of the bot. Always prefers returning false if any kind of error happens.
func (info *GuildInfo) UserIsAdmin(userID DiscordUser) bool {
	if userID == info.Bot.Owner {
		return true
	}
	perms, err := info.Bot.DG.UserPermissions(userID, info.ID) // First get permissions from the cache. If this errors out, default to not an admin
	return err == nil && ((perms & discordgo.PermissionAdministrator) != 0)
}

// UserIsMod returns true if the user is a mod
func (info *GuildInfo) UserIsMod(userID DiscordUser) bool {
	return info.Config.Basic.ModRole != RoleEmpty && info.UserHasRole(userID, info.Config.Basic.ModRole)
}

// GetRoles constructs a string describing the allowed roles for a command
func (info *GuildInfo) GetRoles(command CommandID) string {
	m, ok := info.Config.Modules.CommandRoles[command]
	if !ok {
		return ""
	}

	_, reverse := m["!"]
	s := make([]string, 0, len(m))
	for k := range m {
		if k != RoleExclusion {
			s = append(s, k.Show(info))
		}
	}

	sort.Strings(s)

	if reverse {
		return "Any role except " + strings.Join(s, ", ")
	}
	return strings.Join(s, ", ")
}

// GetChannels constructs a string describing the allowed channels a command can run on
func (info *GuildInfo) GetChannels(command CommandID) string {
	m, ok := info.Config.Modules.CommandChannels[command]
	if !ok {
		return ""
	}

	_, reverse := m["!"]
	s := make([]string, 0, len(m))
	for k := range m {
		c, err := info.Bot.DG.State.Channel(k.String())
		if err == nil {
			s = append(s, "#"+c.Name)
		}
	}

	sort.Strings(s)

	if reverse {
		return "Any channel except " + strings.Join(s, ", ")
	}
	return strings.Join(s, ", ")
}

// FormatUsage constructs a help string for the given command based on it's usage
func (info *GuildInfo) FormatUsage(c Command, usage *CommandUsage) *discordgo.MessageEmbed {
	name := CommandID(strings.ToLower(c.Info().Name))
	r := info.GetRoles(name)
	ch := info.GetChannels(name)
	fields := make([]*discordgo.MessageEmbedField, 0, len(usage.Params))
	use := "> " + info.Config.Basic.CommandPrefix + string(name)
	for _, v := range usage.Params {
		opt := ""
		if v.Optional {
			opt = " [OPTIONAL]"
			use += fmt.Sprintf(" [%s]", v.Name)
		} else {
			use += fmt.Sprintf(" {%s}", v.Name)
		}
		if v.Variadic {
			opt = " (...) " + opt
			use += "..."
		}
		fields = append(fields, &discordgo.MessageEmbedField{Name: "**" + v.Name + "**" + opt, Value: v.Desc, Inline: false})
	}

	if len(ch) > 0 {
		ch = fmt.Sprintf("Available on: %s", ch)
	}
	module, _ := info.commandmap[name]
	embed := &discordgo.MessageEmbed{
		Type: "rich",
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://sweetiebot.io/help/" + string(module) + "/#" + string(name),
			Name:    c.Info().Name + " Command",
			IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", info.Bot.SelfID, info.Bot.SelfAvatar),
		},
		Color:       0xaaaaaa,
		Description: fmt.Sprintf("```\n%s```\n%s\n\n%s", use, usage.Desc, ch),
		Fields:      fields,
	}

	if len(r) > 0 {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "Only usable by: " + r}
	}
	return embed
}

// ParseCommonTime iterates through every single imaginable time format that we could possibly parse
func (info *GuildInfo) ParseCommonTime(s string, user DiscordUser, timestamp time.Time) (time.Time, error) {
	var t time.Time
	var err error
	tz := info.GetTimezone(user)

	for year := 0; year < FormatYearNum; year++ {
		for hours := 0; hours < FormatTimeNum; hours++ {
			for timezone := 0; timezone < FormatZoneNum; timezone++ {
				for monthfirst := 0; monthfirst < 2; monthfirst++ {
					for fullmonth := 0; fullmonth < 2; fullmonth++ {
						format := getTimeFormat(monthfirst != 0, fullmonth != 0, year, hours, timezone)

						if timezone == FormatNoZone {
							t, err = time.ParseInLocation(format, s, tz)
						} else {
							t, err = time.ParseInLocation(format, s, time.UTC)
						}
						if err == nil {
							if year == FormatNoYear {
								t = t.AddDate(info.ApplyTimezone(timestamp, user).Year(), 0, 0)
							}
							return t, err
						}
					}
				}
			}
		}
	}
	return t, err
}

func (info *GuildInfo) setupSilenceRole() {
	guild, err := info.GetGuild()
	if err != nil {
		info.Log("Failed to setup silence roles!")
		return
	}
	for _, ch := range guild.Channels {
		// If there is a silence role, override it on all channels
		if info.Config.Basic.SilenceRole != RoleEmpty {
			allow := 0
			deny := 0
			for _, v := range ch.PermissionOverwrites {
				if strings.ToLower(v.Type) == "role" && info.Config.Basic.SilenceRole.Equals(v.ID) {
					allow = v.Allow
					deny = v.Deny
					break
				}
			}
			if !info.Config.Users.JailChannel.Equals(ch.ID) {
				allow &= (^discordgo.PermissionSendMessages)
				deny |= discordgo.PermissionSendMessages | discordgo.PermissionAddReactions
			} else {
				deny &= (^(discordgo.PermissionSendMessages | discordgo.PermissionReadMessages))
				allow |= discordgo.PermissionSendMessages | discordgo.PermissionReadMessages
			}
			info.ChannelPermissionSet(ch, info.Config.Basic.SilenceRole.String(), "role", allow, deny)
		}

		// If this is the welcome channel and we have a member role, ensure the everyone role can speak in it by overriding the channel permissions.
		if info.Config.Basic.MemberRole != RoleEmpty && info.Config.Users.WelcomeChannel.Equals(ch.ID) {
			allow := 0
			deny := 0
			for _, v := range ch.PermissionOverwrites {
				if strings.ToLower(v.Type) == "role" && info.ID == v.ID {
					allow = v.Allow
					deny = v.Deny
					break
				}
			}
			deny &= (^(discordgo.PermissionSendMessages | discordgo.PermissionReadMessages))
			allow |= discordgo.PermissionSendMessages | discordgo.PermissionReadMessages
			info.ChannelPermissionSet(ch, info.ID, "role", allow, deny)
		}
	}
}

// GetTimezone gets the time.Location of the given user, if it exists, otherwise returns time.UTC
func (info *GuildInfo) GetTimezone(user DiscordUser) *time.Location {
	if user != UserEmpty && info.Bot.DB.Status.Get() {
		loc := info.Bot.DB.GetTimeZone(user.Convert())
		if loc != nil {
			return loc
		}
	}
	if loc, err := tz.LoadLocation(string(info.Config.Users.TimezoneLocation)); err == nil {
		return loc
	}
	return time.UTC
}

// ApplyTimezone transforms the given UTC time into local time for the given user
func (info *GuildInfo) ApplyTimezone(t time.Time, user DiscordUser) time.Time {
	return t.In(info.GetTimezone(user))
}

// FindUsername returns all possible matching IDs for the given username
func (info *GuildInfo) FindUsername(arg string) []uint64 {
	return info.findUsernameInternal(arg, 0)
}

func (info *GuildInfo) findUsernameInternal(arg string, recurse int) []uint64 {
	if len(arg) <= 0 {
		return []uint64{}
	}
	user := arg
	if UserRegex.MatchString(user) {
		return []uint64{SBatoi(user[2 : len(user)-1])}
	}
	if !info.Bot.DB.Status.Get() {
		return []uint64{}
	}
	if discriminantregex.MatchString(user) {
		pos := strings.LastIndex(user, "#")
		if pos >= 0 {
			discriminant, err := strconv.Atoi(user[pos+1:])
			user = strings.ToLower(user[:pos])
			if err == nil {
				return info.Bot.DB.FindUser(user, discriminant, 20, 0)
			}
		}
	}
	r := info.Bot.DB.FindGuildUsers(user, 20, 0, SBatoi(info.ID))
	if len(r) == 0 {
		user = "%" + user + "%"
		r = info.Bot.DB.FindGuildUsers(user, 20, 0, SBatoi(info.ID))
	}
	if len(r) == 0 && arg[0] == '@' && recurse < 1 {
		return info.findUsernameInternal(arg[1:], recurse+1)
	}
	return r
}

// GetUserName returns a string representation of the user's name if possible, otherwise pings them.
func (info *GuildInfo) GetUserName(user DiscordUser) string {
	u := user.String()
	m, _ := info.Bot.DG.State.Member(info.ID, u)
	if m == nil {
		return "<@" + u + ">"
	}
	return info.GetMemberName(m)
}

// GetMemberName gets either the nickname or username of a member
func (info *GuildInfo) GetMemberName(m *discordgo.Member) string {
	if len(m.Nick) > 0 {
		return m.Nick
	}
	return m.User.Username
}

// GetGuild returns the guild object associated with this info object
func (info *GuildInfo) GetGuild() (*discordgo.Guild, error) {
	return info.Bot.DG.State.Guild(info.ID)
}

// IDsToUsernames converts an array of integer IDs to an array of username strings
func (info *GuildInfo) IDsToUsernames(IDs []uint64, discriminator bool) []string {
	s := make([]string, 0, len(IDs))
	for _, v := range IDs {
		m, _ := info.Bot.DG.State.Member(info.ID, SBitoa(v))
		if m != nil {
			if len(m.Nick) > 0 {
				if discriminator {
					s = append(s, fmt.Sprintf("%s (%s#%s)", m.Nick, m.User.Username, m.User.Discriminator))
				} else {
					s = append(s, m.Nick)
				}
			} else {
				if discriminator {
					s = append(s, m.User.Username+"#"+m.User.Discriminator)
				} else {
					s = append(s, m.User.Username)
				}
			}
		} else {
			s = append(s, "<@"+SBitoa(v)+">")
		}
	}
	return s
}

func (info *GuildInfo) GetBotName() string {
	if len(info.BotNick) > 0 {
		return info.BotNick
	}
	return info.Bot.SelfName
}

const (
	CleanMentions  = 1 << iota
	CleanPings     = 1 << iota
	CleanURL       = 1 << iota
	CleanEmotes    = 1 << iota
	CleanCode      = 1 << iota
	CleanCodeBlock = CleanCode | CleanEmotes
	CleanAll       = CleanMentions | CleanURL | CleanEmotes | CleanCode | CleanPings
	CleanMost      = CleanMentions | CleanURL | CleanEmotes | CleanPings
)

// Sanitize the input string according to flags
func (info *GuildInfo) Sanitize(s string, flags int) string {
	if (flags & CleanURL) != 0 {
		s = urlregex.ReplaceAllStringFunc(s, func(str string) string { return "<" + str + ">" })
	}
	if (flags & CleanMentions) != 0 {
		s = info.Bot.DG.ReplaceAllMentions(s, info.Bot.DB, info.ID)
	}
	if (flags & CleanPings) != 0 {
		s = mentionRegex.ReplaceAllStringFunc(s, func(str string) string { return "<\\@" + str[2:] })
	}
	if (flags & CleanEmotes) != 0 {
		s = strings.Replace(s, "[](/", "[\u200B](/", -1)
	}
	if (flags & CleanCode) != 0 {
		s = strings.Replace(s, "```", "\\`\\`\\`", -1)
	}
	return s
}

// BulkDelete Performs a bulk deletion in groups of 100
func (info *GuildInfo) BulkDelete(channel *discordgo.Channel, messages []string) (err error) {
	if channel == nil || channel.GuildID != info.ID {
		return errInvalidChannel
	}
	return info.Bot.DG.BulkDeleteBypass(channel.ID, messages)
}

// ChannelMessageDelete checks the channel guildID before calling the real ChannelMessageDelete
func (info *GuildInfo) ChannelMessageDelete(channel *discordgo.Channel, messageID string) (err error) {
	if channel == nil || channel.GuildID != info.ID {
		return errInvalidChannel
	}
	return info.Bot.DG.ChannelMessageDelete(channel.ID, messageID)
}

// ChannelPermissionSet check the channel guildID before calling the real ChannelPermissionSet
func (info *GuildInfo) ChannelPermissionSet(channel *discordgo.Channel, targetID, targetType string, allow, deny int) (err error) {
	if channel == nil || channel.GuildID != info.ID {
		return errInvalidChannel
	}
	return info.Bot.DG.ChannelPermissionSet(channel.ID, targetID, targetType, allow, deny)
}

// Clean out all commands or modules that no longer exist
func (info *GuildInfo) Clean() {
	for k := range info.Config.Modules.Channels {
		for _, m := range info.Modules {
			if k == ModuleID(strings.ToLower(m.Name())) {
				k = ModuleID("")
				break
			}
		}
		delete(info.Config.Modules.Channels, k)
	}

	for k := range info.Config.Modules.Disabled {
		for _, m := range info.Modules {
			if k == ModuleID(strings.ToLower(m.Name())) {
				k = ModuleID("")
				break
			}
		}
		delete(info.Config.Modules.Disabled, k)
	}
	for k := range info.Config.Modules.CommandRoles {
		if _, ok := info.commands[k]; !ok {
			delete(info.Config.Modules.CommandRoles, k)
		}
	}
	for k := range info.Config.Modules.CommandChannels {
		if _, ok := info.commands[k]; !ok {
			delete(info.Config.Modules.CommandChannels, k)
		}
	}
	for k := range info.Config.Modules.CommandLimits {
		if _, ok := info.commands[k]; !ok {
			delete(info.Config.Modules.CommandLimits, k)
		}
	}
	for k := range info.Config.Modules.CommandDisabled {
		if _, ok := info.commands[k]; !ok {
			delete(info.Config.Modules.CommandDisabled, k)
		}
	}
}

func (info *GuildInfo) ResolveRoleAddError(err error) error {
	if err != nil {
		if perms, err := info.Bot.DG.UserPermissions(info.Bot.SelfID, info.ID); err == nil && (perms&discordgo.PermissionManageRoles) == 0 {
			return errors.New("I can't change roles because I don't have the Manage Roles permission!")
		}
		msg := "http 403 forbidden"
		if len(err.Error()) > len(msg) && strings.ToLower(err.Error()[:len(msg)]) == msg {
			return errors.New("I can't set a role that's above me in the role list! You have to ensure that the bot role is above all the roles you want it to manage in the list of roles.")
		}
	}
	return err
}

func (info *GuildInfo) checkOnCommand(m *discordgo.Message) (ignore bool) {
	for _, h := range info.hooks.OnCommand {
		if info.ProcessModule(DiscordChannel(m.ChannelID), h) {
			ignore = ignore || h.OnCommand(info, m)
		}
	}
	return
}
