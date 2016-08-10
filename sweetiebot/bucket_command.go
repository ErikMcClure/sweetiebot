package sweetiebot

import (
	"math/rand"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type GiveCommand struct {
}

func (c *GiveCommand) Name() string {
	return "Give"
}
func (c *GiveCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(args) < 1 {
		return "[](/sadbot) `You didn't give me anything!`", false
	}
	if info.config.MaxBucket == 0 {
		return "```I don't have a bucket right now (maxbucket is 0).```", false
	}

	arg := ExtraSanitize(strings.Join(args, " "))
	if len(arg) > info.config.MaxBucketLength {
		return "```That's too big! Give me something smaller!'```", false
	}

	_, ok := info.config.Collections["bucket"][arg]
	if ok {
		return "```I already have " + arg + "!```", false
	}

	if len(info.config.Collections["bucket"]) >= info.config.MaxBucket {
		dropped := BucketDropRandom(info)
		info.config.Collections["bucket"][arg] = true
		info.SaveConfig()
		return "```I dropped " + dropped + " and picked up " + arg + ".```", false
	}

	info.config.Collections["bucket"][arg] = true
	info.SaveConfig()
	return "```I picked up " + arg + ".```", false
}
func (c *GiveCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[arbitrary string]", "Gives sweetie an object. If sweetie is carrying too many things, she will drop one of them at random.")
}
func (c *GiveCommand) UsageShort() string { return "Gives something to sweetie." }
func (c *GiveCommand) Roles() []string    { return []string{} }
func (c *GiveCommand) Channels() []string { return []string{"mylittlebot", "bot-debug"} }

func BucketDropRandom(info *GuildInfo) string {
	index := rand.Intn(len(info.config.Collections["bucket"]))
	i := 0
	for k, _ := range info.config.Collections["bucket"] {
		if i == index {
			delete(info.config.Collections["bucket"], k)
			info.SaveConfig()
			return k
		}
		i++
	}
	return ""
}

type DropCommand struct {
}

func (c *DropCommand) Name() string {
	return "Drop"
}

func (c *DropCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	if len(info.config.Collections["bucket"]) == 0 {
		return "```I'm not carrying anything.```", false
	}
	if len(args) < 1 {
		return "```Dropped " + BucketDropRandom(info) + ".```", false
	}
	arg := strings.Join(args, " ")
	_, ok := info.config.Collections["bucket"][arg]
	if !ok {
		return "```I don't have " + arg + "!```", false
	}
	delete(info.config.Collections["bucket"], arg)
	info.SaveConfig()
	return "```Dropped " + arg + ".```", false
}
func (c *DropCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[arbitrary string]", "Drops the specified object from sweetie. If no object is given, makes sweetie drop something at random.")
}
func (c *DropCommand) UsageShort() string { return "Drops something from sweetie's bucket." }

type ListCommand struct {
}

func (c *ListCommand) Name() string {
	return "List"
}
func (c *ListCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	things := MapToSlice(info.config.Collections["bucket"])
	if len(things) == 0 {
		return "```I'm not carrying anything.```", false
	}
	if len(things) == 1 {
		return "```I'm carrying " + things[0] + ".```", false
	}

	return "```I'm carrying " + strings.Join(things[:len(things)-1], ", ") + " and " + things[len(things)-1] + ".```", false
}
func (c *ListCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "", "Lists everything that sweetie has.")
}
func (c *ListCommand) UsageShort() string { return "Lists everything sweetie has." }

type FightCommand struct {
	monster string
	hp      int
}

func (c *FightCommand) Name() string {
	return "Fight"
}
func (c *FightCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {
	things := MapToSlice(info.config.Collections["bucket"])
	if len(things) == 0 {
		return "```I have nothing to fight with!```", false
	}
	if len(c.monster) > 0 && len(args) > 0 {
		return "I'm already fighting " + c.monster + ", I have to defeat them first!", false
	}
	if info.config.MaxFightDamage <= 0 || info.config.MaxFightHP <= 0 {
		return "```MaxFightDamage and MaxFightHP must be greater than zero!```", false
	}
	if len(c.monster) == 0 {
		if len(args) > 0 {
			c.monster = strings.Join(args, " ")
		} else {
			if info.config.UseMemberNames {
				c.monster = sb.db.GetRandomMember(SBatoi(info.Guild.ID))
			} else {
				c.monster = sb.db.GetRandomSpeaker()
			}
		}
		c.hp = 10 + rand.Intn(info.config.MaxFightHP)
		return "```I have engaged " + c.monster + ", who has " + strconv.Itoa(c.hp) + " HP!```", false
	}

	damage := 1 + rand.Intn(info.config.MaxFightDamage)
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
		return "```I throw " + BucketDropRandom(info) + " at " + monster + end, false
	case 1:
		return "```I stab " + monster + " with " + thing + end, false
	case 2:
		return "```I use " + thing + " on " + monster + end, false
	case 3:
		return "```I summon " + thing + end, false
	case 4:
		return "```I cast " + thing + end, false
	case 5:
		return "```I parry a blow and counterattack with " + thing + end, false
	case 6:
		return "```I detonate a " + thing + end, false
	}
	return "```Stuff happens" + end, false
}
func (c *FightCommand) Usage(info *GuildInfo) string {
	return info.FormatUsage(c, "[name]", "Fights a random pony, or [name] if it is provided.")
}
func (c *FightCommand) UsageShort() string { return "Fights a random pony." }
