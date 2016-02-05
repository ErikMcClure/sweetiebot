package sweetiebot

import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/bwmarrin/discordgo"
    "fmt"
    "time"
    "strings"
)

type BotDB struct {
  db *sql.DB
  log Logger
  sql_AddMessage *sql.Stmt
  sql_AddPing *sql.Stmt
  sql_GetPing *sql.Stmt
  sql_GetPingContext *sql.Stmt
  sql_GetPingContextBefore *sql.Stmt
  sql_AddUser *sql.Stmt
  sql_GetUser *sql.Stmt
  sql_GetUserByName *sql.Stmt
  sql_GetRecentMessages *sql.Stmt
  sql_UpdateUserJoinTime *sql.Stmt
  sql_GetNewestUsers *sql.Stmt
  sql_GetAliases *sql.Stmt
  sql_AddTranscript *sql.Stmt
  sql_GetTranscript *sql.Stmt
  sql_RemoveTranscript *sql.Stmt
  sql_AddMarkov *sql.Stmt
  sql_GetMarkovLine *sql.Stmt
  sql_GetMarkovWord *sql.Stmt
  sql_GetRandomQuoteInt *sql.Stmt
  sql_GetRandomQuote *sql.Stmt
  sql_GetSpeechQuoteInt *sql.Stmt
  sql_GetSpeechQuote *sql.Stmt
  sql_GetCharacterQuoteInt *sql.Stmt
  sql_GetCharacterQuote *sql.Stmt
  sql_GetRandomSpeakerInt *sql.Stmt
  sql_GetRandomSpeaker *sql.Stmt
  sql_GetTableCounts *sql.Stmt
  sql_Log *sql.Stmt
}

func DB_Load(log Logger, driver string, conn string) (*BotDB, error) {
  cdb, err := sql.Open(driver, conn)
  r := BotDB{}
  r.db = cdb
  r.log = log
  if(err != nil) { return &r, err }
  
  err = r.db.Ping()
  return &r, err
}

func (db *BotDB) Close() {
  if db.db != nil { db.db.Close() }
}

func (db *BotDB) Prepare(s string) (*sql.Stmt, error) {
  statement, err := db.db.Prepare(s)
  if(err != nil) {
    fmt.Println("Preparing: ", s, "\nSQL Error: ", err.Error())
  }
  return statement, err
}

func (db *BotDB) LoadStatements() error {
  var err error;
  db.sql_AddMessage, err = db.Prepare("CALL AddChat(?,?,?,?,?)");
  db.sql_AddPing, err = db.Prepare("INSERT INTO pings (Message, User) VALUES (?, ?) ON DUPLICATE KEY UPDATE Message = Message");
  db.sql_GetPing, err = db.Prepare("SELECT C.ID, C.Channel FROM pings P RIGHT OUTER JOIN chatlog C ON P.Message = C.ID WHERE P.User = ? OR C.Everyone = 1 ORDER BY Timestamp DESC LIMIT 1 OFFSET ?");
  db.sql_GetPingContext, err  = db.Prepare("SELECT U.Username, C.Message, C.Timestamp FROM chatlog C INNER JOIN users U ON C.Author = U.ID WHERE C.ID >= ? AND C.Channel = ? ORDER BY C.ID ASC LIMIT ?");
  db.sql_GetPingContextBefore, err  = db.Prepare("SELECT U.Username, C.Message, C.Timestamp FROM chatlog C INNER JOIN users U ON C.Author = U.ID WHERE C.ID < ? AND C.Channel = ? ORDER BY C.ID DESC LIMIT ?");
  db.sql_AddUser, err = db.Prepare("CALL AddUser(?,?,?,?,?)");
  db.sql_GetUser, err = db.Prepare("SELECT ID, Email, Username, Avatar, LastSeen FROM users WHERE ID = ?");
  db.sql_GetUserByName, err = db.Prepare("SELECT * FROM users WHERE Username = ?");
  db.sql_GetRecentMessages, err = db.Prepare("SELECT ID, Channel FROM chatlog WHERE Author = ? AND Timestamp >= DATE_SUB(Now(6), INTERVAL ? SECOND)");
  db.sql_UpdateUserJoinTime, err = db.Prepare("CALL UpdateUserJoinTime(?, ?)");
  db.sql_GetNewestUsers, err = db.Prepare("SELECT Username, FirstSeen, LastSeen FROM users ORDER BY FirstSeen DESC LIMIT ?")
  db.sql_GetAliases, err = db.Prepare("SELECT Alias FROM aliases WHERE User = ? ORDER BY Duration DESC LIMIT 10")
  db.sql_AddTranscript, err = db.Prepare("INSERT INTO transcripts (Season, Episode, Line, Speaker, Text) VALUES (?,?,?,?,?)")
  db.sql_GetTranscript, err = db.Prepare("SELECT Season, Episode, Line, Speaker, Text FROM transcripts WHERE Season = ? AND Episode = ? AND Line >= ? AND LINE <= ?")
  db.sql_RemoveTranscript, err = db.Prepare("DELETE FROM transcripts WHERE Season = ? AND Episode = ? AND Line = ?")
  db.sql_AddMarkov, err = db.Prepare("SELECT AddMarkov(?,?,?)")
  db.sql_GetMarkovLine, err = db.Prepare("SELECT GetMarkovLine(?)")
  db.sql_GetMarkovWord, err = db.Prepare("SELECT Phrase FROM markov_transcripts WHERE SpeakerID = (SELECT ID FROM markov_transcripts_speaker WHERE Speaker = ?) AND Phrase = ?")
  db.sql_GetRandomQuoteInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM transcripts WHERE Text != ''))")
  db.sql_GetRandomQuote, err = db.Prepare("SELECT * FROM transcripts WHERE Text != '' LIMIT 1 OFFSET ?")
  db.sql_GetSpeechQuoteInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM transcripts WHERE Speaker != 'ACTION' AND Text != ''))")
  db.sql_GetSpeechQuote, err = db.Prepare("SELECT * FROM transcripts WHERE Speaker != 'ACTION' AND Text != '' LIMIT 1 OFFSET ?")
  db.sql_GetCharacterQuoteInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM transcripts WHERE Speaker = ? AND Text != ''))")
  db.sql_GetCharacterQuote, err = db.Prepare("SELECT * FROM transcripts WHERE Speaker = ? AND Text != '' LIMIT 1 OFFSET ?")
  db.sql_GetRandomSpeakerInt, err = db.Prepare("SELECT FLOOR(RAND()*(SELECT COUNT(*) FROM markov_transcripts_speaker))+1") // We add one here because the speaker IDs start at 1 instead of 0 (because the database is stupid)
  db.sql_GetRandomSpeaker, err = db.Prepare("SELECT Speaker FROM markov_transcripts_speaker WHERE ID = ?")
  db.sql_GetTableCounts, err = db.Prepare("SELECT CONCAT('Chatlog: ', (SELECT COUNT(*) FROM chatlog), ' rows', '\nEditlog: ', (SELECT COUNT(*) FROM editlog), ' rows',  '\nAliases: ', (SELECT COUNT(*) FROM aliases), ' rows',  '\nDebuglog: ', (SELECT COUNT(*) FROM debuglog), ' rows',  '\nPings: ', (SELECT COUNT(*) FROM pings), ' rows',  '\nUsers: ', (SELECT COUNT(*) FROM users), ' rows')")
  db.sql_Log, err = db.Prepare("INSERT INTO debuglog (Message, Timestamp) VALUE(?, Now(6))");
  
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

func (db *BotDB) AddMessage(id uint64, author uint64, message string, channel uint64, everyone bool) {
  _, err := db.sql_AddMessage.Exec(id, author, message, channel, everyone)
  db.log.LogError("AddMessage error: ", err)
}

func (db *BotDB) AddPing(message uint64, user uint64) {
  _, err := db.sql_AddPing.Exec(message, user)
  db.log.LogError("AddPing error: ", err)
}

func (db *BotDB) GetPing(user uint64, offset int) (uint64, uint64) {
  var id uint64
  var channel uint64
  err := db.sql_GetPing.QueryRow(user, offset).Scan(&id, &channel)
  if err == sql.ErrNoRows { return 0, 0 }
  db.log.LogError("GetPing error: ", err)
  return id, channel
}

type PingContext struct{ Author string; Message string; Timestamp time.Time }

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

func (db *BotDB) GetUser(id uint64) (*discordgo.User, time.Time) {
  u := &discordgo.User{}
  var lastseen time.Time
  err := db.sql_GetUser.QueryRow(id).Scan(&u.ID, &u.Email, &u.Username, &u.Avatar, &lastseen)
  db.log.LogError("GetUser error: ", err)
  return u, lastseen
}

func (db *BotDB) GetUserByName(name string) {
    
}

func (db *BotDB) GetRecentMessages(user uint64, duration uint64) []struct { message uint64; channel uint64 } {
  q, err := db.sql_GetRecentMessages.Query(user, duration)
  db.log.LogError("GetRecentMessages error: ", err)
  defer q.Close()
  r := make([]struct { message uint64; channel uint64 }, 0, 4)
  for q.Next() {
     p := struct { message uint64; channel uint64 }{}
     if err := q.Scan(&p.message, &p.channel); err == nil {
       r = append(r, p)
     }
  }
  return r
}

func (db *BotDB) UpdateUserJoinTime(id uint64, joinedat time.Time) {
  _, err := db.sql_UpdateUserJoinTime.Exec(id, joinedat)
  db.log.LogError("UpdateUserJoinTime error: ", err)
}

func (db *BotDB) GetNewestUsers(maxresults int) []struct { Username string; FirstSeen time.Time; LastSeen time.Time } {
  q, err := db.sql_GetNewestUsers.Query(maxresults)
  db.log.LogError("GetNewestUsers error: ", err)
  defer q.Close()
  r := make([]struct { Username string; FirstSeen time.Time; LastSeen time.Time }, 0, maxresults)
  for q.Next() {
     p := struct { Username string; FirstSeen time.Time; LastSeen time.Time }{}
     if err := q.Scan(&p.Username, &p.FirstSeen, &p.LastSeen); err == nil {
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
  Season uint
  Episode uint
  Line uint
  Speaker string
  Text string
}

func (db *BotDB) GetTranscript(season int, episode int, start int, end int) []Transcript {
  q, err := db.sql_GetTranscript.Query(season, episode, start, end)
  db.log.LogError("GetTranscript error: ", err)
  defer q.Close()
  l := end - start + 1
  if l > 100 { l = 100 }
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
func (db *BotDB) AddMarkov(last uint64, speaker string, text string) uint64 {
  var id uint64
  err := db.sql_AddMarkov.QueryRow(last, speaker, text).Scan(&id)
  db.log.LogError("AddMarkov error: ", err)
  return id
}

func (db *BotDB) GetMarkovLine(last uint64) (string, uint64) {
  var r sql.NullString
  err := db.sql_GetMarkovLine.QueryRow(last).Scan(&r)
  db.log.LogError("GetMarkovLine error: ", err)
  if !r.Valid { return "", 0 }
  str := strings.SplitN(r.String, "|", 2) // Being unable to call stored procedures makes this unnecessarily complex
  if len(str) < 2 || len(str[1])<1 {
    return str[0], 0
  }
  return str[0], SBatoi(str[1])
}
func (db *BotDB) GetMarkovWord(speaker string, phrase string) string {
  var r string
  err := db.sql_GetMarkovWord.QueryRow(speaker, phrase).Scan(&r)
  if err == sql.ErrNoRows { return phrase }
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
  if err == sql.ErrNoRows { return Transcript{0,0,0,"",""} }
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