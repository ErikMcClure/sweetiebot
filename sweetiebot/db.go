package sweetiebot

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blackhole12/discordgo"
	"github.com/go-sql-driver/mysql" // Blank import is the correct way to import a sql driver
)

// ErrDuplicateEntry - Error 1062: Duplicate entry for unique key
var ErrDuplicateEntry = errors.New("Error 1062: Duplicate entry for unique key")

// ErrLockWaitTimeout - Error 1205: Lock wait timeout exceeded
var ErrLockWaitTimeout = errors.New("Error 1205: Lock wait timeout exceeded")

// BotDB contains the database connection and all database Prepared statements exposed as functions
type BotDB struct {
	db                        *sql.DB
	Status                    AtomicBool
	lastattempt               time.Time
	log                       logger
	driver                    string
	conn                      string
	statuslock                AtomicFlag
	sqlAddMessage             *sql.Stmt
	sqlAddUser                *sql.Stmt
	sqlAddMember              *sql.Stmt
	sqlSawUser                *sql.Stmt
	sqlSetUserAlias           *sql.Stmt
	sqlRemoveMember           *sql.Stmt
	sqlGetUser                *sql.Stmt
	sqlGetMember              *sql.Stmt
	sqlFindGuildUsers         *sql.Stmt
	sqlFindUser               *sql.Stmt
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
	sqlDeleteSchedule         *sql.Stmt
	sqlCountEvents            *sql.Stmt
	sqlGetEvent               *sql.Stmt
	sqlGetEvents              *sql.Stmt
	sqlGetEventsByType        *sql.Stmt
	sqlGetNextEvent           *sql.Stmt
	sqlGetReminders           *sql.Stmt
	sqlGetScheduleDate        *sql.Stmt
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
	sqlAddItem                *sql.Stmt
	sqlGetItem                *sql.Stmt
	sqlRemoveItem             *sql.Stmt
	sqlAddTag                 *sql.Stmt
	sqlRemoveTag              *sql.Stmt
	sqlCreateTag              *sql.Stmt
	sqlDeleteTag              *sql.Stmt
	sqlGetTag                 *sql.Stmt
	sqlCountTag               *sql.Stmt
	sqlCountItems             *sql.Stmt
	sqlGetItemTags            *sql.Stmt
	sqlGetTags                *sql.Stmt
	sqlImportTag              *sql.Stmt
	sqlRemoveGuild            *sql.Stmt
}

func dbLoad(log logger, driver string, conn string) (*BotDB, error) {
	cdb, err := sql.Open(driver, conn)
	r := BotDB{
		db:          cdb,
		lastattempt: time.Now().UTC(),
		log:         log,
		driver:      driver,
		conn:        conn,
	}
	r.Status.Set(err == nil)
	if err != nil {
		return &r, err
	}

	r.db.SetMaxOpenConns(70)
	err = r.db.Ping()
	r.Status.Set(err == nil)
	return &r, err
}

// Close destroys the database connection
func (db *BotDB) Close() {
	if db.db != nil {
		db.db.Close()
		db.db = nil
	}
}

func (db *BotDB) standardErr(err error) error {
	if err == nil {
		return nil
	}
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		switch mysqlErr.Number {
		case 1062:
			return ErrDuplicateEntry
		case 1205:
			return ErrLockWaitTimeout
		}
	}
	return err
}

// Prepare a sql statement and logs an error if it fails
func (db *BotDB) Prepare(s string) (*sql.Stmt, error) {
	statement, err := db.db.Prepare(s)
	if err != nil {
		fmt.Println("Preparing: ", s, "\nSQL Error: ", err.Error())
	}
	return statement, err
}

// DBReconnectTimeout is the reconnect time interval in seconds
const DBReconnectTimeout = time.Duration(30) * time.Second

// CheckStatus checks if the database connection has been lost
func (db *BotDB) CheckStatus() bool {
	if !db.Status.Get() {
		if db.statuslock.TestAndSet() { // If this was already true, bail out
			return false
		}
		defer db.statuslock.Clear()

		if db.Status.Get() { // If the database was already fixed, return true
			return true
		}

		if db.lastattempt.Add(DBReconnectTimeout).Before(time.Now().UTC()) {
			db.log.Log("Database failure detected! Attempting to reboot database connection...")
			db.lastattempt = time.Now().UTC()
			err := db.db.Ping()
			if err != nil {
				db.log.LogError("Reconnection failed! Another attempt will be made in "+TimeDiff(DBReconnectTimeout)+". Error: ", err)
				return false
			}
			err = db.LoadStatements()                       // If we re-establish connection, we must reload statements in case they were lost or never loaded in the first place
			db.log.LogError("LoadStatements failed: ", err) // if loading the statements fails we're screwed anyway so we just log the error and keep going
			db.Status.Set(true)                             // Only after loading the statements do we set status to true
			db.log.Log("Reconnection succeeded, exiting out of No Database mode.")
		} else { // If not, just fail
			return false
		}
	}

	return true
}

// LoadStatements loads all Prepared statements
func (db *BotDB) LoadStatements() error {
	var err error
	db.sqlAddMessage, err = db.Prepare("CALL AddChat(?,?,?,?,?,?)")
	db.sqlAddUser, err = db.Prepare("CALL AddUser(?,?,?,?,?)")
	db.sqlAddMember, err = db.Prepare("CALL AddMember(?,?,?,?)")
	db.sqlSawUser, err = db.Prepare("UPDATE users SET LastSeen = UTC_TIMESTAMP() WHERE ID = ?")
	db.sqlSetUserAlias, err = db.Prepare("INSERT IGNORE INTO aliases (`User`, Alias, Duration, `Timestamp`)	VALUES (?, ?, 0, UTC_TIMESTAMP())")
	db.sqlRemoveMember, err = db.Prepare("DELETE FROM `members` WHERE Guild = ? AND ID = ?")
	db.sqlGetUser, err = db.Prepare("SELECT ID, Username, Discriminator, Avatar, LastSeen, Location, DefaultServer FROM users WHERE ID = ?")
	db.sqlGetMember, err = db.Prepare("SELECT U.ID, U.Username, U.Discriminator, U.Avatar, U.LastSeen, M.Nickname, M.FirstSeen, M.FirstMessage FROM members M RIGHT OUTER JOIN users U ON U.ID = M.ID WHERE M.ID = ? AND M.Guild = ?")
	db.sqlFindGuildUsers, err = db.Prepare("SELECT DISTINCT M.ID FROM members M LEFT OUTER JOIN aliases A ON A.User = M.ID WHERE M.Guild = ? AND (M.Nickname LIKE ? OR A.Alias LIKE ?) LIMIT ? OFFSET ?")
	db.sqlFindUser, err = db.Prepare("SELECT DISTINCT U.ID FROM users U WHERE U.Discriminator = ? and U.Username LIKE ? LIMIT ? OFFSET ?")
	db.sqlGetNewestUsers, err = db.Prepare("SELECT U.ID, U.Username, U.Avatar, M.FirstSeen FROM members M INNER JOIN users U ON M.ID = U.ID WHERE M.Guild = ? ORDER BY M.FirstSeen DESC LIMIT ?")
	db.sqlGetRecentUsers, err = db.Prepare("SELECT U.ID, U.Username, U.Avatar FROM members M INNER JOIN users U ON M.ID = U.ID WHERE M.Guild = ? AND M.FirstSeen > ? ORDER BY M.FirstSeen DESC")
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
	db.sqlGetTableCounts, err = db.Prepare("SELECT CONCAT('Chatlog: ', (SELECT COUNT(*) FROM chatlog), ' rows', '\nEditlog: ', (SELECT COUNT(*) FROM editlog), ' rows',  '\nAliases: ', (SELECT COUNT(*) FROM aliases), ' rows',  '\nDebuglog: ', (SELECT COUNT(*) FROM debuglog), ' rows',  '\nUsers: ', (SELECT COUNT(*) FROM users), ' rows',  '\nSchedule: ', (SELECT COUNT(*) FROM schedule), ' rows \nMembers: ', (SELECT COUNT(*) FROM members), ' rows \nPolls: ', (SELECT COUNT(*) FROM polls), ' rows \nItems: ', (SELECT COUNT(*) FROM items), ' rows \nTags: ', (SELECT COUNT(*) FROM tags), ' rows \nitemtags: ', (SELECT COUNT(*) FROM itemtags), ' rows');")
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
	db.sqlDeleteSchedule, err = db.Prepare("DELETE FROM `schedule` WHERE ID = ?")
	db.sqlCountEvents, err = db.Prepare("SELECT COUNT(*) FROM schedule WHERE Guild = ?")
	db.sqlGetEvent, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND ID = ?")
	db.sqlGetEvents, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type != 0 AND Type != 4 AND Type != 6 ORDER BY Date ASC LIMIT ?")
	db.sqlGetEventsByType, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = ? ORDER BY Date ASC LIMIT ?")
	db.sqlGetNextEvent, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = ? ORDER BY Date ASC LIMIT 1")
	db.sqlGetReminders, err = db.Prepare("SELECT ID, Date, Type, Data FROM schedule WHERE Guild = ? AND Type = 6 AND Data LIKE ? ORDER BY Date ASC LIMIT ?")
	db.sqlGetScheduleDate, err = db.Prepare("SELECT Date FROM schedule WHERE Guild = ? AND Type = ? AND Data = ?")
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
	db.sqlAddItem, err = db.Prepare("SELECT AddItem(?)")
	db.sqlGetItem, err = db.Prepare("SELECT ID FROM items WHERE Content = ?")
	db.sqlRemoveItem, err = db.Prepare("DELETE M FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID WHERE M.Item = ? AND T.Guild = ?")
	db.sqlAddTag, err = db.Prepare("INSERT INTO itemtags (Item, Tag) VALUES (?, ?)")
	db.sqlRemoveTag, err = db.Prepare("DELETE FROM itemtags WHERE Item = ? AND Tag = ?")
	db.sqlCreateTag, err = db.Prepare("INSERT INTO tags (Name, Guild) VALUES (?, ?)")
	db.sqlDeleteTag, err = db.Prepare("DELETE FROM tags WHERE Name = ? AND Guild = ?")
	db.sqlGetTag, err = db.Prepare("SELECT ID FROM tags WHERE Name = ? AND Guild = ?")
	db.sqlCountTag, err = db.Prepare("SELECT COUNT(*) FROM itemtags WHERE Tag = ?")
	db.sqlCountItems, err = db.Prepare("SELECT COUNT(DISTINCT M.Item) FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID WHERE T.Guild = ?")
	db.sqlGetItemTags, err = db.Prepare("SELECT T.Name FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID WHERE M.Item = ? AND T.Guild = ?")
	db.sqlGetTags, err = db.Prepare("SELECT T.Name, COUNT(M.Item) FROM tags T LEFT OUTER JOIN itemtags M ON T.ID = M.Tag WHERE T.Guild = ? GROUP BY T.Name")
	db.sqlImportTag, err = db.Prepare("INSERT IGNORE INTO itemtags (Item, Tag) SELECT Item, ? FROM itemtags WHERE Tag = ?")
	db.sqlRemoveGuild, err = db.Prepare("CALL RemoveGuild(?)")
	return err
}

// Audit types
const (
	AuditTypeLog     = iota
	AuditTypeAction  = iota
	AuditTypeCommand = iota
)

func (db *BotDB) parseStringResults(q *sql.Rows) []string {
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

// CheckError logs any unknown errors and pings the database to check if it's still there
func (db *BotDB) CheckError(name string, err error) error {
	if err != nil && err != sql.ErrNoRows && err != sql.ErrTxDone && err != ErrDuplicateEntry {
		if db.Status.Get() {
			db.log.LogError(name+" error: ", err)
		}
		if db.db.Ping() != nil {
			db.Status.Set(false)
		}
	}
	return err
}

// AddMessage logs a message to the chatlog
func (db *BotDB) AddMessage(id uint64, author *discordgo.User, message string, channel uint64, guild uint64) {
	_, err := db.sqlAddMessage.Exec(id, SBatoi(author.ID), author.Username, message, channel, guild)
	db.CheckError("AddMessage", err)
}

// PingContext contains a simplified context for a message
type PingContext struct {
	Author    string
	Message   string
	Timestamp time.Time
}

// AddUser adds or updates user information
func (db *BotDB) AddUser(id uint64, username string, discriminator int, avatar string, isonline bool) {
	_, err := db.sqlAddUser.Exec(id, username, discriminator, avatar, isonline)
	db.CheckError("AddUser", err)
}

// AddMember adds or updates guild-specific user information
func (db *BotDB) AddMember(id uint64, guild uint64, firstseen time.Time, nickname string) {
	_, err := db.sqlAddMember.Exec(id, guild, firstseen, nickname)
	db.CheckError("AddMember", err)
}

// SawUser updates a user's lastseen time
func (db *BotDB) SawUser(user uint64, username string) error {
	_, err := db.sqlSawUser.Exec(user)
	err = db.standardErr(err)
	if err != nil || len(username) == 0 {
		return db.CheckError("SawUser", err)
	}
	_, err = db.sqlSetUserAlias.Exec(user, username) // Ensures the username is always in the alias table even if we miss it somehow
	err = db.standardErr(err)
	return db.CheckError("SetUserAlias", err)
}

// RemoveMember removes a user from a guild
func (db *BotDB) RemoveMember(id uint64, guild uint64) error {
	_, err := db.sqlRemoveMember.Exec(guild, id)
	err = db.standardErr(err)
	return db.CheckError("RemoveMember", err)
}

// GetUser gets the guild-independent information about a user (if it exists)
func (db *BotDB) GetUser(id uint64) (*discordgo.User, time.Time, *time.Location, *uint64) {
	u := &discordgo.User{}
	var lastseen time.Time
	var loc sql.NullString
	var guild sql.NullInt64
	var discriminator int
	err := db.sqlGetUser.QueryRow(id).Scan(&u.ID, &u.Username, &discriminator, &u.Avatar, &lastseen, &loc, &guild)
	if discriminator > 0 {
		u.Discriminator = strconv.Itoa(discriminator)
	}
	if err == sql.ErrNoRows || db.CheckError("GetUser", err) != nil {
		return nil, lastseen, nil, nil
	}
	if !guild.Valid {
		return u, lastseen, evalTimeZone(loc), nil
	}
	g := uint64(guild.Int64)
	return u, lastseen, evalTimeZone(loc), &g
}

// GetMember gets all information about a user for the given guild
func (db *BotDB) GetMember(id uint64, guild uint64) (*discordgo.Member, time.Time, *time.Time) {
	m := &discordgo.Member{}
	m.User = &discordgo.User{}
	var lastseen time.Time
	var firstmessage *time.Time
	var joinedat time.Time
	var discriminator int
	err := db.sqlGetMember.QueryRow(id, guild).Scan(&m.User.ID, &m.User.Username, &discriminator, &m.User.Avatar, &lastseen, &m.Nick, &joinedat, &firstmessage)
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

// FindGuildUsers returns all users in a guild that could satisfy the given name.
func (db *BotDB) FindGuildUsers(name string, maxresults uint64, offset uint64, guild uint64) []uint64 {
	q, err := db.sqlFindGuildUsers.Query(guild, name, name, maxresults, offset)
	if db.CheckError("FindGuildUsers", err) != nil {
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

// FindUser returns all users with the given name and discriminator (which should be only one but cache errors can happen)
func (db *BotDB) FindUser(name string, discriminator int, maxresults uint64, offset uint64) []uint64 {
	q, err := db.sqlFindUser.Query(discriminator, name, maxresults, offset)
	if db.CheckError("FindUser", err) != nil {
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

// GetNewestUsers gets the last maxresults users to join the guild
func (db *BotDB) GetNewestUsers(maxresults int, guild uint64) []struct {
	User      *discordgo.User
	FirstSeen time.Time
} {
	q, err := db.sqlGetNewestUsers.Query(guild, maxresults)
	if db.CheckError("GetNewestUsers", err) != nil {
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
		}{&discordgo.User{}, time.Now().UTC()}
		if err := q.Scan(&p.User.ID, &p.User.Username, &p.User.Avatar, &p.FirstSeen); err == nil {
			r = append(r, p)
		}
	}
	return r
}

// GetRecentUsers returns any users whose first message was sent after the given timestamp
func (db *BotDB) GetRecentUsers(since time.Time, guild uint64) []*discordgo.User {
	q, err := db.sqlGetRecentUsers.Query(guild, since)
	if db.CheckError("GetRecentUsers", err) != nil {
		return []*discordgo.User{}
	}
	defer q.Close()
	r := make([]*discordgo.User, 0, 2)
	for q.Next() {
		p := &discordgo.User{}
		if err := q.Scan(&p.ID, &p.Username, &p.Avatar); err == nil {
			r = append(r, p)
		}
	}
	return r
}

// GetAliases returns all aliases for the given user
func (db *BotDB) GetAliases(user uint64) []string {
	q, err := db.sqlGetAliases.Query(user)
	if db.CheckError("GetAliases", err) != nil {
		return []string{}
	}
	defer q.Close()
	return db.parseStringResults(q)
}

// Audit logs an action to the audit log
func (db *BotDB) Audit(ty uint8, user *discordgo.User, message string, guild uint64) {
	var err error
	if user == nil {
		_, err = db.sqlAudit.Exec(ty, nil, message, guild)
	} else {
		_, err = db.sqlAudit.Exec(ty, SBatoi(user.ID), message, guild)
	}

	if err != nil && db.Status.Get() {
		fmt.Println("Logger failed to log to database! ", err.Error())
	}
}

// GetAuditRows returns rows from the audit log
func (db *BotDB) GetAuditRows(start uint64, end uint64, user *uint64, search string, guild uint64) []PingContext {
	var q *sql.Rows
	var err error
	maxresults := end - start
	if maxresults > 50 {
		maxresults = 50
	}

	if user != nil && len(search) > 0 {
		q, err = db.sqlGetAuditRowsUserString.Query(AuditTypeCommand, guild, *user, search, maxresults, start)
	} else if user != nil && len(search) == 0 {
		q, err = db.sqlGetAuditRowsUser.Query(AuditTypeCommand, guild, *user, maxresults, start)
	} else if user == nil && len(search) > 0 {
		q, err = db.sqlGetAuditRowsString.Query(AuditTypeCommand, guild, search, maxresults, start)
	} else {
		q, err = db.sqlGetAuditRows.Query(AuditTypeCommand, guild, maxresults, start)
	}
	if db.CheckError("GetAuditRows", err) != nil {
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

// GetTableCounts returns a debug dump count of the tables
func (db *BotDB) GetTableCounts() string {
	if !db.Status.Get() {
		return "DATABASE ERROR"
	}
	var counts string
	err := db.sqlGetTableCounts.QueryRow().Scan(&counts)
	if db.CheckError("GetTableCounts", err) != nil {
		return err.Error()
	}
	return counts
}

// AddTranscript is used to construct the markov chain
func (db *BotDB) AddTranscript(season int, episode int, line int, speaker string, text string) {
	_, err := db.sqlAddTranscript.Exec(season, episode, line, speaker, text)
	if err != nil {
		db.log.Log("AddTranscript error: ", err.Error(), "\nS", season, "E", episode, ":", line, " ", speaker, ": ", text)
	}
}

// Transcript describes a single line from the transcript
type Transcript struct {
	Season  uint
	Episode uint
	Line    uint
	Speaker string
	Text    string
}

// GetTranscript gets all lines that satisfy the query from the transcripts
func (db *BotDB) GetTranscript(season int, episode int, start int, end int) []Transcript {
	q, err := db.sqlGetTranscript.Query(season, episode, start, end)
	if db.CheckError("GetTranscript", err) != nil {
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

// RemoveTranscript removes a line from the transcripts
func (db *BotDB) RemoveTranscript(season int, episode int, line int) {
	_, err := db.sqlRemoveTranscript.Exec(season, episode, line)
	db.CheckError("RemoveTranscript", err)
}

// AddMarkov adds a line to the markov chain
func (db *BotDB) AddMarkov(last uint64, last2 uint64, speaker string, text string) uint64 {
	var id uint64
	err := db.sqlAddMarkov.QueryRow(last, last2, speaker, text).Scan(&id)
	db.CheckError("AddMarkov", err)
	return id
}

// GetMarkovLine generates a line from the markov chain
func (db *BotDB) GetMarkovLine(last uint64) (string, uint64) {
	var r sql.NullString
	err := db.sqlGetMarkovLine.QueryRow(last).Scan(&r)
	if db.CheckError("GetMarkovLine", err) != nil || !r.Valid {
		return "", 0
	}
	str := strings.SplitN(r.String, "|", 2) // Being unable to call stored procedures makes this unnecessarily complex
	if len(str) < 2 || len(str[1]) < 1 {
		return str[0], 0
	}
	return str[0], SBatoi(str[1])
}

// GetMarkovLine2 generates a line from the markov chain
func (db *BotDB) GetMarkovLine2(last uint64, last2 uint64) (string, uint64, uint64) {
	var r sql.NullString
	err := db.sqlGetMarkovLine2.QueryRow(last, last2).Scan(&r)
	if db.CheckError("GetMarkovLine2", err) != nil || !r.Valid {
		return "", 0, 0
	}
	str := strings.SplitN(r.String, "|", 3) // Being unable to call stored procedures makes this unnecessarily complex
	if len(str) < 3 || len(str[1]) < 1 || len(str[2]) < 1 {
		return str[0], 0, 0
	}
	return str[0], SBatoi(str[1]), SBatoi(str[2])
}

// GetMarkovWord generates a word from the markov chain
func (db *BotDB) GetMarkovWord(speaker string, phrase string) string {
	var r string
	err := db.sqlGetMarkovWord.QueryRow(speaker, phrase).Scan(&r)
	if err == sql.ErrNoRows {
		return phrase
	}
	db.CheckError("GetMarkovWord", err)
	return r
}

// GetRandomQuote gets a random quote from the transcript
func (db *BotDB) GetRandomQuote() Transcript {
	var i uint64
	err := db.sqlGetRandomQuoteInt.QueryRow().Scan(&i)
	var p Transcript
	if db.CheckError("GetRandomQuoteInt", err) == nil {
		err = db.sqlGetRandomQuote.QueryRow(i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
		db.CheckError("GetRandomQuote", err)
	}
	return p
}

// GetSpeechQuote gets a random speech quote from the transcript
func (db *BotDB) GetSpeechQuote() Transcript {
	var i uint64
	err := db.sqlGetSpeechQuoteInt.QueryRow().Scan(&i)
	var p Transcript
	if db.CheckError("GetSpeechQuoteInt", err) == nil {
		err = db.sqlGetSpeechQuote.QueryRow(i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
		db.CheckError("GetSpeechQuote", err)
	}
	return p
}

// GetCharacterQuote gets a random character quote from the transcript
func (db *BotDB) GetCharacterQuote(character string) Transcript {
	var i uint64
	err := db.sqlGetCharacterQuoteInt.QueryRow(character).Scan(&i)
	var p Transcript
	if db.CheckError("GetCharacterQuoteInt ", err) == nil {
		err = db.sqlGetCharacterQuote.QueryRow(character, i).Scan(&p.Season, &p.Episode, &p.Line, &p.Speaker, &p.Text)
		if err == sql.ErrNoRows || db.CheckError("GetCharacterQuote ", err) != nil {
			return Transcript{0, 0, 0, "", ""}
		}
	}
	return p
}

// GetRandomSpeaker gets a random speaker from the transcript
func (db *BotDB) GetRandomSpeaker() string {
	var i uint64
	err := db.sqlGetRandomSpeakerInt.QueryRow().Scan(&i)
	var p string
	if db.CheckError("GetRandomSpeakerInt", err) == nil {
		err = db.sqlGetRandomSpeaker.QueryRow(i).Scan(&p)
		db.CheckError("GetRandomSpeaker", err)
	}
	return p
}

// GetRandomMember gets a random user from the guild
func (db *BotDB) GetRandomMember(guild uint64) string {
	var i uint64
	err := db.sqlGetRandomMemberInt.QueryRow(guild).Scan(&i)
	var p string
	if db.CheckError("GetRandomMemberInt", err) == nil {
		err = db.sqlGetRandomMember.QueryRow(guild, i).Scan(&p)
		db.CheckError("GetRandomMember", err)
	}
	return p
}

// GetRandomWord gets a random word from the markov chain
func (db *BotDB) GetRandomWord() string {
	var i uint64
	err := db.sqlGetRandomWordInt.QueryRow().Scan(&i)
	var p string
	if db.CheckError("GetRandomWordInt", err) == nil {
		err = db.sqlGetRandomWord.QueryRow(i).Scan(&p)
		db.CheckError("GetRandomWord", err)
	}
	return p
}

// CountNewUsers returns a count of users joined in the given duration
func (db *BotDB) CountNewUsers(seconds int64, guild uint64) int {
	var i int
	err := db.sqlCountNewUsers.QueryRow(seconds, guild).Scan(&i)
	db.CheckError("CountNewUsers", err)
	return i
}

// RemoveSchedule removes the event with the given ID and creates a new one after the repeat interval
func (db *BotDB) RemoveSchedule(id uint64) error {
	_, err := db.sqlRemoveSchedule.Exec(id)
	return db.CheckError("RemoveSchedule", err)
}

// DeleteSchedule deletes the event with the given ID regardless of it's repeat interval or activation time.
func (db *BotDB) DeleteSchedule(id uint64) error {
	_, err := db.sqlDeleteSchedule.Exec(id)
	return db.CheckError("DeleteSchedule", err)
}

// AddSchedule adds an event to the schedule
func (db *BotDB) AddSchedule(guild uint64, date time.Time, ty uint8, data string) error {
	var i int
	err := db.sqlCountEvents.QueryRow(guild).Scan(&i)

	if db.CheckError("CountEvents", err) == nil {
		if i >= MaxScheduleRows {
			return fmt.Errorf("Can't have more than %v events!", MaxScheduleRows)
		}
		_, err = db.sqlAddSchedule.Exec(guild, date, ty, data)
		return db.CheckError("AddSchedule", err)
	}
	return err
}

// AddScheduleRepeat adds a repeating event to the schedule
func (db *BotDB) AddScheduleRepeat(guild uint64, date time.Time, repeatinterval uint8, repeat int, ty uint8, data string) error {
	var i int
	err := db.sqlCountEvents.QueryRow(guild).Scan(&i)
	if db.CheckError("CountEvents", err) == nil {
		if i >= MaxScheduleRows {
			return fmt.Errorf("Can't have more than %v events!", MaxScheduleRows)
		}
		_, err := db.sqlAddScheduleRepeat.Exec(guild, date, repeatinterval, repeat, ty, data)
		return db.CheckError("AddScheduleRepeat", err)
	}
	return err
}

// ScheduleEvent describes an event in the schedule
type ScheduleEvent struct {
	ID   uint64
	Date time.Time
	Type uint8
	Data string
}

// GetSchedule gets all events for a guild
func (db *BotDB) GetSchedule(guild uint64) []ScheduleEvent {
	q, err := db.sqlGetSchedule.Query(guild)
	if db.CheckError("GetSchedule", err) != nil {
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

// GetEvent gets the event data for the given ID
func (db *BotDB) GetEvent(guild uint64, id uint64) *ScheduleEvent {
	e := &ScheduleEvent{}
	err := db.sqlGetEvent.QueryRow(guild, id).Scan(&e.ID, &e.Date, &e.Type, &e.Data)
	if err == sql.ErrNoRows || db.CheckError("GetEvent", err) != nil {
		return nil
	}
	return e
}

// GetEvents gets all events for a guild up to maxnum
func (db *BotDB) GetEvents(guild uint64, maxnum int) []ScheduleEvent {
	q, err := db.sqlGetEvents.Query(guild, maxnum)
	if db.CheckError("GetEvents", err) != nil {
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

// GetEventsByType gets all events for a given type
func (db *BotDB) GetEventsByType(guild uint64, ty uint8, maxnum int) []ScheduleEvent {
	q, err := db.sqlGetEventsByType.Query(guild, ty, maxnum)
	if db.CheckError("GetEventsByType", err) != nil {
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

// GetNextEvent gets the next event of the given type
func (db *BotDB) GetNextEvent(guild uint64, ty uint8) ScheduleEvent {
	p := ScheduleEvent{}
	err := db.sqlGetNextEvent.QueryRow(guild, ty).Scan(&p.ID, &p.Date, &p.Type, &p.Data)
	if err == sql.ErrNoRows || db.CheckError("GetNextEvent", err) != nil {
		return ScheduleEvent{0, time.Now().UTC(), 0, ""}
	}
	return p
}

// GetReminders gets reminders for the given user
func (db *BotDB) GetReminders(guild uint64, id string, maxnum int) []ScheduleEvent {
	q, err := db.sqlGetReminders.Query(guild, id+"|%", maxnum)
	if db.CheckError("GetReminders", err) != nil {
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

// GetScheduleDate returns the date for the given event if it exists
func (db *BotDB) GetScheduleDate(guild uint64, ty uint8, data string) *time.Time {
	var timestamp time.Time
	err := db.sqlGetScheduleDate.QueryRow(guild, ty, data).Scan(&timestamp)
	if err == sql.ErrNoRows || db.CheckError("GetScheduleDate", err) != nil {
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

// GetTimeZone returns the evaluated timezone for the user
func (db *BotDB) GetTimeZone(user uint64) *time.Location {
	var loc sql.NullString
	err := db.sqlGetTimeZone.QueryRow(user).Scan(&loc)
	if db.CheckError("GetTimeZone", err) != nil {
		return nil
	}
	return evalTimeZone(loc)
}

// FindTimeZone returns all matching timezone locations
func (db *BotDB) FindTimeZone(s string) []string {
	q, err := db.sqlFindTimeZone.Query(s)
	if db.CheckError("FindTimeZone", err) != nil {
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

// FindTimeZoneOffset finds all timezones with the given offset
func (db *BotDB) FindTimeZoneOffset(s string, minutes int) []string {
	q, err := db.sqlFindTimeZoneOffset.Query(s, minutes, minutes)
	if db.CheckError("FindTimeZoneOffset", err) != nil {
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

// SetTimeZone sets a users timezone location
func (db *BotDB) SetTimeZone(user uint64, tz *time.Location) error {
	_, err := db.sqlSetTimeZone.Exec(tz.String(), user)
	err = db.standardErr(err)
	return db.CheckError("SetTimeZone", err)
}

// RemoveAlias removes an alias from a user, if it exists.
func (db *BotDB) RemoveAlias(user uint64, alias string) error {
	_, err := db.sqlRemoveAlias.Exec(user, alias)
	err = db.standardErr(err)
	return db.CheckError("RemoveAlias", err)
}

// GetUserGuilds returns all guilds a user is on
func (db *BotDB) GetUserGuilds(user uint64) []uint64 {
	q, err := db.sqlGetUserGuilds.Query(user)
	if db.CheckError("GetUserGuilds", err) != nil {
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

// FindEvent finds an event in the schedule
func (db *BotDB) FindEvent(user string, guild uint64, ty uint8) *uint64 {
	var id uint64
	err := db.sqlFindEvent.QueryRow(ty, user, guild).Scan(&id)
	if err == sql.ErrNoRows || db.CheckError("FindEvent", err) != nil {
		return nil
	}
	return &id
}

// SetDefaultServer sets a users default guild
func (db *BotDB) SetDefaultServer(userID uint64, guild uint64) error {
	_, err := db.sqlSetDefaultServer.Exec(guild, userID)
	return db.CheckError("SetDefaultServer", err)
}

// PollData contains the poll name and description
type PollData struct {
	Name        string
	Description string
}

// GetPolls returns all polls for a given guild
func (db *BotDB) GetPolls(guild uint64) []PollData {
	q, err := db.sqlGetPolls.Query(guild)
	if db.CheckError("GetPolls", err) != nil {
		return []PollData{}
	}
	defer q.Close()
	r := make([]PollData, 0, 2)
	for q.Next() {
		var s PollData
		if err := q.Scan(&s.Name, &s.Description); err == nil {
			r = append(r, s)
		}
	}
	return r
}

// GetPoll gets the poll ID with the given name
func (db *BotDB) GetPoll(name string, guild uint64) (uint64, string) {
	var id uint64
	var desc string
	err := db.sqlGetPoll.QueryRow(name, guild).Scan(&id, &desc)
	if err == sql.ErrNoRows || db.CheckError("GetPoll", err) != nil {
		return 0, ""
	}
	return id, desc
}

// PollOptionStruct contains the option index and text
type PollOptionStruct struct {
	Index  uint64
	Option string
}

// GetOptions returns all options for a given poll ID
func (db *BotDB) GetOptions(poll uint64) []PollOptionStruct {
	q, err := db.sqlGetOptions.Query(poll)
	if db.CheckError("GetOptions", err) != nil {
		return []PollOptionStruct{}
	}
	defer q.Close()
	r := make([]PollOptionStruct, 0, 2)
	for q.Next() {
		var s PollOptionStruct
		if err := q.Scan(&s.Index, &s.Option); err == nil {
			r = append(r, s)
		}
	}
	return r
}

// GetOption returns any option for a given poll ID that contains a string
func (db *BotDB) GetOption(poll uint64, option string) *uint64 {
	var id uint64
	err := db.sqlGetOption.QueryRow(poll, option).Scan(&id)
	if err == sql.ErrNoRows || db.CheckError("GetOption", err) != nil {
		return nil
	}
	return &id
}

// PollResultStruct is an index+count pair
type PollResultStruct struct {
	Index uint64
	Count uint64
}

// GetResults returns the results of a poll ID
func (db *BotDB) GetResults(poll uint64) []PollResultStruct {
	q, err := db.sqlGetResults.Query(poll)
	if db.CheckError("GetResults", err) != nil {
		return []PollResultStruct{}
	}
	defer q.Close()
	r := make([]PollResultStruct, 0, 2)
	for q.Next() {
		var s PollResultStruct
		if err := q.Scan(&s.Index, &s.Count); err == nil {
			r = append(r, s)
		}
	}
	return r
}

// AddPoll creates a poll
func (db *BotDB) AddPoll(name string, description string, guild uint64) error {
	_, err := db.sqlAddPoll.Exec(name, description, guild)
	return db.standardErr(err)
}

// AddOption adds an option to an existing poll with a specific index
func (db *BotDB) AddOption(poll uint64, index uint64, option string) error {
	_, err := db.sqlAddOption.Exec(poll, index, option)
	return db.standardErr(err)
}

// AppendOption appends an option to the end of an existing poll
func (db *BotDB) AppendOption(poll uint64, option string) error {
	_, err := db.sqlAppendOption.Exec(option, poll)
	return db.standardErr(err)
}

// AddVote adds or reassigns a users vote on a poll
func (db *BotDB) AddVote(user uint64, poll uint64, option uint64) error {
	_, err := db.sqlAddVote.Exec(poll, user, option, option)
	return db.standardErr(err)
}

// RemovePoll deletes a poll
func (db *BotDB) RemovePoll(name string, guild uint64) error {
	_, err := db.sqlRemovePoll.Exec(name, guild)
	return db.standardErr(err)
}

// CheckOption checks if an option exists
func (db *BotDB) CheckOption(poll uint64, option uint64) bool {
	var name string
	err := db.sqlCheckOption.QueryRow(poll, option).Scan(&name)
	if err == sql.ErrNoRows || db.CheckError("CheckOption", err) != nil {
		return false
	}
	return true
}

// SentMessage doesn't log a message, but sets a user's "firstseen" and "lastseen" values if necessary
func (db *BotDB) SentMessage(user uint64, guild uint64) error {
	_, err := db.sqlSentMessage.Exec(user, guild)
	return db.CheckError("SentMessage", db.standardErr(err))
}

// GetNewcomers returns any users that posted recently
func (db *BotDB) GetNewcomers(lookback int, guild uint64) []uint64 {
	q, err := db.sqlGetNewcomers.Query(guild, lookback)
	if db.CheckError("GetNewcomers", err) != nil {
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

// AddItem adds an item or just returns the ID if it already exists.
func (db *BotDB) AddItem(item string) (uint64, error) {
	var id uint64
	err := db.standardErr(db.sqlAddItem.QueryRow(item).Scan(&id))

	if db.CheckError("AddItem", err) != nil {
		return 0, err
	}
	return id, nil
}

// GetItem returns the ID of the item, if it exists
func (db *BotDB) GetItem(item string) (uint64, error) {
	var id uint64
	err := db.sqlGetItem.QueryRow(item).Scan(&id)
	if err == sql.ErrNoRows || db.CheckError("GetItem", err) != nil {
		return 0, err
	}
	return id, nil
}

// RemoveItem removes an item from all tags on the given server
func (db *BotDB) RemoveItem(item uint64, guild uint64) error {
	_, err := db.sqlRemoveItem.Exec(item, guild)
	return db.CheckError("RemoveItem", db.standardErr(err))
}

// AddTag adds a tag to an item
func (db *BotDB) AddTag(item uint64, tag uint64) error {
	_, err := db.sqlAddTag.Exec(item, tag)
	return db.CheckError("AddTag", db.standardErr(err))
}

// RemoveTag removes a tag to an item
func (db *BotDB) RemoveTag(item uint64, tag uint64) error {
	_, err := db.sqlRemoveTag.Exec(item, tag)
	return db.CheckError("RemoveTag", db.standardErr(err))
}

// CreateTag creates a new tag on the given server
func (db *BotDB) CreateTag(tag string, guild uint64) error {
	_, err := db.sqlCreateTag.Exec(tag, guild)
	return db.CheckError("CreateTag", db.standardErr(err))
}

// DeleteTag delets a tag from the given server
func (db *BotDB) DeleteTag(tag string, guild uint64) error {
	_, err := db.sqlDeleteTag.Exec(tag, guild)
	return db.CheckError("DeleteTag", db.standardErr(err))
}

// GetTag gets the tag ID for a tag name on the given server
func (db *BotDB) GetTag(tag string, guild uint64) (uint64, error) {
	var id uint64
	err := db.standardErr(db.sqlGetTag.QueryRow(tag, guild).Scan(&id))
	if err == sql.ErrNoRows || db.CheckError("GetTag", err) != nil {
		return 0, err
	}
	return id, nil
}

// CountTag returns the number of items with the given tag
func (db *BotDB) CountTag(tag uint64) (uint64, error) {
	var n uint64
	err := db.standardErr(db.sqlCountTag.QueryRow(tag).Scan(&n))
	if err == sql.ErrNoRows || db.CheckError("CountTag", err) != nil {
		return 0, err
	}
	return n, nil
}

// CountItems returns the number of unique items for a given server
func (db *BotDB) CountItems(guild uint64) (uint64, error) {
	var n uint64
	err := db.standardErr(db.sqlCountItems.QueryRow(guild).Scan(&n))
	if db.CheckError("CountItems", err) != nil {
		return 0, err
	}
	return n, nil
}

// GetItemTags returns all tags an item has on the given server
func (db *BotDB) GetItemTags(item uint64, guild uint64) []string {
	q, err := db.sqlGetItemTags.Query(item, guild)
	err = db.standardErr(err)
	if db.CheckError("GetItemTags", err) != nil {
		return []string{}
	}
	defer q.Close()
	r := make([]string, 0, 3)
	for q.Next() {
		var p string
		if err := q.Scan(&p); err == nil {
			r = append(r, p)
		}
	}
	return r
}

// GetTags returns all tags on a server
func (db *BotDB) GetTags(guild uint64) []struct {
	Name  string
	Count int
} {
	q, err := db.sqlGetTags.Query(guild)
	err = db.standardErr(err)
	if db.CheckError("GetTags", err) != nil {
		return []struct {
			Name  string
			Count int
		}{}
	}
	defer q.Close()
	r := make([]struct {
		Name  string
		Count int
	}, 0, 10)
	for q.Next() {
		p := struct {
			Name  string
			Count int
		}{}
		if err := q.Scan(&p.Name, &p.Count); err == nil {
			r = append(r, p)
		}
	}
	return r
}

// ImportTag imports a tag from one server to another
func (db *BotDB) ImportTag(srcTag uint64, destTag uint64) error {
	_, err := db.sqlImportTag.Exec(destTag, srcTag)
	err = db.standardErr(err)
	db.CheckError("ImportTag", err)
	return err
}

// RemoveGuild removes the given guild from the database, if it exists
func (db *BotDB) RemoveGuild(guild uint64) error {
	_, err := db.sqlRemoveGuild.Exec(guild)
	err = db.standardErr(err)
	return db.CheckError("RemoveGuild", err)
}
