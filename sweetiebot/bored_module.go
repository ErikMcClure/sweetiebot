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

func (w *BoredModule) Commands() []Command { return []Command{} }

func (w *BoredModule) Description() string {
	return "After the chat is inactive for a given amount of time, chooses a random action from the `boredcommands` configuration option to run, such posting a link from the bored collection or throwing an item from her bucket."
}

func (w *BoredModule) OnIdle(info *GuildInfo, c *discordgo.Channel) {
	id := c.ID

	if RateLimit(&w.lastmessage, w.IdlePeriod(info)) && CheckShutup(id) && len(info.config.Bored.Commands) > 0 {
		m := &discordgo.Message{ChannelID: id, Content: MapGetRandomItem(info.config.Bored.Commands),
			Author: &discordgo.User{
				ID:       sb.SelfID,
				Username: "Sweetie",
				Verified: true,
				Bot:      true,
			},
		}

		SBProcessCommand(sb.dg, m, info, time.Now().UTC().Unix(), sb.IsDBGuild(info), info.IsDebug(m.ChannelID))
	}
}

func (w *BoredModule) IdlePeriod(info *GuildInfo) int64 {
	return info.config.Bored.Cooldown
}
