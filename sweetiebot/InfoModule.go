package sweetiebot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/blackhole12/discordgo"
)

// InfoModule contains help and about commands
type InfoModule struct {
}

// Name of the module
func (w *InfoModule) Name() string {
	return "Information"
}

// Commands in the module
func (w *InfoModule) Commands() []Command {
	return []Command{
		&helpCommand{},
		&aboutCommand{},
		&rulesCommand{},
		&changelogCommand{},
	}
}

// Description of the module
func (w *InfoModule) Description() string {
	return "Contains commands for getting information about the bot, commands, or the server the bot is in."
}

type helpCommand struct {
}

func (c *helpCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:  "Help",
		Usage: "[PM Only] Generates the list you are looking at right now.",
	}
}

// DumpCommandsModules dumps information about all commands and modules
func DumpCommandsModules(info *GuildInfo, footer string, description string, msg *discordgo.Message) *discordgo.MessageEmbed {
	showdisabled := info.UserIsMod(DiscordUser(msg.Author.ID))
	fields := make([]*discordgo.MessageEmbedField, 0, len(info.Modules))
	for _, v := range info.Modules {
		if strings.ToLower(v.Name()) == "status" && DiscordGuild(info.ID) != info.Bot.MainGuildID {
			continue // Never show the status module outside of the main guild
		}
		rawcmds := v.Commands()
		cmds := make([]Command, 0, len(rawcmds))
		for _, c := range rawcmds {
			if _, err := info.UserCanUseCommand(DiscordUser(msg.Author.ID), c, false); err == nil {
				cmds = append(cmds, c)
			}
		}
		if len(cmds) > 0 {
			s := make([]string, 0, len(cmds))
			for _, c := range cmds {
				s = append(s, c.Info().Name+info.Config.IsCommandDisabled(c))
			}
			if disabled := info.Config.IsModuleDisabled(v); len(disabled) == 0 || len(s) > 0 || showdisabled {
				fields = append(fields, &discordgo.MessageEmbedField{Name: "**" + v.Name() + disabled + "**", Value: strings.Join(s, "\n"), Inline: true})
			}
		} else if disabled := info.Config.IsModuleDisabled(v); len(disabled) == 0 || showdisabled {
			fields = append(fields, &discordgo.MessageEmbedField{Name: "**" + v.Name() + disabled + "**", Value: "*[no commands]*", Inline: true})
		}
	}
	name := info.Bot.AppName + " Commands"
	if info.Silver.Get() {
		name += ` 🥈`
	}
	return &discordgo.MessageEmbed{
		Type: "rich",
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://sweetiebot.io/help/",
			Name:    name,
			IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", info.Bot.SelfID, info.Bot.SelfAvatar),
		},
		Description: description,
		Color:       0x3e92e5,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footer,
		},
	}
}

func (c *helpCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) == 0 {
		return "", true, DumpCommandsModules(info, "For more information on a specific command, type !help [command].", "", msg)
	}
	arg := strings.ToLower(args[0])
	for _, v := range info.Modules {
		if strings.Compare(strings.ToLower(v.Name()), arg) == 0 {
			cmds := v.Commands()
			fields := make([]*discordgo.MessageEmbedField, 0, len(cmds))
			for _, c := range cmds {
				if _, err := info.UserCanUseCommand(DiscordUser(msg.Author.ID), c, false); err == nil {
					fields = append(fields, &discordgo.MessageEmbedField{Name: "**" + c.Info().Name + "**" + info.Config.IsCommandDisabled(c), Value: c.Info().Usage, Inline: false})
				}
			}
			color := 0x56d34f
			if len(info.Config.IsModuleDisabled(v)) > 0 {
				color = 0xd54141
			}

			embed := &discordgo.MessageEmbed{
				Type: "rich",
				Author: &discordgo.MessageEmbedAuthor{
					URL:     "https://sweetiebot.io/help/" + strings.ToLower(v.Name()),
					Name:    v.Name() + " Module Command List" + info.Config.IsModuleDisabled(v),
					IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", info.Bot.SelfID, info.Bot.SelfAvatar),
				},
				Color:       color,
				Description: v.Description(),
				Fields:      fields,
				Footer: &discordgo.MessageEmbedFooter{
					Text: "For more information on a specific command, type !help [command].",
				},
			}
			return "", true, embed
		}
	}
	v, ok := info.commands[CommandID(arg)]
	if !ok {
		parts := strings.Split(arg, ".")
		if len(parts) > 1 {
			s, ok := getConfigHelp(parts[0], parts[1])
			if ok {
				embed := &discordgo.MessageEmbed{
					Type: "rich",
					Author: &discordgo.MessageEmbedAuthor{
						URL:     "https://sweetiebot.io/help/" + parts[0] + "/#" + arg,
						Name:    arg,
						IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", info.Bot.SelfID, info.Bot.SelfAvatar),
					},
					Color:       0x414141,
					Description: s,
				}
				return "", true, embed
			}
		}
		return "```\n" + info.GetBotName() + " doesn't recognize that command, module or config option. You can check what commands " + info.GetBotName() + " knows by typing !help with no arguments.```", false, nil
	}
	return "", true, info.FormatUsage(v, v.Usage(info))
}
func (c *helpCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists all available commands " + info.GetBotName() + " knows, or gives information about the given command. Of course, you should have figured this out by now, since you just typed !help help for some reason.",
		Params: []CommandUsageParam{
			{Name: "command/module", Desc: "The command or module to display help for. You do not need to include a command's parent module, just the command name itself.", Optional: true},
		},
	}
}

type aboutCommand struct {
}

func (c *aboutCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:              "About",
		Usage:             "Displays information about the bot.",
		ServerIndependent: true,
	}
}
func (c *aboutCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	tag := " [release]"
	if info.Bot.Debug {
		tag = " [debug]"
	}
	embed := &discordgo.MessageEmbed{
		Type: "rich",
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://sweetiebot.io",
			Name:    info.GetBotName() + " v" + BotVersion.String() + tag,
			IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.png", info.Bot.SelfID, info.Bot.SelfAvatar),
		},
		Color: 0x3e92e5,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "**Author**", Value: "Blackhole#0173", Inline: true},
			{Name: "**Library**", Value: "discordgo", Inline: true},
			{Name: "**Owner ID**", Value: info.Bot.Owner.String(), Inline: true},
			{Name: "**Presence**", Value: Pluralize(int64(len(info.Bot.Guilds)), " server"), Inline: true},
			{Name: "**Uptime**", Value: TimeDiff(time.Duration(GetTimestamp(msg).Unix()-info.Bot.StartTime) * time.Second), Inline: true},
			{Name: "**Messages Seen**", Value: strconv.FormatUint(uint64(atomic.LoadUint32(&info.Bot.MessageCount)), 10), Inline: true},
			{Name: "**Website**", Value: "https://sweetiebot.io", Inline: false},
			{Name: "**Patreon**", Value: PatreonURL, Inline: false},
			{Name: "**Terms of Service**", Value: "By joining a server using this bot or adding this bot to your server, you give express permission for the bot to collect and store any information it deems necessary to perform its functions, including but not limited to, message content, message metadata, and user metadata.", Inline: false},
		},
	}
	return "", false, embed
}
func (c *aboutCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Displays information about " + info.Bot.AppName + ". What, did you think it would do something else?",
	}
}

type rulesCommand struct {
}

func (c *rulesCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:  "Rules",
		Usage: "Lists the rules of the server.",
	}
}
func (c *rulesCommand) Name() string {
	return "Rules"
}
func (c *rulesCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(info.Config.Information.Rules) == 0 {
		return "```\nI don't know what the rules are in this server... ¯\\_(ツ)_/¯```", false, nil
	}
	if len(args) < 1 {
		rules := make([]string, 0, len(info.Config.Information.Rules)+1)
		rules = append(rules, "Official rules of "+info.Name+":")
		keys := MapIntToSlice(info.Config.Information.Rules)
		sort.Ints(keys)

		for _, v := range keys {
			if !info.Config.Information.HideNegativeRules || v >= 0 {
				rules = append(rules, fmt.Sprintf("%v. %s", v, info.Config.Information.Rules[v]))
			}
		}
		return strings.Join(rules, "\n"), len(rules) > maxPublicRules, nil
	}

	arg, err := strconv.Atoi(args[0])
	if err != nil {
		return "```\nRule index must be a number!```", false, nil
	}
	rule, ok := info.Config.Information.Rules[arg]
	if !ok {
		return "```\nThat's not a rule! Stop making things up!```", false, nil
	}
	return fmt.Sprintf("%v. %s", arg, rule), false, nil
}
func (c *rulesCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists all the rules in this server, or displays the specific rule requested, if it exists. Rules can be set using `" + info.Config.Basic.CommandPrefix + "setconfig rules 1 this is a rule`",
		Params: []CommandUsageParam{
			{Name: "index", Desc: "Index of the rule to display. If omitted, displays all rules.", Optional: true},
		},
	}
}

type changelogCommand struct {
}

func (c *changelogCommand) Name() string {
	return "Changelog"
}
func (c *changelogCommand) Info() *CommandInfo {
	return &CommandInfo{
		Name:  "Changelog",
		Usage: "Retrieves the changelog for the bot.",
	}
}
func (c *changelogCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	v := Version{0, 0, 0, 0}
	if len(args) == 0 {
		versions := make([]string, 0, len(info.Bot.changelog)+1)
		versions = append(versions, "All versions of "+info.GetBotName()+" with a changelog:")
		keys := MapIntToSlice(info.Bot.changelog)
		sort.Ints(keys)
		for i := len(keys) - 1; i >= 0; i-- {
			k := keys[i]
			version := VersionInt(k)
			versions = append(versions, version.String())
		}
		return "```\n" + strings.Join(versions, "\n") + "```", len(versions) > MaxPublicLines, nil
	}
	if strings.ToLower(args[0]) == "current" {
		v = BotVersion
	} else {
		s := strings.Split(args[0], ".")
		if len(s) > 0 {
			i, _ := strconv.Atoi(s[0])
			v.major = byte(i)
		}
		if len(s) > 1 {
			i, _ := strconv.Atoi(s[1])
			v.minor = byte(i)
		}
		if len(s) > 2 {
			i, _ := strconv.Atoi(s[2])
			v.revision = byte(i)
		}
		if len(s) > 3 {
			i, _ := strconv.Atoi(s[3])
			v.build = byte(i)
		}
	}
	log, ok := info.Bot.changelog[v.Integer()]
	if !ok {
		return "```\nThat's not a valid version! Use this command with no arguments to list all valid versions, or use \"current\" to get the most recent changelog.```", false, nil
	}
	return fmt.Sprintf("```\n%s\n--------\n%s```", v.String(), log), false, nil
}
func (c *changelogCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Displays the given changelog for " + info.GetBotName() + ". If no version is given, lists all versions with a changelog. ",
		Params: []CommandUsageParam{
			{Name: "version", Desc: "A version in the format 1.2.3.4. Use \"current\" for the most recent version.", Optional: true},
		},
	}
}
