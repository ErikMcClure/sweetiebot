package boredmodule

import (
	"fmt"
	"math"
	"strings"
	"time"

	bot "../sweetiebot"
	"github.com/blackhole12/discordgo"
)

// BoredModule picks a random action to do whenever a channel has been idle for several minutes (configurable)
type BoredModule struct {
	lastmessage  int64                        // Ensures discord screwing up doesn't make us spam the chatroom.
	lastactivity map[bot.DiscordChannel]int64 // Last time a channel room had activity other than sweetiebot.
	count        map[bot.DiscordChannel]int   // Count of consecutive bored messages per channel
}

// New instance of BoredModule
func New() *BoredModule {
	return &BoredModule{0, make(map[bot.DiscordChannel]int64), make(map[bot.DiscordChannel]int)}
}

// Name of the module
func (w *BoredModule) Name() string {
	return "Bored"
}

// Commands in the module
func (w *BoredModule) Commands() []bot.Command { return []bot.Command{} }

// Description of the module
func (w *BoredModule) Description(info *bot.GuildInfo) string {
	return "After the chat is inactive for a given amount of time, chooses a random action from the `bored.commands` configuration option to run, such posting a link from the bored collection or throwing an item from the bucket. To set what channels this module operates on, use `" + info.Config.Basic.CommandPrefix + "setconfig modules.channels bored #channel1 #channel2...`. To set the list of bored commands, use `" + info.Config.Basic.CommandPrefix + "setconfig bored.commands \"" + info.Config.Basic.CommandPrefix + "command1\" \"" + info.Config.Basic.CommandPrefix + "command2 arg\"...`."
}

func (w *BoredModule) idle(info *bot.GuildInfo, id bot.DiscordChannel, t time.Time) {
	w.count[id]++
	if bot.RateLimit(&w.lastmessage, info.Config.Bored.Cooldown, t.Unix()) && len(info.Config.Bored.Commands) > 0 {
		m := &discordgo.Message{ChannelID: id.String(), Content: bot.MapGetRandomItem(info.Config.Bored.Commands),
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

// OnTick discord hook
func (w *BoredModule) OnTick(info *bot.GuildInfo, t time.Time) {
	channels := info.Config.Modules.Channels[bot.ModuleID(strings.ToLower(w.Name()))]
	for ch := range channels {
		last, exists := info.Bot.GetLastMessage(ch)

		if exists {
			if _, ok := w.lastactivity[ch]; !ok {
				w.lastactivity[ch] = 0
			}
			if w.lastactivity[ch] != last {
				w.lastactivity[ch] = last
				w.count[ch] = 0
			}
			diff := t.Sub(time.Unix(last, 0))
			idle := int64(math.Floor(float64(info.Config.Bored.Cooldown) * (math.Pow(info.Config.Bored.Exponent, float64(w.count[ch])) + float64(w.count[ch]))))

			if diff >= (time.Duration(idle) * time.Second) {
				w.idle(info, ch, t)
			}
		}
	}
}
