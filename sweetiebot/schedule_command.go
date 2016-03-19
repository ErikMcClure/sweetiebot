package sweetiebot

/*import (
  "github.com/bwmarrin/discordgo"
  "strconv"
  "time"
  "strings"
)

type ScheduleCommand struct {
}

func (c *ScheduleCommand) Name() string {
  return "Schedule";  
}
func (c *ScheduleCommand) Process(args []string, msg *discordgo.Message) (string, bool) {
  maxresults := 5
  
  if len(args) > 0 {
    maxresults = strconv.Atoi(args[0])
  }
  if maxresults > 26 { maxresults = 26 }
  if maxresults < 1 { maxresults = 1 }
  
}
func (c *ScheduleCommand) Usage() string { 
  return FormatUsage(c, "[maxresults]", "Lists up to maxresults upcoming episodes that will air from the estimated schedule. Defaults to 5, maximum of 26.") 
}
func (c *ScheduleCommand) UsageShort() string { return "[PM Only] Gets a list of upcoming episodes." }
func (c *ScheduleCommand) Roles() []string { return []string{} }
func (c *ScheduleCommand) Channels() []string { return []string{} }

type NextEpisode struct {
}

func (c *NextEpisode) Name() string {
  return "NextEpisode";  
}
func (c *NextEpisode) Process(args []string, msg *discordgo.Message) (string, bool) {
  
}
func (c *NextEpisode) Usage() string { 
  return FormatUsage(c, "", "Gets the time until the next episode of My Little Pony based on estimated episode airing times.") 
}
func (c *NextEpisode) UsageShort() string { return "Gets time until next episode." }
func (c *NextEpisode) Roles() []string { return []string{} }
func (c *NextEpisode) Channels() []string { return []string{} }

type AddSchedule struct {
}

func (c *AddSchedule) Name() string {
  return "AddSchedule";  
}
func (c *AddSchedule) Process(args []string, msg *discordgo.Message) (string, bool) {
  if len(args) < 1 {
    return "```No time specified! Make sure it's in the format \"02 Jan 06 15:04 MST\"```", false 
  }
  if args[0] == "remove" {
    
  } 
  
  t, err := time.Parse(time.RFC822, args[0])
  if err != nil {
    return "```Error: could not parse time! Make sure it's in the format \"02 Jan 06 15:04 MST\"```", false 
  }
  sb.config.Schedule = append(sb.config.Schedule, t)
  return "```Added time to schedule.```", false
}
func (c *AddSchedule) Usage() string { 
  return FormatUsage(c, "[time|remove] [index]", "Adds an episode time to the schedule using the format \"02 Jan 06 15:04 MST\", or removes the 'index' episode if 'remove' is specified.") 
}
func (c *AddSchedule) UsageShort() string { return "Adds an episode to the schedule." }
func (c *AddSchedule) Roles() []string { return []string{"Princesses", "Royal Guard", "Night Guard"} }
func (c *AddSchedule) Channels() []string { return []string{} }*/