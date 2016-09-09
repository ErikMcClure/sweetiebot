package sweetiebot

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
)

type BotDB struct {
	db                       *sql.DB
	log                      Logger
	sql_AddMessage           *sql.Stmt
	sql_GetMessage           *sql.Stmt
	sql_GetLatestMessage     *sql.Stmt
	sql_AddPing              *sql.Stmt
	sql_GetPing              *sql.Stmt
	sql_GetPingContext       *sql.Stmt
	sql_GetPingContextBefore *sql.Stmt
	sql_AddUser              *sql.Stmt
	sql_AddMember            *sql.Stmt
	sql_GetUser              *sql.Stmt
	sql_FindUsers            *sql.Stmt
	sql_GetRecentMessages    *sql.Stmt
	sql_GetNewestUsers       *sql.Stmt
	sql_GetRecentUsers       *sql.Stmt
	sql_GetAliases           *sql.Stmt
	sql_AddTranscript        *sql.Stmt
	sql_GetTranscript        *sql.Stmt
	sql_RemoveTranscript     *sql.Stmt
	sql_AddMarkov            *sql.Stmt
	sql_GetMarkovLine        *sql.Stmt
	sql_GetMarkovLine2       *sql.Stmt
	sql_GetMarkovWord        *sql.Stmt
	sql_GetRandomQuoteInt    *sql.Stmt
	sql_GetRandomQuote       *sql.Stmt
	sql_GetSpeechQuoteInt    *sql.Stmt
	sql_GetSpeechQuote       *sql.Stmt
	sql_GetCharacterQuoteInt *sql.Stmt
	sql_GetCharacterQuote    *sql.Stmt
	sql_GetRandomSpeakerInt  *sql.Stmt
	sql_GetRandomSpeaker     *sql.Stmt
	sql_GetRandomMemberInt   *sql.Stmt
	sql_GetRandomMember      *sql.Stmt
	sql_GetRandomWordInt     *sql.Stmt
	sql_GetRandomWord        *sql.Stmt
	sql_GetTableCounts       *sql.Stmt
	sql_CountNewUsers        *sql.Stmt
	sql_Log                  *sql.Stmt
	sql_ResetMarkov          *sql.Stmt
	sql_AddSchedule          *sql.Stmt
	sql_AddScheduleRepeat    *sql.Stmt
	sql_GetSchedule          *sql.Stmt
	sql_RemoveSchedule       *sql.Stmt
	sql_CountEvents          *sql.Stmt
	sql_GetEvents            *sql.Stmt
	sql_GetEventsByType      *sql.Stmt
	sql_GetNextEvent         *sql.Stmt
	sql_GetReminders         *sql.Stmt
	sql_GetTimeZone          *sql.Stmt
	sql_SetTimeZone          *sql.Stmt
}

func DB_Load(log Logger, driver string, conn string) (*BotDB, error) {
	cdb, err := sql.Open(driver, conn)
	r := BotDB{}
	r.db = cdb
	r.log = log
	if err != nil {
		return &r, err
	}

	err = r.db.Ping()
	return &r, err
}

func (db *BotDB) Close() {
	if db.db != nil {
		db.db.Close()
	}
}

func (db *BotDB) Prepare(s string) (*sql.Stmt, error) {
	statement, err := db.db.Prepare(s)
	if err != nil {
		fmt.Println("Preparing: ", s, "\nSQL Error: ", err.Error())
	}
	return statement, err
}

func (db *BotDB) LoadStatements() error {
	var err error
	db.sql_AddMessage, err = db.Prepare("CALL AddChat(?,?,?,?,?,?)")
	db.sql_GetMessage, err = db.Prepare("SELECT Author, Message, Timestamp, Channel FROM chatlog WHERE ID = ?")
	db.sql_GetLatestMessage, err = db.Prepare("SELECT Timestamp FROM chatlog WHERE Channel = ? ORDER BY Timestamp DESC LIMIT 1")
	db.sql_AddPing, err = db.Prepare("INSERT INTO pings (Message, User) VALUES (?, ?) ON DUPLICATE KEY UPDATE Message = Message")
	db.sql_GetPing, err = db.Prepare("SELECT C.ID, C.Channel FROM pings P RIGHT OUTER JOIN chatlog C ON P.Message = C.ID WHERE P.User = ? OR (C.Everyone = 1 AND C.Channel != ?) ORDER BY Timestamp DESC LIMIT 1 OFFSET ?")
	db.sql_GetPingContext, err = db.Prepare("SELECT U.Username, C.Message, C.Timestamp FROM chatlog C INNER JOIN users U ON C.Author = U.ID WHERE C.ID >= ? AND C.Channel = ? ORDER BY C.ID ASC LIMIT ?")
	db.sql_GetPingContextBefore, err = db.Prepare("SELECT U.Username, C.Message, C.Timestamp FROM chatlog C INNER JOIN users U ON C.Author = U.ID WHERE C.ID < ? AND C.Channel = ? ORDER BY C.ID DESC LIMIT ?")
	db.sql_AddUser, err = db.Prepare("CALL AddUser(?,?,?,?,?)")
	db.sql_AddMember, err = db.Prepare("CALL AddMember(?,?,?,?)")
	db.sql_GetUser, err = db.Prepare("SELECT ID, Email, Username, Avatar, LastSeen FROM users WHERE ID = ?")
	db.sql_FindUsers, err = db.Prepare("SELECT U.ID FROM users U LEFT OUTER JOIN aliases A ON A.User = U.ID LEFT OUTER JOIN members M ON M.ID = U.ID WHERE U.Username LIKE ? OR M.Nickname LIKE ? OR A.Alias = ? GROUP BY U.ID LIMIT ? OFFSET ?")
	db.sql_GetRecentMessages, err = db.Prepare("SELECT ID, Channel FROM chatlog WHERE Author = ? AND Timestamp >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL ? SECOND)")
	db.sql_GetNewestUsers, err = db.Prepare("SELECT U.ID, U.Email, U.Username, U.Avatar, M.FirstSeen FROM members M INNER JOIN users U ON M.ID = U.ID WHERE M.Guild = ? ORDER BY M.FirstSeen DESC LIMIT ?")
	db.sql_GetRecentUsers, err = db.Prepare("SELECT U.ID, U.Email, U.Username, U.Avatar FROM members M INNER JOIN users U ON M.ID = U.ID WHERE M.Guild = ? AND M.FirstSeen > ? ORDER BY M.FirstSeen DESC")
	db.sql_GetAliases, err = db.Prepare("SELECT Alias FROM aliases WHERE User = ? ORDER BY Duration DESC LIMIT 10")
	db.sql_AddTranscript, err = db.Prepare("INSERT INTO transcripts (Season, Episode, Line, Speaker, Text) VALUES (?,?,?,?,?)")
	db.sql_GetTranscript, err = db.Prepare("SELECT Season, Episode, Line, Speaker, Text FROM transcripts WHERE Season = ? AND Episode = ? AND Line >= ? AND LINE <= ?")
	db.sql_RemoveTranscript, err = db.Prepare("DELETE FROM transcripts WHERE Season = ? AND Episode = ? AND Line = ?")
	db.sql_AddMarkov, err = db.Prepare("SELECT AddMarkov(?,?,?,?)")
	db.sql_GetMarkovLine, err = db.Prepare("SELECT GetMarkovLine(?)")
	db.sql_GetMarkovLine2, err = db.Prepare("SELECT GetMarkovLine2(?,?)")
	db.sql_GetMarkovWord, err = db.Prepare("SELECT Phrase FROM markov_transcripts WHERE SpeakerID = (SELECT ID FROM markov_transcripts_speaker WHERE Speaker = ?) AND Phrase = ?")
	db.sql_GetRandomQuoteInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM transcripts WHERE Text != ''))")
	db.sql_GetRandomQuote, err = db.Prepare("SELECT * FROM transcripts WHERE Text != '' LIMIT 1 OFFSET ?")
	db.sql_GetSpeechQuoteInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM transcripts WHERE Speaker != 'ACTION' AND Text != ''))")
	db.sql_GetSpeechQuote, err = db.Prepare("SELECT * FROM transcripts WHERE Speaker != 'ACTION' AND Text != '' LIMIT 1 OFFSET ?")
	db.sql_GetCharacterQuoteInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM transcripts WHERE Speaker = ? AND Text != ''))")
	db.sql_GetCharacterQuote, err = db.Prepare("SELECT * FROM transcripts WHERE Speaker = ? AND Text != '' LIMIT 1 OFFSET ?")
	db.sql_GetRandomSpeakerInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM markov_transcripts_speaker))")
	db.sql_GetRandomSpeaker, err = db.Prepare("SELECT Speaker FROM markov_transcripts_speaker LIMIT 1 OFFSET ?")
	db.sql_GetRandomMemberInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM members WHERE Guild = ?))")
	db.sql_GetRandomMember, err = db.Prepare("SELECT U.Username FROM members M INNER JOIN users U ON M.ID = U.ID WHERE M.Guild = ? LIMIT 1 OFFSET ?")
	db.sql_GetRandomWordInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM randomwords))")
	db.sql_GetRandomWord, err = db.Prepare("SELECT Phrase FROM randomwords LIMIT 1 OFFSET ?;")
	db.sql_GetTableCounts, err = db.Prepare("SELECT CONCAT('Chatlog: ', (SELECT COUNT(*) FROM chatlog), ' rows', '\nEditlog: ', (SELECT COUNT(*) FROM editlog), ' rows',  '\nAliases: ', (SELECT COUNT(*) FROM aliases), ' rows',  '\nDebuglog: ', (SELECT COUNT(*) FROM debuglog), ' rows',  '\nPings: ', (SELECT COUNT(*) FROM pings), ' rows',  '\nUsers: ', (SELECT COUNT(*) FROM users), ' rows',  '\nSchedule: ', (SELECT COUNT(*) FROM schedule), ' rows \nMembers: ', (SELECT COUNT(*) FROM members), ' rows');")
	db.sql_CountNewUsers, err = db.Prepare("SELECT COUNT(*) FROM members WHERE FirstSeen > DATE_SUB(UTC_TIMESTAMP(), INTERVAL ? SECOND) AND Guild = ?")
	db.sql_Log, err = db.Prepare("INSERT INTO debuglog (Message, Timestamp) VALUE(?, UTC_TIMESTAMP())")
	db.sql_ResetMarkov, err = db.Prepare("CALL ResetMarkov()")
	db.sql_AddSchedule, err = db.Prepare("INSERT INTO schedule (Guild, Date, Type, Data) VALUES (?, ?, ?, ?)")
	db.sql_AddScheduleRepeat, err = db.Prepare("INSERT INTO schedule (Guild, Date, `RepeatInterval`, `Repeat`, Type, Data) VALUES (?, ?, ?, ?, ?, ?)")
	db.sql_GetSchedule, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Date <= UTC_TIMESTAMP() ORDER BY Date ASC")
	db.sql_RemoveSchedule, err = db.Prepare("CALL RemoveSchedule(?)")
	db.sql_CountEvents, err = db.Prepare("SELECT COUNT(*) FROM schedule WHERE Guild = ?")
	db.sql_GetEvents, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type != 0 AND Type != 4 AND Type != 6 ORDER BY Date ASC LIMIT ?")
	db.sql_GetEventsByType, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = ? ORDER BY Date ASC LIMIT ?")
	db.sql_GetNextEvent, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = ? ORDER BY Date ASC LIMIT 1")
	db.sql_GetReminders, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = 6 AND Data LIKE ? ORDER BY Date ASC LIMIT ?")
	db.sql_GetTimeZone, err = db.Prepare("SELECT Timezone FROM users WHERE ID = ?")
	db.sql_SetTimeZone, err = db.Prepare("UPDATE users SET Timezone = ? WHERE ID = ?")
	return err
}

func (db *BotDB) ParseStringResults(q *sql.Rows) []string {
	r := make([]string, 0, 3)
	for q.Next() {
		p := ""
		err := q.Scan(&p)
		if err == nil {
			r = append(r, p)
		}
		db.log.LogError("Row scan error: ", err)
	}
	return r
}

func (db *BotDB) AddMessage(id uint64, author uint64, message string, channel uint64, everyone bool, guild uint64) {
	_, err := db.sql_AddMessage.Exec(id, author, message, channel, everyone, guild)
	db.log.LogError("AddMessage error: ", err)
}

func (db *BotDB) GetMessage(id uint64) (uint64, string, time.Time, uint64) {
	var author uint64
	var message string
	var timestamp time.Time
	var channel uint64
	err := db.sql_GetMessage.QueryRow(id).Scan(&author, &message, &timestamp, &channel)
	if err == sql.ErrNoRows {
		return 0, "", time.Now().UTC(), 0
	}
	db.log.LogError("GetMessage error: ", err)
	return author, message, timestamp, channel
}

func (db *BotDB) GetLatestMessage(channel uint64) time.Time {
	var timestamp time.Time
	err := db.sql_GetLatestMessage.QueryRow(channel).Scan(&timestamp)
	if err == sql.ErrNoRows {
		return time.Now().UTC()
	}
	db.log.LogError("GetLatestMessage error: ", err)
	return timestamp
}

func (db *BotDB) AddPing(message uint64, user uint64) {
	_, err := db.sql_AddPing.Exec(message, user)
	db.log.LogError("AddPing error: ", err)
}

func (db *BotDB) GetPing(user uint64, offset int, modchannel uint64) (uint64, uint64) {
	var id uint64
	var channel uint64
	err := db.sql_GetPing.QueryRow(user, modchannel, offset).Scan(&id, &channel)
	if err == sql.ErrNoRows {
		return 0, 0
	}
	db.log.LogError("GetPing error: ", err)
	return id, channel
}

type PingContext struct {
	Author    string
	Message   string
	Timestamp time.Time
}

func (db *BotDB) GetPingContext(message uint64, channel uint64, maxresults int) []PingContext {
	q, err := db.sql_GetPingContext.Query(message, channel, maxresults)
	db.log.LogError("GetPingContext error: ", err)
	defer q.Close()
	r := make([]PingContext, 0, maxresults)
	for q.Next() {
		p := PingContext{}
		if err := q.Scan(&p.Author, &p.Message, &p.Timestamp); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) GetPingContextBefore(message uint64, channel uint64, maxresults int) []PingContext {
	q, err := db.sql_GetPingContextBefore.Query(message, channel, maxresults)
	db.log.LogError("GetPingContextBefore error: ", err)
	defer q.Close()
	r := make([]PingContext, 0, maxresults)
	for q.Next() {
		p := PingContext{}
		if err := q.Scan(&p.Author, &p.Message, &p.Timestamp); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) AddUser(id uint64, email string, username string, avatar string, verified bool) {
	_, err := db.sql_AddUser.Exec(id, email, username, avatar, verified)
	db.log.LogError("AddUser error: ", err)
}

func (db *BotDB) AddMember(id uint64, guild uint64, firstseen time.Time, nickname string) {
	_, err := db.sql_AddMember.Exec(id, guild, firstseen, nickname)
	db.log.LogError("AddMember error: ", err)
}

func (db *BotDB) GetUser(id uint64) (*discordgo.User, time.Time) {
	u := &discordgo.User{}
	var lastseen time.Time
	err := db.sql_GetUser.QueryRow(id).Scan(&u.ID, &u.Email, &u.Username, &u.Avatar, &lastseen)
	db.log.LogError("GetUser error: ", err)
	return u, lastseen
}

func (db *BotDB) FindUsers(name string, maxresults uint64, offset uint64) []uint64 {
	q, err := db.sql_FindUsers.Query(name, name, name, maxresults, offset)
	db.log.LogError("FindUsers error: ", err)
	defer q.Close()
	r := make([]uint64, 0, 4)
	for q.Next() {
		var p uint64
		if err := q.Scan(&p); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) GetRecentMessages(user uint64, duration uint64) []struct {
	message uint64
	channel uint64
} {
	q, err := db.sql_GetRecentMessages.Query(user, duration)
	db.log.LogError("GetRecentMessages error: ", err)
	defer q.Close()
	r := make([]struct {
		message uint64
		channel uint64
	}, 0, 4)
	for q.Next() {
		p := struct {
			message uint64
			channel uint64
		}{}
		if err := q.Scan(&p.message, &p.channel); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) GetNewestUsers(maxresults int, guild uint64) []struct {
	User      *discordgo.User
	FirstSeen time.Time
} {
	q, err := db.sql_GetNewestUsers.Query(guild, maxresults)
	db.log.LogError("GetNewestUsers error: ", err)
	defer q.Close()
	r := make([]struct {
		User      *discordgo.User
		FirstSeen time.Time
	}, 0, maxresults)
	for q.Next() {
		p := struct {
			User      *discordgo.User
			FirstSeen time.Time
		}{&discordgo.User{}, time.Now()}
		if err := q.Scan(&p.User.ID, &p.User.Email, &p.User.Username, &p.User.Avatar, &p.FirstSeen); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) GetRecentUsers(since time.Time, guild uint64) []*discordgo.User {
	q, err := db.sql_GetRecentUsers.Query(guild, since)
	db.log.LogError("GetRecentUsers error: ", err)
	defer q.Close()
	r := make([]*discordgo.User, 0, 2)
	for q.Next() {
		p := &discordgo.User{}
		if err := q.Scan(&p.ID, &p.Email, &p.Username, &p.Avatar); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) GetAliases(user uint64) []string {
	q, err := db.sql_GetAliases.Query(user)
	db.log.LogError("GetAliases error: ", err)
	defer q.Close()
	return db.ParseStringResults(q)
}

func (db *BotDB) Log(message string) {
	_, err := db.sql_Log.Exec(message)
	if err != nil {
		fmt.Println("Logger failed to log to database! ", err.Error())
	}
}

func (db *BotDB) GetTableCounts() string {
	var counts string
	err := db.sql_GetTableCounts.QueryRow().Scan(&counts)
	db.log.LogError("GetTableCounts error: ", err)
	return counts
}

func (db *BotDB) AddTranscript(season int, episode int, line int, speaker string, text string) {
	_, err := db.sql_AddTranscript.Exec(season, episode, line, speaker, text)
	if err != nil {
		db.log.Log("AddTranscript error: ", err.Error, "\nS", season, "E", episode, ":", line, " ", speaker, ": ", text)
	}
}

type Transcript struct {
	Season  uint
	Episode uint
	Line    uint
	Speaker string
	Text    string
}

func (db *BotDB) GetTranscript(season int, episode int, start int, end int) []Transcript {
	q, err := db.sql_GetTranscript.Query(season, episode, start, end)
	db.log.LogError("GetTranscript error: ", err)
	defer q.Close()
	l := end - start + 1
	if l > 100 {
		l = 100
	}
	r := make([]Transcript, 0, l)
	for q.Next() {
		p := Transcript{}
		if err := q.Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) RemoveTranscript(season int, episode int, line int) {
	_, err := db.sql_RemoveTranscript.Exec(season, episode, line)
	db.log.LogError("RemoveTranscript error: ", err)
}
func (db *BotDB) AddMarkov(last uint64, last2 uint64, speaker string, text string) uint64 {
	var id uint64
	err := db.sql_AddMarkov.QueryRow(last, last2, speaker, text).Scan(&id)
	db.log.LogError("AddMarkov error: ", err)
	return id
}

func (db *BotDB) GetMarkovLine(last uint64) (string, uint64) {
	var r sql.NullString
	err := db.sql_GetMarkovLine.QueryRow(last).Scan(&r)
	db.log.LogError("GetMarkovLine error: ", err)
	if !r.Valid {
		return "", 0
	}
	str := strings.SplitN(r.String, "|", 2) // Being unable to call stored procedures makes this unnecessarily complex
	if len(str) < 2 || len(str[1]) < 1 {
		return str[0], 0
	}
	return str[0], SBatoi(str[1])
}

func (db *BotDB) GetMarkovLine2(last uint64, last2 uint64) (string, uint64, uint64) {
	var r sql.NullString
	err := db.sql_GetMarkovLine2.QueryRow(last, last2).Scan(&r)
	db.log.LogError("GetMarkovLine error: ", err)
	if !r.Valid {
		return "", 0, 0
	}
	str := strings.SplitN(r.String, "|", 3) // Being unable to call stored procedures makes this unnecessarily complex
	if len(str) < 3 || len(str[1]) < 1 || len(str[2]) < 1 {
		return str[0], 0, 0
	}
	return str[0], SBatoi(str[1]), SBatoi(str[2])
}
func (db *BotDB) GetMarkovWord(speaker string, phrase string) string {
	var r string
	err := db.sql_GetMarkovWord.QueryRow(speaker, phrase).Scan(&r)
	if err == sql.ErrNoRows {
		return phrase
	}
	db.log.LogError("GetMarkovWord error: ", err)
	return r
}
func (db *BotDB) GetRandomQuote() Transcript {
	var i uint64
	err := db.sql_GetRandomQuoteInt.QueryRow().Scan(&i)
	db.log.LogError("GetRandomQuoteInt error: ", err)
	var p Transcript
	err = db.sql_GetRandomQuote.QueryRow(i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
	db.log.LogError("GetRandomQuote error: ", err)
	return p
}
func (db *BotDB) GetSpeechQuote() Transcript {
	var i uint64
	err := db.sql_GetSpeechQuoteInt.QueryRow().Scan(&i)
	db.log.LogError("GetSpeechQuoteInt error: ", err)
	var p Transcript
	err = db.sql_GetSpeechQuote.QueryRow(i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
	db.log.LogError("GetSpeechQuote error: ", err)
	return p
}
func (db *BotDB) GetCharacterQuote(character string) Transcript {
	var i uint64
	err := db.sql_GetCharacterQuoteInt.QueryRow(character).Scan(&i)
	db.log.LogError("GetCharacterQuoteInt error: ", err)
	var p Transcript
	err = db.sql_GetCharacterQuote.QueryRow(character, i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
	if err == sql.ErrNoRows {
		return Transcript{0, 0, 0, "", ""}
	}
	db.log.LogError("GetCharacterQuote error: ", err)
	return p
}
func (db *BotDB) GetRandomSpeaker() string {
	var i uint64
	err := db.sql_GetRandomSpeakerInt.QueryRow().Scan(&i)
	db.log.LogError("GetRandomSpeakerInt error: ", err)
	var p string
	err = db.sql_GetRandomSpeaker.QueryRow(i).Scan(&p)
	db.log.LogError("GetRandomSpeaker error: ", err)
	return p
}
func (db *BotDB) GetRandomMember(guild uint64) string {
	var i uint64
	err := db.sql_GetRandomMemberInt.QueryRow(guild).Scan(&i)
	db.log.LogError("GetRandomMemberInt error: ", err)
	var p string
	err = db.sql_GetRandomMember.QueryRow(guild, i).Scan(&p)
	db.log.LogError("GetRandomMember error: ", err)
	return p
}
func (db *BotDB) GetRandomWord() string {
	var i uint64
	err := db.sql_GetRandomWordInt.QueryRow().Scan(&i)
	db.log.LogError("GetRandomWordInt error: ", err)
	var p string
	err = db.sql_GetRandomWord.QueryRow(i).Scan(&p)
	db.log.LogError("GetRandomWord error: ", err)
	return p
}
func (db *BotDB) CountNewUsers(seconds int64, guild uint64) int {
	var i int
	err := db.sql_CountNewUsers.QueryRow(seconds, guild).Scan(&i)
	db.log.LogError("CountNewUsers error: ", err)
	return i
}

func (db *BotDB) RemoveSchedule(id uint64) {
	_, err := db.sql_RemoveSchedule.Exec(id)
	db.log.LogError("RemoveSchedule error: ", err)
}
func (db *BotDB) AddSchedule(guild uint64, date time.Time, ty uint8, data string) bool {
	var i int
	err := db.sql_CountEvents.QueryRow(guild).Scan(&i)
	db.log.LogError("sql_CountEvents error: ", err)
	if err == nil && i < 5000 {
		_, err = db.sql_AddSchedule.Exec(guild, date, ty, data)
		db.log.LogError("AddSchedule error: ", err)
		return err == nil
	}
	return false
}
func (db *BotDB) AddScheduleRepeat(guild uint64, date time.Time, repeatinterval uint8, repeat int, ty uint8, data string) bool {
	var i int
	err := db.sql_CountEvents.QueryRow(guild).Scan(&i)
	db.log.LogError("sql_CountEvents error: ", err)
	if err == nil && i < 5000 {
		_, err := db.sql_AddScheduleRepeat.Exec(guild, date, repeatinterval, repeat, ty, data)
		db.log.LogError("AddScheduleRepeat error: ", err)
		return err == nil
	}
	return false
}

type ScheduleEvent struct {
	ID   uint64
	Date time.Time
	Type uint8
	Data string
}

func (db *BotDB) GetSchedule(guild uint64) []ScheduleEvent {
	q, err := db.sql_GetSchedule.Query(guild)
	db.log.LogError("GetSchedule error: ", err)
	defer q.Close()
	r := make([]ScheduleEvent, 0, 2)
	for q.Next() {
		p := ScheduleEvent{}
		if err := q.Scan(&p.ID, &p.Date, &p.Type, &p.Data); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) GetEvents(guild uint64, maxnum int) []ScheduleEvent {
	q, err := db.sql_GetEvents.Query(guild, maxnum)
	db.log.LogError("GetEvents error: ", err)
	defer q.Close()
	r := make([]ScheduleEvent, 0, 2)
	for q.Next() {
		p := ScheduleEvent{}
		if err := q.Scan(&p.ID, &p.Date, &p.Type, &p.Data); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) GetEventsByType(guild uint64, ty uint8, maxnum int) []ScheduleEvent {
	q, err := db.sql_GetEventsByType.Query(guild, ty, maxnum)
	db.log.LogError("GetEventsByType error: ", err)
	defer q.Close()
	r := make([]ScheduleEvent, 0, 2)
	for q.Next() {
		p := ScheduleEvent{}
		if err := q.Scan(&p.ID, &p.Date, &p.Type, &p.Data); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) GetNextEvent(guild uint64, ty uint8) ScheduleEvent {
	p := ScheduleEvent{}
	err := db.sql_GetNextEvent.QueryRow(guild, ty).Scan(&p.ID, &p.Date, &p.Type, &p.Data)
	if err == sql.ErrNoRows {
		return ScheduleEvent{0, time.Now().UTC(), 0, ""}
	}
	db.log.LogError("GetNextEvent error: ", err)
	return p
}

func (db *BotDB) GetReminders(guild uint64, id string, maxnum int) []ScheduleEvent {
	q, err := db.sql_GetReminders.Query(guild, id+"|%", maxnum)
	db.log.LogError("GetReminders error: ", err)
	defer q.Close()
	r := make([]ScheduleEvent, 0, 2)
	for q.Next() {
		p := ScheduleEvent{}
		if err := q.Scan(&p.ID, &p.Date, &p.Type, &p.Data); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) GetTimeZone(user uint64) sql.NullInt64 {
	var i sql.NullInt64
	err := db.sql_GetTimeZone.QueryRow(user).Scan(&i)
	db.log.LogError("GetTimeZone error: ", err)
	return i
}

func (db *BotDB) SetTimeZone(user uint64, tz int64) error {
	_, err := db.sql_SetTimeZone.Exec(tz, user)
	db.log.LogError("SetTimeZone error: ", err)
	return err
}
