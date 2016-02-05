package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
)

// This module picks a random action to do whenever #manechat has been idle for several minutes (configurable)
type BoredModule struct {
  ModuleEnabled
  lastmessage int64 // Ensures discord screwing up doesn't make us spam the chatroom.
}

func (w *BoredModule) Name() string {
  return "Bored"
}

func (w *BoredModule) Register(hooks *ModuleHooks) {
  w.lastmessage = 0
}
func (w *BoredModule) Channels() []string {
  return []string{"manechat"} // This doesn't really matter because OnIdle will only fire for the manechat.
}
 
func (w *BoredModule) OnIdle(s *discordgo.Session) {
  q := &QuoteCommand{};
  r, _ := q.Process([]string{"action"}, nil)
  sb.SendMessage(sb.ManeChannelID, r) // We pass in nil for the user because this particular function ignores it.
}

func (w *BoredModule) IdlePeriod() int64 {
  return sb.config.Maxbored;
}