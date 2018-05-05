package sweetiebot_test

import (
	"testing"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"../boredmodule"
	"../bucketmodule"
	"../filtermodule"
	"../markovmodule"
	"../miscmodule"
	"../quotemodule"
	"../rolesmodule"
	"../schedulermodule"
	"../spammodule"
	"../statusmodule"
	"../sweetiebot"
	. "../sweetiebot"
	"../tagmodule"
	"../usersmodule"
	"../wittymodule"
)

func loader(guild *sweetiebot.GuildInfo) []sweetiebot.Module {
	modules := make([]sweetiebot.Module, 0, 18)
	modules = append(modules, &sweetiebot.InfoModule{})
	modules = append(modules, &sweetiebot.ConfigModule{})
	modules = append(modules, &sweetiebot.DebugModule{})
	modules = append(modules, statusmodule.New())
	modules = append(modules, usersmodule.New())
	modules = append(modules, tagmodule.New())
	modules = append(modules, schedulermodule.New())
	modules = append(modules, rolesmodule.New())
	modules = append(modules, markovmodule.New())
	modules = append(modules, quotemodule.New())
	modules = append(modules, bucketmodule.New())
	modules = append(modules, boredmodule.New())
	modules = append(modules, miscmodule.New())
	modules = append(modules, wittymodule.New(guild))
	spam := spammodule.New()
	modules = append(modules, spam)
	modules = append(modules, filtermodule.New(guild, spam))

	return modules
}

func TestFuzzer(t *testing.T) {
	sb, dbmock, mock := MockSweetieBot(t)
	mock.Disable = true
	dbmock.MatchExpectationsInOrder(false)

	for i := 0; i < 30000; i++ {
		dbmock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
		dbmock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{}))
		dbmock.ExpectPrepare(".*")
	}

	for _, info := range sb.Guilds {
		info.Modules = loader(info)
		for _, v := range info.Modules {
			info.RegisterModule(v)
			for _, command := range v.Commands() {
				info.AddCommand(command, v)
			}
		}
	}

	sb.EmptyGuild.Modules = loader(sb.EmptyGuild)

	for _, v := range sb.EmptyGuild.Modules {
		sb.EmptyGuild.RegisterModule(v)
		for _, command := range v.Commands() {
			if command.Info().ServerIndependent {
				sb.EmptyGuild.AddCommand(command, v)
			}
		}
	}

	for k, v := range sb.Guilds {
		i := int(k.Convert() & 0xFF)
		for _, m := range v.Modules {
			for _, command := range m.Commands() {
				if command.Info().Name == "Update" {
					continue
				}
				command.Process([]string{}, MockMessage("", TestChannel, 0, TestUserBoring, i), []int{}, v)
			}
		}
	}

	/*for _, v := range sb.Guilds {
		for _, m := range v.Modules {
			for _, command := range m.Commands() {
				if command.Info().Name == "Update" {
					continue
				}
				fmt.Printf("%s - %s - %s\n", v.ID, m.Name(), command.Info().Name)
				CommandFuzzer(command, v, t)
			}
		}
	}

	for _, m := range sb.EmptyGuild.Modules {
		for _, command := range m.Commands() {
			fmt.Printf("%s - %s - %s", v.ID, m.Name(), command.Info().Name)
			CommandFuzzer(command, sb.EmptyGuild, t)
		}
	}*/
}
