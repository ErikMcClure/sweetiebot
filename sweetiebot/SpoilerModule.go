package sweetiebot

import (
	"regexp"
	"strings"

	"github.com/blackhole12/discordgo"
)

// SpoilerModule picks a random action to do whenever #manechat has been idle for several minutes (configurable)
type SpoilerModule struct {
	spoilerban *regexp.Regexp
	lastmsg    int64 // Sanity rate limiter
}

// Name of the module
func (w *SpoilerModule) Name() string {
	return "Spoiler"
}

// Commands in the module
func (w *SpoilerModule) Commands() []Command { return []Command{} }

// Description of the module
func (w *SpoilerModule) Description() string {
	return "Deletes any messages that match a regex created by the spoiler collection on all channels this module is active in, unless a message is in `spoilchannels`."
}

func (w *SpoilerModule) hasSpoiler(info *GuildInfo, m *discordgo.Message) bool {
	cid := SBatoi(m.ChannelID)
	for _, v := range info.config.Spoiler.Channels {
		if cid == v {
			return false // this is a spoiler channel so we don't monitor it
		}
	}
	if w.spoilerban != nil && w.spoilerban.MatchString(strings.ToLower(m.Content)) {
		sb.dg.ChannelMessageDelete(m.ChannelID, m.ID)
		if RateLimit(&w.lastmsg, info.config.Log.Cooldown) {
			info.SendMessage(m.ChannelID, "[](/nospoilers) ```NO SPOILERS! Posting spoilers is a bannable offense. All discussion about new and future content MUST be in #mylittlespoilers.```")
		}
		return true
	}
	return false
}

// OnMessageCreate discord hook
func (w *SpoilerModule) OnMessageCreate(info *GuildInfo, m *discordgo.Message) {
	w.hasSpoiler(info, m)
}

// OnMessageUpdate discord hook
func (w *SpoilerModule) OnMessageUpdate(info *GuildInfo, m *discordgo.Message) {
	w.hasSpoiler(info, m)
}

// OnCommand discord hook
func (w *SpoilerModule) OnCommand(info *GuildInfo, m *discordgo.Message) bool {
	if info.UserHasRole(m.Author.ID, SBitoa(info.config.Basic.AlertRole)) {
		return false
	} // If we are a mod, always allow us to run this command, otherwise we can't unspoil things
	return w.hasSpoiler(info, m)
}

// UpdateRegex updates spoiler module regex
func (w *SpoilerModule) UpdateRegex(info *GuildInfo) bool {
	if len(info.config.Basic.Collections["spoiler"]) < 1 {
		w.spoilerban = nil
		return true
	}
	var err error
	w.spoilerban, err = regexp.Compile("(" + strings.Join(MapToSlice(info.config.Basic.Collections["spoiler"]), "|") + ")")
	return err == nil
}
