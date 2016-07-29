package sweetiebot

import (
	"math/rand"

	"github.com/bwmarrin/discordgo"
)

// This module picks a random action to do whenever #manechat has been idle for several minutes (configurable)
type BoredModule struct {
	Episodegen  *EpisodeGenCommand
	lastmessage int64 // Ensures discord screwing up doesn't make us spam the chatroom.
}

func (w *BoredModule) Name() string {
	return "Bored"
}

func (w *BoredModule) Register(info *GuildInfo) {
	w.lastmessage = 0
	info.hooks.OnIdle = append(info.hooks.OnIdle, w)
}

func (w *BoredModule) OnIdle(info *GuildInfo, c *discordgo.Channel) {
	id := c.ID

	if RateLimit(&w.lastmessage, w.IdlePeriod(info)) && CheckShutup(id) {
		disable := info.config.DisableBored
		if disable < 0 {
			disable = 0
		}
		if disable > 3 {
			disable = 3
		}

		switch rand.Intn(4 - disable) {
		case 0:
			if len(info.config.Collections["bored"]) > 0 {
				info.SendMessage(id, MapGetRandomItem(info.config.Collections["bored"]))
			}
		case 1:
			if len(info.config.Collections["bucket"]) > 0 {
				info.SendMessage(id, "Throws "+BucketDropRandom(info))
			} else {
				info.SendMessage(id, "[Realizes her bucket is empty]")
			}
		case 2:
			q := &QuoteCommand{}
			m := &discordgo.Message{ChannelID: id}
			r, _ := q.Process([]string{}, m, info) // We pass in nil for the user because this particular function ignores it.
			info.SendMessage(id, r)
		case 3:
			m := &discordgo.Message{ChannelID: id}
			r, _ := w.Episodegen.Process([]string{"2"}, m, info)
			info.SendMessage(id, r)
			//case 3: // Removed because tchernobog hates fun
			//  q := &BestPonyCommand{};
			//  m := &discordgo.Message{ChannelID: id}
			//  r, _ := q.Process([]string{}, m) // We pass in nil for the user because this particular function ignores it.
			//  info.SendMessage(id, r)
		}
	}
}

func (w *BoredModule) IdlePeriod(info *GuildInfo) int64 {
	return info.config.Maxbored
}
