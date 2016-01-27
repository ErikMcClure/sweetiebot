package sweetiebot

import (
  "fmt"
  "strconv"
  "time"
  "io/ioutil"
  "github.com/bwmarrin/discordgo"
  "strings"
)

type ModuleHooks struct {
    OnEvent                   []ModuleOnEvent
    OnTypingStart             []ModuleOnTypingStart
    OnMessageCreate           []ModuleOnMessageCreate
    OnMessageUpdate           []ModuleOnMessageUpdate
    OnMessageDelete           []ModuleOnMessageDelete
    OnMessageAck              []ModuleOnMessageAck
    OnUserUpdate              []ModuleOnUserUpdate
    OnPresenceUpdate          []ModuleOnPresenceUpdate
    OnVoiceStateUpdate        []ModuleOnVoiceStateUpdate
    OnGuildUpdate             []ModuleOnGuildUpdate
    OnGuildMemberAdd          []ModuleOnGuildMemberAdd
    OnGuildMemberRemove       []ModuleOnGuildMemberRemove
    OnGuildMemberUpdate       []ModuleOnGuildMemberUpdate
    OnGuildBanAdd             []ModuleOnGuildBanAdd
    OnGuildBanRemove          []ModuleOnGuildBanRemove
    OnEvent_channels          []map[uint64]bool
    OnTypingStart_channels    []map[uint64]bool
    OnMessageCreate_channels  []map[uint64]bool
    OnMessageUpdate_channels  []map[uint64]bool
    OnMessageDelete_channels  []map[uint64]bool
    OnMessageAck_channels     []map[uint64]bool
    OnUserUpdate_channels     []map[uint64]bool
    OnPresenceUpdate_channels []map[uint64]bool
    OnVoiceStateUpdate_channels []map[uint64]bool
    OnGuildUpdate_channels    []map[uint64]bool
    OnGuildMemberAdd_channels []map[uint64]bool
    OnGuildMemberRemove_channels []map[uint64]bool
    OnGuildMemberUpdate_channels []map[uint64]bool
    OnGuildBanAdd_channels    []map[uint64]bool
    OnGuildBanRemove_channels []map[uint64]bool
}

type BotCommand struct {
  c Command
  roles map[uint64]bool
}

type SweetieBot struct {
  db *BotDB
  log *Log
  dg *discordgo.Session
  SelfID string
  GuildID string
  LogChannelID string
  ModChannelID string
  DebugChannelID string
  SilentRole string
  version string
  debug bool
  hooks ModuleHooks
  modules []Module
  commands map[string]BotCommand
  commandlimit *SaturationLimit
}

func (sbot *SweetieBot) AddCommand(c Command) {
  m := make(map[uint64]bool)
  for _, r := range c.Roles() {
    for _, v := range sb.dg.State.Guilds[0].Roles {
      if v.Name == r {
        m[SBatoi(v.ID)] = true
        break
      }
    }
  }
  sbot.commands[strings.ToLower(c.Name())] = BotCommand{c, m}
}

var sb *SweetieBot

func SBatoi(s string) uint64 {
  i, err := strconv.ParseUint(s, 10, 64)
  if err != nil { 
    sb.log.Log("Invalid number ", s)
    return 0 
  }
  return i
}

func IsSpace(b byte) bool {
  return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func ParseArguments(s string) []string {
  r := []string{};
  l := len(s)
  for i := 0; i < l; i++ {
    c := s[i]
    if !IsSpace(c) {
      var start int;
      
      if c == '"' {
        i++
        start = i
        for i<(l-1) && (s[i] != '"' || !IsSpace(s[i+1])) { i++ }
      } else {
        start = i;
        i++
        for i<l && !IsSpace(s[i]) { i++ }
      }
      r = append(r, s[start:i])
    } 
  }
  return r
}

func ProcessModules(channels []map[uint64]bool, channelID string, fn func(i int)) {
  if len(channels)>0 { // only bother doing anything if we actually have hooks to process
    for i, c := range channels {
      if len(channelID)>0 && len(c)>0 { // Only check for channels if we have a channel to check for, and the module actually has specific channels
        _, ok := c[SBatoi(channelID)]
        if !ok { continue; }
      }
      fn(i)
    }
  }
}

// This constructs an XOR operator for booleans
func boolXOR(a bool, b bool) bool {
  return (a && !b) || (!a && b)
}

func SBEvent(s *discordgo.Session, e *discordgo.Event) { ProcessModules(sb.hooks.OnEvent_channels, "", func(i int) { sb.hooks.OnEvent[i].OnEvent(s, e) }) }
func SBReady(s *discordgo.Session, r *discordgo.Ready) {
  fmt.Println("Ready message receieved")
  sb.SelfID = r.User.ID
  g := r.Guilds[0]
  ProcessGuild(g)
  
  for _, v := range g.Members {
    ProcessMember(v)
  }
  
  // We have to initialize commands and modules up here because they depend on the discord channel state
  sb.AddCommand(&EchoCommand{})
  sb.AddCommand(&HelpCommand{})
  
  GenChannels(len(sb.hooks.OnEvent), &sb.hooks.OnEvent_channels, func(i int) []string { return sb.hooks.OnEvent[i].Channels() })
  GenChannels(len(sb.hooks.OnTypingStart), &sb.hooks.OnTypingStart_channels, func(i int) []string { return sb.hooks.OnTypingStart[i].Channels() })
  GenChannels(len(sb.hooks.OnMessageCreate), &sb.hooks.OnMessageCreate_channels, func(i int) []string { return sb.hooks.OnMessageCreate[i].Channels() })
  GenChannels(len(sb.hooks.OnMessageUpdate), &sb.hooks.OnMessageUpdate_channels, func(i int) []string { return sb.hooks.OnMessageUpdate[i].Channels() })
  GenChannels(len(sb.hooks.OnMessageDelete), &sb.hooks.OnMessageDelete_channels, func(i int) []string { return sb.hooks.OnMessageDelete[i].Channels() })
  GenChannels(len(sb.hooks.OnMessageAck), &sb.hooks.OnMessageAck_channels, func(i int) []string { return sb.hooks.OnMessageAck[i].Channels() })
  GenChannels(len(sb.hooks.OnUserUpdate), &sb.hooks.OnUserUpdate_channels, func(i int) []string { return sb.hooks.OnUserUpdate[i].Channels() })
  GenChannels(len(sb.hooks.OnPresenceUpdate), &sb.hooks.OnPresenceUpdate_channels, func(i int) []string { return sb.hooks.OnPresenceUpdate[i].Channels() })
  GenChannels(len(sb.hooks.OnVoiceStateUpdate), &sb.hooks.OnVoiceStateUpdate_channels, func(i int) []string { return sb.hooks.OnVoiceStateUpdate[i].Channels() })
  GenChannels(len(sb.hooks.OnGuildUpdate), &sb.hooks.OnGuildUpdate_channels, func(i int) []string { return sb.hooks.OnGuildUpdate[i].Channels() })
  GenChannels(len(sb.hooks.OnGuildMemberAdd), &sb.hooks.OnGuildMemberAdd_channels, func(i int) []string { return sb.hooks.OnGuildMemberAdd[i].Channels() })
  GenChannels(len(sb.hooks.OnGuildMemberRemove), &sb.hooks.OnGuildMemberRemove_channels, func(i int) []string { return sb.hooks.OnGuildMemberRemove[i].Channels() })
  GenChannels(len(sb.hooks.OnGuildMemberUpdate), &sb.hooks.OnGuildMemberUpdate_channels, func(i int) []string { return sb.hooks.OnGuildMemberUpdate[i].Channels() })
  GenChannels(len(sb.hooks.OnGuildBanAdd), &sb.hooks.OnGuildBanAdd_channels, func(i int) []string { return sb.hooks.OnGuildBanAdd[i].Channels() })
  GenChannels(len(sb.hooks.OnGuildBanRemove), &sb.hooks.OnGuildBanRemove_channels, func(i int) []string { return sb.hooks.OnGuildBanRemove[i].Channels() })
  
  modules := ""
  commands := ""
  
  for _, v := range sb.modules {
    modules += "\n  "
    modules += v.Name() 
  }
  for _, v := range sb.commands {
    commands += "\n  "
    commands += v.c.Name() 
  }
    
  sb.log.Log("[](/sbload)\n Sweetiebot version ", sb.version, " successfully loaded on ", g.Name, ". \n\nActive Modules:", modules, "\n\nActive Commands:", commands);
}
func SBTypingStart(s *discordgo.Session, t *discordgo.TypingStart) { ProcessModules(sb.hooks.OnTypingStart_channels, "", func(i int) { sb.hooks.OnTypingStart[i].OnTypingStart(s, t) }) }
func SBMessageCreate(s *discordgo.Session, m *discordgo.Message) {
  if m.Author == nil { // This shouldn't ever happen but we check for it anyway
    return
  }
	//fmt.Printf("[%s] %20s %20s %s (%s:%s) > %s\n", m.ID, m.ChannelID, m.Timestamp, m.Author.Username, m.Author.ID, m.Author.Email, m.Content); // DEBUG
  
  if m.ChannelID != sb.LogChannelID { // Log this message provided it wasn't sent to the bot-log channel.
    sb.db.AddMessage(SBatoi(m.ID), SBatoi(m.Author.ID), m.ContentWithMentionsReplaced(), SBatoi(m.ChannelID), m.MentionEveryone) 
  }
  if m.Author.ID == sb.SelfID || m.ChannelID == sb.LogChannelID { // ALWAYS discard any of our own messages or our log messages before analysis.
    return
  }
  
  //if !boolXOR(sb.debug, m.ChannelID == sb.DebugChannelID) { // debug builds only respond to the debug channel, and release builds ignore it
  //  return
  //}
  
  // Check if this is a command. If it is, process it as a command, otherwise process it with our modules.
  if len(m.Content) > 1 && m.Content[0] == '!' { // We check for > 1 here because a single character can't possibly be a valid command
    t := time.Now().UTC().Unix()
    if sb.commandlimit.check(3, 20, t) { // if we've hit the saturation limit, post an error (which itself will only post if the error saturation limit hasn't been hit)
      sb.log.Error(m.ChannelID, "You can't input more than 3 commands every 20 seconds!")
      return
    }
    sb.commandlimit.append(t)
    
    args := ParseArguments(m.Content[1:])
    c, ok := sb.commands[strings.ToLower(args[0])]
    if ok {
      if !UserHasAnyRole(m.Author.ID, c.roles) {
        sb.log.Error(m.ChannelID, "You don't have permission to run this command! Allowed Roles: " + strings.Join(c.c.Roles(), ", "))
        return
      }
      s := c.c.Process(args[1:])
      if len(s) > 0 {
        sb.dg.ChannelMessageSend(m.ChannelID, s) 
      }
    } else {
      sb.log.Error(m.ChannelID, "Sorry, '" + args[0] + "' is not a valid command.\nFor a list of valid commands, type !help.")
    }
  } else {
    ProcessModules(sb.hooks.OnMessageCreate_channels, m.ChannelID, func(i int) { sb.hooks.OnMessageCreate[i].OnMessageCreate(s, m) })  
  }  
}

func SBMessageUpdate(s *discordgo.Session, m *discordgo.Message) {
  if m.Author == nil { // Discord sends an update message with an empty author when certain media links are posted
    return
  }
  if m.ChannelID != sb.LogChannelID { // Always ignore messages from the log channel
    sb.db.AddMessage(SBatoi(m.ID), SBatoi(m.Author.ID), m.ContentWithMentionsReplaced(), SBatoi(m.ChannelID), m.MentionEveryone) 
  }
  ProcessModules(sb.hooks.OnMessageUpdate_channels, m.ChannelID, func(i int) { sb.hooks.OnMessageUpdate[i].OnMessageUpdate(s, m) })
}
func SBMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
  ProcessModules(sb.hooks.OnMessageDelete_channels, m.ChannelID, func(i int) { sb.hooks.OnMessageDelete[i].OnMessageDelete(s, m) })
}
func SBMessageAck(s *discordgo.Session, m *discordgo.MessageAck) { ProcessModules(sb.hooks.OnMessageAck_channels, m.ChannelID, func(i int) { sb.hooks.OnMessageAck[i].OnMessageAck(s, m) }) }
func SBUserUpdate(s *discordgo.Session, u *discordgo.User) { ProcessUser(u); ProcessModules(sb.hooks.OnUserUpdate_channels, "", func(i int) { sb.hooks.OnUserUpdate[i].OnUserUpdate(s, u) }) }
func SBPresenceUpdate(s *discordgo.Session, p *discordgo.PresenceUpdate) { ProcessModules(sb.hooks.OnPresenceUpdate_channels, "", func(i int) { sb.hooks.OnPresenceUpdate[i].OnPresenceUpdate(s, p) }) }
func SBVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceState) { ProcessModules(sb.hooks.OnVoiceStateUpdate_channels, "", func(i int) { sb.hooks.OnVoiceStateUpdate[i].OnVoiceStateUpdate(s, v) }) }
func SBGuildUpdate(s *discordgo.Session, g *discordgo.Guild) {
  sb.log.Log("Guild update detected, updating ", g.Name)
  ProcessGuild(g)
  ProcessModules(sb.hooks.OnGuildUpdate_channels, "", func(i int) { sb.hooks.OnGuildUpdate[i].OnGuildUpdate(s, g) })
}
func SBGuildMemberAdd(s *discordgo.Session, u *discordgo.Member) { ProcessMember(u); ProcessModules(sb.hooks.OnGuildMemberAdd_channels, "", func(i int) { sb.hooks.OnGuildMemberAdd[i].OnGuildMemberAdd(s, u) }) }
func SBGuildMemberRemove(s *discordgo.Session, u *discordgo.Member) { ProcessModules(sb.hooks.OnGuildMemberRemove_channels, "", func(i int) { sb.hooks.OnGuildMemberRemove[i].OnGuildMemberRemove(s, u) }) }
func SBGuildMemberDelete(s *discordgo.Session, u *discordgo.Member) { SBGuildMemberRemove(s, u); }
func SBGuildMemberUpdate(s *discordgo.Session, u *discordgo.Member) { ProcessMember(u); ProcessModules(sb.hooks.OnGuildMemberUpdate_channels, "", func(i int) { sb.hooks.OnGuildMemberUpdate[i].OnGuildMemberUpdate(s, u) }) }
func SBGuildBanAdd(s *discordgo.Session, b *discordgo.GuildBan) { ProcessModules(sb.hooks.OnGuildBanAdd_channels, "", func(i int) { sb.hooks.OnGuildBanAdd[i].OnGuildBanAdd(s, b) }) }
func SBGuildBanRemove(s *discordgo.Session, b *discordgo.GuildBan) { ProcessModules(sb.hooks.OnGuildBanRemove_channels, "", func(i int) { sb.hooks.OnGuildBanRemove[i].OnGuildBanRemove(s, b) }) }

func UserHasRole(user string, role string) bool {
  m, err := sb.dg.State.Member(sb.GuildID, user)
  if err == nil {
    for _, v := range m.Roles {
      if v == role {
        return true
      }
    } 
  }
  return false
}

func UserHasAnyRole(user string, roles map[uint64]bool) bool {
  if len(roles) == 0 { return true }
  m, err := sb.dg.State.Member(sb.GuildID, user)
  if err == nil {
    for _, v := range m.Roles {
      _, ok := roles[SBatoi(v)]
      if ok {
        return true
      }
    }
  }
  return false
}

func ProcessUser(u *discordgo.User) uint64 {
  id := SBatoi(u.ID)
  sb.db.AddUser(id, u.Email, u.Username, u.Avatar, u.Verified)
  return id
}

func ProcessMember(u *discordgo.Member) {
  ProcessUser(u.User)
  
  if len(u.JoinedAt) > 0 { // Parse join date and update user table only if it is less than our current first seen date.
    t, err := time.Parse(time.RFC3339Nano, u.JoinedAt)
    if err == nil {
      sb.db.UpdateUserJoinTime(SBatoi(u.User.ID), t)
    } else {
      fmt.Println(err.Error())
    }
  }
}

func ProcessGuild(g *discordgo.Guild) {
  sb.GuildID = g.ID
  
  for _, v := range g.Channels {
    if v.Name == "bot-log" {
      sb.LogChannelID = v.ID
    }
    if v.Name == "ragemuffins" {
      sb.ModChannelID = v.ID
    }
    if v.Name == "bot-debug" {
      sb.DebugChannelID = v.ID
    }
  }
  for _, v := range g.Roles {
    if v.Name == "Silence" {
      sb.SilentRole = v.ID
    }
  }
}

func FindChannelID(name string) string {
  channels := sb.dg.State.Guilds[0].Channels 
  for _, v := range channels {
    if v.Name == name {
      return v.ID
    }
  }
  
  return ""
}

func GenChannels(length int, channels *[]map[uint64]bool, fn func(i int) []string) {
  for i := 0; i < length; i++ {
    channel := make(map[uint64]bool)
    c := fn(i)
    for j := 0; j < len(c); j++ {
      channel[SBatoi(FindChannelID(c[j]))] = true
    }
    
    *channels = append(*channels, channel)
  }
}

func Initialize() {  
  dbauth, _ := ioutil.ReadFile("db.auth")
  discorduser, _ := ioutil.ReadFile("username")  
  discordpass, _ := ioutil.ReadFile("passwd")
  log := &Log{}
  sb = &SweetieBot{
    version: "0.1.4",
    debug: true,
    commands: make(map[string]BotCommand),
    log: log,
    commandlimit: &SaturationLimit{make([]int64, 7, 7), 0, AtomicFlag{0}},
  }
  
  db, errdb := DB_Load(log, "mysql", string(dbauth))
  if errdb == nil {
    defer sb.db.Close();
  } else { 
    fmt.Println("Error loading database", errdb.Error())
    return 
  }
  
  sb.db = db 
  sb.dg = &discordgo.Session{
		State:                  discordgo.NewState(),
		StateEnabled:           true,
		//Compress:               true,
		//ShouldReconnectOnError: true,
    OnEvent: SBEvent,
    OnReady: SBReady,
    OnTypingStart: SBTypingStart,
    OnMessageCreate: SBMessageCreate,
    OnMessageUpdate: SBMessageUpdate,
    OnMessageDelete: SBMessageDelete,
    OnMessageAck: SBMessageAck,
    OnUserUpdate: SBUserUpdate,
    OnPresenceUpdate: SBPresenceUpdate,
    OnVoiceStateUpdate: SBVoiceStateUpdate,
    OnGuildUpdate: SBGuildUpdate,
    OnGuildMemberAdd: SBGuildMemberAdd,
    OnGuildMemberRemove: SBGuildMemberRemove,
    OnGuildMemberUpdate: SBGuildMemberUpdate,
    OnGuildBanAdd: SBGuildBanAdd,
    OnGuildBanRemove: SBGuildBanRemove,
  }
  
  log.Init(sb)
  sb.db.LoadStatements()
  log.Log("Finished loading database statements")
  
  sb.modules = append(sb.modules, &SpamModule{})
  sb.modules = append(sb.modules, &PingModule{})
  sb.modules = append(sb.modules, &EmoteModule{})
  sb.modules = append(sb.modules, &WittyModule{})
  
  for _, v := range sb.modules {
    v.Register(&sb.hooks)
  }
  
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

// HACK: taken out of discordgo
func GUILD_MEMBER(gID, uID string) string { return "https://discordapp.com/api/guilds/" + gID + "/members/" + uID }
  
func GuildMemberEdit(s *discordgo.Session, guildID string, userID string, roleIDs []string) (err error) {
  req := struct{
		Roles []string `json:"roles,omitempty"`
	}{
		Roles: roleIDs,
	}
  _, err = s.Request("PATCH", GUILD_MEMBER(guildID, userID), req)
  return err
}