package sweetiebot

import (
	"math/rand"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type QuoteCommand struct {
}

func (c *QuoteCommand) Name() string {
	return "Quote"
}
func (c *QuoteCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		l := 0
		for _, v := range info.config.Quotes {
			l += len(v)
		}
		if l <= 0 {
			return "```There are no quotes.```", false
		}
		i := rand.Intn(l)

		for k, v := range info.config.Quotes {
			if i < len(v) {
				return "**" + getUserName(k, info) + "**: " + v[i], false
			}
			i -= len(v)
		}
		return "```Error: invalid random quote chosen???```", false
	}

	arg := strings.ToLower(args[0])
	IDs := FindUsername(arg)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	q, ok := info.config.Quotes[IDs[0]]
	l := len(q)
	if !ok || l <= 0 {
		return "```That user has no quotes.```", false
	}
	i := rand.Intn(l)
	if len(args) >= 2 {
		var err error
		i, err = strconv.Atoi(args[1])
		if err != nil {
			return "```Could not parse quote index. Make sure your username is in quotes.```", false
		}
		i--
		if i >= l || i < 0 {
			return "```Invalid quote index. Use !searchquote [user] to list a user's quotes and their indexes.```", false
		}
	}
	return "**" + IDsToUsernames(IDs, info)[0] + "**: " + q[i], false
}
func (c *QuoteCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[user] [quote index]", "If no arguments are specified, returns a random quote. If a user is specified, returns a random quote from that user. If a quote index is specified, returns that specific quote.")
}
func (c *QuoteCommand) UsageShort() string { return "Quotes a user." }

type AddQuoteCommand struct {
}

func (c *AddQuoteCommand) Name() string {
	return "AddQuote"
}
func (c *AddQuoteCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```Must specify username.```", false
	}
	if len(args) < 2 {
		return "```Can't add a blank quote!```", false
	}

	arg := strings.ToLower(args[0])
	IDs := FindUsername(arg)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	if len(info.config.Quotes) == 0 {
		info.config.Quotes = make(map[uint64][]string)
	}
	info.config.Quotes[IDs[0]] = append(info.config.Quotes[IDs[0]], strings.Join(args[1:], " "))
	info.SaveConfig()
	return "```Quote added to " + IDsToUsernames(IDs, info)[0] + ".```", false
}
func (c *AddQuoteCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[user] [quote]", "Adds a quote to the quote database for the given user. If the username has spaces, it must be in quotes. If the user is ambiguous, sweetiebot will return all possible matches.")
}
func (c *AddQuoteCommand) UsageShort() string { return "Adds a quote." }

type RemoveQuoteCommand struct {
}

func (c *RemoveQuoteCommand) Name() string {
	return "RemoveQuote"
}
func (c *RemoveQuoteCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```Must specify username.```", false
	}
	if len(args) < 2 {
		return "```Must specify quote index. Use !searchquote to list them.```", false
	}

	arg := strings.ToLower(args[0])
	index, err := strconv.Atoi(args[1])
	if err != nil {
		return "```Error: could not parse quote index. Did you surround your username with quotes?```", false
	}

	IDs := FindUsername(arg)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}

	index--
	if index >= len(info.config.Quotes[IDs[0]]) || index < 0 {
		return "```Invalid quote index. Use !searchquote [user] to list a user's quotes and their indexes.```", false
	}
	info.config.Quotes[IDs[0]] = append(info.config.Quotes[IDs[0]][:index], info.config.Quotes[IDs[0]][index+1:]...)
	info.SaveConfig()
	return "```Deleted quote #" + strconv.Itoa(index+1) + " from " + IDsToUsernames(IDs, info)[0] + ".```", false
}
func (c *RemoveQuoteCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[user] [quote index]", "Removes the quote with the given quote index from the user's set of quotes. If the username has spaces, it must be in quotes. If the user is ambiguous, sweetiebot will return all possible matches.")
}
func (c *RemoveQuoteCommand) UsageShort() string { return "Removes a quote." }

type SearchQuoteCommand struct {
}

func (c *SearchQuoteCommand) Name() string {
	return "SearchQuote"
}
func (c *SearchQuoteCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```Must specify username.```", false
	}

	arg := strings.ToLower(args[0])
	IDs := FindUsername(arg)
	if len(IDs) == 0 { // no matches!
		return "```Error: Could not find any usernames or aliases matching " + arg + "!```", false
	}
	if len(IDs) > 1 {
		return "```Could be any of the following users or their aliases:\n" + strings.Join(IDsToUsernames(IDs, info), "\n") + "```", len(IDs) > 5
	}
	l := len(info.config.Quotes[IDs[0]])
	if l == 0 {
		return "```That user has no quotes.```", false
	}
	quotes := make([]string, l, l)
	for i := 0; i < l; i++ {
		quotes[i] = strconv.Itoa(i+1) + ". " + info.config.Quotes[IDs[0]][i]
	}
	return "All quotes for " + IDsToUsernames(IDs, info)[0] + ":\n" + strings.Join(quotes, "\n"), l > 6
}
func (c *SearchQuoteCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[user]", "Lists all quotes for the given user.")
}
func (c *SearchQuoteCommand) UsageShort() string { return "Finds a quote." }
