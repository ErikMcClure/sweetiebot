package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
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
  OnMessageDelete(*discordgo.Session, *discordgo.Message)
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

type ModuleOnCommand interface {
  Module
  OnCommand(*discordgo.Session, *discordgo.Message) bool
}

type ModuleOnIdle interface {
  Module
  OnIdle(*discordgo.Session, *discordgo.Channel)
  IdlePeriod() int64 
}

type ModuleEnabledInterface interface {
  IsEnabled() bool
  Enable(bool)
}
type ModuleEnabled struct {
  enabled bool
}

// Modules monitor all incoming messages and users that have joined a given channel.
type Module interface {
  ModuleEnabledInterface
  Name() string
  Register(hooks *ModuleHooks)
  Channels() []string // If no channels are specified, runs on all channels (except bot-log)
}

// Commands are any command that is addressed to the bot, optionally restricted by role.
type Command interface {
  Name() string
  Process([]string, *discordgo.Message) (string, bool)
  Usage() string
  UsageShort() string
  Roles() []string // If no roles are specified, everyone is assumed
}

func (m *ModuleEnabled) IsEnabled() bool {
  return m.enabled
}
func (m *ModuleEnabled) Enable(b bool) {
  m.enabled = b
}

func GetActiveModules() string {
  s := []string{"Active Modules:"}
  for _, v := range sb.modules {
    str := v.Name()
    if !v.IsEnabled() { str += " [disabled]" }
    s = append(s, str)
  }
  return strings.Join(s, "\n  ")
}

func GetActiveCommands() string {
  s := []string{"Active Commands:"}
  for _, v := range sb.commands {
    str := v.c.Name() 
    _, ok := sb.disablecommands[str]
    if ok { str += " [disabled]" }
    s = append(s, str)
  }
  return strings.Join(s, "\n  ")
}


func FormatUsage(c Command, a string, b string) string {
  r := c.Roles()
  if len(r)>0 {
    return a + "\n+" + strings.Join(r, ", +") + "\n\n" + b 
  } else {
    return a + "\n\n" + b
  }
}