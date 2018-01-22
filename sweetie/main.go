package main

import (
	"os"

	"../boredmodule"
	"../bucketmodule"
	"../filtermodule"
	"../markovmodule"
	"../miscmodule"
	"../pollmodule"
	"../quotemodule"
	"../rolesmodule"
	"../schedulermodule"
	"../spammodule"
	"../statusmodule"
	"../sweetiebot"
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
	modules = append(modules, pollmodule.New())
	modules = append(modules, markovmodule.New())
	modules = append(modules, quotemodule.New())
	modules = append(modules, bucketmodule.New())
	modules = append(modules, boredmodule.New())
	modules = append(modules, miscmodule.New())
	modules = append(modules, wittymodule.New(guild))
	modules = append(modules, spammodule.New())
	modules = append(modules, filtermodule.New(guild))

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
