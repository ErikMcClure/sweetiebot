package main

import (
	"io/ioutil"
	"strings"

	"../sweetiebot"
)

func main() {
	token, _ := ioutil.ReadFile("token")
	bot := sweetiebot.New(strings.TrimSpace(string(token)))
	if bot != nil {
		bot.Connect()
	}
}
