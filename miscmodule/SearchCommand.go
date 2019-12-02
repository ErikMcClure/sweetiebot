package miscmodule

import (
	"database/sql"
	"strconv"
	"strings"

	bot "../sweetiebot"
	"github.com/erikmcclure/discordgo"
)

type searchCommand struct {
	lock       bot.AtomicFlag
	statements map[string][]*sql.Stmt
}

func (c *searchCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "Search",
		Usage:     "Performs a complex search on the chat history.",
		Silver:    true,
		Sensitive: true,
	}
}
func msgHighlightMatch(msg string, match string) string {
	if len(match) == 0 {
		return msg
	}
	msg = strings.Replace(msg, "**"+match+"**", match, -1) // this trick helps prevent increasing ** being appended repeatedly.
	msg = strings.Replace(msg, "**"+match, match, -1)      // helps prevent ** from exploding everywhere because discord is bad at isolation.
	return strings.Replace(msg, match, "**"+match+"**", -1)
}
func (c *searchCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if c.lock.TestAndSet() {
		return "```\nSorry, I'm busy processing another request right now. Please try again later!```", false, nil
	}
	defer c.lock.Clear()
	rangebegin := 0
	rangeend := 5
	users := make([]string, 0, 0)
	userIDs := make([]uint64, 0, 0)
	channels := make([]uint64, 0, 0)
	timestamp := bot.GetTimestamp(msg)
	t := timestamp.AddDate(0, 0, 1)
	messages := make([]string, 0, 0)

	// Fill in parameters from args
	for _, v := range args {
		if len(v) > 0 {
			switch {
			case v[0] == '*':
				if len(v) < 2 {
					rangeend = -1
				} else {
					s := strings.Split(v[1:], "-")
					if len(s) > 1 {
						rangebegin, _ = strconv.Atoi(s[0])
						rangeend, _ = strconv.Atoi(s[1])
					} else {
						rangeend, _ = strconv.Atoi(s[0])
					}
				}
			case v[0] == '@' || (len(v) > 1 && v[0] == '<' && v[1] == '@'):
				if len(v) < 2 {
					return "```\nError: No users specified```", false, nil
				}
				users = strings.Split(v, "|")
			case v[0] == '#':
				s := strings.Split(v, "|")
				g, _ := info.GetGuild()
				for _, c := range s {
					ch, err := bot.ParseChannel(c, g)
					if err != nil {
						return bot.ReturnError(err)
					}
					channels = append(channels, ch.Convert())
				}
			case (len(v) > 1 && v[0] == '<' && v[1] == '#'):
				if len(v) < 2 {
					return "```\nError: No channels specified```", false, nil
				}
				s := strings.Split(v, "|")
				for _, c := range s {
					if !bot.ChannelRegex.MatchString(c) {
						return "```\nError: Unknown channel format " + c + " - Must be an actual recognized channel by discord!```", false, nil
					}
					channels = append(channels, bot.SBatoi(c[2:len(c)-1]))
				}
			case v[0] == '~':
				var err error
				t, err = info.ParseCommonTime(v[1:], bot.DiscordUser(msg.Author.ID), bot.GetTimestamp(msg))
				if err != nil {
					return bot.ReturnError(err)
				}
				t = t.UTC()
			default:
				messages = append(messages, v)
			}
		}
	}

	// Resolve usernames that aren't IDs to IDs
	for _, v := range users {
		v = strings.TrimSpace(v)
		if bot.UserRegex.MatchString(v) {
			userIDs = append(userIDs, bot.SBatoi(v[2:len(v)-1]))
		} else if len(v) > 0 {
			IDs := []uint64{}
			IDs = info.FindUsername(v[1:])
			if len(IDs) == 0 { // we failed to resolve this username, so return an error.
				return "```\nError: Could not find any usernames or aliases matching " + v[1:] + "!```", false, nil
			}
			userIDs = append(userIDs, IDs...)
		}
	}

	// If we have no searchable arguments, fail
	if len(messages)+len(userIDs)+len(channels) == 0 {
		return "```\nError: no searchable terms specified! You must have either a message, a user, or a channel.```", false, nil
	}

	// Assemble query string and parameter list
	params := make([]interface{}, 0, 3)
	params = append(params, bot.SBatoi(info.ID))
	query := ""

	if len(userIDs) > 0 {
		temp := make([]string, 0, len(userIDs))
		for _, v := range userIDs {
			temp = append(temp, "C.Author = ?")
			params = append(params, v)
		}
		query += "(" + strings.Join(temp, " OR ") + ") AND "
	}

	if len(channels) > 0 {
		temp := make([]string, 0, len(channels))
		for _, v := range channels {
			temp = append(temp, "C.Channel = ?")
			params = append(params, v)
		}
		query += "(" + strings.Join(temp, " OR ") + ") AND "
	}

	message := strings.Join(messages, " ")
	if len(messages) > 0 {
		query += "C.Message LIKE ? AND "
		params = append(params, "%"+message+"%")
	}

	if t.Before(timestamp) {
		query += "C.Timestamp < ? AND "
		params = append(params, t)
	}

	query += "C.ID != ? AND C.Author != ? AND C.Channel != ? AND C.Message NOT LIKE ? ORDER BY C.Timestamp DESC" // Always exclude the message corresponding to the command and all sweetie bot messages (which also prevents trailing ANDs)
	params = append(params, bot.SBatoi(msg.ID))
	params = append(params, info.Bot.SelfID.Convert())
	params = append(params, info.Config.Basic.ModChannel.Convert())
	params = append(params, info.Config.Basic.CommandPrefix + "search %")

	querylimit := query
	if rangeend >= 0 {
		querylimit += " LIMIT ?"
		if rangebegin > 0 {
			querylimit += " OFFSET ?"
		}
	}

	// if not cached, prepare the statement and store it in a map.
	stmt, ok := c.statements[querylimit]
	if !ok {
		stmt1, err := info.Bot.DB.Prepare("SELECT COUNT(*) FROM chatlog C WHERE C.Guild = ? AND " + query)
		stmt2, err2 := info.Bot.DB.Prepare("SELECT U.Username, C.Message, C.Timestamp, U.ID FROM chatlog C INNER JOIN users U ON C.Author = U.ID WHERE C.Guild = ? AND " + querylimit)

		if err == nil {
			err = err2
		}
		if err != nil {
			return bot.ReturnError(err)
		}
		stmt = []*sql.Stmt{stmt1, stmt2}
		c.statements[querylimit] = stmt
	}

	// Execute the statement as a count if appropriate, otherwise retrieve a list of messages and construct a return message from them.
	count := 0
	err := stmt[0].QueryRow(params...).Scan(&count)
	if err == sql.ErrNoRows {
		return "```\nError: Expected 1 row, but got no rows!```", false, nil
	} else if info.Bot.DB.CheckError("Search Command", err) != nil {
		return "```\nError counting search results.```", false, nil
	}

	if count == 0 {
		return "```\nNo results found.```", false, nil
	}

	strmatch := " matches"
	if count == 1 {
		strmatch = " match"
	} // I hate plural forms
	ret := "```\nSearch results: " + strconv.Itoa(count) + strmatch + ".```\n"

	if rangebegin < 0 || rangeend < 0 {
		return ret, false, nil
	}

	maxresults := info.Config.Miscellaneous.MaxSearchResults
	if maxresults < 1 {
		return "```\nError: spam.maxsearchresults is set to 0 or below, so no search results can be displayed.```", false, nil
	}

	if maxresults > 100 {
		maxresults = 100
	}

	if rangeend >= 0 {
		if rangebegin > 0 { // rangebegin starts at 1, not 0
			if rangeend-rangebegin > maxresults {
				rangeend = rangebegin + maxresults
			}
			if rangeend-rangebegin < 0 {
				rangeend = rangebegin
			}
			params = append(params, rangeend-rangebegin+1)
			params = append(params, rangebegin-1) // adjust this so the beginning starts at 1 instead of 0
		} else {
			if rangeend > maxresults {
				rangeend = maxresults
			}
			params = append(params, rangeend)
		}
	}

	q, err := stmt[1].Query(params...)
	if info.Bot.DB.CheckError("Search Command", err) != nil {
		return "```\nError getting search results.```", false, nil
	}
	defer q.Close()
	r := make([]bot.PingContext, 0, 5)
	for q.Next() {
		p := bot.PingContext{}
		var uid uint64
		if err := q.Scan(&p.Author, &p.Message, &p.Timestamp, &uid); err == nil {
			p.Author = info.GetUserName(bot.NewDiscordUser(uid))
			r = append(r, p)
		}
	}

	if len(r) == 0 {
		return "```\nNo results in range.```", false, nil
	}

	for _, v := range r {
		ret += "[" + info.ApplyTimezone(v.Timestamp, bot.DiscordUser(msg.Author.ID)).Format("1/2 3:04:05PM") + "] " + v.Author + ": " + msgHighlightMatch(v.Message, message) + "\n"
	}

	return info.Sanitize(ret, bot.CleanPings|bot.CleanEmotes|bot.CleanMentions|bot.CleanURL), len(r) > 5, nil
}
func (c *searchCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "This is an arbitrary search command run on the 7 day chat log. All parameters are optional and can be input in any order, and will all be combined into a single search as appropriate, but if no searchable parameters are given, the operation will fail.  Remember that if a username has spaces in it, you have to put the entire username parameter in quotes, not just the username itself! \n\n Example: `" + info.Config.Basic.CommandPrefix + "search #manechat @cloud|@Potluck *4 \"~Sep 8 12:00pm\"`\n This will return the most recent 4 messages said by any user with \"cloud\" in the name, or the user Potluck, in the #manechat channel, before Sept 8 12:00pm.",
		Params: []bot.CommandUsageParam{
			{Name: "*[result-range]", Desc: "Specifies what results should be returned. Specifying '*10' will return the first 10 results, while '*5-10' will return the 5th to the 10th result (inclusive). If you ONLY specify a single * character, it will only return a count of the total number of results.", Optional: true},
			{Name: "@user[|@user2|...]", Desc: "Specifies a target user name to search for. An actual ping will be more effective, as it can directly use the user ID, but a raw username will be searched for in the alias table. Multiple users can be searched for by separating them with `|`, but each user must still be prefixed with `@` even if it's not a ping", Optional: true},
			{Name: "#channel[|#channel2|...]", Desc: "Must be an actual channel recognized by discord, which means it should be an actual ping in the format `#channel`, which will filter results to that channel. Multiple channels can be specified using `|`, the same way users can.", Optional: true},
			{Name: "~timestamp", Desc: "Tells the search to only return messages that appeared before the given timestamp. This parameter MUST BE IN QUOTES or it will not be parsed correctly.", Optional: true},
			{Name: "message", Desc: "Will be constructed from all the remaining unrecognized parameters, so you don't need quotes around the message you're looking for.", Optional: true},
		},
	}
}
