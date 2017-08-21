package sweetiebot

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blackhole12/discordgo"
	_ "github.com/go-sql-driver/mysql"
)

type BotDB struct {
	db                        *sql.DB
	status                    AtomicBool
	lastattempt               time.Time
	log                       logger
	driver                    string
	conn                      string
	statuslock                AtomicFlag
	sqlAddMessage             *sql.Stmt
	sqlGetMessage             *sql.Stmt
	sqlAddUser                *sql.Stmt
	sqlAddMember              *sql.Stmt
	sqlRemoveMember           *sql.Stmt
	sqlGetUser                *sql.Stmt
	sqlGetMember              *sql.Stmt
	sqlFindGuildUsers         *sql.Stmt
	sqlFindUsers              *sql.Stmt
	sqlGetRecentMessages      *sql.Stmt
	sqlGetNewestUsers         *sql.Stmt
	sqlGetRecentUsers         *sql.Stmt
	sqlGetAliases             *sql.Stmt
	sqlAddTranscript          *sql.Stmt
	sqlGetTranscript          *sql.Stmt
	sqlRemoveTranscript       *sql.Stmt
	sqlAddMarkov              *sql.Stmt
	sqlGetMarkovLine          *sql.Stmt
	sqlGetMarkovLine2         *sql.Stmt
	sqlGetMarkovWord          *sql.Stmt
	sqlGetRandomQuoteInt      *sql.Stmt
	sqlGetRandomQuote         *sql.Stmt
	sqlGetSpeechQuoteInt      *sql.Stmt
	sqlGetSpeechQuote         *sql.Stmt
	sqlGetCharacterQuoteInt   *sql.Stmt
	sqlGetCharacterQuote      *sql.Stmt
	sqlGetRandomSpeakerInt    *sql.Stmt
	sqlGetRandomSpeaker       *sql.Stmt
	sqlGetRandomMemberInt     *sql.Stmt
	sqlGetRandomMember        *sql.Stmt
	sqlGetRandomWordInt       *sql.Stmt
	sqlGetRandomWord          *sql.Stmt
	sqlGetTableCounts         *sql.Stmt
	sqlCountNewUsers          *sql.Stmt
	sqlAudit                  *sql.Stmt
	sqlGetAuditRows           *sql.Stmt
	sqlGetAuditRowsUser       *sql.Stmt
	sqlGetAuditRowsString     *sql.Stmt
	sqlGetAuditRowsUserString *sql.Stmt
	sqlResetMarkov            *sql.Stmt
	sqlAddSchedule            *sql.Stmt
	sqlAddScheduleRepeat      *sql.Stmt
	sqlGetSchedule            *sql.Stmt
	sqlRemoveSchedule         *sql.Stmt
	sqlCountEvents            *sql.Stmt
	sqlGetEvent               *sql.Stmt
	sqlGetEvents              *sql.Stmt
	sqlGetEventsByType        *sql.Stmt
	sqlGetNextEvent           *sql.Stmt
	sqlGetReminders           *sql.Stmt
	sqlGetUnsilenceDate       *sql.Stmt
	sqlGetTimeZone            *sql.Stmt
	sqlFindTimeZone           *sql.Stmt
	sqlFindTimeZoneOffset     *sql.Stmt
	sqlSetTimeZone            *sql.Stmt
	sqlRemoveAlias            *sql.Stmt
	sqlGetUserGuilds          *sql.Stmt
	sqlFindEvent              *sql.Stmt
	sqlSetDefaultServer       *sql.Stmt
	sqlGetPolls               *sql.Stmt
	sqlGetPoll                *sql.Stmt
	sqlGetOptions             *sql.Stmt
	sqlGetOption              *sql.Stmt
	sqlGetResults             *sql.Stmt
	sqlAddPoll                *sql.Stmt
	sqlAddOption              *sql.Stmt
	sqlAppendOption           *sql.Stmt
	sqlAddVote                *sql.Stmt
	sqlRemovePoll             *sql.Stmt
	sqlCheckOption            *sql.Stmt
	sqlSentMessage            *sql.Stmt
	sqlGetNewcomers           *sql.Stmt
}

func DB_Load(log logger, driver string, conn string) (*BotDB, error) {
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

	r.db.SetMaxOpenConns(70)
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

const DB_RECONNECT_TIMEOUT = time.Duration(30) * time.Second // Reconnect time interval in seconds

func (db *BotDB) CheckStatus() bool {
	if !db.status.get() {
		if db.statuslock.test_and_set() { // If this was already true, bail out
			return false
		}
		defer db.statuslock.clear()

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
	db.sqlAddMessage, err = db.Prepare("CALL AddChat(?,?,?,?,?,?)")
	db.sqlGetMessage, err = db.Prepare("SELECT Author, Message, Timestamp, Channel FROM chatlog WHERE ID = ?")
	db.sqlAddUser, err = db.Prepare("CALL AddUser(?,?,?,?,?,?,?)")
	db.sqlAddMember, err = db.Prepare("CALL AddMember(?,?,?,?)")
	db.sqlRemoveMember, err = db.Prepare("DELETE FROM `members` WHERE Guild = ? AND ID = ?")
	db.sqlGetUser, err = db.Prepare("SELECT ID, Email, Username, Discriminator, Avatar, LastSeen, Location, DefaultServer FROM users WHERE ID = ?")
	db.sqlGetMember, err = db.Prepare("SELECT U.ID, U.Email, U.Username, U.Discriminator, U.Avatar, U.LastSeen, M.Nickname, M.FirstSeen, M.FirstMessage FROM members M RIGHT OUTER JOIN users U ON U.ID = M.ID WHERE M.ID = ? AND M.Guild = ?")
	db.sqlFindGuildUsers, err = db.Prepare("SELECT U.ID FROM users U LEFT OUTER JOIN aliases A ON A.User = U.ID LEFT OUTER JOIN members M ON M.ID = U.ID WHERE M.Guild = ? AND (U.Username LIKE ? OR M.Nickname LIKE ? OR A.Alias = ?) GROUP BY U.ID LIMIT ? OFFSET ?")
	db.sqlFindUsers, err = db.Prepare("SELECT U.ID FROM users U LEFT OUTER JOIN aliases A ON A.User = U.ID LEFT OUTER JOIN members M ON M.ID = U.ID WHERE U.Username LIKE ? OR M.Nickname LIKE ? OR A.Alias = ? GROUP BY U.ID LIMIT ? OFFSET ?")
	db.sqlGetRecentMessages, err = db.Prepare("SELECT ID, Channel FROM chatlog WHERE Guild = ? AND Author = ? AND Timestamp >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL ? SECOND)")
	db.sqlGetNewestUsers, err = db.Prepare("SELECT U.ID, U.Email, U.Username, U.Avatar, M.FirstSeen FROM members M INNER JOIN users U ON M.ID = U.ID WHERE M.Guild = ? ORDER BY M.FirstSeen DESC LIMIT ?")
	db.sqlGetRecentUsers, err = db.Prepare("SELECT U.ID, U.Email, U.Username, U.Avatar FROM members M INNER JOIN users U ON M.ID = U.ID WHERE M.Guild = ? AND M.FirstSeen > ? ORDER BY M.FirstSeen DESC")
	db.sqlGetAliases, err = db.Prepare("SELECT Alias FROM aliases WHERE User = ? ORDER BY Duration DESC LIMIT 10")
	db.sqlAddTranscript, err = db.Prepare("INSERT INTO transcripts (Season, Episode, Line, Speaker, Text) VALUES (?,?,?,?,?)")
	db.sqlGetTranscript, err = db.Prepare("SELECT Season, Episode, Line, Speaker, Text FROM transcripts WHERE Season = ? AND Episode = ? AND Line >= ? AND LINE <= ?")
	db.sqlRemoveTranscript, err = db.Prepare("DELETE FROM transcripts WHERE Season = ? AND Episode = ? AND Line = ?")
	db.sqlAddMarkov, err = db.Prepare("SELECT AddMarkov(?,?,?,?)")
	db.sqlGetMarkovLine, err = db.Prepare("SELECT GetMarkovLine(?)")
	db.sqlGetMarkovLine2, err = db.Prepare("SELECT GetMarkovLine2(?,?)")
	db.sqlGetMarkovWord, err = db.Prepare("SELECT Phrase FROM markov_transcripts WHERE SpeakerID = (SELECT ID FROM markov_transcripts_speaker WHERE Speaker = ?) AND Phrase = ?")
	db.sqlGetRandomQuoteInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM transcripts WHERE Text != ''))")
	db.sqlGetRandomQuote, err = db.Prepare("SELECT * FROM transcripts WHERE Text != '' LIMIT 1 OFFSET ?")
	db.sqlGetSpeechQuoteInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM transcripts WHERE Speaker != 'ACTION' AND Text != ''))")
	db.sqlGetSpeechQuote, err = db.Prepare("SELECT * FROM transcripts WHERE Speaker != 'ACTION' AND Text != '' LIMIT 1 OFFSET ?")
	db.sqlGetCharacterQuoteInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM transcripts WHERE Speaker = ? AND Text != ''))")
	db.sqlGetCharacterQuote, err = db.Prepare("SELECT * FROM transcripts WHERE Speaker = ? AND Text != '' LIMIT 1 OFFSET ?")
	db.sqlGetRandomSpeakerInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM markov_transcripts_speaker))")
	db.sqlGetRandomSpeaker, err = db.Prepare("SELECT Speaker FROM markov_transcripts_speaker LIMIT 1 OFFSET ?")
	db.sqlGetRandomMemberInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM members WHERE Guild = ?))")
	db.sqlGetRandomMember, err = db.Prepare("SELECT U.Username FROM members M INNER JOIN users U ON M.ID = U.ID WHERE M.Guild = ? LIMIT 1 OFFSET ?")
	db.sqlGetRandomWordInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM randomwords))")
	db.sqlGetRandomWord, err = db.Prepare("SELECT Phrase FROM randomwords LIMIT 1 OFFSET ?;")
	db.sqlGetTableCounts, err = db.Prepare("SELECT CONCAT('Chatlog: ', (SELECT COUNT(*) FROM chatlog), ' rows', '\nEditlog: ', (SELECT COUNT(*) FROM editlog), ' rows',  '\nAliases: ', (SELECT COUNT(*) FROM aliases), ' rows',  '\nDebuglog: ', (SELECT COUNT(*) FROM debuglog), ' rows',  '\nUsers: ', (SELECT COUNT(*) FROM users), ' rows',  '\nSchedule: ', (SELECT COUNT(*) FROM schedule), ' rows \nMembers: ', (SELECT COUNT(*) FROM members), ' rows');")
	db.sqlCountNewUsers, err = db.Prepare("SELECT COUNT(*) FROM members WHERE FirstSeen > DATE_SUB(UTC_TIMESTAMP(), INTERVAL ? SECOND) AND Guild = ?")
	db.sqlAudit, err = db.Prepare("INSERT INTO debuglog (Type, User, Message, Timestamp, Guild) VALUE(?, ?, ?, UTC_TIMESTAMP(), ?)")
	db.sqlGetAuditRows, err = db.Prepare("SELECT U.Username, D.Message, D.Timestamp, U.ID FROM debuglog D INNER JOIN users U ON D.User = U.ID WHERE D.Type = ? AND D.Guild = ? ORDER BY D.Timestamp DESC LIMIT ? OFFSET ?")
	db.sqlGetAuditRowsUser, err = db.Prepare("SELECT U.Username, D.Message, D.Timestamp, U.ID FROM debuglog D INNER JOIN users U ON D.User = U.ID WHERE D.Type = ? AND D.Guild = ? AND D.User = ? ORDER BY D.Timestamp DESC LIMIT ? OFFSET ?")
	db.sqlGetAuditRowsString, err = db.Prepare("SELECT U.Username, D.Message, D.Timestamp, U.ID FROM debuglog D INNER JOIN users U ON D.User = U.ID WHERE D.Type = ? AND D.Guild = ? AND D.Message LIKE ? ORDER BY D.Timestamp DESC LIMIT ? OFFSET ?")
	db.sqlGetAuditRowsUserString, err = db.Prepare("SELECT U.Username, D.Message, D.Timestamp, U.ID FROM debuglog D INNER JOIN users U ON D.User = U.ID WHERE D.Type = ? AND D.Guild = ? AND D.User = ? AND D.Message LIKE ? ORDER BY D.Timestamp DESC LIMIT ? OFFSET ?")
	db.sqlResetMarkov, err = db.Prepare("CALL ResetMarkov()")
	db.sqlAddSchedule, err = db.Prepare("INSERT INTO schedule (Guild, Date, Type, Data) VALUES (?, ?, ?, ?)")
	db.sqlAddScheduleRepeat, err = db.Prepare("INSERT INTO schedule (Guild, Date, `RepeatInterval`, `Repeat`, Type, Data) VALUES (?, ?, ?, ?, ?, ?)")
	db.sqlGetSchedule, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Date <= UTC_TIMESTAMP() ORDER BY Date ASC")
	db.sqlRemoveSchedule, err = db.Prepare("CALL RemoveSchedule(?)")
	db.sqlCountEvents, err = db.Prepare("SELECT COUNT(*) FROM schedule WHERE Guild = ?")
	db.sqlGetEvent, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE ID = ?")
	db.sqlGetEvents, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type != 0 AND Type != 4 AND Type != 6 ORDER BY Date ASC LIMIT ?")
	db.sqlGetEventsByType, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = ? ORDER BY Date ASC LIMIT ?")
	db.sqlGetNextEvent, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = ? ORDER BY Date ASC LIMIT 1")
	db.sqlGetReminders, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = 6 AND Data LIKE ? ORDER BY Date ASC LIMIT ?")
	db.sqlGetUnsilenceDate, err = db.Prepare("SELECT Date FROM schedule WHERE Guild = ? AND Type = 8 AND Data = ?")
	db.sqlGetTimeZone, err = db.Prepare("SELECT Location FROM users WHERE ID = ?")
	db.sqlFindTimeZone, err = db.Prepare("SELECT Location FROM timezones WHERE Location LIKE ?")
	db.sqlFindTimeZoneOffset, err = db.Prepare("SELECT Location FROM timezones WHERE Location LIKE ? AND (Offset = ? OR DST = ?)")
	db.sqlSetTimeZone, err = db.Prepare("UPDATE users SET Location = ? WHERE ID = ?")
	db.sqlRemoveAlias, err = db.Prepare("DELETE FROM aliases WHERE User = ? AND Alias = ?")
	db.sqlGetUserGuilds, err = db.Prepare("SELECT Guild FROM members WHERE ID = ?")
	db.sqlFindEvent, err = db.Prepare("SELECT ID FROM `schedule` WHERE `Type` = ? AND `Data` = ? AND `Guild` = ?")
	db.sqlSetDefaultServer, err = db.Prepare("UPDATE users SET DefaultServer = ? WHERE ID = ?")
	db.sqlGetPolls, err = db.Prepare("SELECT Name, Description FROM polls WHERE Guild = ? ORDER BY ID DESC")
	db.sqlGetPoll, err = db.Prepare("SELECT ID, Description FROM polls WHERE Name = ? AND Guild = ?")
	db.sqlGetOptions, err = db.Prepare("SELECT `Index`, `Option` FROM polloptions WHERE Poll = ? ORDER BY `Index` ASC")
	db.sqlGetOption, err = db.Prepare("SELECT `Index` FROM polloptions WHERE poll = ? AND `Option` = ?")
	db.sqlGetResults, err = db.Prepare("SELECT `Option`,COUNT(user) FROM `votes` WHERE `Poll` = ? GROUP BY `Option` ORDER BY `Option` ASC")
	db.sqlAddPoll, err = db.Prepare("INSERT INTO polls(Name, Description, Guild) VALUES (?, ?, ?)")
	db.sqlAddOption, err = db.Prepare("INSERT INTO polloptions(Poll, `Index`, `Option`) VALUES (?, ?, ?)")
	db.sqlAppendOption, err = db.Prepare("INSERT INTO polloptions(Poll, `Index`, `Option`) SELECT Poll, MAX(`index`)+1, ? FROM polloptions WHERE poll = ?")
	db.sqlAddVote, err = db.Prepare("INSERT INTO votes (Poll, User, `Option`) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE `Option` = ?")
	db.sqlRemovePoll, err = db.Prepare("DELETE FROM polls WHERE Name = ? AND Guild = ?")
	db.sqlCheckOption, err = db.Prepare("SELECT `Option` FROM polloptions WHERE poll = ? AND `Index` = ?")
	db.sqlSentMessage, err = db.Prepare("UPDATE `members` SET `FirstMessage` = UTC_TIMESTAMP() WHERE ID = ? AND Guild = ? AND `FirstMessage` IS NULL")
	db.sqlGetNewcomers, err = db.Prepare("SELECT ID FROM `members` WHERE `Guild` = ? AND `FirstMessage` > DATE_SUB(UTC_TIMESTAMP(), INTERVAL ? SECOND)")
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
		if db.status.get() {
			db.log.LogError(name+" error: ", err)
		}
		if err != sql.ErrNoRows && err != sql.ErrTxDone {
			if db.db.Ping() != nil {
				db.status.set(false)
			}
			return true
		}
	}
	return false
}

func (db *BotDB) AddMessage(id uint64, author uint64, message string, channel uint64, everyone bool, guild uint64) {
	_, err := db.sqlAddMessage.Exec(id, author, message, channel, everyone, guild)
	db.CheckError("AddMessage", err)
}

func (db *BotDB) GetMessage(id uint64) (uint64, string, time.Time, uint64) {
	var author uint64
	var message string
	var timestamp time.Time
	var channel uint64
	err := db.sqlGetMessage.QueryRow(id).Scan(&author, &message, &timestamp, &channel)
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
	_, err := db.sqlAddUser.Exec(id, email, username, discriminator, avatar, verified, isonline)
	db.CheckError("AddUser", err)
}

func (db *BotDB) AddMember(id uint64, guild uint64, firstseen time.Time, nickname string) {
	_, err := db.sqlAddMember.Exec(id, guild, firstseen, nickname)
	db.CheckError("AddMember", err)
}
func (db *BotDB) RemoveMember(id uint64, guild uint64) error {
	_, err := db.sqlRemoveMember.Exec(guild, id)
	db.CheckError("RemoveMember", err)
	return err
}

func (db *BotDB) GetUser(id uint64) (*discordgo.User, time.Time, *time.Location, *uint64) {
	u := &discordgo.User{}
	var lastseen time.Time
	var loc sql.NullString
	var guild sql.NullInt64
	var discriminator int = 0
	err := db.sqlGetUser.QueryRow(id).Scan(&u.ID, &u.Email, &u.Username, &discriminator, &u.Avatar, &lastseen, &loc, &guild)
	if discriminator > 0 {
		u.Discriminator = strconv.Itoa(discriminator)
	}
	if err == sql.ErrNoRows || db.CheckError("GetUser", err) {
		return nil, lastseen, nil, nil
	}
	if !guild.Valid {
		return u, lastseen, evalTimeZone(loc), nil
	}
	g := uint64(guild.Int64)
	return u, lastseen, evalTimeZone(loc), &g
}

func (db *BotDB) GetMember(id uint64, guild uint64) (*discordgo.Member, time.Time, *time.Time) {
	m := &discordgo.Member{}
	m.User = &discordgo.User{}
	var lastseen time.Time
	var firstmessage *time.Time
	var joinedat time.Time
	var discriminator int = 0
	err := db.sqlGetMember.QueryRow(id, guild).Scan(&m.User.ID, &m.User.Email, &m.User.Username, &discriminator, &m.User.Avatar, &lastseen, &m.Nick, &joinedat, &firstmessage)
	if !joinedat.IsZero() {
		m.JoinedAt = joinedat.Format(time.RFC3339)
	}
	if discriminator > 0 {
		m.User.Discriminator = strconv.Itoa(discriminator)
	}
	if err == sql.ErrNoRows {
		m.User, lastseen, _, _ = db.GetUser(id)
		if m.User == nil {
			return nil, lastseen, firstmessage
		}
		return m, lastseen, firstmessage
	}
	db.CheckError("GetMember", err)
	return m, lastseen, firstmessage
}

func (db *BotDB) FindGuildUsers(name string, maxresults uint64, offset uint64, guild uint64) []uint64 {
	q, err := db.sqlFindGuildUsers.Query(guild, name, name, name, maxresults, offset)
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
	q, err := db.sqlFindUsers.Query(name, name, name, maxresults, offset)
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
	q, err := db.sqlGetRecentMessages.Query(guild, user, duration)
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
	q, err := db.sqlGetNewestUsers.Query(guild, maxresults)
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
	q, err := db.sqlGetRecentUsers.Query(guild, since)
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
	q, err := db.sqlGetAliases.Query(user)
	if db.CheckError("GetAliases", err) {
		return []string{}
	}
	defer q.Close()
	return db.ParseStringResults(q)
}

func (db *BotDB) Audit(ty uint8, user *discordgo.User, message string, guild uint64) {
	var err error
	if user == nil {
		_, err = db.sqlAudit.Exec(ty, nil, message, guild)
	} else {
		_, err = db.sqlAudit.Exec(ty, SBatoi(user.ID), message, guild)
	}

	if err != nil && sb.db.status.get() {
		fmt.Println("Logger failed to log to database! ", err.Error())
	}
}

func (db *BotDB) GetAuditRows(start uint64, end uint64, user *uint64, search string, guild uint64) []PingContext {
	var q *sql.Rows
	var err error
	maxresults := end - start
	if maxresults > 50 {
		maxresults = 50
	}

	if user != nil && len(search) > 0 {
		q, err = db.sqlGetAuditRowsUserString.Query(AUDIT_TYPE_COMMAND, guild, *user, search, maxresults, start)
	} else if user != nil && len(search) == 0 {
		q, err = db.sqlGetAuditRowsUser.Query(AUDIT_TYPE_COMMAND, guild, *user, maxresults, start)
	} else if user == nil && len(search) > 0 {
		q, err = db.sqlGetAuditRowsString.Query(AUDIT_TYPE_COMMAND, guild, search, maxresults, start)
	} else {
		q, err = db.sqlGetAuditRows.Query(AUDIT_TYPE_COMMAND, guild, maxresults, start)
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
	if !db.status.get() {
		return "DATABASE ERROR"
	}
	var counts string
	err := db.sqlGetTableCounts.QueryRow().Scan(&counts)
	if db.CheckError("GetTableCounts", err) {
		return "DATABASE ERROR"
	}
	return counts
}

func (db *BotDB) AddTranscript(season int, episode int, line int, speaker string, text string) {
	_, err := db.sqlAddTranscript.Exec(season, episode, line, speaker, text)
	if err != nil {
		db.log.Log("AddTranscript error: ", err.Error(), "\nS", season, "E", episode, ":", line, " ", speaker, ": ", text)
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
	q, err := db.sqlGetTranscript.Query(season, episode, start, end)
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
	_, err := db.sqlRemoveTranscript.Exec(season, episode, line)
	db.CheckError("RemoveTranscript", err)
}
func (db *BotDB) AddMarkov(last uint64, last2 uint64, speaker string, text string) uint64 {
	var id uint64
	err := db.sqlAddMarkov.QueryRow(last, last2, speaker, text).Scan(&id)
	db.CheckError("AddMarkov", err)
	return id
}

func (db *BotDB) GetMarkovLine(last uint64) (string, uint64) {
	var r sql.NullString
	err := db.sqlGetMarkovLine.QueryRow(last).Scan(&r)
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
	var r sql.NullString
	err := db.sqlGetMarkovLine2.QueryRow(last, last2).Scan(&r)
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
	var r string
	err := db.sqlGetMarkovWord.QueryRow(speaker, phrase).Scan(&r)
	if err == sql.ErrNoRows {
		return phrase
	}
	db.CheckError("GetMarkovWord", err)
	return r
}
func (db *BotDB) GetRandomQuote() Transcript {
	var i uint64
	err := db.sqlGetRandomQuoteInt.QueryRow().Scan(&i)
	var p Transcript
	if !db.CheckError("GetRandomQuoteInt", err) {
		err = db.sqlGetRandomQuote.QueryRow(i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
		db.CheckError("GetRandomQuote", err)
	}
	return p
}
func (db *BotDB) GetSpeechQuote() Transcript {
	var i uint64
	err := db.sqlGetSpeechQuoteInt.QueryRow().Scan(&i)
	var p Transcript
	if !db.CheckError("GetSpeechQuoteInt", err) {
		err = db.sqlGetSpeechQuote.QueryRow(i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
		db.CheckError("GetSpeechQuote", err)
	}
	return p
}
func (db *BotDB) GetCharacterQuote(character string) Transcript {
	var i uint64
	err := db.sqlGetCharacterQuoteInt.QueryRow(character).Scan(&i)
	var p Transcript
	if !db.CheckError("GetCharacterQuoteInt ", err) {
		err = db.sqlGetCharacterQuote.QueryRow(character, i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
		if err == sql.ErrNoRows || db.CheckError("GetCharacterQuote ", err) {
			return Transcript{0, 0, 0, "", ""}
		}
	}
	return p
}
func (db *BotDB) GetRandomSpeaker() string {
	var i uint64
	err := db.sqlGetRandomSpeakerInt.QueryRow().Scan(&i)
	var p string
	if !db.CheckError("GetRandomSpeakerInt", err) {
		err = db.sqlGetRandomSpeaker.QueryRow(i).Scan(&p)
		db.CheckError("GetRandomSpeaker", err)
	}
	return p
}
func (db *BotDB) GetRandomMember(guild uint64) string {
	var i uint64
	err := db.sqlGetRandomMemberInt.QueryRow(guild).Scan(&i)
	var p string
	if !db.CheckError("GetRandomMemberInt", err) {
		err = db.sqlGetRandomMember.QueryRow(guild, i).Scan(&p)
		db.CheckError("GetRandomMember", err)
	}
	return p
}
func (db *BotDB) GetRandomWord() string {
	var i uint64
	err := db.sqlGetRandomWordInt.QueryRow().Scan(&i)
	var p string
	if !db.CheckError("GetRandomWordInt", err) {
		err = db.sqlGetRandomWord.QueryRow(i).Scan(&p)
		db.CheckError("GetRandomWord", err)
	}
	return p
}
func (db *BotDB) CountNewUsers(seconds int64, guild uint64) int {
	var i int
	err := db.sqlCountNewUsers.QueryRow(seconds, guild).Scan(&i)
	db.CheckError("CountNewUsers", err)
	return i
}

func (db *BotDB) RemoveSchedule(id uint64) {
	_, err := db.sqlRemoveSchedule.Exec(id)
	db.CheckError("RemoveSchedule", err)
}
func (db *BotDB) AddSchedule(guild uint64, date time.Time, ty uint8, data string) bool {
	var i int
	err := db.sqlCountEvents.QueryRow(guild).Scan(&i)

	if !db.CheckError("CountEvents", err) && i < 5000 {
		_, err = db.sqlAddSchedule.Exec(guild, date, ty, data)
		return !db.CheckError("AddSchedule", err)
	}
	return false
}
func (db *BotDB) AddScheduleRepeat(guild uint64, date time.Time, repeatinterval uint8, repeat int, ty uint8, data string) bool {
	var i int
	err := db.sqlCountEvents.QueryRow(guild).Scan(&i)
	if !db.CheckError("CountEvents", err) && i < 5000 {
		_, err := db.sqlAddScheduleRepeat.Exec(guild, date, repeatinterval, repeat, ty, data)
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
	q, err := db.sqlGetSchedule.Query(guild)
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
	e := &ScheduleEvent{}
	err := db.sqlGetEvent.QueryRow(id).Scan(&e.ID, &e.Date, &e.Type, &e.Data)
	if err == sql.ErrNoRows || db.CheckError("GetEvent", err) {
		return nil
	}
	return e
}

func (db *BotDB) GetEvents(guild uint64, maxnum int) []ScheduleEvent {
	q, err := db.sqlGetEvents.Query(guild, maxnum)
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
	q, err := db.sqlGetEventsByType.Query(guild, ty, maxnum)
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
	p := ScheduleEvent{}
	err := db.sqlGetNextEvent.QueryRow(guild, ty).Scan(&p.ID, &p.Date, &p.Type, &p.Data)
	if err == sql.ErrNoRows || db.CheckError("GetNextEvent", err) {
		return ScheduleEvent{0, time.Now().UTC(), 0, ""}
	}
	return p
}

func (db *BotDB) GetReminders(guild uint64, id string, maxnum int) []ScheduleEvent {
	q, err := db.sqlGetReminders.Query(guild, id+"|%", maxnum)
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
	var timestamp time.Time
	err := db.sqlGetUnsilenceDate.QueryRow(guild, id).Scan(&timestamp)
	if err == sql.ErrNoRows || db.CheckError("GetUnsilenceDate", err) {
		return nil
	}
	return &timestamp
}

func evalTimeZone(loc sql.NullString) *time.Location {
	if loc.Valid && len(loc.String) > 0 {
		l, err := time.LoadLocation(loc.String)
		if err == nil {
			return l
		}
	}
	return nil
}

func (db *BotDB) GetTimeZone(user uint64) *time.Location {
	var loc sql.NullString
	err := db.sqlGetTimeZone.QueryRow(user).Scan(&loc)
	if db.CheckError("GetTimeZone", err) {
		return nil
	}
	return evalTimeZone(loc)
}

func (db *BotDB) FindTimeZone(s string) []string {
	q, err := db.sqlFindTimeZone.Query(s)
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
	q, err := db.sqlFindTimeZoneOffset.Query(s, minutes, minutes)
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
	_, err := db.sqlSetTimeZone.Exec(tz.String(), user)
	db.CheckError("SetTimeZone", err)
	return err
}

func (db *BotDB) RemoveAlias(user uint64, alias string) {
	_, err := db.sqlRemoveAlias.Exec(user, alias)
	db.CheckError("RemoveAlias", err)
}

func (db *BotDB) GetUserGuilds(user uint64) []uint64 {
	q, err := db.sqlGetUserGuilds.Query(user)
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
	var id uint64
	err := db.sqlFindEvent.QueryRow(ty, user, guild).Scan(&id)
	if err == sql.ErrNoRows || db.CheckError("FindEvent", err) {
		return nil
	}
	return &id
}

func (db *BotDB) SetDefaultServer(user uint64, server uint64) error {
	_, err := db.sqlSetDefaultServer.Exec(server, user)
	db.CheckError("SetDefaultServer", err)
	return err
}

func (db *BotDB) GetPolls(server uint64) []struct {
	name        string
	description string
} {
	q, err := db.sqlGetPolls.Query(server)
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
	var id uint64
	var desc string
	err := db.sqlGetPoll.QueryRow(name, server).Scan(&id, &desc)
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
	q, err := db.sqlGetOptions.Query(poll)
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
	var id uint64
	err := db.sqlGetOption.QueryRow(poll, option).Scan(&id)
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
	q, err := db.sqlGetResults.Query(poll)
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
	_, err := db.sqlAddPoll.Exec(name, description, server)
	db.CheckError("AddPoll", err)
	return err
}

func (db *BotDB) AddOption(poll uint64, index uint64, option string) error {
	_, err := db.sqlAddOption.Exec(poll, index, option)
	db.CheckError("AddOption", err)
	return err
}

func (db *BotDB) AppendOption(poll uint64, option string) error {
	_, err := db.sqlAppendOption.Exec(option, poll)
	db.CheckError("AppendOption", err)
	return err
}

func (db *BotDB) AddVote(user uint64, poll uint64, option uint64) error {
	_, err := db.sqlAddVote.Exec(poll, user, option, option)
	db.CheckError("AddVote", err)
	return err
}

func (db *BotDB) RemovePoll(name string, server uint64) error {
	_, err := db.sqlRemovePoll.Exec(name, server)
	db.CheckError("RemovePoll", err)
	return err
}

func (db *BotDB) CheckOption(poll uint64, option uint64) bool {
	var name string
	err := db.sqlCheckOption.QueryRow(poll, option).Scan(&name)
	if err == sql.ErrNoRows || db.CheckError("CheckOption", err) {
		return false
	}
	return true
}

func (db *BotDB) SentMessage(user uint64, guild uint64) error {
	_, err := db.sqlSentMessage.Exec(user, guild)
	db.CheckError("SentMessage", err)
	return err
}

func (db *BotDB) GetNewcomers(lookback int, guild uint64) []uint64 {
	q, err := db.sqlGetNewcomers.Query(guild, lookback)
	if db.CheckError("GetNewcomers", err) {
		return []uint64{}
	}
	defer q.Close()
	r := make([]uint64, 0, 2)
	for q.Next() {
		var id uint64
		if err := q.Scan(&id); err == nil {
			r = append(r, id)
		}
	}
	return r
}
