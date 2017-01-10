package sweetiebot

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type MarkovModule struct {
}

func (w *MarkovModule) Name() string {
	return "Markov"
}

func (w *MarkovModule) Register(info *GuildInfo) {}

func (w *MarkovModule) Commands() []Command {
	return []Command{
		&EpisodeGenCommand{},
		&EpisodeQuoteCommand{},
		&ShipCommand{},
		&BestPonyCommand{},
	}
}

func (w *MarkovModule) Description() string { return "Generates content using markov chains." }

type EpisodeGenCommand struct {
	lock AtomicFlag
}

func (c *EpisodeGenCommand) Name() string {
	return "episodegen"
}
func (c *EpisodeGenCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if c.lock.test_and_set() {
		return "```Sorry, I'm busy processing another request right now. Please try again later!```", false, nil
	}
	defer c.lock.clear()
	maxlines := info.config.Markov.DefaultLines
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
	for i := 0; i < maxlines; i++ {
		if double {
			line, prev, prev2 = sb.db.GetMarkovLine2(prev, prev2)
		} else {
			line, prev = sb.db.GetMarkovLine(prev)
		}
		if len(line) > 0 {
			lines = append(lines, line)
		} else {
			maxlines++
		}
	}

	return strings.Join(lines, "\n"), len(lines) > 5 || !CheckShutup(msg.ChannelID), nil
}
func (c *EpisodeGenCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Randomly generates a my little pony episode using a markov chain, up to a maximum line count of `lines`. Will be sent via PM if the line count exceeds 5.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "lines", Desc: "Number of dialogue lines to generate", Optional: false},
			CommandUsageParam{Name: "single", Desc: "The markov chain uses double-lookback by default, if this is specified, will revert to single-lookback, which produces much more chaotic results.", Optional: false},
		},
	}
}
func (c *EpisodeGenCommand) UsageShort() string { return "Randomly generates episodes." }

type EpisodeQuoteCommand struct {
}

var quoteargregex = regexp.MustCompile("s[0-9]+e[0-9]+")

func (c *EpisodeQuoteCommand) Name() string {
	return "EpisodeQuote"
}
func (c *EpisodeQuoteCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !CheckShutup(msg.ChannelID) {
		return "", false, nil
	}
	S := 0
	E := 0
	L := 0
	diff := 0
	var lines []Transcript
	if len(args) < 1 {
		lines = []Transcript{sb.db.GetRandomQuote()}
	} else {
		arg := strings.ToLower(args[0])
		switch arg {
		case "action":
			lines = []Transcript{sb.db.GetCharacterQuote("ACTION")}
		case "speech":
			lines = []Transcript{sb.db.GetSpeechQuote()}
		default:
			if quoteargregex.MatchString(arg) {
				n, err := fmt.Sscanf(arg, "s%de%d:%d-%d", &S, &E, &L, &diff)
				if err != nil {
					if n < 3 {
						info.log.Log("quote scan error: ", err.Error())
						return "```Error: Could not parse your request. Be sure it is in the format S0E00:000-000. Example: S4E22:7-14```", false, nil
					}
					if n < 4 {
						diff = L
					}
				}
				diff--
				L--

				diff -= L
				if diff >= info.config.Markov.MaxLines {
					diff = info.config.Markov.MaxLines - 1
				}
				lines = sb.db.GetTranscript(S, E, L, L+diff)
			} else { // Otherwise this is a character quote request
				lines = []Transcript{sb.db.GetCharacterQuote(arg)}
				if lines[0].Season == 0 {
					return "```Error: Could not find character " + arg + " in the transcripts. Make sure you specify the entire name and spelled it correctly!```", false, nil
				}
			}
		}
	}

	process := make([]string, 0, len(lines))
	for _, v := range lines {
		l := ""
		if v.Speaker == "ACTION" {
			if v.Text != "" {
				l = "[" + v.Text + "]"
			}
		} else {
			l = "**" + v.Speaker + "**: " + v.Text
		}
		process = append(process, l)
	}
	return strings.Join(process, "\n"), len(process) > info.config.Markov.MaxPMlines, nil
}
func (c *EpisodeQuoteCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "If the S0E00:000-000 format is used, returns all the lines from the given season and episode, between the starting and ending line numbers (inclusive). Returns a maximum of " + strconv.Itoa(info.config.Markov.MaxLines) + " lines, but a line count above 5 will be sent in a private message. \n\nIf \"action\" is specified, returns a random action quote from the show.\n\nIf \"speech\" is specified, returns a random quote from one of the characters in the show.\n\nIf a \"Character Name\" is specified, it attempts to quote a random line from the show spoken by that character. If the character can't be found, returns an error. The character name doesn't have to be in quotes unless it has spaces in it, but you must specify the entire name.\n\nIf no arguments are specified, quotes a completely random line from the show.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "S0E00:000-000|action|speech|\"Character Name\"", Desc: "Example: `!quote S4E22:7-14`", Optional: true},
		},
	}
}
func (c *EpisodeQuoteCommand) UsageShort() string {
	return "Quotes random or specific lines from the show."
}

type ShipCommand struct {
}

func (c *ShipCommand) Name() string {
	return "ship"
}
func (c *ShipCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !CheckShutup(msg.ChannelID) {
		return "", false, nil
	}
	var a string
	var b string
	if info.config.Markov.UseMemberNames {
		a = sb.db.GetRandomMember(SBatoi(info.Guild.ID))
		b = sb.db.GetRandomMember(SBatoi(info.Guild.ID))
	} else {
		a = sb.db.GetRandomSpeaker()
		b = sb.db.GetRandomSpeaker()
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
		s = "%s and %s, sitting in a tree, K-I-S-S-- well, you know the rest."
	case 3:
		s = "%s falls head over hooves for %s."
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
		s = "%s and %s get REALLY drunk, and then-- wait, this channel is SFW."
	}

	return fmt.Sprintf("```\n"+s+"```", a, b), false, nil
}
func (c *ShipCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Generates a random pairing of ponies. If a first or second argument is supplied, uses those names instead.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "first", Desc: "The first name in the ship.", Optional: true},
			CommandUsageParam{Name: "second", Desc: "The second name in the ship.", Optional: true},
		},
	}
}
func (c *ShipCommand) UsageShort() string { return "Generates a random ship." }

type BestPonyCommand struct {
}

func (c *BestPonyCommand) Name() string {
	return "BestPony"
}
func (c *BestPonyCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !CheckShutup(msg.ChannelID) {
		return "", false, nil
	}
	a := strings.ToLower(sb.db.GetRandomWord())
	b := strings.ToLower(sb.db.GetRandomWord())
	s := ""
	switch rand.Int31n(3) {
	case 0:
		s = "%s %s is best pony."
	case 1:
		s = "%s %s is the bestest pony."
	case 2:
		s = "%s %s is the best pony."
	}

	return fmt.Sprintf("```\n"+s+"```", strings.ToUpper(a[:1])+a[1:], strings.ToUpper(b[:1])+b[1:]), false, nil
}
func (c *BestPonyCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{Desc: "Generates a random pony name."}
}
func (c *BestPonyCommand) UsageShort() string { return "Generates a random pony name." }
