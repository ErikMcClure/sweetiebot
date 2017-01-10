package sweetiebot

import (
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var diceregex = regexp.MustCompile("[0-9]*d[0-9]+")

type RollCommand struct {
}

func (c *RollCommand) Name() string {
	return "Roll"
}
func (c *RollCommand) OpSplit(s string) []string {
	r := []string{}
	last := 0
	for i, v := range s {
		switch v {
		case '+':
			fallthrough
		case '-':
			fallthrough
		case '*':
			fallthrough
		case '/':
			fallthrough
		case '(':
			fallthrough
		case ')':
			fallthrough
		case ',':
			fallthrough
		case '^':
			fallthrough
		case '&':
			fallthrough
		case '|':
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
func (c *RollCommand) EatSymbols(args []string, index *int, s ...string) int {
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
func (c *RollCommand) Eval(args []string, index *int, info *GuildInfo) float64 {
	//info.log.Log(strings.Join(args, "\u00B7"))
	var r float64
	if c.EatSymbols(args, index, "+", "-") == 1 {
		r = -c.Factor(args, index, info)
	} else {
		r = c.Factor(args, index, info)
	}

	for *index < len(args) {
		switch c.EatSymbols(args, index, "+", "-") {
		case 0:
			r += c.Factor(args, index, info)
		case 1:
			r -= c.Factor(args, index, info)
		case -1:
			return r
		}
	}

	return r
}
func (c *RollCommand) Factor(args []string, index *int, info *GuildInfo) float64 {
	r := c.Term(args, index, info)
	for *index < len(args) {
		switch c.EatSymbols(args, index, "*", "/") {
		case 0:
			r *= c.Term(args, index, info)
		case 1:
			r /= c.Term(args, index, info)
		case -1:
			return r
		}
	}
	return r
}
func (c *RollCommand) Term(args []string, index *int, info *GuildInfo) float64 {
	r := c.Bitwise(args, index, info)
	for *index < len(args) {
		switch c.EatSymbols(args, index, "^") {
		case 0:
			r = math.Pow(r, c.Bitwise(args, index, info))
		case -1:
			return r
		}
	}
	return r
}
func (c *RollCommand) Bitwise(args []string, index *int, info *GuildInfo) float64 {
	r := c.Value(args, index, info)
	for *index < len(args) {
		switch c.EatSymbols(args, index, "&", "|") {
		case 0:
			r = float64(int64(r) & int64(c.Value(args, index, info)))
		case 1:
			r = float64(int64(r) | int64(c.Value(args, index, info)))
		case -1:
			return r
		}
	}
	return r
}
func (c *RollCommand) Eval1ArgFunc(args []string, index *int, fn func(float64) float64, info *GuildInfo) float64 {
	*index++
	if c.EatSymbols(args, index, "(") == 0 {
		r := c.Eval(args, index, info)
		if c.EatSymbols(args, index, ")") != 0 {
			info.log.Log("Expression missing ending ')': ", strings.Join(args, ""))
		}
		return fn(r)
	}
	info.log.Log("Function has no parameters??? ", strings.Join(args, ""))
	return 0.0
}
func (c *RollCommand) Eval2ArgFunc(args []string, index *int, fn func(float64, float64) float64, info *GuildInfo) float64 {
	*index++
	if c.EatSymbols(args, index, "(") == 0 {
		r := c.Eval(args, index, info)
		if c.EatSymbols(args, index, ",") != 0 {
			info.log.Log("Expression missing second argument: ", strings.Join(args, ""))
			return 0.0
		}
		r2 := c.Eval(args, index, info)
		if c.EatSymbols(args, index, ")") != 0 {
			info.log.Log("Expression missing ending ')': ", strings.Join(args, ""))
		}
		return fn(r, r2)
	}
	info.log.Log("Function has no parameters??? ", strings.Join(args, ""))
	return 0.0
}
func (c *RollCommand) Value(args []string, index *int, info *GuildInfo) float64 {
	if c.EatSymbols(args, index, "(") == 0 {
		r := c.Eval(args, index, info)
		if c.EatSymbols(args, index, ")") != 0 {
			info.log.Log("Expression missing ending ')': ", strings.Join(args, ""))
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
		r = c.Eval1ArgFunc(args, index, math.Abs, info)
	case "acos":
		r = c.Eval1ArgFunc(args, index, math.Acos, info)
	case "acosh":
		r = c.Eval1ArgFunc(args, index, math.Acosh, info)
	case "asin":
		r = c.Eval1ArgFunc(args, index, math.Asin, info)
	case "asinh":
		r = c.Eval1ArgFunc(args, index, math.Asinh, info)
	case "atan":
		r = c.Eval1ArgFunc(args, index, math.Atan, info)
	case "atan2":
		r = c.Eval2ArgFunc(args, index, math.Atan2, info)
	case "atanh":
		r = c.Eval1ArgFunc(args, index, math.Atanh, info)
	case "cbrt":
		r = c.Eval1ArgFunc(args, index, math.Cbrt, info)
	case "ceil":
		r = c.Eval1ArgFunc(args, index, math.Ceil, info)
	case "cos":
		r = c.Eval1ArgFunc(args, index, math.Cos, info)
	case "cosh":
		r = c.Eval1ArgFunc(args, index, math.Cosh, info)
	case "erf":
		r = c.Eval1ArgFunc(args, index, math.Erf, info)
	case "exp":
		r = c.Eval1ArgFunc(args, index, math.Exp, info)
	case "floor":
		r = c.Eval1ArgFunc(args, index, math.Floor, info)
	case "gamma":
		r = c.Eval1ArgFunc(args, index, math.Gamma, info)
	case "log":
		r = c.Eval1ArgFunc(args, index, math.Log, info)
	case "log10":
		r = c.Eval1ArgFunc(args, index, math.Log10, info)
	case "log2":
		r = c.Eval1ArgFunc(args, index, math.Log2, info)
	case "min":
		r = c.Eval2ArgFunc(args, index, math.Min, info)
	case "max":
		r = c.Eval2ArgFunc(args, index, math.Max, info)
	case "mod":
		r = c.Eval2ArgFunc(args, index, math.Mod, info)
	case "pow":
		r = c.Eval2ArgFunc(args, index, math.Pow, info)
	case "remainder":
		r = c.Eval2ArgFunc(args, index, math.Remainder, info)
	case "sin":
		r = c.Eval1ArgFunc(args, index, math.Sin, info)
	case "sinh":
		r = c.Eval1ArgFunc(args, index, math.Sinh, info)
	case "sqrt":
		r = c.Eval1ArgFunc(args, index, math.Sqrt, info)
	case "tan":
		r = c.Eval1ArgFunc(args, index, math.Tan, info)
	case "tanh":
		r = c.Eval1ArgFunc(args, index, math.Tanh, info)
	case "trunc":
		r = c.Eval1ArgFunc(args, index, math.Trunc, info)
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
			info.log.Log("could not parse value: ", err.Error())
		}
		*index++
		r = a
	}

	return r
}
func (c *RollCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```Nothing to roll or calculate!```", false, nil
	}
	index := 0
	r := c.Eval(c.OpSplit(strings.Join(args, "")), &index, info)
	s := strconv.FormatFloat(r, 'f', -1, 64)
	return "```\n" + s + "```", false, nil
}
func (c *RollCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Evaluates an arbitrary mathematical expression, replacing all **N**d**X** values with the sum of `n` random numbers from 1 to **X**, inclusive. For example, `!roll d10` will return 1-10, whereas `!roll 2d10 + 2` will return a number between 4 and 22.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "expression", Desc: "The mathematical expression to parse.", Optional: false},
		},
	}
}
func (c *RollCommand) UsageShort() string { return "Evaluates a dice expression." }
