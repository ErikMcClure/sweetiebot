package sweetiebot

import (
  "strconv"
  "time"
  "strings"
)

type NewUsersCommand struct {
}

func (c *NewUsersCommand) Name() string {
  return "newusers";  
}
func (c *NewUsersCommand) Process(args []string) string {
  maxresults := 5
  if len(args) > 0 { maxresults, _ = strconv.Atoi(args[0]) }
  if maxresults < 1 { return "```How I return no results???```" }
  if maxresults > 30 { maxresults = 30 }
  r := sb.db.GetNewestUsers(maxresults)
  s := make([]string, 0, len(r))
  
  for _, v := range r {
    s = append(s, v.Username + "  (joined: " + v.FirstSeen.Format(time.ANSIC) + ")") 
  }
  return "```" + strings.Join(s, "\n") + "```"
}
func (c *NewUsersCommand) Usage() string { 
  return FormatUsage(c, "[maxresults]", "Lists up to maxresults users, starting with the newest user to join the server. Defaults to 5 results, returns a maximum of 30.") 
}
func (c *NewUsersCommand) UsageShort() string { return "Gets a list of the most recent users to join the server." }
func (c *NewUsersCommand) Roles() []string { return []string{"Princesses", "Royal Guard"} }