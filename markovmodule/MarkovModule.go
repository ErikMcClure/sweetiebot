package markovmodule

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	bot "../sweetiebot"
	"github.com/blackhole12/discordgo"
)

// MarkovModule generates content using markov chains
type MarkovModule struct {
}

// New MarkovModule
func New() *MarkovModule {
	return &MarkovModule{}
}

// Name of the module
func (w *MarkovModule) Name() string {
	return "Markov"
}

// Commands in the module
func (w *MarkovModule) Commands() []bot.Command {
	return []bot.Command{
		&episodeGenCommand{},
		&episodeQuoteCommand{},
		&shipCommand{},
	}
}

// Description of the module
func (w *MarkovModule) Description() string { return "Generates content using markov chains." }

type episodeGenCommand struct {
	lock bot.AtomicFlag
}

func (c *episodeGenCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "episodegen",
		Usage: "Randomly generates episodes.",
	}
}
func (c *episodeGenCommand) Name() string {
	return "episodegen"
}
func (c *episodeGenCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if c.lock.TestAndSet() {
		return "```\nSorry, I'm busy processing another request right now. Please try again later!```", false, nil
	}
	defer c.lock.Clear()
	maxlines := info.Config.Markov.DefaultLines
	double := true
	if len(args) > 0 {
		maxlines, _ = strconv.Atoi(args[0])
	}
	if len(args) > 1 {
		double = (strings.ToLower(args[1]) != "single")
	}
	if maxlines > 50 {
		maxlines = 50
	}
	if maxlines <= 0 {
		maxlines = 1
	}
	var prev uint64
	var prev2 uint64
	prev = 0
	prev2 = 0
	lines := make([]string, 0, maxlines)
	line := ""
	for i := 0; i < maxlines && info.Bot.DB.Status.Get(); i++ {
		if double {
			line, prev, prev2 = info.Bot.DB.GetMarkovLine2(prev, prev2)
		} else {
			line, prev = info.Bot.DB.GetMarkovLine(prev)
		}
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n"), len(lines) > bot.MaxPublicLines, nil
}
func (c *episodeGenCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Randomly generates a my little pony episode using a markov chain, up to a maximum line count of `lines`. Will be sent via PM if the line count exceeds 5.",
		Params: []bot.CommandUsageParam{
			{Name: "lines", Desc: "Number of dialogue lines to generate", Optional: true},
			{Name: "single", Desc: "The markov chain uses double-lookback by default, if this is specified, will revert to single-lookback, which produces much more chaotic results.", Optional: true},
		},
	}
}
func (c *episodeGenCommand) UsageShort() string { return "Randomly generates episodes." }

var quoteargregex = regexp.MustCompile("s[0-9]+e[0-9]+")

type episodeQuoteCommand struct {
}

func (c *episodeQuoteCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "EpisodeQuote",
		Usage: "Quotes random or specific lines from the show.",
	}
}
func (c *episodeQuoteCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	S := 0
	E := 0
	L := 0
	diff := 0
	var lines []bot.Transcript
	if len(args) < 1 {
		lines = []bot.Transcript{info.Bot.DB.GetRandomQuote()}
	} else {
		arg := strings.ToLower(args[0])
		switch arg {
		case "action":
			lines = []bot.Transcript{info.Bot.DB.GetCharacterQuote("")}
		case "speech":
			lines = []bot.Transcript{info.Bot.DB.GetSpeechQuote()}
		default:
			if quoteargregex.MatchString(arg) {
				n, err := fmt.Sscanf(arg, "s%de%d:%d-%d", &S, &E, &L, &diff)
				if err != nil {
					if n < 3 {
						return "```\nError: Could not parse your request. Be sure it is in the format S0E00:000-000. Example: S4E22:7-14```", false, nil
					}
					if n < 4 {
						diff = L
					}
				}
				diff--
				L--

				diff -= L
				if diff >= info.Config.Markov.MaxLines {
					diff = info.Config.Markov.MaxLines - 1
				}
				lines = info.Bot.DB.GetTranscript(S, E, L, L+diff)
			} else { // Otherwise this is a character quote request
				lines = []bot.Transcript{info.Bot.DB.GetCharacterQuote(arg)}
				if lines[0].Season == 0 {
					return "```\nError: Could not find character " + arg + " in the transcripts. Make sure you specify the entire name and spelled it correctly!```", false, nil
				}
			}
		}
	}

	process := make([]string, 0, len(lines))
	for _, v := range lines {
		l := ""
		if v.Speaker == "" {
			if v.Text != "" {
				l = "[" + v.Text + "]"
			}
		} else {
			l = "**" + v.Speaker + "**: " + v.Text
		}
		process = append(process, l)
	}
	return strings.Join(process, "\n"), len(process) > info.Config.Markov.MaxPMlines, nil
}
func (c *episodeQuoteCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "If the S0E00:000-000 format is used, returns all the lines from the given season and episode, between the starting and ending line numbers (inclusive). Returns a maximum of " + strconv.Itoa(info.Config.Markov.MaxLines) + " lines, but a line count above 5 will be sent in a private message. \n\nIf \"action\" is specified, returns a random action quote from the show.\n\nIf \"speech\" is specified, returns a random quote from one of the characters in the show.\n\nIf a \"Character Name\" is specified, it attempts to quote a random line from the show spoken by that character. If the character can't be found, returns an error. The character name doesn't have to be in quotes unless it has spaces in it, but you must specify the entire name.\n\nIf no arguments are specified, quotes a completely random line from the show.",
		Params: []bot.CommandUsageParam{
			{Name: "S0E00:000-000|action|speech|\"Character Name\"", Desc: "Example: `" + info.Config.Basic.CommandPrefix + "quote S4E22:7-14`", Optional: true},
		},
	}
}

type shipCommand struct {
}

func (c *shipCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "ship",
		Usage: "Generates a random ship.",
	}
}
func (c *shipCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	var a string
	var b string
	if info.Config.Markov.UseMemberNames {
		a = info.Bot.DB.GetRandomMember(bot.SBatoi(info.ID))
		b = info.Bot.DB.GetRandomMember(bot.SBatoi(info.ID))
	} else {
		a = info.Bot.DB.GetRandomSpeaker()
		b = info.Bot.DB.GetRandomSpeaker()
	}
	s := ""
	if len(args) > 0 {
		a = args[0]
	}
	if len(args) > 1 {
		b = args[1]
	}
	switch rand.Int31n(11) {
	case 0:
		s = "%s is %s's overly attached fianc\u00e9."
	case 1:
		s = "%s boops %s."
	case 2:
		s = "%s and %s, sitting in a tree, K-I-S-S— well, you know the rest."
	case 3:
		if info.Config.Markov.UseMemberNames {
			s = "%s falls head over heels for %s."
		} else {
			s = "%s falls head over hooves for %s."
		}
	case 4:
		s = "%s watches %s sleep at night."
	case 5:
		s = "%s is secretly in love with %s."
	case 6:
		s = "%s x %s"
	case 7:
		s = "%s and %s argue all the time and refuse to admit their feelings for each other."
	case 8:
		s = "%s makes %s's heart flutter."
	case 9:
		s = "%s stumbles on %s's darkest secret, only for it to bring them closer together."
	case 10:
		s = "%s and %s get REALLY drunk, and then—"
	}

	return fmt.Sprintf("```\n"+s+"```", a, b), false, nil
}
func (c *shipCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Generates a random pairing of ponies. If a first or second argument is supplied, uses those names instead.",
		Params: []bot.CommandUsageParam{
			{Name: "first", Desc: "The first name in the ship.", Optional: true},
			{Name: "second", Desc: "The second name in the ship.", Optional: true},
		},
	}
}
