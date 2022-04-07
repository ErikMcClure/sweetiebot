package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/erikmcclure/sweetiebot/sweetiebot"
)

func main() {
	if len(os.Args) > 1 {
		if strings.ToLower(os.Args[1]) == "version" {
			fmt.Print(sweetiebot.BotVersion.Integer())
			return
		}
	}
	fmt.Println("Sweetie Bot Updater v" + sweetiebot.BotVersion.String())

	sb := sweetiebot.SweetieBot{Selfhoster: &sweetiebot.Selfhost{}}
	sb.Selfhoster.Version = sweetiebot.BotVersion.Integer()
	hostfile, err := ioutil.ReadFile("selfhost.json")
	if err != nil {
		fmt.Println("Error opening selfhost.json, aborting update attempt: ", err.Error())
		return
	}
	json.Unmarshal(hostfile, &sb)

	if err := sb.Selfhoster.DoUpdate(sb.DBAuth, sb.Token); err != nil {
		fmt.Println(err)
	}
}
