package miscmodule

import (
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	bot "../sweetiebot"
	"github.com/erikmcclure/discordgo"
)

var diceregex = regexp.MustCompile("[0-9]*d[0-9]+")

type rollCommand struct {
}

func (c *rollCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:              "Roll",
		Usage:             "Evaluates a dice expression.",
		ServerIndependent: true,
	}
}
func (c *rollCommand) opSplit(s string) []string {
	r := []string{}
	last := 0
	for i, v := range s {
		switch v {
		case '+', '-', '*', '/', '(', ')', ',', '^', '&', '|':
			if last != i {
				r = append(r, s[last:i], s[i:i+1])
			} else {
				r = append(r, s[i:i+1])
			}
			last = i + 1
		}
	}
	r = append(r, s[last:])
	return r
}
func (c *rollCommand) eatSymbols(args []string, index *int, s ...string) int {
	if *index >= len(args) {
		return -1
	}
	for i, v := range s {
		if args[*index] == v {
			*index++
			return i
		}
	}
	return -1
}
func (c *rollCommand) eval(args []string, index *int, info *bot.GuildInfo) float64 {
	//info.Log(strings.Join(args, "\u00B7"))
	var r float64
	if c.eatSymbols(args, index, "+", "-") == 1 {
		r = -c.factor(args, index, info)
	} else {
		r = c.factor(args, index, info)
	}

	for *index < len(args) {
		switch c.eatSymbols(args, index, "+", "-") {
		case 0:
			r += c.factor(args, index, info)
		case 1:
			r -= c.factor(args, index, info)
		case -1:
			return r
		}
	}

	return r
}
func (c *rollCommand) factor(args []string, index *int, info *bot.GuildInfo) float64 {
	r := c.term(args, index, info)
	for *index < len(args) {
		switch c.eatSymbols(args, index, "*", "/") {
		case 0:
			r *= c.term(args, index, info)
		case 1:
			r /= c.term(args, index, info)
		case -1:
			return r
		}
	}
	return r
}
func (c *rollCommand) term(args []string, index *int, info *bot.GuildInfo) float64 {
	r := c.bitwise(args, index, info)
	for *index < len(args) {
		switch c.eatSymbols(args, index, "^") {
		case 0:
			r = math.Pow(r, c.bitwise(args, index, info))
		case -1:
			return r
		}
	}
	return r
}
func (c *rollCommand) bitwise(args []string, index *int, info *bot.GuildInfo) float64 {
	r := c.value(args, index, info)
	for *index < len(args) {
		switch c.eatSymbols(args, index, "&", "|") {
		case 0:
			r = float64(int64(r) & int64(c.value(args, index, info)))
		case 1:
			r = float64(int64(r) | int64(c.value(args, index, info)))
		case -1:
			return r
		}
	}
	return r
}
func (c *rollCommand) eval1ArgFunc(args []string, index *int, fn func(float64) float64, info *bot.GuildInfo) float64 {
	*index++
	if c.eatSymbols(args, index, "(") == 0 {
		r := c.eval(args, index, info)
		if c.eatSymbols(args, index, ")") != 0 {
			panic("Expression missing ending ')': " + strings.Join(args, ""))
		}
		return fn(r)
	}
	panic("Function has no parameters??? " + strings.Join(args, ""))
}
func (c *rollCommand) eval2ArgFunc(args []string, index *int, fn func(float64, float64) float64, info *bot.GuildInfo) float64 {
	*index++
	if c.eatSymbols(args, index, "(") == 0 {
		r := c.eval(args, index, info)
		if c.eatSymbols(args, index, ",") != 0 {
			panic("Expression missing second argument: " + strings.Join(args, ""))
		}
		r2 := c.eval(args, index, info)
		if c.eatSymbols(args, index, ")") != 0 {
			panic("Expression missing ending ')': " + strings.Join(args, ""))
		}
		return fn(r, r2)
	}
	panic("Function has no parameters??? " + strings.Join(args, ""))
}
func (c *rollCommand) value(args []string, index *int, info *bot.GuildInfo) float64 {
	if c.eatSymbols(args, index, "(") == 0 {
		r := c.eval(args, index, info)
		if c.eatSymbols(args, index, ")") != 0 {
			panic("Expression missing ending ')': " + strings.Join(args, ""))
		}
		return r
	}
	if *index >= len(args) {
		return 0.0
	}
	if diceregex.MatchString(args[*index]) {
		dice := strings.SplitN(args[*index], "d", 2)
		var multiplier int64
		var num int64
		var n int64
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
			return 0.0
		}
		if multiplier > 9999 {
			multiplier = 0
		}
		*index++
		n = 0
		for ; multiplier > 0; multiplier-- {
			n += rand.Int63n(num) + 1
		}
		return float64(n)
	}

	r := 0.0
	switch strings.ToLower(args[*index]) {
	case "abs":
		r = c.eval1ArgFunc(args, index, math.Abs, info)
	case "acos":
		r = c.eval1ArgFunc(args, index, math.Acos, info)
	case "acosh":
		r = c.eval1ArgFunc(args, index, math.Acosh, info)
	case "asin":
		r = c.eval1ArgFunc(args, index, math.Asin, info)
	case "asinh":
		r = c.eval1ArgFunc(args, index, math.Asinh, info)
	case "atan":
		r = c.eval1ArgFunc(args, index, math.Atan, info)
	case "atan2":
		r = c.eval2ArgFunc(args, index, math.Atan2, info)
	case "atanh":
		r = c.eval1ArgFunc(args, index, math.Atanh, info)
	case "cbrt":
		r = c.eval1ArgFunc(args, index, math.Cbrt, info)
	case "ceil":
		r = c.eval1ArgFunc(args, index, math.Ceil, info)
	case "cos":
		r = c.eval1ArgFunc(args, index, math.Cos, info)
	case "cosh":
		r = c.eval1ArgFunc(args, index, math.Cosh, info)
	case "erf":
		r = c.eval1ArgFunc(args, index, math.Erf, info)
	case "exp":
		r = c.eval1ArgFunc(args, index, math.Exp, info)
	case "floor":
		r = c.eval1ArgFunc(args, index, math.Floor, info)
	case "gamma":
		r = c.eval1ArgFunc(args, index, math.Gamma, info)
	case "log":
		r = c.eval1ArgFunc(args, index, math.Log, info)
	case "log10":
		r = c.eval1ArgFunc(args, index, math.Log10, info)
	case "log2":
		r = c.eval1ArgFunc(args, index, math.Log2, info)
	case "min":
		r = c.eval2ArgFunc(args, index, math.Min, info)
	case "max":
		r = c.eval2ArgFunc(args, index, math.Max, info)
	case "mod":
		r = c.eval2ArgFunc(args, index, math.Mod, info)
	case "pow":
		r = c.eval2ArgFunc(args, index, math.Pow, info)
	case "remainder":
		r = c.eval2ArgFunc(args, index, math.Remainder, info)
	case "sin":
		r = c.eval1ArgFunc(args, index, math.Sin, info)
	case "sinh":
		r = c.eval1ArgFunc(args, index, math.Sinh, info)
	case "sqrt":
		r = c.eval1ArgFunc(args, index, math.Sqrt, info)
	case "tan":
		r = c.eval1ArgFunc(args, index, math.Tan, info)
	case "tanh":
		r = c.eval1ArgFunc(args, index, math.Tanh, info)
	case "trunc":
		r = c.eval1ArgFunc(args, index, math.Trunc, info)
	case "pi":
		r = math.Pi
		*index++
	case "e":
		r = math.E
		*index++
	case "phi":
		r = math.Phi
		*index++
	default:
		a, err := strconv.ParseFloat(args[*index], 64)
		if err != nil {
			panic("could not parse value: " + err.Error())
		}
		*index++
		r = a
	}

	return r
}
func (c *rollCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (retval string, b bool, embed *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nNothing to roll or calculate!```", false, nil
	}
	defer func() {
		if r := recover(); r != nil {
			retval = "```ERROR: " + r.(string) + "```"
		}
	}()
	index := 0
	r := c.eval(c.opSplit(strings.Join(args, "")), &index, info)
	s := strconv.FormatFloat(r, 'f', -1, 64)
	return "```\n" + s + "```", false, nil
}
func (c *rollCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Evaluates an arbitrary mathematical expression, replacing all **N**d**X** values with the sum of `n` random numbers from 1 to **X**, inclusive. For example, `" + info.Config.Basic.CommandPrefix + "roll d10` will return 1-10, whereas `" + info.Config.Basic.CommandPrefix + "roll 2d10 + 2` will return a number between 4 and 22.",
		Params: []bot.CommandUsageParam{
			{Name: "expression", Desc: "The mathematical expression to parse.", Optional: false},
		},
	}
}
