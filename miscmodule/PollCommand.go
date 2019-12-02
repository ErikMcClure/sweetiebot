package miscmodule

import (
	"fmt"
	"strings"

	bot "../sweetiebot"
	"github.com/erikmcclure/discordgo"
)

type pollCommand struct {
}

func (c *pollCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:              "Poll",
		Usage:             "Analyzes an emoji reaction poll.",
		ServerIndependent: true,
	}
}

func showEmoji(e *discordgo.Emoji) string {
	if e.ID != "" && e.Name != "" {
		if e.Animated {
			return "<a:" + e.Name + ":" + e.ID + ">"
		}
		return "<:" + e.Name + ":" + e.ID + ">"
	}
	if e.Name != "" {
		return e.Name
	}
	return e.ID
}
func (c *pollCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	var target *discordgo.Message
	if len(args) < 1 {
		lastid := msg.ID
		const Lookback int = 5

		for i := 0; i < Lookback; i++ {
			messages, err := info.Bot.DG.ChannelMessages(msg.ChannelID, 99, lastid, "", "")
			info.LogError("Error encountered while attempting to retrieve messages: ", err)
			if len(messages) == 0 || err != nil {
				break
			}
			lastid = messages[len(messages)-1].ID
			for _, v := range messages {
				if len(v.Reactions) > 0 {
					target = v
					i = Lookback
					break
				}
			}
		}
	} else {
		var err error
		if target, err = info.Bot.DG.ChannelMessage(msg.ChannelID, strings.TrimSpace(args[0])); err != nil {
			return "```\nError retrieving message, are you sure that message is in this channel? It can't be a message from another channel.```", false, nil
		}
	}

	if target == nil {
		return "```\nError: could not find any recent message in this channel with emoji reactions.```", false, nil
	}
	max := 0
	for _, v := range target.Reactions {
		if v.Count > max {
			max = v.Count
		}
	}

	str := make([]string, 0, len(target.Reactions))
	desc, _ := target.ContentWithMoreMentionsReplaced(&info.Bot.DG.Session)
	if len(desc) > 0 {
		str = append(str, desc)
	}

	votes := make(map[string]int)
	for _, v := range target.Reactions {
		if max <= 100 { // It's impossible to get more than 100 user reactions to a message, so give up if the max exceeds 100
			if users, err := info.Bot.DG.MessageReactions(target.ChannelID, target.ID, v.Emoji.APIName(), 100); err == nil {
				for _, v := range users {
					n := votes[v.ID] // If this doesn't exist it will get the zero value, which happens to be what we want anyway
					votes[v.ID] = n + 1
				}
			} else {
				fmt.Println(err.Error())
			}
		}

		normalized := v.Count
		if max > 10 {
			normalized = int(float32(v.Count) * (10.0 / float32(max)))
		}
		if v.Count > 0 && normalized < 1 {
			normalized = 1
		}

		graph := ""
		for i := 0; i < 10; i++ {
			if i < normalized {
				graph += "\u2588" // this isn't very efficient but the maximum is 10 so it doesn't matter
			} else {
				graph += "\u2591"
			}
		}
		str = append(str, fmt.Sprintf("%s %s (%v)", graph, showEmoji(v.Emoji), v.Count))
	}

	if len(votes) > 0 {
		most := 0
		mostID := ""
		for k, v := range votes {
			if v > most {
				mostID = k
				most = v
			}
		}
		if most > 1 {
			str = append(str, fmt.Sprintf("\nThe user with the most votes was %s with %v.", info.GetUserName(bot.DiscordUser(mostID)), most))
		}
	}
	return strings.Join(str, "\n"), len(str) > 21, nil
}
func (c *pollCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Analyzes an emoji reaction poll and displays the results as a graph. If anyone voted more than once, also displays who voted the most (does not work for messages with more than 100 votes on a single emoji reaction).",
		Params: []bot.CommandUsageParam{
			{Name: "message", Desc: "Message ID of the message to analyze. If omitted, searches the current channel for the last message with an emoji reaction and analyzes that.", Optional: true},
		},
	}
}
