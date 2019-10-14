package markovmodule

import (
	"fmt"
	"math"
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
func (w *MarkovModule) Description(info *bot.GuildInfo) string {
	return "Generates content using markov chains."
}

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

func getMarkovWord(info *bot.GuildInfo, prev uint32, prev2 uint32) (uint32, uint32, uint32) {
	p := uint64(prev2)<<32 | uint64(prev)
	if table, ok := info.Bot.Markov.Chain[p]; ok {
		total := 0
		for _, v := range table {
			total += v
		}

		total = rand.Intn(total)

		for k, v := range table {
			if total -= v; total < 0 {
				pair := info.Bot.Markov.Mapping[k]
				return uint32(pair >> 32), uint32(pair), k
			}
		}
	}

	return math.MaxUint32, math.MaxUint32, math.MaxUint32
}

func (c *episodeGenCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if c.lock.TestAndSet() {
		return "```\nSorry, I'm busy processing another request right now. Please try again later!```", false, nil
	}
	defer c.lock.Clear()
	if info.Bot.Markov == nil {
		return "```\nMarkov chain has not been created yet!```", false, nil
	}

	maxlines := info.Config.Markov.DefaultLines
	if len(args) > 0 {
		maxlines, _ = strconv.Atoi(args[0])
	}
	if maxlines > 50 {
		maxlines = 50
	}
	if maxlines <= 0 {
		maxlines = 1
	}
	var prev uint32
	var prev2 uint32
	prev = 0
	prev2 = 0
	lines := make([]string, 0, maxlines)

	for i := 0; i < maxlines; i++ {
		speakerID, wordID, next := getMarkovWord(info, prev, prev2)
		if wordID == math.MaxUint32 || speakerID == math.MaxUint32 {
			break
		}

		word := info.Bot.Markov.Phrases[wordID]
		speaker := info.Bot.Markov.Speakers[speakerID]
		prev2 = prev
		prev = next
		line := "[" + word

		if len(speaker) != 0 {
			line = "**" + speaker + ":** " + strings.ToUpper(word[:1]) + word[1:]
		}

		for max := 0; max < 300; max++ {
			capitalize := word == "." || word == "!" || word == "?"

			s, w, n := getMarkovWord(info, prev, prev2)
			if w == math.MaxUint32 || n == math.MaxUint32 || s != speakerID {
				prev = 0
				prev2 = 0
				break
			}
			prev2 = prev
			prev = n
			word := info.Bot.Markov.Phrases[w]

			switch word {
			case ".", "!", "?", ",":
				line += word
			default:
				if capitalize {
					line += " " + strings.ToUpper(word[:1]) + word[1:]
				} else {
					line += " " + word
				}
			}
		}

		if len(speaker) == 0 {
			line += "]"
		}

		if len(line) > 0 {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n"), len(lines) > bot.MaxPublicLines, nil
}
func (c *episodeGenCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Randomly generates a my little pony episode using a markov chain, up to a maximum line count of `lines`.",
		Params: []bot.CommandUsageParam{
			{Name: "lines", Desc: "Number of dialogue lines to generate", Optional: true},
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
	var a string
	var b string
	if info.Config.Markov.UseMemberNames {
		if g, err := info.GetGuild(); err != nil {
			return "```\nError: " + err.Error() + "```", false, nil
		} else {
			a = info.GetMemberName(g.Members[rand.Intn(len(g.Members))])
			b = info.GetMemberName(g.Members[rand.Intn(len(g.Members))])
		}
	} else if info.Bot.Markov != nil && len(info.Bot.Markov.Speakers) > 0 {
		a = info.Bot.Markov.Speakers[rand.Intn(len(info.Bot.Markov.Speakers))]
		b = info.Bot.Markov.Speakers[rand.Intn(len(info.Bot.Markov.Speakers))]
	} else {
		return "```\nNo speakers to choose from on Markov table!```", false, nil
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
