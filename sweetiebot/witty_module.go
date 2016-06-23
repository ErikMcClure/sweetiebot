package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "regexp"
  "math/rand"
)

// This module is intended for any witty comments sweetie bot makes in response to what users say or do.
type WittyModule struct {
  lastdelete int64
  lastcomment int64
  shutupregex *regexp.Regexp
  wittyregex *regexp.Regexp
  triggerregex []*regexp.Regexp
  remarks [][]string
}
  
func (w *WittyModule) Name() string {
  return "Witty"
}

func (w *WittyModule) Register(hooks *ModuleHooks) {
  w.lastdelete = 0
  w.lastcomment = 0
  w.shutupregex = regexp.MustCompile("shut ?up,? (sb|sweetie ?bot)")
  w.UpdateRegex()
  hooks.OnMessageDelete = append(hooks.OnMessageDelete, w)
  hooks.OnMessageCreate = append(hooks.OnMessageCreate, w)
}

func (w *WittyModule) UpdateRegex() bool {
  l := len(sb.config.Witty)
  w.triggerregex = make([]*regexp.Regexp, 0, l)
  w.remarks = make([][]string, 0, l)
  if l < 1 {
    w.wittyregex = nil
    return true
  }

  var err error
  w.wittyregex, err = regexp.Compile("(" + strings.Join(MapStringToSlice(sb.config.Witty), "|") + ")")

  if err == nil {
    var r *regexp.Regexp
    for k, v := range sb.config.Witty {
      r, err = regexp.Compile(k)
      if err != nil { break }
      w.triggerregex = append(w.triggerregex, r)
      w.remarks = append(w.remarks, strings.Split(v, "|"))
    }
  }

  if len(w.triggerregex) != len(w.remarks) { // This should never happen but we check just in case
    sb.log.Log("ERROR! triggers do not equal remarks!!")
    return false
  }
  return err == nil
}
  
func (w *WittyModule) SendWittyComment(channel string, comment string) {
  if RateLimit(&w.lastcomment, sb.config.Maxwit) {
    sb.SendMessage(channel, comment)
  }
}
func (w *WittyModule)  OnMessageCreate(s *discordgo.Session, m *discordgo.Message) {
  str := strings.ToLower(m.Content)
  /*if w.shutupregex.MatchString(str) {
    if CheckRateLimit(&sb.lastshutup, sb.config.Maxshutup) {
      sb.SendMessage(m.ChannelID, "[](/sadbot) `Sorry! (All comments and public commands disabled in #manechat for the next " + TimeDiff(time.Duration(sb.config.Maxshutup) * time.Second) + ").`")
    }
    sb.lastshutup = time.Now().UTC().Unix()
  }*/
  if CheckRateLimit(&w.lastcomment, sb.config.Maxwit) && CheckShutup(m.ChannelID) {
    if w.wittyregex != nil && w.wittyregex.MatchString(str) {
      for i := 0; i < len(w.triggerregex); i++ {
        if w.triggerregex[i].MatchString(str) {
          w.SendWittyComment(m.ChannelID, w.remarks[i][rand.Intn(len(w.remarks[i]))])
          break
        }
      }
    }
  }
}

func (w *WittyModule) OnMessageDelete(s *discordgo.Session, m *discordgo.Message) {
  //if RateLimit(&w.lastdelete, 60) { // It turns out this triggers when the bot itself deletes things, which looks awkward - maybe this can be fixed?
  //  sb.SendMessage(m.ChannelID, "[](/sbstare) `I SAW THAT`")
  //} 
}


type AddWitCommand struct {
  wit *WittyModule
}

func (c *AddWitCommand) Name() string {
  return "AddWit";  
}
func (c *AddWitCommand) Remove(wit string) bool {
  wit = strings.ToLower(wit)
  _, ok := sb.config.Witty[wit]
  if ok {
    delete(sb.config.Witty, wit)
  }
  return ok
}

func (c *AddWitCommand) Process(args []string, msg *discordgo.Message) (string, bool) {  
  if len(args) < 2 {
    return "```You must provide both a trigger and a remark (both must be in quotes if they have spaces).```", false
  }
  
  trigger := args[0]
  remark := args[1]
  if len(sb.config.Witty) <= 0 {
    sb.config.Witty = make(map[string]string)
  }
  sb.config.Witty[trigger] = remark
  sb.SaveConfig()
  r := c.wit.UpdateRegex()
  if !r {
    c.Remove(trigger)
    c.wit.UpdateRegex()
    return "```Failed to add " + trigger + " because regex compilation failed.```", false
  }
  return "```Adding " + trigger + " and recompiled the wittyremarks regex.```", false
}
func (c *AddWitCommand) Usage() string { 
  return FormatUsage(c, "[trigger] [response]", "Adds a [response] that is triggered by [trigger]. The trigger may be any valid regex string, but it must be in quotes if it has spaces.") 
}
func (c *AddWitCommand) UsageShort() string { return "Adds a line to wittyremarks." }


type RemoveWitCommand struct {
  wit *WittyModule
}

func (c *RemoveWitCommand) Name() string {
  return "RemoveWit";  
}
func (c *RemoveWitCommand) Remove(wit string) bool {
  wit = strings.ToLower(wit)
  _, ok := sb.config.Witty[wit]
  if ok {
    delete(sb.config.Witty, wit)
  }
  return ok
}

func (c *RemoveWitCommand) Process(args []string, msg *discordgo.Message) (string, bool) {  
  if len(args) < 1 {
    return "```You must provide both a trigger to remove!```", false
  }

  arg := strings.Join(args, " ")
  if !c.Remove(arg) {
    return "```Could not find " + arg + "!```", false
  }
  sb.SaveConfig()
  c.wit.UpdateRegex()
  return "```Removed " + arg + " and recompiled the wittyremarks regex.```", false
}
func (c *RemoveWitCommand) Usage() string { 
  return FormatUsage(c, "[trigger]", "Removes [trigger] from wittyremarks, provided it exists.") 
}
func (c *RemoveWitCommand) UsageShort() string { return "Removes a remark from wittyremarks." }