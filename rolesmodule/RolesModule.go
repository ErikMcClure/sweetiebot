package rolesmodule

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	bot "../sweetiebot"
	"github.com/blackhole12/discordgo"
)

var errNotUserAssignable = errors.New("not a user-assignable role")
var pingRegex = regexp.MustCompile("<[@#](!|&)?[0-9]+>")

// RolesModule contains commands for manipulating user-assignable roles.
type RolesModule struct {
}

// New RolesModule
func New() *RolesModule {
	return &RolesModule{}
}

// Name of the module
func (w *RolesModule) Name() string {
	return "Roles"
}

// Commands in the module
func (w *RolesModule) Commands() []bot.Command {
	return []bot.Command{
		&addRoleCommand{},
		&createRoleCommand{},
		&joinRoleCommand{},
		&listRoleCommand{},
		&leaveRoleCommand{},
		&removeRoleCommand{},
		&deleteRoleCommand{},
	}
}

// Description of the module
func (w *RolesModule) Description(info *bot.GuildInfo) string {
	return "Contains commands for manipulating user-assignable roles. Pay close attention to the difference between `addrole` vs. `createrole` and `removerole` vs. `deleterole`. Users who simply want to join or leave a user-assignable role should be using the `joinrole` and `leaverole` commands, not any other commands."
}

// OnGuildRoleDelete keeps things tidy by making sure no deleted roles are user-assignable
func (w *RolesModule) OnGuildRoleDelete(info *bot.GuildInfo, r *discordgo.GuildRoleDelete) {
	delete(info.Config.Users.Roles, bot.DiscordRole(r.RoleID))
	info.SaveConfig()
}

// GetUserAssignableRole gets a role by it's name, but only if it's user-assignable
func GetUserAssignableRole(role string, info *bot.GuildInfo) (*discordgo.Role, error) {
	r, err := bot.GetRoleByName(role, info)
	if err != nil {
		return nil, err
	}
	_, ok := info.Config.Users.Roles[bot.DiscordRole(r.ID)]
	if !ok || bot.DiscordRole(r.ID) == info.Config.Basic.ModRole { // Make sure you can't screw up badly enough to let silenced users unsilence themselves
		return nil, errNotUserAssignable
	}
	return r, nil
}

// GetRoleByNameOrPing gets a role by its name or by pinging it
func GetRoleByNameOrPing(role string, info *bot.GuildInfo) (*discordgo.Role, error) {
	if bot.RoleRegex.MatchString(role) {
		r, err := bot.ParseRole(role, nil)
		if err != nil {
			return nil, err
		}
		if r == bot.RoleEmpty || r == bot.RoleExclusion {
			return nil, errNotUserAssignable
		}
		_, ok := info.Config.Users.Roles[r]
		if !ok || r == info.Config.Basic.ModRole {
			return nil, errNotUserAssignable
		}

		roles, err := info.Bot.DG.GuildRoles(info.ID)
		if err != nil {
			return nil, err
		}
		for _, v := range roles {
			if r.Equals(v.ID) {
				return v, nil
			}
		}
		return nil, bot.ErrRoleNoMatch
	}
	return GetUserAssignableRole(role, info)
}

type createRoleCommand struct {
}

func (c *createRoleCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "CreateRole",
		Usage:     "Creates a new user-assignable role.",
		Sensitive: true,
	}
}

func (c *createRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou must provide a new role name!```", false, nil
	}
	g, _ := info.GetGuild()
	role := msg.Content[indices[0]:]
	if pingRegex.MatchString(role) {
		return "```\nDon't do that. Give the role a name that isn't a ping.```", false, nil
	}
	roles := bot.FindRole(role, g)
	if len(roles) > 0 {
		return "```\nThat role already exists! Use " + info.Config.Basic.CommandPrefix + "addrole to make it user-assignable if it isn't already.```", false, nil
	}

	r, err := info.Bot.DG.GuildRoleCreate(info.ID)
	if err == nil {
		r, err = info.Bot.DG.GuildRoleEdit(info.ID, r.ID, role, 0, false, 0, true)
	}
	if err != nil {
		return "```\nCould not create role! " + err.Error() + "```", false, nil
	}
	if len(info.Config.Users.Roles) == 0 {
		info.Config.Users.Roles = make(map[bot.DiscordRole]bool)
	}
	info.Config.Users.Roles[bot.DiscordRole(r.ID)] = true
	info.SaveConfig()
	return fmt.Sprintf("```Created the %s role. By default, it has no permissions and can be pinged by users, but you can change these settings if you like. Use "+info.Config.Basic.CommandPrefix+"deleterole to delete it.```", r.Name), false, nil
}
func (c *createRoleCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Creates a new role and makes it user-assignable.",
		Params: []bot.CommandUsageParam{
			{Name: "name/id", Desc: "Name of a new role that doesn't already exist.", Optional: false},
		},
	}
}

type addRoleCommand struct {
}

func (c *addRoleCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "AddRole",
		Usage:     "Sets a role as user-assignable.",
		Sensitive: true,
	}
}

func (c *addRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou must either ping or provide the name of an existing role!```", false, nil
	}
	g, _ := info.GetGuild()
	role, err := bot.ParseRole(msg.Content[indices[0]:], g)
	if err != nil {
		return bot.ReturnError(err)
	}
	if role == bot.RoleEmpty || role == bot.RoleExclusion {
		return "```\nThat's not a role! Use " + info.Config.Basic.CommandPrefix + "createroll to create a new role.```", false, nil
	}
	if info.Config.Basic.ModRole == role {
		return "```\nYou can't make the moderator role user-assignable you maniac!```", false, nil
	}
	_, ok := info.Config.Users.Roles[role]
	if ok {
		return "```\nThat role is already user-assignable!```", false, nil
	}
	roles, err := info.Bot.DG.GuildRoles(info.ID)
	if err != nil {
		return "```\nCould not get roles! + " + err.Error() + "```", false, nil
	}
	for _, v := range roles {
		if role.Equals(v.ID) {
			if len(info.Config.Users.Roles) == 0 {
				info.Config.Users.Roles = make(map[bot.DiscordRole]bool)
			}
			info.Config.Users.Roles[role] = true
			info.SaveConfig()
			return "```\n" + v.Name + " is now a user-assignable role. You can change the name or permissions of the role without worrying about messing something up.```", false, nil
		}
	}
	return "```\nThat's not a role in this server! Are you sure the role exists and you spelled it correctly?```", false, nil
}
func (c *addRoleCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Adds an existing role to the list of user-assignable roles.",
		Params: []bot.CommandUsageParam{
			{Name: "name/id", Desc: "Name or ping of an existing role.", Optional: false},
		},
	}
}

type joinRoleCommand struct {
}

func (c *joinRoleCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "JoinRole",
		Usage: "Add yourself to a user-assignable role.",
	}
}

func (c *joinRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou must provide a role name!```", false, nil
	}
	r, err := GetUserAssignableRole(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(err)
	}
	hasrole := info.UserHasRole(bot.DiscordUser(msg.Author.ID), bot.DiscordRole(r.ID))
	err = info.ResolveRoleAddError(info.Bot.DG.GuildMemberRoleAdd(info.ID, msg.Author.ID, r.ID)) // Try adding the role no matter what, just in case discord screwed up
	if hasrole {
		return "```\nYou already have that role.```", false, nil
	}
	if err != nil {
		return "```\nError adding role! " + err.Error() + "```", false, nil
	}
	pingable := ""
	if r.Mentionable {
		pingable = " You may ping everyone in the role via @" + r.Name + ", but do so sparingly."
	}
	return fmt.Sprintf("```You now have the %s role. You can remove yourself from the role via "+info.Config.Basic.CommandPrefix+"leaverole %s, or list everyone in it via "+info.Config.Basic.CommandPrefix+"listrole %s.%s```", r.Name, r.Name, r.Name, pingable), false, nil
}
func (c *joinRoleCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Adds you to a role, provided it is user-assignable. You should use the name of the role, not a ping, so you don't piss everyone off.",
		Params: []bot.CommandUsageParam{
			{Name: "name", Desc: "Name of the role you want to join.", Optional: false},
		},
	}
}

type listRoleCommand struct {
}

func (c *listRoleCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "ListRole",
		Usage: "Lists everyone in a user-assignable role.",
	}
}

func (c *listRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		roles, err := info.Bot.DG.GuildRoles(info.ID)
		if err != nil {
			return fmt.Sprintf("```Error getting roles: %s```", err.Error()), false, nil
		}
		s := []string{}
		for _, v := range roles {
			_, ok := info.Config.Users.Roles[bot.DiscordRole(v.ID)]
			if ok {
				s = append(s, v.Name)
			}
		}
		return "```\nAll available user-assignable roles: " + strings.Join(s, ", ") + "```", false, nil
	}
	r, err := GetUserAssignableRole(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(err)
	}

	guild, err := info.GetGuild()
	if err != nil {
		return "```\nGuild not in state?!```", false, nil
	}
	info.Bot.DG.State.RLock() // We can't just extract the pointer here because append() is used on guild.Members, so we copy the entire slice instead
	members := append([]*discordgo.Member{}, guild.Members...)
	info.Bot.DG.State.RUnlock()
	out := []string{}
	for _, v := range members {
		if info.UserHasRole(bot.DiscordUser(v.User.ID), bot.DiscordRole(r.ID)) {
			if len(v.Nick) > 0 {
				out = append(out, v.Nick)
			} else {
				out = append(out, v.User.Username)
			}
		}
	}
	if len(out) == 0 {
		return "```\nThat role has no users in it!```", false, nil
	}

	return fmt.Sprintf("```Members of %s: %s```", r.Name, strings.Join(out, ", ")), false, nil
}
func (c *listRoleCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Lists everyone that has the given role, provided it is user-assignable. You should use the name of the role, not a ping, so you don't piss everyone off.",
		Params: []bot.CommandUsageParam{
			{Name: "name", Desc: "Name of the role.", Optional: false},
		},
	}
}

type leaveRoleCommand struct {
}

func (c *leaveRoleCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:  "LeaveRole",
		Usage: "Remove yourself from a user-assignable role.",
	}
}
func (c *leaveRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou must provide a role name!```", false, nil
	}
	r, err := GetUserAssignableRole(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(err)
	}
	hasrole := info.UserHasRole(bot.DiscordUser(msg.Author.ID), bot.DiscordRole(r.ID))
	err = info.ResolveRoleAddError(info.Bot.DG.GuildMemberRoleRemove(info.ID, msg.Author.ID, r.ID)) // Try removing it no matter what in case discord screwed up
	if !hasrole {
		return "```\nYou don't have that role.```", false, nil
	}
	if err != nil {
		return "```\nError removing role! " + err.Error() + "```", false, nil
	}
	return fmt.Sprintf("```You no longer have the %s role.```", r.Name), false, nil
}
func (c *leaveRoleCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Removes you from a role, provided it is user-assignable and you are in it. You should use the name of the role, not a ping, so you don't piss everyone off.",
		Params: []bot.CommandUsageParam{
			{Name: "name", Desc: "Name of the role you want to leave.", Optional: false},
		},
	}
}

type removeRoleCommand struct {
}

func (c *removeRoleCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "RemoveRole",
		Usage:     "Remove a role from list of user-assignable roles.",
		Sensitive: true,
	}
}

func (c *removeRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou must provide either a role name, or a role ping.```", false, nil
	}

	r, err := GetRoleByNameOrPing(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(info.ResolveRoleAddError(err))
	}
	delete(info.Config.Users.Roles, bot.DiscordRole(r.ID))
	info.SaveConfig()
	return fmt.Sprintf("```The %s role is no longer user-assignable, but it has NOT been deleted! Use "+info.Config.Basic.CommandPrefix+"deleterole to delete a user-assignable role.```", r.Name), false, nil
}
func (c *removeRoleCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Removes a role from the list of user-assignable roles, but DOES NOT DELETE IT. If you want to also delete the role, use " + info.Config.Basic.CommandPrefix + "deleterole.",
		Params: []bot.CommandUsageParam{
			{Name: "name", Desc: "Name or ping of the role you no longer want user-assignable.", Optional: false},
		},
	}
}
func (c *removeRoleCommand) UsageShort() string {
	return "Remove a role from list of user-assignable roles."
}

type deleteRoleCommand struct {
}

func (c *deleteRoleCommand) Info() *bot.CommandInfo {
	return &bot.CommandInfo{
		Name:      "DeleteRole",
		Usage:     "Deletes a user-assignable role.",
		Sensitive: true,
	}
}

func (c *deleteRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *bot.GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```\nYou must provide either a role name, or a role ping.```", false, nil
	}
	r, err := GetRoleByNameOrPing(msg.Content[indices[0]:], info)
	if err != nil {
		return bot.ReturnError(info.ResolveRoleAddError(err))
	}
	err = info.Bot.DG.GuildRoleDelete(info.ID, r.ID)
	if err != nil {
		return "```\nError deleting role! " + err.Error() + "```", false, nil
	}
	return fmt.Sprintf("```The %s role has been deleted from the server.```", r.Name), false, nil
}
func (c *deleteRoleCommand) Usage(info *bot.GuildInfo) *bot.CommandUsage {
	return &bot.CommandUsage{
		Desc: "Completely deletes a user-assignable role. Cannot be used to delete roles that aren't user-assignable to prevent accidents.",
		Params: []bot.CommandUsageParam{
			{Name: "name", Desc: "Name or ping of the role you want to delete.", Optional: false},
		},
	}
}
