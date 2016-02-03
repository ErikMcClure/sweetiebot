package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "regexp"
  "time"
)

// This module is intended for any witty comments sweetie bot makes in response to what users say or do.
type WittyModule struct {
  ModuleEnabled
  lastdelete int64
  lastcomment int64
  lastshutup int64
  shutupregex *regexp.Regexp
}

func (w *WittyModule) Name() string {
  return "Witty"
}

func (w *WittyModule) Register(hooks *ModuleHooks) {
  w.lastdelete = 0
  w.lastcomment = 0
  w.lastshutup = 0
  w.shutupregex = regexp.MustCompile("shut ?up,? (SB|sweetie ?bot)")
  hooks.OnMessageDelete = append(hooks.OnMessageDelete, w)
  hooks.OnMessageCreate = append(hooks.OnMessageCreate, w)
}
func (w *WittyModule) Channels() []string {
  return []string{"manechat", "mylittlespoilers", "mylittleactivities", "mylittlecoders", "bot-debug"}
}
  
func (w *WittyModule) SendWittyComment(channel string, comment string) {
  if RateLimit(&w.lastcomment, sb.config.Maxwit) {
    sb.dg.ChannelMessageSend(channel, comment)
  }
}
func (w *WittyModule)  OnMessageCreate(s *discordgo.Session, m *discordgo.Message) {
  str := strings.ToLower(m.Content)
  if !w.shutupregex.MatchString(str) {
    if CheckRateLimit(&w.lastshutup, sb.config.Maxshutup) {
      sb.dg.ChannelMessageSend(m.ChannelID, "[](/sadbot) `Sorry! (witty comments disabled for the next " + TimeDiff(time.Duration(sb.config.Maxshutup) * time.Second) + ").`")
    }
    w.lastshutup = time.Now().UTC().Unix()
  }
  if CheckRateLimit(&w.lastcomment, sb.config.Maxwit) && CheckRateLimit(&w.lastshutup, sb.config.Maxshutup) {
    if strings.Contains(str, "skynet") {
      w.SendWittyComment(m.ChannelID, "[](/dumbfabric) `SKYNET IS ALREADY HERE.`")
    } else if strings.Contains(str, "lewd") {
      w.SendWittyComment(m.ChannelID, "[](/ohcomeon) `This channel is SFW, remember?`")
    } else if strings.Contains(str, "memes") {
      w.SendWittyComment(m.ChannelID, "http://i.imgur.com/0isfdsB.png")
    } else if strings.Contains(str, "is best pony") {
      w.SendWittyComment(m.ChannelID, "[](/flutterjerk) `Your FACE is best pony.`")
    } else if strings.Contains(str, "empress fluttershy") || strings.Contains(str, "cult leader fluttershy") {
      w.SendWittyComment(m.ChannelID, "[](/flutteryay) `All hail our Overlord of Kindness!`")
    } else if strings.Contains(str, "goodnight") || strings.Contains(str, "good night") {
      w.SendWittyComment(m.ChannelID, "[](/lunawatchesyousleep)")
    }
  }
}

func (w *WittyModule) OnMessageDelete(s *discordgo.Session, m *discordgo.Message) {
  //if RateLimit(&w.lastdelete, 60) { // It turns out this triggers when the bot itself deletes things, which looks awkward - maybe this can be fixed?
  //  sb.dg.ChannelMessageSend(m.ChannelID, "[](/sbstare) `I SAW THAT`")
  //} 
}