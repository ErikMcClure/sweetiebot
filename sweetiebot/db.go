package sweetiebot

import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/bwmarrin/discordgo"
    "fmt"
    "time"
)

type BotDB struct {
  db *sql.DB
  log Logger
  sql_AddMessage *sql.Stmt
  sql_AddPing *sql.Stmt
  sql_GetPings *sql.Stmt
  sql_AddUser *sql.Stmt
  sql_GetUser *sql.Stmt
  sql_GetUserByName *sql.Stmt
  sql_GetRecentMessages *sql.Stmt
  sql_UpdateUserJoinTime *sql.Stmt
  sql_GetNewestUsers *sql.Stmt
  sql_GetAliases *sql.Stmt
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
  db.sql_GetPings, err = db.Prepare("SELECT C.ID FROM pings P INNER JOIN chatlog C ON P.Message = C.ID WHERE P.User = ? OR C.Everyone = 1 ORDER BY Timestamp DESC");
  db.sql_AddUser, err = db.Prepare("CALL AddUser(?,?,?,?,?)");
  db.sql_GetUser, err = db.Prepare("SELECT ID, Email, Username, Avatar FROM users WHERE ID = ?");
  db.sql_GetUserByName, err = db.Prepare("SELECT * FROM users WHERE Username = ?");
  db.sql_GetRecentMessages, err = db.Prepare("SELECT ID, Channel FROM chatlog WHERE Author = ? AND Timestamp >= DATE_SUB(Now(6), INTERVAL ? SECOND)");
  db.sql_UpdateUserJoinTime, err = db.Prepare("CALL UpdateUserJoinTime(?, ?)");
  db.sql_GetNewestUsers, err = db.Prepare("SELECT Username, FirstSeen, LastSeen FROM users ORDER BY FirstSeen DESC LIMIT ?")
  db.sql_GetAliases, err = db.Prepare("SELECT Alias FROM aliases WHERE User = ? ORDER BY Duration DESC LIMIT 10")
  db.sql_Log, err = db.Prepare("INSERT INTO debuglog (Message, Timestamp) VALUE(?, Now(6))");
  
  return err
}

func (db *BotDB) AddMessage(id uint64, author uint64, message string, channel uint64, everyone bool) {
  _, err := db.sql_AddMessage.Exec(id, author, message, channel, everyone)
  db.log.LogError("AddMessage error: ", err)
}

func (db *BotDB) AddPing(message uint64, user uint64) {
  _, err := db.sql_AddPing.Exec(message, user)
  db.log.LogError("AddPing error: ", err)
}

func (db *BotDB) GetPings(user uint64) {
    
}

func (db *BotDB) AddUser(id uint64, email string, username string, avatar string, verified bool) {
  _, err := db.sql_AddUser.Exec(id, email, username, avatar, verified)
  db.log.LogError("AddUser error: ", err)
}

func (db *BotDB) GetUser(id uint64) *discordgo.User {
  u := &discordgo.User{}
  err := db.sql_GetUser.QueryRow(id).Scan(&u.ID, &u.Email, &u.Username, &u.Avatar)
  db.log.LogError("GetUser error: ", err)
  return u
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
     db.log.LogError("GetRecentMessages row scan error: ", err)
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
     db.log.LogError("GetNewestUsers row scan error: ", err)
  }
  return r
}
  
func (db *BotDB) GetAliases(user uint64) []string {
  q, err := db.sql_GetAliases.Query(user)
  db.log.LogError("GetAliases error: ", err)
  defer q.Close()
  r := make([]string, 0, 3)
  for q.Next() {
     p := ""
     if err := q.Scan(&p); err == nil {
       r = append(r, p)
     }
     db.log.LogError("GetAliases row scan error: ", err)
  }
  return r
}

func (db *BotDB) Log(message string) {
  _, err := db.sql_Log.Exec(message)
  if err != nil {
    fmt.Println("Logger failed to log to database! ", err.Error())
  }
}

