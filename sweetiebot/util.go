package sweetiebot

import (
  "strconv"
  "time"
)

func Pluralize(i int64, s string) string {
  if i == 1 { return strconv.FormatInt(i, 10) + s }
  return strconv.FormatInt(i, 10) + s + "s"
}

func TimeDiff(d time.Duration) string {
  seconds := int64(d.Seconds())
  if seconds <= 60 { return Pluralize(seconds, " second") }
  if seconds <= 60*60 { return Pluralize(seconds/60, " minute") }
  if seconds <= 60*60*24 { return Pluralize(seconds/3600, " hour") }
  return Pluralize(seconds/86400, " day")
}

func SBatoi(s string) uint64 {
  i, err := strconv.ParseUint(s, 10, 64)
  if err != nil { 
    sb.log.Log("Invalid number ", s, ":", err.Error())
    return 0 
  }
  return i
}

func IsSpace(b byte) bool {
  return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func ParseArguments(s string) []string {
  r := []string{};
  l := len(s)
  for i := 0; i < l; i++ {
    c := s[i]
    if !IsSpace(c) {
      var start int;
      
      if c == '"' {
        i++
        start = i
        for i<(l-1) && (s[i] != '"' || !IsSpace(s[i+1])) { i++ }
      } else {
        start = i;
        i++
        for i<l && !IsSpace(s[i]) { i++ }
      }
      r = append(r, s[start:i])
    } 
  }
  return r
}

// This constructs an XOR operator for booleans
func boolXOR(a bool, b bool) bool {
  return (a && !b) || (!a && b)
}

func UserHasRole(user string, role string) bool {
  m, err := sb.dg.State.Member(sb.GuildID, user)
  if err == nil {
    for _, v := range m.Roles {
      if v == role {
        return true
      }
    } 
  }
  return false
}

func UserHasAnyRole(user string, roles map[uint64]bool) bool {
  if len(roles) == 0 { return true }
  m, err := sb.dg.State.Member(sb.GuildID, user)
  if err == nil {
    for _, v := range m.Roles {
      _, ok := roles[SBatoi(v)]
      if ok {
        return true
      }
    }
  }
  return false
}

func ReadUserPingArg(args []string) (uint64, string) {
  if len(args) < 1 {
    return 0, "```You must provide a user to search for.```"
  }
  if len(args[0]) < 3 || args[0][0] != '<' || args[0][1] != '@' {
    return 0, "```The first argument must be an actual ping for the target user, not just their name typed out.```"
  }
  return SBatoi(args[0][2:len(args[0])-1]), ""
}

func SinceUTC(t time.Time) time.Duration {
  return time.Now().UTC().Sub(t.Add(8*time.Hour))
}