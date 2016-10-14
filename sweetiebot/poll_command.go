package sweetiebot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type PollCommand struct {
}

func (c *PollCommand) Name() string {
	return "Poll"
}
func (c *PollCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	gID := SBatoi(info.Guild.ID)
	if len(args) < 1 {
		polls := sb.db.GetPolls(gID)
		str := make([]string, 0, len(polls)+1)
		str = append(str, "All active polls:")

		for _, v := range polls {
			str = append(str, v.name)
		}
		return strings.Join(str, "\n"), len(str) > 5
	}
	arg := strings.ToLower(strings.Join(args, " "))
	id, desc := sb.db.GetPoll(arg, gID)
	if id == 0 {
		return "```That poll doesn't exist!```", false
	}
	options := sb.db.GetOptions(id)

	str := make([]string, 0, len(options)+2)
	str = append(str, desc)

	for _, v := range options {
		str = append(str, fmt.Sprintf("%v. %s", v.index, v.option))
	}

	return strings.Join(str, "\n"), len(str) > 8
}
func (c *PollCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[poll]", "Displays currently active polls or possible options for a given poll.")
}
func (c *PollCommand) UsageShort() string { return "Displays poll description and options." }

type CreatePollCommand struct {
}

func (c *CreatePollCommand) Name() string {
	return "CreatePoll"
}
func (c *CreatePollCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 3 {
		return "```You must provide a name, a description, and one or more options to create the poll. Example: !createpoll pollname \"Description With Space\" \"Option 1\" \"Option 2\"```", false
	}
	gID := SBatoi(info.Guild.ID)
	name := strings.ToLower(args[0])
	err := sb.db.AddPoll(name, args[1], gID)
	if err != nil {
		return "```Error creating poll.```", false
	}
	poll, _ := sb.db.GetPoll(name, gID)
	if poll == 0 {
		return "```Error: Orphaned poll!```", false
	}

	for k, v := range args[2:] {
		err = sb.db.AddOption(poll, uint64(k+1), v)
		if err != nil {
			return fmt.Sprintf("```Error adding option %v:%s. Did you try to add the same option twice? Each option must be unique!```", k+1, v), false
		}
	}

	return fmt.Sprintf("```Successfully created %s poll.```", name), false
}
func (c *CreatePollCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[name] [description] [option 1] [option 2...]", "Creates a new poll with the given name, description, and options. All arguments MUST use quotes if they have spaces. It's suggested that the poll name not have any spaces because this makes this difficult for other commands. \n\nExample usage: !createpoll pollname \"Description With Space\" \"Option 1\" NoSpaceOption")
}
func (c *CreatePollCommand) UsageShort() string { return "Creates a poll." }

type DeletePollCommand struct {
}

func (c *DeletePollCommand) Name() string {
	return "DeletePoll"
}
func (c *DeletePollCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "```You have to give me a poll name to delete!```", false
	}
	arg := strings.Join(args, " ")
	gID := SBatoi(info.Guild.ID)
	id, _ := sb.db.GetPoll(arg, gID)
	if id == 0 {
		return "```That poll doesn't exist!```", false
	}
	err := sb.db.RemovePoll(arg, gID)
	if err != nil {
		return "```Error removing poll.```", false
	}
	return fmt.Sprintf("```Successfully removed %s.```", arg), false
}
func (c *DeletePollCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[poll]", "Removes the poll with the given poll name.")
}
func (c *DeletePollCommand) UsageShort() string { return "Deletes a poll." }

type VoteCommand struct {
}

func (c *VoteCommand) Name() string {
	return "Vote"
}
func (c *VoteCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 2 {
		return "```You have to provide both a poll name and the option you want to vote for! Use !poll without any arguments to list all active polls.```", false
	}
	name := strings.ToLower(args[0])
	gID := SBatoi(info.Guild.ID)
	id, _ := sb.db.GetPoll(name, gID)
	if id == 0 {
		return "```That poll doesn't exist! Use !poll with no arguments to list all active polls.```", false
	}

	option, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		opt := sb.db.GetOption(id, strings.Join(args[1:], " "))
		if opt == nil {
			return "```That's not one of the poll options! You have to either type in the exact name of the option you want, or provide the numeric index. Use !poll to list the available options.```", false
		}
		option = *opt
	} else if !sb.db.CheckOption(id, option) {
		return "```That's not a valid option index! Use !poll to get all available options for this poll.```", false
	}

	err = sb.db.AddVote(SBatoi(msg.Author.ID), id, option)
	if err != nil {
		return "```Error adding vote.```", false
	}

	return "```Voted! Use !results to check the results.```", false
}
func (c *VoteCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[poll] [option]", "Adds your vote to a given poll. If you have already voted in the poll, it changes your vote instead. Option should be the numeric index of the option you want to vote for.")
}
func (c *VoteCommand) UsageShort() string { return "Votes in a poll." }

type ResultsCommand struct {
}

func (c *ResultsCommand) Name() string {
	return "Results"
}
func (c *ResultsCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	gID := SBatoi(info.Guild.ID)
	if len(args) < 1 {
		return "```You have to give me a valid poll name!```", false
	}
	arg := strings.ToLower(strings.Join(args, " "))
	id, desc := sb.db.GetPoll(arg, gID)
	if id == 0 {
		return "```That poll doesn't exist!```", false
	}
	results := sb.db.GetResults(id)
	options := sb.db.GetOptions(id)
	max := uint64(0)
	for _, v := range results {
		if v.count > max {
			max = v.count
		}
	}

	str := make([]string, 0, len(results)+2)
	str = append(str, desc)
	k := 0
	var count uint64
	for _, v := range options {
		count = 0
		if k < len(results) && v.index == results[k].index {
			count = results[k].count
			k++
		}
		normalized := count
		if max > 10 {
			normalized = uint64(float32(count) * (10.0 / float32(max)))
		}
		if count > 0 && normalized < 1 {
			normalized = 1
		}

		graph := ""
		for i := 0; i < 10; i++ {
			if uint64(i) < normalized {
				graph += "\u2588" // this isn't very efficient but the maximum is 10 so it doesn't matter
			} else {
				graph += "\u2591"
			}
		}
		buf := ""
		if v.index < 10 && len(options) > 9 {
			buf = "_"
		}
		str = append(str, fmt.Sprintf("`%s%v. `%s %s (%v votes)", buf, v.index, graph, v.option, count))
	}

	return strings.Join(str, "\n"), len(str) > 9
}
func (c *ResultsCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[poll]", "Displays the results of the given poll, if it exists.")
}
func (c *ResultsCommand) UsageShort() string { return "Displays results of a poll." }
