package sweetiebot

import "github.com/blackhole12/discordgo"
import "fmt"
import "strings"

type RolesModule struct {
}

// Name of the module
func (w *RolesModule) Name() string {
	return "Roles"
}

// Commands in the module
func (w *RolesModule) Commands() []Command {
	return []Command{
		&addRoleCommand{},
		&joinRoleCommand{},
		&listRoleCommand{},
		&leaveRoleCommand{},
		&removeRoleCommand{},
		&deleteRoleCommand{},
	}
}

// Description of the module
func (w *RolesModule) Description() string {
	return "Contains commands for manipulating user-assignable roles."
}

// OnGuildRoleDelete keeps things tidy by making sure no deleted roles are user-assignable
func (w *RolesModule) OnGuildRoleDelete(info *GuildInfo, r *discordgo.GuildRoleDelete) {
	delete(info.config.Users.Roles, SBatoi(r.RoleID))
	info.SaveConfig()
}

// GetRoleByName gets a role by its name
func GetRoleByName(role string, info *GuildInfo) (*discordgo.Role, error) {
	roles, err := sb.dg.GuildRoles(info.ID)
	role = strings.ToLower(role)
	if err != nil {
		info.LogError("GuildRoles(): ", err)
		return nil, err
	}
	for _, v := range roles {
		if strings.ToLower(v.Name) == role {
			return v, nil
		}
	}
	return nil, nil
}

// GetUserAssignableRole gets a role by it's name, but only if it's user-assignable
func GetUserAssignableRole(role string, info *GuildInfo) (*discordgo.Role, uint64, string) {
	r, err := GetRoleByName(role, info)
	if err != nil {
		return nil, 0, "```Error: Couldn't get roles!```"
	}
	if r == nil {
		return nil, 0, "```That's not a role name!```"
	}
	id := SBatoi(r.ID)
	_, ok := info.config.Users.Roles[id]
	if !ok || id == info.config.Spam.SilentRole || id == info.config.Basic.AlertRole { // Make sure you can't screw up badly enough to let silenced users unsilence themselves
		return nil, 0, "```That's not a user-assignable role!```"
	}
	return r, id, ""
}

// GetRoleByNameOrPing gets a role by its name or by pinging it
func GetRoleByNameOrPing(role string, info *GuildInfo) (*discordgo.Role, uint64, string) {
	if mentionregex.MatchString(role) {
		role = StripPing(role)
		id := SBatoi(role)
		if id == 0 {
			return nil, 0, "```Invalid role ping!```"
		}
		_, ok := info.config.Users.Roles[id]
		if !ok || id == info.config.Spam.SilentRole || id == info.config.Basic.AlertRole {
			return nil, 0, "```That's not a user-assignable role!```"
		}
		roles, err := sb.dg.GuildRoles(info.ID)
		if err != nil {
			return nil, 0, "```Couldn't get roles! + " + err.Error() + "```"
		}
		for _, v := range roles {
			if v.ID == role {
				return v, id, ""
			}
		}
		return nil, 0, "```That's not a role in this server! Are you sure you pinged a role, and not a user?```"
	}
	return GetUserAssignableRole(role, info)
}

type addRoleCommand struct {
}

func (c *addRoleCommand) Name() string {
	return "AddRole"
}

func (c *addRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You must provide either a new role name, or a ping of an existing role!```", false, nil
	}
	if mentionregex.MatchString(args[0]) {
		role := StripPing(args[0])
		r := SBatoi(role)
		if r == 0 {
			return "```Invalid role ping!```", false, nil
		}
		if r == info.config.Basic.AlertRole {
			return "```You can't make the moderator role user-assignable you maniac!```", false, nil
		}
		if r == info.config.Spam.SilentRole {
			return "```You can't make the silence role user-assignable you maniac!```", false, nil
		}
		_, ok := info.config.Users.Roles[r]
		if ok {
			return "```That role is already user-assignable!```", false, nil
		}
		roles, err := sb.dg.GuildRoles(info.ID)
		if err != nil {
			return "```Could not get roles! + " + err.Error() + "```", false, nil
		}
		for _, v := range roles {
			if v.ID == role {
				info.config.Users.Roles[r] = true
				info.SaveConfig()
				return "```" + v.Name + " is now a user-assignable role. You can change the name or permissions of the role without worrying about messing something up.```", false, nil
			}
		}
		return "```That's not a role in this server! Are you sure you pinged a role, and not a user?```", false, nil
	}

	role := msg.Content[indices[0]:]
	check, err := GetRoleByName(role, info)
	if err != nil {
		return "```Error: Couldn't get roles!```", false, nil
	}
	if check != nil {
		return "```That's already a role name in this server. If you want to set an existing role as user-assignable, you must ping the role.```", false, nil
	}
	r, err := sb.dg.GuildRoleCreate(info.ID)
	if err == nil {
		r, err = sb.dg.GuildRoleEdit(info.ID, r.ID, role, 0, false, 0, true)
	}
	if err != nil {
		return "```Could not create role! " + err.Error() + "```", false, nil
	}
	info.config.Users.Roles[SBatoi(r.ID)] = true
	info.SaveConfig()
	return fmt.Sprintf("```Created the %s role. By default, it has no permissions and can be pinged by users, but you can change these settings if you like. Use "+info.config.Basic.CommandPrefix+"deleterole to delete it.```", r.Name), false, nil
}
func (c *addRoleCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Either creates a new role, or adds an existing role to Sweetie's list of user-assignable roles. To create a new role, simply put in the name of the new role. To set an existing role as user-assignable, ping the role instead, via @role.",
		Params: []CommandUsageParam{
			{Name: "name/id", Desc: "Name of the new role, or a ping of an existing role.", Optional: false},
		},
	}
}
func (c *addRoleCommand) UsageShort() string { return "Creates or sets a role as user-assignable." }

type joinRoleCommand struct {
}

func (c *joinRoleCommand) Name() string {
	return "JoinRole"
}

func (c *joinRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You must provide a role name!```", false, nil
	}
	role := msg.Content[indices[0]:]
	r, _, e := GetUserAssignableRole(role, info)
	if len(e) > 0 {
		return e, false, nil
	}
	hasrole := info.UserHasRole(msg.Author.ID, r.ID)
	err := sb.dg.GuildMemberRoleAdd(info.ID, msg.Author.ID, r.ID) // Try adding the role no matter what, just in case discord screwed up
	if hasrole {
		return "```You already have that role.```", false, nil
	}
	if err != nil {
		return "```Error adding role! " + err.Error() + "```", false, nil
	}
	pingable := ""
	if r.Mentionable {
		pingable = " You may ping everyone in the role via @" + r.Name + ", but do so sparingly."
	}
	return fmt.Sprintf("```You now have the %s role. You can remove yourself from the role via "+info.config.Basic.CommandPrefix+"leaverole %s, or list everyone in it via "+info.config.Basic.CommandPrefix+"listrole %s.%s```", r.Name, r.Name, r.Name, pingable), false, nil
}
func (c *joinRoleCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Adds you to a role, provided it is user-assignable. You should use the name of the role, not a ping, so you don't piss everyone off.",
		Params: []CommandUsageParam{
			{Name: "name", Desc: "Name of the role you want to join.", Optional: false},
		},
	}
}
func (c *joinRoleCommand) UsageShort() string { return "Add yourself to a user-assignable role." }

type listRoleCommand struct {
}

func (c *listRoleCommand) Name() string {
	return "ListRole"
}

func (c *listRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		roles, err := sb.dg.GuildRoles(info.ID)
		if err != nil {
			return fmt.Sprintf("```Error getting roles: %s```", err.Error()), false, nil
		}
		s := []string{}
		for _, v := range roles {
			_, ok := info.config.Users.Roles[SBatoi(v.ID)]
			if ok {
				s = append(s, v.Name)
			}
		}
		return "```All available user-assignable roles: " + strings.Join(s, ", ") + "```", false, nil
	}
	role := msg.Content[indices[0]:]
	r, _, e := GetUserAssignableRole(role, info)
	if len(e) > 0 {
		return e, false, nil
	}

	guild, err := sb.dg.State.Guild(info.ID)
	if err != nil {
		return "```Guild not in state?!```", false, nil
	}
	sb.dg.State.RLock()
	defer sb.dg.State.RUnlock()
	out := []string{}
	for _, v := range guild.Members {
		if info.UserHasRole(v.User.ID, r.ID) {
			if len(v.Nick) > 0 {
				out = append(out, v.Nick)
			} else {
				out = append(out, v.User.Username)
			}
		}
	}
	if len(out) == 0 {
		return "```That role has no users in it!```", false, nil
	}

	return fmt.Sprintf("```Members of %s: %s```", r.Name, strings.Join(out, ", ")), false, nil
}
func (c *listRoleCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Lists everyone that has the given role, provided it is user-assignable. You should use the name of the role, not a ping, so you don't piss everyone off.",
		Params: []CommandUsageParam{
			{Name: "name", Desc: "Name of the role.", Optional: false},
		},
	}
}
func (c *listRoleCommand) UsageShort() string { return "Lists everyone in a user-assignable role." }

type leaveRoleCommand struct {
}

func (c *leaveRoleCommand) Name() string {
	return "LeaveRole"
}

func (c *leaveRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You must provide a role name!```", false, nil
	}
	role := msg.Content[indices[0]:]
	r, _, e := GetUserAssignableRole(role, info)
	if len(e) > 0 {
		return e, false, nil
	}
	hasrole := info.UserHasRole(msg.Author.ID, r.ID)
	err := sb.dg.GuildMemberRoleRemove(info.ID, msg.Author.ID, r.ID) // Try removing it no matter what in case discord screwed up
	if !hasrole {
		return "```You don't have that role.```", false, nil
	}
	if err != nil {
		return "```Error removing role! " + err.Error() + "```", false, nil
	}
	return fmt.Sprintf("```You no longer have the %s role.```", r.Name), false, nil
}
func (c *leaveRoleCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Removes you from a role, provided it is user-assignable and you are in it. You should use the name of the role, not a ping, so you don't piss everyone off.",
		Params: []CommandUsageParam{
			{Name: "name", Desc: "Name of the role you want to leave.", Optional: false},
		},
	}
}
func (c *leaveRoleCommand) UsageShort() string { return "Remove yourself from a user-assignable role." }

type removeRoleCommand struct {
}

func (c *removeRoleCommand) Name() string {
	return "RemoveRole"
}

func (c *removeRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You must provide either a role name, or a role ping.```", false, nil
	}
	role := msg.Content[indices[0]:]
	r, id, e := GetRoleByNameOrPing(role, info)
	if len(e) > 0 {
		return e, false, nil
	}
	delete(info.config.Users.Roles, id)
	info.SaveConfig()
	return fmt.Sprintf("```The %s role is no longer user-assignable, but it has NOT been deleted! Use "+info.config.Basic.CommandPrefix+"deleterole to delete a user-assignable role.```", r.Name), false, nil
}
func (c *removeRoleCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Removes a role from the list of user-assignable roles, but DOES NOT DELETE IT. If you want to also delete the role, use " + info.config.Basic.CommandPrefix + "deleterole.",
		Params: []CommandUsageParam{
			{Name: "name", Desc: "Name or ping of the role you no longer want user-assignable.", Optional: false},
		},
	}
}
func (c *removeRoleCommand) UsageShort() string {
	return "Remove a role from list of user-assignable roles."
}

type deleteRoleCommand struct {
}

func (c *deleteRoleCommand) Name() string {
	return "DeleteRole"
}

func (c *deleteRoleCommand) Process(args []string, msg *discordgo.Message, indices []int, info *GuildInfo) (string, bool, *discordgo.MessageEmbed) {
	if len(args) < 1 {
		return "```You must provide either a role name, or a role ping.```", false, nil
	}
	role := msg.Content[indices[0]:]
	r, _, e := GetRoleByNameOrPing(role, info)
	if len(e) > 0 {
		return e, false, nil
	}
	err := sb.dg.GuildRoleDelete(info.ID, r.ID)
	if err != nil {
		return "```Error deleting role! " + err.Error() + "```", false, nil
	}
	return fmt.Sprintf("```The %s role has been deleted from the server.```", r.Name), false, nil
}
func (c *deleteRoleCommand) Usage(info *GuildInfo) *CommandUsage {
	return &CommandUsage{
		Desc: "Completely deletes a user-assignable role. Cannot be used to delete roles that aren't user-assignable to prevent accidents.",
		Params: []CommandUsageParam{
			{Name: "name", Desc: "Name or ping of the role you want to delete.", Optional: false},
		},
	}
}
func (c *deleteRoleCommand) UsageShort() string { return "Deletes a user-assignable role." }
