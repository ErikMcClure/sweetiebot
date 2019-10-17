package sweetiebot

const (
	STRING_INVALID_COMMAND                   = iota
	STRING_PM_FAILURE                        = iota
	STRING_CHECK_PM                          = iota
	STRING_DATABASE_ERROR                    = iota
	STRING_NO_SERVER                         = iota
	STRING_COMMANDS_LIMIT                    = iota
	STRING_COMMAND_LIMIT                     = iota
	STRING_SETUP_MESSAGE                     = iota
	STRING_SPAM_DESCRIPTION                  = iota
	STRING_SPAM_ERROR_UNSILENCING            = iota
	STRING_SPAM_UNSILENCING                  = iota
	STRING_SPAM_EMBEDDED_URLS                = iota
	STRING_SPAM_TRUNCATED                    = iota
	STRING_SPAM_KILLING_SPAMMER_DETAIL       = iota
	STRING_SPAM_AUTOBANNED_REASON            = iota
	STRING_SPAM_BAN_ALERT                    = iota
	STRING_SPAM_ERROR_RETRIEVE_MESSAGES      = iota
	STRING_SPAM_WILL_BE_UNSILENCED           = iota
	STRING_SPAM_SILENCE_ALERT                = iota
	STRING_SPAM_KILLING_SPAMMER              = iota
	STRING_SPAM_REASON_MESSAGES              = iota
	STRING_SPAM_REASON_FILES                 = iota
	STRING_SPAM_REASON_IMAGES                = iota
	STRING_SPAM_REASON_PINGS                 = iota
	STRING_SPAM_REASON_LENGTH                = iota
	STRING_SPAM_REASON_NEWLINES              = iota
	STRING_SPAM_REASON_COPY                  = iota
	STRING_SPAM_GUILD_NOT_FOUND              = iota
	STRING_SPAM_VERIFICATION_LEVEL_ERROR     = iota
	STRING_SPAM_LOCKDOWN_DISENGAGE_FAILURE   = iota
	STRING_SPAM_LOCKDOWN_DISENGAGE           = iota
	STRING_SPAM_USER_JOINED                  = iota
	STRING_SPAM_JOINED_APPEND                = iota
	STRING_SPAM_RAIDSILENCE_ALL_POSTFIX      = iota
	STRING_SPAM_RAIDSILENCE_ENGAGED          = iota
	STRING_SPAM_RAID_DETECTED                = iota
	STRING_SPAM_LOCKDOWN_ENGAGE_FAILURE      = iota
	STRING_SPAM_LOCKDOWN_ENGAGE              = iota
	STRING_SPAM_RAIDSILENCE_USAGE            = iota
	STRING_SPAM_RAIDSILENCE_ARGS_ERROR       = iota
	STRING_SPAM_RAIDSILENCE_ARGS             = iota
	STRING_SPAM_RAIDSILENCE_DATABASE_ERROR   = iota
	STRING_SPAM_RAIDSILENCE_SET_RAID         = iota
	STRING_SPAM_RAIDSILENCE_DETECTION        = iota
	STRING_SPAM_RAIDSILENCE_SET              = iota
	STRING_SPAM_RAIDSILENCE_DESCRIPTION      = iota
	STRING_SPAM_RAIDSILENCE_DESCRIPTION_NAME = iota
	STRING_SPAM_WIPE_USAGE                   = iota
	STRING_SPAM_WIPE_ARG_ERROR               = iota
	STRING_SPAM_WIPE_PM_ERROR                = iota
	STRING_SPAM_WIPE_CHANNEL_ERROR           = iota
	STRING_SPAM_WIPE_NO_MESSAGES             = iota
	STRING_SPAM_WIPE_RETRIEVAL_ERROR         = iota
	STRING_SPAM_WIPE_DELETED                 = iota
	STRING_SPAM_WIPE_DESCRIPTION             = iota
	STRING_SPAM_WIPE_CHANNEL                 = iota
	STRING_SPAM_WIPE_MESSAGES                = iota
	STRING_SPAM_PRESSURE_USAGE               = iota
	STRING_SPAM_PRESSURE_ARG_ERROR           = iota
	STRING_SPAM_PRESSURE_DESCRIPTION         = iota
	STRING_SPAM_PRESSURE_USER                = iota
	STRING_SPAM_RAID_USAGE                   = iota
	STRING_SPAM_RAID_NONE                    = iota
	STRING_SPAM_RAID_USERS                   = iota
	STRING_SPAM_RAID_DESCRIPTION             = iota
	STRING_SPAM_BANRAID_USAGE                = iota
	STRING_SPAM_BANRAID_REASON               = iota
	STRING_SPAM_BANRAID_RESULT               = iota
	STRING_SPAM_BANRAID_DESCRIPTION          = iota
	STRING_USERS_BAN_MOD_ERROR               = iota
	STRING_USERS_SILENCE_USAGE               = iota
	STRING_USERS_SILENCE_ARG_ERROR           = iota
	STRING_USERS_SILENCE_ERROR               = iota
	STRING_USERS_SILENCE_MOD_ERROR           = iota
	STRING_USERS_SILENCE_ALREADY_SILENCED    = iota
	STRING_USERS_SILENCE_WILL_BE_UNSILENCED  = iota
	STRING_USERS_SILENCE_REASON              = iota
	STRING_USERS_SILENCE                     = iota
	STRING_USERS_SILENCE_DESCRIPTION         = iota
	STRING_USERS_SILENCE_USER                = iota
	STRING_USERS_SILENCE_DURATION            = iota
	STRING_USERS_UNSILENCE_USAGE             = iota
	STRING_USERS_UNSILENCE_ARG_ERROR         = iota
	STRING_USERS_UNSILENCE_ERROR             = iota
	STRING_USERS_UNSILENCE_MOD_ERROR         = iota
	STRING_USERS_UNSILENCE                   = iota
	STRING_USERS_UNSILENCE_DESCRIPTION       = iota
	STRING_USERS_UNSILENCE_USER              = iota
)

// System-wide string map that can be substituted at runtime
var StringMap = map[int]string{
	STRING_INVALID_COMMAND:                   "Sorry, %s is not a valid command.\nFor a list of valid commands, type %shelp.",
	STRING_PM_FAILURE:                        "I tried to send you a Private Message, but it failed! Try PMing me the command directly.",
	STRING_CHECK_PM:                          "```\nCheck your Private Messages for my reply!```",
	STRING_DATABASE_ERROR:                    "```\nA temporary database error means I can't process any private message commands right now.```",
	STRING_NO_SERVER:                         "```\nCannot determine what server you belong to! Use !defaultserver to set which server I should use when you PM me.```",
	STRING_COMMANDS_LIMIT:                    "You can't input more than %v commands every %s!%s",
	STRING_COMMAND_LIMIT:                     "You can only run that command once every %s!%s",
	STRING_SETUP_MESSAGE:                     "You haven't set up the bot yet! Run the %ssetup command first and follow the instructions.",
	STRING_SPAM_DESCRIPTION:                  "Tracks all channels it is active on for spammers. Each message someone sends generates \"pressure\", which decays rapidly. Long messages, messages with links, or messages with pings will generate more pressure. If a user generates too much pressure, they will be silenced and the moderators notified. Also detects groups of people joining at the same time and alerts the moderators of a potential raid.\n\nTo force this module to ignore a specific channel, use this command: `%ssetconfig modules.channels spam ! #channelname`. If the bot is silencing everyone, you should re-run `%ssetup OVERRIDE` to reset the spam configuration. If you want to have a containment channel where silenced members can talk, use `%ssetconfig welcomechannel #channelname`.\n\n**IF THE BOT IS OVERWHELMED BY A RAID, FOLLOW THESE INSTRUCTIONS CAREFULLY:** Due to rate limits, the bot can be overwhelmed by spammers using hundreds of different accounts. As a last resort, you can tell the bot to **ban everyone who has sent their first message in the past 3 minutes** by running this command: `%sbannewcomers`. Only use this as a **last resort**, as it can easily ban people who joined and were caught up in the raid.",
	STRING_SPAM_ERROR_UNSILENCING:            "```\nError unsilencing member: %v```",
	STRING_SPAM_UNSILENCING:                  "```\nUnsilenced %v.```",
	STRING_SPAM_EMBEDDED_URLS:                "\nEmbedded URLs: ",
	STRING_SPAM_TRUNCATED:                    "... [truncated]",
	STRING_SPAM_KILLING_SPAMMER_DETAIL:       "Killing spammer %s (pressure: %v -> %v). Last message sent on #%s in %s: \n%s%s",
	STRING_SPAM_AUTOBANNED_REASON:            "Autobanned for %v in the welcome channel.",
	STRING_SPAM_BAN_ALERT:                    "Alert: <@%v> was banned for %v in the welcome channel.",
	STRING_SPAM_ERROR_RETRIEVE_MESSAGES:      "Error encountered while attempting to retrieve messages: ",
	STRING_SPAM_WILL_BE_UNSILENCED:           ", or they will be unsilenced automatically in %v",
	STRING_SPAM_SILENCE_ALERT:                "Alert: <@%v> was silenced for %v. Please investigate%v",
	STRING_SPAM_KILLING_SPAMMER:              "Killing spammer %v",
	STRING_SPAM_REASON_MESSAGES:              "spamming too many messages",
	STRING_SPAM_REASON_FILES:                 "attaching too many files",
	STRING_SPAM_REASON_IMAGES:                "spamming too many images",
	STRING_SPAM_REASON_PINGS:                 "pinging too many people",
	STRING_SPAM_REASON_LENGTH:                "sending a really long message",
	STRING_SPAM_REASON_NEWLINES:              "using too many newlines",
	STRING_SPAM_REASON_COPY:                  "copy+pasting the same message",
	STRING_SPAM_GUILD_NOT_FOUND:              "Guild cannot be found in state?!",
	STRING_SPAM_VERIFICATION_LEVEL_ERROR:     "The verification level is at %v instead of %v, which means it was manually changed by someone other than %v, so it has not been restored.",
	STRING_SPAM_LOCKDOWN_DISENGAGE_FAILURE:   "Could not disengage lockdown! Make sure you've given the %v role the Manage Server permission, you'll have to manually restore it yourself this time.",
	STRING_SPAM_LOCKDOWN_DISENGAGE:           "Lockdown disengaged, server verification levels restored.",
	STRING_SPAM_USER_JOINED:                  "%v (joined: %v)",
	STRING_SPAM_RAIDSILENCE_ALL_POSTFIX:      "Use `%vraidsilenceall to silence them!",
	STRING_SPAM_RAIDSILENCE_ENGAGED:          "RaidSilence has been engaged and the following users silenced:",
	STRING_SPAM_RAID_DETECTED:                " Possible Raid Detected! ",
	STRING_SPAM_LOCKDOWN_ENGAGE_FAILURE:      "Could not engage lockdown! Make sure you've given %v the Manage Server permission, or disable the lockdown entirely via `%vsetconfig spam.lockdownduration 0`.",
	STRING_SPAM_LOCKDOWN_ENGAGE:              "Lockdown engaged! Server verification level will be reset in %v seconds. This lockdown can be manually ended via `%vraidsilence off/alert/log`.",
	STRING_SPAM_RAIDSILENCE_USAGE:            "Toggle raid silencing.",
	STRING_SPAM_RAIDSILENCE_ARGS_ERROR:       "```\nYou must provide a raid silence level (either all, raid, or off).```",
	STRING_SPAM_RAIDSILENCE_ARGS:             "```\nOnly all, raid, and off are valid raid silence levels.```",
	STRING_SPAM_RAIDSILENCE_DATABASE_ERROR:   "```\nRaidSilence was engaged, but a database error prevents me from retroactively applying it!```",
	STRING_SPAM_RAIDSILENCE_SET_RAID:         "```\nRaid silence level set to %v.```",
	STRING_SPAM_RAIDSILENCE_DETECTION:        "```\nDetected a recent raid. All users from the raid have been silenced:",
	STRING_SPAM_RAIDSILENCE_SET:              "```\nRaid silence level set to %v.```",
	STRING_SPAM_RAIDSILENCE_DESCRIPTION:      "Toggles silencing new members during raids. This does not affect spam detection, only new members joining the server.",
	STRING_SPAM_RAIDSILENCE_DESCRIPTION_NAME: "`all` will always silence all new members. `raid` will only silence new members if a raid is detected, up to `spam.raidtime*2` seconds after the raid is detected. `off` disables raid silencing.",
	STRING_SPAM_WIPE_USAGE:                   "Wipes a given channel",
	STRING_SPAM_WIPE_ARG_ERROR:               "```\nYou must specify the duration.```",
	STRING_SPAM_WIPE_PM_ERROR:                "```\nCan't delete messages in a PM!```",
	STRING_SPAM_WIPE_CHANNEL_ERROR:           "```\nThat channel isn't on this server!```",
	STRING_SPAM_WIPE_NO_MESSAGES:             "```\nThere's no point deleting 0 messages!.```",
	STRING_SPAM_WIPE_RETRIEVAL_ERROR:         "```\nError retrieving messages. Are you sure you gave %v a channel that exists? This won't work in PMs! %v```",
	STRING_SPAM_WIPE_DELETED:                 "Deleted %v messages in <#%s>.",
	STRING_SPAM_WIPE_DESCRIPTION:             "Removes all messages in a channel sent within the last N seconds, or remove the last N messages if 'm' is appended to the number. Examples: ```\n%swipe 23m``` ```\n%swipe #channel 10```",
	STRING_SPAM_WIPE_CHANNEL:                 "The channel to delete from. You must use the #channel format so discord actually highlights the channel, otherwise it won't work. If omitted, uses the current channel",
	STRING_SPAM_WIPE_MESSAGES:                "Specifies the number of seconds to look back. The command deletes all messages sent up to this many seconds ago. If you append 'm' to this number, it will instead delete exactly that many messages.",
	STRING_SPAM_PRESSURE_USAGE:               "Gets a user's pressure.",
	STRING_SPAM_PRESSURE_ARG_ERROR:           "```\nYou must provide a user to search for.```",
	STRING_SPAM_PRESSURE_DESCRIPTION:         "Gets the current spam pressure of a user.",
	STRING_SPAM_PRESSURE_USER:                "User to retrieve pressure from.",
	STRING_SPAM_RAID_USAGE:                   "Lists users in most recent raid.",
	STRING_SPAM_RAID_NONE:                    "```\nNo raid has occurred within the past %s.```",
	STRING_SPAM_RAID_USERS:                   "Users in latest raid: ",
	STRING_SPAM_RAID_DESCRIPTION:             "Lists all users that are considered part of the most recent raid, if there was one.",
	STRING_SPAM_BANRAID_USAGE:                "Bans all users in most recent raid.",
	STRING_SPAM_BANRAID_REASON:               "Banned by %s#%s via the %sbanraid command.",
	STRING_SPAM_BANRAID_RESULT:               "```\nBanned %v users. The ban log will reflect who ran this command.```",
	STRING_SPAM_BANRAID_DESCRIPTION:          "Bans all users that are considered part of the most recent raid, if there was one. Use %vgetraid to check who will be banned before using this command.",
	STRING_USERS_BAN_MOD_ERROR:               "```\nCan't ban %s because they're a moderator or an admin!```",
	STRING_USERS_SILENCE_USAGE:               "Silences a user.",
	STRING_USERS_SILENCE_ARG_ERROR:           "```\nYou must provide a user to silence.```",
	STRING_USERS_SILENCE_ERROR:               "```\nError occurred trying to silence %s: %s```",
	STRING_USERS_SILENCE_MOD_ERROR:           "```\nCannot silence %s because they're a moderator or admin!```",
	STRING_USERS_SILENCE_ALREADY_SILENCED:    "```\n%v is already silenced!```",
	STRING_USERS_SILENCE_WILL_BE_UNSILENCED:  "```\n%s is already silenced, and will be unsilenced in %s```",
	STRING_USERS_SILENCE_REASON:              " because %v",
	STRING_USERS_SILENCE:                     "```\nSilenced %s%s.```",
	STRING_USERS_SILENCE_DESCRIPTION:         "Silences the given user.",
	STRING_USERS_SILENCE_USER:                "A ping of the user, or simply their name.",
	STRING_USERS_SILENCE_DURATION:            "If the keyword `for:` is used after the username, looks for a duration of the form `for: 50 MINUTES` and creates an unsilence event that will be fired after that much time has passed from now.",
	STRING_USERS_UNSILENCE_USAGE:             "Unsilences a user.",
	STRING_USERS_UNSILENCE_ARG_ERROR:         "```\nYou must provide a user to unsilence.```",
	STRING_USERS_UNSILENCE_ERROR:             "```\nError unsilencing member: %v```",
	STRING_USERS_UNSILENCE_MOD_ERROR:         "```\nCannot unsilence %s because they are a mod or admin. Remove the status yourself!```",
	STRING_USERS_UNSILENCE:                   "```\nUnsilenced %v.```",
	STRING_USERS_UNSILENCE_DESCRIPTION:       "Unsilences the given user.",
	STRING_USERS_UNSILENCE_USER:              "A ping of the user, or simply their name.",
}
