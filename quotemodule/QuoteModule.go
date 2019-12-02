package quotemodule

import (
	"math"
	"math/rand"
	"strconv"
	"strings"

	bot "../sweetiebot"
	"github.com/erikmcclure/discordgo"
)

// QuoteModule manages the quoting system
type QuoteModule struct {
}

// New QuoteModule
func New() *QuoteModule {
	return &QuoteModule{}
}

// Name of the module
func (w *QuoteModule) Name() string {
	return "Quotes"
}

// Commands in the module
func (w *QuoteModule) Commands() []bot.Command {
	return []bot.Command{
		&quoteCommand{},
		&addquoteCommand{},
		&removequoteCommand{},
		&searchQuoteCommand{},
	}
}

// Description of the module
func (w *QuoteModule) Description(info *bot.GuildInfo) string {
	return "Manages a database of quotes attributed to a specific user ID. These quotes will persist if the user leaves the server."
}

type quoteCommand struct {
}

func (c *quoteCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "Quote",
		Usage: "Quotes a user.",
	}
}
func (c *quoteCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		l := 0
		for _, v := range info.Config.Quote.Quotes {
			l += len(v)
		}
		if l <= 0 {
			return "```\nThere are no quotes.```", false, nil
		}
		i := rand.Intn(l)

		for k, v := range info.Config.Quote.Quotes {
			if i < len(v) {
				return "**" + info.GetUserName(k) + "**: " + v[i], false, nil
			}
			i -= len(v)
		}
		return "```\nError: invalid random quote chosen???```", false, nil
	}
	last := 1
	if len(args) > 1 {
		last = len(args) - 1
	}
	i := -1
	if len(args) > 1 {
		var err error
		if i, err = strconv.Atoi(args[last]); err != nil {
			last++
		} else if i <= 0 {
			i = math.MaxInt32
		}
		i--
	}
	user, err := bot.ParseUser(strings.Join(args[:last], " "), info)
	if err != nil {
		return bot.ReturnError(err)
	}
	q, ok := info.Config.Quote.Quotes[user]
	l := len(q)
	if !ok || l <= 0 {
		return "```\nThat user has no quotes.```", false, nil
	}
	if i < 0 {
		i = rand.Intn(l)
	}
	if i >= l || i < 0 {
		return "```\nInvalid quote index. Use " + info.Config.Basic.CommandPrefix + "searchquote [user] to list a user's quotes and their indexes.```", false, nil
	}
	return "**" + info.GetUserName(user) + "**: " + q[i], false, nil
}
func (c *quoteCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "If no arguments are specified, returns a random quote. If a user is specified, returns a random quote from that user. If a quote index is specified, returns that specific quote.",
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: "A @user ping or simply the name of the user to quote.", Optional: true},
			{Name: "quote", Desc: "A specific quote index. Use `" + info.Config.Basic.CommandPrefix + "searchquote` to find a quote index.", Optional: true},
		},
	}
}

type addquoteCommand struct {
}

func (c *addquoteCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "AddQuote",
		Usage:     "Adds a quote.",
		Sensitive: true,
	}
}

func (c *addquoteCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nMust specify username.```", false, nil
	}
	if len(args) < 2 {
		return "```\nCan't add a blank quote!```", false, nil
	}

	user, err := bot.ParseUser(args[0], info)
	if err != nil {
		return bot.ReturnError(err)
	}

	if len(info.Config.Quote.Quotes) == 0 {
		info.Config.Quote.Quotes = make(map[bot.DiscordUser][]string)
	}
	info.Config.Quote.Quotes[user] = append(info.Config.Quote.Quotes[user], msg.Content[indices[1]:])
	info.SaveConfig()
	return "```\nQuote added to " + info.GetUserName(user) + ".```", false, nil
}
func (c *addquoteCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Adds a quote to the quote database for the given user. If the user is ambiguous, returns all possible matches.",
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: "A @user ping or simply the name of the user to quote. If the username has spaces, it must be in quotes.", Optional: false},
			{Name: "quote", Desc: "A specific quote index. Use `" + info.Config.Basic.CommandPrefix + "searchquote` to find a quote index.", Optional: false},
		},
	}
}

type removequoteCommand struct {
}

func (c *removequoteCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "RemoveQuote",
		Usage:     "Removes a quote.",
		Sensitive: true,
	}
}
func (c *removequoteCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nMust specify username.```", false, nil
	}
	if len(args) < 2 {
		return "```\nMust specify quote index. Use " + info.Config.Basic.CommandPrefix + "searchquote to list them.```", false, nil
	}

	last := len(args) - 1
	user, err := bot.ParseUser(strings.Join(args[:last], " "), info)
	if err != nil {
		return bot.ReturnError(err)
	}
	index, err := strconv.Atoi(args[last])
	if err != nil {
		return "```\nError: could not parse quote index. Did you surround your username with quotes? Use " + info.Config.Basic.CommandPrefix + "searchquote to find a quote index.```", false, nil
	}

	index--
	if index >= len(info.Config.Quote.Quotes[user]) || index < 0 {
		return "```\nInvalid quote index. Use " + info.Config.Basic.CommandPrefix + "searchquote [user] to list a user's quotes and their indexes.```", false, nil
	}
	info.Config.Quote.Quotes[user] = append(info.Config.Quote.Quotes[user][:index], info.Config.Quote.Quotes[user][index+1:]...)
	info.SaveConfig()
	return "```\nDeleted quote #" + strconv.Itoa(index+1) + " from " + info.GetUserName(user) + ".```", false, nil
}
func (c *removequoteCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Removes the quote with the given quote index from the user's set of quotes. If the user is ambiguous, returns all possible matches.",
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: "A @user ping or simply the name of the user to quote. If the username has spaces, it must be in quotes.", Optional: false},
			{Name: "quote", Desc: "A specific quote index. Use `" + info.Config.Basic.CommandPrefix + "searchquote` to find a quote index.", Optional: false},
		},
	}
}

type searchQuoteCommand struct {
}

func (c *searchQuoteCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "SearchQuote",
		Usage: "Finds a quote.",
	}
}
func (c *searchQuoteCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		s := make([]uint64, 0, len(info.Config.Quote.Quotes))
		for k, v := range info.Config.Quote.Quotes {
			if len(v) > 0 { // Map entries can have 0 quotes associated with them
				s = append(s, k.Convert())
			}
		}
		return "```\nThe following users have at least one quote:\n" + strings.Join(info.IDsToUsernames(s, true), "\n") + "```", len(s) > bot.MaxPublicLines, nil
	}

	user, err := bot.ParseUser(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(err)
	}

	l := len(info.Config.Quote.Quotes[user])
	if l == 0 {
		return "```\nThat user has no quotes.```", false, nil
	}
	quotes := make([]string, l, l)
	for i := 0; i < l; i++ {
		quotes[i] = strconv.Itoa(i+1) + ". " + info.Config.Quote.Quotes[user][i]
	}
	return "All quotes for " + info.GetUserName(user) + ":\n" + strings.Join(quotes, "\n"), l > bot.MaxPublicLines, nil
}
func (c *searchQuoteCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Lists all quotes for the given user.",
		Params: []bot.CommandUsageParam{
			{Name: "user", Desc: "A @user ping or simply the name of the user to quote. If the username has spaces, it must be in quotes.", Optional: false},
		},
	}
}
