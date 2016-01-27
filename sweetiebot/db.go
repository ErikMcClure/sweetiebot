package sweetiebot

import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
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
    db.log.Log("Preparing: ", s, "\nSQL Error: ", err.Error())
  }
  return statement, err
}

func (db *BotDB) LoadStatements() error {
  var err error;
  db.sql_AddMessage, err = db.Prepare("CALL AddChat(?,?,?,?,?)");
  db.sql_AddPing, err = db.Prepare("INSERT INTO pings (Message, User) VALUES (?, ?) ON DUPLICATE KEY UPDATE Message = Message");
  db.sql_GetPings, err = db.Prepare("SELECT * FROM pings P INNER JOIN chatlog C ON P.Message = C.ID WHERE P.User = ? OR C.Everyone = 1");
  db.sql_AddUser, err = db.Prepare("CALL AddUser(?,?,?,?,?)");
  db.sql_GetUser, err = db.Prepare("SELECT * FROM users WHERE ID = ?");
  db.sql_GetUserByName, err = db.Prepare("SELECT * FROM users WHERE Username = ?");
  db.sql_GetRecentMessages, err = db.Prepare("SELECT ID, Channel FROM chatlog WHERE Author = ? AND Timestamp >= DATE_SUB(Now(6), INTERVAL ? SECOND)");
  db.sql_UpdateUserJoinTime, err = db.Prepare("CALL UpdateUserJoinTime(?, ?)");
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

func (db *BotDB) GetUser(id uint64) {
    
}

func (db *BotDB) GetUserByName(name string) {
    
}

type MessageChannelPair struct {
  message uint64
  channel uint64
}

func (db *BotDB) GetRecentMessages(user uint64, duration uint64) []MessageChannelPair {
  q, err := db.sql_GetRecentMessages.Query(user, duration)
  db.log.LogError("GetRecentMessages error: ", err)
  defer q.Close()
  r := make([]MessageChannelPair, 0, 4)
  for q.Next() {
     p := MessageChannelPair{}
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
  
func (db *BotDB) Log(message string) {
  _, err := db.sql_Log.Exec(message)
  if err != nil {
    fmt.Println("Logger failed to log to database! ", err.Error())
  }
}

