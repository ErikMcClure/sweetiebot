package sweetiebot

import (
	"regexp"
	"strings"

	"github.com/blackhole12/discordgo"
)

// EmoteModule detects banned emotes and deletes them
type EmoteModule struct {
	emoteban *regexp.Regexp
	lastmsg  int64
}

// Name of the module
func (w *EmoteModule) Name() string {
	return "Emote"
}

// Commands in the module
func (w *EmoteModule) Commands() []Command { return []Command{} }

// Description of the module
func (w *EmoteModule) Description() string {
	return "Keeps a list of banned emotes that are either seizure-inducing or way too big, and deletes any messages that use them in any channels this module is active in."
}

func (w *EmoteModule) hasBigEmote(info *GuildInfo, m *discordgo.Message) bool {
	if w.emoteban.MatchString(m.Content) {
		sb.dg.ChannelMessageDelete(m.ChannelID, m.ID)
		if RateLimit(&w.lastmsg, 5) {
			info.SendMessage(m.ChannelID, "`That emote isn't allowed here! Try to avoid using large or disturbing emotes, as they can be problematic.`")
		}
		return true
	}
	return false
}

// OnMessageCreate discord hook
func (w *EmoteModule) OnMessageCreate(info *GuildInfo, m *discordgo.Message) {
	w.hasBigEmote(info, m)
}

// OnMessageUpdate discord hook
func (w *EmoteModule) OnMessageUpdate(info *GuildInfo, m *discordgo.Message) {
	w.hasBigEmote(info, m)
}

// OnCommand discord hook
func (w *EmoteModule) OnCommand(info *GuildInfo, m *discordgo.Message) bool {
	if info.UserHasRole(m.Author.ID, SBitoa(info.config.Basic.AlertRole)) {
		return false
	}
	return w.hasBigEmote(info, m)
}

// UpdateRegex updates the emote module regex
func (w *EmoteModule) UpdateRegex(info *GuildInfo) bool {
	var err error
	w.emoteban, err = regexp.Compile("\\[\\]\\(\\/r?(" + strings.Join(MapToSlice(info.config.Basic.Collections["emote"]), "|") + ")[-) \"]")
	return err == nil
}
