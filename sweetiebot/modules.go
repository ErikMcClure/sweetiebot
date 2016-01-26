package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
)

// Giving each possible hook function its own interface ensures each module
// only has to define functions for the hooks it actually cares about

type ModuleOnEvent interface {
  Module
  OnEvent(*discordgo.Session, *discordgo.Event)
}

type ModuleOnTypingStart interface {
  Module
  OnTypingStart(*discordgo.Session, *discordgo.TypingStart)
}

type ModuleOnMessageCreate interface {
  Module
  OnMessageCreate(*discordgo.Session, *discordgo.Message)
}

type ModuleOnMessageUpdate interface {
  Module
  OnMessageUpdate(*discordgo.Session, *discordgo.Message)
}

type ModuleOnMessageDelete interface {
  Module
  OnMessageDelete(*discordgo.Session, *discordgo.MessageDelete)
}

type ModuleOnMessageAck interface {
  Module
  OnMessageAck(*discordgo.Session, *discordgo.MessageAck)
}

type ModuleOnUserUpdate interface {
  Module
  OnUserUpdate(*discordgo.Session, *discordgo.User)
}

type ModuleOnPresenceUpdate interface {
  Module
  OnPresenceUpdate(*discordgo.Session, *discordgo.PresenceUpdate)
}

type ModuleOnVoiceStateUpdate interface {
  Module
  OnVoiceStateUpdate(*discordgo.Session, *discordgo.VoiceState)
}

type ModuleOnGuildUpdate interface {
  Module
  OnGuildUpdate(*discordgo.Session, *discordgo.Guild)
}

type ModuleOnGuildMemberAdd interface {
  Module
  OnGuildMemberAdd(*discordgo.Session, *discordgo.Member)
}

type ModuleOnGuildMemberRemove interface {
  Module
  OnGuildMemberRemove(*discordgo.Session, *discordgo.Member)
}

type ModuleOnGuildMemberUpdate interface {
  Module
  OnGuildMemberUpdate(*discordgo.Session, *discordgo.Member)
}

type ModuleOnGuildBanAdd interface {
  Module
  OnGuildBanAdd(*discordgo.Session, *discordgo.GuildBan)
}

type ModuleOnGuildBanRemove interface {
  Module
  OnGuildBanRemove(*discordgo.Session, *discordgo.GuildBan)
}

// Modules monitor all incoming messages and users that have joined a given channel.
type Module interface {
  Name() string
  Register(hooks *ModuleHooks)
  Channels() []string // If no channels are specified, runs on all channels (except bot-log)
}

// Commands are any command that is addressed to the bot, optionally filtered by channel.
type Command interface {
  Name() string
  Process(args []string)
  Usage() string
  UsageShort() string
  Roles() []string // If no roles are specified, everyone is assumed
}