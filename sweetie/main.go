package main

import (
	"os"

	"github.com/erikmcclure/sweetiebot/boredmodule"
	"github.com/erikmcclure/sweetiebot/bucketmodule"
	"github.com/erikmcclure/sweetiebot/countersmodule"
	"github.com/erikmcclure/sweetiebot/filtermodule"
	"github.com/erikmcclure/sweetiebot/markovmodule"
	"github.com/erikmcclure/sweetiebot/miscmodule"
	"github.com/erikmcclure/sweetiebot/quotemodule"
	"github.com/erikmcclure/sweetiebot/rolesmodule"
	"github.com/erikmcclure/sweetiebot/schedulermodule"
	"github.com/erikmcclure/sweetiebot/spammodule"
	"github.com/erikmcclure/sweetiebot/statusmodule"
	"github.com/erikmcclure/sweetiebot/sweetiebot"
	"github.com/erikmcclure/sweetiebot/tagmodule"
	"github.com/erikmcclure/sweetiebot/usersmodule"
	"github.com/erikmcclure/sweetiebot/wittymodule"
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
	modules = append(modules, countersmodule.New())
	modules = append(modules, wittymodule.New(guild))
	spam := spammodule.New()
	modules = append(modules, spam)
	modules = append(modules, filtermodule.New(guild, spam))

	return modules
}
func mainCode() int {
	bot := sweetiebot.New("", loader)
	if bot != nil {
		return bot.Connect()
	}
	return 0
}
func main() { os.Exit(mainCode()) }
