package main

import (
  "../sweetiebot"
	"io/ioutil"
  "strings"
)

func main() {
  token, _ := ioutil.ReadFile("token")
  sweetiebot.Initialize(strings.TrimSpace(string(token)))
}