package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "math/rand"
  "strings"
  "strconv"
)

// This module picks a random action to do whenever #example has been idle for several minutes (configurable)
type BoredModule struct {
  ModuleEnabled
  Episodegen *EpisodeGenCommand
  lastmessage int64 // Ensures discord screwing up doesn't make us spam the chatroom.
}

func (w *BoredModule) Name() string {
  return "Bored"
}

func (w *BoredModule) Register(hooks *ModuleHooks) {
  w.lastmessage = 0
  hooks.OnIdle = append(hooks.OnIdle, w);
}
func (w *BoredModule) Channels() []string {
  return []string{"example"} // This doesn't really matter because OnIdle will only fire for the example.
}
 
func (w *BoredModule) OnIdle(s *discordgo.Session, c *discordgo.Channel) {
  id := c.ID
  
  if RateLimit(&w.lastmessage, w.IdlePeriod()) && CheckShutup(id) {
    switch rand.Intn(3) {
      case 0:
        q := &QuoteCommand{};
        m := &discordgo.Message{ChannelID: id}
        r, _ := q.Process([]string{}, m) // We pass in nil for the user because this particular function ignores it.
        sb.SendMessage(id, r) 
      case 1:
        m := &discordgo.Message{ChannelID: id}
        r, _ := w.Episodegen.Process([]string{"2"}, m)
        sb.SendMessage(id, r)
      case 2:
        if len(sb.config.BoredLines) > 0 {
          sb.SendMessage(id, sb.config.BoredLines[rand.Intn(len(sb.config.BoredLines))])
        }
      //case 3: // Removed because tchernobog hates fun
      //  q := &BestPonyCommand{};
      //  m := &discordgo.Message{ChannelID: id}
      //  r, _ := q.Process([]string{}, m) // We pass in nil for the user because this particular function ignores it.
      //  sb.SendMessage(id, r) 
    }
  }
}

func (w *BoredModule) IdlePeriod() int64 {
  return sb.config.Maxbored;
}


type AddBoredCommand struct {
}

func (c *AddBoredCommand) Name() string {
  return "AddBored";  
}
func (c *AddBoredCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    return "```No phrase to add.```", false
  }
  sb.config.BoredLines = append(sb.config.BoredLines, strings.Join(args, " "))
  sb.SaveConfig()
  return "```Added line to BoredLines. Length of BoredLines: " + strconv.Itoa(len(sb.config.BoredLines)) + ".```", false
}
func (c *AddBoredCommand) Usage() string { 
  return FormatUsage(c, "[arbitrary string]", "Adds a line to boredlines (no quotes are required).") 
}
func (c *AddBoredCommand) UsageShort() string { return "Adds a line to boredlines." }
func (c *AddBoredCommand) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *AddBoredCommand) Channels() []string { return []string{} }