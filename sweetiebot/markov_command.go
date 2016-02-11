package sweetiebot

import (
  "github.com/bwmarrin/discordgo"
  "strings"
  "strconv"
  "fmt"
  "regexp"
  "math/rand"
)

type EpisodeGenCommand struct {
  lock AtomicFlag
}

func (c *EpisodeGenCommand) Name() string {
  return "episodegen";  
}
func (c *EpisodeGenCommand) Process(args []string, user *discordgo.User, channel string) (string, bool) {
  if c.lock.test_and_set() {
    return "```Sorry, I'm busy processing another request right now. Please try again later!```", false
  }
  defer c.lock.clear()
  maxlines := sb.config.Defaultmarkovlines
  if len(args) > 0 {
    maxlines, _ = strconv.Atoi(args[0])
  }
  if maxlines > sb.config.Maxmarkovlines { maxlines = sb.config.Maxmarkovlines }
  if maxlines <= 0 { maxlines = 1 }
  var prev uint64
  prev = 0
  lines := make([]string, 0, maxlines)
  line := ""
  for i := 0; i < maxlines; i++ {
    line, prev = sb.db.GetMarkovLine(prev)
    lines = append(lines, line);
  }
  
  return strings.Join(lines, "\n"), len(lines)>5 || !CheckShutup(channel)
}
func (c *EpisodeGenCommand) Usage() string { 
  return FormatUsage(c, "[lines]", "Randomly generates a my little pony episode using a markov chain, up to a maximum line count of [lines]. Will be sent via PM if the line count exceeds 5.") 
}
func (c *EpisodeGenCommand) UsageShort() string { return "Randomly generates episodes." }
func (c *EpisodeGenCommand) Roles() []string { return []string{} }

type QuoteCommand struct {
}

var quoteargregex = regexp.MustCompile("s[0-9]+e[0-9]+")

func (c *QuoteCommand) Name() string {
  return "Quote";  
}
func (c *QuoteCommand) Process(args []string, user *discordgo.User, channel string) (string, bool) {
  if !CheckShutup(channel) { return "", false }
  S := 0
  E := 0
  L := 0
  diff := 0
  var lines []Transcript
  if len(args) < 1 {
    lines = []Transcript{sb.db.GetRandomQuote()}
  } else {
    arg := strings.ToLower(args[0])
    switch arg {
      case "action":
        lines = []Transcript{sb.db.GetCharacterQuote("ACTION")}
      case "speech":
        lines = []Transcript{sb.db.GetSpeechQuote()}
      default:
        if quoteargregex.MatchString(arg) {
          n, err := fmt.Sscanf(arg, "s%de%d:%d-%d", &S, &E, &L, &diff)
          if err != nil {
            if n < 3 {
              sb.log.Log("quote scan error: ", err.Error())
              return "```Error: Could not parse your request. Be sure it is in the format S0E00:000-000. Example: S4E22:7-14```", false
            }
            if n < 4 { diff = L }
          }
          diff--
          L--
          
          diff -= L  
          if diff >= sb.config.Maxquotelines { diff = sb.config.Maxquotelines-1 }
          lines = sb.db.GetTranscript(S, E, L, L+diff)
        } else { // Otherwise this is a character quote request
          lines = []Transcript{sb.db.GetCharacterQuote(arg)}
          if lines[0].Season == 0 {
            return "```Error: Could not find character " + arg + " in the transcripts. Make sure you specify the entire name and spelled it correctly!```", false
          }
        }
    }
  }
  
  process := make([]string, 0, len(lines))
  for _, v := range lines {
    l := ""
    if v.Speaker == "ACTION" {
      if v.Text != "" {
        l = "[" + v.Text + "]"
      }
    } else {
      l = "**" + v.Speaker + "**: " + v.Text
    }
    process = append(process, l)
  }
  return strings.Join(process, "\n"), len(process) > sb.config.MaxPMlines
}
func (c *QuoteCommand) Usage() string { 
  return FormatUsage(c, "[S0E00:000-000|action|speech|\"Character Name\"]", "If the S0E00:000-000 format is used, returns all the lines from the given season and episode, between the starting and ending line numbers (inclusive). Returns a maximum of " + strconv.Itoa(sb.config.Maxquotelines) + " lines, but a line count above 5 will be sent in a private message. Example: !quote S4E22:7-14\n\nIf \"action\" is specified, returns a random action quote from the show.\n\nIf \"speech\" is specified, returns a random quote from one of the characters in the show.\n\nIf a \"Character Name\" is specified, it attempts to quote a random line from the show spoken by that character. If the character can't be found, returns an error. The character name doesn't have to be in quotes unless it has spaces in it, but you must specify the entire name.\n\nIf no arguments are specified, quotes a completely random line from the show.") 
}
func (c *QuoteCommand) UsageShort() string { return "Quotes random or specific lines from the show." }
func (c *QuoteCommand) Roles() []string { return []string{} }

type ShipCommand struct {
}

func (c *ShipCommand) Name() string {
  return "ship";  
}
func (c *ShipCommand) Process(args []string, user *discordgo.User, channel string) (string, bool) {
  if !CheckShutup(channel) { return "", false }
  a := sb.db.GetRandomSpeaker()
  b := sb.db.GetRandomSpeaker()
  s := ""
  if len(args) > 0 {
    a = args[0]
  }
  if len(args) > 1 {
    b = args[1]
  }
  switch rand.Int31n(11) {
    case 0:
      s = "%s is %s's overly attached fianc\u00e9."
    case 1:
      s = "%s boops %s."
    case 2:
      s = "%s and %s, sitting in a tree, K-I-S-S-- well, you know the rest."
    case 3:
      s = "%s falls head over hooves for %s."
    case 4:
      s = "%s watches %s sleep at night."
    case 5:
      s = "%s is secretly in love with %s."
    case 6:  
      s = "%s x %s"
    case 7:
      s = "%s and %s argue all the time and refuse to admit their feelings for each other."
    case 8:
      s = "%s makes %s's heart flutter."
    case 9:
      s = "%s stumbles on %s's darkest secret, only for it to bring them closer together."
    case 10:
      s = "%s and %s get REALLY drunk, and then-- wait, this channel is SFW."
  }
  
  return fmt.Sprintf("```" + s + "```", a, b), false
}
func (c *ShipCommand) Usage() string { 
  return FormatUsage(c, "[first] [second]", "Generates a random pairing of ponies from the show. If a first or second argument is supplied, uses those ponies instead.") 
}
func (c *ShipCommand) UsageShort() string { return "Generates a random ship." }
func (c *ShipCommand) Roles() []string { return []string{} }