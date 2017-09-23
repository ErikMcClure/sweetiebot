package main

import (
	"database/sql"
	"io/ioutil"
	"os"
	"strings"

	"../sweetiebot"
)

func loader() []sweetiebot.Module {
	modules := make([]sweetiebot.Module, 0, 6)
	modules = append(modules, &sweetiebot.StatusModule{})
	modules = append(modules, &sweetiebot.DebugModule{})
	modules = append(modules, &sweetiebot.UsersModule{})
	modules = append(modules, &sweetiebot.TagModule{Cache: make(map[string]*sql.Stmt)})
	modules = append(modules, &sweetiebot.ScheduleModule{})
	modules = append(modules, &sweetiebot.RolesModule{})
	modules = append(modules, &sweetiebot.PollModule{})
	modules = append(modules, &sweetiebot.HelpModule{})
	modules = append(modules, &sweetiebot.MarkovModule{})
	modules = append(modules, &sweetiebot.QuoteModule{})
	modules = append(modules, &sweetiebot.BucketModule{})
	modules = append(modules, &sweetiebot.ConfigModule{})
	return modules
}
func mainCode() int {
	token, _ := ioutil.ReadFile("token")
	bot := sweetiebot.New(strings.TrimSpace(string(token)), loader)
	if bot != nil {
		return bot.Connect()
	}
	return 0
}
func main() { os.Exit(mainCode()) }
