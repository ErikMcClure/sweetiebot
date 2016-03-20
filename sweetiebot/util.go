package sweetiebot

import (
  "strconv"
  "time"
  "io/ioutil"
  "strings"
  "fmt"
  "regexp"
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
  return time.Now().UTC().Sub(t.Add(7*time.Hour))
}

func IngestEpisode(file string, season int, episode int) {
  f, err := ioutil.ReadFile(file)
  if err != nil { fmt.Println(err.Error()) }
  s := strings.Split(strings.Replace(string(f), "\r", "", -1), "\n")
  
  songmode := false
  lastcharacter := ""
  adjust := 0
  for i := 0; i < len(s); i++ {
    if len(s[i]) > 0 {
      if s[i][0] == '[' {
        action := s[i][1:len(s[i])-1]
        sb.db.AddTranscript(season, episode, i - adjust, "ACTION", action)
        if !songmode {
          lastcharacter = action
        }
      } else {
        split := strings.SplitN(s[i], ":", 2)
        songmode = (len(split) < 2)
        if songmode {
          prev := sb.db.GetTranscript(season, episode, i - 1 - adjust, i - 1 - adjust)
          if len(prev) != 1 { fmt.Println(season, " ", episode, " ", i - adjust); return }
          if prev[0].Speaker == "ACTION" && prev[0].Text == lastcharacter { 
            adjust++ 
            sb.db.RemoveTranscript(season, episode, i - adjust)
          }
          sb.db.AddTranscript(season, episode, i - adjust, lastcharacter, strings.TrimSpace(split[0]))
        } else {
          lastcharacter = strings.TrimSpace(split[0])
          sb.db.AddTranscript(season, episode, i - adjust, lastcharacter, strings.TrimSpace(split[1]))
        }
      }
    } else {
      sb.db.AddTranscript(season, episode, i - adjust, "ACTION", "")
    }
  }
}

func BuildMarkov(season_start int, episode_start int) {
  regex := regexp.MustCompile("[^~!@#$%^&*()_+`=[\\];,./<>?\" \n\r\f\t\v]+[?!.]?")
  
  var prev uint64
  for season := season_start; season <= 5; season++ {
    for episode := episode_start; episode <= 26; episode++ {
      fmt.Println("Begin Episode", episode, "Season", season)
      prev = 0
      lines := sb.db.GetTranscript(season, episode, 0, 999999)
      fmt.Println("Got", len(lines), "lines")
      
      for i := 0; i < len(lines); i++ {
        if len(lines[i].Text) == 0 {
          if lines[i].Speaker != "ACTION" {
            fmt.Println("UNKNOWN SPEAKER: ", lines[i].Speaker)
          }
          prev = sb.db.AddMarkov(prev, lines[i].Speaker, "")
          continue
        }
        words := regex.FindAllString(lines[i].Text, -1)
        for j := 0; j < len(words); j++ {
          l := len(words[j])
          ch := words[j][l-1]
          switch ch {
            case '.', '!', '?':
              words[j] = words[j][:l-1]
          }
          if sb.db.GetMarkovWord(lines[i].Speaker, words[j]) != words[j] {
            words[j] = strings.ToLower(words[j])
          }
          prev = sb.db.AddMarkov(prev, lines[i].Speaker, words[j])
          
          switch ch {
            case '.', '!', '?':
            prev = sb.db.AddMarkov(prev, lines[i].Speaker, string(ch))
            //prev = sb.db.AddMarkov(prev, "ACTION", "")
          }
        }
      }
    }
  }
}