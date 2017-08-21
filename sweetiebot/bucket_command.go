package sweetiebot

import (
	"math/rand"
	"strconv"
	"strings"

	"fmt"

	"github.com/blackhole12/discordgo"
)

// BucketModule manages Sweetie's bucket
type BucketModule struct {
}

// Name of the module
func (w *BucketModule) Name() string {
	return "Bucket"
}

// Commands in the module
func (w *BucketModule) Commands() []Command {
	return []Command{
		&giveCommand{},
		&dropCommand{},
		&listCommand{},
		&fightCommand{"", 0},
	}
}

// Description of the module
func (w *BucketModule) Description() string { return "Manages Sweetie's bucket functionality." }

type giveCommand struct {
}

func (c *giveCommand) Name() string {
	return "Give"
}
func (c *giveCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "[](/sadbot) `You didn't give me anything!`", false, nil
	}
	if info.config.Bucket.MaxItems == 0 {
		return "```I don't have a bucket right now (bucket.maxitems is 0).```", false, nil
	}

	arg := ExtraSanitize(msg.Content[indices[0]:])
	if len(arg) > info.config.Bucket.MaxItemLength {
		return "```That's too big! Give me something smaller!```", false, nil
	}

	_, ok := info.config.Basic.Collections["bucket"][arg]
	if ok {
		return "```I already have " + arg + "!```", false, nil
	}

	if len(info.config.Basic.Collections["bucket"]) >= info.config.Bucket.MaxItems {
		dropped := BucketDropRandom(info)
		info.config.Basic.Collections["bucket"][arg] = true
		info.SaveConfig()
		return "```I dropped " + dropped + " and picked up " + arg + ".```", false, nil
	}

	info.config.Basic.Collections["bucket"][arg] = true
	info.SaveConfig()
	return "```I picked up " + arg + ".```", false, nil
}
func (c *giveCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Gives sweetie an object. If sweetie is carrying too many things, she will drop one of them at random.",
		Params: []CommandUsageParam{
			{Name: "arbitrary string", Desc: fmt.Sprintf("An arbitrary string up to %v letters long. Quotes are not required, but cannot be empty.", info.config.Bucket.MaxItemLength), Optional: false},
		},
	}
}
func (c *giveCommand) UsageShort() string { return "Gives something to sweetie." }

// BucketDropRandom removes a random item from the bucket and returns it
func BucketDropRandom(info *GuildInfo) string {
	index := rand.Intn(len(info.config.Basic.Collections["bucket"]))
	i := 0
	for k := range info.config.Basic.Collections["bucket"] {
		if i == index {
			delete(info.config.Basic.Collections["bucket"], k)
			info.SaveConfig()
			return k
		}
		i++
	}
	return ""
}

type dropCommand struct {
}

func (c *dropCommand) Name() string {
	return "Drop"
}

func (c *dropCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(info.config.Basic.Collections["bucket"]) == 0 {
		return "[Realizes her bucket is empty]", false, nil
	}
	if len(args) < 1 {
		return "Throws " + BucketDropRandom(info), false, nil
	}
	arg := msg.Content[indices[0]:]
	_, ok := info.config.Basic.Collections["bucket"][arg]
	if !ok {
		return "```I don't have " + arg + "!```", false, nil
	}
	delete(info.config.Basic.Collections["bucket"], arg)
	info.SaveConfig()
	return "```Dropped " + arg + ".```", false, nil
}
func (c *dropCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Drops the specified object from sweetie. If no object is given, makes sweetie throw something at random.",
		Params: []CommandUsageParam{
			{Name: "arbitrary string", Desc: fmt.Sprintf("An arbitrary string up to %v letters long.", info.config.Bucket.MaxItemLength), Optional: true},
		},
	}
}
func (c *dropCommand) UsageShort() string { return "Drops something from sweetie's bucket." }

type listCommand struct {
}

func (c *listCommand) Name() string {
	return "List"
}
func (c *listCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	things := MapToSlice(info.config.Basic.Collections["bucket"])
	if len(things) == 0 {
		return "```I'm not carrying anything.```", false, nil
	}
	if len(things) == 1 {
		return "```I'm carrying " + things[0] + ".```", false, nil
	}

	return "```I'm carrying " + strings.Join(things[:len(things)-1], ", ") + " and " + things[len(things)-1] + ".```", false, nil
}
func (c *listCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists everything in Sweetie's bucket.",
	}
}
func (c *listCommand) UsageShort() string { return "Lists everything in Sweetie's bucket." }

type fightCommand struct {
	monster string
	hp      int
}

func (c *fightCommand) Name() string {
	return "Fight"
}
func (c *fightCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	things := MapToSlice(info.config.Basic.Collections["bucket"])
	if len(things) == 0 {
		return "```I have nothing to fight with!```", false, nil
	}
	if len(c.monster) > 0 && len(args) > 0 {
		return "I'm already fighting " + c.monster + ", I have to defeat them first!", false, nil
	}
	if info.config.Bucket.MaxFightDamage <= 0 || info.config.Bucket.MaxFightHP <= 0 {
		return "```MaxFightDamage and MaxFightHP must be greater than zero!```", false, nil
	}
	if len(c.monster) == 0 {
		if len(args) > 0 {
			c.monster = ExtraSanitize(msg.Content[indices[0]:])
		} else {
			if !sb.db.CheckStatus() {
				return "```A temporary database outage is preventing this command from being executed.```", false, nil
			}
			if info.config.Markov.UseMemberNames {
				c.monster = ExtraSanitize(sb.db.GetRandomMember(SBatoi(info.ID)))
			} else {
				c.monster = sb.db.GetRandomSpeaker()
			}
		}
		c.hp = 10 + rand.Intn(info.config.Bucket.MaxFightHP)
		return "```I have engaged " + c.monster + ", who has " + strconv.Itoa(c.hp) + " HP!```", false, nil
	}

	damage := 1 + rand.Intn(info.config.Bucket.MaxFightDamage)
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
		return "```I throw " + BucketDropRandom(info) + " at " + monster + end, false, nil
	case 1:
		return "```I stab " + monster + " with " + thing + end, false, nil
	case 2:
		return "```I use " + thing + " on " + monster + end, false, nil
	case 3:
		return "```I summon " + thing + end, false, nil
	case 4:
		return "```I cast " + thing + end, false, nil
	case 5:
		return "```I parry a blow and counterattack with " + thing + end, false, nil
	case 6:
		return "```I detonate a " + thing + end, false, nil
	}
	return "```Stuff happens" + end, false, nil
}
func (c *fightCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Fights a random user, generated character or [name] if it is provided.",
		Params: []CommandUsageParam{
			{Name: "name", Desc: "An arbitrary name for sweetie to fight.", Optional: true},
		},
	}
}
func (c *fightCommand) UsageShort() string { return "Fights a random user or keyword." }
