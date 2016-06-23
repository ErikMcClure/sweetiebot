package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
)

// This module sucks up all the pings in a message and adds them to the database for the !lastping command
type PingModule struct {
  channels *map[uint64]bool
}

func (w *PingModule) Name() string {
  return "Ping"
}

func (w *PingModule) Register(hooks *ModuleHooks) {
  hooks.OnMessageCreate = append(hooks.OnMessageCreate, w)
  hooks.OnMessageUpdate = append(hooks.OnMessageUpdate, w)
}
  
func (w *PingModule)  OnMessageCreate(s *discordgo.Session, m *discordgo.Message) {
  w.OnMessageUpdate(s, m)
}

func SBAddPings(m *discordgo.Message) {
  id := SBatoi(m.ID)
  for _, v := range m.Mentions {
    sb.db.AddPing(id, SBatoi(v.ID))
  }
}

func (w *PingModule)  OnMessageUpdate(s *discordgo.Session, m *discordgo.Message) {
  SBAddPings(m)
}