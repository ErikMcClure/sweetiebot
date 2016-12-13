package sweetiebot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type HelpCommand struct {
}

func (c *HelpCommand) Name() string {
	return "Help"
}
func (c *HelpCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) == 0 {
		s := []string{"Sweetie Bot knows the following commands. For more information on a specific command, type !help [command].\n"}
		commands := GetCommandsInOrder(info.commands)
		for _, v := range commands {
			s = append(s, v+": "+info.commands[v].UsageShort())
		}

		return "```" + strings.Join(s, "\n") + "```", true
	}
	v, ok := info.commands[strings.ToLower(args[0])]
	if !ok {
		return "``` Sweetie Bot doesn't recognize that command. You can check what commands Sweetie Bot knows by typing !help.```", false
	}
	return "```> !" + v.Name() + " " + v.Usage(info) + "```", true
}
func (c *HelpCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[command]", "Lists all available commands Sweetie Bot knows, or gives information about the given command. Of course, you should have figured this out by now, since you just typed !help help for some reason.")
}
func (c *HelpCommand) UsageShort() string {
	return "[PM Only] Generates the list you are looking at right now."
}

type AboutCommand struct {
}

func (c *AboutCommand) Name() string {
	return "About"
}
func (c *AboutCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
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
		Color: 0xa800ff,
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{Name: "Author", Value: "Blackhole#8270", Inline: true},
			&discordgo.MessageEmbedField{Name: "Library", Value: "discordgo", Inline: true},
			&discordgo.MessageEmbedField{Name: "Owner ID(s)", Value: strings.Join(owners, ", "), Inline: true},
			&discordgo.MessageEmbedField{Name: "Presence", Value: Pluralize(int64(len(sb.guilds)), " server"), Inline: true},
		},
	}
	info.SendEmbed(msg.ChannelID, embed)
	return "", false
}
func (c *AboutCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "", "Displays information about Sweetie Bot. What, did you think it would do something else?")
}
func (c *AboutCommand) UsageShort() string { return "Displays information about Sweetie Bot." }

type RulesCommand struct {
}

func (c *RulesCommand) Name() string {
	return "Rules"
}
func (c *RulesCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(info.config.Rules) == 0 {
		return "```I don't know what the rules are in this server... ¯\\_(ツ)_/¯```", false
	}
	if len(args) < 1 {
		rules := make([]string, 0, len(info.config.Rules)+1)
		rules = append(rules, "Official rules of "+info.Guild.Name+":")
		keys := MapIntToSlice(info.config.Rules)
		sort.Ints(keys)

		for _, v := range keys {
			rules = append(rules, fmt.Sprintf("%v. %s", v, info.config.Rules[v]))
		}
		return strings.Join(rules, "\n"), len(rules) > 4
	}

	arg, err := strconv.Atoi(args[0])
	if err != nil {
		return "```Rule index must be a number!```", false
	}
	rule, ok := info.config.Rules[arg]
	if !ok {
		return "```That's not a rule! Stop making things up!```", false
	}
	return fmt.Sprintf("%v. %s", arg, rule), false
}
func (c *RulesCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[index]", "Lists all the rules in this server, or displays the specific rule requested, if it exists. Rules can be set using \"!setconfig rules 1 this is a rule\"")
}
func (c *RulesCommand) UsageShort() string { return "Lists the rules of the server." }

type ChangelogCommand struct {
}

func (c *ChangelogCommand) Name() string {
	return "Changelog"
}
func (c *ChangelogCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
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
		return "```" + strings.Join(versions, "\n") + "```", len(versions) > 6
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
		return "```That's not a valid version of Sweetie Bot! Use this command with no arguments to list all valid versions, or use \"current\" to get the most recent changelog.```", false
	}
	return fmt.Sprintf("```%s\n--------\n%s```", v.String(), log), false
}
func (c *ChangelogCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[version]", "Displays the given changelog for Sweetie Bot. If no version is given, lists all versions with an associated changelog. Use \"current\" to get the changelog for the most recent version.")
}
func (c *ChangelogCommand) UsageShort() string { return "Retrieves the changelog for Sweetie Bot." }
