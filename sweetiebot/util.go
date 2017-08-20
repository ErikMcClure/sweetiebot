package sweetiebot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blackhole12/discordgo"
)

func Pluralize(i int64, s string) string {
	if i == 1 {
		return strconv.FormatInt(i, 10) + s
	}
	return strconv.FormatInt(i, 10) + s + "s"
}

func TimeDiff(d time.Duration) string {
	seconds := int64(d.Seconds())
	if seconds <= 60 {
		return Pluralize(seconds, " second")
	}
	if seconds <= 60*60 {
		return Pluralize((seconds+1)/60, " minute")
	}
	days := (seconds + 100) / 86400
	hours := (seconds + 10 - (days * 86400)) / 3600
	minutes := (seconds - (days * 86400) - (hours * 3600)) / 60

	if days == 0 && minutes > 2 {
		return Pluralize(hours, " hour") + " and " + Pluralize(minutes, " minute")
	}
	if days == 0 {
		return Pluralize(hours, " hour")
	}
	if days >= 365 && (days%365) == 0 {
		return Pluralize(days/365, " year")
	}
	if days >= 365 {
		return Pluralize(days/365, " year") + " and " + Pluralize(days%365, " day")
	}
	if hours > 1 {
		return Pluralize(days, " day") + " and " + Pluralize(hours, " hour")
	}
	return Pluralize(days, " day")
}

func PingAtoi(s string) uint64 {
	if len(s) > 2 && (s[:2] == "<#" || s[:2] == "<@") {
		return SBatoi(s[2 : len(s)-1])
	}
	return SBatoi(s)
}
func StripPing(s string) string {
	if len(s) > 2 && (s[:2] == "<#" || s[:2] == "<@") {
		if len(s) >= 3 && (s[2:3] == "!" || s[2:3] == "&") {
			return s[3 : len(s)-1]
		}
		return s[2 : len(s)-1]
	}
	return s
}
func SBatoi(s string) uint64 {
	if len(s) < 1 {
		return 0
	}
	if s[:1] == "!" || s[:1] == "&" {
		s = s[1:]
	}
	i, err := strconv.ParseUint(strings.Replace(s, "\u200B", "", -1), 10, 64)
	if err != nil {
		fmt.Println("Invalid number ", s, ":", err.Error())
		return 0
	}
	return i
}
func SBitoa(i uint64) string {
	return strconv.FormatUint(i, 10)
}
func IsSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r'
}

func IDsToUsernames(IDs []uint64, info *GuildInfo, discriminator bool) []string {
	s := make([]string, 0, len(IDs))
	gid := SBatoi(info.ID)
	for _, v := range IDs {
		var m *discordgo.Member = nil
		if sb.db.status.get() {
			m, _, _ = sb.db.GetMember(v, gid)
		}
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
func ParseArguments(s string) ([]string, []int) {
	r := []string{}
	indices := []int{}
	l := len(s)
	for i := 0; i < l; i++ {
		c := s[i]
		if !IsSpace(c) {
			indices = append(indices, i+1) // This is i+1 because we send in m.Content[1:]
			var start int
			var end int
			if c == '"' && (i < 1 || s[i-1] != '\\') {
				i++
				start = i
				for i < l && (s[i] != '"' || s[i-1] == '\\') {
					i++
				}
				if s[i-1] == '\\' {
					end = i - 1
				} else {
					end = i
				}
			} else {
				start = i
				i++
				for i < l && !IsSpace(s[i]) && (s[i] != '"' || s[i-1] == '\\') {
					i++
				}
				end = i
			}
			r = append(r, s[start:end])
		}
	}
	return r, indices
}

// This constructs an XOR operator for booleans
func boolXOR(a bool, b bool) bool {
	return (a && !b) || (!a && b)
}

func (info *GuildInfo) UserHasRole(user string, role string) bool {
	m, err := info.GetMember(user)
	if err == nil {
		for _, v := range m.Roles {
			if v == role {
				return true
			}
		}
	}
	return false
}

func (info *GuildInfo) UserHasAnyRole(user string, roles map[string]bool) bool {
	if len(roles) == 0 {
		return true
	}
	m, err := info.GetMember(user)
	_, reverse := roles["!"]
	if err == nil {
		for _, v := range m.Roles {
			_, ok := roles[v]
			if ok {
				return !reverse
			}
		}
	}
	return reverse
}

// Attempts to get a member from the guild by checking the state first before making the REST API call.
func (info *GuildInfo) GetMember(id string) (*discordgo.Member, error) {
	m, err := sb.dg.State.Member(info.ID, id)
	if err == nil {
		return m, nil
	}
	return sb.dg.GuildMember(info.ID, id)
}

// If a member does not exist, this function creates an entry in the state and returns that.
func (info *GuildInfo) GetMemberCreate(u *discordgo.User) *discordgo.Member {
	m, err := sb.dg.State.Member(info.ID, u.ID)
	if err == nil {
		return m
	}

	m, err = sb.dg.GuildMember(info.ID, u.ID)
	if err != nil || m == nil {
		m = &discordgo.Member{info.ID, "", "", false, false, u, []string{}}
	}
	sb.dg.State.MemberAdd(m)
	return m
}

func ReadUserPingArg(args []string) (uint64, string) {
	if len(args) < 1 {
		return 0, "```You must provide a user to search for.```"
	}
	if len(args[0]) < 3 || args[0][0] != '<' || args[0][1] != '@' {
		return 0, "```The first argument must be an actual ping for the target user, not just their name typed out.```"
	}
	return SBatoi(args[0][2 : len(args[0])-1]), ""
}

func SinceUTC(t time.Time) time.Duration {
	return time.Now().UTC().Sub(t)
}

func getTimezone(info *GuildInfo, user *discordgo.User) *time.Location {
	if user != nil && sb.db.status.get() {
		loc := sb.db.GetTimeZone(SBatoi(user.ID))
		if loc != nil {
			return loc
		}
	}
	loc, err := time.LoadLocation(info.config.Users.TimezoneLocation)
	if err == nil {
		return loc
	}
	return time.UTC
}
func ApplyTimezone(t time.Time, info *GuildInfo, user *discordgo.User) time.Time {
	return t.In(getTimezone(info, user))
}
func IngestEpisode(file string, season int, episode int) {
	f, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err.Error())
	}
	s := strings.Split(strings.Replace(string(f), "\r", "", -1), "\n")

	songmode := false
	lastcharacter := ""
	adjust := 0
	for i := 0; i < len(s); i++ {
		if len(s[i]) > 0 {
			if s[i][0] == '[' {
				action := s[i][1 : len(s[i])-1]
				sb.db.AddTranscript(season, episode, i-adjust, "ACTION", action)
				if !songmode {
					lastcharacter = action
				}
			} else {
				split := strings.SplitN(s[i], ":", 2)
				songmode = (len(split) < 2)
				if songmode {
					prev := sb.db.GetTranscript(season, episode, i-1-adjust, i-1-adjust)
					if len(prev) != 1 {
						fmt.Println(season, " ", episode, " ", i-adjust)
						return
					}
					if prev[0].Speaker == "ACTION" && prev[0].Text == lastcharacter {
						adjust++
						sb.db.RemoveTranscript(season, episode, i-adjust)
					}
					sb.db.AddTranscript(season, episode, i-adjust, lastcharacter, strings.TrimSpace(split[0]))
				} else {
					lastcharacter = strings.TrimSpace(split[0])
					sb.db.AddTranscript(season, episode, i-adjust, lastcharacter, strings.TrimSpace(split[1]))
				}
			}
		} else {
			sb.db.AddTranscript(season, episode, i-adjust, "ACTION", "")
		}
	}
}

func SplitSpeaker(speaker string) []string {
	speakers := strings.Split(strings.Replace(speaker, ", and", " and", -1), " and ")
	speakers = append(strings.Split(speakers[0], ","), speakers[1:]...)
	for i, s := range speakers {
		speakers[i] = strings.Trim(strings.TrimSpace(strings.Replace(s, "Young", "", -1)), "\"")
	}
	return speakers
}

func BuildMarkov(season_start int, episode_start int) {
	regex := regexp.MustCompile("[^~!@#$%^&*()_+`=[\\];,./<>?\" \n\r\f\t\v]+[?!.]?")

	sb.db.sql_ResetMarkov.Exec()

	var cur uint64
	var prev uint64
	var prev2 uint64
	for season := season_start; season <= 5; season++ {
		for episode := episode_start; episode <= 26; episode++ {
			fmt.Println("Begin Episode", episode, "Season", season)
			prev = 0
			prev2 = 0
			lines := sb.db.GetTranscript(season, episode, 0, 999999)
			//lines := []Transcript{ {1, 1, 1, "Twilight", "Twilight went to the bakery to buy some cakes."}, {1, 1, 1, "Twilight", "Twilight went to the library to buy some books"} }
			fmt.Println("Got", len(lines), "lines")

			for i := 0; i < len(lines); i++ {
				if len(lines[i].Text) == 0 {
					if lines[i].Speaker != "ACTION" {
						fmt.Println("UNKNOWN SPEAKER: ", lines[i].Speaker)
					}
					cur = sb.db.AddMarkov(prev, prev2, lines[i].Speaker, "")
					prev2 = 0
					prev = cur // Cur will always be 0 here.
					continue
				}
				words := regex.FindAllString(lines[i].Text, -1)
				speakers := SplitSpeaker(lines[i].Speaker)
				for _, speaker := range speakers {
					if len(speaker) == 0 {
						fmt.Println("EMPTY SPEAKER GENERATED FROM \""+lines[i].Speaker+"\" ON LINE: ", lines[i].Text)
						fmt.Println(speakers)
					}
					for j, _ := range words {
						l := len(words[j])
						ch := words[j][l-1]
						switch ch {
						case '.', '!', '?':
							words[j] = words[j][:l-1]
						}
						if sb.db.GetMarkovWord(speaker, words[j]) != words[j] {
							words[j] = strings.ToLower(words[j])
						}
						//fmt.Println("AddMarkov: ", prev, prev2, speaker, words[j])
						cur = sb.db.AddMarkov(prev, prev2, speaker, words[j])
						prev2 = prev
						prev = cur

						switch ch {
						case '.', '!', '?':
							//fmt.Println("AddMarkov: ", prev, prev2, speaker, string(ch))
							cur = sb.db.AddMarkov(prev, prev2, speaker, string(ch))
							prev2 = 0
							prev = 0
							//prev = sb.db.AddMarkov(prev, "ACTION", "")
						}
					}
				}
			}
		}
	}
}

func FindUsername(user string, info *GuildInfo) []uint64 {
	if len(user) <= 0 {
		return []uint64{}
	}
	if userregex.MatchString(user) {
		return []uint64{SBatoi(user[2 : len(user)-1])}
	}
	if !sb.db.status.get() {
		return []uint64{}
	}
	discriminant := ""
	username := ""
	if discriminantregex.MatchString(user) {
		pos := strings.LastIndex(user, "#")
		if pos >= 0 {
			discriminant = user[pos+1:]
			user = user[:pos]
			username = strings.ToLower(user)
		}
	}
	r := sb.db.FindGuildUsers(user, 20, 0, SBatoi(info.ID))
	if len(r) == 0 {
		user = "%" + user + "%"
		r = sb.db.FindGuildUsers(user, 20, 0, SBatoi(info.ID))
	}
	if len(r) == 0 {
		r = sb.db.FindUsers(user, 20, 0)
	}
	if len(discriminant) > 0 {
		for _, v := range r {
			m, err := info.GetMember(SBitoa(v))
			if err == nil && m.User.Discriminator == discriminant && strings.ToLower(m.User.Username) == username {
				return []uint64{v}
			}
		}
		return []uint64{}
	}
	return r
}

func GetCommandsInOrder(m map[string]Command) []string {
	s := make([]string, 0, len(m))
	for k, _ := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

func MapGetRandomItem(m map[string]bool) string {
	index := rand.Intn(len(m))
	for k, _ := range m {
		if index == 0 {
			return k
		}
		index--
	}

	return "SOMETHING IMPOSSIBLE HAPPENED IN UTIL.GO MapGetRandomItem()! Somebody drag Cloud Hop out of bed and tell him his bot is broken."
}

func MapToSlice(m map[string]bool) []string {
	s := make([]string, 0, len(m))
	for k, _ := range m {
		s = append(s, k)
	}
	return s
}

func MapIntToSlice(m map[int]string) []int {
	s := make([]int, 0, len(m))
	for k, _ := range m {
		s = append(s, k)
	}
	return s
}

func MapStringToSlice(m map[string]string) []string {
	s := make([]string, 0, len(m))
	for k, _ := range m {
		s = append(s, k)
	}
	return s
}

func RemoveSliceString(s *[]string, item string) bool {
	for i := 0; i < len(*s); i++ {
		if (*s)[i] == item {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return true
		}
	}
	return false
}

func RemoveSliceInt(s *[]uint64, item uint64) bool {
	for i := 0; i < len(*s); i++ {
		if (*s)[i] == item {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return true
		}
	}
	return false
}

func CheckMapNilBool(m *map[string]bool) {
	if len(*m) <= 0 {
		*m = make(map[string]bool)
	}
}

func CheckMapNilString(m *map[string]string) {
	if len(*m) <= 0 {
		*m = make(map[string]string)
	}
}

func FindIntSlice(item uint64, s []uint64) bool {
	for _, v := range s {
		if v == item {
			return true
		}
	}
	return false
}
func getUserName(user uint64, info *GuildInfo) string {
	var m *discordgo.Member = nil
	if sb.db.status.get() {
		m, _, _ = sb.db.GetMember(user, SBatoi(info.ID))
	}
	if m == nil {
		return "<@" + SBitoa(user) + ">"
	}
	if len(m.Nick) > 0 {
		return m.Nick
	}
	return m.User.Username
}
func sanitizementionhelper(s string) string {
	return "<\\@" + s[2:]
}
func SanitizeMentions(s string) string {
	return mentionregex.ReplaceAllStringFunc(s, sanitizementionhelper)
}

func replacementionhelper(s string) string {
	if !sb.db.status.get() {
		return s
	}
	u, _, _, _ := sb.db.GetUser(SBatoi(StripPing(s)))
	if u == nil {
		return s
	}
	return u.Username
}
func ReplaceAllMentions(s string) string {
	return SanitizeMentions(userregex.ReplaceAllStringFunc(s, replacementionhelper))
}

func ReplaceAllRolePings(s string, info *GuildInfo) string {
	roles, err := sb.dg.GuildRoles(info.ID)
	if err != nil {
		return s
	}

	return roleregex.ReplaceAllStringFunc(s, func(s string) string {
		r := StripPing(s)
		for _, v := range roles {
			if v.ID == r {
				return v.Name
			}
		}
		return s
	})
}
func RestrictCommand(v string, roles map[string]map[string]bool, alertrole uint64) {
	_, ok := roles[v]
	if !ok && alertrole != 0 {
		roles[v] = make(map[string]bool)
		roles[v][SBitoa(alertrole)] = true
	}
}

// I'm going to find whoever is responsible for not including
// this in Go's standard library and dump them in the middle of
// Death Valley with an entire cactus shoved up their ass.
func AbsInt(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

type legacyBotConfig struct {
	Version               int                        `json:"version"`
	LastVersion           int                        `json:"lastversion"`
	Maxerror              int64                      `json:"maxerror"`
	Maxwit                int64                      `json:"maxwit"`
	Maxbored              int64                      `json:"maxbored"`
	BoredCommands         map[string]bool            `json:"boredcommands"`
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
	MaxSpamRemoveLookback int                        `json:maxspamremovelookback`
	IgnoreInvalidCommands bool                       `json:"ignoreinvalidcommands"`
	UseMemberNames        bool                       `json:"usemembernames"`
	Importable            bool                       `json:"importable"`
	HideNegativeRules     bool                       `json:"hidenegativerules"`
	Timezone              int                        `json:"timezone"`
	TimezoneLocation      string                     `json:"timezonelocation"`
	AutoSilence           int                        `json:"autosilence"`
	AlertRole             uint64                     `json:"alertrole"`
	SilentRole            uint64                     `json:"silentrole"`
	LogChannel            uint64                     `json:"logchannel"`
	ModChannel            uint64                     `json:"modchannel"`
	WelcomeChannel        uint64                     `json:"welcomechannel"`
	WelcomeMessage        string                     `json:"welcomemessage"`
	SilenceMessage        string                     `json:"silencemessage"`
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
	Rules                 map[int]string             `json:"rules"`
}

type legacyBotConfigV10 struct {
	Basic struct {
		Commandperduration int   `json:"commandperduration"`
		Commandmaxduration int64 `json:"commandmaxduration"`
	} `json:"basic"`
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

// migrate settings from earlier config version
func MigrateSettings(config []byte, guild *GuildInfo) error {
	err := json.Unmarshal(config, &guild.config)
	if err != nil {
		return err
	}

	if guild.config.Version < 10 {
		legacy := legacyBotConfig{}
		err := json.Unmarshal(config, &legacy)
		if err != nil {
			return err
		}

		if legacy.Version == 0 {
			newcommands := []string{"addevent", "addbirthday", "autosilence", "silence", "unsilence", "wipewelcome"}
			if len(legacy.Command_roles) == 0 {
				legacy.Command_roles = make(map[string]map[string]bool)
			}
			for _, v := range newcommands {
				RestrictCommand(v, legacy.Command_roles, legacy.AlertRole)
			}
			legacy.MaxImageSpam = 3
			legacy.MaxAttachSpam = 1
			legacy.MaxPingSpam = 24
			legacy.MaxMessageSpam = make(map[int64]int)
			legacy.MaxMessageSpam[1] = 4
			legacy.MaxMessageSpam[9] = 10
			legacy.MaxMessageSpam[12] = 15
		}

		if legacy.Version <= 1 {
			if len(legacy.Aliases) == 0 {
				legacy.Aliases = make(map[string]string)
			}
			legacy.Aliases["cute"] = "pick cute"
			RestrictCommand("new", legacy.Command_roles, legacy.AlertRole)
			RestrictCommand("addquote", legacy.Command_roles, legacy.AlertRole)
			RestrictCommand("removequote", legacy.Command_roles, legacy.AlertRole)
		}

		if legacy.Version <= 2 {
			RestrictCommand("removealias", legacy.Command_roles, legacy.AlertRole)
		}

		if legacy.Version <= 3 {
			legacy.BoredCommands = make(map[string]bool)
		}

		if legacy.Version <= 4 {
			RestrictCommand("delete", legacy.Command_roles, legacy.AlertRole)
		}

		if legacy.Version <= 5 {
			legacy.TimezoneLocation = "Etc/GMT"
			if legacy.Timezone < 0 {
				legacy.TimezoneLocation += "+"
			}
			legacy.TimezoneLocation += strconv.Itoa(-legacy.Timezone) // Etc has the sign reversed
		}

		if legacy.Version <= 6 {
			RestrictCommand("createpoll", legacy.Command_roles, legacy.AlertRole)
			RestrictCommand("deletepoll", legacy.Command_roles, legacy.AlertRole)
		}
		if legacy.Version <= 7 {
			RestrictCommand("addoption", legacy.Command_roles, legacy.AlertRole)
		}
		if legacy.Version <= 8 {
			RestrictCommand("echoembed", legacy.Command_roles, legacy.AlertRole)
		}

		guild.config.Basic.AlertRole = legacy.AlertRole
		guild.config.Basic.Aliases = legacy.Aliases
		guild.config.Basic.Collections = legacy.Collections
		guild.config.Basic.FreeChannels = legacy.FreeChannels
		guild.config.Basic.IgnoreInvalidCommands = legacy.IgnoreInvalidCommands
		guild.config.Basic.Importable = legacy.Importable
		guild.config.Basic.ModChannel = legacy.ModChannel
		guild.config.Modules.CommandChannels = legacy.Command_channels
		guild.config.Modules.CommandDisabled = legacy.Command_disabled
		guild.config.Modules.CommandLimits = legacy.Command_limits
		guild.config.Modules.CommandRoles = legacy.Command_roles
		guild.config.Modules.CommandMaxDuration = legacy.Commandmaxduration
		guild.config.Modules.CommandPerDuration = legacy.Commandperduration
		guild.config.Modules.Channels = legacy.Module_channels
		guild.config.Modules.Disabled = legacy.Module_disabled
		guild.config.Spam.AutoSilence = legacy.AutoSilence
		//guild.config.Spam.MaxAttach = legacy.MaxAttachSpam
		//guild.config.Spam.MaxImages = legacy.MaxImageSpam
		//guild.config.Spam.MaxMessages = legacy.MaxMessageSpam
		//guild.config.Spam.MaxPings = legacy.MaxPingSpam
		guild.config.Spam.RaidTime = legacy.MaxRaidTime
		guild.config.Spam.MaxRemoveLookback = legacy.MaxSpamRemoveLookback
		guild.config.Spam.RaidSize = legacy.RaidSize
		guild.config.Spam.SilenceMessage = legacy.SilenceMessage
		guild.config.Spam.SilentRole = legacy.SilentRole
		guild.config.Bucket.MaxItems = legacy.MaxBucket
		guild.config.Bucket.MaxItemLength = legacy.MaxBucketLength
		guild.config.Bucket.MaxFightDamage = legacy.MaxFightDamage
		guild.config.Bucket.MaxFightHP = legacy.MaxFightHP
		guild.config.Markov.DefaultLines = legacy.Defaultmarkovlines
		guild.config.Markov.MaxPMlines = legacy.MaxPMlines
		guild.config.Markov.MaxLines = legacy.Maxquotelines
		guild.config.Markov.UseMemberNames = legacy.UseMemberNames
		guild.config.Users.TimezoneLocation = legacy.TimezoneLocation
		guild.config.Users.WelcomeChannel = legacy.WelcomeChannel
		guild.config.Users.WelcomeMessage = legacy.WelcomeMessage
		guild.config.Bored.Commands = legacy.BoredCommands
		guild.config.Bored.Cooldown = legacy.Maxbored
		guild.config.Help.HideNegativeRules = legacy.HideNegativeRules
		guild.config.Help.Rules = legacy.Rules
		guild.config.Log.Channel = legacy.LogChannel
		guild.config.Log.Cooldown = legacy.Maxerror
		guild.config.Witty.Cooldown = legacy.Maxwit
		guild.config.Witty.Responses = legacy.Witty
		guild.config.Schedule.BirthdayRole = legacy.BirthdayRole
		guild.config.Search.MaxResults = legacy.Maxsearchresults
		guild.config.Spoiler.Channels = legacy.SpoilChannels
		guild.config.Status.Cooldown = legacy.StatusDelayTime
		guild.config.Quote.Quotes = legacy.Quotes
	}

	if guild.config.Version == 10 {
		legacy := legacyBotConfigV10{}
		err := json.Unmarshal(config, &legacy)
		if err == nil {
			guild.config.Modules.CommandMaxDuration = legacy.Basic.Commandmaxduration
			guild.config.Modules.CommandPerDuration = legacy.Basic.Commandperduration
		} else {
			fmt.Println(err.Error())
		}
	}

	if guild.config.Version <= 11 {
		RestrictCommand("getaudit", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
	}

	if guild.config.Version <= 12 {
		guild.config.Spam.BasePressure = 10.0
		guild.config.Spam.MaxPressure = 60.0
		guild.config.Spam.ImagePressure = ((guild.config.Spam.MaxPressure - guild.config.Spam.BasePressure) / 6.0)
		guild.config.Spam.PingPressure = ((guild.config.Spam.MaxPressure - guild.config.Spam.BasePressure) / 24.0)
		guild.config.Spam.LengthPressure = ((guild.config.Spam.MaxPressure - guild.config.Spam.BasePressure) / (2000.0 * 4))
		guild.config.Spam.RepeatPressure = guild.config.Spam.BasePressure
		guild.config.Spam.PressureDecay = 2.5

		legacy := legacyBotConfigV12{}
		err := json.Unmarshal(config, &legacy)
		if err == nil {
			if legacy.Spam.MaxImages > 0 {
				guild.config.Spam.ImagePressure = ((guild.config.Spam.MaxPressure - guild.config.Spam.BasePressure) / float32(legacy.Spam.MaxImages+1))
			} else {
				guild.config.Spam.ImagePressure = 0
			}
			if legacy.Spam.MaxPings > 0 {
				guild.config.Spam.PingPressure = ((guild.config.Spam.MaxPressure - guild.config.Spam.BasePressure) / float32(legacy.Spam.MaxPings+1))
			} else {
				guild.config.Spam.PingPressure = 0
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	if guild.config.Version <= 13 {
		legacy := legacyBotConfigV13{}
		err := json.Unmarshal(config, &legacy)
		if err == nil {
			guild.config.Users.Roles = make(map[uint64]bool, len(legacy.Basic.Groups))
			idmap := make(map[string]string, len(legacy.Basic.Groups)) // Map initial group name to new role ID

			for k, v := range legacy.Basic.Groups {
				role := k
				check, err := GetRoleByName(role, guild)
				if check != nil {
					role = "sb-" + role
				}
				r, err := sb.dg.GuildRoleCreate(guild.ID)
				if err == nil {
					r, err = sb.dg.GuildRoleEdit(guild.ID, r.ID, role, 0, false, 0, true)
				}
				if err == nil {
					idmap[strings.ToLower(k)] = r.ID
					guild.config.Users.Roles[SBatoi(r.ID)] = true

					for u := range v {
						err = sb.dg.GuildMemberRoleAdd(guild.ID, u, r.ID)
						if err != nil {
							fmt.Println(err)
						}
					}
				} else {
					fmt.Println(err)
				}
			}

			stmt, err := sb.db.Prepare("SELECT ID, Data FROM schedule WHERE Guild = ? AND Type = 7")
			stmt2, err := sb.db.Prepare("UPDATE schedule SET Data = ? WHERE ID = ?")
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

	if guild.config.Version <= 14 {
		RestrictCommand("addrole", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		RestrictCommand("removerole", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		RestrictCommand("deleterole", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
	}

	if guild.config.Version <= 15 {
		RestrictCommand("bannewcomers", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		guild.config.Spam.LockdownDuration = 120
	}

	if guild.config.Version <= 16 {
		guild.config.Basic.CommandPrefix = "!"
	}

	if guild.config.Version <= 17 {
		guild.config.SetupDone = true
	}

	if guild.config.Version <= 18 {
		RestrictCommand("banraid", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		RestrictCommand("getraid", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		RestrictCommand("wipe", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		RestrictCommand("bannewcomers", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		RestrictCommand("getpressure", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		guild.config.Spam.LinePressure = (guild.config.Spam.MaxPressure - guild.config.Spam.BasePressure) / 70.0
	}

	if guild.config.Version != 19 {
		guild.config.Version = 19 // set version to most recent config version
		guild.SaveConfig()
	}
	return nil
}

const (
	FORMAT_PARTIALYEAR = 0
	FORMAT_FULLYEAR    = 1
	FORMAT_NOYEAR      = 2
	FORMAT_STANDARD    = 0
	FORMAT_MILITARY    = 1
	FORMAT_NOTIME      = 2
	FORMAT_ZONEOFFSET  = 0
	FORMAT_ZONEHOURS   = 1
	FORMAT_ZONENAME    = 2
	FORMAT_NOZONE      = 3
)

func getTimeFormat(monthfirst bool, fullmonth bool, year int, hours int, timezone int) string {
	month := "Jan"
	if fullmonth {
		month = "January"
	}
	date := "_2 " + month
	if monthfirst {
		date = month + " _2"
	}
	switch year {
	case FORMAT_PARTIALYEAR:
		date += " 06"
	case FORMAT_FULLYEAR:
		date += " 2006"
	}
	switch hours {
	case FORMAT_STANDARD:
		date += " 3:04pm"
	case FORMAT_MILITARY:
		date += " 15:04"
	}
	switch timezone {
	case FORMAT_ZONEOFFSET:
		date += " -0700"
	case FORMAT_ZONEHOURS:
		date += " -07"
	case FORMAT_ZONENAME:
		date += " MST"
	}
	return date
}
func parseCommonTime(s string, info *GuildInfo, user *discordgo.User) (time.Time, error) {
	var t time.Time
	var err error
	tz := getTimezone(info, user)

	// Iterate through every single imaginable time format that we could possibly parse
	for year := 0; year < 3; year++ {
		for hours := 0; hours < 3; hours++ {
			for timezone := 0; timezone < 4; timezone++ {
				for monthfirst := 0; monthfirst < 2; monthfirst++ {
					for fullmonth := 0; fullmonth < 2; fullmonth++ {
						format := getTimeFormat(monthfirst != 0, fullmonth != 0, year, hours, timezone)

						if timezone == FORMAT_NOZONE {
							t, err = time.ParseInLocation(format, s, tz)
						} else {
							t, err = time.Parse(format, s)
						}
						if err == nil {
							if year == FORMAT_NOYEAR {
								t = t.AddDate(ApplyTimezone(time.Now().UTC(), info, user).Year(), 0, 0)
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

func getAllPerms(info *GuildInfo, user string) (int64, error) {
	m, err := sb.dg.State.Member(info.ID, user)
	if err != nil {
		return 0, err
	}
	var perms int64 = 0
	for _, r := range m.Roles {
		role, err := sb.dg.State.Role(info.ID, r)
		if err != nil {
			perms |= int64(role.Permissions)
		}
	}
	return perms, nil
}

func findServers(name string, guilds []uint64) []*GuildInfo {
	name = strings.ToLower(name)
	info := make([]*GuildInfo, 0, len(guilds))
	for _, g := range guilds {
		sb.guildsLock.RLock()
		guild, ok := sb.guilds[g]
		sb.guildsLock.RUnlock()
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

func getDefaultServer(user uint64) *GuildInfo {
	_, _, _, server := sb.db.GetUser(user)
	if server == nil {
		return nil
	}
	sb.guildsLock.RLock()
	defer sb.guildsLock.RUnlock()
	info, ok := sb.guilds[*server]
	if !ok {
		return nil
	}
	return info
}

func snowflakeTime(id uint64) time.Time {
	return time.Unix(int64(((id>>22)+DiscordEpoch)/1000), 0)
}

func setupSilenceRole(info *GuildInfo) {
	if info.config.Spam.SilentRole > 0 {
		guild, err := sb.dg.State.Guild(info.ID)
		if err != nil {
			info.Log("Failed to setup silence roles!")
			return
		}
		for _, ch := range guild.Channels {
			if SBatoi(ch.ID) != info.config.Users.WelcomeChannel {
				allow := 0
				deny := 0
				for _, v := range ch.PermissionOverwrites {
					if strings.ToLower(v.Type) == "role" && SBatoi(v.ID) == info.config.Spam.SilentRole {
						allow = v.Allow
						deny = v.Deny
						break
					}
				}
				allow &= (^0x00000800)
				deny |= 0x00000800
				sb.dg.ChannelPermissionSet(ch.ID, SBitoa(info.config.Spam.SilentRole), "role", allow, deny)
			}
		}
	}
}
