package wittymodule

import (
	"math/rand"
	"regexp"
	"strings"
	"time"

	bot "../sweetiebot"
	"github.com/erikmcclure/discordgo"
)

// WittyModule is intended for any witty comments sweetie bot makes in response to what users say or do.
type WittyModule struct {
	lastdelete   int64
	lastcomment  int64
	wittyregex   *regexp.Regexp
	triggerregex []*regexp.Regexp
	remarks      [][]string
}

// New instance of WittyModule
func New(guild *bot.GuildInfo) *WittyModule {
	w := &WittyModule{lastcomment: 0, lastdelete: 0}
	w.UpdateRegex(guild)
	return w
}

// Name of the module
func (w *WittyModule) Name() string {
	return "Witty"
}

// Commands in the module
func (w *WittyModule) Commands() []bot.Command {
	return []bot.Command{
		&addWitCommand{w},
		&removeWitCommand{w},
	}
}

// Description of the module
func (w *WittyModule) Description(info *bot.GuildInfo) string {
	return "In response to certain patterns (determined by a regex) will post a response picked randomly from a list of them associated with that trigger. Rate limits itself to make sure it isn't too annoying."
}

// UpdateRegex updates the witty module regex
func (w *WittyModule) UpdateRegex(info *bot.GuildInfo) bool {
	l := len(info.Config.Witty.Responses)
	w.triggerregex = make([]*regexp.Regexp, 0, l)
	w.remarks = make([][]string, 0, l)
	if l < 1 {
		w.wittyregex = nil
		return true
	}

	var err error
	w.wittyregex, err = regexp.Compile("(" + strings.Join(bot.MapStringToSlice(info.Config.Witty.Responses), "|") + ")")

	if err == nil {
		var r *regexp.Regexp
		for k, v := range info.Config.Witty.Responses {
			r, err = regexp.Compile(k)
			if err != nil {
				break
			}
			w.triggerregex = append(w.triggerregex, r)
			w.remarks = append(w.remarks, strings.Split(v, "|"))
		}
	}

	if len(w.triggerregex) != len(w.remarks) { // This should never happen but we check just in case
		info.Log("ERROR! triggers do not equal remarks!!")
		return false
	}
	return err == nil
}

func (w *WittyModule) sendWittyComment(channelID string, comment string, timestamp time.Time, info *bot.GuildInfo) {
	if bot.RateLimit(&w.lastcomment, info.Config.Witty.Cooldown, timestamp.Unix()) {
		info.SendMessage(bot.DiscordChannel(channelID), comment)
	}
}

// OnMessageCreate discord hook
func (w *WittyModule) OnMessageCreate(info *bot.GuildInfo, m *discordgo.Message) {
	str := strings.ToLower(m.Content)
	timestamp := bot.GetTimestamp(m)
	if bot.CheckRateLimit(&w.lastcomment, info.Config.Witty.Cooldown, timestamp.Unix()) {
		if w.wittyregex != nil && w.wittyregex.MatchString(str) {
			for i := range w.triggerregex {
				if w.triggerregex[i].MatchString(str) {
					w.sendWittyComment(m.ChannelID, w.remarks[i][rand.Intn(len(w.remarks[i]))], timestamp, info)
					break
				}
			}
		}
	}
}

type addWitCommand struct {
	wit *WittyModule
}

func (c *addWitCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "AddWit",
		Usage:     "Adds a line to wittyremarks.",
		Sensitive: true,
	}
}

func witRemove(wit string, info *bot.GuildInfo) bool {
	wit = strings.ToLower(wit)
	_, ok := info.Config.Witty.Responses[wit]
	if ok {
		delete(info.Config.Witty.Responses, wit)
	}
	return ok
}

func (c *addWitCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 2 {
		return "```\nYou must provide both a trigger and a remark (both must be in quotes if they have spaces).```", false, nil
	}

	trigger := strings.ToLower(args[0])
	remark := args[1]

	bot.CheckMapNilString(&info.Config.Witty.Responses)
	info.Config.Witty.Responses[trigger] = remark
	info.SaveConfig()
	if !c.wit.UpdateRegex(info) {
		witRemove(trigger, info)
		c.wit.UpdateRegex(info)
		return "```\nFailed to add " + trigger + " because regex compilation failed.```", false, nil
	}
	return "```\nAdding " + trigger + " and recompiled the wittyremarks regex.```", false, nil
}
func (c *addWitCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Adds a `response` that is triggered by `trigger`.",
		Params: []bot.CommandUsageParam{
			{Name: "trigger", Desc: "Any valid regex string, but it must be in quotes if it has spaces.", Optional: false},
			{Name: "response", Desc: "All possible responses, split up by `|`. Also requires quotes if it has spaces.", Optional: false},
		},
	}
}

type removeWitCommand struct {
	wit *WittyModule
}

func (c *removeWitCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "RemoveWit",
		Usage:     "Removes a line from wittyremarks.",
		Sensitive: true,
	}
}

func (c *removeWitCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou must provide both a trigger to remove!```", false, nil
	}

	arg := strings.Join(args, " ")
	if !witRemove(arg, info) {
		return "```\nCould not find " + arg + "!```", false, nil
	}
	info.SaveConfig()
	c.wit.UpdateRegex(info)
	return "```\nRemoved " + arg + " and recompiled the wittyremarks regex.```", false, nil
}
func (c *removeWitCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Removes `trigger` from wittyremarks, provided it exists.",
		Params: []bot.CommandUsageParam{
			{Name: "trigger", Desc: "Any valid regex string.", Optional: false},
		},
	}
}
