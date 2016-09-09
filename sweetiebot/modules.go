package sweetiebot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Giving each possible hook function its own interface ensures each module
// only has to define functions for the hooks it actually cares about

type ModuleOnEvent interface {
	Module
	OnEvent(*GuildInfo, *discordgo.Event)
}

type ModuleOnTypingStart interface {
	Module
	OnTypingStart(*GuildInfo, *discordgo.TypingStart)
}

type ModuleOnMessageCreate interface {
	Module
	OnMessageCreate(*GuildInfo, *discordgo.Message)
}

type ModuleOnMessageUpdate interface {
	Module
	OnMessageUpdate(*GuildInfo, *discordgo.Message)
}

type ModuleOnMessageDelete interface {
	Module
	OnMessageDelete(*GuildInfo, *discordgo.Message)
}

type ModuleOnMessageAck interface {
	Module
	OnMessageAck(*GuildInfo, *discordgo.MessageAck)
}

type ModuleOnPresenceUpdate interface {
	Module
	OnPresenceUpdate(*GuildInfo, *discordgo.PresenceUpdate)
}

type ModuleOnVoiceStateUpdate interface {
	Module
	OnVoiceStateUpdate(*GuildInfo, *discordgo.VoiceState)
}

type ModuleOnGuildUpdate interface {
	Module
	OnGuildUpdate(*GuildInfo, *discordgo.Guild)
}

type ModuleOnGuildMemberAdd interface {
	Module
	OnGuildMemberAdd(*GuildInfo, *discordgo.Member)
}

type ModuleOnGuildMemberRemove interface {
	Module
	OnGuildMemberRemove(*GuildInfo, *discordgo.Member)
}

type ModuleOnGuildMemberUpdate interface {
	Module
	OnGuildMemberUpdate(*GuildInfo, *discordgo.Member)
}

type ModuleOnGuildBanAdd interface {
	Module
	OnGuildBanAdd(*GuildInfo, *discordgo.GuildBan)
}

type ModuleOnGuildBanRemove interface {
	Module
	OnGuildBanRemove(*GuildInfo, *discordgo.GuildBan)
}

type ModuleOnCommand interface {
	Module
	OnCommand(*GuildInfo, *discordgo.Message) bool
}

type ModuleOnIdle interface {
	Module
	OnIdle(*GuildInfo, *discordgo.Channel)
	IdlePeriod(*GuildInfo) int64
}

type ModuleOnTick interface {
	Module
	OnTick(*GuildInfo)
}

// Modules monitor all incoming messages and users that have joined a given channel.
type Module interface {
	Name() string
	Register(*GuildInfo)
}

// Commands are any command that is addressed to the bot, optionally restricted by role.
type Command interface {
	Name() string
	Process([]string, *discordgo.Message, *GuildInfo) (string, bool)
	Usage(*GuildInfo) string
	UsageShort() string
}

func (info *GuildInfo) GetActiveModules() string {
	s := []string{"Active Modules:"}
	for _, v := range info.modules {
		str := v.Name()
		_, ok := info.config.Module_disabled[strings.ToLower(str)]
		if ok {
			str += " [disabled]"
		}
		s = append(s, str)
	}
	return strings.Join(s, "\n  ")
}

func (info *GuildInfo) GetActiveCommands() string {
	s := []string{"Active Commands:"}
	for _, v := range info.commands {
		str := v.Name()
		_, disabled := info.config.Command_disabled[strings.ToLower(str)]
		_, restricted := sb.RestrictedCommands[strings.ToLower(str)]
		if restricted && !sb.IsDBGuild(info) {
			str += " [not available]"
		} else if disabled {
			str += " [disabled]"
		}
		s = append(s, str)
	}
	return strings.Join(s, "\n  ")
}

func (info *GuildInfo) GetRoles(c Command) string {
	m, ok := info.config.Command_roles[strings.ToLower(c.Name())]
	if !ok {
		return ""
	}

	s := make([]string, 0, len(m))
	for k, _ := range m {
		for _, v := range info.Guild.Roles {
			if v.ID == k {
				s = append(s, v.Name)
			}
		}
	}

	return strings.Join(s, ", ")
}

func (info *GuildInfo) FormatUsage(c Command, a string, b string) string {
	r := info.GetRoles(c)
	if len(r) > 0 {
		return a + "\n+" + r + "\n\n" + b
	} else {
		return a + "\n\n" + b
	}
}
