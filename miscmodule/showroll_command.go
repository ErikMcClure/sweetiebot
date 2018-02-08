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
		Name:              "Showroll",
		Usage:             "Evaluates a dice expression.",
		ServerIndependent: true,
	}
}
func (c *showrollCommand) eval(args []string, index *int, info *bot.GuildInfo) string {
	var s string

	for *index < len(args) {
		s += value(args, index) + "\n"
	}

	return s
}
func (c *showrollCommand) value(args []string, index *int, info *bot.GuildInfo) string {
	if diceregex.MatchString(args[*index]) {
		dice := strings.SplitN(args[*index], "d", 2)
		var multiplier int64
		var num int64
		var s string
		multiplier = 1
		num = 1
		if len(dice) > 1 {
			if len(dice[0]) > 0 {
				multiplier, _ = strconv.ParseInt(dice[0], 10, 64)
			}
			num, _ = strconv.ParseInt(dice[1], 10, 64)
		} else {
			num, _ = strconv.ParseInt(dice[0], 10, 64)
		}
		if num < 1 {
			return ""
		}
		if multiplier > 9999 {
			multiplier = 0
		}
		s = "Rolling " + args[*index] + ": "
		for ; multiplier > 0; multiplier-- {
			s += strconv.FormatInt(rand.Int63n(num)+1, 10)
			if multiplier > 1 {
				s += " + "
			}
		}
		*index++
		return s
	}
	panic("Could not parse dice expression. Try !calculate for advanced expressions")
}
func (c *showrollCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (retval string, b bool, embed *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nNothing to roll!```", false, nil
	}
	defer func() {
		if s := recover(); s != nil {
			retval = "```ERROR: " + s.(string) + "```"
		}
	}()
	index := 0
	s := c.eval(args, &index, info)
	return "```\n" + s + "```", false, nil
}
func (c *showrollCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Evaluates a dice roll expression (**N**d**X**), returning the individual die results. For example, `" + info.Config.Basic.CommandPrefix + "roll d10` will return 1-10, whereas `" + info.Config.Basic.CommandPrefix + "showroll 4d6` will return: `6 + 4 + 2 + 3`",
		Params: []bot.CommandUsageParam{
			{Name: "expression", Desc: "The dice expression to parse.", Optional: false},
		},
	}
}
