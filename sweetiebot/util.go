package sweetiebot

import (
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/blackhole12/discordgo"
)

// ErrRoleNoMatch is thrown when a role name doesn't exist on the server
var ErrRoleNoMatch = errors.New("role doesn't exist on this server")

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
		//fmt.Println("Invalid number ", s, ":", err.Error())
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
	for i := range *s {
		if (*s)[i] == item {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return true
		}
	}
	return false
}

// CheckMapNilBool creates a new map if its nil
func CheckMapNilBool(m *map[CommandID]bool) {
	if len(*m) <= 0 {
		*m = make(map[CommandID]bool)
	}
}

// CheckMapNilString creates a new map if its nil
func CheckMapNilString(m *map[string]string) {
	if len(*m) <= 0 {
		*m = make(map[string]string)
	}
}

// ReplaceAllRolePings finds any role pings and replaces them with the role name
func ReplaceAllRolePings(s string, info *GuildInfo) string {
	guild, err := info.GetGuild()
	if err != nil {
		return s
	}
	return RoleRegex.ReplaceAllStringFunc(s, func(s string) string {
		r := StripPing(s)
		for _, v := range guild.Roles {
			if v.ID == r {
				return v.Name
			}
		}
		return s
	})
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

// Time format options
const (
	FormatPartialYear = 0
	FormatFullYear    = 1
	FormatNoYear      = 2
	FormatYearNum     = 3
	FormatStandard    = 0
	FormatMilitary    = 1
	FormatNoTime      = 2
	FormatTimeNum     = 3
	FormatZoneOffset  = 0
	FormatZoneHours   = 1
	FormatNoZone      = 2
	FormatZoneNum     = 3
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
	}
	return date
}

// SnowflakeTime returns the time a snowflake ID was created
func SnowflakeTime(id uint64) time.Time {
	return time.Unix(int64(((id>>22)+DiscordEpoch)/1000), 0)
}

// WaitForPID loops until there is no longer any running process with the given PID, or returns immediately if no valid ID is given
func WaitForPID(arg string) {
	pid, err := strconv.Atoi(arg)
	if err == nil {
		for {
			p, err := os.FindProcess(pid)
			if err != nil { // On windows, this will return an error if the process doesn't exist
				break
			}
			if err == nil && p != nil {
				err = p.Signal(syscall.Signal(0)) // On POSIX, this returns "no such process" if it doesn't exist
				if err.Error() == "no such process" {
					break
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// GetCurrentDir relative to the executable
func GetCurrentDir() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}

// GuildMemberPermissions gets all permissions for a user, ignoring channel specific overrides
func GuildMemberPermissions(member *discordgo.Member, guild *discordgo.Guild) (apermissions int) {
	if member.User.ID == guild.OwnerID {
		apermissions = discordgo.PermissionAll
		return
	}

	for _, role := range guild.Roles {
		if role.ID == guild.ID {
			apermissions |= role.Permissions
			break
		}
	}

	for _, role := range guild.Roles {
		for _, roleID := range member.Roles {
			if role.ID == roleID {
				apermissions |= role.Permissions
				break
			}
		}
	}

	if apermissions&discordgo.PermissionAdministrator != 0 {
		apermissions |= discordgo.PermissionAllChannel
	}

	return
}

// MemberHasRole returns true if the already resolved member object has the given role ID
func MemberHasRole(m *discordgo.Member, role DiscordRole) bool {
	for _, v := range m.Roles {
		if role.Equals(v) {
			return true
		}
	}
	return false
}
func typeIsPrivate(ty discordgo.ChannelType) bool {
	return ty != discordgo.ChannelTypeGuildText && ty != discordgo.ChannelTypeGuildCategory && ty != discordgo.ChannelTypeGuildVoice
}

// ParseRepeatInterval returns what interval a string refers to
func ParseRepeatInterval(s string) uint8 {
	switch strings.ToLower(s) {
	case "seconds", "second":
		return 1
	case "minutes", "minute":
		return 2
	case "hours", "hour":
		return 3
	case "days", "day":
		return 4
	case "weeks", "week":
		return 5
	case "months", "month":
		return 6
	case "quarters", "quarter":
		return 7
	case "years", "year":
		return 8
	}
	return 255
}

// GetTimestamp returns the timestamp of the last edit of the message or time.Now() if there is no valid timestamp
func GetTimestamp(m *discordgo.Message) time.Time {
	if len(m.EditedTimestamp) > 0 {
		if t, err := m.EditedTimestamp.Parse(); err == nil {
			return t.UTC()
		}
	}
	if t, err := m.Timestamp.Parse(); err == nil {
		return t.UTC()
	}
	return time.Now().UTC()
}

// GetJoinedAt returns either the time the member joined or time.Now() if there is an error
func GetJoinedAt(m *discordgo.Member) time.Time {
	if t, err := m.JoinedAt.Parse(); err == nil {
		return t
	}
	return time.Now().UTC()
}

// GetRoleByName gets a role by its name
func GetRoleByName(role string, info *GuildInfo) (*discordgo.Role, error) {
	roles, err := info.Bot.DG.GuildRoles(info.ID)
	role = strings.ToLower(role)
	if err != nil {
		return nil, err
	}
	for _, v := range roles {
		if strings.ToLower(v.Name) == role {
			return v, nil
		}
	}
	return nil, ErrRoleNoMatch
}

func findChannel(name string, guild *discordgo.Guild) (s []*discordgo.Channel) {
	name = strings.ToLower(name)
	for _, c := range guild.Channels {
		if strings.ToLower(c.Name) == name {
			s = append(s, c)
		}
	}
	return
}

// FindRole returns all roles with the given name
func FindRole(name string, guild *discordgo.Guild) (s []*discordgo.Role) {
	name = strings.ToLower(name)
	for _, r := range guild.Roles {
		if strings.ToLower(r.Name) == name {
			s = append(s, r)
		}
	}
	return
}

// ReturnError formats an error message and returns it as a message
func ReturnError(err error) (string, bool, *discordgo.MessageEmbed) {
	return "```\nError: " + err.Error() + "```", false, nil
}

/*
type transcriptLine struct {
	Character string `json:"character"`
	Text      string `json:"text"`
}

func (sb *SweetieBot) ingestTranscript(data []byte) error {
	transcript := make(map[int]map[int][]transcriptLine)
	if err := json.Unmarshal(data, &transcript); err != nil {
		return err
	}
	for season, v := range transcript {
		for episode, lines := range v {
			for i := 0; i < len(lines); i++ {
				if err := sb.DB.AddTranscript(season, episode, i, lines[i].Character, lines[i].Text); err != nil {
					fmt.Printf("Error at S%vE%v:%v %v:%v", season, episode, i, lines[i].Character, lines[i].Text)
				}
			}
		}
	}
	return nil
}*/

func splitSpeaker(speaker string) []string {
	if len(speaker) == 0 {
		return []string{""}
	}
	speakers := strings.Split(strings.Replace(speaker, ", and", " and", -1), " and ")
	speakers = append(strings.Split(speakers[0], ","), speakers[1:]...)
	for i, s := range speakers {
		speakers[i] = strings.Trim(strings.TrimSpace(strings.Replace(s, "Young", "", -1)), "\"")
	}
	return speakers
}

func (sb *SweetieBot) buildMarkov() {
	if !sb.DB.Status.Get() {
		return
	}

	regex := regexp.MustCompile("[^~!@#$%^&*()_+`=[\\];,./<>?\" \n\r\f\t\v]+[?!.]?")

	markov := &markovChain{
		Speakers: make([]string, 0, 50),
		Phrases:  make([]string, 0, 50),
		Mapping:  make([]uint64, 0, 50),
		Chain:    make(map[uint64]map[uint32]int),
	}

	speakerhash := make(map[string]uint32)
	wordhash := make(map[string]uint32)
	maphash := make(map[uint64]uint32)

	addmarkov := func(prev uint32, prev2 uint32, speaker string, word string) uint32 {
		word = strings.ToLower(word)
		if _, exists := wordhash[word]; !exists {
			wordhash[word] = uint32(len(markov.Phrases))
			markov.Phrases = append(markov.Phrases, word)
		}

		s := speakerhash[speaker]
		w := wordhash[word]
		id := uint64(s)<<32 | uint64(w)
		if _, ok := maphash[id]; !ok {
			maphash[id] = uint32(len(markov.Mapping))
			markov.Mapping = append(markov.Mapping, id)
		}

		p := uint64(prev2)<<32 | uint64(prev)
		cur := maphash[id]
		if _, ok := markov.Chain[p]; !ok {
			markov.Chain[p] = make(map[uint32]int)
		}
		if _, ok := markov.Chain[p][cur]; !ok {
			markov.Chain[p][cur] = 0
		}
		markov.Chain[p][cur]++

		return cur
	}

	var cur uint32
	var prev uint32
	var prev2 uint32

	for season := 1; season <= 15; season++ {
		for episode := 1; episode <= 99; episode++ {
			prev = 0
			prev2 = 0
			lines := sb.DB.GetTranscript(season, episode, 0, 999999)
			if len(lines) == 0 {
				break
			}

			for i := range lines {
				words := regex.FindAllString(lines[i].Text, -1)
				speakers := splitSpeaker(lines[i].Speaker)
				for _, speaker := range speakers {
					if _, exists := speakerhash[speaker]; !exists {
						speakerhash[speaker] = uint32(len(markov.Speakers))
						markov.Speakers = append(markov.Speakers, speaker)
					}

					for j := range words {
						l := len(words[j])
						ch := words[j][l-1]
						switch ch {
						case '.', '!', '?':
							words[j] = words[j][:l-1]
						}

						//fmt.Println("AddMarkov: ", prev, prev2, speaker, words[j])
						cur = addmarkov(prev, prev2, speaker, words[j])
						prev2 = prev
						prev = cur

						switch ch {
						case '.', '!', '?':
							//fmt.Println("AddMarkov: ", prev, prev2, speaker, string(ch))
							cur = addmarkov(prev, prev2, speaker, string(ch))
							prev2 = 0
							prev = 0
						}
					}
				}
			}
		}
	}

	sb.Markov = markov
}
