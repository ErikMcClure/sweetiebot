package sweetiebot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blackhole12/discordgo"
)

type Logger interface {
	Log(args ...interface{})
	LogError(msg string, err error)
}

type GuildInfo struct {
	ID           string // Cache the ID because it doesn't change
	Name         string // Cache the name to reduce locking
	OwnerID      string
	lastlogerr   int64
	commandLock  sync.RWMutex
	command_last map[string]map[string]int64
	commandlimit *SaturationLimit
	config       BotConfig
	emotemodule  *EmoteModule
	hooks        ModuleHooks
	modules      []Module
	commands     map[string]Command
	lockdown     discordgo.VerificationLevel // if -1 no lockdown was initiated, otherwise remembers the previous lockdown setting
	lastlockdown time.Time
}

func (info *GuildInfo) AddCommand(c Command) {
	info.commands[strings.ToLower(c.Name())] = c
}

func (info *GuildInfo) SaveConfig() {
	data, err := json.Marshal(info.config)
	if err == nil {
		if len(data) > sb.MaxConfigSize {
			info.Log("Error saving config file: Config file is too large! Config files cannot exceed " + strconv.Itoa(sb.MaxConfigSize) + " bytes.")
		} else {
			err = ioutil.WriteFile(info.ID+".json", data, 0664)
			if err != nil {
				info.Log("Error saving config file: ", err.Error())
			}
		}
	} else {
		info.Log("Error writing json: ", err.Error())
	}
}

func deleteFromMapReflect(f reflect.Value, k string) string {
	f.SetMapIndex(reflect.ValueOf(k), reflect.Value{})
	return "Deleted " + k
}

func (info *GuildInfo) SetConfig(name string, value string, extra ...string) (string, bool) {
	names := strings.SplitN(strings.ToLower(name), ".", 3)
	t := reflect.ValueOf(&info.config).Elem()
	for i := 0; i < t.NumField(); i++ {
		if strings.ToLower(t.Type().Field(i).Name) == names[0] {
			if len(names) < 2 {
				return "Can't set a configuration category! Use \"Category.Option\" to set a specific option.", false
			}
			switch t.Field(i).Kind() {
			case reflect.Struct:
				for j := 0; j < t.Field(i).NumField(); j++ {
					if strings.ToLower(t.Field(i).Type().Field(j).Name) == names[1] {
						f := t.Field(i).Field(j)
						switch f.Interface().(type) {
						case string:
							f.SetString(value)
						case int, int8, int16, int32, int64:
							k, _ := strconv.ParseInt(value, 10, 64)
							f.SetInt(k)
						case uint, uint8, uint16, uint32:
							k, _ := strconv.ParseUint(value, 10, 64)
							f.SetUint(k)
						case float32, float64:
							k, _ := strconv.ParseFloat(value, 32)
							f.SetFloat(k)
						case uint64:
							f.SetUint(PingAtoi(value))
						case []uint64:
							f.Set(reflect.MakeSlice(reflect.TypeOf(f.Interface()), 0, 1+len(extra)))
							if len(value) > 0 {
								f.Set(reflect.Append(f, reflect.ValueOf(PingAtoi(value))))
								for _, k := range extra {
									f.Set(reflect.Append(f, reflect.ValueOf(PingAtoi(k))))
								}
							}
						case bool:
							f.SetBool(value == "true")
						case map[string]string:
							value = strings.ToLower(value)
							if len(extra) == 0 {
								return "No extra parameter given for " + name, false
							}
							if f.IsNil() {
								f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
							}
							if len(extra[0]) == 0 {
								return deleteFromMapReflect(f, value), false
							}

							f.SetMapIndex(reflect.ValueOf(value), reflect.ValueOf(extra[0]))
							return value + ": " + extra[0], true
						case map[string]int64:
							value = strings.ToLower(value)
							if len(extra) == 0 {
								return "No extra parameter given for " + name, false
							}
							if f.IsNil() {
								f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
							}
							if len(extra[0]) == 0 {
								return deleteFromMapReflect(f, value), false
							}

							k, _ := strconv.ParseInt(extra[0], 10, 64)
							f.SetMapIndex(reflect.ValueOf(value), reflect.ValueOf(k))
							return value + ": " + strconv.FormatInt(k, 10), true
						case map[int64]int:
							ivalue, err := strconv.ParseInt(value, 10, 64)
							if err != nil {
								return value + " is not an integer.", false
							}
							if len(extra) == 0 {
								return "No extra parameter given for " + name, false
							}
							if f.IsNil() {
								f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
							}
							if len(extra[0]) == 0 {
								f.SetMapIndex(reflect.ValueOf(ivalue), reflect.Value{})
								return "Deleted " + value, false
							}

							k, _ := strconv.Atoi(extra[0])
							f.SetMapIndex(reflect.ValueOf(ivalue), reflect.ValueOf(k))
							return value + ": " + strconv.Itoa(k), true
						case map[uint64]float32:
							ivalue := PingAtoi(value)
							if ivalue == 0 {
								return value + " is not an integer.", false
							}
							if len(extra) == 0 {
								return "No extra parameter given for " + name, false
							}
							if f.IsNil() {
								f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
							}
							if len(extra[0]) == 0 {
								f.SetMapIndex(reflect.ValueOf(ivalue), reflect.Value{})
								return "Deleted " + value, false
							}

							k, _ := strconv.ParseFloat(extra[0], 32)
							f.SetMapIndex(reflect.ValueOf(ivalue), reflect.ValueOf(float32(k)))
							return fmt.Sprintf("%s: %f", value, k), true
						case map[int]string:
							ivalue, err := strconv.Atoi(value)
							if err != nil {
								return value + " is not an integer.", false
							}
							if len(extra) == 0 {
								return "No extra parameter given for " + name, false
							}
							if f.IsNil() {
								f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
							}
							if len(extra[0]) == 0 {
								f.SetMapIndex(reflect.ValueOf(ivalue), reflect.Value{})
								return "Deleted " + value, false
							}

							e := strings.Join(extra, " ")
							f.SetMapIndex(reflect.ValueOf(ivalue), reflect.ValueOf(e))
							return value + ": " + e, true
						case map[string]bool:
							f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
							f.SetMapIndex(reflect.ValueOf(StripPing(value)), reflect.ValueOf(true))
							stripped := []string{StripPing(value)}
							for _, k := range extra {
								f.SetMapIndex(reflect.ValueOf(StripPing(k)), reflect.ValueOf(true))
								stripped = append(stripped, StripPing(k))
							}
							return "[" + strings.Join(stripped, ", ") + "]", true
						case map[string]map[string]bool:
							value = strings.ToLower(value)
							if f.IsNil() {
								f.Set(reflect.MakeMap(reflect.TypeOf(f.Interface())))
							}
							if len(extra) == 0 {
								return deleteFromMapReflect(f, value), false
							}

							m := reflect.MakeMap(reflect.TypeOf(f.Interface()).Elem())
							stripped := []string{}
							for _, k := range extra {
								m.SetMapIndex(reflect.ValueOf(StripPing(k)), reflect.ValueOf(true))
								stripped = append(stripped, StripPing(k))
							}
							f.SetMapIndex(reflect.ValueOf(value), m)
							return value + ": [" + strings.Join(stripped, ", ") + "]", true
						default:
							info.Log(name + " is an unknown type " + f.Type().Name())
							return "That config option has an unknown type!", false
						}
						return fmt.Sprint(f.Interface()), true
					}
				}
			default:
				return "Not a configuration category!", false
			}
		}
	}
	return "Could not find configuration parameter " + name + "!", false
}

func sbemotereplace(s string) string {
	return strings.Replace(s, "[](/", "[\u200B](/", -1)
}

func (info *GuildInfo) sanitizeOutput(message string) string {
	if info.emotemodule != nil && info.emotemodule.emoteban != nil {
		message = info.emotemodule.emoteban.ReplaceAllStringFunc(message, sbemotereplace)
	}
	return message
}

func (info *GuildInfo) SendEmbed(channelID string, embed *discordgo.MessageEmbed) bool {
	ch, private := channelIsPrivate(channelID)
	if !private && ch.GuildID != info.ID {
		if SBatoi(channelID) != info.config.Log.Channel {
			info.Log("Attempted to send message to ", channelID, ", which isn't on this server.")
		}
		return false
	}
	if channelID == "heartbeat" {
		atomic.AddUint32(&sb.heartbeat, 1)
	} else {
		fields := embed.Fields
		for len(fields) > 25 {
			embed.Fields = fields[:25]
			fields = fields[25:]
			sb.dg.ChannelMessageSendEmbed(channelID, embed)
		}
		embed.Fields = fields
		sb.dg.ChannelMessageSendEmbed(channelID, embed)
	}
	return true
}

type sbRequestBuffer struct {
	buffer []*discordgo.MessageSend
	count  int
}

func (b *sbRequestBuffer) Append(m *discordgo.MessageSend) int {
	b.buffer = append(b.buffer, m)
	b.count += len(m.Content)
	if b.count+len(b.buffer) >= 1999 { // add one for each message in the buffer for added newlines
		return len(b.buffer)
	}
	return 0
}

func (b *sbRequestBuffer) Process() (*discordgo.MessageSend, int) {
	if len(b.buffer) < 1 {
		return nil, 0
	}

	if len(b.buffer) == 1 {
		msg := b.buffer[0]
		b.buffer = nil
		b.count = 0
		return msg, 0
	}

	count := len(b.buffer[0].Content)
	msg := make([]string, 1, len(b.buffer))
	msg[0] = b.buffer[0].Content
	i := 1

	for i < len(b.buffer) && (count+i+len(b.buffer[i].Content)) < 2000 {
		msg = append(msg, b.buffer[i].Content)
		count += len(b.buffer[i].Content)
		i++
	}

	b.count -= count
	if i >= len(b.buffer) {
		b.buffer = nil
	} else {
		b.buffer = b.buffer[i:]
	}

	return &discordgo.MessageSend{
		Content: strings.Join(msg, "\n"),
	}, len(b.buffer)
}

// RequestPostWithBuffer uses a buffer and a buffer combination function to combine multiple messages if there are fewer than minRequests requests left in the current bucket
func (info *GuildInfo) RequestPostWithBuffer(urlStr string, data *discordgo.MessageSend, minRemaining int) (response []byte, err error) {
	b := sb.dg.Ratelimiter.GetBucket(urlStr)
	b.Lock()
	if b.Userdata == nil {
		b.Userdata = &sbRequestBuffer{nil, 0}
	}
	buffer := b.Userdata.(*sbRequestBuffer)

	// data can be nil here, which tells the buffer to check if it's full
	remain := buffer.Append(data)
	softwait := sb.dg.Ratelimiter.GetWaitTime(b, minRemaining)

	if remain == 0 && softwait > 0 {
		b.Release(nil)
		time.Sleep(softwait)
		b.Lock()
	}

	for {
		data, remain = buffer.Process()

		if data != nil {
			if wait := sb.dg.Ratelimiter.GetWaitTime(b, 1); wait > 0 {
				fmt.Printf("Hit rate limit in buffered request, sleeping for %v (%v remaining)\n", wait, remain)
				time.Sleep(wait)
			}

			b.Remaining--
			softwait = sb.dg.Ratelimiter.GetWaitTime(b, minRemaining)
			var body []byte
			body, err = json.Marshal(data)
			if err == nil {
				response, err = sb.dg.RequestWithLockedBucket("POST", urlStr, "application/json", body, b, 0)
			} else {
				b.Release(nil)
				break
			}
		} else {
			b.Release(nil)
			break
		}

		// If we have nothing left to do, bail out early to avoid extra work
		if remain == 0 {
			break
		}

		// If we ran out of breathing room on our bucket, sleep until the end of the soft limit
		if softwait > 0 {
			time.Sleep(softwait)
		}
		b.Lock() // Re-lock the bucket
	}

	return
}

func (info *GuildInfo) sendContent(channelID string, message string, minRequest int) {
	_, err := info.RequestPostWithBuffer(discordgo.EndpointChannelMessages(channelID), &discordgo.MessageSend{
		Content: info.sanitizeOutput(message),
	}, minRequest)
	if err != nil {
		fmt.Println("Failed to send message: ", err.Error())
	}
}

func (info *GuildInfo) SendMessage(channelID string, message string) bool {
	ch, private := channelIsPrivate(channelID)
	if !private && ch.GuildID != info.ID {
		if SBatoi(channelID) != info.config.Log.Channel {
			info.Log("Attempted to send message to ", channelID, ", which isn't on this server.")
		}
		return false
	}

	for len(message) > 1999 { // discord has a 2000 character limit
		if message[0:3] == "```" && message[len(message)-3:] == "```" {
			index := strings.LastIndex(message[:1995], "\n")
			if index < 10 { // Ensure we process at least 10 characters to prevent an infinite loop
				index = 1995
			}
			info.sendContent(channelID, message[:index]+"```", 1)
			message = "```\n" + message[index:]
		} else {
			index := strings.LastIndex(message[:1999], "\n")
			if index < 10 {
				index = 1999
			}
			info.sendContent(channelID, message[:index], 1)
			message = message[index:]
		}
	}
	go info.sendContent(channelID, message, 2)

	//sb.dg.ChannelMessageSend(channelID, info.sanitizeOutput(message))
	return true
}

func (info *GuildInfo) ProcessModule(channelID string, m Module) bool {
	_, disabled := info.config.Modules.Disabled[strings.ToLower(m.Name())]
	if disabled {
		return false
	}

	c := info.config.Modules.Channels[strings.ToLower(m.Name())]
	if len(channelID) > 0 && len(c) > 0 { // Only check for channels if we have a channel to check for, and the module actually has specific channels
		_, reverse := c["!"]
		_, ok := c[channelID]
		return ok != reverse
	}
	return true
}

func (info *GuildInfo) SwapStatusLoop() {
	if sb.IsMainGuild(info) {
		for !sb.quit.get() {
			d := info.config.Status.Cooldown
			if d < 1 {
				d = 1
			}
			time.Sleep(time.Duration(d) * time.Second) // Prevent you from setting this to 0 because that's bad
			if len(info.config.Basic.Collections["status"]) > 0 {
				sb.dg.UpdateStatus(0, MapGetRandomItem(info.config.Basic.Collections["status"]))
			}
		}
	}
}

func (info *GuildInfo) IsDebug(channel string) bool {
	debugchannel, isdebug := sb.DebugChannels[info.ID]
	if isdebug {
		return channel == debugchannel
	}
	return false
}

func (info *GuildInfo) ProcessMember(u *discordgo.Member) {
	ProcessUser(u.User, nil)

	t := time.Now().UTC()
	if len(u.JoinedAt) > 0 { // Parse join date and update user table only if it is less than our current first seen date.
		t, _ = time.Parse(time.RFC3339, u.JoinedAt)
	}
	if sb.db.CheckStatus() {
		sb.db.AddMember(SBatoi(u.User.ID), SBatoi(info.ID), t, u.Nick)
	}
}

func (info *GuildInfo) UserBulkUpdate(members []*discordgo.Member) {
	valueArgs := make([]interface{}, 0, len(members)*6)
	valueStrings := make([]string, 0, len(members))

	for _, m := range members {
		valueStrings = append(valueStrings, "(?,?,?,?,?,?,UTC_TIMESTAMP(), UTC_TIMESTAMP())")
		discriminator, _ := strconv.Atoi(m.User.Discriminator)
		valueArgs = append(valueArgs, SBatoi(m.User.ID), m.User.Email, m.User.Username, discriminator, m.User.Avatar, m.User.Verified)
	}

	stmt := fmt.Sprintf("INSERT IGNORE INTO users (ID, Email, Username, Discriminator, Avatar, Verified, LastSeen, LastNameChange) VALUES %s", strings.Join(valueStrings, ","))
	_, err := sb.db.db.Exec(stmt, valueArgs...)
	info.LogError("Error in UserBulkUpdate", err)
}

func (info *GuildInfo) MemberBulkUpdate(members []*discordgo.Member) {
	valueArgs := make([]interface{}, 0, len(members)*4)
	valueStrings := make([]string, 0, len(members))

	for _, m := range members {
		valueStrings = append(valueStrings, "(?,?,?,?,UTC_TIMESTAMP())")
		t := time.Now().UTC()
		if len(m.JoinedAt) > 0 { // Parse join date and update user table only if it is less than our current first seen date.
			t, _ = time.Parse(time.RFC3339, m.JoinedAt)
		}
		valueArgs = append(valueArgs, SBatoi(m.User.ID), SBatoi(info.ID), t, m.Nick)
	}
	stmt := fmt.Sprintf("INSERT IGNORE INTO members (ID, Guild, FirstSeen, Nickname, LastNickChange) VALUES %s", strings.Join(valueStrings, ","))
	_, err := sb.db.db.Exec(stmt, valueArgs...)
	info.LogError("Error in MemberBulkUpdate", err)
}

func (info *GuildInfo) ProcessGuild(g *discordgo.Guild) {
	info.Name = g.Name
	info.OwnerID = g.OwnerID
	const chunksize int = 1000

	if len(g.Members) > 0 && sb.db.CheckStatus() {
		// First process userdata
		i := chunksize
		for i < len(g.Members) {
			info.UserBulkUpdate(g.Members[i-chunksize : i])
			i += chunksize
		}
		info.UserBulkUpdate(g.Members[i-chunksize:])

		// Then process member data
		i = chunksize
		for i < len(g.Members) {
			info.MemberBulkUpdate(g.Members[i-chunksize : i])
			i += chunksize
		}
		info.MemberBulkUpdate(g.Members[i-chunksize:])
	}
}

func (info *GuildInfo) FindChannelID(name string) string {
	guild, err := sb.dg.State.Guild(info.ID)
	if err != nil {
		return ""
	}
	sb.dg.State.RLock()
	defer sb.dg.State.RUnlock()
	for _, v := range guild.Channels {
		if v.Name == name {
			return v.ID
		}
	}

	return ""
}

func (info *GuildInfo) HasChannel(id string) bool {
	c, err := sb.dg.State.Channel(id)
	if err != nil {
		return false
	}
	return c.GuildID == info.ID
}

func (info *GuildInfo) Log(args ...interface{}) {
	s := fmt.Sprint(args...)
	fmt.Printf("[%s] %s\n", time.Now().Format(time.Stamp), s)
	if sb.db != nil && info != nil && sb.IsMainGuild(info) && sb.db.status.get() {
		sb.db.Audit(AUDIT_TYPE_LOG, nil, s, SBatoi(info.ID))
	}
	if info != nil && info.config.Log.Channel > 0 {
		info.SendMessage(SBitoa(info.config.Log.Channel), "```\n"+s+"```")
	}
}

func (info *GuildInfo) LogError(msg string, err error) {
	if err != nil {
		info.Log(msg, err.Error())
	}
}

func (info *GuildInfo) Error(channelID string, message string) {
	if info != nil && RateLimit(&info.lastlogerr, info.config.Log.Cooldown) { // Don't print more than one error message every n seconds.
		info.SendMessage(channelID, "```\n"+message+"```")
	}
	//Log(message); // Always log it to the debug log. TODO: This is really annoying, maybe we shouldn't do this
}
