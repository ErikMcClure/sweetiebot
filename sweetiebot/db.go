package sweetiebot

import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "fmt"
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
  db.sql_AddMessage, err = db.Prepare("CALL AddChat(?,?,?,?)");
  db.sql_AddPing, err = db.Prepare("INSERT INTO pings (Message, User) VALUES (?, ?)");
  db.sql_GetPings, err = db.Prepare("SELECT * FROM pings P INNER JOIN chatlog C ON P.Message = C.ID WHERE P.User = ? OR C.Everyone = 1");
  db.sql_AddUser, err = db.Prepare("CALL AddUser(?,?,?,?,?)");
  db.sql_GetUser, err = db.Prepare("SELECT * FROM users WHERE ID = ?");
  db.sql_GetUserByName, err = db.Prepare("SELECT * FROM users WHERE Username = ?");
  db.sql_Log, err = db.Prepare("INSERT INTO debuglog (Message, Timestamp) VALUE(?, Now(6))");
  return err
}

func (db *BotDB) AddMessage(id uint64, author uint64, message string, channel uint64, everyone bool) {
  _, err := db.sql_AddMessage.Query(id, author, message, channel, everyone)
  db.log.LogError("AddMessage error: ", err)
}

func (db *BotDB) AddPing(message uint64, user uint64) {
  _, err := db.sql_AddPing.Query(message, user)
  db.log.LogError("AddPing error: ", err)
}

func (db *BotDB) GetPings(user uint64) {
    
}

func (db *BotDB) AddUser(id uint64, email string, username string, avatar string, verified bool) {
  _, err := db.sql_AddUser.Query(id, email, username, avatar, verified)
  db.log.LogError("AddUser error: ", err)
}

func (db *BotDB) GetUser(id uint64) {
    
}

func (db *BotDB) GetUserByName(name string) {
    
}

func (db *BotDB) Log(message string) {
    _, err := db.sql_Log.Query(message)
    if err != nil {
      fmt.Println("Logger failed to log to database! ", err.Error())
    }
}

