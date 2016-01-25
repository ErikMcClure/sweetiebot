package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
)

// Modules monitor all incoming messages and users that have joined a given channel.
type Module interface {
  ProcessMsg(msg *discordgo.Message)
  ProcessUser(ID int)
}

// Commands are any command that is addressed to the bot, optionally filtered by channel.
type Command interface {
  Name() string
  Process(args []string)
}