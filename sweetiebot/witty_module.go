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

func (w *WittyModule) Register(info *GuildInfo) {
  w.lastdelete = 0
  w.lastcomment = 0
  w.shutupregex = regexp.MustCompile("shut ?up,? (sb|sweetie ?bot)")
  w.UpdateRegex(info)
  info.hooks.OnMessageDelete = append(info.hooks.OnMessageDelete, w)
  info.hooks.OnMessageCreate = append(info.hooks.OnMessageCreate, w)
}

func (w *WittyModule) UpdateRegex(info *GuildInfo) bool {
  l := len(info.config.Witty)
  w.triggerregex = make([]*regexp.Regexp, 0, l)
  w.remarks = make([][]string, 0, l)
  if l < 1 {
    w.wittyregex = nil
    return true
  }

  var err error
  w.wittyregex, err = regexp.Compile("(" + strings.Join(MapStringToSlice(info.config.Witty), "|") + ")")

  if err == nil {
    var r *regexp.Regexp
    for k, v := range info.config.Witty {
      r, err = regexp.Compile(k)
      if err != nil { break }
      w.triggerregex = append(w.triggerregex, r)
      w.remarks = append(w.remarks, strings.Split(v, "|"))
    }
  }

  if len(w.triggerregex) != len(w.remarks) { // This should never happen but we check just in case
    info.log.Log("ERROR! triggers do not equal remarks!!")
    return false
  }
  return err == nil
}
  
func (w *WittyModule) SendWittyComment(channel string, comment string, info *GuildInfo) {
  if RateLimit(&w.lastcomment, info.config.Maxwit) {
    info.SendMessage(channel, comment)
  }
}
func (w *WittyModule)  OnMessageCreate(info *GuildInfo, m *discordgo.Message) {
  str := strings.ToLower(m.Content)
  /*if w.shutupregex.MatchString(str) {
    if CheckRateLimit(&info.lastshutup, info.config.Maxshutup) {
      info.SendMessage(m.ChannelID, "[](/sadbot) `Sorry! (All comments and public commands disabled in #manechat for the next " + TimeDiff(time.Duration(info.config.Maxshutup) * time.Second) + ").`")
    }
    info.lastshutup = time.Now().UTC().Unix()
  }*/
  if CheckRateLimit(&w.lastcomment, info.config.Maxwit) && CheckShutup(m.ChannelID) {
    if w.wittyregex != nil && w.wittyregex.MatchString(str) {
      for i := 0; i < len(w.triggerregex); i++ {
        if w.triggerregex[i].MatchString(str) {
          w.SendWittyComment(m.ChannelID, w.remarks[i][rand.Intn(len(w.remarks[i]))], info)
          break
        }
      }
    }
  }
}

func (w *WittyModule) OnMessageDelete(info *GuildInfo, m *discordgo.Message) {
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
func WitRemove(wit string, info *GuildInfo) bool {
  wit = strings.ToLower(wit)
  _, ok := info.config.Witty[wit]
  if ok {
    delete(info.config.Witty, wit)
  }
  return ok
}

func (c *AddWitCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {  
  if len(args) < 2 {
    return "```You must provide both a trigger and a remark (both must be in quotes if they have spaces).```", false
  }
  
  trigger := args[0]
  remark := args[1]
  
  CheckMapNilString(&info.config.Witty)
  info.config.Witty[trigger] = remark
  info.SaveConfig()
  r := c.wit.UpdateRegex(info)
  if !r {
    WitRemove(trigger, info)
    c.wit.UpdateRegex(info)
    return "```Failed to add " + trigger + " because regex compilation failed.```", false
  }
  return "```Adding " + trigger + " and recompiled the wittyremarks regex.```", false
}
func (c *AddWitCommand) Usage(info *GuildInfo) string { 
  return info.FormatUsage(c, "[trigger] [response]", "Adds a [response] that is triggered by [trigger]. The trigger may be any valid regex string, but it must be in quotes if it has spaces.") 
}
func (c *AddWitCommand) UsageShort() string { return "Adds a line to wittyremarks." }


type RemoveWitCommand struct {
  wit *WittyModule
}

func (c *RemoveWitCommand) Name() string {
  return "RemoveWit";  
}

func (c *RemoveWitCommand) Process(args []string, msg *discordgo.Message, info *GuildInfo) (string, bool) {  
  if len(args) < 1 {
    return "```You must provide both a trigger to remove!```", false
  }

  arg := strings.Join(args, " ")
  if !WitRemove(arg, info) {
    return "```Could not find " + arg + "!```", false
  }
  info.SaveConfig()
  c.wit.UpdateRegex(info)
  return "```Removed " + arg + " and recompiled the wittyremarks regex.```", false
}
func (c *RemoveWitCommand) Usage(info *GuildInfo) string { 
  return info.FormatUsage(c, "[trigger]", "Removes [trigger] from wittyremarks, provided it exists.") 
}
func (c *RemoveWitCommand) UsageShort() string { return "Removes a remark from wittyremarks." }