package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "math/rand"
)

type GiveCommand struct {
}

func (c *GiveCommand) Name() string {
  return "Give";  
}
func (c *GiveCommand) Process(args []string, msg *discordgo.Message) (string, bool) {  
  if len(args) < 1 {
    return "[](/sadbot) `You didn't give me anything!`", false
  }
  if sb.config.MaxBucket == 0 {
    return "```I don't have a bucket right now.```", false 
  }

  arg := strings.Join(args, " ")
  if len(sb.config.Bucket) <= 0 {
    sb.config.Bucket = make(map[string]bool)
  }
  _, ok := sb.config.Bucket[arg]
  if ok {
    return "```I already have " + arg + "!```", false
  }

  if len(sb.config.Bucket) >= sb.config.MaxBucket {
    dropped := BucketDropRandom()
    sb.config.Bucket[arg] = true
    sb.SaveConfig()
    return "```I dropped " + dropped + " and picked up " + arg + ".```", false
  }

  sb.config.Bucket[arg] = true
  sb.SaveConfig()
  return "```I picked up " + arg + ".```", false
}
func (c *GiveCommand) Usage() string { 
  return FormatUsage(c, "[arbitrary string]", "Gives sweetie an object. If sweetie is carrying too many things, she will drop one of them at random.") 
}
func (c *GiveCommand) UsageShort() string { return "Gives something to sweetie." }
func (c *GiveCommand) Roles() []string { return []string{} }
func (c *GiveCommand) Channels() []string { return []string{} }

func BucketDropRandom() string {
  index := rand.Intn(len(sb.config.Bucket))
  i := 0
  for k, _ := range sb.config.Bucket {
    if i == index {
      delete(sb.config.Bucket, k)
      sb.SaveConfig()
      return k
    }
    i++
  }
  return ""
}

type DropCommand struct {
}

func (c *DropCommand) Name() string {
  return "Drop";  
}

func (c *DropCommand) Process(args []string, msg *discordgo.Message) (string, bool) {  
  if len(sb.config.Bucket) == 0 {
    return "```I'm not carrying anything.```", false
  }
  if len(args) < 1 {
    return "```Dropped " + BucketDropRandom() + ".```", false
  }
  arg := strings.Join(args, " ")
  _, ok := sb.config.Bucket[arg]
  if !ok {
    return "```I don't have " + arg + "!```", false
  }
  delete(sb.config.Bucket, arg)
  sb.SaveConfig()
  return "```Dropped " + arg + ".```", false
}
func (c *DropCommand) Usage() string { 
  return FormatUsage(c, "[arbitrary string]", "Drops the specified object from sweetie. If no object is given, makes sweetie drop something at random.") 
}
func (c *DropCommand) UsageShort() string { return "Drops something from sweetie's bucket." }
func (c *DropCommand) Roles() []string { return []string{} }
func (c *DropCommand) Channels() []string { return []string{} }


type ListCommand struct {
}

func (c *ListCommand) Name() string {
  return "List";  
}
func (c *ListCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  things := MapToSlice(sb.config.Bucket)
  if len(things) == 0 {
    return "```I'm not carrying anything.```", false
  }
  if len(things) == 1 {
    return "```I'm carrying " + things[0] + ".```", false
  }

  return "```I'm carrying " + strings.Join(things[:len(things)-1], ", ") + " and " + things[len(things)-1] + ".```", false
}
func (c *ListCommand) Usage() string { 
  return FormatUsage(c, "", "Lists everything that sweetie has.") 
}
func (c *ListCommand) UsageShort() string { return "Lists everything sweetie has." }
func (c *ListCommand) Roles() []string { return []string{} }
func (c *ListCommand) Channels() []string { return []string{} }