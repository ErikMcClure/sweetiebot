package sweetiebot

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
)

type BotDB struct {
	db                         *sql.DB
	status                     AtomicBool
	lastattempt                time.Time
	log                        Logger
	driver                     string
	conn                       string
	statuslock                 sync.RWMutex
	sql_AddMessage             *sql.Stmt
	sql_GetMessage             *sql.Stmt
	sql_AddUser                *sql.Stmt
	sql_AddMember              *sql.Stmt
	sql_GetUser                *sql.Stmt
	sql_GetMember              *sql.Stmt
	sql_FindGuildUsers         *sql.Stmt
	sql_FindUsers              *sql.Stmt
	sql_GetRecentMessages      *sql.Stmt
	sql_GetNewestUsers         *sql.Stmt
	sql_GetRecentUsers         *sql.Stmt
	sql_GetAliases             *sql.Stmt
	sql_AddTranscript          *sql.Stmt
	sql_GetTranscript          *sql.Stmt
	sql_RemoveTranscript       *sql.Stmt
	sql_AddMarkov              *sql.Stmt
	sql_GetMarkovLine          *sql.Stmt
	sql_GetMarkovLine2         *sql.Stmt
	sql_GetMarkovWord          *sql.Stmt
	sql_GetRandomQuoteInt      *sql.Stmt
	sql_GetRandomQuote         *sql.Stmt
	sql_GetSpeechQuoteInt      *sql.Stmt
	sql_GetSpeechQuote         *sql.Stmt
	sql_GetCharacterQuoteInt   *sql.Stmt
	sql_GetCharacterQuote      *sql.Stmt
	sql_GetRandomSpeakerInt    *sql.Stmt
	sql_GetRandomSpeaker       *sql.Stmt
	sql_GetRandomMemberInt     *sql.Stmt
	sql_GetRandomMember        *sql.Stmt
	sql_GetRandomWordInt       *sql.Stmt
	sql_GetRandomWord          *sql.Stmt
	sql_GetTableCounts         *sql.Stmt
	sql_CountNewUsers          *sql.Stmt
	sql_Audit                  *sql.Stmt
	sql_GetAuditRows           *sql.Stmt
	sql_GetAuditRowsUser       *sql.Stmt
	sql_GetAuditRowsString     *sql.Stmt
	sql_GetAuditRowsUserString *sql.Stmt
	sql_ResetMarkov            *sql.Stmt
	sql_AddSchedule            *sql.Stmt
	sql_AddScheduleRepeat      *sql.Stmt
	sql_GetSchedule            *sql.Stmt
	sql_RemoveSchedule         *sql.Stmt
	sql_CountEvents            *sql.Stmt
	sql_GetEvent               *sql.Stmt
	sql_GetEvents              *sql.Stmt
	sql_GetEventsByType        *sql.Stmt
	sql_GetNextEvent           *sql.Stmt
	sql_GetReminders           *sql.Stmt
	sql_GetUnsilenceDate       *sql.Stmt
	sql_GetTimeZone            *sql.Stmt
	sql_FindTimeZone           *sql.Stmt
	sql_FindTimeZoneOffset     *sql.Stmt
	sql_SetTimeZone            *sql.Stmt
	sql_RemoveAlias            *sql.Stmt
	sql_GetUserGuilds          *sql.Stmt
	sql_FindEvent              *sql.Stmt
	sql_SetDefaultServer       *sql.Stmt
	sql_GetPolls               *sql.Stmt
	sql_GetPoll                *sql.Stmt
	sql_GetOptions             *sql.Stmt
	sql_GetOption              *sql.Stmt
	sql_GetResults             *sql.Stmt
	sql_AddPoll                *sql.Stmt
	sql_AddOption              *sql.Stmt
	sql_AppendOption           *sql.Stmt
	sql_AddVote                *sql.Stmt
	sql_RemovePoll             *sql.Stmt
	sql_CheckOption            *sql.Stmt
}

func DB_Load(log Logger, driver string, conn string) (*BotDB, error) {
	cdb, err := sql.Open(driver, conn)
	r := BotDB{}
	r.db = cdb
	r.status.set(err == nil)
	r.lastattempt = time.Now().UTC()
	r.log = log
	r.driver = driver
	r.conn = conn
	if err != nil {
		return &r, err
	}

	err = r.db.Ping()
	r.status.set(err == nil)
	return &r, err
}

func (db *BotDB) Close() {
	if db.db != nil {
		db.db.Close()
		db.db = nil
	}
}

func (db *BotDB) Prepare(s string) (*sql.Stmt, error) {
	statement, err := db.db.Prepare(s)
	if err != nil {
		fmt.Println("Preparing: ", s, "\nSQL Error: ", err.Error())
	}
	return statement, err
}

const DB_RECONNECT_TIMEOUT = time.Duration(120) * time.Second // Reconnect time interval in seconds

func (db *BotDB) CheckStatus() bool {
	if !db.status.get() {
		db.statuslock.Lock()
		defer db.statuslock.Unlock()

		if db.status.get() { // If the database was already fixed, return true
			return true
		}

		if db.lastattempt.Add(DB_RECONNECT_TIMEOUT).Before(time.Now().UTC()) {
			db.log.Log("Database failure detected! Attempting to reboot database connection...")
			db.lastattempt = time.Now().UTC()
			err := db.db.Ping()
			if err != nil {
				db.log.LogError("Reconnection failed! Another attempt will be made in "+TimeDiff(DB_RECONNECT_TIMEOUT)+". Error: ", err)
				return false
			}
			err = db.LoadStatements()                       // If we re-establish connection, we must reload statements in case they were lost or never loaded in the first place
			db.log.LogError("LoadStatements failed: ", err) // if loading the statements fails we're screwed anyway so we just log the error and keep going
			db.status.set(true)                             // Only after loading the statements do we set status to true
			db.log.Log("Reconnection succeeded, exiting out of No Database mode.")
		} else { // If not, just fail
			return false
		}
	}

	return true
}

func (db *BotDB) LoadStatements() error {
	var err error
	db.sql_AddMessage, err = db.Prepare("CALL AddChat(?,?,?,?,?,?)")
	db.sql_GetMessage, err = db.Prepare("SELECT Author, Message, Timestamp, Channel FROM chatlog WHERE ID = ?")
	db.sql_AddUser, err = db.Prepare("CALL AddUser(?,?,?,?,?,?,?)")
	db.sql_AddMember, err = db.Prepare("CALL AddMember(?,?,?,?)")
	db.sql_GetUser, err = db.Prepare("SELECT ID, Email, Username, Discriminator, Avatar, LastSeen, Timezone, Location, DefaultServer FROM users WHERE ID = ?")
	db.sql_GetMember, err = db.Prepare("SELECT U.ID, U.Email, U.Username, U.Discriminator, U.Avatar, U.LastSeen, M.Nickname, M.FirstSeen FROM members M RIGHT OUTER JOIN users U ON U.ID = M.ID WHERE M.ID = ? AND M.Guild = ?")
	db.sql_FindGuildUsers, err = db.Prepare("SELECT U.ID FROM users U LEFT OUTER JOIN aliases A ON A.User = U.ID LEFT OUTER JOIN members M ON M.ID = U.ID WHERE M.Guild = ? AND (U.Username LIKE ? OR M.Nickname LIKE ? OR A.Alias = ?) GROUP BY U.ID LIMIT ? OFFSET ?")
	db.sql_FindUsers, err = db.Prepare("SELECT U.ID FROM users U LEFT OUTER JOIN aliases A ON A.User = U.ID LEFT OUTER JOIN members M ON M.ID = U.ID WHERE U.Username LIKE ? OR M.Nickname LIKE ? OR A.Alias = ? GROUP BY U.ID LIMIT ? OFFSET ?")
	db.sql_GetRecentMessages, err = db.Prepare("SELECT ID, Channel FROM chatlog WHERE Guild = ? AND Author = ? AND Timestamp >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL ? SECOND)")
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
	db.sql_GetTableCounts, err = db.Prepare("SELECT CONCAT('Chatlog: ', (SELECT COUNT(*) FROM chatlog), ' rows', '\nEditlog: ', (SELECT COUNT(*) FROM editlog), ' rows',  '\nAliases: ', (SELECT COUNT(*) FROM aliases), ' rows',  '\nDebuglog: ', (SELECT COUNT(*) FROM debuglog), ' rows',  '\nUsers: ', (SELECT COUNT(*) FROM users), ' rows',  '\nSchedule: ', (SELECT COUNT(*) FROM schedule), ' rows \nMembers: ', (SELECT COUNT(*) FROM members), ' rows');")
	db.sql_CountNewUsers, err = db.Prepare("SELECT COUNT(*) FROM members WHERE FirstSeen > DATE_SUB(UTC_TIMESTAMP(), INTERVAL ? SECOND) AND Guild = ?")
	db.sql_Audit, err = db.Prepare("INSERT INTO debuglog (Type, User, Message, Timestamp, Guild) VALUE(?, ?, ?, UTC_TIMESTAMP(), ?)")
	db.sql_GetAuditRows, err = db.Prepare("SELECT U.Username, D.Message, D.Timestamp, U.ID FROM debuglog D INNER JOIN users U ON D.User = U.ID WHERE D.Type = ? AND D.Guild = ? ORDER BY D.Timestamp DESC LIMIT ? OFFSET ?")
	db.sql_GetAuditRowsUser, err = db.Prepare("SELECT U.Username, D.Message, D.Timestamp, U.ID FROM debuglog D INNER JOIN users U ON D.User = U.ID WHERE D.Type = ? AND D.Guild = ? AND D.User = ? ORDER BY D.Timestamp DESC LIMIT ? OFFSET ?")
	db.sql_GetAuditRowsString, err = db.Prepare("SELECT U.Username, D.Message, D.Timestamp, U.ID FROM debuglog D INNER JOIN users U ON D.User = U.ID WHERE D.Type = ? AND D.Guild = ? AND D.Message LIKE ? ORDER BY D.Timestamp DESC LIMIT ? OFFSET ?")
	db.sql_GetAuditRowsUserString, err = db.Prepare("SELECT U.Username, D.Message, D.Timestamp, U.ID FROM debuglog D INNER JOIN users U ON D.User = U.ID WHERE D.Type = ? AND D.Guild = ? AND D.User = ? AND D.Message LIKE ? ORDER BY D.Timestamp DESC LIMIT ? OFFSET ?")
	db.sql_ResetMarkov, err = db.Prepare("CALL ResetMarkov()")
	db.sql_AddSchedule, err = db.Prepare("INSERT INTO schedule (Guild, Date, Type, Data) VALUES (?, ?, ?, ?)")
	db.sql_AddScheduleRepeat, err = db.Prepare("INSERT INTO schedule (Guild, Date, `RepeatInterval`, `Repeat`, Type, Data) VALUES (?, ?, ?, ?, ?, ?)")
	db.sql_GetSchedule, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Date <= UTC_TIMESTAMP() ORDER BY Date ASC")
	db.sql_RemoveSchedule, err = db.Prepare("CALL RemoveSchedule(?)")
	db.sql_CountEvents, err = db.Prepare("SELECT COUNT(*) FROM schedule WHERE Guild = ?")
	db.sql_GetEvent, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE ID = ?")
	db.sql_GetEvents, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type != 0 AND Type != 4 AND Type != 6 ORDER BY Date ASC LIMIT ?")
	db.sql_GetEventsByType, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = ? ORDER BY Date ASC LIMIT ?")
	db.sql_GetNextEvent, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = ? ORDER BY Date ASC LIMIT 1")
	db.sql_GetReminders, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = 6 AND Data LIKE ? ORDER BY Date ASC LIMIT ?")
	db.sql_GetUnsilenceDate, err = db.Prepare("SELECT Date FROM schedule WHERE Guild = ? AND Type = 8 AND Data = ?")
	db.sql_GetTimeZone, err = db.Prepare("SELECT Timezone, Location FROM users WHERE ID = ?")
	db.sql_FindTimeZone, err = db.Prepare("SELECT Location FROM timezones WHERE Location LIKE ?")
	db.sql_FindTimeZoneOffset, err = db.Prepare("SELECT Location FROM timezones WHERE Location LIKE ? AND (Offset = ? OR DST = ?)")
	db.sql_SetTimeZone, err = db.Prepare("UPDATE users SET Location = ? WHERE ID = ?")
	db.sql_RemoveAlias, err = db.Prepare("DELETE FROM aliases WHERE User = ? AND Alias = ?")
	db.sql_GetUserGuilds, err = db.Prepare("SELECT Guild FROM members WHERE ID = ?")
	db.sql_FindEvent, err = db.Prepare("SELECT ID FROM `schedule` WHERE `Type` = ? AND `Data` = ? AND `Guild` = ?")
	db.sql_SetDefaultServer, err = db.Prepare("UPDATE users SET DefaultServer = ? WHERE ID = ?")
	db.sql_GetPolls, err = db.Prepare("SELECT Name, Description FROM polls WHERE Guild = ? ORDER BY ID DESC")
	db.sql_GetPoll, err = db.Prepare("SELECT ID, Description FROM polls WHERE Name = ? AND Guild = ?")
	db.sql_GetOptions, err = db.Prepare("SELECT `Index`, `Option` FROM polloptions WHERE Poll = ? ORDER BY `Index` ASC")
	db.sql_GetOption, err = db.Prepare("SELECT `Index` FROM polloptions WHERE poll = ? AND `Option` = ?")
	db.sql_GetResults, err = db.Prepare("SELECT `Option`,COUNT(user) FROM `votes` WHERE `Poll` = ? GROUP BY `Option` ORDER BY `Option` ASC")
	db.sql_AddPoll, err = db.Prepare("INSERT INTO polls(Name, Description, Guild) VALUES (?, ?, ?)")
	db.sql_AddOption, err = db.Prepare("INSERT INTO polloptions(Poll, `Index`, `Option`) VALUES (?, ?, ?)")
	db.sql_AppendOption, err = db.Prepare("INSERT INTO polloptions(Poll, `Index`, `Option`) SELECT Poll, MAX(`index`)+1, ? FROM polloptions WHERE poll = ?")
	db.sql_AddVote, err = db.Prepare("INSERT INTO votes (Poll, User, `Option`) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE `Option` = ?")
	db.sql_RemovePoll, err = db.Prepare("DELETE FROM polls WHERE Name = ? AND Guild = ?")
	db.sql_CheckOption, err = db.Prepare("SELECT `Option` FROM polloptions WHERE poll = ? AND `Index` = ?")
	return err
}

const (
	AUDIT_TYPE_LOG     = iota
	AUDIT_TYPE_ACTION  = iota
	AUDIT_TYPE_COMMAND = iota
)

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
func (db *BotDB) CheckError(name string, err error) bool {
	if err != nil {
		db.log.LogError(name+" error: ", err)
		if err != sql.ErrNoRows && err != sql.ErrTxDone {
			db.status.set(false)
			return true
		}
	}
	return false
}

func (db *BotDB) AddMessage(id uint64, author uint64, message string, channel uint64, everyone bool, guild uint64) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_AddMessage.Exec(id, author, message, channel, everyone, guild)
	db.CheckError("AddMessage", err)
}

func (db *BotDB) GetMessage(id uint64) (uint64, string, time.Time, uint64) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var author uint64
	var message string
	var timestamp time.Time
	var channel uint64
	err := db.sql_GetMessage.QueryRow(id).Scan(&author, &message, &timestamp, &channel)
	if err == sql.ErrNoRows || db.CheckError("GetMessage", err) {
		return 0, "", time.Now().UTC(), 0
	}
	return author, message, timestamp, channel
}

type PingContext struct {
	Author    string
	Message   string
	Timestamp time.Time
}

func (db *BotDB) AddUser(id uint64, email string, username string, discriminator int, avatar string, verified bool, isonline bool) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_AddUser.Exec(id, email, username, discriminator, avatar, verified, isonline)
	db.CheckError("AddUser", err)
}

func (db *BotDB) AddMember(id uint64, guild uint64, firstseen time.Time, nickname string) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_AddMember.Exec(id, guild, firstseen, nickname)
	db.CheckError("AddMember", err)
}

func (db *BotDB) GetUser(id uint64) (*discordgo.User, time.Time, *time.Location, *uint64) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	u := &discordgo.User{}
	var lastseen time.Time
	var i sql.NullInt64
	var loc sql.NullString
	var guild sql.NullInt64
	var discriminator int = 0
	err := db.sql_GetUser.QueryRow(id).Scan(&u.ID, &u.Email, &u.Username, &discriminator, &u.Avatar, &lastseen, &i, &loc, &guild)
	if discriminator > 0 {
		u.Discriminator = strconv.Itoa(discriminator)
	}
	if err == sql.ErrNoRows || db.CheckError("GetUser", err) {
		return nil, lastseen, nil, nil
	}
	if !guild.Valid {
		return u, lastseen, evalTimeZone(i, loc), nil
	}
	g := uint64(guild.Int64)
	return u, lastseen, evalTimeZone(i, loc), &g
}

func (db *BotDB) GetMember(id uint64, guild uint64) (*discordgo.Member, time.Time) {
	db.statuslock.RLock()
	m := &discordgo.Member{}
	m.User = &discordgo.User{}
	var lastseen time.Time
	var discriminator int = 0
	err := db.sql_GetMember.QueryRow(id, guild).Scan(&m.User.ID, &m.User.Email, &m.User.Username, &discriminator, &m.User.Avatar, &lastseen, &m.Nick, &m.JoinedAt)
	if discriminator > 0 {
		m.User.Discriminator = strconv.Itoa(discriminator)
	}
	if err == sql.ErrNoRows {
		db.statuslock.RUnlock()
		m.User, lastseen, _, _ = db.GetUser(id)
		if m.User == nil {
			return nil, lastseen
		}
		return m, lastseen
	}
	db.CheckError("GetMember", err)
	db.statuslock.RUnlock()
	return m, lastseen
}

func (db *BotDB) FindGuildUsers(name string, maxresults uint64, offset uint64, guild uint64) []uint64 {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_FindGuildUsers.Query(guild, name, name, name, maxresults, offset)
	if db.CheckError("FindGuildUsers", err) {
		return []uint64{}
	}
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

func (db *BotDB) FindUsers(name string, maxresults uint64, offset uint64) []uint64 {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_FindUsers.Query(name, name, name, maxresults, offset)
	if db.CheckError("FindUsers", err) {
		return []uint64{}
	}
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

func (db *BotDB) GetRecentMessages(user uint64, duration uint64, guild uint64) []struct {
	message uint64
	channel uint64
} {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetRecentMessages.Query(guild, user, duration)
	if db.CheckError("GetRecentMessages", err) {
		return []struct {
			message uint64
			channel uint64
		}{}
	}
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
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetNewestUsers.Query(guild, maxresults)
	if db.CheckError("GetNewestUsers", err) {
		return []struct {
			User      *discordgo.User
			FirstSeen time.Time
		}{}
	}
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
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetRecentUsers.Query(guild, since)
	if db.CheckError("GetRecentUsers", err) {
		return []*discordgo.User{}
	}
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
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetAliases.Query(user)
	if db.CheckError("GetAliases", err) {
		return []string{}
	}
	defer q.Close()
	return db.ParseStringResults(q)
}

func (db *BotDB) Audit(ty uint8, user *discordgo.User, message string, guild uint64) {
	var err error
	if user == nil {
		_, err = db.sql_Audit.Exec(ty, nil, message, guild)
	} else {
		_, err = db.sql_Audit.Exec(ty, SBatoi(user.ID), message, guild)
	}

	if err != nil && sb.db.status.get() {
		fmt.Println("Logger failed to log to database! ", err.Error())
	}
}

func (db *BotDB) GetAuditRows(start uint64, end uint64, user *uint64, search string, guild uint64) []PingContext {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var q *sql.Rows
	var err error
	maxresults := end - start
	if maxresults > 50 {
		maxresults = 50
	}

	if user != nil && len(search) > 0 {
		q, err = db.sql_GetAuditRowsUserString.Query(AUDIT_TYPE_COMMAND, guild, *user, search, maxresults, start)
	} else if user != nil && len(search) == 0 {
		q, err = db.sql_GetAuditRowsUser.Query(AUDIT_TYPE_COMMAND, guild, *user, maxresults, start)
	} else if user == nil && len(search) > 0 {
		q, err = db.sql_GetAuditRowsString.Query(AUDIT_TYPE_COMMAND, guild, search, maxresults, start)
	} else {
		q, err = db.sql_GetAuditRows.Query(AUDIT_TYPE_COMMAND, guild, maxresults, start)
	}
	if db.CheckError("GetAuditRows", err) {
		return []PingContext{}
	}
	defer q.Close()
	r := make([]PingContext, 0, 5)
	for q.Next() {
		p := PingContext{}
		var uid uint64
		if err := q.Scan(&p.Author, &p.Message, &p.Timestamp, &uid); err == nil {
			r = append(r, p)
		}
	}
	return r
}

func (db *BotDB) GetTableCounts() string {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	if !db.status.get() {
		return "DATABASE ERROR"
	}
	var counts string
	err := db.sql_GetTableCounts.QueryRow().Scan(&counts)
	if db.CheckError("GetTableCounts", err) {
		return "DATABASE ERROR"
	}
	return counts
}

func (db *BotDB) AddTranscript(season int, episode int, line int, speaker string, text string) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
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
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetTranscript.Query(season, episode, start, end)
	if db.CheckError("GetTranscript", err) {
		return []Transcript{}
	}
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
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_RemoveTranscript.Exec(season, episode, line)
	db.CheckError("RemoveTranscript", err)
}
func (db *BotDB) AddMarkov(last uint64, last2 uint64, speaker string, text string) uint64 {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var id uint64
	err := db.sql_AddMarkov.QueryRow(last, last2, speaker, text).Scan(&id)
	db.CheckError("AddMarkov", err)
	return id
}

func (db *BotDB) GetMarkovLine(last uint64) (string, uint64) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var r sql.NullString
	err := db.sql_GetMarkovLine.QueryRow(last).Scan(&r)
	if db.CheckError("GetMarkovLine", err) || !r.Valid {
		return "", 0
	}
	str := strings.SplitN(r.String, "|", 2) // Being unable to call stored procedures makes this unnecessarily complex
	if len(str) < 2 || len(str[1]) < 1 {
		return str[0], 0
	}
	return str[0], SBatoi(str[1])
}

func (db *BotDB) GetMarkovLine2(last uint64, last2 uint64) (string, uint64, uint64) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var r sql.NullString
	err := db.sql_GetMarkovLine2.QueryRow(last, last2).Scan(&r)
	if db.CheckError("GetMarkovLine2", err) || !r.Valid {
		return "", 0, 0
	}
	str := strings.SplitN(r.String, "|", 3) // Being unable to call stored procedures makes this unnecessarily complex
	if len(str) < 3 || len(str[1]) < 1 || len(str[2]) < 1 {
		return str[0], 0, 0
	}
	return str[0], SBatoi(str[1]), SBatoi(str[2])
}
func (db *BotDB) GetMarkovWord(speaker string, phrase string) string {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var r string
	err := db.sql_GetMarkovWord.QueryRow(speaker, phrase).Scan(&r)
	if err == sql.ErrNoRows {
		return phrase
	}
	db.CheckError("GetMarkovWord", err)
	return r
}
func (db *BotDB) GetRandomQuote() Transcript {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var i uint64
	err := db.sql_GetRandomQuoteInt.QueryRow().Scan(&i)
	var p Transcript
	if db.CheckError("GetRandomQuoteInt", err) {
		err = db.sql_GetRandomQuote.QueryRow(i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
		db.CheckError("GetRandomQuote", err)
	}
	return p
}
func (db *BotDB) GetSpeechQuote() Transcript {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var i uint64
	err := db.sql_GetSpeechQuoteInt.QueryRow().Scan(&i)
	var p Transcript
	if db.CheckError("GetSpeechQuoteInt", err) {
		err = db.sql_GetSpeechQuote.QueryRow(i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
		db.CheckError("GetSpeechQuote", err)
	}
	return p
}
func (db *BotDB) GetCharacterQuote(character string) Transcript {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var i uint64
	err := db.sql_GetCharacterQuoteInt.QueryRow(character).Scan(&i)
	var p Transcript
	if db.CheckError("GetCharacterQuoteInt ", err) {
		err = db.sql_GetCharacterQuote.QueryRow(character, i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
		if err == sql.ErrNoRows || db.CheckError("GetCharacterQuote ", err) {
			return Transcript{0, 0, 0, "", ""}
		}
	}
	return p
}
func (db *BotDB) GetRandomSpeaker() string {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var i uint64
	err := db.sql_GetRandomSpeakerInt.QueryRow().Scan(&i)
	var p string
	if db.CheckError("GetRandomSpeakerInt", err) {
		err = db.sql_GetRandomSpeaker.QueryRow(i).Scan(&p)
		db.CheckError("GetRandomSpeaker", err)
	}
	return p
}
func (db *BotDB) GetRandomMember(guild uint64) string {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var i uint64
	err := db.sql_GetRandomMemberInt.QueryRow(guild).Scan(&i)
	var p string
	if db.CheckError("GetRandomMemberInt", err) {
		err = db.sql_GetRandomMember.QueryRow(guild, i).Scan(&p)
		db.CheckError("GetRandomMember", err)
	}
	return p
}
func (db *BotDB) GetRandomWord() string {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var i uint64
	err := db.sql_GetRandomWordInt.QueryRow().Scan(&i)
	var p string
	if db.CheckError("GetRandomWordInt", err) {
		err = db.sql_GetRandomWord.QueryRow(i).Scan(&p)
		db.CheckError("GetRandomWord", err)
	}
	return p
}
func (db *BotDB) CountNewUsers(seconds int64, guild uint64) int {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var i int
	err := db.sql_CountNewUsers.QueryRow(seconds, guild).Scan(&i)
	db.CheckError("CountNewUsers", err)
	return i
}

func (db *BotDB) RemoveSchedule(id uint64) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_RemoveSchedule.Exec(id)
	db.CheckError("RemoveSchedule", err)
}
func (db *BotDB) AddSchedule(guild uint64, date time.Time, ty uint8, data string) bool {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var i int
	err := db.sql_CountEvents.QueryRow(guild).Scan(&i)

	if !db.CheckError("CountEvents", err) && i < 5000 {
		_, err = db.sql_AddSchedule.Exec(guild, date, ty, data)
		return !db.CheckError("AddSchedule", err)
	}
	return false
}
func (db *BotDB) AddScheduleRepeat(guild uint64, date time.Time, repeatinterval uint8, repeat int, ty uint8, data string) bool {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var i int
	err := db.sql_CountEvents.QueryRow(guild).Scan(&i)
	if !db.CheckError("CountEvents", err) && i < 5000 {
		_, err := db.sql_AddScheduleRepeat.Exec(guild, date, repeatinterval, repeat, ty, data)
		return !db.CheckError("AddScheduleRepeat", err)
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
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetSchedule.Query(guild)
	if db.CheckError("GetSchedule", err) {
		return []ScheduleEvent{}
	}
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

func (db *BotDB) GetEvent(id uint64) *ScheduleEvent {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	e := &ScheduleEvent{}
	err := db.sql_GetEvent.QueryRow(id).Scan(&e.ID, &e.Date, &e.Type, &e.Data)
	if err == sql.ErrNoRows || db.CheckError("GetEvent", err) {
		return nil
	}
	return e
}

func (db *BotDB) GetEvents(guild uint64, maxnum int) []ScheduleEvent {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetEvents.Query(guild, maxnum)
	if db.CheckError("GetEvents", err) {
		return []ScheduleEvent{}
	}
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
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetEventsByType.Query(guild, ty, maxnum)
	if db.CheckError("GetEventsByType", err) {
		return []ScheduleEvent{}
	}
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
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	p := ScheduleEvent{}
	err := db.sql_GetNextEvent.QueryRow(guild, ty).Scan(&p.ID, &p.Date, &p.Type, &p.Data)
	if err == sql.ErrNoRows || db.CheckError("GetNextEvent", err) {
		return ScheduleEvent{0, time.Now().UTC(), 0, ""}
	}
	return p
}

func (db *BotDB) GetReminders(guild uint64, id string, maxnum int) []ScheduleEvent {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetReminders.Query(guild, id+"|%", maxnum)
	if db.CheckError("GetReminders", err) {
		return []ScheduleEvent{}
	}
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

func (db *BotDB) GetUnsilenceDate(guild uint64, id uint64) *time.Time {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var timestamp time.Time
	err := db.sql_GetUnsilenceDate.QueryRow(guild, id).Scan(&timestamp)
	if err == sql.ErrNoRows || db.CheckError("GetUnsilenceDate", err) {
		return nil
	}
	return &timestamp
}

func evalTimeZone(i sql.NullInt64, loc sql.NullString) *time.Location {
	if loc.Valid && len(loc.String) > 0 {
		l, err := time.LoadLocation(loc.String)
		if err == nil {
			return l
		}
	}
	if i.Valid {
		return time.FixedZone("Legacy/GMT"+strconv.FormatInt(i.Int64, 10), int(i.Int64*3600))
	}
	return nil
}

func (db *BotDB) GetTimeZone(user uint64) *time.Location {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var i sql.NullInt64
	var loc sql.NullString
	err := db.sql_GetTimeZone.QueryRow(user).Scan(&i, &loc)
	if db.CheckError("GetTimeZone", err) {
		return nil
	}
	return evalTimeZone(i, loc)
}

func (db *BotDB) FindTimeZone(s string) []string {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_FindTimeZone.Query(s)
	if db.CheckError("FindTimeZone", err) {
		return []string{}
	}
	defer q.Close()
	r := make([]string, 0, 2)
	for q.Next() {
		var s string
		if err := q.Scan(&s); err == nil {
			r = append(r, s)
		}
	}
	return r
}

func (db *BotDB) FindTimeZoneOffset(s string, minutes int) []string {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_FindTimeZoneOffset.Query(s, minutes, minutes)
	if db.CheckError("FindTimeZoneOffset", err) {
		return []string{}
	}
	defer q.Close()
	r := make([]string, 0, 2)
	for q.Next() {
		var s string
		if err := q.Scan(&s); err == nil {
			r = append(r, s)
		}
	}
	return r
}

func (db *BotDB) SetTimeZone(user uint64, tz *time.Location) error {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_SetTimeZone.Exec(tz.String(), user)
	db.CheckError("SetTimeZone", err)
	return err
}

func (db *BotDB) RemoveAlias(user uint64, alias string) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_RemoveAlias.Exec(user, alias)
	db.CheckError("RemoveAlias", err)
}

func (db *BotDB) GetUserGuilds(user uint64) []uint64 {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetUserGuilds.Query(user)
	if db.CheckError("GetUserGuilds", err) {
		return []uint64{}
	}
	defer q.Close()
	r := make([]uint64, 0, 2)
	for q.Next() {
		var s uint64
		if err := q.Scan(&s); err == nil {
			r = append(r, s)
		}
	}
	return r
}

func (db *BotDB) FindEvent(user string, guild uint64, ty uint8) *uint64 {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var id uint64
	err := db.sql_FindEvent.QueryRow(ty, user, guild).Scan(&id)
	if err == sql.ErrNoRows || db.CheckError("FindEvent", err) {
		return nil
	}
	return &id
}

func (db *BotDB) SetDefaultServer(user uint64, server uint64) error {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_SetDefaultServer.Exec(server, user)
	db.CheckError("SetDefaultServer", err)
	return err
}

func (db *BotDB) GetPolls(server uint64) []struct {
	name        string
	description string
} {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetPolls.Query(server)
	if db.CheckError("GetPolls", err) {
		return []struct {
			name        string
			description string
		}{}
	}
	defer q.Close()
	r := make([]struct {
		name        string
		description string
	}, 0, 2)
	for q.Next() {
		var s struct {
			name        string
			description string
		}
		if err := q.Scan(&s.name, &s.description); err == nil {
			r = append(r, s)
		}
	}
	return r
}

func (db *BotDB) GetPoll(name string, server uint64) (uint64, string) {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var id uint64
	var desc string
	err := db.sql_GetPoll.QueryRow(name, server).Scan(&id, &desc)
	if err == sql.ErrNoRows || db.CheckError("GetPoll", err) {
		return 0, ""
	}
	return id, desc
}

type PollOptionStruct struct {
	index  uint64
	option string
}

func (db *BotDB) GetOptions(poll uint64) []PollOptionStruct {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetOptions.Query(poll)
	if db.CheckError("GetOptions", err) {
		return []PollOptionStruct{}
	}
	defer q.Close()
	r := make([]PollOptionStruct, 0, 2)
	for q.Next() {
		var s PollOptionStruct
		if err := q.Scan(&s.index, &s.option); err == nil {
			r = append(r, s)
		}
	}
	return r
}

func (db *BotDB) GetOption(poll uint64, option string) *uint64 {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var id uint64
	err := db.sql_GetOption.QueryRow(poll, option).Scan(&id)
	if err == sql.ErrNoRows || db.CheckError("GetOption", err) {
		return nil
	}
	return &id
}

type PollResultStruct struct {
	index uint64
	count uint64
}

func (db *BotDB) GetResults(poll uint64) []PollResultStruct {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	q, err := db.sql_GetResults.Query(poll)
	if db.CheckError("GetResults", err) {
		return []PollResultStruct{}
	}
	defer q.Close()
	r := make([]PollResultStruct, 0, 2)
	for q.Next() {
		var s PollResultStruct
		if err := q.Scan(&s.index, &s.count); err == nil {
			r = append(r, s)
		}
	}
	return r
}

func (db *BotDB) AddPoll(name string, description string, server uint64) error {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_AddPoll.Exec(name, description, server)
	db.CheckError("AddPoll", err)
	return err
}

func (db *BotDB) AddOption(poll uint64, index uint64, option string) error {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_AddOption.Exec(poll, index, option)
	db.CheckError("AddOption", err)
	return err
}

func (db *BotDB) AppendOption(poll uint64, option string) error {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_AppendOption.Exec(option, poll)
	db.CheckError("AppendOption", err)
	return err
}

func (db *BotDB) AddVote(user uint64, poll uint64, option uint64) error {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_AddVote.Exec(poll, user, option, option)
	db.CheckError("AddVote", err)
	return err
}

func (db *BotDB) RemovePoll(name string, server uint64) error {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	_, err := db.sql_RemovePoll.Exec(name, server)
	db.CheckError("RemovePoll", err)
	return err
}

func (db *BotDB) CheckOption(poll uint64, option uint64) bool {
	db.statuslock.RLock()
	defer db.statuslock.RUnlock()
	var name string
	err := db.sql_CheckOption.QueryRow(poll, option).Scan(&name)
	if err == sql.ErrNoRows || db.CheckError("CheckOption", err) {
		return false
	}
	return true
}
