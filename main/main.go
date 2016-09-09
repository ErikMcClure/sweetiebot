package main

import (
	"io/ioutil"
	"strings"

	"../sweetiebot"
)

func main() {
	token, _ := ioutil.ReadFile("token")
	sweetiebot.Initialize(strings.TrimSpace(string(token)))
}
