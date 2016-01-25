package sweetiebot

import (
  "fmt"
  "time"
  "sync/atomic"
  "io/ioutil"
  "github.com/bwmarrin/discordgo"
)

type SweetieBot struct {
  db *BotDB
  log *Log
  dg *discordgo.Session
  GuildID string
  LogChannelID string
  ModChannelID string
  version string
}
var sb *SweetieBot

func MessageHook(s *discordgo.Session, m *discordgo.Message) {
	fmt.Printf("[%s] %20s %20s %s (%s:%s) > %s\n", m.ID, m.ChannelID, m.Timestamp, m.Author.Username, m.Author.ID, m.Author.Email, m.Content);
  // Log this message provided it wasn't sent to the bot-log channel.
  // ALWAYS discard any of our own messages before analysis.
  
  // Process message with modules and commands for all channels first
  
  // If this is a command but we've hit our command saturation limit, emit an error message.
  
  // Now process for any modules or commands specific to this channel
}

var lastdeletion int64

func MessageDeletion(s *discordgo.Session, m *discordgo.MessageDelete) {
  t := time.Now().UTC().Unix()
  d := lastdeletion // perform a read so it doesn't change on us
  if t - d > 60 {
    if atomic.CompareAndSwapInt64(&lastdeletion, d, t) { // If the swapped failed, it means another thread already sent a message and swapped it out, so don't send a message.
      sb.dg.ChannelMessageSend(m.ChannelID, "[](/sbstare) `I SAW THAT`")
    }
  } 
}

func ProcessReady(s *discordgo.Session, r *discordgo.Ready) {
  g := r.Guilds[0]
  sb.GuildID = g.ID
  
  for _, v := range g.Channels {
    fmt.Println(v.Name)
    if v.Name == "bot-log" {
      sb.LogChannelID = v.ID
    }
    if v.Name == "ragemuffins" {
      sb.ModChannelID = v.ID
    }
  }
  
  sb.log.Log("[](/sbload) `Sweetiebot version " + sb.version + " successfully loaded on " + g.Name + ".`");
}

func Initialize() {
  lastdeletion = 0
  dbauth, _ := ioutil.ReadFile("db.auth")
  discorduser, _ := ioutil.ReadFile("username")  
  discordpass, _ := ioutil.ReadFile("passwd")
  sb = &SweetieBot{}
  sb.version = "0.1.0";
  log := &Log{}
  sb.log = log
  
  db, errdb := DB_Load(log, "mysql", string(dbauth))
  if errdb == nil { defer sb.db.Close(); }
  sb.db = db 
  sb.dg = &discordgo.Session{
    OnMessageCreate: MessageHook,
    OnMessageDelete: MessageDeletion,
    OnReady: ProcessReady,
  }
  log.Init(sb)
  sb.db.LoadStatements()
  log.Log("Finished loading database statements")
  log.LogError("Error loading database: ", errdb)
  token, err := sb.dg.Login(string(discorduser), string(discordpass))
  if err != nil {
    log.LogError("Discord login failed: ", err)
    return; // this will close the db because we deferred db.Close()
  }
  if token != "" {
      sb.dg.Token = token
  }

  log.LogError("Error opening websocket connection: ", sb.dg.Open());
  log.LogError("Websocket handshake failure: ", sb.dg.Handshake());
  fmt.Println("Connection established");
  log.LogError("Connection error", sb.dg.Listen());
}