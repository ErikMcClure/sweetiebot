package main

import (
  "../sweetiebot"
	"io/ioutil"
)

func main() {
  token, _ := ioutil.ReadFile("token")
  sweetiebot.Initialize(string(token))
}