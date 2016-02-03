package sweetiebot

import (
  "fmt"
  "strconv"
  "time"
  "io/ioutil"
  "github.com/bwmarrin/discordgo"
  "strings"
  "encoding/json"
  "reflect"
  "math/rand"
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
    OnCommand                 []ModuleOnCommand
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
    OnCommand_channels        []map[uint64]bool
}

type BotCommand struct {
  c Command
  roles map[uint64]bool
}

type BotConfig struct {
  Debug bool               `json:"debug"`
  Maxerror int64           `json:"maxerror"`
  Maxwit int64             `json:"maxwit"`
  Maxspam int              `json:"maxspam"`
  Maxbored int64           `json:"maxbored"`
  MaxPMlines int           `json:"maxpmlines"`
  Maxquotelines int        `json:"maxquotelines"`
  Maxmarkovlines int       `json:"maxmarkovlines"`
  Defaultmarkovlines int   `json:"defaultmarkovlines"`
  Commandperduration int   `json:"commandperduration"`
  Commandmaxduration int64 `json:"commandmaxduration"`
  Emotes []string          `json:"emotes"` // we can't unmarshal into a map, unfortunately
  Spoilers []string        `json:"spoilers"`
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
  ManeChannelID string
  SilentRole string
  version string
  hooks ModuleHooks
  modules []Module
  commands map[string]BotCommand
  commandlimit *SaturationLimit
  disablecommands map[string]bool
  princessrole map[uint64]bool
  quit bool
  config BotConfig
}

var sb *SweetieBot
var emotecommand *BanEmoteCommand

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

func (sbot *SweetieBot) SaveConfig() {
  data, err := json.Marshal(sb.config)
  if err == nil {
    ioutil.WriteFile("config.json", data, 0)
  } else {
    sbot.log.Log("Error writing json: ", err.Error())
  }
}

func (sbot *SweetieBot) SetConfig(name string, value string) (string, bool) {
  name = strings.ToLower(name)
  t := reflect.ValueOf(&sbot.config).Elem()
  n := t.NumField()
  for i := 0; i < n; i++ {
    if strings.ToLower(t.Type().Field(i).Name) == name {
      f := t.Field(i)
      switch t.Field(i).Interface().(type) {
        case string:
          f.SetString(value)
        case int, int8, int16, int32, int64:
          k, _ := strconv.ParseInt(value, 10, 64)
          f.SetInt(k)
        case uint, uint8, uint16, uint32, uint64:
          k, _ := strconv.ParseUint(value, 10, 64)
          f.SetUint(k)
        case bool:
          f.SetBool(value == "true")
        default:
          sbot.log.Log(name + " is an unknown type " + t.Field(i).Type().Name())
          return "", false
      }
      sbot.SaveConfig()
      return fmt.Sprint(t.Field(i).Interface()), true
    }
  }
  return "", false
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

func SBEvent(s *discordgo.Session, e *discordgo.Event) { ProcessModules(sb.hooks.OnEvent_channels, "", func(i int) { if(sb.hooks.OnEvent[i].IsEnabled()) { sb.hooks.OnEvent[i].OnEvent(s, e) } }) }
func SBReady(s *discordgo.Session, r *discordgo.Ready) {
  fmt.Println("Ready message receieved")
  sb.SelfID = r.User.ID
  g := r.Guilds[0]
  ProcessGuild(g)
  
  for _, v := range g.Members {
    ProcessMember(v)
  }
  
  for _, v := range sb.dg.State.Guilds[0].Roles {
    if v.Name == "Princess" {
      sb.princessrole[SBatoi(v.ID)] = true
      break
    }
  }
  
  // We have to initialize commands and modules up here because they depend on the discord channel state
  sb.AddCommand(&EchoCommand{})
  sb.AddCommand(&HelpCommand{})
  sb.AddCommand(&NewUsersCommand{})
  sb.AddCommand(&EnableCommand{})
  sb.AddCommand(&DisableCommand{})
  sb.AddCommand(&UpdateCommand{})
  sb.AddCommand(&AKACommand{})
  sb.AddCommand(&AboutCommand{})
  sb.AddCommand(&LastPingCommand{})
  sb.AddCommand(&SetConfigCommand{})
  sb.AddCommand(&GetConfigCommand{})
  sb.AddCommand(emotecommand)
  sb.AddCommand(&LastSeenCommand{})
  sb.AddCommand(&DumpTablesCommand{})
  sb.AddCommand(&EpisodeGenCommand{})
  sb.AddCommand(&QuoteCommand{})
  sb.AddCommand(&ShipCommand{})
  
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
  GenChannels(len(sb.hooks.OnCommand), &sb.hooks.OnCommand_channels, func(i int) []string { return sb.hooks.OnCommand[i].Channels() })

  sb.log.Log("[](/sbload)\n Sweetiebot version ", sb.version, " successfully loaded on ", g.Name, ". \n\n", GetActiveModules(), "\n\n", GetActiveCommands());
}

func SBTypingStart(s *discordgo.Session, t *discordgo.TypingStart) { ProcessModules(sb.hooks.OnTypingStart_channels, "", func(i int) { if(sb.hooks.OnTypingStart[i].IsEnabled()) { sb.hooks.OnTypingStart[i].OnTypingStart(s, t) } }) }
func SBMessageCreate(s *discordgo.Session, m *discordgo.Message) {
  if m.Author == nil { // This shouldn't ever happen but we check for it anyway
    return
  }
  
  if m.ChannelID != sb.LogChannelID { // Log this message provided it wasn't sent to the bot-log channel.
    sb.db.AddMessage(SBatoi(m.ID), SBatoi(m.Author.ID), m.ContentWithMentionsReplaced(), SBatoi(m.ChannelID), m.MentionEveryone) 
  }
  if m.Author.ID == sb.SelfID || m.ChannelID == sb.LogChannelID { // ALWAYS discard any of our own messages or our log messages before analysis.
    return
  }
  
  if boolXOR(sb.config.Debug, m.ChannelID == sb.DebugChannelID) { // debug builds only respond to the debug channel, and release builds ignore it
    return
  }
  
  // Check if this is a command. If it is, process it as a command, otherwise process it with our modules.
  if len(m.Content) > 1 && m.Content[0] == '!' && (len(m.Content) < 2 || m.Content[1] != '!') { // We check for > 1 here because a single character can't possibly be a valid command
    t := time.Now().UTC().Unix()
    ch, err := sb.dg.State.Channel(m.ChannelID)
    sb.log.LogError("Error retrieving channel ID " + m.ChannelID + ": ", err)
    
    if err != nil || (!ch.IsPrivate && m.ChannelID != sb.DebugChannelID) { // Private channels are not limited, nor is the debug channel
      if sb.commandlimit.check(sb.config.Commandperduration, sb.config.Commandmaxduration, t) { // if we've hit the saturation limit, post an error (which itself will only post if the error saturation limit hasn't been hit)
        sb.log.Error(m.ChannelID, "You can't input more than 3 commands every 30 seconds!")
        return
      }
      sb.commandlimit.append(t)
    }
    
    ignore := false
    ProcessModules(sb.hooks.OnCommand_channels, m.ChannelID, func(i int) { if(sb.hooks.OnCommand[i].IsEnabled()) { ignore = ignore || sb.hooks.OnCommand[i].OnCommand(s, m) } })  
    if ignore { // if true, a module wants us to ignore this command
      return
    }
    
    args := ParseArguments(m.Content[1:])
    c, ok := sb.commands[strings.ToLower(args[0])]
    if ok {
      subroles := c.roles
      _, ok := sb.disablecommands[c.c.Name()]
      if ok { subroles = sb.princessrole }
      if !UserHasAnyRole(m.Author.ID, subroles) {
        sb.log.Error(m.ChannelID, "You don't have permission to run this command! Allowed Roles: " + strings.Join(c.c.Roles(), ", "))
        return
      }
      result, usepm := c.c.Process(args[1:], m.Author)
      if len(result) > 0 {
        targetchannel := m.ChannelID
        if usepm && !ch.IsPrivate {
          channel, err := s.UserChannelCreate(m.Author.ID)
          sb.log.LogError("Error opening private channel: ", err);
          if err == nil {
            targetchannel = channel.ID
            if rand.Float32() < 0.01 {
              s.ChannelMessageSend(m.ChannelID, "Check your ~~privilege~~ Private Messages for my reply!")
            } else {
              s.ChannelMessageSend(m.ChannelID, "```Check your Private Messages for my reply!```")
            }
          }
        } 
        for len(result) > 1999 { // discord has a 2000 character limit
          index := strings.LastIndex(result[:1999], "\n")
          if index < 0 { index = 1999 }
          s.ChannelMessageSend(targetchannel, result[:index])
          result = result[index:]
        }
        s.ChannelMessageSend(targetchannel, result)
      }
    } else {
      sb.log.Error(m.ChannelID, "Sorry, '" + args[0] + "' is not a valid command.\nFor a list of valid commands, type !help.")
    }
  } else {
    ProcessModules(sb.hooks.OnMessageCreate_channels, m.ChannelID, func(i int) { if(sb.hooks.OnMessageCreate[i].IsEnabled()) { sb.hooks.OnMessageCreate[i].OnMessageCreate(s, m) } })  
  }  
}

func SBMessageUpdate(s *discordgo.Session, m *discordgo.Message) {
  if m.Author == nil { // Discord sends an update message with an empty author when certain media links are posted
    return
  }
  if m.ChannelID != sb.LogChannelID { // Always ignore messages from the log channel
    sb.db.AddMessage(SBatoi(m.ID), SBatoi(m.Author.ID), m.ContentWithMentionsReplaced(), SBatoi(m.ChannelID), m.MentionEveryone) 
  }
  ProcessModules(sb.hooks.OnMessageUpdate_channels, m.ChannelID, func(i int) { if(sb.hooks.OnMessageUpdate[i].IsEnabled()) { sb.hooks.OnMessageUpdate[i].OnMessageUpdate(s, m) } })
}
func SBMessageDelete(s *discordgo.Session, m *discordgo.Message) {
  ProcessModules(sb.hooks.OnMessageDelete_channels, m.ChannelID, func(i int) { if(sb.hooks.OnMessageDelete[i].IsEnabled()) { sb.hooks.OnMessageDelete[i].OnMessageDelete(s, m) } })
}
func SBMessageAck(s *discordgo.Session, m *discordgo.MessageAck) { ProcessModules(sb.hooks.OnMessageAck_channels, m.ChannelID, func(i int) { if(sb.hooks.OnMessageAck[i].IsEnabled()) { sb.hooks.OnMessageAck[i].OnMessageAck(s, m) } }) }
func SBUserUpdate(s *discordgo.Session, u *discordgo.User) { ProcessUser(u); ProcessModules(sb.hooks.OnUserUpdate_channels, "", func(i int) { if(sb.hooks.OnUserUpdate[i].IsEnabled()) { sb.hooks.OnUserUpdate[i].OnUserUpdate(s, u) } }) }
func SBPresenceUpdate(s *discordgo.Session, p *discordgo.PresenceUpdate) { ProcessUser(p.User); ProcessModules(sb.hooks.OnPresenceUpdate_channels, "", func(i int) { if(sb.hooks.OnPresenceUpdate[i].IsEnabled()) { sb.hooks.OnPresenceUpdate[i].OnPresenceUpdate(s, p) } }) }
func SBVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceState) { ProcessModules(sb.hooks.OnVoiceStateUpdate_channels, "", func(i int) { if(sb.hooks.OnVoiceStateUpdate[i].IsEnabled()) { sb.hooks.OnVoiceStateUpdate[i].OnVoiceStateUpdate(s, v) } }) }
func SBGuildUpdate(s *discordgo.Session, g *discordgo.Guild) {
  sb.log.Log("Guild update detected, updating ", g.Name)
  ProcessGuild(g)
  ProcessModules(sb.hooks.OnGuildUpdate_channels, "", func(i int) { if(sb.hooks.OnGuildUpdate[i].IsEnabled()) { sb.hooks.OnGuildUpdate[i].OnGuildUpdate(s, g) } })
}
func SBGuildMemberAdd(s *discordgo.Session, u *discordgo.Member) { ProcessMember(u); ProcessModules(sb.hooks.OnGuildMemberAdd_channels, "", func(i int) { if(sb.hooks.OnGuildMemberAdd[i].IsEnabled()) { sb.hooks.OnGuildMemberAdd[i].OnGuildMemberAdd(s, u) } }) }
func SBGuildMemberRemove(s *discordgo.Session, u *discordgo.Member) { ProcessModules(sb.hooks.OnGuildMemberRemove_channels, "", func(i int) { if(sb.hooks.OnGuildMemberRemove[i].IsEnabled()) { sb.hooks.OnGuildMemberRemove[i].OnGuildMemberRemove(s, u) } }) }
func SBGuildMemberDelete(s *discordgo.Session, u *discordgo.Member) { SBGuildMemberRemove(s, u); }
func SBGuildMemberUpdate(s *discordgo.Session, u *discordgo.Member) { ProcessMember(u); ProcessModules(sb.hooks.OnGuildMemberUpdate_channels, "", func(i int) { if(sb.hooks.OnGuildMemberUpdate[i].IsEnabled()) { sb.hooks.OnGuildMemberUpdate[i].OnGuildMemberUpdate(s, u) } }) }
func SBGuildBanAdd(s *discordgo.Session, b *discordgo.GuildBan) { ProcessModules(sb.hooks.OnGuildBanAdd_channels, "", func(i int) { if(sb.hooks.OnGuildBanAdd[i].IsEnabled()) { sb.hooks.OnGuildBanAdd[i].OnGuildBanAdd(s, b) } }) }
func SBGuildBanRemove(s *discordgo.Session, b *discordgo.GuildBan) { ProcessModules(sb.hooks.OnGuildBanRemove_channels, "", func(i int) { if(sb.hooks.OnGuildBanRemove[i].IsEnabled()) { sb.hooks.OnGuildBanRemove[i].OnGuildBanRemove(s, b) } }) }
func SBUserSettingsUpdate(s *discordgo.Session, m map[string]interface{}) { fmt.Println("OnUserSettingsUpdate called") }

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
    if v.Name == "manechat" {
      sb.ManeChannelID = v.ID
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
      id := FindChannelID(c[j])
      if len(id) > 0 {
        channel[SBatoi(id)] = true
      } else {
        sb.log.Log("Could not find channel ", c[j])
      }
    }
    
    *channels = append(*channels, channel)
  }
}

func WaitForInput() {
	var input string
	fmt.Scanln(&input)
	sb.quit = true
}

func Initialize() {  
  dbauth, _ := ioutil.ReadFile("db.auth")
  discorduser, _ := ioutil.ReadFile("username")  
  discordpass, _ := ioutil.ReadFile("passwd")
  config, _ := ioutil.ReadFile("config.json")

  sb = &SweetieBot{
    version: "0.3.3",
    commands: make(map[string]BotCommand),
    log: &Log{},
    commandlimit: &SaturationLimit{[]int64{}, 0, AtomicFlag{0}},
    disablecommands: make(map[string]bool),
  }
  
  errjson := json.Unmarshal(config, &sb.config)
  if errjson != nil { fmt.Println("Error reading config file: ", errjson.Error()) }
  fmt.Println("Config settings: ", sb.config)
  
  sb.commandlimit.times = make([]int64, sb.config.Commandperduration*2, sb.config.Commandperduration*2);
  
  db, errdb := DB_Load(sb.log, "mysql", strings.TrimSpace(string(dbauth)))
  if errdb != nil { 
    fmt.Println("Error loading database", errdb.Error())
    return 
  }
  
  sb.db = db 
  sb.dg = &discordgo.Session{
		State:                  discordgo.NewState(),
		StateEnabled:           true,
		Compress:               true,
		ShouldReconnectOnError: true,
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
    OnUserSettingsUpdate: SBUserSettingsUpdate,
  }
  
  sb.log.Init(sb)
  sb.db.LoadStatements()
  sb.log.Log("Finished loading database statements")
  emotecommand = &BanEmoteCommand{}

  //BuildMarkov(5, 20)
  //return
  
  sb.modules = append(sb.modules, &SpamModule{})
  sb.modules = append(sb.modules, &PingModule{})
  sb.modules = append(sb.modules, &emotecommand.emotes)
  sb.modules = append(sb.modules, &WittyModule{})
  sb.modules = append(sb.modules, &BoredModule{})
  
  for _, v := range sb.modules {
    v.Enable(true)
    v.Register(&sb.hooks)
  }
  
  err := sb.dg.Login(strings.TrimSpace(string(discorduser)), strings.TrimSpace(string(discordpass)))
  if err != nil {
    sb.log.LogError("Discord login failed: ", err)
    return; // this will close the db because we deferred db.Close()
  }
  sb.log.LogError("Error opening websocket connection: ", sb.dg.Open());
  //sb.log.LogError("Websocket handshake failure: ", sb.dg.Handshake());
  fmt.Println("Connection established");
  //sb.log.LogError("Connection error", sb.dg.Listen());
  
  if sb.config.Debug { // The server does not necessarily tie a standard input to the program
    go WaitForInput()
  }  
  for !sb.quit { time.Sleep(400 * time.Millisecond) }
  fmt.Println("Sweetiebot quitting");
  sb.db.Close();
}