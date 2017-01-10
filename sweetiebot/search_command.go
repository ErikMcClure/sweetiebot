package sweetiebot

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type SearchCommand struct {
	emotes     *EmoteModule
	lock       AtomicFlag
	statements map[string][]*sql.Stmt
}

func (c *SearchCommand) Name() string {
	return "Search"
}
func MsgHighlightMatch(msg string, match string) string {
	if len(match) == 0 {
		return msg
	}
	msg = strings.Replace(msg, "**"+match+"**", match, -1) // this trick helps prevent increasing ** being appended repeatedly.
	msg = strings.Replace(msg, "**"+match, match, -1)      // helps prevent ** from exploding everywhere because discord is bad at isolation.
	return strings.Replace(msg, match, "**"+match+"**", -1)
}
func (c *SearchCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if c.lock.test_and_set() {
		return "```Sorry, I'm busy processing another request right now. Please try again later!```", false, nil
	}
	defer c.lock.clear()
	rangebegin := 0
	rangeend := 5
	users := make([]string, 0, 0)
	userIDs := make([]uint64, 0, 0)
	channels := make([]uint64, 0, 0)
	t := time.Now().UTC().AddDate(0, 0, 1)
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
			case v[0] == '@' || (v[0] == '<' && v[1] == '@'):
				if len(v) < 2 {
					return "```Error: No users specified```", false, nil
				}
				users = strings.Split(v, "|")
			case v[0] == '#':
				return "```Error: Unknown channel format " + v + " - Must be an actual recognized channel by discord!```", false, nil
			case (v[0] == '<' && v[1] == '#'):
				if len(v) < 2 {
					return "```Error: No channels specified```", false, nil
				}
				s := strings.Split(v, "|")
				for _, c := range s {
					if !channelregex.MatchString(c) {
						return "```Error: Unknown channel format " + c + " - Must be an actual recognized channel by discord!```", false, nil
					}
					channels = append(channels, SBatoi(c[2:len(c)-1]))
				}
			case v[0] == '~':
				var err error
				t, err = parseCommonTime(v[1:], info, msg.Author)
				if err != nil {
					return "```Error: " + err.Error() + "```", false, nil
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
		if userregex.MatchString(v) {
			userIDs = append(userIDs, SBatoi(v[2:len(v)-1]))
		} else {
			IDs := FindUsername(v[1:], info)
			if len(IDs) == 0 { // we failed to resolve this username, so return an error.
				return "```Error: Could not find any usernames or aliases matching " + v[1:] + "!```", false, nil
			}
			userIDs = append(userIDs, IDs...)
		}
	}

	// If we have no searchable arguments, fail
	if len(messages)+len(userIDs)+len(channels) == 0 {
		return "```Error: no searchable terms specified! You must have either a message, a user, or a channel.```", false, nil
	}

	// Assemble query string and parameter list
	params := make([]interface{}, 0, 3)
	params = append(params, SBatoi(info.Guild.ID))
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

	if t.Before(time.Now().UTC()) {
		query += "C.Timestamp < ? AND "
		params = append(params, t)
	}

	cid := SBatoi(msg.ChannelID)
	for _, v := range info.config.Spoiler.Channels {
		if cid != v {
			query += "C.Channel != ? AND "
			params = append(params, v)
		}
	}

	query += "C.ID != ? AND C.Author != ? AND C.Channel != ? AND C.Message NOT LIKE '!search %' ORDER BY C.Timestamp DESC" // Always exclude the message corresponding to the command and all sweetie bot messages (which also prevents trailing ANDs)
	params = append(params, SBatoi(msg.ID))
	params = append(params, SBatoi(sb.SelfID))
	params = append(params, info.config.Basic.ModChannel)

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
		stmt1, err := sb.db.Prepare("SELECT COUNT(*) FROM chatlog C WHERE C.Guild = ? AND " + query)
		stmt2, err2 := sb.db.Prepare("SELECT U.Username, C.Message, C.Timestamp, U.ID FROM chatlog C INNER JOIN users U ON C.Author = U.ID WHERE C.Guild = ? AND " + querylimit)
		if err == nil {
			err = err2
		}
		if err != nil {
			info.log.Log(err.Error())
			return "```Error: Failed to prepare statement!```", false, nil
		}
		stmt = []*sql.Stmt{stmt1, stmt2}
		c.statements[querylimit] = stmt
	}

	// Execute the statement as a count if appropriate, otherwise retrieve a list of messages and construct a return message from them.
	count := 0
	err := stmt[0].QueryRow(params...).Scan(&count)
	if err == sql.ErrNoRows {
		return "```Error: Expected 1 row, but got no rows!```", false, nil
	}

	if count == 0 {
		return "```No results found.```", false, nil
	}

	strmatch := " matches"
	if count == 1 {
		strmatch = " match"
	} // I hate plural forms
	ret := "```Search results: " + strconv.Itoa(count) + strmatch + ".```\n"

	if rangebegin < 0 || rangeend < 0 {
		return ret, false, nil
	}

	if rangeend >= 0 {
		if rangebegin > 0 { // rangebegin starts at 1, not 0
			if rangeend-rangebegin > info.config.Search.MaxResults {
				rangeend = rangebegin + info.config.Search.MaxResults
			}
			if rangeend-rangebegin < 0 {
				rangeend = rangebegin
			}
			params = append(params, rangeend-rangebegin+1)
			params = append(params, rangebegin-1) // adjust this so the beginning starts at 1 instead of 0
		} else {
			if rangeend > info.config.Search.MaxResults {
				rangeend = info.config.Search.MaxResults
			}
			params = append(params, rangeend)
		}
	}

	q, err := stmt[1].Query(params...)
	info.log.LogError("Search error: ", err)
	defer q.Close()
	r := make([]PingContext, 0, 5)
	for q.Next() {
		p := PingContext{}
		var uid uint64
		if err := q.Scan(&p.Author, &p.Message, &p.Timestamp, &uid); err == nil {
			p.Author = getUserName(uid, info)
			r = append(r, p)
		}
	}

	if len(r) == 0 {
		return "```No results in range.```", false, nil
	}

	for _, v := range r {
		ret += "[" + ApplyTimezone(v.Timestamp, info, msg.Author).Format("1/2 3:04:05PM") + "] " + v.Author + ": " + MsgHighlightMatch(v.Message, message) + "\n"
	}

	ret = strings.Replace(ret, "http://", "http\u200B://", -1)
	ret = strings.Replace(ret, "https://", "https\u200B://", -1)
	return ReplaceAllMentions(ret), len(r) > 5, nil
	//return c.emotes.emoteban.ReplaceAllStringFunc(ret, emotereplace), len(r) > 5
}
func (c *SearchCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "This is an arbitrary search command run on sweetiebot's 7 day chat log. All parameters are optional and can be input in any order, and will all be combined into a single search as appropriate, but if no searchable parameters are given, the operation will fail.  Remember that if a username has spaces in it, you have to put the entire username parameter in quotes, not just the username itself! \n\n Example: `!search #manechat @cloud|@JamesNotABot *4 \"~Sep 8 12:00pm\"`\n This will return the most recent 4 messages said by any user with \"cloud\" in the name, or the user JamesNotABot, in the #manechat channel, before Sept 8 12:00pm.",
		Params: []CommandUsageParam{
			CommandUsageParam{Name: "*[result-range]", Desc: "Specifies what results should be returned. Specifing '*10' will return the first 10 results, while '*5-10' will return the 5th to the 10th result (inclusive). If you ONLY specify a single * character, it will only return a count of the total number of results.", Optional: true},
			CommandUsageParam{Name: "@user[|@user2|...]", Desc: "Specifies a target user name to search for. An actual ping will be more effective, as it can directly use the user ID, but a raw username will be searched for in the alias table. Multiple users can be searched for by seperating them with `|`, but each user must still be prefixed with `@` even if it's not a ping", Optional: true},
			CommandUsageParam{Name: "#channel[|#channel2|...]", Desc: "Must be an actual channel recognized by discord, which means it should be an actual ping in the format `#channel`, which will filter results to that channel. Multiple channels can be specified using `|`, the same way users can.", Optional: true},
			CommandUsageParam{Name: "~timestamp", Desc: "Tells the search to only return messages that appeared before the given timestamp. This parameter MUST BE IN QUOTES or it will not be parsed correctly.", Optional: true},
			CommandUsageParam{Name: "message", Desc: "Will be constructed from all the remaining unrecognized parameters, so you don't need quotes around the message you're looking for.", Optional: true},
		},
	}
}
func (c *SearchCommand) UsageShort() string { return "Performs a complex search on the chat history." }
