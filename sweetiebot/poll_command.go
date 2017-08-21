package sweetiebot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blackhole12/discordgo"
)

// PollModule manages the polling system
type PollModule struct {
}

// Name of the module
func (w *PollModule) Name() string {
	return "Polls"
}

// Commands in the module
func (w *PollModule) Commands() []Command {
	return []Command{
		&pollCommand{},
		&createPollCommand{},
		&deletePollCommand{},
		&voteCommand{},
		&resultsCommand{},
		&addOptionCommand{},
	}
}

// Description of the module
func (w *PollModule) Description() string { return "Manages the polling system." }

type pollCommand struct {
}

func (c *pollCommand) Name() string {
	return "Poll"
}
func (c *pollCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	gID := SBatoi(info.ID)
	if len(args) < 1 {
		polls := sb.db.GetPolls(gID)
		str := make([]string, 0, len(polls)+1)
		str = append(str, "All active polls:")

		for _, v := range polls {
			str = append(str, v.name)
		}
		return strings.Join(str, "\n"), len(str) > 5, nil
	}
	arg := strings.ToLower(msg.Content[indices[0]:])
	id, desc := sb.db.GetPoll(arg, gID)
	if id == 0 {
		return "```That poll doesn't exist!```", false, nil
	}
	options := sb.db.GetOptions(id)

	str := make([]string, 0, len(options)+2)
	str = append(str, desc)

	for _, v := range options {
		str = append(str, fmt.Sprintf("%v. %s", v.index, v.option))
	}

	return strings.Join(str, "\n"), len(str) > 11, nil
}
func (c *pollCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Displays currently active polls or possible options for a given poll.",
		Params: []CommandUsageParam{
			{Name: "poll", Desc: "Name of a specific poll to display.", Optional: true},
		},
	}
}
func (c *pollCommand) UsageShort() string { return "Displays poll description and options." }

type createPollCommand struct {
}

func (c *createPollCommand) Name() string {
	return "CreatePoll"
}
func (c *createPollCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 3 {
		return "```You must provide a name, a description, and one or more options to create the poll. Example: " + info.config.Basic.CommandPrefix + "createpoll pollname \"Description With Space\" \"Option 1\" \"Option 2\"```", false, nil
	}
	gID := SBatoi(info.ID)
	name := strings.ToLower(args[0])
	err := sb.db.AddPoll(name, args[1], gID)
	if err != nil {
		return "```Error creating poll, make sure you haven't used this name already.```", false, nil
	}
	poll, _ := sb.db.GetPoll(name, gID)
	if poll == 0 {
		return "```Error: Orphaned poll!```", false, nil
	}

	for k, v := range args[2:] {
		err = sb.db.AddOption(poll, uint64(k+1), v)
		if err != nil {
			return fmt.Sprintf("```Error adding option %v:%s. Did you try to add the same option twice? Each option must be unique!```", k+1, v), false, nil
		}
	}

	return fmt.Sprintf("```Successfully created %s poll.```", name), false, nil
}
func (c *createPollCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Creates a new poll with the given name, description, and options. All arguments MUST use quotes if they have spaces. \n\nExample usage: `" + info.config.Basic.CommandPrefix + "createpoll pollname \"Description With Space\" \"Option 1\" NoSpaceOption`",
		Params: []CommandUsageParam{
			{Name: "name", Desc: "Name of the new poll. It's suggested to not use spaces because this makes things difficult for other commands. ", Optional: false},
			{Name: "description", Desc: "Poll description that appears when displaying it.", Optional: false},
			{Name: "options", Desc: "Name of the new poll. It's suggested to not use spaces because this makes things difficult for other commands. ", Optional: true, Variadic: true},
		},
	}
}
func (c *createPollCommand) UsageShort() string { return "Creates a poll." }

type deletePollCommand struct {
}

func (c *deletePollCommand) Name() string {
	return "DeletePoll"
}
func (c *deletePollCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```You have to give me a poll name to delete!```", false, nil
	}
	arg := msg.Content[indices[0]:]
	gID := SBatoi(info.ID)
	id, _ := sb.db.GetPoll(arg, gID)
	if id == 0 {
		return "```That poll doesn't exist!```", false, nil
	}
	err := sb.db.RemovePoll(arg, gID)
	if err != nil {
		return "```Error removing poll.```", false, nil
	}
	return fmt.Sprintf("```Successfully removed %s.```", arg), false, nil
}
func (c *deletePollCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Removes the poll with the given poll name.",
		Params: []CommandUsageParam{
			{Name: "poll", Desc: "Name of the poll to delete.", Optional: false},
		},
	}
}
func (c *deletePollCommand) UsageShort() string { return "Deletes a poll." }

type voteCommand struct {
}

func (c *voteCommand) Name() string {
	return "Vote"
}
func (c *voteCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	gID := SBatoi(info.ID)
	if len(args) < 2 {
		polls := sb.db.GetPolls(gID)
		lastpoll := ""
		if len(polls) > 0 {
			lastpoll = fmt.Sprintf(" The most recent poll is \"%s\".", polls[0].name)
		}
		return fmt.Sprintf("```You have to provide both a poll name and the option you want to vote for!%s Use "+info.config.Basic.CommandPrefix+"poll without any arguments to list all active polls.```", lastpoll), false, nil
	}
	name := strings.ToLower(args[0])
	id, _ := sb.db.GetPoll(name, gID)
	if id == 0 {
		return "```That poll doesn't exist! Use " + info.config.Basic.CommandPrefix + "poll with no arguments to list all active polls.```", false, nil
	}

	option, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		opt := sb.db.GetOption(id, msg.Content[indices[1]:])
		if opt == nil {
			return fmt.Sprintf("```That's not one of the poll options! You have to either type in the exact name of the option you want, or provide the numeric index. Use \""+info.config.Basic.CommandPrefix+"poll %s\" to list the available options.```", name), false, nil
		}
		option = *opt
	} else if !sb.db.CheckOption(id, option) {
		return fmt.Sprintf("```That's not a valid option index! Use \""+info.config.Basic.CommandPrefix+"poll %s\" to get all available options for this poll.```", name), false, nil
	}

	err = sb.db.AddVote(SBatoi(msg.Author.ID), id, option)
	if err != nil {
		return "```Error adding vote.```", false, nil
	}

	return "```Voted! Use " + info.config.Basic.CommandPrefix + "results to check the results.```", false, nil
}
func (c *voteCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Adds your vote to a given poll. If you have already voted in the poll, it changes your vote instead.",
		Params: []CommandUsageParam{
			{Name: "poll", Desc: "Name of the poll you want to vote in.", Optional: false},
			{Name: "option", Desc: "The numeric index of the option you want to vote for, or the precise text of the option instead.", Optional: false},
		},
	}
}
func (c *voteCommand) UsageShort() string { return "Votes in a poll." }

type resultsCommand struct {
}

func (c *resultsCommand) Name() string {
	return "Results"
}
func (c *resultsCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	gID := SBatoi(info.ID)
	if len(args) < 1 {
		return "```You have to give me a valid poll name! Use \"" + info.config.Basic.CommandPrefix + "poll\" to list active polls.```", false, nil
	}
	arg := strings.ToLower(msg.Content[indices[0]:])
	id, desc := sb.db.GetPoll(arg, gID)
	if id == 0 {
		return "```That poll doesn't exist! Use \"" + info.config.Basic.CommandPrefix + "poll\" to list active polls.```", false, nil
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

	return strings.Join(str, "\n"), len(str) > 11, nil
}
func (c *resultsCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Displays the results of the given poll, if it exists.",
		Params: []CommandUsageParam{
			{Name: "poll", Desc: "Name of the poll to view.", Optional: false},
		},
	}
}
func (c *resultsCommand) UsageShort() string { return "Displays results of a poll." }

type addOptionCommand struct {
}

func (c *addOptionCommand) Name() string {
	return "AddOption"
}
func (c *addOptionCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.db.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```You have to give me a poll name to add an option to!```", false, nil
	}
	if len(args) < 2 {
		return "```You have to give me an option to add!```", false, nil
	}
	gID := SBatoi(info.ID)
	id, _ := sb.db.GetPoll(args[0], gID)
	if id == 0 {
		return "```That poll doesn't exist!```", false, nil
	}
	arg := msg.Content[indices[1]:]
	err := sb.db.AppendOption(id, arg)
	if err != nil {
		return "```Error appending option, make sure no other option has this value!```", false, nil
	}
	return fmt.Sprintf("```Successfully added %s to %s.```", arg, args[0]), false, nil
}
func (c *addOptionCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Appends an option to a poll.",
		Params: []CommandUsageParam{
			{Name: "poll", Desc: "Name of the poll to modify.", Optional: false},
			{Name: "option", Desc: "The option to append to the end of the poll.", Optional: false},
		},
	}
}
func (c *addOptionCommand) UsageShort() string { return "Appends an option to a poll." }
