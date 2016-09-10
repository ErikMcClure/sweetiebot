package sweetiebot

import (
	"time"

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

	if RateLimit(&w.lastmessage, w.IdlePeriod(info)) && CheckShutup(id) && len(info.config.BoredCommands) > 0 {
		m := &discordgo.Message{ChannelID: id, Content: MapGetRandomItem(info.config.BoredCommands),
			Author: &discordgo.User{
				ID:       sb.SelfID,
				Username: "Sweetie",
				Verified: true,
				Bot:      true,
			},
		}

		SBProcessCommand(sb.dg, m, info, time.Now().UTC().Unix(), sb.IsDBGuild(info), false, info.IsDebug(m.ChannelID), nil)
	}
}

func (w *BoredModule) IdlePeriod(info *GuildInfo) int64 {
	return info.config.Maxbored
}
