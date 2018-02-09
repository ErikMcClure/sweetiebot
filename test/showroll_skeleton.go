package main

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

// don't have a sweetiebot instance
// (or even go binary :P)
// testing this on https://go-sandbox.com/
var diceregex = regexp.MustCompile("[0-9]*d[0-9]+")

type showrollCommand struct {
}

func eval(args []string, index *int) string {
	var s string
	for *index < len(args) {
		s += value(args, index) + "\n"
	}
	return s
}
func value(args []string, index *int) string {
	*index++
	if diceregex.MatchString(args[*index-1]) {
		dice := strings.SplitN(args[*index-1], "d", 2)
		var multiplier, num, threshold, fail int64 = 1, 1, 0, 0
		var s string
		s = "Rolling " + args[*index-1] + ": "
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
					return s + "Can't roll that! Check dice expression."
				}
			}
			if strings.Contains(dice[1], "f") {
				fdice := strings.SplitN(dice[1], "f", 2)
				dice[1] = fdice[0]
				fail, _ = strconv.ParseInt(fdice[1], 10, 64)
				if fail == 0 {
					return s + "Can't roll that! Check dice expression."
				}
			}
			num, _ = strconv.ParseInt(dice[1], 10, 64)
		} else {
			num, _ = strconv.ParseInt(dice[0], 10, 64)
		}
		if multiplier < 1 || num < 1 {
			return s + "Can't roll that! Check dice expression."
		}
		if multiplier > 9999 {
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
	return "Could not parse dice expression. Try !calculate for advanced expressions."
}
func Process(args []string) (retval string) {
	if len(args) < 1 {
		return "```\nNothing to roll!```"
	}
	defer func() {
		if s := recover(); s != nil {
			retval = "```ERROR: " + s.(string) + "```"
		}
	}()
	index := 0
	s := eval(args, &index)
	return "```\n" + s + "```"
}
func main() {
	var argsGood []string
	rand.Seed(2626)
	argsGood = append(argsGood, "6d6", "12d20", "17d6", "d6")
	println(Process(argsGood))

	var argsExtra []string
	rand.Seed(656)
	argsExtra = append(argsExtra, "12d6t5f1", "8d20f5", "6d10t5", "6d10f1t5")
	println(Process(argsExtra))

	var argsBad []string
	rand.Seed(683)
	argsBad = append(argsBad, "potato", "0d1", "-8d7", "1d0", "6d-6", "6d", "10000d6", "6d10fish", "6d10tx")
	println(Process(argsBad))
}

// dice expression (stuff in [] are optional):
// [x]dx[tx][fx]
// where:
//	x: number of dice to roll (postive integer < 10000; optional, defaults to 1)
//	dx: the type of dice to roll, where x is the number of sides (required)
//	tx: the threshold to use for hit counting, (x is postive integer < 10000; optional)
//	fx: the fail threshold to use for hit counting, (x is postive integer < 10000; optional)
