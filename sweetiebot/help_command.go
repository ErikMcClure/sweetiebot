package sweetiebot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type HelpModule struct {
}

func (w *HelpModule) Name() string {
	return "Help/About"
}

func (w *HelpModule) Register(info *GuildInfo) {}

func (w *HelpModule) Commands() []Command {
	return []Command{
		&HelpCommand{},
		&AboutCommand{},
		&RulesCommand{},
		&ChangelogCommand{},
	}
}

func (w *HelpModule) Description() string {
	return "Contains commands for getting information about Sweetie Bot, her commands, or the server she is in."
}

type HelpCommand struct {
}

func (c *HelpCommand) Name() string {
	return "Help"
}

func DumpCommandsModules(channelID string, info *GuildInfo, footer string, description string) *discordgo.MessageEmbed {
	fields := make([]*discordgo.MessageEmbedField, 0, len(info.modules))
	for _, v := range info.modules {
		cmds := v.Commands()
		if len(cmds) > 0 {
			s := make([]string, 0, len(cmds))
			for _, c := range cmds {
				s = append(s, c.Name()+info.IsCommandDisabled(c.Name()))
			}
			fields = append(fields, &discordgo.MessageEmbedField{Name: v.Name() + info.IsModuleDisabled(v.Name()), Value: strings.Join(s, "\n"), Inline: true})
		} else {
			fields = append(fields, &discordgo.MessageEmbedField{Name: v.Name() + info.IsModuleDisabled(v.Name()), Value: "*[no commands]*", Inline: true})
		}
	}
	return &discordgo.MessageEmbed{
		Type: "rich",
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://github.com/blackhole12/sweetiebot",
			Name:    "Sweetie Bot Commands",
			IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", sb.SelfID, sb.SelfAvatar),
		},
		Description: description,
		Color:       0x3e92e5,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footer,
		},
	}
}

func (c *HelpCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) == 0 {
		return "", true, DumpCommandsModules(msg.ChannelID, info, "For more information on a specific command, type !help [command].", "")
	}
	arg := strings.ToLower(args[0])
	for _, v := range info.modules {
		if strings.Compare(strings.ToLower(v.Name()), arg) == 0 {
			cmds := v.Commands()
			fields := make([]*discordgo.MessageEmbedField, 0, len(cmds))
			for _, c := range cmds {
				fields = append(fields, &discordgo.MessageEmbedField{Name: c.Name() + info.IsCommandDisabled(c.Name()), Value: c.UsageShort(), Inline: false})
			}
			color := 0x56d34f
			if len(info.IsModuleDisabled(v.Name())) > 0 {
				color = 0xd54141
			}

			embed := &discordgo.MessageEmbed{
				Type: "rich",
				Author: &discordgo.MessageEmbedAuthor{
					URL:     "https://github.com/blackhole12/sweetiebot#modules",
					Name:    v.Name() + " Module Command List" + info.IsModuleDisabled(v.Name()),
					IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", sb.SelfID, sb.SelfAvatar),
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
	v, ok := info.commands[strings.ToLower(args[0])]
	if !ok {
		return "```Sweetie Bot doesn't recognize that command or module. You can check what commands Sweetie Bot knows by typing !help with no arguments.```", false, nil
	}
	return "", true, info.FormatUsage(v, v.Usage(info))
}
func (c *HelpCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists all available commands Sweetie Bot knows, or gives information about the given command. Of course, you should have figured this out by now, since you just typed !help help for some reason.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "command/module", Desc: "The command or module to display help for. You do not need to include a command's parent module, just the command name itself.", Optional: true},
		},
	}
}
func (c *HelpCommand) UsageShort() string {
	return "[PM Only] Generates the list you are looking at right now."
}

type AboutCommand struct {
}

func (c *AboutCommand) Name() string {
	return "About"
}
func (c *AboutCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	tag := " [release]"
	if sb.Debug {
		tag = " [debug]"
	}
	owners := make([]string, 0, len(sb.Owners))
	for k, _ := range sb.Owners {
		owners = append(owners, SBitoa(k))
	}
	embed := &discordgo.MessageEmbed{
		Type: "rich",
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://github.com/blackhole12/sweetiebot",
			Name:    "Sweetie Bot v" + sb.version.String() + tag,
			IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", sb.SelfID, sb.SelfAvatar),
		},
		Color: 0x3e92e5,
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{Name: "Author", Value: "Blackhole#8270", Inline: true},
			&discordgo.MessageEmbedField{Name: "Library", Value: "discordgo", Inline: true},
			&discordgo.MessageEmbedField{Name: "Owner ID(s)", Value: strings.Join(owners, ", "), Inline: true},
			&discordgo.MessageEmbedField{Name: "Presence", Value: Pluralize(int64(len(sb.guilds)), " server"), Inline: true},
		},
	}
	return "", false, embed
}
func (c *AboutCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Displays information about Sweetie Bot. What, did you think it would do something else?",
	}
}
func (c *AboutCommand) UsageShort() string { return "Displays information about Sweetie Bot." }

type RulesCommand struct {
}

func (c *RulesCommand) Name() string {
	return "Rules"
}
func (c *RulesCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(info.config.Help.Rules) == 0 {
		return "```I don't know what the rules are in this server... ¯\\_(ツ)_/¯```", false, nil
	}
	if len(args) < 1 {
		rules := make([]string, 0, len(info.config.Help.Rules)+1)
		rules = append(rules, "Official rules of "+info.Guild.Name+":")
		keys := MapIntToSlice(info.config.Help.Rules)
		sort.Ints(keys)

		for _, v := range keys {
			if !info.config.Help.HideNegativeRules || v >= 0 {
				rules = append(rules, fmt.Sprintf("%v. %s", v, info.config.Help.Rules[v]))
			}
		}
		return strings.Join(rules, "\n"), len(rules) > 4, nil
	}

	arg, err := strconv.Atoi(args[0])
	if err != nil {
		return "```Rule index must be a number!```", false, nil
	}
	rule, ok := info.config.Help.Rules[arg]
	if !ok {
		return "```That's not a rule! Stop making things up!```", false, nil
	}
	return fmt.Sprintf("%v. %s", arg, rule), false, nil
}
func (c *RulesCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists all the rules in this server, or displays the specific rule requested, if it exists. Rules can be set using `!setconfig rules 1 this is a rule`",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "index", Desc: "Index of the rule to display. If omitted, displays all rules.", Optional: true},
		},
	}
}
func (c *RulesCommand) UsageShort() string { return "Lists the rules of the server." }

type ChangelogCommand struct {
}

func (c *ChangelogCommand) Name() string {
	return "Changelog"
}
func (c *ChangelogCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	v := Version{0, 0, 0, 0}
	if len(args) == 0 {
		versions := make([]string, 0, len(sb.changelog)+1)
		versions = append(versions, "All versions of Sweetie Bot with a changelog:")
		keys := MapIntToSlice(sb.changelog)
		sort.Ints(keys)
		for i := len(keys) - 1; i >= 0; i-- {
			k := keys[i]
			version := Version{byte(k >> 24), byte((k >> 16) & 0xFF), byte((k >> 8) & 0xFF), byte(k & 0xFF)}
			versions = append(versions, version.String())
		}
		return "```\n" + strings.Join(versions, "\n") + "```", len(versions) > 6, nil
	}
	if strings.ToLower(args[0]) == "current" {
		v = sb.version
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
	log, ok := sb.changelog[v.Integer()]
	if !ok {
		return "```That's not a valid version of Sweetie Bot! Use this command with no arguments to list all valid versions, or use \"current\" to get the most recent changelog.```", false, nil
	}
	return fmt.Sprintf("```\n%s\n--------\n%s```", v.String(), log), false, nil
}
func (c *ChangelogCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Displays the given changelog for Sweetie Bot. If no version is given, lists all versions with a changelog. ",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "version", Desc: "A version in the format 1.2.3.4. Use \"current\" for the most recent version.", Optional: true},
		},
	}
}
func (c *ChangelogCommand) UsageShort() string { return "Retrieves the changelog for Sweetie Bot." }
