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

// Pluralize converts i to a string, then appends str to the end, then appends s if it's plural
func Pluralize(i int64, str string) string {
	if i == 1 {
		return strconv.FormatInt(i, 10) + str
	}
	return strconv.FormatInt(i, 10) + str + "s"
}

// TimeDiff gets the largest nonzero time value and displays it
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

// PingAtoi extracts the internal ping ID and converts it to an integer
func PingAtoi(s string) uint64 {
	if len(s) > 2 && (s[:2] == "<#" || s[:2] == "<@") {
		return SBatoi(s[2 : len(s)-1])
	}
	return SBatoi(s)
}

// StripPing strips the ping or channel information and returns the resulting string
func StripPing(s string) string {
	if len(s) > 2 && (s[:2] == "<#" || s[:2] == "<@") {
		if len(s) >= 3 && (s[2:3] == "!" || s[2:3] == "&") {
			return s[3 : len(s)-1]
		}
		return s[2 : len(s)-1]
	}
	return s
}

// SBatoi converts a string to a uint64. Returns 0 if there is an error.
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

// SBitoa converts a uint64 to a string
func SBitoa(i uint64) string {
	return strconv.FormatUint(i, 10)
}

// IsSpace returns true if the byte is considered whitespace in ASCII
func IsSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r'
}

// IDsToUsernames converts an array of integer IDs to an array of username strings
func IDsToUsernames(IDs []uint64, info *GuildInfo, discriminator bool) []string {
	s := make([]string, 0, len(IDs))
	gid := SBatoi(info.ID)
	for _, v := range IDs {
		var m *discordgo.Member
		if sb.DB.status.get() {
			m, _, _ = sb.DB.GetMember(v, gid)
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

// ParseArguments transforms a command line into an array of distinct arguments, while respecting quotes
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

// boolXOR constructs an XOR operator for booleans
func boolXOR(a bool, b bool) bool {
	return (a && !b) || (!a && b)
}

// ReadUserPingArg performs common error handling for resolving user pings
func ReadUserPingArg(args []string) (uint64, string) {
	if len(args) < 1 {
		return 0, "```You must provide a user to search for.```"
	}
	if len(args[0]) < 3 || args[0][0] != '<' || args[0][1] != '@' {
		return 0, "```The first argument must be an actual ping for the target user, not just their name typed out.```"
	}
	return SBatoi(args[0][2 : len(args[0])-1]), ""
}

// SinceUTC returns the difference between now and the given time, in UTC
func SinceUTC(t time.Time) time.Duration {
	return time.Now().UTC().Sub(t)
}

// getTimezone gets the time.Location of the given user, if it exists, otherwise returns time.UTC
func getTimezone(info *GuildInfo, user *discordgo.User) *time.Location {
	if user != nil && sb.DB.status.get() {
		loc := sb.DB.GetTimeZone(SBatoi(user.ID))
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

// ApplyTimezone transforms the given UTC time into local time for the given user
func ApplyTimezone(t time.Time, info *GuildInfo, user *discordgo.User) time.Time {
	return t.In(getTimezone(info, user))
}

func ingestEpisode(file string, season int, episode int) {
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
				sb.DB.AddTranscript(season, episode, i-adjust, "ACTION", action)
				if !songmode {
					lastcharacter = action
				}
			} else {
				split := strings.SplitN(s[i], ":", 2)
				songmode = (len(split) < 2)
				if songmode {
					prev := sb.DB.GetTranscript(season, episode, i-1-adjust, i-1-adjust)
					if len(prev) != 1 {
						fmt.Println(season, " ", episode, " ", i-adjust)
						return
					}
					if prev[0].Speaker == "ACTION" && prev[0].Text == lastcharacter {
						adjust++
						sb.DB.RemoveTranscript(season, episode, i-adjust)
					}
					sb.DB.AddTranscript(season, episode, i-adjust, lastcharacter, strings.TrimSpace(split[0]))
				} else {
					lastcharacter = strings.TrimSpace(split[0])
					sb.DB.AddTranscript(season, episode, i-adjust, lastcharacter, strings.TrimSpace(split[1]))
				}
			}
		} else {
			sb.DB.AddTranscript(season, episode, i-adjust, "ACTION", "")
		}
	}
}

func splitSpeaker(speaker string) []string {
	speakers := strings.Split(strings.Replace(speaker, ", and", " and", -1), " and ")
	speakers = append(strings.Split(speakers[0], ","), speakers[1:]...)
	for i, s := range speakers {
		speakers[i] = strings.Trim(strings.TrimSpace(strings.Replace(s, "Young", "", -1)), "\"")
	}
	return speakers
}

func buildMarkov(seasonStart int, episodeStart int) {
	regex := regexp.MustCompile("[^~!@#$%^&*()_+`=[\\];,./<>?\" \n\r\f\t\v]+[?!.]?")

	sb.DB.sqlResetMarkov.Exec()

	var cur uint64
	var prev uint64
	var prev2 uint64
	for season := seasonStart; season <= 5; season++ {
		for episode := episodeStart; episode <= 26; episode++ {
			fmt.Println("Begin Episode", episode, "Season", season)
			prev = 0
			prev2 = 0
			lines := sb.DB.GetTranscript(season, episode, 0, 999999)
			//lines := []Transcript{ {1, 1, 1, "Twilight", "Twilight went to the bakery to buy some cakes."}, {1, 1, 1, "Twilight", "Twilight went to the library to buy some books"} }
			fmt.Println("Got", len(lines), "lines")

			for i := 0; i < len(lines); i++ {
				if len(lines[i].Text) == 0 {
					if lines[i].Speaker != "ACTION" {
						fmt.Println("UNKNOWN SPEAKER: ", lines[i].Speaker)
					}
					cur = sb.DB.AddMarkov(prev, prev2, lines[i].Speaker, "")
					prev2 = 0
					prev = cur // Cur will always be 0 here.
					continue
				}
				words := regex.FindAllString(lines[i].Text, -1)
				speakers := splitSpeaker(lines[i].Speaker)
				for _, speaker := range speakers {
					if len(speaker) == 0 {
						fmt.Println("EMPTY SPEAKER GENERATED FROM \""+lines[i].Speaker+"\" ON LINE: ", lines[i].Text)
						fmt.Println(speakers)
					}
					for j := range words {
						l := len(words[j])
						ch := words[j][l-1]
						switch ch {
						case '.', '!', '?':
							words[j] = words[j][:l-1]
						}
						if sb.DB.GetMarkovWord(speaker, words[j]) != words[j] {
							words[j] = strings.ToLower(words[j])
						}
						//fmt.Println("AddMarkov: ", prev, prev2, speaker, words[j])
						cur = sb.DB.AddMarkov(prev, prev2, speaker, words[j])
						prev2 = prev
						prev = cur

						switch ch {
						case '.', '!', '?':
							//fmt.Println("AddMarkov: ", prev, prev2, speaker, string(ch))
							cur = sb.DB.AddMarkov(prev, prev2, speaker, string(ch))
							prev2 = 0
							prev = 0
							//prev = sb.DB.AddMarkov(prev, "ACTION", "")
						}
					}
				}
			}
		}
	}
}

// FindUsername returns all possible matching IDs for the given username
func FindUsername(arg string, info *GuildInfo, serveronly bool) []uint64 {
	return findUsernameInternal(arg, info, serveronly, 0)
}

func findUsernameInternal(arg string, info *GuildInfo, serveronly bool, recurse int) []uint64 {
	if len(arg) <= 0 {
		return []uint64{}
	}
	user := arg
	if userregex.MatchString(user) {
		return []uint64{SBatoi(user[2 : len(user)-1])}
	}
	if !sb.DB.status.get() {
		return []uint64{}
	}
	discriminant := ""
	if discriminantregex.MatchString(user) {
		pos := strings.LastIndex(user, "#")
		if pos >= 0 {
			discriminant = user[pos+1:]
			user = strings.ToLower(user[:pos])
		}
	}
	r := sb.DB.FindGuildUsers(user, 20, 0, SBatoi(info.ID))
	if len(r) == 0 {
		user = "%" + user + "%"
		r = sb.DB.FindGuildUsers(user, 20, 0, SBatoi(info.ID))
	}
	if len(r) == 0 && !serveronly {
		r = sb.DB.FindUsers(user, 20, 0)
	}
	if len(discriminant) > 0 {
		for _, v := range r {
			m, err := info.GetMember(SBitoa(v))
			if err == nil && m.User.Discriminator == discriminant && strings.ToLower(m.User.Username) == user {
				return []uint64{v}
			}
		}
		for _, v := range r {
			m, err := info.GetMember(SBitoa(v))
			if err == nil && m.User.Discriminator == discriminant && strings.ToLower(m.Nick) == user {
				return []uint64{v}
			}
		}
		if arg[0] != '@' {
			return []uint64{}
		}
	}
	if arg[0] == '@' && recurse < 1 {
		return findUsernameInternal(arg[1:], info, serveronly, recurse+1)
	}
	return r
}

// GetCommandsInOrder extracts the keys out of a map string and then sorts them alphabetically
func GetCommandsInOrder(m map[string]Command) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

// MapGetRandomItem returns a random item from a map without removing it
func MapGetRandomItem(m map[string]bool) string {
	index := rand.Intn(len(m))
	for k := range m {
		if index == 0 {
			return k
		}
		index--
	}

	return "SOMETHING IMPOSSIBLE HAPPENED IN UTIL.GO MapGetRandomItem()! Somebody drag Cloud Hop out of bed and tell him his bot is broken."
}

// MapToSlice for map[string]bool
func MapToSlice(m map[string]bool) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	return s
}

// MapIntToSlice for map[int]bool
func MapIntToSlice(m map[int]string) []int {
	s := make([]int, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	return s
}

// MapStringToSlice for map[string]string
func MapStringToSlice(m map[string]string) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	return s
}

// RemoveSliceString finds and removes an item from the slice
func RemoveSliceString(s *[]string, item string) bool {
	for i := 0; i < len(*s); i++ {
		if (*s)[i] == item {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveSliceInt finds and removes an item from the slice
func RemoveSliceInt(s *[]uint64, item uint64) bool {
	for i := 0; i < len(*s); i++ {
		if (*s)[i] == item {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return true
		}
	}
	return false
}

// CheckMapNilBool creates a new map if its nil
func CheckMapNilBool(m *map[string]bool) {
	if len(*m) <= 0 {
		*m = make(map[string]bool)
	}
}

// CheckMapNilString creates a new map if its nil
func CheckMapNilString(m *map[string]string) {
	if len(*m) <= 0 {
		*m = make(map[string]string)
	}
}

// FindIntSlice returns true if the given int is in the slice
func FindIntSlice(item uint64, s []uint64) bool {
	for _, v := range s {
		if v == item {
			return true
		}
	}
	return false
}

// getUserName returns a string representation of the user's name if possible, otherwise pings them.
func getUserName(user uint64, info *GuildInfo) string {
	var m *discordgo.Member
	if sb.DB.status.get() {
		m, _, _ = sb.DB.GetMember(user, SBatoi(info.ID))
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

// SanitizeMentions escapes all mentions in a string
func SanitizeMentions(s string) string {
	return mentionregex.ReplaceAllStringFunc(s, sanitizementionhelper)
}

func replacementionhelper(s string) string {
	if !sb.DB.status.get() {
		return s
	}
	u, _, _, _ := sb.DB.GetUser(SBatoi(StripPing(s)))
	if u == nil {
		return s
	}
	return u.Username
}

// ReplaceAllMentions replaces mentions with usernames
func ReplaceAllMentions(s string) string {
	return SanitizeMentions(userregex.ReplaceAllStringFunc(s, replacementionhelper))
}

// ReplaceAllRolePings finds any role pings and replaces them with the role name
func ReplaceAllRolePings(s string, info *GuildInfo) string {
	roles, err := sb.DG.GuildRoles(info.ID)
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

func restrictCommand(v string, roles map[string]map[string]bool, alertrole uint64) {
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

type legacyBotConfigV19 struct {
	Basic struct {
		Collections map[string]map[string]bool `json:"collections"`
	} `json:"basic"`
}

// MigrateSettings from earlier config version
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
				restrictCommand(v, legacy.Command_roles, legacy.AlertRole)
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
			restrictCommand("new", legacy.Command_roles, legacy.AlertRole)
			restrictCommand("addquote", legacy.Command_roles, legacy.AlertRole)
			restrictCommand("removequote", legacy.Command_roles, legacy.AlertRole)
		}

		if legacy.Version <= 2 {
			restrictCommand("removealias", legacy.Command_roles, legacy.AlertRole)
		}

		if legacy.Version <= 3 {
			legacy.BoredCommands = make(map[string]bool)
		}

		if legacy.Version <= 4 {
			restrictCommand("delete", legacy.Command_roles, legacy.AlertRole)
		}

		if legacy.Version <= 5 {
			legacy.TimezoneLocation = "Etc/GMT"
			if legacy.Timezone < 0 {
				legacy.TimezoneLocation += "+"
			}
			legacy.TimezoneLocation += strconv.Itoa(-legacy.Timezone) // Etc has the sign reversed
		}

		if legacy.Version <= 6 {
			restrictCommand("createpoll", legacy.Command_roles, legacy.AlertRole)
			restrictCommand("deletepoll", legacy.Command_roles, legacy.AlertRole)
		}
		if legacy.Version <= 7 {
			restrictCommand("addoption", legacy.Command_roles, legacy.AlertRole)
		}
		if legacy.Version <= 8 {
			restrictCommand("echoembed", legacy.Command_roles, legacy.AlertRole)
		}

		guild.config.Basic.AlertRole = legacy.AlertRole
		guild.config.Basic.Aliases = legacy.Aliases
		guild.config.Collections = legacy.Collections
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
		restrictCommand("getaudit", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
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
				r, err := sb.DG.GuildRoleCreate(guild.ID)
				if err == nil {
					r, err = sb.DG.GuildRoleEdit(guild.ID, r.ID, role, 0, false, 0, true)
				}
				if err == nil {
					idmap[strings.ToLower(k)] = r.ID
					guild.config.Users.Roles[SBatoi(r.ID)] = true

					for u := range v {
						err = sb.DG.GuildMemberRoleAdd(guild.ID, u, r.ID)
						if err != nil {
							fmt.Println(err)
						}
					}
				} else {
					fmt.Println(err)
				}
			}

			stmt, err := sb.DB.Prepare("SELECT ID, Data FROM schedule WHERE Guild = ? AND Type = 7")
			stmt2, err := sb.DB.Prepare("UPDATE schedule SET Data = ? WHERE ID = ?")
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
		restrictCommand("addrole", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		restrictCommand("removerole", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		restrictCommand("deleterole", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
	}

	if guild.config.Version <= 15 {
		restrictCommand("bannewcomers", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		guild.config.Spam.LockdownDuration = 120
	}

	if guild.config.Version <= 16 {
		guild.config.Basic.CommandPrefix = "!"
	}

	if guild.config.Version <= 17 {
		guild.config.SetupDone = true
	}

	if guild.config.Version <= 18 {
		restrictCommand("banraid", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		restrictCommand("getraid", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		restrictCommand("wipe", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		restrictCommand("bannewcomers", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		restrictCommand("getpressure", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		guild.config.Spam.LinePressure = (guild.config.Spam.MaxPressure - guild.config.Spam.BasePressure) / 70.0
	}

	if guild.config.Version <= 19 {
		sb.guildsLock.Lock()
		if len(guild.config.Collections) == 0 {
			guild.config.Collections = make(map[string]map[string]bool)
		}
		legacy := legacyBotConfigV19{}
		err := json.Unmarshal(config, &legacy)
		if err == nil {
			guild.config.Collections["bucket"] = legacy.Basic.Collections["bucket"]
			guild.config.Collections["emote"] = legacy.Basic.Collections["emote"]
			guild.config.Collections["status"] = legacy.Basic.Collections["status"]
			guild.config.Collections["spoiler"] = legacy.Basic.Collections["spoiler"]
			delete(legacy.Basic.Collections, "bucket")
			delete(legacy.Basic.Collections, "emote")
			delete(legacy.Basic.Collections, "status")
			delete(legacy.Basic.Collections, "spoiler")

			gID := SBatoi(guild.ID)
			for k, v := range legacy.Basic.Collections {
				if len(v) > 0 {
					fmt.Println("Importing:", k)
					sb.DB.CreateTag(k, gID)
					tag, err := sb.DB.GetTag(k, gID)
					if err == nil {
						for item := range v {
							id, err := sb.DB.AddItem(item)
							if err == nil || err != ErrDuplicateEntry {
								sb.DB.AddTag(id, tag)
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
		sb.guildsLock.Unlock()
		restrictCommand("addset", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		restrictCommand("removeset", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
		restrictCommand("searchset", guild.config.Modules.CommandRoles, guild.config.Basic.AlertRole)
	}

	if guild.config.Version != 20 {
		guild.config.Version = 20 // set version to most recent config version
		guild.SaveConfig()
	}
	return nil
}

// Time format options
const (
	FormatPartialYear = 0
	FormatFullYear    = 1
	FormatNoYear      = 2
	FormatStandard    = 0
	FormatMilitary    = 1
	FormatNoTime      = 2
	FormatZoneOffset  = 0
	FormatZoneHours   = 1
	FormatZoneName    = 2
	FormatNoZone      = 3
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
	case FormatPartialYear:
		date += " 06"
	case FormatFullYear:
		date += " 2006"
	}
	switch hours {
	case FormatStandard:
		date += " 3:04pm"
	case FormatMilitary:
		date += " 15:04"
	}
	switch timezone {
	case FormatZoneOffset:
		date += " -0700"
	case FormatZoneHours:
		date += " -07"
	case FormatZoneName:
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

						if timezone == FormatNoZone {
							t, err = time.ParseInLocation(format, s, tz)
						} else {
							t, err = time.Parse(format, s)
						}
						if err == nil {
							if year == FormatNoYear {
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
	m, err := sb.DG.State.Member(info.ID, user)
	if err != nil {
		return 0, err
	}
	var perms int64
	for _, r := range m.Roles {
		role, err := sb.DG.State.Role(info.ID, r)
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
	_, _, _, server := sb.DB.GetUser(user)
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
		guild, err := sb.DG.State.Guild(info.ID)
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
				sb.DG.ChannelPermissionSet(ch.ID, SBitoa(info.config.Spam.SilentRole), "role", allow, deny)
			}
		}
	}
}

// UnsilenceMember unsilences the member, if they are silenced
func UnsilenceMember(user uint64, info *GuildInfo) error {
	m, err := info.GetMember(SBitoa(user))
	if err == nil {
		sb.DG.State.Lock()
		RemoveSliceString(&m.Roles, SBitoa(info.config.Spam.SilentRole))
		sb.DG.State.Unlock()
	}

	return sb.DG.GuildMemberRoleRemove(info.ID, SBitoa(user), SBitoa(info.config.Spam.SilentRole))
}

func assignRoleDiscord(userID string, roleID string, info *GuildInfo) {
	err := sb.DG.GuildMemberRoleAdd(info.ID, userID, roleID)
	info.LogError(fmt.Sprintf("GuildMemberRoleAdd(%s, %s, %s) returned error: ", info.ID, userID, roleID), err)
}

// assignRoleMember adds a role to a member that already exists
func assignRoleMember(userID string, roleID string, info *GuildInfo) int8 {
	defer assignRoleDiscord(userID, roleID, info)
	m, merr := info.GetMember(userID)
	if merr == nil { // Manually set our internal state to say this role is set to prevent race conditions
		sb.DG.State.Lock()
		defer sb.DG.State.Unlock()
		if info.MemberHasRole(m, roleID) {
			return 1
		}
		m.Roles = append(m.Roles, roleID)
	}

	return 0
}
