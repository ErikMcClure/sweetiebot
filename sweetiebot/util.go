package sweetiebot

import (
  "strconv"
  "time"
  "io/ioutil"
  "strings"
  "fmt"
  "regexp"
  "math/rand"
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

func PingAtoi(s string) uint64 {
  if s[:2] == "<#" || s[:2] == "<@" {
    return SBatoi(s[2:len(s)-1])
  }
  return SBatoi(s)
}
func SBatoi(s string) uint64 {
  if s[:1] == "!" || s[:1] == "&" { s = s[1:] }
  i, err := strconv.ParseUint(strings.Replace(s, "\u200B", "", -1), 10, 64)
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

func UserHasAnyRole(user string, roles map[string]bool) bool {
  if len(roles) == 0 { return true }
  m, err := sb.dg.State.Member(sb.GuildID, user)
  if err == nil {
    for _, v := range m.Roles {
      _, ok := roles[v]
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

func SplitSpeaker(speaker string) []string {
  speakers := strings.Split(strings.Replace(speaker, ", and", " and", -1), " and ")
  speakers = append(strings.Split(speakers[0], ","), speakers[1:]...)
  for i, s := range speakers {
    speakers[i] = strings.Trim(strings.TrimSpace(strings.Replace(s, "Young", "", -1)), "\"")
  }
  return speakers
}

func BuildMarkov(season_start int, episode_start int) {
  regex := regexp.MustCompile("[^~!@#$%^&*()_+`=[\\];,./<>?\" \n\r\f\t\v]+[?!.]?")
  
  sb.db.sql_ResetMarkov.Exec()
    
  var cur uint64
  var prev uint64
  var prev2 uint64
  for season := season_start; season <= 5; season++ {
    for episode := episode_start; episode <= 26; episode++ {
      fmt.Println("Begin Episode", episode, "Season", season)
      prev = 0
      prev2 = 0
      lines := sb.db.GetTranscript(season, episode, 0, 999999)
      //lines := []Transcript{ {1, 1, 1, "Twilight", "Twilight went to the bakery to buy some cakes."}, {1, 1, 1, "Twilight", "Twilight went to the library to buy some books"} }
      fmt.Println("Got", len(lines), "lines")
      
      for i := 0; i < len(lines); i++ {
        if len(lines[i].Text) == 0 {
          if lines[i].Speaker != "ACTION" {
            fmt.Println("UNKNOWN SPEAKER: ", lines[i].Speaker)
          }
          cur = sb.db.AddMarkov(prev, prev2, lines[i].Speaker, "")
          prev2 = 0
          prev = cur // Cur will always be 0 here.
          continue
        }
        words := regex.FindAllString(lines[i].Text, -1)
        speakers := SplitSpeaker(lines[i].Speaker)
        for _, speaker := range speakers {
          if len(speaker) == 0 {
            fmt.Println("EMPTY SPEAKER GENERATED FROM \"" + lines[i].Speaker + "\" ON LINE: ", lines[i].Text);
            fmt.Println(speakers)
          }
          for j , _ := range words {
            l := len(words[j])
            ch := words[j][l-1]
            switch ch {
              case '.', '!', '?':
                words[j] = words[j][:l-1]
            }
            if sb.db.GetMarkovWord(speaker, words[j]) != words[j] {
              words[j] = strings.ToLower(words[j])
            }
            //fmt.Println("AddMarkov: ", prev, prev2, speaker, words[j])
            cur = sb.db.AddMarkov(prev, prev2, speaker, words[j])
            prev2 = prev
            prev = cur
            
            switch ch {
              case '.', '!', '?':
              //fmt.Println("AddMarkov: ", prev, prev2, speaker, string(ch))
              cur = sb.db.AddMarkov(prev, prev2, speaker, string(ch))
              prev2 = 0
              prev = 0
              //prev = sb.db.AddMarkov(prev, "ACTION", "")
            }
          }
        }
      }
    }
  }
}

func MapGetRandomItem(m map[string]bool) string {
  index := rand.Intn(len(m))
  for k, _ := range m {
    if index == 0 {
      return k
    }
    index--
  }
  
  return "SOMETHING IMPOSSIBLE HAPPENED IN UTIL.GO MapGetRandomItem()! Somebody drag Cloud Hop out of bed and tell him his bot is broken."
}

func MapToSlice(m map[string]bool) []string {
  s := make([]string, 0, len(m))
  for k, _ := range m {
    s = append(s, k)
  }
  return s
}

func MapStringToSlice(m map[string]string) []string {
  s := make([]string, 0, len(m))
  for k, _ := range m {
    s = append(s, k)
  }
  return s
}

func RemoveSliceString(s *[]string, item string) bool {
  for i := 0; i < len(*s); i++ {
    if (*s)[i] == item {
      *s = append((*s)[:i], (*s)[i+1:]...)
      return true
    }
  }
  return false
}

func RemoveSliceInt(s *[]uint64, item uint64) bool {
  for i := 0; i < len(*s); i++ {
    if (*s)[i] == item {
      *s = append((*s)[:i], (*s)[i+1:]...)
      return true
    }
  }
  return false
}