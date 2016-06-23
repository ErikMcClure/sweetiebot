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

// Modules monitor all incoming messages and users that have joined a given channel.
type Module interface {
  Name() string
  Register(hooks *ModuleHooks)
}

// Commands are any command that is addressed to the bot, optionally restricted by role.
type Command interface {
  Name() string
  Process([]string, *discordgo.Message) (string, bool)
  Usage() string
  UsageShort() string
}

func GetActiveModules() string {
  s := []string{"Active Modules:"}
  for _, v := range sb.modules {
    str := v.Name()
    _, ok := sb.config.Module_disabled[str]
    if ok { str += " [disabled]" }
    s = append(s, str)
  }
  return strings.Join(s, "\n  ")
}

func GetActiveCommands() string {
  s := []string{"Active Commands:"}
  for _, v := range sb.commands {
    str := v.Name() 
    _, ok := sb.config.Command_disabled[str]
    if ok { str += " [disabled]" }
    s = append(s, str)
  }
  return strings.Join(s, "\n  ")
}

func GetRoles(c Command) string {
  m, ok := sb.config.Command_roles[c.Name()]
  if !ok {
    return "";
  }
  
  s := make([]string, 0, len(m))
  for k, _ := range m { 
    for _, v := range sb.dg.State.Guilds[0].Roles {
      if v.ID == k {
        s = append(s, v.Name)
      }
    }
  }

  return strings.Join(s, ", ")
}

func FormatUsage(c Command, a string, b string) string {
  r := GetRoles(c)
  if len(r)>0 {
    return a + "\n+" + r + "\n\n" + b 
  } else {
    return a + "\n\n" + b
  }
}