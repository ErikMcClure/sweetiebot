package boredmodule

import (
	"fmt"
	"time"

	bot "../sweetiebot"
	"github.com/blackhole12/discordgo"
)

// BoredModule picks a random action to do whenever a channel has been idle for several minutes (configurable)
type BoredModule struct {
	lastmessage int64 // Ensures discord screwing up doesn't make us spam the chatroom.
}

// New instance of BoredModule
func New() *BoredModule {
	return &BoredModule{0}
}

// Name of the module
func (w *BoredModule) Name() string {
	return "Bored"
}

// Commands in the module
func (w *BoredModule) Commands() []bot.Command { return []bot.Command{} }

// Description of the module
func (w *BoredModule) Description() string {
	return "After the chat is inactive for a given amount of time, chooses a random action from the `boredcommands` configuration option to run, such posting a link from the bored collection or throwing an item from the bucket."
}

// OnIdle discord hook
func (w *BoredModule) OnIdle(info *bot.GuildInfo, c *discordgo.Channel, t time.Time) {
	id := c.ID
	if bot.RateLimit(&w.lastmessage, w.IdlePeriod(info), t.Unix()) && len(info.Config.Bored.Commands) > 0 {
		m := &discordgo.Message{ChannelID: id, Content: bot.MapGetRandomItem(info.Config.Bored.Commands),
			Author: &discordgo.User{
				ID:       info.Bot.SelfID.String(),
				Username: "Sweetie",
				Verified: true,
				Bot:      true,
			},
			Timestamp: discordgo.Timestamp(t.Format(time.RFC3339Nano)),
		}
		fmt.Println("Sending bored command ", m.Content, " on ", id)

		info.Bot.ProcessCommand(m, info, t.Unix(), info.IsDebug(bot.DiscordChannel(m.ChannelID)), false)
	}
}

// IdlePeriod discord hook
func (w *BoredModule) IdlePeriod(info *bot.GuildInfo) int64 {
	return info.Config.Bored.Cooldown
}
