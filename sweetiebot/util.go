package sweetiebot

import (
	"errors"
	"math/rand"
	"os"
	"path/filepath"
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
	return roleregex.ReplaceAllStringFunc(s, func(s string) string {
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
	if len(m.JoinedAt) > 0 {
		if t, err := time.Parse(time.RFC3339, m.JoinedAt); err == nil {
			return t
		}
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
