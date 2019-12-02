package sweetiebot

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/erikmcclure/discordgo"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestAddCommand(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	info := sb.Guilds[NewDiscordGuild(TestServer)]
	Check(len(info.commands), 0, t)

	info.AddCommand(mockCommand("TEST"), mockModule("TESTMODULE"))
	if len(info.commands) < 1 || info.commands["test"].Info().Name != "TEST" {
		t.Errorf("info.commands state is invalid: %v", info.commands)
	}

	info.commands = make(map[CommandID]Command)
	info.AddCommand(mockCommand("test"), mockModule("testmodule"))
	if len(info.commands) < 1 || info.commands["test"].Info().Name != "test" {
		t.Errorf("info.commands state is invalid: %v", info.commands)
	}
}

func TestSaveConfig(t *testing.T) {
	sb, dbmock, _ := MockSweetieBot(t)
	for k, v := range sb.Guilds {
		Check(v.SaveConfig(), nil, t)
		if _, err := os.Stat(fmt.Sprintf("%v.json", k)); err != nil {
			t.Errorf("%v.json does not exist", k)
		}
	}

	for k := range sb.Guilds {
		os.Remove(fmt.Sprintf("%v.json", k))
	}
	sb.MaxConfigSize = 1
	for k, v := range sb.Guilds {
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", MockAny{}, MockAny{}, MockAny{}, MockAny{}, 0)
		dbmock.ExpectExec("INSERT INTO debuglog.*").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
		Check(v.SaveConfig(), errConfigFileTooLarge, t)
		if _, err := os.Stat(fmt.Sprintf("%v.json", k)); err == nil {
			t.Errorf("%v.json exists", k)
		}
	}

	for k := range sb.Guilds {
		os.Remove(fmt.Sprintf("%v.json", k))
	}
}

func MockMessageEmbed(start int, end int) *discordgo.MessageEmbed {
	msg := &discordgo.MessageEmbed{
		Fields: make([]*discordgo.MessageEmbedField, (end-start)+1, (end-start)+1),
	}
	for i := start; i <= end; i++ {
		msg.Fields[i-start] = &discordgo.MessageEmbedField{
			Name: strconv.Itoa(i),
		}
	}
	return msg
}

func TestSendEmbed(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	msg1 := MockMessageEmbed(1, 1)

	Check(sb.Guilds[NewDiscordGuild(TestServer)].SendEmbed(NewDiscordChannel(uint64(TestChannelFree|1)), msg1), errInvalidChannel, t)
	for k, v := range sb.Guilds {
		mock.Expect(v.Bot.DG.ChannelMessageSendEmbed, NewDiscordChannel(TestChannelPrivate).String(), MockMessageEmbed(1, 1))
		v.SendEmbed(NewDiscordChannel(TestChannelPrivate), msg1)
		h := v.Bot.heartbeat
		Check(v.SendEmbed("heartbeat", msg1), nil, t)
		Check(v.Bot.heartbeat, h+1, t)
		ch := NewDiscordChannel(TestChannel | (k.Convert() & 0xFF))
		mock.Expect(v.Bot.DG.ChannelMessageSendEmbed, ch.String(), MockMessageEmbed(1, 25))
		v.SendEmbed(ch, MockMessageEmbed(1, 25))
		mock.Expect(v.Bot.DG.ChannelMessageSendEmbed, ch.String(), MockMessageEmbed(1, 25))
		mock.Expect(v.Bot.DG.ChannelMessageSendEmbed, ch.String(), MockMessageEmbed(26, 26))
		v.SendEmbed(ch, MockMessageEmbed(1, 26))
		mock.Expect(v.Bot.DG.ChannelMessageSendEmbed, ch.String(), MockMessageEmbed(1, 25))
		mock.Expect(v.Bot.DG.ChannelMessageSendEmbed, ch.String(), MockMessageEmbed(26, 27))
		v.SendEmbed(ch, MockMessageEmbed(1, 27))
		mock.Expect(v.Bot.DG.ChannelMessageSendEmbed, ch.String(), MockMessageEmbed(1, 25))
		mock.Expect(v.Bot.DG.ChannelMessageSendEmbed, ch.String(), MockMessageEmbed(26, 50))
		mock.Expect(v.Bot.DG.ChannelMessageSendEmbed, ch.String(), MockMessageEmbed(51, 75))
		mock.Expect(v.Bot.DG.ChannelMessageSendEmbed, ch.String(), MockMessageEmbed(76, 80))
		v.SendEmbed(ch, MockMessageEmbed(1, 80))
	}
}

func TestRequestPostWithBuffer(t *testing.T) {

	/*sendReq := func(rl *discordgo.RateLimiter, endpoint string) {
		bucket := rl.LockBucket(endpoint)

		headers := http.Header(make(map[string][]string))

		headers.Set("X-RateLimit-Remaining", "0")
		// Reset for approx 2 seconds from now
		headers.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second*2).Unix(), 10))
		headers.Set("Date", time.Now().Format(time.RFC850))

		err := bucket.Release(headers)
		if err != nil {
			t.Errorf("Release returned error: %v", err)
		}
	}*/
}

func MockMessageContent(length int) []byte {
	r := make([]byte, length, length)
	for k := range r {
		r[k] = '\n'
	}
	return r
}
func TestSendMessage(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	msg1 := string(MockMessageContent(1))

	Check(sb.Guilds[NewDiscordGuild(TestServer)].SendMessage(NewDiscordChannel(TestChannelFree|1), msg1), errInvalidChannel, t)
	endpointprivate := discordgo.EndpointChannelMessages(strconv.Itoa(TestChannelPrivate))
	for k, v := range sb.Guilds {
		v.Bot.DG.Ratelimiter.GetBucket(endpointprivate).Remaining = 999
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointprivate, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(NewDiscordChannel(TestChannelPrivate), msg1)
		h := v.Bot.heartbeat
		Check(v.SendMessage("heartbeat", msg1), nil, t)
		Check(v.Bot.heartbeat, h+1, t)
		ch := NewDiscordChannel(TestChannel | (k.Convert() & 0xFF))
		endpointchannel := discordgo.EndpointChannelMessages(ch.String())
		v.Bot.DG.Ratelimiter.GetBucket(endpointchannel).Remaining = 999
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, msg1)
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, string(MockMessageContent(1999)))
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, string(MockMessageContent(2000)))
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, string(MockMessageContent(6000)))
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, "`")
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, "``")
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, "```")
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, "````")
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, "`````")
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, "``````")
		for i := 1; i < 30; i++ {
			mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
			v.SendMessage(ch, "```"+string(MockMessageContent(i))+"```")
		}
		for i := 1980; i < 1994; i++ {
			mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
			v.SendMessage(ch, "```"+string(MockMessageContent(i))+"```")
		}
		for i := 1995; i < 2001; i++ {
			mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
			mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
			v.SendMessage(ch, "```"+string(MockMessageContent(i))+"```")
		}
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", endpointchannel, "application/json", MockAny{}, MockAny{}, 0)
		v.SendMessage(ch, "```"+string(MockMessageContent(4000))+"```")
	}
}

func TestProcessModule(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	disabled := mockModule("ModuleDisabled")
	channelinclusive := mockModule("ModuleChannelInclusive")
	channelexclusive := mockModule("ModuleChannelExclusive")
	enabled := mockModule("Module")
	for _, v := range sb.Guilds {
		v.Config.Modules.Disabled["moduledisabled"] = true
		v.Config.Modules.Channels["modulechannelinclusive"] = map[DiscordChannel]bool{"1234": true}
		v.Config.Modules.Channels["modulechannelexclusive"] = map[DiscordChannel]bool{"1234": true, ChannelExclusion: true}
		Check(v.ProcessModule("1234", disabled), false, t)
		Check(v.ProcessModule("1111", disabled), false, t)
		Check(v.ProcessModule("", disabled), false, t)
		Check(v.ProcessModule("1234", channelinclusive), true, t)
		Check(v.ProcessModule("1111", channelinclusive), false, t)
		Check(v.ProcessModule("", channelinclusive), true, t)
		Check(v.ProcessModule("1234", channelexclusive), false, t)
		Check(v.ProcessModule("1111", channelexclusive), true, t)
		Check(v.ProcessModule("", channelexclusive), true, t)
		Check(v.ProcessModule("1234", enabled), true, t)
		Check(v.ProcessModule("1111", enabled), true, t)
		Check(v.ProcessModule("", enabled), true, t)
	}
}

func TestIsDebug(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	sb.DebugChannels[NewDiscordGuild(TestServer)] = NewDiscordChannel(TestChannel)
	sb.DebugChannels[NewDiscordGuild(TestServer|1)] = NewDiscordChannel(TestChannelFree | 1)
	Check(sb.Guilds[NewDiscordGuild(TestServer)].IsDebug(NewDiscordChannel(TestChannel)), true, t)
	Check(sb.Guilds[NewDiscordGuild(TestServer)].IsDebug(NewDiscordChannel(TestChannelFree)), false, t)
	Check(sb.Guilds[NewDiscordGuild(TestServer|1)].IsDebug(NewDiscordChannel(TestChannel)), false, t)
	Check(sb.Guilds[NewDiscordGuild(TestServer|1)].IsDebug(NewDiscordChannel(TestChannel|1)), false, t)
	Check(sb.Guilds[NewDiscordGuild(TestServer|1)].IsDebug(NewDiscordChannel(TestChannelFree|1)), true, t)
	Check(sb.Guilds[NewDiscordGuild(TestServer|2)].IsDebug(NewDiscordChannel(TestChannel)), false, t)
	Check(sb.Guilds[NewDiscordGuild(TestServer|2)].IsDebug(NewDiscordChannel(TestChannel|2)), false, t)
	Check(sb.Guilds[NewDiscordGuild(TestServer|2)].IsDebug(NewDiscordChannel(TestChannelFree|2)), false, t)
}

func TestProcessMember(t *testing.T) {
	sb, dbmock, _ := MockSweetieBot(t)
	for _, v := range sb.Guilds {
		m := mockDiscordMember(TestUserAssigned, 0)
		dbmock.ExpectExec("CALL AddUser\\(\\?,\\?,\\?,\\?\\)").WithArgs(TestUserAssigned, m.User.Username, SBatoi(m.User.Discriminator), false).WillReturnResult(sqlmock.NewResult(0, 0))
		dbmock.ExpectExec("CALL AddMember\\(\\?,\\?,\\?,\\?\\)").WithArgs(TestUserAssigned, SBatoi(v.ID), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
		v.ProcessMember(m)
	}
}

func TestProcessGuild(t *testing.T) {
	sb, dbmock, _ := MockSweetieBot(t)
	g := mockDiscordGuild(12)
	args := make([]driver.Value, 4*len(g.Members), 4*len(g.Members))
	for k := range args {
		args[k] = sqlmock.AnyArg()
	}

	args2 := make([]driver.Value, 3*len(g.Members), 3*len(g.Members))
	for k := range args2 {
		args2[k] = sqlmock.AnyArg()
	}

	for _, v := range sb.Guilds {
		dbmock.ExpectExec("INSERT IGNORE INTO users.*").WithArgs(args2...).WillReturnResult(sqlmock.NewResult(0, 0))
		dbmock.ExpectExec("INSERT IGNORE INTO members.*").WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 0))
		v.ProcessGuild(g)
		Check(v.Name, "Test Server 12", t)
		Check(v.OwnerID, NewDiscordUser(TestOwnerServer|12), t)
	}
}

func TestFindChannelID(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for k, v := range sb.Guilds {
		channels := []int{TestChannel, TestChannelSpoil, TestChannelFree, TestChannelLog, TestChannelMod, TestChannelBored, TestChannelJail, TestChannelWelcome}
		i := int(k.Convert() & 0xFF)
		for _, c := range channels {
			Check(v.FindChannelID(mockDiscordChannel(c, i).Name), strconv.Itoa(c|i), t)
		}
		Check(v.FindChannelID(mockDiscordChannel(TestChannelPrivate, i).Name), "", t)
		Check(v.FindChannelID("asdf"), "", t)
		Check(v.FindChannelID(""), "", t)
	}
}

func TestLog(t *testing.T) {
	sb, dbmock, _ := MockSweetieBot(t)
	for _, v := range sb.Guilds {
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", discordgo.EndpointChannelMessages(v.Config.Log.Channel.String()), "application/json", MockAny{}, MockAny{}, 0)
		dbmock.ExpectExec("INSERT INTO debuglog.*").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
		v.Log("Test")
	}
}

func TestLogError(t *testing.T) {
	sb, dbmock, _ := MockSweetieBot(t)
	for _, v := range sb.Guilds {
		v.LogError("Test", nil)
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", discordgo.EndpointChannelMessages(v.Config.Log.Channel.String()), "application/json", MockAny{}, MockAny{}, 0)
		dbmock.ExpectExec("INSERT INTO debuglog.*").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
		v.LogError("test: ", errors.New("Ignore this error"))
	}
}

func TestSendError(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	for _, v := range sb.Guilds {
		mock.Expect(v.Bot.DG.RequestWithLockedBucket, "POST", discordgo.EndpointChannelMessages(strconv.Itoa(TestChannel)), "application/json", MockAny{}, MockAny{}, 0)
		v.SendError(NewDiscordChannel(TestChannel), "test", time.Now().UTC().Unix())
	}
}

func TestUserHasRole(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF
		Check(v.UserHasRole(NewDiscordUser(TestUserAssigned|i), NewDiscordRole(TestRoleAssign|i)), true, t)
		Check(v.UserHasRole(NewDiscordUser(TestUserAssigned|i), NewDiscordRole(TestRoleUser|i)), false, t)
		Check(v.UserHasRole(NewDiscordUser(TestMod|i), NewDiscordRole(TestRoleMod|i)), true, t)
		Check(v.UserHasRole(NewDiscordUser(TestUserSilence|i), NewDiscordRole(TestRoleSilence|i)), true, t)
		Check(v.UserHasRole(NewDiscordUser(TestUserSilence|i), NewDiscordRole(TestRoleMod|i)), false, t)
		Check(v.UserHasRole(NewDiscordUser(TestUserNonAssign|i), NewDiscordRole(TestRoleUser|i)), true, t)
		Check(v.UserHasRole(NewDiscordUser(TestUserNonAssign|i), NewDiscordRole(TestRoleSilence|i)), false, t)
		Check(v.UserHasRole(NewDiscordUser(TestUserBoring|i), NewDiscordRole(TestRoleAssign|i)), false, t)
		Check(v.UserHasRole(NewDiscordUser(TestUserBoring|i), NewDiscordRole(TestRoleUser|i)), false, t)
		Check(v.UserHasRole(NewDiscordUser(TestUserBoring|i), NewDiscordRole(TestRoleMod|i)), false, t)
		Check(v.UserHasRole(NewDiscordUser(TestUserBoring|i), NewDiscordRole(TestRoleSilence|i)), false, t)
	}
}
func TestUserCanUseCommand(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF
		any := mockCommand("Any")
		disabled := mockCommand("Disabled")
		restricted := mockCommandFull(CommandInfo{Name: "Restricted", Restricted: true})
		main := mockCommandFull(CommandInfo{Name: "MainGuild", MainInstance: true})
		mod := mockCommandFull(CommandInfo{Name: "Mod", Sensitive: true})
		exclude := mockCommand("Exclude")
		list := mockCommand("List")
		module := mockModule("TESTMODULE")

		commands := []Command{any, disabled, restricted, main, mod, exclude, list}
		for _, command := range commands {
			v.AddCommand(command, module)
		}
		v.Config.Modules.CommandDisabled = map[CommandID]bool{"disabled": true}
		v.Config.Modules.CommandRoles["mod"] = map[DiscordRole]bool{NewDiscordRole(TestRoleMod | i): true}
		v.Config.Modules.CommandRoles["exclude"] = map[DiscordRole]bool{NewDiscordRole(TestRoleUser | i): true, "!": true}
		v.Config.Modules.CommandRoles["list"] = map[DiscordRole]bool{NewDiscordRole(TestRoleUser | i): true, NewDiscordRole(TestRoleAssign | i): true}

		fn := func(user DiscordUser, expected []Command, bypass bool, ignore bool) {
			for _, command := range commands {
				expect := false
				for _, c := range expected {
					if c == command && (!c.Info().MainInstance || k == sb.MainGuildID || user == sb.Owner) {
						expect = true
						break
					}
				}

				mock.Expect(v.Bot.DG.GuildMember, MockAny{}, MockAny{})
				b, err := v.UserCanUseCommand(user, command, false)
				Check(b, bypass, t)
				if expect {
					if !Check(err, nil, t) {
						fmt.Println(k, command.Info().Name)
						v.UserCanUseCommand(user, command, false)
					}
				} else if !CheckNot(err, nil, t) {
					fmt.Println(k, command.Info().Name)
				}
				mock.Expect(v.Bot.DG.GuildMember, MockAny{}, MockAny{})
				b, err = v.UserCanUseCommand(user, command, true)
				Check(b, bypass, t)
				if ignore && expect {
					Check(err, nil, t)
				} else {
					CheckNot(err, nil, t)
				}
			}
		}

		fn(NewDiscordUser(TestOwnerBot), []Command{any, restricted, disabled, mod, main, exclude, list}, true, true)
		fn(NewDiscordUser(TestOwnerServer|i), []Command{any, disabled, mod, main, exclude, list}, true, true)
		/*fn(NewDiscordUser(TestSelfID), []Command{any, mod, main, exclude, list}, true, true)
		fn(NewDiscordUser(TestAdminMod|i), []Command{any, disabled, mod, main, exclude, list}, true, true)
		fn(NewDiscordUser(TestAdmin|i), []Command{any, disabled, mod, main, exclude, list}, true, true)
		fn(NewDiscordUser(TestMod|i), []Command{any, mod, main, exclude}, false, true)
		fn(NewDiscordUser(TestUserAssigned|i), []Command{any, main, exclude, list}, false, false)
		fn(NewDiscordUser(TestUserNonAssign|i), []Command{any, main, list}, false, false)
		fn(NewDiscordUser(TestUserBoring|i), []Command{any, main, exclude}, false, false)
		fn(NewDiscordUser(TestUserSilence|i), []Command{}, false, false)
		fn(NewDiscordUser(TestUserBot|i), []Command{any, main, exclude}, false, false)*/
	}
}
func TestUserIsMod(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF
		Check(v.UserIsMod(NewDiscordUser(TestUserAssigned|i)), false, t)
		Check(v.UserIsMod(NewDiscordUser(TestMod|i)), true, t)
		Check(v.UserIsMod(NewDiscordUser(TestUserSilence|i)), false, t)
		Check(v.UserIsMod(NewDiscordUser(TestOwnerBot|i)), false, t)
		Check(v.UserIsMod(NewDiscordUser(TestUserBoring|i)), false, t)
		Check(v.UserIsMod(NewDiscordUser(TestAdminMod|i)), true, t)
	}
}
func TestUserIsAdmin(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF
		Check(v.UserIsAdmin(NewDiscordUser(TestUserAssigned|i)), false, t)
		Check(v.UserIsAdmin(NewDiscordUser(TestMod|i)), false, t)
		Check(v.UserIsAdmin(NewDiscordUser(TestUserSilence|i)), false, t)
		Check(v.UserIsAdmin(NewDiscordUser(TestOwnerBot|i)), true, t)
		Check(v.UserIsAdmin(NewDiscordUser(TestOwnerServer|i)), true, t)
		Check(v.UserIsAdmin(NewDiscordUser(TestAdmin|i)), true, t)
		Check(v.UserIsAdmin(NewDiscordUser(TestAdminMod|i)), true, t)
		Check(v.UserIsAdmin(NewDiscordUser(TestUserBoring|i)), false, t)
	}
}
func TestGetRoles(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF
		v.Config.Modules.CommandRoles["one"] = map[DiscordRole]bool{NewDiscordRole(TestRoleMod | i): true}
		v.Config.Modules.CommandRoles["exclude"] = map[DiscordRole]bool{NewDiscordRole(TestRoleUser | i): true, "!": true}
		v.Config.Modules.CommandRoles["roles"] = map[DiscordRole]bool{NewDiscordRole(TestRoleUser | i): true, NewDiscordRole(TestRoleAssign | i): true}
		Check(v.GetRoles(""), "", t)
		Check(v.GetRoles("one"), "@Mod Role", t)
		Check(v.GetRoles("exclude"), "Any role except @User Role", t)
		Check(v.GetRoles("roles"), "@User Assignable, @User Role", t)
	}
}
func TestGetChannels(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF
		v.Config.Modules.CommandChannels["one"] = map[DiscordChannel]bool{NewDiscordChannel(TestChannel | i): true}
		v.Config.Modules.CommandChannels["exclude"] = map[DiscordChannel]bool{NewDiscordChannel(TestChannelBored | i): true, "!": true}
		v.Config.Modules.CommandChannels["roles"] = map[DiscordChannel]bool{NewDiscordChannel(TestChannelFree | i): true, NewDiscordChannel(TestChannelBored | i): true}
		Check(v.GetChannels(""), "", t)
		Check(v.GetChannels("one"), "#Test Channel", t)
		Check(v.GetChannels("exclude"), "Any channel except #Bored Channel", t)
		Check(v.GetChannels("roles"), "#Bored Channel, #Free Channel", t)
	}
}
func TestGetTimezone(t *testing.T) {
	sb, dbmock, _ := MockSweetieBot(t)
	loc, _ := time.LoadLocation("America/New_York")

	for _, v := range sb.Guilds {
		Check(v.GetTimezone(UserEmpty), time.UTC, t)
		dbmock.ExpectQuery("SELECT Location FROM users WHERE ID = \\?").WithArgs(0).WillReturnRows(sqlmock.NewRows([]string{"Location"}).AddRow("America/New_York"))
		Check(v.GetTimezone(DiscordUser("0")).String(), loc.String(), t)
		dbmock.ExpectQuery("SELECT Location FROM users WHERE ID = \\?").WithArgs(0).WillReturnRows(sqlmock.NewRows([]string{"Location"}))
		Check(v.GetTimezone(DiscordUser("0")), time.UTC, t)
		v.Config.Users.TimezoneLocation = "America/New_York"
		Check(v.GetTimezone(UserEmpty).String(), loc.String(), t)
		dbmock.ExpectQuery("SELECT Location FROM users WHERE ID = \\?").WithArgs(0).WillReturnRows(sqlmock.NewRows([]string{"Location"}))
		Check(v.GetTimezone(DiscordUser("0")).String(), loc.String(), t)
	}
}

func TestParseCommonTime(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for _, v := range sb.Guilds {
		tm, err := v.ParseCommonTime("Jun 30 2016", UserEmpty, time.Now())
		Check(err, nil, t)
		Check(tm, time.Date(2016, 6, 30, 0, 0, 0, 0, time.UTC), t)
	}
}

func TestSetupSilenceRole(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for _, v := range sb.DG.State.Guilds {
		g := sb.Guilds[DiscordGuild(v.ID)]
		for _, c := range v.Channels {
			if g.Config.Users.JailChannel.Equals(c.ID) {
				allow := discordgo.PermissionSendMessages | discordgo.PermissionReadMessages | discordgo.PermissionReadMessageHistory
				mock.Expect(g.Bot.DG.ChannelPermissionSet, c.ID, g.Config.Basic.SilenceRole.String(), "role", allow, 0)
			} else {
				deny := discordgo.PermissionSendMessages | discordgo.PermissionAddReactions
				if len(c.PermissionOverwrites) > 0 && c.PermissionOverwrites[0].ID == g.Config.Basic.SilenceRole.String() {
					deny = discordgo.PermissionAllText | discordgo.PermissionAddReactions
				}
				mock.Expect(g.Bot.DG.ChannelPermissionSet, c.ID, g.Config.Basic.SilenceRole.String(), "role", 0, deny)
			}
			if g.Config.Users.WelcomeChannel.Equals(c.ID) {
				allow := discordgo.PermissionSendMessages | discordgo.PermissionReadMessages
				mock.Expect(g.Bot.DG.ChannelPermissionSet, c.ID, g.ID, "role", allow, 0)
			}
		}
		g.setupSilenceRole()
	}
}

func TestGetUserName(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	for k, v := range sb.Guilds {
		i := int(k.Convert() & 0xFF)
		Check(v.GetUserName(NewDiscordUser(0)), "<@0>", t)
		Check(v.GetUserName(NewDiscordUser(TestUserBoring|uint64(i))), "Boring User:"+SBitoa(k.Convert()), t)
		m, _ := sb.DG.State.Member(v.ID, strconv.Itoa(TestUserBoring|i))
		m.Nick = ""
		Check(v.GetUserName(NewDiscordUser(TestUserBoring|uint64(i))), "Boring User", t)
	}
}

func TestIDsToUsernames(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	for k, v := range sb.Guilds {
		i := uint64(k.Convert() & 0xFF)
		m, _ := sb.DG.State.Member(v.ID, strconv.Itoa(TestUserBoring|int(i)))
		m.Nick = ""
		d := v.IDsToUsernames([]uint64{0, TestUserBoring | i, TestUserBot | i}, false)
		Check(d[0], "<@0>", t)
		Check(d[1], "Boring User", t)
		Check(d[2], "Bot User:"+SBitoa(k.Convert()), t)
		nd := v.IDsToUsernames([]uint64{0, TestUserBoring | i, TestUserBot | i}, true)
		Check(nd[0], "<@0>", t)
		Check(nd[1], fmt.Sprintf("Boring User#100%v", i), t)
		Check(nd[2], fmt.Sprintf("Bot User:%v (Bot User#100%v)", k, i), t)
	}
}

func TestGetBotName(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	for _, v := range sb.Guilds {
		Check(v.GetBotName(), sb.SelfName, t)
		v.BotNick = "test"
		Check(v.GetBotName(), "test", t)
	}
}

func TestSanitize(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	for k, v := range sb.Guilds {
		i := k.Convert() & 0xFF
		megastring := fmt.Sprintf("<@%v> <@!0> http://www.asdf.com https://asdf2.sdfs [](/emote) ```", TestUserBoring|i)

		Check(v.Sanitize(megastring, 0), megastring, t)
		Check(v.Sanitize("", 0), "", t)
		Check(v.Sanitize("", CleanAll), "", t)
		Check(v.Sanitize("a", CleanAll), "a", t)
		Check(v.Sanitize("<@>", CleanMentions), "<@>", t)
		Check(v.Sanitize("<@!>", CleanMentions), "<@!>", t)
		Check(v.Sanitize("<@0>", CleanMentions), "<@0>", t)
		Check(v.Sanitize("<@!0>", CleanMentions), "<@!0>", t)
		Check(v.Sanitize(NewDiscordUser(TestUserBoring|i).Display(), CleanMentions), "Boring User:"+strconv.Itoa(int(k.Convert())), t)
		Check(v.Sanitize(fmt.Sprintf("<@!%v>", TestUserBoring|i), CleanMentions), "Boring User:"+strconv.Itoa(int(k.Convert())), t)
		Check(v.Sanitize("<@>", CleanPings), "<@>", t)
		Check(v.Sanitize("<@!>", CleanPings), "<@!>", t)
		Check(v.Sanitize("<@0>", CleanPings), "<\\@0>", t)
		Check(v.Sanitize("<@!0>", CleanPings), "<\\@!0>", t)
		Check(v.Sanitize("<@1234578>", CleanPings), "<\\@1234578>", t)
		Check(v.Sanitize("<@!1234578>", CleanPings), "<\\@!1234578>", t)
		Check(v.Sanitize("http:/", CleanURL), "http:/", t)
		Check(v.Sanitize("https:/", CleanURL), "https:/", t)
		Check(v.Sanitize("http://www.asdf.com", CleanURL), "<http://www.asdf.com>", t)
		Check(v.Sanitize("https://asdf2.sdfs", CleanURL), "<https://asdf2.sdfs>", t)
		Check(v.Sanitize("asdf http://www.asdf.com https://asdf2.sdfs d", CleanURL), "asdf <http://www.asdf.com> <https://asdf2.sdfs> d", t)
		Check(v.Sanitize("[](", CleanEmotes), "[](", t)
		Check(v.Sanitize("[](/blah)", CleanEmotes), "[\u200B](/blah)", t)
		Check(v.Sanitize("```[]```", CleanCode), "\\`\\`\\`[]\\`\\`\\`", t)
		Check(v.Sanitize("`````````", CleanCode), "\\`\\`\\`\\`\\`\\`\\`\\`\\`", t)

		Check(v.Sanitize(megastring, CleanAll), "Boring User:"+strconv.Itoa(int(k.Convert()))+" <\\@!0> <http://www.asdf.com> <https://asdf2.sdfs> [\u200B](/emote) \\`\\`\\`", t)
	}
}

func TestBulkDelete(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)

	for k, v := range sb.Guilds {
		i := int(k.Convert() & 0xFF)
		ch := mockDiscordChannel(TestChannelFree, i)
		Check(v.BulkDelete(nil, []string{}), errInvalidChannel, t)
		Check(v.BulkDelete(mockDiscordChannel(1234, 999), []string{}), errInvalidChannel, t)
		mock.Expect(v.Bot.DG.ChannelMessagesBulkDelete, ch.ID, []string{})
		Check(v.BulkDelete(ch, []string{}), nil, t)
	}
}

func TestChannelMessageDelete(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	for k, v := range sb.Guilds {
		i := int(k.Convert() & 0xFF)
		ch := mockDiscordChannel(TestChannelFree, i)
		Check(v.ChannelMessageDelete(nil, ""), errInvalidChannel, t)
		Check(v.ChannelMessageDelete(mockDiscordChannel(1234, 999), ""), errInvalidChannel, t)
		mock.Expect(v.Bot.DG.ChannelMessageDelete, ch.ID, "")
		Check(v.ChannelMessageDelete(ch, ""), nil, t)
		mock.Expect(v.Bot.DG.ChannelMessageDelete, ch.ID, "0")
		Check(v.ChannelMessageDelete(ch, "0"), nil, t)
	}
}

func TestChannelPermissionSet(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	for k, v := range sb.Guilds {
		i := int(k.Convert() & 0xFF)
		ch := mockDiscordChannel(TestChannelFree, i)
		Check(v.ChannelPermissionSet(nil, "", "", 0, 0), errInvalidChannel, t)
		Check(v.ChannelPermissionSet(mockDiscordChannel(1234, 999), "", "", 0, 0), errInvalidChannel, t)
		mock.Expect(v.Bot.DG.ChannelPermissionSet, ch.ID, "", "", 0, 0)
		Check(v.ChannelPermissionSet(ch, "", "", 0, 0), nil, t)
		mock.Expect(v.Bot.DG.ChannelPermissionSet, ch.ID, "1", "2", 3, 4)
		Check(v.ChannelPermissionSet(ch, "1", "2", 3, 4), nil, t)
	}
}

func TestNewGuildInfo(t *testing.T) {
	sb, _, _ := MockSweetieBot(t)
	g := NewGuildInfo(sb, &discordgo.Guild{
		ID:      "1",
		Name:    "2",
		OwnerID: "3",
	})
	Check(g.ID, "1", t)
	Check(g.Name, "2", t)
	Check(g.OwnerID, DiscordUser("3"), t)
}
