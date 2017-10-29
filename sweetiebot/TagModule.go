package sweetiebot

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/blackhole12/discordgo"
)

// TagModule contains commands for manipulating Sweetie Bot's tags
type TagModule struct {
	Cache map[string]*sql.Stmt
}

// Name of the module
func (w *TagModule) Name() string {
	return "Tag"
}

// Commands in the module
func (w *TagModule) Commands() []Command {
	return []Command{
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
	return "Contains commands for manipulating Sweetie Bot's tags."
}
func (w *TagModule) prepStatement(query string, tags string) (*sql.Stmt, error) {
	stmt, ok := w.Cache[query]
	if !ok {
		var err error
		stmt, err = sb.DB.Prepare(query)
		if err != nil {
			//return nil, fmt.Errorf("```Invalid tag expression: %s```", err.Error())
			return nil, fmt.Errorf("```Invalid tag expression: %s```", tags)
		}
		w.Cache[query] = stmt
	}
	return stmt, nil
}

func (w *TagModule) execStatement(query string, tags string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := w.prepStatement(query, tags)
	if err != nil {
		return nil, err
	}

	q, err := stmt.Query(args...)
	if sb.DB.CheckError("Query: "+query, err) {
		return nil, fmt.Errorf("```Error executing statement: %s```", err.Error())
	}

	return q, nil
}

func getTagIDs(tags []string, guild uint64) ([]uint64, error) {
	tagIDs := make([]uint64, len(tags), len(tags))
	for k, v := range tags {
		id, err := sb.DB.GetTag(v, guild)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("```The %s tag does not exist!```", v)
		}
		tagIDs[k] = id
	}
	return tagIDs, nil
}

// BuildWhereClause returns a valid mySQL WHERE clause and argument list from a logical tag expression
func BuildWhereClause(arg string) (string, []string) {
	args := tagargregex.FindAllString(arg, -1)
	arg = tagargregex.ReplaceAllString(arg, "M.Item IN (SELECT Item FROM itemtags WHERE Tag = ?)")
	arg = strings.Replace(arg, "-", " NOT ", -1)
	arg = strings.Replace(arg, "+", " AND ", -1)
	arg = strings.Replace(arg, "|", " OR ", -1)
	return arg, args
}

type addCommand struct {
}

func (c *addCommand) Name() string {
	return "Add"
}
func (c *addCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.DB.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```No tags given```", false, nil
	}
	if len(args) < 2 {
		return "```Can't add empty string!```", false, nil
	}

	gID := SBatoi(info.ID)
	if count, _ := sb.DB.CountItems(gID); count >= maxUniqueItems {
		return fmt.Sprintf("```Can't have more than %v unique items in a server!```", maxUniqueItems), false, nil
	}

	tags := strings.Split(args[0], "+")
	tagIDs, err := getTagIDs(tags, gID)
	if err != nil {
		return err.Error(), false, nil
	}

	item := msg.Content[indices[1]:]
	id, err := sb.DB.AddItem(item)
	if err != nil && err != ErrDuplicateEntry {
		return "Error: " + err.Error(), false, nil
	}
	for _, v := range tagIDs {
		sb.DB.AddTag(id, v)
	}

	for k := range tags {
		count, err := sb.DB.CountTag(tagIDs[k])
		if err == nil {
			tags[k] = fmt.Sprintf("%s: %v items", tags[k], count)
		}
	}
	itemtags := sb.DB.GetItemTags(id, gID)
	return fmt.Sprintf("```\n%s: %s\n---\n%s```", PartialSanitize(item), PartialSanitize(strings.Join(itemtags, ", ")), PartialSanitize(strings.Join(tags, "\n"))), false, nil
}
func (c *addCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Adds [arbitrary string] to [tags]. If the item already exists, simply adds the tags to the existing item.",
		Params: []CommandUsageParam{
			{Name: "tag(s)", Desc: "The name of a tag. Specify multiple tags with \"tag1+tag2\" (with quotes if there are spaces).", Optional: false},
			{Name: "arbitrary string", Desc: "Arbitrary string to add tags to. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}
func (c *addCommand) UsageShort() string { return "Adds an item to a set of tags." }

type getCommand struct {
}

func (c *getCommand) Name() string {
	return "GetTags"
}
func (c *getCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.DB.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```No item given```", false, nil
	}

	gID := SBatoi(info.ID)

	item := msg.Content[indices[0]:]
	id, err := sb.DB.GetItem(item)
	if err == sql.ErrNoRows {
		return "```That item has no tags.```", false, nil
	}

	tags := sb.DB.GetItemTags(id, gID)
	if len(tags) == 0 {
		return "```That item has no tags.```", false, nil
	}
	return fmt.Sprintf("```%s: %s```", PartialSanitize(item), PartialSanitize(strings.Join(tags, ", "))), false, nil
}
func (c *getCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Returns all tags associated with [arbitrary string], or tells you the item has no tags",
		Params: []CommandUsageParam{
			{Name: "arbitrary string", Desc: "Arbitrary string to get the tags of. Quotes aren't necessary, but cannot be empty.", Optional: false},
		},
	}
}
func (c *getCommand) UsageShort() string { return "Gets all tags an item has, if any." }

type removeCommand struct {
}

func (c *removeCommand) Name() string {
	return "Remove"
}
func (c *removeCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.DB.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```You must specify what item you want to remove!```", false, nil
	}

	gID := SBatoi(info.ID)
	if len(args) < 2 || args[0] == "*" {
		item := msg.Content[indices[0]:]
		if args[0] == "*" && len(args) > 1 {
			item = msg.Content[indices[1]:]
		}
		id, err := sb.DB.GetItem(item)
		if err == sql.ErrNoRows {
			return fmt.Sprintf("```%s doesn't exist!```", item), false, nil
		}
		if err == nil {
			if len(sb.DB.GetItemTags(id, gID)) == 0 {
				return fmt.Sprintf("```%s doesn't exist!```", item), false, nil
			}
			err = sb.DB.RemoveItem(id, gID)
		}
		if err != nil {
			return fmt.Sprintf("```Error removing %s: %s```", item, err.Error()), false, nil
		}
		return fmt.Sprintf("```Removed %s from all tags.```", PartialSanitize(item)), false, nil
	}

	tags := strings.Split(args[0], "+")
	tagIDs, err := getTagIDs(tags, gID)
	if err != nil {
		return err.Error(), false, nil
	}

	item := msg.Content[indices[1]:]
	id, err := sb.DB.GetItem(item)
	if err != nil {
		return fmt.Sprintf("```That item doesn't exist!```"), false, nil
	}
	for _, v := range tagIDs {
		sb.DB.RemoveTag(id, v)
	}

	itemtags := sb.DB.GetItemTags(id, gID)
	if len(itemtags) == 0 {
		return fmt.Sprintf("```Removed %s (item has no tags).```", PartialSanitize(item)), false, nil
	}

	for k := range tags {
		count, err := sb.DB.CountTag(tagIDs[k])
		if err == nil {
			tags[k] = fmt.Sprintf("%s: %v items", tags[k], count)
		}
	}
	return fmt.Sprintf("```%s: %s\n---\n%s```", PartialSanitize(item), PartialSanitize(strings.Join(itemtags, ", ")), PartialSanitize(strings.Join(tags, "\n"))), false, nil
}
func (c *removeCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Removes [arbitrary string] from [tags], unless [tags] is *, in which case it removes it from all tags.",
		Params: []CommandUsageParam{
			{Name: "tag(s)", Desc: "The name of a tag. Specify multiple tags with \"tag1+tag2\" (with quotes if there are spaces), or use * to remove it from all tags.", Optional: true},
			{Name: "arbitrary string", Desc: "Arbitrary string to remove from the given tags. If this has spaces, don't omit the tag argument - use * instead.", Optional: false},
		},
	}
}
func (c *removeCommand) UsageShort() string { return "Removes tags from an item." }

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

func (c *tagsCommand) Name() string {
	return "Tags"
}

// ShowAllTags builds an embed message containing all tags
func ShowAllTags(message string, info *GuildInfo) *discordgo.MessageEmbed {
	tags := sb.DB.GetTags(SBatoi(info.ID))
	items, _ := sb.DB.CountItems(SBatoi(info.ID))
	fields := make(memberFields, len(tags), len(tags))

	for k, v := range tags {
		fields[k] = &discordgo.MessageEmbedField{Name: v.Name, Value: fmt.Sprintf("%v items", v.Count), Inline: true}
	}
	sort.Sort(fields)
	return &discordgo.MessageEmbed{
		Type: "rich",
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://github.com/blackhole12/sweetiebot",
			Name:    "Sweetie Bot Tags",
			IconURL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%v/%s.jpg", sb.SelfID, sb.SelfAvatar),
		},
		Description: message + fmt.Sprintf(" Tags: %v. Unique Items: %v", len(tags), items),
		Color:       0x3e92e5,
		Fields:      fields,
	}
}
func (c *tagsCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.DB.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "", false, ShowAllTags("No tag specified.", info)
	}

	arg := msg.Content[indices[0]:]
	clause, tags := BuildWhereClause(arg)
	gID := SBatoi(info.ID)
	tagIDs, err := getTagIDs(tags, gID)
	if err != nil {
		return err.Error(), false, nil
	}

	params := make([]interface{}, len(tagIDs), len(tagIDs))
	for k, v := range tagIDs {
		params[k] = v
	}
	params = append(params, gID)
	q, err := c.w.execStatement("SELECT I.Content FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID INNER JOIN items I ON M.Item = I.ID WHERE "+clause+" AND T.Guild = ? GROUP BY I.Content", arg, params...)
	if err != nil {
		return err.Error(), false, nil
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
	if len(r) > maxTagResults && !info.UserHasRole(msg.Author.ID, SBitoa(info.config.Basic.AlertRole)) {
		s = strings.Join(r[:maxTagResults], "\n")
		arg += " (truncated)"
	} else {
		s = strings.Join(r, "\n")
	}
	s = strings.Replace(s, "```", "\\`\\`\\`", -1)
	s = strings.Replace(s, "[](/", "[\u200B](/", -1)
	return fmt.Sprintf("```\n%v items satisfy %s:\n%s```", len(r), arg, s), len(r) > maxPublicLines, nil
}
func (c *tagsCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists all the tags in this server, or the contents of a specific tag search string.",
		Params: []CommandUsageParam{
			{Name: "tag(s)", Desc: "An optional arbitrary tag search using the syntax `tag1|(tag2+(-tag3))`, which translates to `tag1 OR (tag2 AND NOT tag3)`.", Optional: true},
		},
	}
}
func (c *tagsCommand) UsageShort() string { return "Lists all tags." }

type pickCommand struct {
	w *TagModule
}

func (c *pickCommand) Name() string {
	return "Pick"
}
func (c *pickCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.DB.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	gID := SBatoi(info.ID)
	var stmt *sql.Stmt
	var params []interface{}
	arg := "any tag"
	if len(args) < 1 || args[0] == "*" {
		var err error
		stmt, err = c.w.prepStatement("SELECT I.Content FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID INNER JOIN items I ON M.Item = I.ID WHERE T.Guild = ? GROUP BY I.Content ORDER BY RAND() LIMIT 1", "")
		if err != nil {
			return err.Error(), false, nil
		}
		params = []interface{}{gID}
	} else {
		//arg = msg.Content[indices[0]:]
		arg = args[0]
		clause, tags := BuildWhereClause(arg)
		tagIDs, err := getTagIDs(tags, gID)
		if err != nil {
			return err.Error(), false, nil
		}

		stmt, err = c.w.prepStatement("SELECT I.Content FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID INNER JOIN items I ON M.Item = I.ID WHERE "+clause+" AND T.Guild = ? GROUP BY I.Content ORDER BY RAND() LIMIT 1", arg)
		if err != nil {
			return err.Error(), false, nil
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
	}
	return ReplaceAllMentions(item), false, nil
}
func (c *pickCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Picks a random item from the given tags and displays it. If no tags are given, picks an item at random from all possible tags.",
		Params: []CommandUsageParam{
			{Name: "tag(s)", Desc: "An arbitrary tag search using the syntax `tag1|(tag2+(-tag3))`, which translates to `tag1 OR (tag2 AND NOT tag3)`. If set to *, picks from all tags.", Optional: true},
		},
	}
}
func (c *pickCommand) UsageShort() string { return "Picks a random item." }

type newCommand struct {
}

func (c *newCommand) Name() string {
	return "New"
}
func (c *newCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.DB.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```You have to provide a new tag name.```", false, nil
	}

	tag := strings.ToLower(args[0])
	if strings.ContainsAny(tag, "+-|()*") {
		return "```Don't make tag names with +, -, |, *, or () in them, dumbass!```", false, nil
	}

	gID := SBatoi(info.ID)
	err := sb.DB.CreateTag(tag, gID)
	if err == ErrDuplicateEntry {
		return "```That tag already exists!```", false, nil
	} else if err != nil {
		return "```Error creating tag: " + err.Error() + "```", false, nil
	}
	return "```Created the " + tag + " tag.```", false, nil
}
func (c *newCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Creates a new tag with the given name, provided the tag does not already exist.",
		Params: []CommandUsageParam{
			{Name: "tag", Desc: "The name of the new tag. You can use spaces if you put it in quotes, but this is not recommended.", Optional: false},
		},
	}
}
func (c *newCommand) UsageShort() string { return "Creates a new tag." }

type deleteCommand struct {
}

func (c *deleteCommand) Name() string {
	return "Delete"
}
func (c *deleteCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.DB.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```You have to provide a tag name.```", false, nil
	}

	tag := strings.ToLower(args[0])
	gID := SBatoi(info.ID)
	if _, err := sb.DB.GetTag(tag, gID); err == sql.ErrNoRows {
		return "```That tag doesn't exist!```", false, nil
	}

	if err := sb.DB.DeleteTag(tag, gID); err != nil {
		return "```Error deleting tag: " + err.Error() + "```", false, nil
	}
	return "```Deleted the " + tag + " tag.```", false, nil
}
func (c *deleteCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Deletes a tag with the given name, removing the tag from any items it was associated with, and deleting any orphaned items with no tags.",
		Params: []CommandUsageParam{
			{Name: "tag", Desc: "The name of the tag.", Optional: false},
		},
	}
}
func (c *deleteCommand) UsageShort() string { return "Deletes a tag." }

type searchTagsCommand struct {
	w *TagModule
}

func (c *searchTagsCommand) Name() string {
	return "SearchTags"
}
func (c *searchTagsCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.DB.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```You have to provide a tag query (in quotes if there are spaces).```", false, nil
	}
	arg := args[0]
	clause := "1=1"
	tags := []string{}
	if arg != "*" {
		clause, tags = BuildWhereClause(arg)
	}

	gID := SBatoi(info.ID)
	tagIDs, err := getTagIDs(tags, gID)
	if err != nil {
		return err.Error(), false, nil
	}

	params := make([]interface{}, len(tagIDs), len(tagIDs))
	for k, v := range tagIDs {
		params[k] = v
	}
	params = append(params, gID)

	if len(args) < 2 {
		stmt, err := c.w.prepStatement("SELECT COUNT(DISTINCT I.Content) FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID INNER JOIN items I ON M.Item = I.ID WHERE ("+clause+") AND T.Guild = ?", arg)
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
		return err.Error(), false, nil
	}

	search := "%" + msg.Content[indices[1]:] + "%"
	params = append(params, search)

	q, err := c.w.execStatement("SELECT I.Content FROM itemtags M INNER JOIN tags T ON M.Tag = T.ID INNER JOIN items I ON M.Item = I.ID WHERE ("+clause+") AND T.Guild = ? AND I.Content LIKE ? GROUP BY I.Content", arg, params...)
	if err != nil {
		return err.Error(), false, nil
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

	s := strings.Join(r, "\n")
	s = strings.Replace(s, "```", "\\`\\`\\`", -1)
	s = strings.Replace(s, "[](/", "[\u200B](/", -1)
	return fmt.Sprintf("```\nAll items satisfying %s that contain that string:\n%s```", arg, s), false, nil
}
func (c *searchTagsCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Returns all items that match both the string specified and the tags provided. If no string is specified, just counts how many items match the query.",
		Params: []CommandUsageParam{
			{Name: "tags", Desc: "An arbitrary tag search using the syntax `tag1|(tag2+(-tag3))`, which translates to `tag1 OR (tag2 AND NOT tag3)`, or * to search all tags.", Optional: false},
			{Name: "arbitrary string", Desc: "Arbitrary string to search for.", Optional: true},
		},
	}
}
func (c *searchTagsCommand) UsageShort() string { return "Searches tags for a string." }

type importCommand struct {
}

func (c *importCommand) Name() string {
	return "Import"
}
func (c *importCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if !sb.DB.CheckStatus() {
		return "```A temporary database outage is preventing this command from being executed.```", false, nil
	}
	if len(args) < 1 {
		return "```No source server provided.```", false, nil
	}

	other := []*GuildInfo{}
	str := args[0]
	exact := false
	func() {
		sb.guildsLock.RLock()
		defer sb.guildsLock.RUnlock()
		for _, v := range sb.guilds {
			if strings.Compare(strings.ToLower(v.Name), strings.ToLower(str)) == 0 {
				if !exact {
					other = []*GuildInfo{}
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

	if len(other) > 1 {
		names := make([]string, len(other), len(other))
		for i := 0; i < len(other); i++ {
			names[i] = other[i].Name
		}
		return fmt.Sprintf("```Could be any of the following servers: \n%s```", PartialSanitize(strings.Join(names, "\n"))), len(names) > maxPublicLines, nil
	}
	if len(other) < 1 {
		return fmt.Sprintf("```Could not find any server matching %s!```", args[0]), false, nil
	}
	if !other[0].config.Basic.Importable {
		return "```That server has not made their tags importable by other servers. If this is a public server, you can ask a moderator on that server to run \"" + info.config.Basic.CommandPrefix + "setconfig importable true\" if they wish to make their tags public.```", false, nil
	}

	if len(args) < 2 {
		return "```No source tag provided.```", false, nil
	}
	source := args[1]
	target := source
	if len(args) > 2 {
		target = args[2]
	}

	otherGID := SBatoi(other[0].ID)
	sourceTag, err := sb.DB.GetTag(source, otherGID)
	if err != nil {
		return fmt.Sprintf("```The source tag (%s) does not exist on the source server (%s)!```", source, other[0].Name), false, nil
	}

	gID := SBatoi(info.ID)
	targetTag, err := sb.DB.GetTag(target, gID)
	if err != nil {
		return fmt.Sprintf("```The target tag (%s) does not exist on this server! Please manually create this tag using !new if you actually intended this.```", target), false, nil
	}

	if err = sb.DB.ImportTag(sourceTag, targetTag); err != nil {
		return fmt.Sprintf("```Error importing tags: %s```", err.Error()), false, nil
	}

	return fmt.Sprintf("```Successfully merged \"%s\" from %s into \"%s\" on this server.```", source, other[0].Name, target), false, nil
}
func (c *importCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Adds all elements from the source tag on the source server to the target tag on this server. If no target is specified, attempts to copy all items into a tag of the same name as the source. Example: ```" + info.config.Basic.CommandPrefix + "import ExampleServer cool notcool```",
		Params: []CommandUsageParam{
			{Name: "source server", Desc: "The exact name of the source server to copy from.", Optional: false},
			{Name: "source tag", Desc: "Name of the tag to copy from on the source server.", Optional: false},
			{Name: "target tag", Desc: "The target tag to copy to on this server. If omitted, defaults to the source tag name.", Optional: true},
		},
	}
}
func (c *importCommand) UsageShort() string {
	return "Imports all items matching a tag from another server."
}
