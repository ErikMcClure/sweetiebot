package sweetiebot

// BotInstance represents an instance of the bot using a given token that still runs off the same database as all the other instances
type BotInstance struct {
	DG          *DiscordGoSession
	SelfID      string
	SelfAvatar  string
	SelfName    string
	AppID       uint64
	AppName     string
	Token       string `json:"token"`
	MainGuildID uint64 `json:"mainguildid"`
	isUserMode  bool   `json:"runasuser"` // True if running as a user for some godawful reason
}
