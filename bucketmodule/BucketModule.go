package bucketmodule

import (
	"math/rand"
	"strconv"
	"strings"

	"fmt"

	bot "../sweetiebot"
	"github.com/blackhole12/discordgo"
)

// BucketModule manages the bucket
type BucketModule struct {
}

// New instance of BucketModule
func New() *BucketModule {
	return &BucketModule{}
}

// Name of the module
func (w *BucketModule) Name() string {
	return "Bucket"
}

// Commands in the module
func (w *BucketModule) Commands() []bot.Command {
	return []bot.Command{
		&giveCommand{},
		&dropCommand{},
		&listCommand{},
		&fightCommand{"", 0},
	}
}

// Description of the module
func (w *BucketModule) Description(info *bot.GuildInfo) string {
	return "Manages the bot's bucket functionality. If the bucket isn't working, make sure you've enabled it via `" + info.Config.Basic.CommandPrefix + "enable bucket` and ensure that her maximum bucket size is greater than 0."
}

type giveCommand struct {
}

func (c *giveCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "Give",
		Usage: "Gives something to the bot.",
	}
}
func (c *giveCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou didn't give me anything!```", false, nil
	}
	if info.Config.Bucket.MaxItems == 0 {
		return "```\nI don't have a bucket right now (bucket.maxitems is 0).```", false, nil
	}

	arg := info.Sanitize(msg.Content[indices[0]:], bot.CleanMentions|bot.CleanPings|bot.CleanEmotes|bot.CleanCode)
	if len(arg) > info.Config.Bucket.MaxItemLength {
		return "```\nThat's too big! Give me something smaller!```", false, nil
	}

	if len(info.Config.Bucket.Items) == 0 {
		info.Config.Bucket.Items = make(map[string]bool)
	}
	_, ok := info.Config.Bucket.Items[arg]
	if ok {
		return "```\nI already have " + arg + "!```", false, nil
	}

	if len(info.Config.Bucket.Items) >= info.Config.Bucket.MaxItems {
		dropped := BucketDropRandom(info)
		info.Config.Bucket.Items[arg] = true
		info.SaveConfig()
		return "```\nI dropped " + dropped + " and picked up " + arg + ".```", false, nil
	}

	info.Config.Bucket.Items[arg] = true
	info.SaveConfig()
	return "```\nI picked up " + arg + ".```", false, nil
}
func (c *giveCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Gives " + info.GetBotName() + " an object. If " + info.GetBotName() + " is carrying too many things, one will be dropped at random.",
		Params: []bot.CommandUsageParam{
			{Name: "arbitrary string", Desc: fmt.Sprintf("An arbitrary string up to %v letters long. Quotes are not required, but cannot be empty.", info.Config.Bucket.MaxItemLength), Optional: false},
		},
	}
}

// BucketDropRandom removes a random item from the bucket and returns it
func BucketDropRandom(info *bot.GuildInfo) string {
	index := rand.Intn(len(info.Config.Bucket.Items))
	i := 0
	for k := range info.Config.Bucket.Items {
		if i == index {
			delete(info.Config.Bucket.Items, k)
			info.SaveConfig()
			return k
		}
		i++
	}
	return ""
}

type dropCommand struct {
}

func (c *dropCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "Drop",
		Usage: "Drops something from the bot's bucket.",
	}
}
func (c *dropCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(info.Config.Bucket.Items) == 0 {
		return "[Realizes the bucket is empty]", false, nil
	}
	if len(args) < 1 {
		return "Throws " + BucketDropRandom(info), false, nil
	}
	arg := msg.Content[indices[0]:]
	_, ok := info.Config.Bucket.Items[arg]
	if !ok {
		return "```\nI don't have " + arg + "!```", false, nil
	}
	delete(info.Config.Bucket.Items, arg)
	info.SaveConfig()
	return "```\nDropped " + arg + ".```", false, nil
}
func (c *dropCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Drops the specified object from " + info.GetBotName() + ". If no object is given, makes " + info.GetBotName() + " throw something at random.",
		Params: []bot.CommandUsageParam{
			{Name: "arbitrary string", Desc: fmt.Sprintf("An arbitrary string up to %v letters long.", info.Config.Bucket.MaxItemLength), Optional: true},
		},
	}
}

type listCommand struct {
}

func (c *listCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "List",
		Usage: "Lists everything in the bucket.",
	}
}
func (c *listCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	things := bot.MapToSlice(info.Config.Bucket.Items)
	if len(things) == 0 {
		return "```\nI'm not carrying anything.```", false, nil
	}
	if len(things) == 1 {
		return "```\nI'm carrying " + things[0] + ".```", false, nil
	}

	return "```\nI'm carrying " + strings.Join(things[:len(things)-1], ", ") + " and " + things[len(things)-1] + ".```", false, nil
}
func (c *listCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Lists everything in the bucket.",
	}
}

type fightCommand struct {
	monster string
	hp      int
}

func (c *fightCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "Fight",
		Usage: "Fights a random user or keyword.",
	}
}
func (c *fightCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	things := bot.MapToSlice(info.Config.Bucket.Items)
	if len(things) == 0 {
		return "```\nI have nothing to fight with!```", false, nil
	}
	if len(c.monster) > 0 && len(args) > 0 {
		return "I'm already fighting " + c.monster + ", I have to defeat them first!", false, nil
	}
	if info.Config.Bucket.MaxFightDamage <= 0 || info.Config.Bucket.MaxFightHP <= 0 {
		return "```\nMaxFightDamage and MaxFightHP must be greater than zero!```", false, nil
	}
	if len(c.monster) == 0 {
		if len(args) > 0 {
			c.monster = info.Sanitize(msg.Content[indices[0]:], bot.CleanMentions|bot.CleanPings|bot.CleanCode|bot.CleanEmotes)
		} else {
			if info.Config.Markov.UseMemberNames {
				g, err := info.GetGuild()
				if err != nil {
					return "```\nError: " + err.Error() + "```", false, nil
				}
				c.monster = info.Sanitize(info.GetMemberName(g.Members[rand.Intn(len(g.Members))]), bot.CleanMentions|bot.CleanPings|bot.CleanCode|bot.CleanEmotes)
			} else if info.Bot.Markov != nil && len(info.Bot.Markov.Speakers) > 0 {
				c.monster = info.Bot.Markov.Speakers[rand.Intn(len(info.Bot.Markov.Speakers))]
			} else {
				return "```\nNo speakers in markov map!```", false, nil
			}
		}
		c.hp = 10 + rand.Intn(info.Config.Bucket.MaxFightHP)
		return "```\nI have engaged " + c.monster + ", who has " + strconv.Itoa(c.hp) + " HP!```", false, nil
	}

	damage := 1 + rand.Intn(info.Config.Bucket.MaxFightDamage)
	c.hp -= damage
	end := " and deal " + strconv.Itoa(damage) + " damage!"
	monster := c.monster
	if c.hp <= 0 {
		end += " " + monster + " has been defeated!"
		c.monster = ""
	}
	end += "```"
	thing := things[rand.Intn(len(things))]
	switch rand.Intn(7) {
	case 0:
		return "```\nI throw " + BucketDropRandom(info) + " at " + monster + end, false, nil
	case 1:
		return "```\nI stab " + monster + " with " + thing + end, false, nil
	case 2:
		return "```\nI use " + thing + " on " + monster + end, false, nil
	case 3:
		return "```\nI summon " + thing + end, false, nil
	case 4:
		return "```\nI cast " + thing + end, false, nil
	case 5:
		return "```\nI parry a blow and counterattack with " + thing + end, false, nil
	case 6:
		return "```\nI detonate " + thing + end, false, nil
	}
	return "```\nStuff happens" + end, false, nil
}
func (c *fightCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Fights a random user, generated character or [name] if it is provided.",
		Params: []bot.CommandUsageParam{
			{Name: "name", Desc: "An arbitrary name for " + info.GetBotName() + " to fight.", Optional: true},
		},
	}
}
