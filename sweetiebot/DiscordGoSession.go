package sweetiebot

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/blackhole12/discordgo"
)

var errNotChannel = errors.New("string is not a valid channel")
var errNotRole = errors.New("string is not a valid role")
var errNotUser = errors.New("string is not a valid user")

// DiscordChannel stores a channel ID
type DiscordChannel string

// DiscordRole stores a role ID
type DiscordRole string

// DiscordUser stores a user ID
type DiscordUser string

// DiscordGuild stores a guild ID
type DiscordGuild string

const (
	ChannelEmpty     = DiscordChannel("")
	ChannelExclusion = DiscordChannel("!")
	RoleEmpty        = DiscordRole("")
	RoleExclusion    = DiscordRole("!")
	UserEmpty        = DiscordUser("")
	GuildEmpty       = DiscordGuild("")
)

// Display channel as a ping
func (ch DiscordChannel) Display() string {
	return fmt.Sprintf("<#%v>", ch)
}

// Display role as a ping
func (r DiscordRole) Display() string {
	return fmt.Sprintf("<@&%v>", r)
}

// Display user as a ping
func (u DiscordUser) Display() string {
	return fmt.Sprintf("<@%v>", u)
}

// Show channel name if available, or display ping
func (ch DiscordChannel) Show(info *GuildInfo) string {
	if channel, err := info.Bot.DG.State.Channel(string(ch)); err == nil {
		return "#" + channel.Name
	}
	return ch.Display()
}

// Show role name if available, or display ping
func (r DiscordRole) Show(info *GuildInfo) string {
	if role, err := info.Bot.DG.State.Role(info.ID, string(r)); err == nil {
		return "@" + role.Name
	}
	return r.Display()
}

// Convert channel to integer
func (ch DiscordChannel) Convert() (i uint64) {
	i, _ = strconv.ParseUint(string(ch), 10, 64)
	return
}

// Convert role to integer
func (r DiscordRole) Convert() (i uint64) {
	i, _ = strconv.ParseUint(string(r), 10, 64)
	return
}

// Convert user to integer
func (u DiscordUser) Convert() (i uint64) {
	i, _ = strconv.ParseUint(string(u), 10, 64)
	return
}

// Convert guild to integer
func (g DiscordGuild) Convert() (i uint64) {
	i, _ = strconv.ParseUint(string(g), 10, 64)
	return
}

func (ch DiscordChannel) String() string {
	return string(ch)
}

func (r DiscordRole) String() string {
	return string(r)
}

func (u DiscordUser) String() string {
	return string(u)
}

func (g DiscordGuild) String() string {
	return string(g)
}

// UnmarshalJSON is a custom unmarshal function for JSON
func (ch *DiscordChannel) UnmarshalJSON(d []byte) error {
	s := ""
	err := json.Unmarshal(d, &s)
	if err == nil {
		*ch = DiscordChannel(s)
	} else {
		var i uint64
		err = json.Unmarshal(d, &i)
		if err == nil {
			*ch = DiscordChannel(strconv.FormatUint(i, 10))
		}
	}
	return err
}

// UnmarshalJSON is a custom unmarshal function for JSON
func (r *DiscordRole) UnmarshalJSON(d []byte) error {
	s := ""
	err := json.Unmarshal(d, &s)
	if err == nil {
		*r = DiscordRole(s)
	} else {
		var i uint64
		err = json.Unmarshal(d, &i)
		if err == nil {
			*r = DiscordRole(strconv.FormatUint(i, 10))
		}
	}
	return err
}

// Equals channel id
func (ch DiscordChannel) Equals(s string) bool {
	return ch != ChannelEmpty && string(ch) == s
}

// Equals role id
func (r DiscordRole) Equals(s string) bool {
	return r != RoleEmpty && string(r) == s
}

// Equals user id
func (u DiscordUser) Equals(s string) bool {
	return u != UserEmpty && string(u) == s
}

// Equals guild id
func (g DiscordGuild) Equals(s string) bool {
	return g != GuildEmpty && string(g) == s
}

// NewDiscordChannel constructs a new DiscordChannel from an integer
func NewDiscordChannel(i uint64) DiscordChannel {
	return DiscordChannel(strconv.FormatUint(i, 10))
}

// NewDiscordRole constructs a new DiscordRole from an integer
func NewDiscordRole(i uint64) DiscordRole {
	return DiscordRole(strconv.FormatUint(i, 10))
}

// NewDiscordUser constructs a new DiscordUser from an integer
func NewDiscordUser(i uint64) DiscordUser {
	return DiscordUser(strconv.FormatUint(i, 10))
}

// NewDiscordGuild constructs a new DiscordGuild from an integer
func NewDiscordGuild(i uint64) DiscordGuild {
	return DiscordGuild(strconv.FormatUint(i, 10))
}

// ParseChannel resolves multiple different channel tagging formats
func ParseChannel(s string, guild *discordgo.Guild) (DiscordChannel, error) {
	if len(s) == 0 {
		return ChannelEmpty, nil
	}
	if s == "!" {
		return ChannelExclusion, nil
	}
	if s[0] == '<' {
		matches := ChannelRegex.FindStringSubmatch(s)
		if len(matches) < 2 || len(matches[1]) == 0 {
			return ChannelEmpty, errNotChannel
		}
		s = matches[1]
	} else if guild != nil {
		var ch []*discordgo.Channel
		if s[0] == '#' {
			ch = findChannel(s[1:], guild)
		}
		if len(ch) == 0 {
			ch = findChannel(s, guild)
		}
		if len(ch) > 0 {
			if len(ch) > 1 {
				join := make([]string, len(ch), len(ch))
				for k, v := range ch {
					join[k] = v.Name + " (" + v.ID + ")"
				}
				return ChannelEmpty, errors.New("could be any of the following: " + strings.Join(join, ", "))
			}
			s = ch[0].ID
		}
	}
	if _, err := strconv.ParseUint(s, 10, 64); err != nil { // Check that it's a valid integer
		return ChannelEmpty, errNotChannel
	}
	return DiscordChannel(s), nil
}

// ParseRole resolves multiple different role tagging formats
func ParseRole(s string, guild *discordgo.Guild) (DiscordRole, error) {
	if len(s) == 0 {
		return RoleEmpty, nil
	}
	if s == "!" {
		return RoleExclusion, nil
	}
	if s[0] == '<' {
		matches := RoleRegex.FindStringSubmatch(s)
		if len(matches) < 2 || len(matches[1]) == 0 {
			return RoleEmpty, errNotRole
		}
		s = matches[1]
	} else if guild != nil {
		var r []*discordgo.Role
		if s[0] == '@' {
			r = FindRole(s[1:], guild)
		}
		if len(r) == 0 {
			r = FindRole(s, guild)
		}
		if len(r) > 0 {
			if len(r) > 1 {
				join := make([]string, len(r), len(r))
				for k, v := range r {
					join[k] = v.Name + " (" + v.ID + ")"
				}
				return RoleEmpty, errors.New("could be any of the following: " + strings.Join(join, ", "))
			}
			s = r[0].ID
		}
	}
	if _, err := strconv.ParseUint(s, 10, 64); err != nil { // Check that it's a valid integer
		return RoleEmpty, errNotRole
	}
	return DiscordRole(s), nil
}

// ParseUser resolves multiple different user tagging formats
func ParseUser(s string, info *GuildInfo) (DiscordUser, error) {
	if len(s) == 0 {
		return UserEmpty, errNotUser
	}
	if s[0] == '<' {
		matches := UserRegex.FindStringSubmatch(s)
		if len(matches) < 2 || len(matches[1]) == 0 {
			return UserEmpty, errNotUser
		}
		s = matches[1]
	} else if info != nil {
		var IDs []uint64
		s = strings.ToLower(s)
		IDs = info.FindUsername(s)
		if len(IDs) == 0 {
			return UserEmpty, errNotUser
		}
		if len(IDs) > 1 {
			join := info.IDsToUsernames(IDs, true)
			return UserEmpty, errors.New("Could be any of the following users or their aliases:\n" + info.Sanitize(strings.Join(join, "\n"), CleanCodeBlock))
		}
		return NewDiscordUser(IDs[0]), nil
	}
	if _, err := strconv.ParseUint(s, 10, 64); err != nil { // Check that it's a valid integer
		return UserEmpty, errNotUser
	}
	return DiscordUser(s), nil
}

// DiscordGoSession overrides the discordgo session, allowing us to extend it and also lets us mock the base class methods for testing
type DiscordGoSession struct {
	discordgo.Session
}

// RemoveRole removes a role from the state and then sends a request to discord to remove it
func (s *DiscordGoSession) RemoveRole(guildID string, userID DiscordUser, role DiscordRole) error {
	m, err := s.GetMember(userID, guildID)
	if err == nil {
		nroles := make([]string, len(m.Roles)) // We set this to a new slice so we can atomically replace it on x86 architectures, avoiding a lock
		copy(nroles, m.Roles)
		RemoveSliceString(&nroles, role.String())
		m.Roles = nroles
	}

	return s.GuildMemberRoleRemove(guildID, userID.String(), role.String())
}

// GetMember attempts to get a member from the guild by checking the state first before making the REST API call.
func (s *DiscordGoSession) GetMember(userID DiscordUser, guildID string) (*discordgo.Member, error) {
	m, err := s.State.Member(guildID, userID.String())
	if err == nil {
		return m, nil
	}
	return s.GuildMember(guildID, userID.String())
}

// GetMemberCreate creates a member if they don't exist, so it is guaranteed to return a Member
func (s *DiscordGoSession) GetMemberCreate(u *discordgo.User, guildID string) *discordgo.Member {
	m, err := s.State.Member(guildID, u.ID)
	if err == nil {
		return m
	}

	m, err = s.GuildMember(guildID, u.ID)
	if err != nil || m == nil {
		m = &discordgo.Member{guildID, "", "", false, false, u, []string{}, ""}
	}
	s.State.MemberAdd(m)
	return m
}

// ChangeBotName changes the username and avatar of the bot
func (s *DiscordGoSession) ChangeBotName(name string, avatarfile string) error {
	data := ""
	if len(avatarfile) > 0 {
		binary, _ := ioutil.ReadFile(avatarfile)
		avatar := base64.StdEncoding.EncodeToString(binary)
		data = fmt.Sprintf("data:image/%s;base64,%s", filepath.Ext(avatarfile), avatar)
	}
	_, err := s.UserUpdate("", "", name, data, "")
	return err
}

// UserHasAnyRole returns true if the user ID (as a string) has any of the role IDs given (as a map of strings)
func (s *DiscordGoSession) UserHasAnyRole(user DiscordUser, guildID string, roles map[DiscordRole]bool) bool {
	if len(roles) == 0 {
		return true
	}
	m, err := s.GetMember(user, guildID)
	_, reverse := roles[RoleExclusion]
	if err == nil {
		for _, v := range m.Roles {
			if _, ok := roles[DiscordRole(v)]; ok {
				return !reverse
			}
		}
	}
	return reverse
}

// UserPermissions gets all permissions for a user, ignoring channel specific overrides
func (s *DiscordGoSession) UserPermissions(userID DiscordUser, guildID string) (int, error) {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		return 0, err
	}

	member, err := s.State.Member(guild.ID, userID.String())
	if err != nil {
		return 0, err
	}

	s.State.RLock()
	defer s.State.RUnlock()
	return GuildMemberPermissions(member, guild), nil
}

// ReplaceAllMentions replaces mentions with usernames
func (s *DiscordGoSession) ReplaceAllMentions(str string, db *BotDB, guildID string) string {
	if len(guildID) > 0 {
		return UserRegex.ReplaceAllStringFunc(str, func(match string) string {
			m, err := s.State.Member(guildID, StripPing(match))
			if err != nil || m == nil {
				return match
			}
			if len(m.Nick) > 0 {
				return m.Nick
			}
			return m.User.Username
		})
	}

	return UserRegex.ReplaceAllStringFunc(str, func(match string) string {
		u, _, _, _ := db.GetUser(SBatoi(StripPing(match)))
		if u == nil {
			return match
		}
		return u.Username
	})
}

// BulkDeleteBypass deletes in batches of 99
func (s *DiscordGoSession) BulkDeleteBypass(channelID string, messages []string) (err error) {
	i := 0
	n := len(messages)
	for (n - i) > 99 {
		err := s.ChannelMessagesBulkDelete(channelID, messages[i:i+99])
		if err != nil {
			return err
		}
		i += 99
	}
	return s.ChannelMessagesBulkDelete(channelID, messages[i:])
}
