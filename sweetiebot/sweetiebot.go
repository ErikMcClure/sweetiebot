package sweetiebot

import (
  "fmt"
  "strconv"
  "time"
  "io/ioutil"
  "github.com/bwmarrin/discordgo"
  "database/sql"
  "strings"
  "encoding/json"
  "reflect"
  "math/rand"
  "regexp"
  "encoding/base64"
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
    OnIdle                    []ModuleOnIdle
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
  Maxspoiltime int64       `json:"maxspoiltime"`       
  MaxPMlines int           `json:"maxpmlines"`
  Maxquotelines int        `json:"maxquotelines"`
  Maxmarkovlines int       `json:"maxmarkovlines"`
  Maxsearchresults int     `json:"maxsearchresults"`
  Defaultmarkovlines int   `json:"defaultmarkovlines"`
  Maxshutup int64          `json:"maxshutup"`
  Commandperduration int   `json:"commandperduration"`
  Commandmaxduration int64 `json:"commandmaxduration"`
  StatusDelayTime int      `json:"statusdelaytime"`
  Emotes []string          `json:"emotes"` // TODO: go can unmarshal into map[string] types now
  BoredLines []string      `json:"boredlines"`
  Spoilers []string        `json:"spoilers"`
  WittyTriggers []string   `json:"wittytriggers"`
  WittyRemarks []string    `json:"wittyremarks"`
  Schedule []time.Time     `json:"schedule"`
  Statuses []string        `json:"statuses"`
  Groups map[string]map[string]bool `json:"groups"`
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
  BotChannelID string
  SpoilerChannelID string
  SilentRole string
  lastshutup int64
  version string
  hooks ModuleHooks
  modules []Module
  commands map[string]BotCommand
  command_channels map[string]map[uint64]bool
  commandlimit *SaturationLimit
  disablecommands map[string]bool
  princessrole map[uint64]bool
  quit bool
  config BotConfig
  emotemodule *EmoteModule
  aliases map[string]string
}

var sb *SweetieBot
var channelregex = regexp.MustCompile("<#[0-9]+>")
var userregex = regexp.MustCompile("<@[0-9]+>")

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
  ch := c.Channels()
  channel := make(map[uint64]bool)
  for j := 0; j < len(ch); j++ {
    id := FindChannelID(ch[j])
    if len(id) > 0 {
      channel[SBatoi(id)] = true
    } else {
      sb.log.Log("Could not find channel ", ch[j])
    }
  }
  sbot.command_channels[c.Name()] = channel
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

func sbemotereplace(s string) string {
  return strings.Replace(s, "[](/", "[\u200B](/", -1)
}

var blockmessages = []string {"BUCK OFF", "ITS NOT WORKING", "GO AWAY", "STOP IT", "GIVE UP", "HAHAHAHA NO", "ABSOLUTELY NOT", "STOP BEING DUMB", "THIS IS SO STUPID", "BOTHER SOMEONE ELSE", "GET A JOB", "GET A LIFE"}
 
func SanitizeOutput(message string) string {
  message = strings.Replace(message, "@here", blockmessages[rand.Intn(len(blockmessages))], -1)
  message = strings.Replace(message, "@everyone", blockmessages[rand.Intn(len(blockmessages))], -1)
  message = strings.Replace(message, "@", "@\u200b", -1)
  message = sb.emotemodule.emoteban.ReplaceAllStringFunc(message, sbemotereplace)
  return message;
}
func (sbot *SweetieBot) SendMessage(channelID string, message string) {
  sbot.dg.ChannelMessageSend(channelID, SanitizeOutput(message));
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

func SwapStatusLoop() {
  for !sb.quit {
    sz := len(sb.config.Statuses)
    if sz > 0 {
      sb.dg.UpdateStatus(0, sb.config.Statuses[rand.Intn(sz)])
      fmt.Println("Changed Status")
    }
    time.Sleep(time.Duration(sb.config.StatusDelayTime)*time.Second)
  }
}

func ChangeBotName(s *discordgo.Session, name string, avatarfile string) {
  binary, _ := ioutil.ReadFile(avatarfile)
  email, _ := ioutil.ReadFile("email")
  password, _ := ioutil.ReadFile("passwd")
  avatar := base64.StdEncoding.EncodeToString(binary)
    
  _, err := s.UserUpdate(strings.TrimSpace(string(email)), strings.TrimSpace(string(password)), name, "data:image/jpeg;base64," + avatar, "")
  if err != nil {
    fmt.Println(err.Error())
  } else {
    fmt.Println("Changed username successfully")
  }
}

//func SBEvent(s *discordgo.Session, e *discordgo.Event) { ProcessModules(sb.hooks.OnEvent_channels, "", func(i int) { if(sb.hooks.OnEvent[i].IsEnabled()) { sb.hooks.OnEvent[i].OnEvent(s, e) } }) }
func SBReady(s *discordgo.Session, r *discordgo.Ready) {
  fmt.Println("Ready message receieved, waiting for guilds...")
  go SwapStatusLoop()
  sb.SelfID = r.User.ID
  
  // Only used to change sweetiebot's name or avatar
  //ChangeBotName(s, "Sweetie", "avatar.jpg")
}

func AttachToGuild(g *discordgo.Guild) {
  fmt.Println("Initializing...")
  ProcessGuild(g);
  
  for _, v := range g.Members {
    ProcessMember(v)
  }
  
  for _, v := range g.Roles {
    if v.Name == "Princesses" {
      sb.princessrole[SBatoi(v.ID)] = true
      break
    }
  }
  
  episodegencommand := &EpisodeGenCommand{}
  sb.emotemodule = &EmoteModule{}
  spoilermodule := &SpoilerModule{}
  wittymodule := &WittyModule{}
  sb.modules = append(sb.modules, &SpamModule{})
  sb.modules = append(sb.modules, &PingModule{})
  sb.modules = append(sb.modules, sb.emotemodule)
  sb.modules = append(sb.modules, wittymodule)
  sb.modules = append(sb.modules, &BoredModule{Episodegen: episodegencommand})
  sb.modules = append(sb.modules, spoilermodule)
  
  for _, v := range sb.modules {
    v.Enable(true)
    v.Register(&sb.hooks)
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
  sb.AddCommand(&BanEmoteCommand{sb.emotemodule})
  sb.AddCommand(&LastSeenCommand{})
  sb.AddCommand(&DumpTablesCommand{})
  sb.AddCommand(episodegencommand)
  sb.AddCommand(&QuoteCommand{})
  sb.AddCommand(&ShipCommand{})
  sb.AddCommand(&AddBoredCommand{})
  sb.AddCommand(&AddSpoilerCommand{spoilermodule})
  sb.AddCommand(&AddWitCommand{wittymodule})
  sb.AddCommand(&SearchCommand{emotes: sb.emotemodule, statements: make(map[string][]*sql.Stmt)})
  sb.AddCommand(&AddStatusCommand{})
  sb.AddCommand(&SetStatusCommand{})
  sb.AddCommand(&AddGroupCommand{})
  sb.AddCommand(&JoinGroupCommand{})
  sb.AddCommand(&ListGroupCommand{})
  sb.AddCommand(&LeaveGroupCommand{})
  sb.AddCommand(&PingCommand{})
  sb.AddCommand(&PurgeGroupCommand{})
  
  sb.aliases = make(map[string]string)
  sb.aliases["listgroups"] = "listgroup"
  
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

  go IdleCheckLoop()
  
  debug := ". \n\n"
  if sb.config.Debug {
    debug = ".\n[DEBUG BUILD]\n\n"
  }
  sb.log.Log("[](/sbload)\n Sweetiebot version ", sb.version, " successfully loaded on ", g.Name, debug, GetActiveModules(), "\n\n", GetActiveCommands());
}

func SBTypingStart(s *discordgo.Session, t *discordgo.TypingStart) { ProcessModules(sb.hooks.OnTypingStart_channels, "", func(i int) { if(sb.hooks.OnTypingStart[i].IsEnabled()) { sb.hooks.OnTypingStart[i].OnTypingStart(s, t) } }) }
func SBMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
  if m.Author == nil { // This shouldn't ever happen but we check for it anyway
    return
  }
  
  ch, err := sb.dg.State.Channel(m.ChannelID)
  sb.log.LogError("Error retrieving channel ID " + m.ChannelID + ": ", err)
  
  if m.ChannelID != sb.LogChannelID && !ch.IsPrivate { // Log this message provided it wasn't sent to the bot-log channel or in a PM
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
    
    if err != nil || (!ch.IsPrivate && m.ChannelID != sb.DebugChannelID && m.ChannelID != sb.BotChannelID) { // Private channels are not limited, nor is the debug channel
      if sb.commandlimit.check(sb.config.Commandperduration, sb.config.Commandmaxduration, t) { // if we've hit the saturation limit, post an error (which itself will only post if the error saturation limit hasn't been hit)
        sb.log.Error(m.ChannelID, "You can't input more than 3 commands every 30 seconds!")
        return
      }
      sb.commandlimit.append(t)
    }
    
    ignore := false
    ProcessModules(sb.hooks.OnCommand_channels, m.ChannelID, func(i int) { if(sb.hooks.OnCommand[i].IsEnabled()) { ignore = ignore || sb.hooks.OnCommand[i].OnCommand(s, m.Message) } })  
    if ignore { // if true, a module wants us to ignore this command
      return
    }
    
    args := ParseArguments(m.Content[1:])
    arg := strings.ToLower(args[0])
    alias, ok := sb.aliases[arg]
    if ok { arg = alias }
    c, ok := sb.commands[arg]    
    if ok {
      cch := sb.command_channels[c.c.Name()]
      if !ch.IsPrivate && len(cch) > 0 {
        _, ok = cch[SBatoi(m.ChannelID)]
        if !ok {
          return
        }
      }
      subroles := c.roles
      _, ok := sb.disablecommands[c.c.Name()]
      if ok { subroles = sb.princessrole }
      if !UserHasAnyRole(m.Author.ID, subroles) {
        sb.log.Error(m.ChannelID, "You don't have permission to run this command! Allowed Roles: " + strings.Join(c.c.Roles(), ", "))
        return
      }
      result, usepm := c.c.Process(args[1:], m.Message)
      if len(result) > 0 {
        targetchannel := m.ChannelID
        if usepm && !ch.IsPrivate {
          channel, err := s.UserChannelCreate(m.Author.ID)
          sb.log.LogError("Error opening private channel: ", err);
          if err == nil {
            targetchannel = channel.ID
            if rand.Float32() < 0.01 {
              sb.SendMessage(m.ChannelID, "Check your ~~privilege~~ Private Messages for my reply!")
            } else {
              sb.SendMessage(m.ChannelID, "```Check your Private Messages for my reply!```")
            }
          }
        } 
        for len(result) > 1999 { // discord has a 2000 character limit
          index := strings.LastIndex(result[:1999], "\n")
          if index < 0 { index = 1999 }
          sb.SendMessage(targetchannel, result[:index])
          result = result[index:]
        }
        sb.SendMessage(targetchannel, result)
      }
    } else {
      if args[0] != "airhorn" {
        sb.log.Error(m.ChannelID, "Sorry, " + args[0] + " is not a valid command.\nFor a list of valid commands, type !help.")
      }
    }
  } else {
    ProcessModules(sb.hooks.OnMessageCreate_channels, m.ChannelID, func(i int) { if(sb.hooks.OnMessageCreate[i].IsEnabled()) { sb.hooks.OnMessageCreate[i].OnMessageCreate(s, m.Message) } })  
  }  
}

func SBMessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
  if boolXOR(sb.config.Debug, m.ChannelID == sb.DebugChannelID) { return }
  if m.Author == nil { // Discord sends an update message with an empty author when certain media links are posted
    return
  }
  if m.ChannelID != sb.LogChannelID { // Always ignore messages from the log channel
    sb.db.AddMessage(SBatoi(m.ID), SBatoi(m.Author.ID), m.ContentWithMentionsReplaced(), SBatoi(m.ChannelID), m.MentionEveryone) 
  }
  ProcessModules(sb.hooks.OnMessageUpdate_channels, m.ChannelID, func(i int) { if(sb.hooks.OnMessageUpdate[i].IsEnabled()) { sb.hooks.OnMessageUpdate[i].OnMessageUpdate(s, m.Message) } })
}
func SBMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
  if boolXOR(sb.config.Debug, m.ChannelID == sb.DebugChannelID) { return }
  ProcessModules(sb.hooks.OnMessageDelete_channels, m.ChannelID, func(i int) { if(sb.hooks.OnMessageDelete[i].IsEnabled()) { sb.hooks.OnMessageDelete[i].OnMessageDelete(s, m.Message) } })
}
func SBMessageAck(s *discordgo.Session, m *discordgo.MessageAck) { ProcessModules(sb.hooks.OnMessageAck_channels, m.ChannelID, func(i int) { if(sb.hooks.OnMessageAck[i].IsEnabled()) { sb.hooks.OnMessageAck[i].OnMessageAck(s, m) } }) }
func SBUserUpdate(s *discordgo.Session, m *discordgo.UserUpdate) { ProcessUser(m.User); ProcessModules(sb.hooks.OnUserUpdate_channels, "", func(i int) { if(sb.hooks.OnUserUpdate[i].IsEnabled()) { sb.hooks.OnUserUpdate[i].OnUserUpdate(s, m.User) } }) }
func SBUserSettingsUpdate(s *discordgo.Session, m *discordgo.UserSettingsUpdate) { fmt.Println("OnUserSettingsUpdate called") }
func SBPresenceUpdate(s *discordgo.Session, m *discordgo.PresenceUpdate) { ProcessUser(m.User); ProcessModules(sb.hooks.OnPresenceUpdate_channels, "", func(i int) { if(sb.hooks.OnPresenceUpdate[i].IsEnabled()) { sb.hooks.OnPresenceUpdate[i].OnPresenceUpdate(s, m) } }) }
func SBVoiceStateUpdate(s *discordgo.Session, m *discordgo.VoiceStateUpdate) { ProcessModules(sb.hooks.OnVoiceStateUpdate_channels, "", func(i int) { if(sb.hooks.OnVoiceStateUpdate[i].IsEnabled()) { sb.hooks.OnVoiceStateUpdate[i].OnVoiceStateUpdate(s, m.VoiceState) } }) }
func SBGuildUpdate(s *discordgo.Session, m *discordgo.GuildUpdate) {
  sb.log.Log("Guild update detected, updating ", m.Name)
  ProcessGuild(m.Guild)
  ProcessModules(sb.hooks.OnGuildUpdate_channels, "", func(i int) { if(sb.hooks.OnGuildUpdate[i].IsEnabled()) { sb.hooks.OnGuildUpdate[i].OnGuildUpdate(s, m.Guild) } })
}
func SBGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) { ProcessMember(m.Member); ProcessModules(sb.hooks.OnGuildMemberAdd_channels, "", func(i int) { if(sb.hooks.OnGuildMemberAdd[i].IsEnabled()) { sb.hooks.OnGuildMemberAdd[i].OnGuildMemberAdd(s, m.Member) } }) }
func SBGuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove) { ProcessModules(sb.hooks.OnGuildMemberRemove_channels, "", func(i int) { if(sb.hooks.OnGuildMemberRemove[i].IsEnabled()) { sb.hooks.OnGuildMemberRemove[i].OnGuildMemberRemove(s, m.Member) } }) }
func SBGuildMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) { ProcessMember(m.Member); ProcessModules(sb.hooks.OnGuildMemberUpdate_channels, "", func(i int) { if(sb.hooks.OnGuildMemberUpdate[i].IsEnabled()) { sb.hooks.OnGuildMemberUpdate[i].OnGuildMemberUpdate(s, m.Member) } }) }
func SBGuildBanAdd(s *discordgo.Session, m *discordgo.GuildBanAdd) { ProcessModules(sb.hooks.OnGuildBanAdd_channels, "", func(i int) { if(sb.hooks.OnGuildBanAdd[i].IsEnabled()) { sb.hooks.OnGuildBanAdd[i].OnGuildBanAdd(s, m.GuildBan) } }) }
func SBGuildBanRemove(s *discordgo.Session, m *discordgo.GuildBanRemove) { ProcessModules(sb.hooks.OnGuildBanRemove_channels, "", func(i int) { if(sb.hooks.OnGuildBanRemove[i].IsEnabled()) { sb.hooks.OnGuildBanRemove[i].OnGuildBanRemove(s, m.GuildBan) } }) }
func SBGuildCreate(s *discordgo.Session, m *discordgo.GuildCreate) { ProcessGuildCreate(m.Guild) }

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

func ProcessGuildCreate(g *discordgo.Guild) {
  AttachToGuild(g);
}

func ProcessGuild(g *discordgo.Guild) {
  sb.GuildID = g.ID
  
  for _, v := range g.Channels {
    switch v.Name {
      case "bot-log":
        sb.LogChannelID = v.ID
      case "ragemuffins":
        sb.ModChannelID = v.ID
      case "bot-debug":
        sb.DebugChannelID = v.ID
      case "example":
        sb.ManeChannelID = v.ID
      case "mylittlebot":
        sb.BotChannelID = v.ID
      case "mylittlespoilers":
        sb.SpoilerChannelID = v.ID
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

func IdleCheckLoop() {
  for !sb.quit {
    c, _ := sb.dg.State.Channel(sb.ManeChannelID)
    t := sb.db.GetLatestMessage(SBatoi(sb.ManeChannelID))
    diff := SinceUTC(t);
    for _, v := range sb.hooks.OnIdle {
      if v.IsEnabled() && diff >= (time.Duration(v.IdlePeriod())*time.Second) {
        v.OnIdle(sb.dg, c);
      }
    }
    time.Sleep(30*time.Second)
  }  
}

func WaitForInput() {
	var input string
	fmt.Scanln(&input)
	sb.quit = true
}

func Initialize(Token string) {  
  dbauth, _ := ioutil.ReadFile("db.auth")
  config, _ := ioutil.ReadFile("config.json")

  sb = &SweetieBot{
    version: "0.5.1",
    commands: make(map[string]BotCommand),
    command_channels: make(map[string]map[uint64]bool),
    log: &Log{},
    commandlimit: &SaturationLimit{[]int64{}, 0, AtomicFlag{0}},
    disablecommands: make(map[string]bool),
    princessrole: make(map[uint64]bool),
    lastshutup: 0,
  }
  
  errjson := json.Unmarshal(config, &sb.config)
  if errjson != nil { fmt.Println("Error reading config file: ", errjson.Error()) }
  fmt.Println("Config settings: ", sb.config)
  
  sb.commandlimit.times = make([]int64, sb.config.Commandperduration*2, sb.config.Commandperduration*2);
  
  db, err := DB_Load(sb.log, "mysql", strings.TrimSpace(string(dbauth)))
  if err != nil { 
    fmt.Println("Error loading database", err.Error())
    return 
  }
  
  sb.db = db 
  sb.dg, err = discordgo.New(Token)
  if err != nil {
    fmt.Println("Error creating discord session", err.Error())
    return
  }
  
  sb.dg.AddHandler(SBReady)
  sb.dg.AddHandler(SBTypingStart)
  sb.dg.AddHandler(SBMessageCreate)
  sb.dg.AddHandler(SBMessageUpdate)
  sb.dg.AddHandler(SBMessageDelete)
  sb.dg.AddHandler(SBMessageAck)
  sb.dg.AddHandler(SBUserUpdate)
  sb.dg.AddHandler(SBUserSettingsUpdate)
  sb.dg.AddHandler(SBPresenceUpdate)
  sb.dg.AddHandler(SBVoiceStateUpdate)
  sb.dg.AddHandler(SBGuildUpdate)
  sb.dg.AddHandler(SBGuildMemberAdd)
  sb.dg.AddHandler(SBGuildMemberRemove)
  sb.dg.AddHandler(SBGuildMemberUpdate)
  sb.dg.AddHandler(SBGuildBanAdd)
  sb.dg.AddHandler(SBGuildBanRemove)
  sb.dg.AddHandler(SBGuildCreate)
  
  sb.log.Init(sb)
  sb.db.LoadStatements()
  sb.log.Log("Finished loading database statements")

  // Code to fix uppercase groups
  for k, v := range sb.config.Groups {
      l := strings.ToLower(k)
      _, ok := sb.config.Groups[l]
      if !ok {
        delete(sb.config.Groups, k)
        sb.config.Groups[l] = v
      }
  }
  sb.SaveConfig()
  
  //BuildMarkov(5, 20)
  //return
  
  sb.log.LogError("Error opening websocket connection: ", sb.dg.Open());
  fmt.Println("Connection established");
  
  if sb.config.Debug { // The server does not necessarily tie a standard input to the program
    go WaitForInput()
  }  
  for !sb.quit { time.Sleep(400 * time.Millisecond) }
  fmt.Println("Sweetiebot quitting");
  sb.db.Close();
}