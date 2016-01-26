package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
)

type WittyModule struct {
  lastdelete int64
}

func (w *WittyModule) Name() string {
  return "Witty Module"
}

func (w *WittyModule) Register(hooks *ModuleHooks) {
  w.lastdelete = 0
  hooks.OnMessageDelete = append(hooks.OnMessageDelete, w)
}
func (w *WittyModule) Channels() []string {
  return []string{}
}
  
func (w *WittyModule) OnMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
  if RateLimit(&w.lastdelete, 60) {
    sb.dg.ChannelMessageSend(m.ChannelID, "[](/sbstare) `I SAW THAT`")
  } 
}