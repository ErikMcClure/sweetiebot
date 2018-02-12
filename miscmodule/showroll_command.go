package miscmodule

import (
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	bot "../sweetiebot"
	"github.com/blackhole12/discordgo"
)

var diceregex = regexp.MustCompile("[0-9]*d[0-9]+")

type showrollCommand struct {
}

func (c *showrollCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:              "ShowRoll",
		Usage:             "Evaluates a dice expression, returning indivual dice results.",
		ServerIndependent: true,
	}
}

func (c *showrollCommand) value(args []string, index *int, prefix *bot.GuildInfo.Config.Basic.CommandPrefix) string {
	*index++
	string errmsg := "I can't figure out your dice expression... Try " + prefix + "help showroll for more information."
	if diceregex.MatchString(args[*index-1]) {
		dice := strings.SplitN(args[*index-1], "d", 2)
		var multiplier, num, threshold, fail int64 = 1, 1, 0, 0
		s := "Rolling " + args[*index-1] + ": "
		if len(dice) > 1 {
			if len(dice[0]) > 0 {
				multiplier, _ = strconv.ParseInt(dice[0], 10, 64)
			}
			if strings.Contains(dice[1], "t") {
				tdice := strings.SplitN(dice[1], "t", 2)
				dice[1] = tdice[0]
				if strings.Contains(tdice[1], "f") {
					fdice := strings.SplitN(tdice[1], "f", 2)
					threshold, _ = strconv.ParseInt(fdice[0], 10, 64)
					fail, _ = strconv.ParseInt(fdice[1], 10, 64)
				} else {
					threshold, _ = strconv.ParseInt(tdice[1], 10, 64)
				}
				if threshold == 0 {
					return s + errsmg
				}
			}
			if strings.Contains(dice[1], "f") {
				fdice := strings.SplitN(dice[1], "f", 2)
				dice[1] = fdice[0]
				fail, _ = strconv.ParseInt(fdice[1], 10, 64)
				if fail == 0 {
					return s + errmsg
				}
			}
			num, _ = strconv.ParseInt(dice[1], 10, 64)
		} else {
			num, _ = strconv.ParseInt(dice[0], 10, 64)
		}
		if fail < 0 || fail > num {
			return s + "That's a silly fail threshold, filly!"
		}
		if threshold < 0 || threshold > num {
			return s + "That's a silly success threshold, filly!"
		}
		if multiplier < 1 || num < 1 {
			return s + errmsg
		}
		if multiplier > 250 {
			return s + "I don't have that many dice..."
		}
		var n int64
		var t int = 0
		var f int = 0
		for ; multiplier > 0; multiplier-- {
			n = rand.Int63n(num) + 1
			s += strconv.FormatInt(n, 10)
			if multiplier > 1 {
				s += " + "
			}
			if threshold > 0 {
				if n >= threshold {
					t++
				}
			}
			if fail > 0 {
				if n <= fail {
					f++
				}
			}
		}
		if t > 0 {
			s += "\n" + strconv.Itoa(t) + " successes!"
		}
		if f > 0 {
			s += "\n" + strconv.Itoa(f) + " failures!"
		}
		return s
	}
	return errmsg
}

func (c *showrollCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (retval string, b bool, embed *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nNothing to roll...```", false, nil
	}
	index := 0
	var s string
	for *index < len(args) {
		s += value(args, index, info.Config.Basic.CommandPrefix) + "\n"
	}
	return "```\n" + s + "```", false, nil
}

func (c *showrollCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: `Evaluates a dice roll expression, returning the individual die results. Can also optionally report hit counting for success and fail thresholds.\n
Acceptable expressions are defined as [**N**]d**X**[t**Y**][f**Z**] where:\n
N: number of dice to roll (postive integer < 250; optional, defaults to 1)\n
dX: the type of dice to roll, where X is the number of sides (required)\n
tY: the threshold to use for hit counting, (x is postive integer; optional)\n
fZ: the fail threshold to use for hit counting, (x is postive integer; optional)\n
\n
Examples:\n
d6: Rolls a single 6-sided die\n
4d20: Rolls 4 20-sided dice\n
12d6t5: Rolls 12 6-sided dice, and counts the number that score 5 or higher\n
17d10t8f2: Rolls 17 10-sided dice, counts number that roll 8 or higher (successes) and 2 or lower (fails)`,
		Params: []bot.CommandUsageParam{
			{Name: "expression", Desc: "The dice expression to parse (e.g. `12d6t5f1`; see command description for more details).", Optional: false},
		},
	}
}
