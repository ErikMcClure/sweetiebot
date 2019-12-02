package sweetiebot

import (
	"time"

	"github.com/erikmcclure/discordgo"
)

// Module monitors all incoming requests depending on what module interfaces they implement
type Module interface {
	Name() string
	Commands() []Command
	Description(*GuildInfo) string
	//Config() interface{}
}

// Giving each possible hook function its own interface ensures each module
// only has to define functions for the hooks it actually cares about

// ModuleOnEvent hook interface
type ModuleOnEvent interface {
	Module
	OnEvent(*GuildInfo, *discordgo.Event)
}

// ModuleOnMessageCreate hook interface
type ModuleOnMessageCreate interface {
	Module
	OnMessageCreate(*GuildInfo, *discordgo.Message)
}

// ModuleOnMessageUpdate hook interface
type ModuleOnMessageUpdate interface {
	Module
	OnMessageUpdate(*GuildInfo, *discordgo.Message)
}

// ModuleOnMessageDelete hook interface
type ModuleOnMessageDelete interface {
	Module
	OnMessageDelete(*GuildInfo, *discordgo.Message)
}

// ModuleOnVoiceStateUpdate hook interface
type ModuleOnVoiceStateUpdate interface {
	Module
	OnVoiceStateUpdate(*GuildInfo, *discordgo.VoiceState)
}

// ModuleOnGuildUpdate hook interface
type ModuleOnGuildUpdate interface {
	Module
	OnGuildUpdate(*GuildInfo, *discordgo.Guild)
}

// ModuleOnGuildMemberAdd hook interface
type ModuleOnGuildMemberAdd interface {
	Module
	OnGuildMemberAdd(*GuildInfo, *discordgo.Member, time.Time)
}

// ModuleOnGuildMemberRemove hook interface
type ModuleOnGuildMemberRemove interface {
	Module
	OnGuildMemberRemove(*GuildInfo, *discordgo.Member, time.Time)
}

// ModuleOnGuildMemberUpdate hook interface
type ModuleOnGuildMemberUpdate interface {
	Module
	OnGuildMemberUpdate(*GuildInfo, *discordgo.Member, time.Time)
}

// ModuleOnGuildBanAdd hook interface
type ModuleOnGuildBanAdd interface {
	Module
	OnGuildBanAdd(*GuildInfo, *discordgo.GuildBanAdd)
}

// ModuleOnGuildBanRemove hook interface
type ModuleOnGuildBanRemove interface {
	Module
	OnGuildBanRemove(*GuildInfo, *discordgo.GuildBanRemove)
}

// ModuleOnGuildRoleDelete hook interface
type ModuleOnGuildRoleDelete interface {
	Module
	OnGuildRoleDelete(*GuildInfo, *discordgo.GuildRoleDelete)
}

// ModuleOnCommand hook interface
type ModuleOnCommand interface {
	Module
	OnCommand(*GuildInfo, *discordgo.Message) bool
}

// ModuleOnTick hook interface
type ModuleOnTick interface {
	Module
	OnTick(*GuildInfo, time.Time)
}

// CommandUsageParam describes a single parameter to a command
type CommandUsageParam struct {
	Name     string
	Desc     string
	Optional bool
	Variadic bool
}

// CommandUsage defines the help parameters for a command
type CommandUsage struct {
	Desc   string
	Params []CommandUsageParam
}

// CommandInfo defines the properties of a command
type CommandInfo struct {
	Name              string
	Usage             string
	ServerIndependent bool
	Sensitive         bool
	Restricted        bool
	Silver            bool
	MainInstance      bool
}

// Command is any command that is addressed to the bot, optionally restricted by role.
type Command interface {
	Info() *CommandInfo
	Process([]string, *discordgo.Message, []int, *GuildInfo) (string, bool, *discordgo.MessageEmbed)
	Usage(*GuildInfo) *CommandUsage
}

type moduleHooks struct {
	OnEvent             []ModuleOnEvent
	OnMessageCreate     []ModuleOnMessageCreate
	OnMessageUpdate     []ModuleOnMessageUpdate
	OnMessageDelete     []ModuleOnMessageDelete
	OnGuildUpdate       []ModuleOnGuildUpdate
	OnGuildMemberAdd    []ModuleOnGuildMemberAdd
	OnGuildMemberRemove []ModuleOnGuildMemberRemove
	OnGuildMemberUpdate []ModuleOnGuildMemberUpdate
	OnGuildBanAdd       []ModuleOnGuildBanAdd
	OnGuildBanRemove    []ModuleOnGuildBanRemove
	OnGuildRoleDelete   []ModuleOnGuildRoleDelete
	OnCommand           []ModuleOnCommand
	OnTick              []ModuleOnTick
}

// RegisterModule registers a module with this guild
func (info *GuildInfo) RegisterModule(m Module) {
	if h, ok := m.(ModuleOnEvent); ok {
		info.hooks.OnEvent = append(info.hooks.OnEvent, h)
	}
	if h, ok := m.(ModuleOnMessageCreate); ok {
		info.hooks.OnMessageCreate = append(info.hooks.OnMessageCreate, h)
	}
	if h, ok := m.(ModuleOnMessageUpdate); ok {
		info.hooks.OnMessageUpdate = append(info.hooks.OnMessageUpdate, h)
	}
	if h, ok := m.(ModuleOnMessageDelete); ok {
		info.hooks.OnMessageDelete = append(info.hooks.OnMessageDelete, h)
	}
	if h, ok := m.(ModuleOnGuildUpdate); ok {
		info.hooks.OnGuildUpdate = append(info.hooks.OnGuildUpdate, h)
	}
	if h, ok := m.(ModuleOnGuildMemberAdd); ok {
		info.hooks.OnGuildMemberAdd = append(info.hooks.OnGuildMemberAdd, h)
	}
	if h, ok := m.(ModuleOnGuildMemberRemove); ok {
		info.hooks.OnGuildMemberRemove = append(info.hooks.OnGuildMemberRemove, h)
	}
	if h, ok := m.(ModuleOnGuildMemberUpdate); ok {
		info.hooks.OnGuildMemberUpdate = append(info.hooks.OnGuildMemberUpdate, h)
	}
	if h, ok := m.(ModuleOnGuildBanAdd); ok {
		info.hooks.OnGuildBanAdd = append(info.hooks.OnGuildBanAdd, h)
	}
	if h, ok := m.(ModuleOnGuildBanRemove); ok {
		info.hooks.OnGuildBanRemove = append(info.hooks.OnGuildBanRemove, h)
	}
	if h, ok := m.(ModuleOnGuildRoleDelete); ok {
		info.hooks.OnGuildRoleDelete = append(info.hooks.OnGuildRoleDelete, h)
	}
	if h, ok := m.(ModuleOnCommand); ok {
		info.hooks.OnCommand = append(info.hooks.OnCommand, h)
	}
	if h, ok := m.(ModuleOnTick); ok {
		info.hooks.OnTick = append(info.hooks.OnTick, h)
	}
}
