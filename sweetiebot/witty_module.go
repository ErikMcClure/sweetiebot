package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
)

// This module is intended for any witty comments sweetie bot makes in response to what users say or do.
type WittyModule struct {
  lastdelete int64
  lastcomment int64
}

func (w *WittyModule) Name() string {
  return "Witty Module"
}

func (w *WittyModule) Register(hooks *ModuleHooks) {
  w.lastdelete = 0
  w.lastcomment = 0
  hooks.OnMessageDelete = append(hooks.OnMessageDelete, w)
  hooks.OnMessageCreate = append(hooks.OnMessageCreate, w)
}
func (w *WittyModule) Channels() []string {
  return []string{}
}
  
func (w *WittyModule)  OnMessageCreate(s *discordgo.Session, m *discordgo.Message) {
  if RateLimit(&w.lastcomment, 120) {
    if strings.Contains(strings.ToLower(m.Content), "skynet") {
      sb.dg.ChannelMessageSend(m.ChannelID, "[](/dumbfabric) `SKYNET IS ALREADY HERE.`")
    }
  }
}

func (w *WittyModule) OnMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
  //if RateLimit(&w.lastdelete, 60) { // It turns out this triggers when the bot itself deletes things, which looks awkward
  //  sb.dg.ChannelMessageSend(m.ChannelID, "[](/sbstare) `I SAW THAT`")
  //} 
}