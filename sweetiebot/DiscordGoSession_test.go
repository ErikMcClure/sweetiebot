package sweetiebot

import (
	"fmt"
	"strconv"
	"testing"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/erikmcclure/discordgo"
)

func (s *DiscordGoSession) AddHandler(handler interface{}) func() {
	mock.Input(interface{}(s.AddHandler), handler)
	return func() {}
}
func (s *DiscordGoSession) Channel(channelID string) (st *discordgo.Channel, err error) {
	if channelID == strconv.Itoa(TestChannelPrivate) {
		return &discordgo.Channel{
			ID:      strconv.Itoa(TestChannelPrivate),
			GuildID: "",
			Name:    "Private Channel",
			Topic:   "",
			Type:    discordgo.ChannelTypeDM,
			NSFW:    false,
		}, nil
	}
	if channelID == strconv.Itoa(TestChannelGroupDM) {
		return &discordgo.Channel{
			ID:      strconv.Itoa(TestChannelGroupDM),
			GuildID: "",
			Name:    "Private Channel",
			Topic:   "",
			Type:    discordgo.ChannelTypeGroupDM,
			NSFW:    false,
		}, nil
	}
	mock.Input(interface{}(s.Channel), channelID)
	return s.State.Channel(channelID)
}
func (s *DiscordGoSession) ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string) (st []*discordgo.Message, err error) {
	mock.Input(interface{}(s.ChannelMessages), channelID, limit, beforeID, afterID, aroundID)
	return
}
func (s *DiscordGoSession) ChannelMessage(channelID, messageID string) (st *discordgo.Message, err error) {
	mock.Input(interface{}(s.ChannelMessage), channelID, messageID)
	return
}
func (s *DiscordGoSession) ChannelMessageSend(channelID string, content string) (st *discordgo.Message, err error) {
	mock.Input(interface{}(s.ChannelMessageSend), channelID, content)
	return
}
func (s *DiscordGoSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend) (st *discordgo.Message, err error) {
	mock.Input(interface{}(s.ChannelMessageSendComplex), channelID, data)
	return
}
func (s *DiscordGoSession) ChannelMessagesBulkDelete(channelID string, messages []string) (err error) {
	mock.Input(interface{}(s.ChannelMessagesBulkDelete), channelID, messages)
	return nil
}

func (s *DiscordGoSession) ChannelMessageDelete(channelID, messageID string) (err error) {
	mock.Input(interface{}(s.ChannelMessageDelete), channelID, messageID)
	return nil
}
func (s *DiscordGoSession) ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed) (m *discordgo.Message, err error) {
	mock.Input(interface{}(s.ChannelMessageSendEmbed), channelID, embed)
	return
}
func (s *DiscordGoSession) ChannelPermissionSet(channelID, targetID, targetType string, allow, deny int) (err error) {
	mock.Input(interface{}(s.ChannelPermissionSet), channelID, targetID, targetType, allow, deny)
	return
}
func (s *DiscordGoSession) Guild(guildID string) (st *discordgo.Guild, err error) {
	mock.Input(interface{}(s.Guild), guildID)
	return s.State.Guild(guildID)
}
func (s *DiscordGoSession) GuildEdit(guildID string, g discordgo.GuildParams) (st *discordgo.Guild, err error) {
	mock.Input(interface{}(s.GuildEdit), guildID, g)
	return
}

func (s *DiscordGoSession) GuildBanCreate(guildID, userID string, days int) (err error) {
	mock.Input(interface{}(s.GuildBanCreate), guildID, userID, days)
	return nil
}
func (s *DiscordGoSession) GuildBanCreateWithReason(guildID, userID, reason string, days int) (err error) {
	mock.Input(interface{}(s.GuildBanCreateWithReason), guildID, userID, reason, days)
	return nil
}

func (s *DiscordGoSession) GuildBanDelete(guildID, userID string) (err error) {
	mock.Input(interface{}(s.GuildBanDelete), guildID, userID)
	return nil
}
func (s *DiscordGoSession) GuildMembers(guildID string, after string, limit int) (st []*discordgo.Member, err error) {
	mock.Input(interface{}(s.GuildMembers), guildID, after, limit)
	return
}

func (s *DiscordGoSession) GuildMember(guildID, userID string) (st *discordgo.Member, err error) {
	mock.Input(interface{}(s.GuildMember), guildID, userID)
	return s.State.Member(guildID, userID)
}
func (s *DiscordGoSession) GuildMemberRoleAdd(guildID, userID, roleID string) (err error) {
	mock.Input(interface{}(s.GuildMemberRoleAdd), guildID, userID, roleID)
	return
}
func (s *DiscordGoSession) GuildMemberRoleRemove(guildID, userID, roleID string) (err error) {
	mock.Input(interface{}(s.GuildMemberRoleRemove), guildID, userID, roleID)
	return
}
func (s *DiscordGoSession) GuildChannels(guildID string) (st []*discordgo.Channel, err error) {
	mock.Input(interface{}(s.GuildChannels), guildID)
	var g *discordgo.Guild
	g, err = s.State.Guild(guildID)
	if err == nil {
		st = g.Channels
	}
	return
}
func (s *DiscordGoSession) GuildRoles(guildID string) (st []*discordgo.Role, err error) {
	mock.Input(interface{}(s.GuildRoles), guildID)
	var g *discordgo.Guild
	g, err = s.State.Guild(guildID)
	if err == nil {
		st = g.Roles
	}
	return
}
func (s *DiscordGoSession) GuildRoleCreate(guildID string) (st *discordgo.Role, err error) {
	mock.Input(interface{}(s.GuildRoleCreate), guildID)
	return
}
func (s *DiscordGoSession) GuildRoleEdit(guildID, roleID, name string, color int, hoist bool, perm int, mention bool) (st *discordgo.Role, err error) {
	mock.Input(interface{}(s.GuildRoleEdit), guildID, roleID, name, color, hoist, perm, mention)
	return
}
func (s *DiscordGoSession) GuildRoleDelete(guildID, roleID string) (err error) {
	mock.Input(interface{}(s.GuildRoleDelete), guildID, roleID)
	return
}

func (s *DiscordGoSession) User(userID string) (st *discordgo.User, err error) {
	mock.Input(interface{}(s.User), userID)
	for _, g := range s.State.Guilds {
		for _, m := range g.Members {
			if m.User.ID == userID {
				return m.User, nil
			}
		}
	}
	return nil, discordgo.ErrStateNotFound
}
func (s *DiscordGoSession) UserUpdate(email, password, username, avatar, newPassword string) (st *discordgo.User, err error) {
	mock.Input(interface{}(s.UserUpdate), email, password, username, avatar, newPassword)
	return
}
func (s *DiscordGoSession) UserChannelCreate(recipientID string) (st *discordgo.Channel, err error) {
	mock.Input(interface{}(s.UserChannelCreate), recipientID)
	return &discordgo.Channel{
		ID:   recipientID + "10",
		Type: discordgo.ChannelTypeDM,
	}, nil
}
func (s *DiscordGoSession) UpdateStatus(idle int, game string) (err error) {
	mock.Input(interface{}(s.UpdateStatus), idle, game)
	return nil
}
func (s *DiscordGoSession) Open() (err error) {
	mock.Input(interface{}(s.Open))
	return nil
}
func (s *DiscordGoSession) Close() (err error) {
	mock.Input(interface{}(s.Close))
	return nil
}
func (s *DiscordGoSession) RequestWithLockedBucket(method, urlStr, contentType string, b []byte, bucket *discordgo.Bucket, sequence int) (response []byte, err error) {
	defer bucket.Unlock()
	mock.Input(interface{}(s.RequestWithLockedBucket), method, urlStr, contentType, b, bucket, sequence)
	return
}

func TestRemoveRole(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF
		member := DiscordUser(strconv.Itoa(TestMod | int(i)))
		m, _ := v.Bot.DG.GetMember(member, v.ID)
		modrole := DiscordRole(strconv.Itoa(TestRoleMod | int(i)))
		memberrole := DiscordRole(strconv.Itoa(TestRoleMember | int(i)))
		Check(m.Roles[0], modrole.String(), t)
		mock.Expect(v.Bot.DG.GuildMemberRoleRemove, v.ID, member.String(), modrole.String())
		v.Bot.DG.RemoveRole(v.ID, member, modrole)
		Check(len(m.Roles), 1, t)
		mock.Expect(v.Bot.DG.GuildMemberRoleRemove, v.ID, member.String(), modrole.String())
		v.Bot.DG.RemoveRole(v.ID, member, modrole)
		Check(len(m.Roles), 1, t)
		mock.Expect(v.Bot.DG.GuildMemberRoleRemove, v.ID, member.String(), "asdf")
		v.Bot.DG.RemoveRole(v.ID, member, "asdf")
		Check(len(m.Roles), 1, t)
		mock.Expect(v.Bot.DG.GuildMemberRoleRemove, v.ID, member.String(), memberrole.String())
		v.Bot.DG.RemoveRole(v.ID, member, memberrole)
		Check(len(m.Roles), 0, t)
		mock.Expect(v.Bot.DG.GuildMemberRoleRemove, v.ID, member.String(), memberrole.String())
		v.Bot.DG.RemoveRole(v.ID, member, memberrole)
		Check(len(m.Roles), 0, t)
		mock.Expect(v.Bot.DG.GuildMemberRoleRemove, v.ID, member.String(), "asdf")
		v.Bot.DG.RemoveRole(v.ID, member, "asdf")
		Check(len(m.Roles), 0, t)
		mock.Expect(v.Bot.DG.GuildMember, v.ID, "0")
		mock.Expect(v.Bot.DG.GuildMemberRoleRemove, v.ID, "0", "asdf")
		v.Bot.DG.RemoveRole(v.ID, DiscordUser("0"), DiscordRole("asdf"))
		mock.Expect(v.Bot.DG.GuildMember, "0", "0")
		mock.Expect(v.Bot.DG.GuildMemberRoleRemove, "0", "0", "asdf")
		v.Bot.DG.RemoveRole("0", DiscordUser("0"), DiscordRole("asdf"))
	}
}

func TestGetMember(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF

		m, err := v.Bot.DG.GetMember(NewDiscordUser(TestUserBoring|i), v.ID)
		Check(err, nil, t)
		Check(m.User.ID, NewDiscordUser(TestUserBoring|i).String(), t)
		mock.Expect(v.Bot.DG.GuildMember, v.ID, "0")
		v.Bot.DG.GetMember("0", v.ID)
	}
}

func TestGetMemberCreate(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for k, v := range sb.Guilds {
		i := int(k.Convert() & 0xFF)
		s := strconv.Itoa(i)
		m := v.Bot.DG.GetMemberCreate(&discordgo.User{ID: strconv.Itoa(TestUserBoring | i)}, v.ID)
		Check(m.User.ID, strconv.Itoa(TestUserBoring|i), t)
		mock.Expect(v.Bot.DG.GuildMember, v.ID, s)
		Check(v.Bot.DG.GetMemberCreate(&discordgo.User{ID: s}, v.ID).User.ID, s, t)
		m, _ = v.Bot.DG.State.Member(v.ID, s)
		Check(m.User.ID, s, t)
	}
}

func TestUserHasAnyRole(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF

		Check(v.Bot.DG.UserHasAnyRole("0", v.ID, map[DiscordRole]bool{}), true, t)
		mock.Expect(v.Bot.DG.GuildMember, v.ID, "0")
		Check(v.Bot.DG.UserHasAnyRole("0", v.ID, map[DiscordRole]bool{"1": true}), false, t)
		mock.Expect(v.Bot.DG.GuildMember, v.ID, "0")
		Check(v.Bot.DG.UserHasAnyRole("0", v.ID, map[DiscordRole]bool{"0": true, "1": true, "2": true}), false, t)
		Check(v.Bot.DG.UserHasAnyRole(NewDiscordUser(TestUserBoring|i), v.ID, map[DiscordRole]bool{}), true, t)
		Check(v.Bot.DG.UserHasAnyRole(NewDiscordUser(TestUserBoring|i), v.ID, map[DiscordRole]bool{NewDiscordRole(TestRoleUser | i): true, "1": true, "2": true}), false, t)
		Check(v.Bot.DG.UserHasAnyRole(NewDiscordUser(TestUserAssigned|i), v.ID, map[DiscordRole]bool{}), true, t)
		Check(v.Bot.DG.UserHasAnyRole(NewDiscordUser(TestUserAssigned|i), v.ID, map[DiscordRole]bool{NewDiscordRole(TestRoleMod | i): true}), false, t)
		Check(v.Bot.DG.UserHasAnyRole(NewDiscordUser(TestUserAssigned|i), v.ID, map[DiscordRole]bool{NewDiscordRole(TestRoleMod | i): true, NewDiscordRole(TestRoleSilence | i): true}), false, t)
		Check(v.Bot.DG.UserHasAnyRole(NewDiscordUser(TestUserAssigned|i), v.ID, map[DiscordRole]bool{NewDiscordRole(TestRoleAssign | i): true}), true, t)
		Check(v.Bot.DG.UserHasAnyRole(NewDiscordUser(TestUserAssigned|i), v.ID, map[DiscordRole]bool{NewDiscordRole(TestRoleAssign | i): true, NewDiscordRole(TestRoleMod | i): true, NewDiscordRole(TestRoleSilence | i): true}), true, t)
	}
}

func TestUserPermissions(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	_, err := sb.DG.UserPermissions("0", "0")
	Check(err, discordgo.ErrStateNotFound, t)

	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF

		_, err := v.Bot.DG.UserPermissions("0", v.ID)
		Check(err, discordgo.ErrStateNotFound, t)
		p, _ := v.Bot.DG.UserPermissions(NewDiscordUser(TestSelfID), v.ID)
		Check(p, 0, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestOwnerBot), v.ID)
		Check(p, 0, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestOwnerServer|i), v.ID)
		Check(p, discordgo.PermissionAll, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestAdminMod|i), v.ID)
		Check(p, discordgo.PermissionAllChannel|discordgo.PermissionAdministrator, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestAdmin|i), v.ID)
		Check(p, discordgo.PermissionAllChannel|discordgo.PermissionAdministrator, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestMod|i), v.ID)
		Check(p, mockDiscordRole(TestRoleMod, int(i)).Permissions, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestUserAssigned|i), v.ID)
		Check(p, mockDiscordRole(TestRoleAssign, int(i)).Permissions, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestUserNonAssign|i), v.ID)
		Check(p, mockDiscordRole(TestRoleUser, int(i)).Permissions, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestUserBoring|i), v.ID)
		Check(p, mockDiscordRole(TestRoleMember, int(i)).Permissions, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestUserSilence|i), v.ID)
		Check(p, 0, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestUserNew|i), v.ID)
		Check(p, 0, t)
		p, _ = v.Bot.DG.UserPermissions(NewDiscordUser(TestUserBot|i), v.ID)
		Check(p, 0, t)
	}
}

func TestBulkDeleteBypass(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for _, v := range sb.Guilds {
		mock.Expect(v.Bot.DG.ChannelMessagesBulkDelete, "a", []string{})
		v.Bot.DG.BulkDeleteBypass("a", []string{})
		mock.Expect(v.Bot.DG.ChannelMessagesBulkDelete, "a", []string{"0"})
		v.Bot.DG.BulkDeleteBypass("a", []string{"0"})
		msgs := make([]string, 100, 100)
		for k := range msgs {
			msgs[k] = strconv.Itoa(k)
		}
		mock.Expect(v.Bot.DG.ChannelMessagesBulkDelete, "a", msgs[:99])
		mock.Expect(v.Bot.DG.ChannelMessagesBulkDelete, "a", msgs[99:])
		v.Bot.DG.BulkDeleteBypass("a", msgs)
		msgs = make([]string, 300, 300)
		for k := range msgs {
			msgs[k] = strconv.Itoa(k)
		}
		mock.Expect(v.Bot.DG.ChannelMessagesBulkDelete, "a", msgs[:99])
		mock.Expect(v.Bot.DG.ChannelMessagesBulkDelete, "a", msgs[99:198])
		mock.Expect(v.Bot.DG.ChannelMessagesBulkDelete, "a", msgs[198:297])
		mock.Expect(v.Bot.DG.ChannelMessagesBulkDelete, "a", msgs[297:])
		v.Bot.DG.BulkDeleteBypass("a", msgs)
	}
}

func TestDiscordChannel(t *testing.T) {
	ch := DiscordChannel("")
	Check(ch.Convert(), uint64(0), t)
	Check(ch.Display(), "<#>", t)
	Check(ch.Equals(""), false, t)
	Check(ch == ChannelEmpty, true, t)
	Check(ch, ChannelEmpty, t)
	Check(ch.String(), "", t)
	CheckNot(ch, NewDiscordChannel(0), t)
	ch = DiscordChannel("1")
	Check(ch.Convert(), uint64(1), t)
	Check(ch.Display(), "<#1>", t)
	Check(ch.Equals("1"), true, t)
	Check(ch != ChannelEmpty, true, t)
	Check(ch.String(), "1", t)
	Check(ch, NewDiscordChannel(1), t)
	Check(NewDiscordChannel(1).String(), "1", t)
	Check(NewDiscordChannel(1).Convert(), uint64(1), t)
	ch = DiscordChannel("a")
	Check(ch.Convert(), uint64(0), t)
	Check(ch.Display(), "<#a>", t)
	Check(ch.Equals("a"), true, t)
	Check(ch != ChannelEmpty, true, t)
	Check(ch.String(), "a", t)
	CheckNot(ch, NewDiscordChannel(0), t)
}
func TestDiscordRole(t *testing.T) {
	r := DiscordRole("")
	Check(r.Convert(), uint64(0), t)
	Check(r.Display(), "<@&>", t)
	Check(r.Equals(""), false, t)
	Check(r == RoleEmpty, true, t)
	Check(r, RoleEmpty, t)
	Check(r.String(), "", t)
	CheckNot(r, NewDiscordRole(0), t)
	r = DiscordRole("1")
	Check(r.Convert(), uint64(1), t)
	Check(r.Display(), "<@&1>", t)
	Check(r.Equals("1"), true, t)
	Check(r != RoleEmpty, true, t)
	Check(r.String(), "1", t)
	Check(r, NewDiscordRole(1), t)
	Check(NewDiscordRole(1).String(), "1", t)
	Check(NewDiscordRole(1).Convert(), uint64(1), t)
	r = DiscordRole("a")
	Check(r.Convert(), uint64(0), t)
	Check(r.Display(), "<@&a>", t)
	Check(r.Equals("a"), true, t)
	Check(r != RoleEmpty, true, t)
	Check(r.String(), "a", t)
	CheckNot(r, NewDiscordRole(0), t)
}
func TestDiscordUser(t *testing.T) {
	u := DiscordUser("")
	Check(u.Convert(), uint64(0), t)
	Check(u.Display(), "<@>", t)
	Check(u.Equals(""), false, t)
	Check(u == UserEmpty, true, t)
	Check(u, UserEmpty, t)
	Check(u.String(), "", t)
	CheckNot(u, NewDiscordUser(0), t)
	u = DiscordUser("1")
	Check(u.Convert(), uint64(1), t)
	Check(u.Display(), "<@1>", t)
	Check(u.Equals("1"), true, t)
	Check(u != UserEmpty, true, t)
	Check(u.String(), "1", t)
	Check(u, NewDiscordUser(1), t)
	Check(NewDiscordUser(1).String(), "1", t)
	Check(NewDiscordUser(1).Convert(), uint64(1), t)
	u = DiscordUser("a")
	Check(u.Convert(), uint64(0), t)
	Check(u.Display(), "<@a>", t)
	Check(u.Equals("a"), true, t)
	Check(u != UserEmpty, true, t)
	Check(u.String(), "a", t)
	CheckNot(u, NewDiscordUser(0), t)
}
func TestDiscordGuild(t *testing.T) {
	g := DiscordGuild("")
	Check(g.Convert(), uint64(0), t)
	Check(g.Equals(""), false, t)
	Check(g == GuildEmpty, true, t)
	Check(g, GuildEmpty, t)
	Check(g.String(), "", t)
	g = DiscordGuild("1")
	Check(g.Convert(), uint64(1), t)
	Check(g.Equals("1"), true, t)
	Check(g != GuildEmpty, true, t)
	Check(g.String(), "1", t)
	Check(g, NewDiscordGuild(1), t)
	Check(NewDiscordGuild(1).String(), "1", t)
	Check(NewDiscordGuild(1).Convert(), uint64(1), t)
	g = DiscordGuild("a")
	Check(g.Convert(), uint64(0), t)
	Check(g.Equals("a"), true, t)
	Check(g != GuildEmpty, true, t)
	Check(g.String(), "a", t)
	CheckNot(g, NewDiscordGuild(0), t)
}

func TestParseChannel(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	nilresults := []string{"", "!!", "akjhdfkj", "akjh dfkj", "<", "<>", "<#>", "<#a>", "#a", "Jail Channe"}
	for _, v := range nilresults {
		ch, err := ParseChannel(v, nil)
		Check(ch, ChannelEmpty, t)
		if v != "" {
			CheckNot(err, nil, t)
		}
	}
	ch, err := ParseChannel("!", nil)
	Check(ch, ChannelExclusion, t)
	Check(err, nil, t)
	ch, err = ParseChannel("0", nil)
	Check(ch, DiscordChannel("0"), t)
	Check(err, nil, t)
	ch, err = ParseChannel("<#0>", nil)
	Check(ch, DiscordChannel("0"), t)
	Check(err, nil, t)

	for _, g := range sb.DG.State.Guilds {
		i := DiscordGuild(g.ID).Convert() & 0xFF
		for _, v := range nilresults {
			ch, err := ParseChannel(v, g)
			Check(ch, ChannelEmpty, t)
			if v != "" {
				CheckNot(err, nil, t)
			}
		}
		ch, err := ParseChannel("!", g)
		Check(ch, ChannelExclusion, t)
		Check(err, nil, t)
		ch, err = ParseChannel("0", g)
		Check(ch, DiscordChannel("0"), t)
		Check(err, nil, t)
		ch, err = ParseChannel("<#0>", g)
		Check(ch, DiscordChannel("0"), t)
		Check(err, nil, t)

		ch, err = ParseChannel(fmt.Sprintf("<#%v>", TestChannel|i), g)
		Check(ch, DiscordChannel(fmt.Sprintf("%v", TestChannel|i)), t)
		Check(err, nil, t)
		ch, err = ParseChannel("Jail Channel", g)
		Check(ch, DiscordChannel(fmt.Sprintf("%v", TestChannelJail|i)), t)
		Check(err, nil, t)
		ch, err = ParseChannel("#Jail Channel", g)
		Check(ch, DiscordChannel(fmt.Sprintf("%v", TestChannelJail|i)), t)
		Check(err, nil, t)
		ch, err = ParseChannel("Test Channel", g)
		Check(ch, ChannelEmpty, t)
		CheckNot(err, nil, t)
	}
}

func TestParseRole(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	nilresults := []string{"", "!!", "akjhdfkj", "akjh dfkj", "<", "<>", "<@>", "<@0>", "<@&>", "<@&a>", "<@a>", "@a", "@&a", "Silent Rol"}
	for _, v := range nilresults {
		r, err := ParseRole(v, nil)
		Check(r, RoleEmpty, t)
		if v != "" {
			CheckNot(err, nil, t)
		}
	}
	r, err := ParseRole("!", nil)
	Check(r, RoleExclusion, t)
	Check(err, nil, t)
	r, err = ParseRole("0", nil)
	Check(r, DiscordRole("0"), t)
	Check(err, nil, t)
	r, err = ParseRole("<@&0>", nil)
	Check(r, DiscordRole("0"), t)
	Check(err, nil, t)

	for _, g := range sb.DG.State.Guilds {
		i := DiscordGuild(g.ID).Convert() & 0xFF
		for _, v := range nilresults {
			r, err := ParseRole(v, g)
			Check(r, RoleEmpty, t)
			if v != "" {
				CheckNot(err, nil, t)
			}
		}
		r, err := ParseRole("!", g)
		Check(r, RoleExclusion, t)
		Check(err, nil, t)
		r, err = ParseRole("0", g)
		Check(r, DiscordRole("0"), t)
		Check(err, nil, t)
		r, err = ParseRole("<@&0>", g)
		Check(r, DiscordRole("0"), t)
		Check(err, nil, t)
		r, err = ParseRole(fmt.Sprintf("<@&%v>", TestRoleAdmin|i), g)
		Check(r, DiscordRole(fmt.Sprintf("%v", TestRoleAdmin|i)), t)
		Check(err, nil, t)
		r, err = ParseRole("Silent Role", g)
		Check(r, DiscordRole(fmt.Sprintf("%v", TestRoleSilence|i)), t)
		Check(err, nil, t)
		r, err = ParseRole("@Silent Role", g)
		Check(r, DiscordRole(fmt.Sprintf("%v", TestRoleSilence|i)), t)
		Check(err, nil, t)
		r, err = ParseRole("User Assignable", g)
		Check(r, RoleEmpty, t)
		CheckNot(err, nil, t)
	}
}

func TestParseUser(t *testing.T) {
	sb, dbmock, _ := MockSweetieBot(t)

	nilresults := []string{"", "!!", "akjhdfkj", "akjh dfkj", "<", "<>", "<@>", "<@a>", "@a"}
	for _, v := range nilresults {
		u, err := ParseUser(v, nil)
		Check(u, UserEmpty, t)
		CheckNot(err, nil, t)
	}
	u, err := ParseUser("0", nil)
	Check(u, DiscordUser("0"), t)
	Check(err, nil, t)
	u, err = ParseUser("<@0>", nil)
	Check(u, DiscordUser("0"), t)
	Check(err, nil, t)

	for k, g := range sb.Guilds {
		i := k.Convert() & 0xFF
		for _, v := range nilresults {
			if len(v) > 0 && v[0] != '<' {
				v2 := "%" + v + "%"
				dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), v, v, 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
				dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), v2, v2, 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
				if v[0] == '@' {
					v2 := "%" + v[1:] + "%"
					dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), v[1:], v[1:], 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
					dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), v2, v2, 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
				}
			}
			u, err := ParseUser(v, g)
			Check(u, UserEmpty, t)
			CheckNot(err, nil, t)
		}
		dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), "0", "0", 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
		dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), "%0%", "%0%", 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
		u, err = ParseUser("0", g)
		Check(u, UserEmpty, t)
		CheckNot(err, nil, t)
		u, err = ParseUser("<@0>", g)
		Check(u, DiscordUser("0"), t)
		Check(err, nil, t)
		u, err = ParseUser(fmt.Sprintf("<@%v>", TestUserAssigned|i), g)
		Check(u, DiscordUser(fmt.Sprintf("%v", TestUserAssigned|i)), t)
		Check(err, nil, t)
		dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), "boring user", "boring user", 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(TestUserBoring | i))
		u, err = ParseUser("Boring User", g)
		Check(u, DiscordUser(fmt.Sprintf("%v", TestUserBoring|i)), t)
		Check(err, nil, t)
		dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), "@boring user", "@boring user", 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
		dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), "%@boring user%", "%@boring user%", 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
		dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), "boring user", "boring user", 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(TestUserBoring | i))
		u, err = ParseUser("@Boring User", g)
		Check(u, DiscordUser(fmt.Sprintf("%v", TestUserBoring|i)), t)
		Check(err, nil, t)
		dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(k.Convert(), "user", "user", 20, 0).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(TestUserBoring | i).AddRow(TestUserBot | i).AddRow(TestUserAssigned | i))
		u, err = ParseUser("User", g)
		Check(u, UserEmpty, t)
		CheckNot(err, nil, t)
	}
}
