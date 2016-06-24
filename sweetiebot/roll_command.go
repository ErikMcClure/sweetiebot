package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "strconv"
  "regexp"
  "math"
  "math/rand"
)

var diceregex = regexp.MustCompile("[0-9]*d[0-9]+")

type RollCommand struct {
}

func (c *RollCommand) Name() string {
  return "Roll";  
}
func (c *RollCommand) OpSplit(s string) []string {
  r := []string{};
  last := 0;
  for i, v := range s {
    switch v {
      case '+': fallthrough
      case '-': fallthrough
      case '*': fallthrough
      case '/': fallthrough
      case '(': fallthrough
      case ')': fallthrough
      case ',': fallthrough
      case '^': fallthrough
      case '&': fallthrough
      case '|':
        if last != i {
          r = append(r, s[last:i], s[i:i+1])
        } else {
          r = append(r, s[i:i+1])
        }
        last = i+1
    }
  }
  r = append(r, s[last:])
  return r
}
func (c *RollCommand) EatSymbols(args []string, index *int, s... string) int {
  if *index >= len(args) { return -1 }
  for i, v := range s {
    if args[*index] == v {
      *index++
      return i
    }
  }
  return -1
}
func (c *RollCommand) Eval(args []string, index *int) float64 {
  //sb.log.Log(strings.Join(args, "\u00B7"))
  var r float64
  if c.EatSymbols(args, index, "+", "-") == 1 {
    r = -c.Factor(args, index)
  } else {
    r = c.Factor(args, index)
  }
  
  for *index < len(args) {
    switch c.EatSymbols(args, index, "+", "-") {
      case 0:
        r += c.Factor(args, index)
      case 1:
        r -= c.Factor(args, index)
      case -1:
        return r
    }
  }

  return r
}
func (c *RollCommand) Factor(args []string, index *int) float64 {
  r := c.Term(args, index)
  for *index < len(args) {
    switch c.EatSymbols(args, index, "*", "/") {
      case 0:
        r *= c.Term(args, index)
      case 1:
        r /= c.Term(args, index)
      case -1:
        return r
    }
  }
  return r
}
func (c *RollCommand) Term(args []string, index *int) float64 {
  r := c.Bitwise(args, index)
  for *index < len(args) {
    switch c.EatSymbols(args, index, "^") {
      case 0:
        r = math.Pow(r, c.Bitwise(args, index))
      case -1:
        return r
    }
  }
  return r
}
func (c *RollCommand) Bitwise(args []string, index *int) float64 {
  r := c.Value(args, index)
  for *index < len(args) {
    switch c.EatSymbols(args, index, "&", "|") {
      case 0:
        r = float64(int64(r) & int64(c.Value(args, index)))
      case 1:
        r = float64(int64(r) | int64(c.Value(args, index)))
      case -1:
        return r
    }
  }
  return r
}
func (c *RollCommand) Eval1ArgFunc(args []string, index *int, fn func(float64)float64) float64 {
  *index++
  if c.EatSymbols(args, index, "(") == 0 {
    r := c.Eval(args, index)
    if c.EatSymbols(args, index, ")") != 0 {
      sb.log.Log("Expression missing ending ')': ", strings.Join(args, ""))
    }
    return fn(r)
  }
  sb.log.Log("Function has no parameters??? ", strings.Join(args, ""))
  return 0.0
}
func (c *RollCommand) Eval2ArgFunc(args []string, index *int, fn func(float64, float64)float64) float64 {
  *index++
  if c.EatSymbols(args, index, "(") == 0 {
    r := c.Eval(args, index)
    if c.EatSymbols(args, index, ",") != 0 {
      sb.log.Log("Expression missing second argument: ", strings.Join(args, ""))
      return 0.0
    }
    r2 := c.Eval(args, index)
    if c.EatSymbols(args, index, ")") != 0 {
      sb.log.Log("Expression missing ending ')': ", strings.Join(args, ""))
    }
    return fn(r, r2)
  }
  sb.log.Log("Function has no parameters??? ", strings.Join(args, ""))
  return 0.0
}
func (c *RollCommand) Value(args []string, index *int) float64 {
  if c.EatSymbols(args, index, "(") == 0 {
    r := c.Eval(args, index)
    if c.EatSymbols(args, index, ")") != 0 {
      sb.log.Log("Expression missing ending ')': ", strings.Join(args, ""))
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
      r = c.Eval1ArgFunc(args, index, math.Abs)
    case "acos":
      r = c.Eval1ArgFunc(args, index, math.Acos)
    case "acosh":
      r = c.Eval1ArgFunc(args, index, math.Acosh)
    case "asin":
      r = c.Eval1ArgFunc(args, index, math.Asin)
    case "asinh":
      r = c.Eval1ArgFunc(args, index, math.Asinh)
    case "atan":
      r = c.Eval1ArgFunc(args, index, math.Atan)
    case "atan2":
      r = c.Eval2ArgFunc(args, index, math.Atan2)
    case "atanh":
      r = c.Eval1ArgFunc(args, index, math.Atanh)
    case "cbrt":
      r = c.Eval1ArgFunc(args, index, math.Cbrt)
    case "ceil":
      r = c.Eval1ArgFunc(args, index, math.Ceil)
    case "cos":
      r = c.Eval1ArgFunc(args, index, math.Cos)
    case "cosh":
      r = c.Eval1ArgFunc(args, index, math.Cosh)
    case "erf":
      r = c.Eval1ArgFunc(args, index, math.Erf)
    case "exp":
      r = c.Eval1ArgFunc(args, index, math.Exp)
    case "floor":
      r = c.Eval1ArgFunc(args, index, math.Floor)
    case "gamma":
      r = c.Eval1ArgFunc(args, index, math.Gamma)
    case "log":
      r = c.Eval1ArgFunc(args, index, math.Log)
    case "log10":
      r = c.Eval1ArgFunc(args, index, math.Log10)
    case "log2":
      r = c.Eval1ArgFunc(args, index, math.Log2)
    case "min":
      r = c.Eval2ArgFunc(args, index, math.Min)
    case "max":
      r = c.Eval2ArgFunc(args, index, math.Max)
    case "mod":
      r = c.Eval2ArgFunc(args, index, math.Mod)
    case "pow":
      r = c.Eval2ArgFunc(args, index, math.Pow)
    case "remainder":
      r = c.Eval2ArgFunc(args, index, math.Remainder)
    case "sin":
      r = c.Eval1ArgFunc(args, index, math.Sin)
    case "sinh":
      r = c.Eval1ArgFunc(args, index, math.Sinh)
    case "sqrt":
      r = c.Eval1ArgFunc(args, index, math.Sqrt)
    case "tan":
      r = c.Eval1ArgFunc(args, index, math.Tan)
    case "tanh":
      r = c.Eval1ArgFunc(args, index, math.Tanh)
    case "trunc":
      r = c.Eval1ArgFunc(args, index, math.Trunc)
    default:
      a, err := strconv.ParseFloat(args[*index], 64)
      if err != nil {
        sb.log.Log("could not parse value: ", err.Error())
      }
      *index++
      r = a
  }

  return r
}
func (c *RollCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    return "```Nothing to roll or calculate!```", false
  }
  index := 0
  r := c.Eval(c.OpSplit(strings.Join(args, "")), &index)
  s := strconv.FormatFloat(r, 'f', -1, 64)  
  return "```" + s + "```", false
}
func (c *RollCommand) Usage() string { 
  return FormatUsage(c, "[expression]", "Evaluates an arbitrary mathematical expression, replacing all **N**d**X** values with the sum of n random numbers from 1 to **X**, inclusive. For example, !roll d10 will return 1-10, whereas !roll 2d10 + 2 will return a number between 4 and 22.") 
}
func (c *RollCommand) UsageShort() string { return "Evaluates a dice expression." } 