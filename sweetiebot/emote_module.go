package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "regexp"
)

// The emote module detects banned emotes and deletes them
type EmoteModule struct {
  emoteban *regexp.Regexp
}

func (w *EmoteModule) Name() string {
  return "Emote Module"
}

func (w *EmoteModule) Register(hooks *ModuleHooks) {
  w.emoteban = regexp.MustCompile("\\[\\]\\(\\/r?(canada|BlockJuice|octybelleintensifies|angstybloom|alltheclops|bob|darklelicious|flutterbutts|juice|doitfor24)")
  hooks.OnMessageCreate = append(hooks.OnMessageCreate, w)
  hooks.OnMessageUpdate = append(hooks.OnMessageUpdate, w)
}
func (w *EmoteModule) Channels() []string {
  return []string{}
}
  
func (w *EmoteModule)  OnMessageCreate(s *discordgo.Session, m *discordgo.Message) {
  w.OnMessageUpdate(s, m)
}
  
func (w *EmoteModule)  OnMessageUpdate(s *discordgo.Session, m *discordgo.Message) {
  if w.emoteban.Match([]byte(m.Content)) {
    s.ChannelMessageDelete(m.ChannelID, m.ID)
    s.ChannelMessageSend(m.ChannelID, "`That emote was way too big! Try to avoid using large emotes, as they can clutter up the chatroom.`")
  }
}