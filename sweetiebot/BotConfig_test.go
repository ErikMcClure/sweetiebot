package sweetiebot

import (
	"fmt"
	"reflect"
	"testing"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/blackhole12/discordgo"
)

func TestDisabledCheck(t *testing.T) {
	t.Parallel()

	config := BotConfig{}
	Check(config.IsCommandDisabled(mockCommand("test")), "", t)
	Check(config.IsModuleDisabled(mockModule("test")), "", t)

	config.Modules.Disabled = make(map[ModuleID]bool)
	config.Modules.CommandDisabled = make(map[CommandID]bool)
	config.Modules.Disabled["foo"] = true
	config.Modules.Disabled["Bar"] = true
	config.Modules.CommandDisabled["foo2"] = true
	config.Modules.CommandDisabled["Bar2"] = true
	Check(config.IsCommandDisabled(mockCommand("test")), "", t)
	Check(config.IsCommandDisabled(mockCommand("TEST")), "", t)
	Check(config.IsCommandDisabled(mockCommand("Foo2")), " [disabled]", t)
	Check(config.IsCommandDisabled(mockCommand("foo2")), " [disabled]", t)
	Check(config.IsCommandDisabled(mockCommand("Bar2")), "", t) // This should fail because we're supposed to store the lower-case version
	Check(config.IsCommandDisabled(mockCommand("bar2")), "", t)

	Check(config.IsModuleDisabled(mockModule("test")), "", t)
	Check(config.IsModuleDisabled(mockModule("TEST")), "", t)
	Check(config.IsModuleDisabled(mockModule("Foo")), " [disabled]", t)
	Check(config.IsModuleDisabled(mockModule("foo")), " [disabled]", t)
	Check(config.IsModuleDisabled(mockModule("Bar")), "", t)
	Check(config.IsModuleDisabled(mockModule("bar")), "", t)
}

/*Aliases               map[string]string `json:"aliases"`
Channels           map[string]map[string]bool `json:"modulechannels"`
CommandLimits      map[string]int64           `json:"Commandlimits"`
*/

func TestSetConfig(t *testing.T) {
	t.Parallel()

	config := &BotConfig{}
	fnImportable := func(name string) {
		config.Basic.Importable = false
		name, _ = FixRequest(name, reflect.ValueOf(config).Elem())
		if s, ok := config.SetConfig(nil, name, "true"); !ok {
			t.Errorf("SetConfig(%s) returned %v", name, s)
		}
		if config.Basic.Importable != true {
			t.Errorf("Importable not true")
		}
	}
	if _, ok := config.SetConfig(nil, "importable", "true"); ok {
		t.Errorf("SetConfig shouldn't have found Importable by itself")
	}

	fnImportable("importable")
	fnImportable("importable")
	fnImportable("basic.importable")
	fnImportable("basic.IMPORTABLE")
	fnImportable("BASIC.importable")
	fnImportable("BASIC.IMPORTABLE")

	sb, dbmock, _ := MockSweetieBot(t)
	info := NewGuildInfo(sb, &discordgo.Guild{})
	info.commands["1234"] = mockCommand("1234")
	info.commands["1"] = mockCommand("1")
	info.commands[""] = mockCommand("")
	info.Modules = []Module{mockModule(""), mockModule("1")}
	dbmock.ExpectQuery("SELECT DISTINCT M.ID FROM members.*").WithArgs(sqlmock.AnyArg(), "1", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(1))

	fnSetInterface := func(name string, value interface{}) {
		name, _ = FixRequest(name, reflect.ValueOf(config).Elem())
		if s, ok := config.SetConfig(info, name, fmt.Sprintf("%v", value)); !ok {
			t.Errorf("SetConfig(%s) returned %v", name, s)
		}
	}
	fnSetInterface("modrole", 1234)
	Check(config.Basic.ModRole, DiscordRole("1234"), t)
	fnSetInterface("CommandPrefix", "^")
	Check(config.Basic.CommandPrefix, "^", t)
	fnSetInterface("CommandPerDuration", 12345)
	Check(config.Modules.CommandPerDuration, 12345, t)
	fnSetInterface("CommandMaxDuration", 123456)
	Check(config.Modules.CommandMaxDuration, int64(123456), t)

	fnFreeChannels := func(value DiscordChannel, extra ...string) {
		name, _ := FixRequest("FreeChannels", reflect.ValueOf(config).Elem())
		if s, ok := config.SetConfig(info, name, value.String(), extra...); !ok {
			t.Errorf("SetConfig(FreeChannels) returned %v", s)
		}
		if len(value) == 0 {
			Check(len(config.Basic.FreeChannels), 0, t)
		} else {
			Check(len(config.Basic.FreeChannels), len(extra)+1, t)
			if _, ok := config.Basic.FreeChannels[value]; !ok {
				t.Errorf("FreeChannels doesn't have %s", value)
			}
			for _, v := range extra {
				if _, ok := config.Basic.FreeChannels[DiscordChannel(v)]; !ok {
					t.Errorf("FreeChannels doesn't have %s", v)
				}
			}
		}
	}
	fnFreeChannels(DiscordChannel("123"), "654")
	fnFreeChannels(DiscordChannel("234"), "654", "34643")
	fnFreeChannels(ChannelEmpty)
	fnFreeChannels(DiscordChannel("2345"))

	fnCommandLimits := func(key string, value int64) {
		name, _ := FixRequest("CommandLimits", reflect.ValueOf(config).Elem())
		str := ""
		if value != 0 {
			str = fmt.Sprintf("%v", value)
		}
		s, ok := config.SetConfig(info, name, key, str)
		if (value != 0 && !ok) || (value == 0 && s == fmt.Sprintf("Deleted %v", value)) {
			t.Errorf("SetConfig(CommandLimits) returned %v", s)
		}
		if _, ok := config.Modules.CommandLimits[CommandID(key)]; ok == (value == 0) {
			t.Errorf("CommandLimits key state incorrect %s, %v\n%v", key, value, config.Modules.CommandLimits)
		}
	}

	fnCommandLimits("1234", 87)
	fnCommandLimits("1234", 0)
	fnCommandLimits("", 1)
	fnCommandLimits("", 0)

	// Attempt to set every single value to a blank value, which should reveal any configuration values that don't have a known type
	p := reflect.ValueOf(config).Elem()
	for i := 0; i < p.NumField(); i++ {
		name := p.Type().Field(i).Name + "."
		switch p.Field(i).Kind() {
		case reflect.Struct:
			for j := 0; j < p.Field(i).NumField(); j++ {
				path := name + p.Field(i).Type().Field(j).Name
				str, _ := config.SetConfig(info, path, "", "")
				if !CheckNot(str, "That config option has an unknown type!", t) {
					fmt.Println(path)
				}

				switch p.Field(i).Field(j).Interface().(type) {
				case bool:
					config.SetConfig(info, path, "true")
					Check(p.Field(i).Field(j).Interface().(bool), true, t)
					continue
				}

				config.SetConfig(info, path, "1", "1")
				_, ok := getConfigHelp(p.Type().Field(i).Name, p.Field(i).Type().Field(j).Name)
				if !Check(ok, true, t) {
					fmt.Println(path)
				}
				switch m := p.Field(i).Field(j).Interface().(type) {
				case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, float32, float64, uint64, DiscordChannel, DiscordRole, DiscordUser:
					Check(fmt.Sprint(m), "1", t)
				case []uint64:
					Check(m[0], 1, t)
				case map[string]string:
					v, _ := m["1"]
					Check(v, "1", t)
				case map[CommandID]int64:
					v, _ := m["1"]
					Check(v, int64(1), t)
				case map[DiscordChannel]float32:
					v, _ := m["1"]
					Check(v, float32(1.0), t)
				case map[int]string:
					v, _ := m[1]
					Check(v, "1", t)
				case map[DiscordChannel]bool:
					_, ok := m["1"]
					Check(ok, true, t)
				case map[DiscordRole]bool:
					_, ok := m["1"]
					Check(ok, true, t)
				case map[string]bool:
					_, ok := m["1"]
					Check(ok, true, t)
				case map[ModuleID]bool:
					_, ok := m["1"]
					Check(ok, true, t)
				case map[CommandID]bool:
					_, ok := m["1"]
					Check(ok, true, t)
				case map[CommandID]map[DiscordChannel]bool:
					v, ok := m["1"]
					Check(ok, true, t)
					_, ok = v["1"]
					Check(ok, true, t)
				case map[ModuleID]map[DiscordChannel]bool:
					v, ok := m["1"]
					Check(ok, true, t)
					_, ok = v["1"]
					Check(ok, true, t)
				case map[string]map[string]bool:
					v, ok := m["1"]
					Check(ok, true, t)
					_, ok = v["1"]
					Check(ok, true, t)
				case map[DiscordUser][]string:
					v, ok := m["1"]
					if !Check(ok, true, t) {
						t.Error("failure: ", path)
					}
					Check(v[0], "1", t)
				case map[CommandID]map[DiscordRole]bool:
					v, ok := m["1"]
					Check(ok, true, t)
					_, ok = v["1"]
					Check(ok, true, t)
				case map[string]map[DiscordChannel]bool:
					v, ok := m["1"]
					Check(ok, true, t)
					_, ok = v["1"]
					Check(ok, true, t)
				default:
					t.Error("Invalid config type: ", path)
				}
			}
		}
	}
}
