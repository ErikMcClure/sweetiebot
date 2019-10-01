package tagmodule

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"

	bot "../sweetiebot"
	"github.com/blackhole12/discordgo"
)

const (
	maxPublicUniqueItems = 5000
	maxTagResults        = 50
)

var tagargregex = regexp.MustCompile("[^-+()| ][^-+()|]*")

// TagModule contains commands for manipulating tags
type TagModule struct {
	Cache map[string]*sql.Stmt
}

// New instance of TagModule
func New() *TagModule {
	return &TagModule{make(map[string]*sql.Stmt)}
}

// Name of the module
func (w *TagModule) Name() string {
	return "Tag"
}

// Commands in the module
func (w *TagModule) Commands() []bot.Command {
	return []bot.Command{
		&addCommand{},
		&getCommand{},
		&removeCommand{},
		&tagsCommand{w},
		&pickCommand{w},
		&newCommand{},
		&deleteCommand{},
		&searchTagsCommand{w},
		&importCommand{},
	}
}

// Description of the module
func (w *TagModule) Description() string {
	return "Contains commands for manipulating tags."
}
func (w *TagModule) prepStatement(query string, tags string, db *bot.BotDB) (*sql.Stmt, error) {
	stmt, ok := w.Cache[query]
	if !ok {
		var err error
		stmt, err = db.Prepare(query)
		if err != nil {
			//return nil, fmt.Errorf("Invalid tag expression: %s", err.Error())
			return nil, fmt.Errorf("Invalid tag expression: %s", tags)
		}
		w.Cache[query] = stmt
	}
	return stmt, nil
}

func (w *TagModule) execStatement(query string, tags string, db *bot.BotDB, args ...interface{}) (*sql.Rows, error) {
	stmt, err := w.prepStatement(query, tags, db)
	if err != nil {
		return nil, err
	}

	q, err := stmt.Query(args...)
	return q, db.CheckError("Query: "+query, err)
}

func getTagIDs(tags []string, guild uint64, db *bot.BotDB) ([]uint64, error) {
	tagIDs := make([]uint64, len(tags), len(tags))
	for k, v := range tags {
		id, err := db.GetTag(v, guild)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("The %s tag does not exist!", v)
		} else if err != nil {
			return nil, err
		}
		tagIDs[k] = id
	}
	return tagIDs, nil
}

// ValidateWhereClause ensures that all parentheses are properly closed in a given string
func ValidateWhereClause(arg string) bool {
	count := 0
	for _, v := range arg {
		if v == '(' {
			count++
		} else if v == ')' {
			count--
		}
		if count < 0 {
			return false
		}
	}
	return count == 0
}

// BuildWhereClause returns a valid mySQL WHERE clause and argument list from a logical tag expression
func BuildWhereClause(arg string) (string, []string) {
	args := tagargregex.FindAllString(arg, -1)
	arg = tagargregex.ReplaceAllString(arg, "M.Item IN (SELECT Item FROM itemtags WHERE Tag = ?)")
	arg = strings.Replace(arg, "-", " NOT ", -1)
	arg = strings.Replace(arg, "+", " AND ", -1)
	arg = strings.Replace(arg, "|", " OR ", -1)
	arg = strings.Replace(arg, "Tag = ?) OR M.Item IN (SELECT Item FROM itemtags WHERE", "Tag = ? OR", -1)
	return arg, args
}

type addCommand struct {
}

func (c *addCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "Add",
		Usage:     "Adds an item to a set of tags.",
		Sensitive: true,
	}
}
func (c *addCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nNo tags given```", false, nil
	}
	if len(args) < 2 {
		return "```\nCan't add empty string!```", false, nil
	}

	var max uint64 = maxPublicUniqueItems
	if info.Silver.Get() {
		max = info.Bot.MaxUniqueItems
	}
	gID := bot.SBatoi(info.ID)
	if count, _ := info.Bot.DB.CountItems(gID); count >= max {
		if info.Silver.Get() {
			return fmt.Sprintf("```Can't have more than %v unique items in a server!```", max), false, nil
		}
		return fmt.Sprintf("```Can't have more than %v unique items in a server! If you want up to %v items, upgrade to Silver for $1 a month by contributing: %s```", max, info.Bot.MaxUniqueItems, bot.PatreonURL), false, nil
	}

	tags := strings.Split(args[0], "+")
	tagIDs, err := getTagIDs(tags, gID, info.Bot.DB)
	if err != nil {
		return bot.ReturnError(err)
	}

	item := msg.Content[indices[1]:]
	id, err := info.Bot.DB.AddItem(item)
	if err != nil && err != bot.ErrDuplicateEntry {
		return bot.ReturnError(err)
	}
	for _, v := range tagIDs {
		info.Bot.DB.AddTag(id, v)
	}

	for k := range tags {
		count, err := info.Bot.DB.CountTag(tagIDs[k])
		if err == nil {
			tags[k] = fmt.Sprintf("%s: %v items", tags[k], count)
		}
	}
	itemtags := info.Bot.DB.GetItemTags(id, gID)
	return fmt.Sprintf("```\n%s: %s\n---\n%s```", info.Sanitize(item, bot.CleanCodeBlock), info.Sanitize(strings.Join(itemtags, ", "), bot.CleanCodeBlock), info.Sanitize(strings.Join(tags, "\n"), bot.CleanCodeBlock)), false, nil
}
func (c *addCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Adds [arbitrary string] to [tags]. If the item already exists, simply adds the tags to the existing item.",
		Params: []bot.CommandUsageParam{
			{Name: "tag(s)", Desc: "The name of a tag. Specify multiple tags with \"tag1+tag2\" (with quotes if there are spaces).", Optional: false},
			{Name: "arbitrary string", Desc: "Arbitrary string to add tags to. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}

type getCommand struct {
}

func (c *getCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "GetTags",
		Usage: "Gets all tags an item has, if any.",
	}
}
func (c *getCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nNo item given```", false, nil
	}

	gID := bot.SBatoi(info.ID)

	item := msg.Content[indices[0]:]
	id, err := info.Bot.DB.GetItem(item)
	if err == sql.ErrNoRows {
		return "```\nThat item has no tags.```", false, nil
	} else if err != nil {
		return bot.ReturnError(err)
	}

	tags := info.Bot.DB.GetItemTags(id, gID)
	if len(tags) == 0 {
		return "```\nThat item has no tags.```", false, nil
	}
	return fmt.Sprintf("```%s: %s```", info.Sanitize(item, bot.CleanCodeBlock), info.Sanitize(strings.Join(tags, ", "), bot.CleanCodeBlock)), false, nil
}
func (c *getCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Returns all tags associated with [arbitrary string], or tells you the item has no tags",
		Params: []bot.CommandUsageParam{
			{Name: "arbitrary string", Desc: "Arbitrary string to get the tags of. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}

type removeCommand struct {
}

func (c *removeCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "Remove",
		Usage:     "Removes tags from an item.",
		Sensitive: true,
	}
}

func (c *removeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou must specify what item you want to remove!```", false, nil
	}

	gID := bot.SBatoi(info.ID)
	if len(args) < 2 || args[0] == "*" {
		item := msg.Content[indices[0]:]
		if args[0] == "*" && len(args) > 1 {
			item = msg.Content[indices[1]:]
		}
		id, err := info.Bot.DB.GetItem(item)
		if err == sql.ErrNoRows {
			return fmt.Sprintf("```%s doesn't exist!```", item), false, nil
		}
		if err == nil {
			if len(info.Bot.DB.GetItemTags(id, gID)) == 0 {
				return fmt.Sprintf("```%s doesn't exist!```", item), false, nil
			}
			err = info.Bot.DB.RemoveItem(id, gID)
		}
		if err != nil {
			return bot.ReturnError(err)
		}
		return fmt.Sprintf("```Removed %s from all tags.```", info.Sanitize(item, bot.CleanCodeBlock)), false, nil
	}

	tags := strings.Split(args[0], "+")
	tagIDs, err := getTagIDs(tags, gID, info.Bot.DB)
	if err != nil {
		return bot.ReturnError(err)
	}

	item := msg.Content[indices[1]:]
	id, err := info.Bot.DB.GetItem(item)
	if err == sql.ErrNoRows {
		return fmt.Sprintf("```That item doesn't exist!```"), false, nil
	} else if err != nil {
		return bot.ReturnError(err)
	}

	for _, v := range tagIDs {
		info.Bot.DB.RemoveTag(id, v)
	}

	itemtags := info.Bot.DB.GetItemTags(id, gID)
	if len(itemtags) == 0 {
		return fmt.Sprintf("```Removed %s (item has no tags).```", info.Sanitize(item, bot.CleanCodeBlock)), false, nil
	}

	for k := range tags {
		count, err := info.Bot.DB.CountTag(tagIDs[k])
		if err == nil {
			tags[k] = fmt.Sprintf("%s: %v items", tags[k], count)
		}
	}
	return fmt.Sprintf("```%s: %s\n---\n%s```", info.Sanitize(item, bot.CleanCodeBlock), info.Sanitize(strings.Join(itemtags, ", "), bot.CleanCodeBlock), info.Sanitize(strings.Join(tags, "\n"), bot.CleanCodeBlock)), false, nil
}
func (c *removeCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Removes [arbitrary string] from [tags], unless [tags] is *, in which case it removes it from all tags.",
		Params: []bot.CommandUsageParam{
			{Name: "tag(s)", Desc: "The name of a tag. Specify multiple tags with \"tag1+tag2\" (with quotes if there are spaces), or use * to remove it from all tags.", Optional: true},
			{Name: "arbitrary string", Desc: "Arbitrary string to remove from the given tags. If this has spaces, don't omit the tag argument - use * instead.", Optional: false},
		},
	}
}

type memberFields []*discordgo.MessageEmbedField

func (f memberFields) Len() int {
	return len(f)
}

func (f memberFields) Less(i, j int) bool {
	return strings.Compare(f[i].Name, f[j].Name) < 0
}

func (f memberFields) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type tagsCommand struct {
	w *TagModule
}

func (c *tagsCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "Tags",
		Usage: "Lists all tags.",
	}
}

// ShowAllTags builds an embed message containing all tags
func ShowAllTags(message string, info *bot.GuildInfo) *discordgo.MessageEmbed {
	tags := info.Bot.DB.GetTags(bot.SBatoi(info.ID))
	items, _ := info.Bot.DB.CountItems(bot.SBatoi(info.ID))
	fields := make(memberFields, len(tags), len(tags))

	for k, v := range tags {
		fields[k] = &discordgo.MessageEmbedField{Name: "**" + v.Name + "**", Value: fmt.Sprintf("%v items", v.Count), Inline: true}
	}
	sort.Sort(fields)
	return &discordgo.MessageEmbed{
		Type: "rich",
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://github.com/blackhole12/sweetiebot",
			Name:    info.GetBotName() + " Tags",
			IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", info.Bot.SelfID, info.Bot.SelfAvatar),
		},
		Description: message + fmt.Sprintf(" Tags: %v. Unique Items: %v", len(tags), items),
		Color:       0x3e92e5,
		Fields:      fields,
	}
}
func (c *tagsCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "", false, ShowAllTags("No tag specified.", info)
	}

	arg := msg.Content[indices[0]:]
	if !ValidateWhereClause(arg) {
		return "```\nMismatched parentheses!```", false, nil
	}
	clause, tags := BuildWhereClause(arg)
	gID := bot.SBatoi(info.ID)
	tagIDs, err := getTagIDs(tags, gID, info.Bot.DB)
	if err != nil {
		return bot.ReturnError(err)
	}

	params := make([]interface{}, len(tagIDs), len(tagIDs))
	for k, v := range tagIDs {
		params[k] = v
	}
	params = append(params, gID)
	q, err := c.w.execStatement("SELECT I.Content FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID INNER JOIN items I ON M.Item = I.ID WHERE "+clause+" AND T.Guild = ? GROUP BY I.Content", arg, info.Bot.DB, params...)
	if err != nil {
		return bot.ReturnError(err)
	}

	defer q.Close()
	r := make([]string, 0, 3)
	for q.Next() {
		var p string
		if err := q.Scan(&p); err == nil {
			r = append(r, p)
		}
	}

	if len(r) == 0 {
		return fmt.Sprintf("```\nNo items match %s!```", arg), false, nil
	}

	var s string
	userID := bot.DiscordUser(msg.Author.ID)
	if len(r) > maxTagResults && !info.UserIsMod(userID) && !info.UserIsAdmin(userID) {
		s = strings.Join(r[:maxTagResults], "\n")
		arg += " (truncated)"
	} else {
		s = strings.Join(r, "\n")
	}
	s = strings.Replace(s, "```", "\\`\\`\\`", -1)
	s = strings.Replace(s, "[](/", "[\u200B](/", -1)
	return fmt.Sprintf("```\n%v items satisfy %s:\n%s```", len(r), arg, s), len(r) > bot.MaxPublicLines, nil
}
func (c *tagsCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Lists all the tags in this server, or the contents of a specific tag search string.",
		Params: []bot.CommandUsageParam{
			{Name: "tag(s)", Desc: "An optional arbitrary tag search using the syntax `tag1|(tag2+(-tag3))`, which translates to `tag1 OR (tag2 AND NOT tag3)`.", Optional: true},
		},
	}
}

type pickCommand struct {
	w *TagModule
}

func (c *pickCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "Pick",
		Usage: "Picks a random item.",
	}
}

func (c *pickCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	gID := bot.SBatoi(info.ID)
	var stmt *sql.Stmt
	var params []interface{}
	arg := "any tag"
	if len(args) < 1 || args[0] == "*" {
		var err error
		stmt, err = c.w.prepStatement("SELECT I.Content FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID INNER JOIN items I ON M.Item = I.ID WHERE T.Guild = ? GROUP BY I.Content ORDER BY RAND() LIMIT 1", "", info.Bot.DB)
		if err != nil {
			return bot.ReturnError(err)
		}
		params = []interface{}{gID}
	} else {
		//arg = msg.Content[indices[0]:]
		arg = args[0]
		if !ValidateWhereClause(arg) {
			return "```\nMismatched parentheses!```", false, nil
		}
		clause, tags := BuildWhereClause(arg)
		tagIDs, err := getTagIDs(tags, gID, info.Bot.DB)
		if err != nil {
			return bot.ReturnError(err)
		}

		stmt, err = c.w.prepStatement("SELECT I.Content FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID INNER JOIN items I ON M.Item = I.ID WHERE "+clause+" AND T.Guild = ? GROUP BY I.Content ORDER BY RAND() LIMIT 1", arg, info.Bot.DB)
		if err != nil {
			return bot.ReturnError(err)
		}

		params = make([]interface{}, len(tagIDs), len(tagIDs))
		params = append(params, gID)
		for k, v := range tagIDs {
			params[k] = v
		}
	}

	var item string
	q := stmt.QueryRow(params...)
	err := q.Scan(&item)
	if err == sql.ErrNoRows {
		return fmt.Sprintf("```No items were returned by %s!```", arg), false, nil
	} else if err != nil {
		return bot.ReturnError(err)
	}
	return info.Sanitize(item, bot.CleanMentions|bot.CleanPings), false, nil
}
func (c *pickCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Picks a random item from the given tags and displays it. If no tags are given, picks an item at random from all possible tags.",
		Params: []bot.CommandUsageParam{
			{Name: "tag(s)", Desc: "An arbitrary tag search using the syntax `tag1|(tag2+(-tag3))`, which translates to `tag1 OR (tag2 AND NOT tag3)`. If set to *, picks from all tags.", Optional: true},
		},
	}
}

type newCommand struct {
}

func (c *newCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "New",
		Usage:     "Creates a new tag.",
		Sensitive: true,
	}
}
func (c *newCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou have to provide a new tag name.```", false, nil
	}

	tag := strings.ToLower(args[0])
	if strings.ContainsAny(tag, "+-|()*") {
		return "```\nDon't make tag names with +, -, |, *, or () in them, dumbass!```", false, nil
	}

	gID := bot.SBatoi(info.ID)
	err := info.Bot.DB.CreateTag(tag, gID)
	if err == bot.ErrDuplicateEntry {
		return "```\nThat tag already exists!```", false, nil
	} else if err != nil {
		return bot.ReturnError(err)
	}
	return "```\nCreated the " + tag + " tag.```", false, nil
}
func (c *newCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Creates a new tag with the given name, provided the tag does not already exist.",
		Params: []bot.CommandUsageParam{
			{Name: "tag", Desc: "The name of the new tag. You can use spaces if you put it in quotes, but this is not recommended.", Optional: false},
		},
	}
}

type deleteCommand struct {
}

func (c *deleteCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "Delete",
		Usage:     "Deletes a tag.",
		Sensitive: true,
	}
}

func (c *deleteCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou have to provide a tag name.```", false, nil
	}

	tag := strings.ToLower(args[0])
	gID := bot.SBatoi(info.ID)
	if _, err := info.Bot.DB.GetTag(tag, gID); err == sql.ErrNoRows {
		return "```\nThat tag doesn't exist!```", false, nil
	} else if err != nil {
		return bot.ReturnError(err)
	}

	if err := info.Bot.DB.DeleteTag(tag, gID); err != nil {
		return bot.ReturnError(err)
	}
	return "```\nDeleted the " + tag + " tag.```", false, nil
}
func (c *deleteCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Deletes a tag with the given name, removing the tag from any items it was associated with, and deleting any orphaned items with no tags.",
		Params: []bot.CommandUsageParam{
			{Name: "tag", Desc: "The name of the tag.", Optional: false},
		},
	}
}

type searchTagsCommand struct {
	w *TagModule
}

func (c *searchTagsCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "SearchTags",
		Usage: "Searches tags for a string.",
	}
}

func (c *searchTagsCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nYou have to provide a tag query (in quotes if there are spaces).```", false, nil
	}
	arg := args[0]
	clause := "1=1"
	tags := []string{}
	if arg != "*" {
		if !ValidateWhereClause(arg) {
			return "```\nMismatched parentheses!```", false, nil
		}
		clause, tags = BuildWhereClause(arg)
	}

	gID := bot.SBatoi(info.ID)
	tagIDs, err := getTagIDs(tags, gID, info.Bot.DB)
	if err != nil {
		return bot.ReturnError(err)
	}

	params := make([]interface{}, len(tagIDs), len(tagIDs))
	for k, v := range tagIDs {
		params[k] = v
	}
	params = append(params, gID)

	if len(args) < 2 {
		stmt, err := c.w.prepStatement("SELECT COUNT(DISTINCT I.Content) FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID INNER JOIN items I ON M.Item = I.ID WHERE ("+clause+") AND T.Guild = ?", arg, info.Bot.DB)
		if err == nil {
			var count uint64
			q := stmt.QueryRow(params...)
			err := q.Scan(&count)
			if err == sql.ErrNoRows {
				return fmt.Sprintf("```No items were returned by %s!```", arg), false, nil
			} else if err == nil {
				return fmt.Sprintf("```Number of items matching %s: %v```", arg, count), false, nil
			}
		}
		return bot.ReturnError(err)
	}

	search := "%" + msg.Content[indices[1]:] + "%"
	params = append(params, search)

	q, err := c.w.execStatement("SELECT I.Content FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID INNER JOIN items I ON M.Item = I.ID WHERE ("+clause+") AND T.Guild = ? AND I.Content LIKE ? GROUP BY I.Content", arg, info.Bot.DB, params...)
	if err != nil {
		return bot.ReturnError(err)
	}

	defer q.Close()
	r := make([]string, 0, 3)
	for q.Next() {
		var p string
		if err := q.Scan(&p); err == nil {
			r = append(r, p)
		}
	}

	if len(r) == 0 {
		return fmt.Sprintf("```\nNo items match %s that contain that string!```", arg), false, nil
	}

	count := len(r)
	if len(r) > maxTagResults {
		r = r[:maxTagResults]
	}
	s := strings.Join(r, "\n")
	s = strings.Replace(s, "```", "\\`\\`\\`", -1)
	s = strings.Replace(s, "[](/", "[\u200B](/", -1)
	return fmt.Sprintf("```\nAll %v items satisfying %s that contain that string:\n%s```", count, arg, s), len(r) > bot.MaxPublicLines, nil
}
func (c *searchTagsCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Returns all items that match both the string specified and the tags provided. If no string is specified, just counts how many items match the query.",
		Params: []bot.CommandUsageParam{
			{Name: "tags", Desc: "An arbitrary tag search using the syntax `tag1|(tag2+(-tag3))`, which translates to `tag1 OR (tag2 AND NOT tag3)`, or * to search all tags.", Optional: false},
			{Name: "arbitrary string", Desc: "Arbitrary string to search for.", Optional: true},
		},
	}
}

type importCommand struct {
}

func (c *importCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "Import",
		Usage:     "Imports all items matching a tag from another server.",
		Sensitive: true,
	}
}

func (c *importCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !info.Bot.DB.CheckStatus() {
		return "```\nA temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```\nNo source server provided.```", false, nil
	}

	other := []*bot.GuildInfo{}
	str := args[0]
	matches := bot.UserRegex.FindStringSubmatch(str)
	if len(matches) < 2 || len(matches[1]) == 0 {
		exact := false
		func() {
			info.Bot.GuildsLock.RLock()
			defer info.Bot.GuildsLock.RUnlock()
			for _, v := range info.Bot.Guilds {
				if strings.Compare(strings.ToLower(v.Name), strings.ToLower(str)) == 0 {
					if !exact {
						other = []*bot.GuildInfo{}
						exact = true
					}
					other = append(other, v)
				} else if !exact {
					if strings.Contains(strings.ToLower(v.Name), strings.ToLower(str)) {
						other = append(other, v)
					}
				}
			}
		}()
	} else {
		func() {
			info.Bot.GuildsLock.RLock()
			defer info.Bot.GuildsLock.RUnlock()
			for _, v := range info.Bot.Guilds {
				if strings.Compare(v.ID, matches[1]) == 0 {
					other = append(other, v)
				}
			}
		}()
	}

	if len(other) > 1 {
		names := make([]string, len(other), len(other))
		for i := range names {
			names[i] = other[i].Name
		}
		return fmt.Sprintf("```Could be any of the following servers: \n%s```", info.Sanitize(strings.Join(names, "\n"), bot.CleanCodeBlock)), len(names) > bot.MaxPublicLines, nil
	}
	if len(other) < 1 {
		return fmt.Sprintf("```Could not find any server matching %s!```", args[0]), false, nil
	}
	if !other[0].Config.Basic.Importable {
		return "```\nThat server has not made their tags importable by other servers. If this is a public server, you can ask a moderator on that server to run \"" + info.Config.Basic.CommandPrefix + "setconfig importable true\" if they wish to make their tags public.```", false, nil
	}

	if len(args) < 2 {
		return "```\nNo source tag provided.```", false, nil
	}
	source := args[1]
	target := source
	if len(args) > 2 {
		target = args[2]
	}

	otherGID := bot.SBatoi(other[0].ID)
	sourceTag, err := info.Bot.DB.GetTag(source, otherGID)
	if err == sql.ErrNoRows {
		return fmt.Sprintf("```The source tag (%s) does not exist on the source server (%s)!```", source, other[0].Name), false, nil
	} else if err != nil {
		return bot.ReturnError(err)
	}

	gID := bot.SBatoi(info.ID)
	targetTag, err := info.Bot.DB.GetTag(target, gID)
	if err == sql.ErrNoRows {
		return fmt.Sprintf("```The target tag (%s) does not exist on this server! Please manually create this tag using !new if you actually intended this.```", target), false, nil
	} else if err != nil {
		return bot.ReturnError(err)
	}

	if err = info.Bot.DB.ImportTag(sourceTag, targetTag); err != nil {
		return bot.ReturnError(err)
	}

	return fmt.Sprintf("```Successfully merged \"%s\" from %s into \"%s\" on this server.```", source, other[0].Name, target), false, nil
}
func (c *importCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Adds all elements from the source tag on the source server to the target tag on this server. If no target is specified, attempts to copy all items into a tag of the same name as the source. Example: ```" + info.Config.Basic.CommandPrefix + "import Manechat cool notcool```",
		Params: []bot.CommandUsageParam{
			{Name: "source server", Desc: "The exact name of the source server to copy from, or the server ID in the form <@9999999999>", Optional: false},
			{Name: "source tag", Desc: "Name of the tag to copy from on the source server.", Optional: false},
			{Name: "target tag", Desc: "The target tag to copy to on this server. If omitted, defaults to the source tag name.", Optional: true},
		},
	}
}
