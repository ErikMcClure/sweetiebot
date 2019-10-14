package sweetiebot

import (
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/blackhole12/discordgo"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func Check(result interface{}, expected interface{}, t *testing.T) bool {
	if result != expected {
		_, fn, line, _ := runtime.Caller(1)
		fmt.Printf("[%s:%v] Expected %v but got %v\n", filepath.Base(fn), line, expected, result)
		t.Fail()
		return false
	}
	return true
}
func CheckNot(result interface{}, expected interface{}, t *testing.T) bool {
	if result == expected {
		_, fn, line, _ := runtime.Caller(1)
		fmt.Printf("[%s:%v] Unexpected result: %v\n", filepath.Base(fn), line, result)
		t.Fail()
		return false
	}
	return true
}

var mock *Mock

const (
	MaxServers         = 10
	_                  = iota
	TestSelfID         = 12345727
	TestOwnerBot       = (iota << MaxServers) - 1
	TestOwnerServer    = iota << MaxServers
	TestAdminMod       = iota << MaxServers
	TestAdmin          = iota << MaxServers
	TestMod            = iota << MaxServers
	TestUserAssigned   = iota << MaxServers
	TestUserNonAssign  = iota << MaxServers
	TestUserBoring     = iota << MaxServers
	TestUserSilence    = iota << MaxServers
	TestUserBot        = iota << MaxServers
	TestUserNew        = iota << MaxServers
	TestRoleAdmin      = iota << MaxServers
	TestRoleMod        = iota << MaxServers
	TestRoleUser       = iota << MaxServers
	TestRoleMember     = iota << MaxServers
	TestRoleAssign     = iota << MaxServers
	TestRoleAssign2    = iota << MaxServers
	TestRoleSilence    = iota << MaxServers
	TestServer         = iota << MaxServers
	TestChannel        = iota << MaxServers
	TestChannel2       = iota << MaxServers
	TestChannelSpoil   = iota << MaxServers
	TestChannelFree    = iota << MaxServers
	TestChannelLog     = iota << MaxServers
	TestChannelMod     = iota << MaxServers
	TestChannelBored   = iota << MaxServers
	TestChannelJail    = iota << MaxServers
	TestChannelWelcome = iota << MaxServers
	TestChannelPrivate = iota << MaxServers
	TestChannelGroupDM = iota << MaxServers
	NumServers         = 3
)

func mockDiscordRole(role int, index int) *discordgo.Role {
	name := "testrole"
	perms := 0
	switch role {
	case TestRoleAdmin:
		name = "Admin Role"
		perms = discordgo.PermissionAdministrator
	case TestRoleMod:
		name = "Mod Role"
		perms = discordgo.PermissionAllText | discordgo.PermissionManageRoles | discordgo.PermissionManageMessages | discordgo.PermissionManageChannels
	case TestRoleUser:
		name = "User Role"
		perms = discordgo.PermissionSendMessages | discordgo.PermissionReadMessages | discordgo.PermissionReadMessageHistory | discordgo.PermissionSendTTSMessages
	case TestRoleMember:
		name = "Member Role"
		perms = discordgo.PermissionSendMessages | discordgo.PermissionReadMessages
	case TestRoleAssign2:
		fallthrough
	case TestRoleAssign:
		name = "User Assignable"
		perms = discordgo.PermissionSendMessages | discordgo.PermissionReadMessages | discordgo.PermissionReadMessageHistory | discordgo.PermissionSendTTSMessages
	case TestRoleSilence:
		name = "Silent Role"
		perms = 0
	}

	return &discordgo.Role{
		ID:          strconv.Itoa(role | index),
		Name:        name,
		Mentionable: true,
		Hoist:       true,
		Color:       role,
		Position:    0,
		Permissions: perms,
	}
}

func mockDiscordMember(member int, index int) *discordgo.Member {
	name := "member"
	roles := []string{}

	switch member {
	case TestOwnerBot:
		name = "Bot Owner"
	case TestOwnerServer:
		name = "Server Owner"
	case TestAdminMod:
		name = "Admin/Mod User"
		roles = append(roles, strconv.Itoa(TestRoleMod|index), strconv.Itoa(TestRoleAdmin|index))
	case TestAdmin:
		name = "Admin User"
		roles = append(roles, strconv.Itoa(TestRoleAdmin|index), strconv.Itoa(TestRoleMember|index))
	case TestMod:
		name = "Mod User"
		roles = append(roles, strconv.Itoa(TestRoleMod|index), strconv.Itoa(TestRoleMember|index))
	case TestUserNonAssign:
		name = "User With Non user-assignable Role"
		roles = append(roles, strconv.Itoa(TestRoleUser|index), strconv.Itoa(TestRoleMember|index))
	case TestUserAssigned:
		name = "User With user-assignable Role"
		roles = append(roles, strconv.Itoa(TestRoleAssign|index), strconv.Itoa(TestRoleMember|index))
	case TestUserBoring:
		name = "Boring User"
		roles = append(roles, strconv.Itoa(TestRoleMember|index))
	case TestUserSilence:
		name = "Silenced User"
		roles = append(roles, strconv.Itoa(TestRoleSilence|index))
	case TestUserNew:
		name = "New User"
	case TestUserBot:
		name = "Bot User"
	}

	return &discordgo.Member{
		GuildID: strconv.Itoa(TestServer | index),
		Nick:    name + ":" + strconv.Itoa(TestServer|index),
		User: &discordgo.User{
			ID:            strconv.Itoa(member | index),
			Email:         name + "@fake.com",
			Username:      name,
			Discriminator: strconv.Itoa(1000 + index),
			Bot:           member == TestUserBot,
		},
		Roles: roles,
	}
}

func mockDiscordChannel(channel int, index int) *discordgo.Channel {
	name := "Test"
	perms := []*discordgo.PermissionOverwrite{}

	disallowEveryone := &discordgo.PermissionOverwrite{
		ID:   strconv.Itoa(TestServer | index),
		Type: "role",
		Deny: discordgo.PermissionAllText,
	}
	allowMods := &discordgo.PermissionOverwrite{
		ID:    strconv.Itoa(TestRoleMod | index),
		Type:  "role",
		Allow: discordgo.PermissionAllText,
	}
	allowSilence := &discordgo.PermissionOverwrite{
		ID:    strconv.Itoa(TestRoleSilence | index),
		Type:  "role",
		Allow: discordgo.PermissionReadMessageHistory | discordgo.PermissionReadMessages | discordgo.PermissionSendMessages,
	}
	disallowSilence := &discordgo.PermissionOverwrite{
		ID:   strconv.Itoa(TestRoleSilence | index),
		Type: "role",
		Deny: discordgo.PermissionAllText,
	}

	switch channel {
	case TestChannel2:
		fallthrough
	case TestChannel:
		name = "Test Channel"
		perms = append(perms, disallowSilence)
	case TestChannelSpoil:
		name = "Spoiler Channel"
		perms = append(perms, disallowSilence)
	case TestChannelFree:
		name = "Free Channel"
		perms = append(perms, disallowSilence)
	case TestChannelLog:
		name = "Log Channel"
		perms = append(perms, disallowEveryone, allowMods)
	case TestChannelMod:
		name = "Mod Channel"
		perms = append(perms, disallowEveryone, allowMods)
	case TestChannelBored:
		name = "Bored Channel"
		perms = append(perms, disallowSilence)
	case TestChannelJail:
		name = "Jail Channel"
		perms = append(perms, disallowEveryone, allowSilence, allowMods)
	case TestChannelWelcome:
		name = "Welcome Channel"
		perms = append(perms, disallowEveryone, allowSilence, allowMods)
	}
	return &discordgo.Channel{
		ID:                   strconv.Itoa(channel | index),
		GuildID:              strconv.Itoa(TestServer | index),
		Name:                 name,
		Topic:                "Fake topic",
		Type:                 discordgo.ChannelTypeGuildText,
		NSFW:                 false,
		PermissionOverwrites: perms,
	}
}

func mockDiscordGuild(index int) *discordgo.Guild {
	return &discordgo.Guild{
		ID:                strconv.Itoa(TestServer | index),
		Name:              "Test Server " + strconv.Itoa(index),
		OwnerID:           strconv.Itoa(TestOwnerServer | index),
		VerificationLevel: discordgo.VerificationLevelLow,
		Large:             false,
		Unavailable:       false,
		Roles: []*discordgo.Role{
			mockDiscordRole(TestRoleAdmin, index),
			mockDiscordRole(TestRoleMod, index),
			mockDiscordRole(TestRoleUser, index),
			mockDiscordRole(TestRoleMember, index),
			mockDiscordRole(TestRoleAssign, index),
			mockDiscordRole(TestRoleAssign2, index),
			mockDiscordRole(TestRoleSilence, index),
		},
		Emojis: []*discordgo.Emoji{},
		Members: []*discordgo.Member{
			mockDiscordMember(TestOwnerBot, 0),
			mockDiscordMember(TestOwnerServer, index),
			mockDiscordMember(TestAdminMod, index),
			mockDiscordMember(TestAdmin, index),
			mockDiscordMember(TestMod, index),
			mockDiscordMember(TestUserAssigned, index),
			mockDiscordMember(TestUserNonAssign, index),
			mockDiscordMember(TestUserBoring, index),
			mockDiscordMember(TestUserSilence, index),
			mockDiscordMember(TestUserBot, index),
		},
		Presences: []*discordgo.Presence{},
		Channels: []*discordgo.Channel{
			mockDiscordChannel(TestChannel, index),
			mockDiscordChannel(TestChannel2, index),
			mockDiscordChannel(TestChannelSpoil, index),
			mockDiscordChannel(TestChannelFree, index),
			mockDiscordChannel(TestChannelLog, index),
			mockDiscordChannel(TestChannelMod, index),
			mockDiscordChannel(TestChannelBored, index),
			mockDiscordChannel(TestChannelJail, index),
			mockDiscordChannel(TestChannelWelcome, index),
		},
		VoiceStates: []*discordgo.VoiceState{},
	}
}

// Generate fake discordgo session
func mockDiscordGo() *DiscordGoSession {
	dg, _ := discordgo.New()
	s := &DiscordGoSession{*dg}
	for i := 0; i < NumServers; i++ {
		s.State.GuildAdd(mockDiscordGuild(i))
	}

	return s
}

func mockBotDB() (*BotDB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}

	botdb := &BotDB{
		db:          db,
		lastattempt: time.Now().UTC(),
		log:         &emptyLog{},
		driver:      "mysql",
		conn:        "",
	}
	for i := 0; i < 67; i++ {
		mock.ExpectPrepare(".*")
	}
	botdb.Status.Set(botdb.LoadStatements() == nil)
	return botdb, mock
}

type CommandMocker struct {
	CommandInfo
}

func (c *CommandMocker) Info() *CommandInfo {
	return &c.CommandInfo
}
func (c *CommandMocker) Process([]string, *discordgo.Message, []int, *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	return "", false, nil
}
func (c *CommandMocker) Usage(*GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: c.Name,
	}
}

func mockCommand(name string) Command {
	return &CommandMocker{CommandInfo{
		Name: name,
	}}
}

func mockCommandFull(info CommandInfo) Command {
	return &CommandMocker{info}
}

type ModuleMocker struct {
	Store string
}

func (m *ModuleMocker) Name() string {
	return m.Store
}
func (m *ModuleMocker) Commands() []Command {
	return []Command{}
}
func (m *ModuleMocker) Description(info *bot.GuildInfo) string {
	return m.Store
}

func mockModule(name string) Module {
	return &ModuleMocker{name}
}

func MockSweetieBot(t *testing.T) (*SweetieBot, sqlmock.Sqlmock, *Mock) {
	db, dbmock := mockBotDB()
	sb := &SweetieBot{
		Owner:          NewDiscordUser(TestOwnerBot),
		DG:             mockDiscordGo(),
		DB:             db,
		SelfName:       "Sweetie Bot",
		SelfID:         NewDiscordUser(TestSelfID),
		AppName:        "Sweetie Bot",
		MainGuildID:    NewDiscordGuild(TestServer | 0),
		DebugChannels:  make(map[DiscordGuild]DiscordChannel),
		Guilds:         make(map[DiscordGuild]*GuildInfo),
		MaxConfigSize:  1000000,
		MaxUniqueItems: 25000,
		StartTime:      time.Now().UTC().Unix(),
		heartbeat:      4294967290,
		memberChan:     make(chan *GuildInfo, 1500),
		Selfhoster:     &Selfhost{SelfhostBase{BotVersion.Integer()}, AtomicBool{0}, sync.Map{}},
	}
	sb.EmptyGuild = NewGuildInfo(sb, &discordgo.Guild{})
	sb.EmptyGuild.Config.FillConfig()
	sb.EmptyGuild.Config.SetupDone = true

	mock = NewMock(t)

	for _, guild := range sb.DG.State.Guilds {
		info := &GuildInfo{
			ID:           guild.ID,
			Name:         guild.Name,
			OwnerID:      DiscordUser(guild.OwnerID),
			commandLast:  make(map[string]int64),
			commandlimit: &SaturationLimit{[]int64{}, 0, AtomicFlag{0}},
			commands:     make(map[CommandID]Command),
			commandmap:   make(map[CommandID]ModuleID),
			lastlogerr:   0,
			Bot:          sb,
			Config:       *DefaultConfig(),
		}
		info.Config.FillConfig()
		id := DiscordGuild(guild.ID)
		sb.Guilds[id] = info
		args := make([]driver.Value, 3*len(guild.Members), 3*len(guild.Members))
		for k := range args {
			args[k] = sqlmock.AnyArg()
		}
		dbmock.ExpectExec("INSERT IGNORE INTO users.*").WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 0))

		args = make([]driver.Value, 4*len(guild.Members), 4*len(guild.Members))
		for k := range args {
			args[k] = sqlmock.AnyArg()
		}
		dbmock.ExpectExec("INSERT IGNORE INTO members.*").WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 0))
		info.ProcessGuild(guild)
		sb.Selfhoster.CheckGuilds(map[DiscordGuild]*GuildInfo{id: info})
		i := id.Convert() & ((1 << MaxServers) - 1)
		info.Config.Modules.Channels["bored"] = map[DiscordChannel]bool{NewDiscordChannel(TestChannelBored | i): true}
		info.Config.Log.Channel = NewDiscordChannel(TestChannelLog | i)
		info.Config.Basic.ModRole = NewDiscordRole(TestRoleMod | i)
		info.Config.Basic.BotChannel = NewDiscordChannel(TestChannelFree | i)
		info.Config.Basic.FreeChannels[NewDiscordChannel(TestChannelFree|i)] = true
		info.Config.Basic.ModChannel = NewDiscordChannel(TestChannelMod | i)
		info.Config.Basic.SilenceRole = NewDiscordRole(TestRoleSilence | i)
		info.Config.Basic.MemberRole = NewDiscordRole(TestRoleMember | i)
		info.Config.Users.JailChannel = NewDiscordChannel(TestChannelJail | i)
		info.Config.Users.WelcomeChannel = NewDiscordChannel(TestChannelWelcome | i)
		info.Config.SetupDone = true
	}
	sb.Guilds[NewDiscordGuild(uint64(TestServer|2))].Silver.Set(true)
	//sb.Guilds[NewDiscordGuild(uint64(TestServer|2))].Config.Basic.MemberRole = RoleEmpty

	return sb, dbmock, mock
}

func MockMessage(content string, channel int, t int64, user int, index int) *discordgo.Message {
	return &discordgo.Message{
		ID:        "123456789",
		ChannelID: strconv.Itoa(channel | index),
		Content:   content,
		Timestamp: discordgo.Timestamp(time.Unix(t, 0).Format(time.RFC3339)),
		Author:    mockDiscordMember(user, index).User,
	}
}

func TestProcessCommand(t *testing.T) {
	sb, dbmock, _ := MockSweetieBot(t)

	dbmock.ExpectQuery("SELECT .* FROM users.*").WithArgs(TestUserBoring).WillReturnRows(sqlmock.NewRows([]string{}))
	dbmock.ExpectQuery("SELECT Guild FROM members*").WithArgs(TestUserBoring).WillReturnRows(sqlmock.NewRows([]string{"Guild"}).AddRow(TestServer))
	mock.Expect(sb.DG.RequestWithLockedBucket, "POST", MockAny{}, "application/json", MockAny{}, MockAny{}, 0)
	sb.ProcessCommand(MockMessage("!about", TestChannelPrivate, 100000, TestUserBoring, 0), nil, 100000, false, false)

	fnguild := func(info *GuildInfo) {
		modules := []Module{&InfoModule{}}
		for _, v := range modules {
			info.RegisterModule(v)
			for _, command := range v.Commands() {
				info.AddCommand(command, v)
			}
		}
	}
	fnguild(sb.EmptyGuild)
	for _, v := range sb.Guilds {
		fnguild(v)
	}

	Check(mock.Check(), true, t)
	dbmock.ExpectQuery("SELECT .* FROM users.*").WillReturnRows(sqlmock.NewRows([]string{"ID", "Username", "Discriminator", "LastSeen", "Location", "DefaultServer"}).AddRow(0, "", 0, time.Now(), "", TestServer))
	dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannelPrivate), MockAny{})
	sb.ProcessCommand(MockMessage("!about", TestChannelPrivate, 1000, TestUserBoring, 0), nil, 1000, false, false)
	Check(mock.Check(), true, t)
	dbmock.ExpectQuery("SELECT .* FROM users.*").WillReturnRows(sqlmock.NewRows([]string{}))
	dbmock.ExpectQuery("SELECT Guild FROM members*").WithArgs(TestUserBoring).WillReturnRows(sqlmock.NewRows([]string{"Guild"}).AddRow(TestServer))
	dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannelPrivate), MockAny{})
	sb.ProcessCommand(MockMessage("!about", TestChannelPrivate, 20001, TestUserBoring, 0), nil, 20001, false, true)
	Check(mock.Check(), true, t)
	dbmock.ExpectQuery("SELECT .* FROM users.*").WillReturnRows(sqlmock.NewRows([]string{}))
	dbmock.ExpectQuery("SELECT Guild FROM members*").WithArgs(TestUserBoring).WillReturnRows(sqlmock.NewRows([]string{"Guild"}))
	dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannelPrivate), MockAny{})
	sb.ProcessCommand(MockMessage("!about", TestChannelPrivate, 30001, TestUserBoring, 0), nil, 30001, false, true)
	Check(mock.Check(), true, t)

	dbmock.ExpectQuery("SELECT .* FROM users.*").WillReturnRows(sqlmock.NewRows([]string{}))
	dbmock.ExpectQuery("SELECT Guild FROM members*").WithArgs(0).WillReturnRows(sqlmock.NewRows([]string{"Guild"}))
	dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannelPrivate), MockAny{})
	sb.ProcessCommand(MockMessage("!about", TestChannelPrivate, 40002, 0, 0), nil, 40002, false, false)
	dbmock.ExpectQuery("SELECT .* FROM users.*").WillReturnRows(sqlmock.NewRows([]string{}))
	dbmock.ExpectQuery("SELECT Guild FROM members*").WithArgs(0).WillReturnRows(sqlmock.NewRows([]string{"Guild"}))
	dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannelPrivate), MockAny{})
	sb.ProcessCommand(MockMessage("!about", TestChannelPrivate, 50003, 0, 0), nil, 50003, false, true)
	dbmock.ExpectQuery("SELECT .* FROM users.*").WillReturnRows(sqlmock.NewRows([]string{}))
	dbmock.ExpectQuery("SELECT Guild FROM members*").WithArgs(TestUserBoring).WillReturnRows(sqlmock.NewRows([]string{"Guild"}))
	dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannelPrivate), MockAny{})
	sb.ProcessCommand(MockMessage("!about", TestChannelPrivate, 60004, TestUserBoring, 0), nil, 60004, true, true)
	dbmock.ExpectQuery("SELECT .* FROM users.*").WillReturnRows(sqlmock.NewRows([]string{}))
	dbmock.ExpectQuery("SELECT Guild FROM members*").WithArgs(TestUserBoring).WillReturnRows(sqlmock.NewRows([]string{"Guild"}))
	dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannelPrivate), MockAny{})
	sb.ProcessCommand(MockMessage("!about", TestChannelPrivate, 70005, TestUserBoring, 0), nil, 70005, true, true)
	dbmock.ExpectQuery("SELECT .* FROM users.*").WillReturnRows(sqlmock.NewRows([]string{}))
	dbmock.ExpectQuery("SELECT Guild FROM members*").WithArgs(TestUserBoring).WillReturnRows(sqlmock.NewRows([]string{"Guild"}))
	mock.Expect(sb.DG.ChannelMessageSend, strconv.Itoa(TestChannelPrivate), MockAny{})
	sb.ProcessCommand(MockMessage("!asdf", TestChannelPrivate, 80005, TestUserBoring, 0), nil, 80005, true, true)
	sb.ProcessCommand(MockMessage("", TestChannelPrivate, 90000, TestUserBoring, 0), nil, 90000, false, false)
	sb.ProcessCommand(MockMessage("!", TestChannelPrivate, 99000, TestUserBoring, 0), nil, 99000, false, false)
	sb.ProcessCommand(MockMessage("!!", TestChannelPrivate, 99900, TestUserBoring, 0), nil, 99900, false, false)
	sb.ProcessCommand(MockMessage("~", TestChannelPrivate, 99990, TestUserBoring, 0), nil, 99990, false, false)
	sb.ProcessCommand(MockMessage("about", TestChannelPrivate, 99999, TestUserBoring, 0), nil, 99999, false, false)

	//for k, v := range sb.Guilds {
	//	i := int(k.Convert() & 0xFF)
	for i := 0; i < 3; i++ {
		v := sb.Guilds[NewDiscordGuild(uint64(i|TestServer))]
		fmt.Println(i)
		Check(mock.Check(), true, t)
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		sb.ProcessCommand(MockMessage("!about", TestChannel, 1000000, TestUserBoring, i), v, 1000000, false, false)
		v.Config.Basic.CommandPrefix = ""
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		sb.ProcessCommand(MockMessage("!about", TestChannel, 1000000, TestUserBoring, i), v, 1000000, false, false)
		v.Config.Basic.CommandPrefix = "asdf"
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		sb.ProcessCommand(MockMessage("!about", TestChannel, 1000000, TestUserBoring, i), v, 1000000, false, false)
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.Expect(sb.DG.RequestWithLockedBucket, "POST", MockAny{}, "application/json", MockAny{}, MockAny{}, 0)
		sb.ProcessCommand(MockMessage("!about", TestChannel, 1000000, TestUserBoring, i), v, 1000000, false, false)
		Check(mock.Check(), true, t) // Check that the command saturation works
		v.Config.Basic.CommandPrefix = "~"
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		sb.ProcessCommand(MockMessage("~about", TestChannel, 2000000, TestUserBoring, i), v, 2000000, false, false)
		Check(mock.Check(), true, t)
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3000000, TestUserBoring, i), v, 3000000, false, false)
		Check(mock.Check(), true, t)
		v.Config.Basic.IgnoreInvalidCommands = true
		sb.ProcessCommand(MockMessage("~asdf", TestChannel, 3001000, TestUserBoring, i), v, 3001000, false, false)
		Check(mock.Check(), true, t)
		v.Config.Basic.CommandPrefix = "!"
		v.Config.Basic.IgnoreInvalidCommands = false
		mock.Expect(sb.DG.GuildMember, strconv.Itoa(TestServer|i), strconv.Itoa(TestSelfID))
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3004000, TestSelfID, i), v, 3004000, false, false)

		v.Config.Modules.CommandDisabled["about"] = true
		mock.Expect(sb.DG.GuildMember, strconv.Itoa(TestServer|i), strconv.Itoa(TestSelfID))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3008000, TestSelfID, i), v, 3008000, false, false)

		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3100000, TestUserBoring, i), v, 3100000, false, false)
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3200000, TestMod, i), v, 3200000, false, false)
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3300000, TestAdmin, i), v, 3300000, false, false)
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3400000, TestOwnerBot, i), v, 3400000, false, false)
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3500000, TestOwnerServer, i), v, 3500000, false, false)

		delete(v.Config.Modules.CommandDisabled, "about")
		v.Config.Modules.CommandChannels["about"] = map[DiscordChannel]bool{DiscordChannel("0"): true}

		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3600000, TestUserBoring, i), v, 3600000, false, false)
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3700000, TestMod, i), v, 3700000, false, false)
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3800000, TestAdmin, i), v, 3800000, false, false)

		v.Config.Modules.CommandChannels["about"] = map[DiscordChannel]bool{DiscordChannel("0"): true, ChannelExclusion: true}
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3800000, TestUserBoring, i), v, 3800000, false, false)
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3900000, TestMod, i), v, 3900000, false, false)
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3800000, TestAdmin, i), v, 3800000, false, false)

		v.Config.Modules.CommandLimits["about"] = 30
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3900000, TestUserBoring, i), v, 3900000, false, false)
		mock.Expect(sb.DG.RequestWithLockedBucket, "POST", MockAny{}, "application/json", MockAny{}, MockAny{}, 0)
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3900000, TestUserBoring, i), v, 3900000, false, false)
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3900000, TestUserBoring, i), v, 3900000, false, false)
		mock.Expect(sb.DG.RequestWithLockedBucket, "POST", MockAny{}, "application/json", MockAny{}, MockAny{}, 0)
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3900006, TestUserBoring, i), v, 3900006, false, false)
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3900036, TestUserBoring, i), v, 3900036, false, false)

		v.Config.Basic.FreeChannels[NewDiscordChannel(uint64(TestChannel|i))] = true
		for k := 0; k < 6; k++ {
			mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
			dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
			sb.ProcessCommand(MockMessage("!about", TestChannel, 3900036, TestUserBoring, i), v, 3900036, false, false)
		}

		delete(v.Config.Modules.CommandLimits, "about")
		v.Config.Basic.Aliases["about"] = "rules"

		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 3900036, TestUserBoring, i), v, 3900036, false, false)

		v.Config.Basic.Aliases["aaaaaa"] = "about"
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!AAAAAA", TestChannel, 4000000, TestUserBoring, i), v, 4000000, false, false)

		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestChannel|i), MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!AAAAAA", TestChannel, 4001000, TestUserBoring, i), v, 4001000, false, false)

		mock.Expect(sb.DG.RequestWithLockedBucket, "POST", MockAny{}, "application/json", MockAny{}, MockAny{}, 0)
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!help asdf", TestChannel, 3900036, TestUserBoring, i), v, 3900036, false, false)

		mock.Expect(sb.DG.GuildMember, v.ID, strconv.Itoa(TestUserNew))
		mock.Expect(sb.DG.GuildMember, v.ID, strconv.Itoa(TestUserNew))
		mock.Expect(sb.DG.GuildMember, v.ID, strconv.Itoa(TestUserNew))
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!about", TestChannel, 6005000, TestUserNew, 0), v, 6005000, false, false)

		mock.Expect(sb.DG.UserChannelCreate, strconv.Itoa(TestUserBoring|i))
		mock.Expect(sb.DG.RequestWithLockedBucket, "POST", MockAny{}, "application/json", MockAny{}, MockAny{}, 0)
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestUserBoring|i)+"10", MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!HELP help", TestChannel, 3900036, TestUserBoring, i), v, 3900036, false, false)

		v.Config.Basic.Aliases["ffff"] = "help"

		mock.Expect(sb.DG.UserChannelCreate, strconv.Itoa(TestUserBoring|i))
		mock.Expect(sb.DG.RequestWithLockedBucket, "POST", MockAny{}, "application/json", MockAny{}, MockAny{}, 0)
		mock.Expect(sb.DG.ChannelMessageSendEmbed, strconv.Itoa(TestUserBoring|i)+"10", MockAny{})
		dbmock.ExpectExec("INSERT INTO debuglog .*").WillReturnResult(sqlmock.NewResult(1, 1))
		sb.ProcessCommand(MockMessage("!FfFf help", TestChannel, 3904036, TestUserBoring, i), v, 3904036, false, false)
	}
}

func TestChannelIsPrivate(t *testing.T) {
	//sb, dbmock := MockSweetieBot(t)
}

func TestGetChannelGuild(t *testing.T) {

}

func TestGetGuildFromID(t *testing.T) {

}

func TestFindServers(t *testing.T) {

}

func TestGetDefaultServer(t *testing.T) {

}

// Feeds a set of problematic strings into a command to ensure no input can crash the bot.
func CommandFuzzer(command Command, info *GuildInfo, t *testing.T) {
	i := int(SBatoi(info.ID) & 0xFF)
	strings := []string{"0", "", "-1", "8446744073709551615", "null", "<@0>", "<@!0>", "<@&0>", "<#0>", "@everyone", "true", "false", "about", "help", "!", "-", "#$*&%^#&", "\"", " ",
		fmt.Sprintf("<#%v>", TestChannel|i), fmt.Sprintf("<#%v>", TestChannelLog|i), fmt.Sprintf("<@!%v>", TestOwnerServer|i),
		fmt.Sprintf("<@%v>", TestSelfID), fmt.Sprintf("<@&%v>", TestRoleAdmin|i)}
	strings2 := append(strings, strings...) // Duplicate strings so we can take slices of it all the way to the end

	for k := range strings {
		content := ""
		indices := []int{}
		for j := 0; j < 10; j++ {
			indices = append(indices, len(content))
			content += strings2[k+j] + " "
		}

		command.Process(strings2[k:k+10], MockMessage(content, TestChannel, 0, TestSelfID, i), indices, info)
		command.Process(strings2[k:k+10],
			&discordgo.Message{
				ID:        "123456789",
				ChannelID: "heartbeat",
				Content:   content,
				Timestamp: discordgo.Timestamp(time.Unix(1, 0).Format(time.RFC3339)),
				Author:    mockDiscordMember(TestSelfID, 0).User,
			}, indices, info)
		command.Process(strings2[k:k+10],
			&discordgo.Message{
				ID:        "123456789",
				ChannelID: "heartbeat",
				Content:   content,
				Timestamp: discordgo.Timestamp(time.Unix(1, 0).Format(time.RFC3339)),
				Author:    mockDiscordMember(TestUserBoring, i).User,
			}, indices, info)
		command.Process(strings2[k:k+10],
			&discordgo.Message{
				ID:        "123456789",
				ChannelID: strconv.Itoa(TestChannelPrivate),
				Content:   content,
				Timestamp: discordgo.Timestamp(time.Unix(1, 0).Format(time.RFC3339)),
				Author:    mockDiscordMember(TestUserBoring, i).User,
			}, indices, info)

		channels := []int{TestChannel, TestChannelGroupDM, TestChannelFree, TestChannelLog, TestChannelMod}
		users := []int{TestAdmin, TestOwnerBot, TestOwnerServer, TestMod, TestAdminMod, TestUserBoring, TestUserBot, TestUserSilence}

		for t, channel := range channels {
			for _, user := range users {
				command.Process(strings2[k:k+10], MockMessage(content, channel, int64(t), user, i), indices, info)
			}
		}
	}
}
