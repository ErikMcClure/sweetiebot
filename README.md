# Sweetie Bot
Sweetie Bot is an administration bot for Discord servers. Her primary function is anti-spam, by detecting potential spammers, silencing them, and deleting their messages. This helps immunize the chat against bot raids. She also keeps a log of the chat and its users, and provides a command to find the last message that pinged a given user.

### To add Sweetie Bot to your server, use [this link](https://discordapp.com/oauth2/authorize?client_id=171790139712864257&scope=bot&permissions=535948390).

**If you have questions about Sweetie Bot, please join its [support channel](https://discord.gg/t2gVQvN)**

**If you use Sweetie Bot, consider [contributing to its Patreon](https://www.patreon.com/erikmcclure) to help pay for hosting and maintenence costs.**

## Adding Sweetiebot To Your Server

A limited version of sweetiebot can be added to any server. Simply follow [this link](https://discordapp.com/oauth2/authorize?client_id=171790139712864257&scope=bot&permissions=535948390) to add her to your server. The limited version of sweetiebot does not have a chatlog, which means !search is unavailable. The status change loop and !setstatus are also disabled. All other commands and modules still function, however. 

## Configuration

Upon being added to a server, Sweetiebot will begin with all her commands and modules disabled, pending configuration. **Only the owner of a server can setup Sweetie Bot. No one else can, even if they have admin rights.** This is to ensure that members of the server cannot abuse the bot during the configuration process - the owner of the server can run any command, even if it's disabled (except for !update, !removealias and !announce, which can only be run by the bot owner). Sweetiebot should send you a PM when she is first added with instructions on how to run the `!setup` command. **You must run `!setup` to configure Sweetie Bot for your server!** `!setup` takes the following parameters, in order:

* **Mod Role** should be set to a role shared by all moderators. It is used to alert moderators and also allows the moderators to bypass command restrictions imposed by certain modules.
* **Mod Channel** should be set to whatever channel the moderators would like to recieve notifications on, such as potential raids, spammers being silenced, etc.
* **Log Channel** [OPTIONAL] should be set to a channel that recieves log messages about errors and initialization. Usually this channel is only visible to the bot and the moderators.

For example: `!setup @Mods #staff-lounge #Bot-log`

`!setup` will automatically restrict all sensitive commands to `alertrole` and enable a default set of modules. **You must PING the role or channel that you are adding to the bot!** To ensure your moderator role can be pinged: Go to Server Settings -> Roles and select your mod role, then make sure "Allow anyone to @mention this role" is checked. You won't be able to properly configure Sweetie Bot over PM, because you won't be able to specify the channels. Make sure you configure her in a place where Discord autocompletes `#channelname` for you and highlights it.

**DO NOT GIVE SWEETIE BOT ADMINISTRATIVE PERMISSIONS OR THE ABILITY TO PING EVERYONE!** Sweetie bot does not and will never attempt to filter `@everyone` pings, because if you don't want her to be able to ping everyone, you shouldn't give her the ability to do so in the first place. Sweetie bot only requires the following permissions: `Manage Roles`, `Ban Members`, `Manage Messages`, plus all the default read/write permissions given to everyone.

Additional configuration is optional via `!setconfig` depending on what features of the bot are being used.

### Using !setconfig
Basic configuration parameters can be set with `!setconfig <parameter name> <value>`. To get a list of configuration parameters, use `!getconfig`. To output the current value of a paramter, use `!getconfig <paramater name>`.

Certain configuration parameters are more complex. They can either be maps, lists, or maps of lists. This type information is listed when using `!getconfig`. Parameters that are lists simply take multiple values instead of one. Setting a list parameter to a set of values will *replace* the current list of values.

    !setconfig <list parameter> <value 1> <value 2> <value 3> <etc...>
    !setconfig bored.commands !drop "!pick cute"

Maps are a set of key-value pairs. Unlike lists, each invocation of `!setconfig` will set just a single key-value pair and won't affect any others. If a key already exists, the value of that key will be overwritten. If the value is set to "", the key will be deleted.

    !setconfig <map parameter> <key> <value>
    !setconfig basic.aliases listbucket list
    !setconfig basic.aliases listbucket ""

Maps of lists simply map their keys to entire lists of values instead of just one value. The syntax is similar to setting a single map value:

    !setconfig <maplist parameter> <key> <value 1> <value 2> <value 3> <etc...>
    !setconfig modules.commandchannels roll #channel1 #channel2

However, to delete a value from a maplist, you simply call `!setconfig modules.commandchannels <key>` with no values at all:

    !setconfig modules.commandchannels roll

### Basic
* **IgnoreInvalidCommands:** If true, Sweetie Bot won't display an error if a nonsensical command is used. This helps her co-exist with other bots that also use the `!` prefix.
* **Importable:** If true, the collections on this server will be importable into another server where sweetie is.
* **AlertRole:** This is intended to point at a moderator role shared by all admins and moderators of the server for notification purposes.
* **ModChannel:** This should point at the hidden moderator channel, or whatever channel moderates want to be notified on.
* **FreeChannels [list]:** This is a list of all channels that are exempt from rate limiting. Usually set to the dedicated `#botabuse` channel in a server.
* **BotChannel:** Allows you to designate a particular channel for Sweetie Bot to point users to if they try to send too many commands at once. This channel is usually also included in `Basic.FreeChannels`.
* **Aliases [map]:** Can be used to redirect commands, such as making `!listroles` call the `!listrole` command. Useful for making shortcuts. Example: `!setconfig Basic.Aliases kawaii "pick cute"` sets an alias mapping `!kawaii arg1...` to `!pick cute arg1...`, preserving all arguments that are passed to the alias.
* **Collections [maplist]:** All the collections used by sweetiebot. Manipulate it via `!add` and `!remove`
* **ListenToBots:** If true, sweetiebot will process bot messages and allow them to run commands. Bots can never trigger anti-spam. Defaults to false.

### Modules
* **Channels [maplist]:** A mapping of what channels a given module can operate on. If no mapping is given, a module operates on all channels. If "!" is included as a channel, it switches from a whitelist to a blacklist, enabling you to exclude certain channels instead of allow certain channels.
* **Disabled [list]:** A list of disabled modules.
* **CommandRoles [maplist]:** A map of which roles are allowed to run which command. If no mapping exists, everyone can run the command.
* **CommandChannels [maplist]:** A map of which channels commands are allowed to run on. No entry means a command can be run anywhere. If "!" is included as a channel, it switches from a whitelist to a blacklist, enabling you to exclude certain channels instead of allow certain channels.
* **CommandLimits [map]:** A map of timeouts for commands. A value of 30 means the command can't be used more than once every 30 seconds.
* **CommandDisabled [list]:** A list of disabled commands.
* **Commandperduration:** Maximum number of commands that can be run within `commandmaxduration` seconds. Default: 3
* **Commandmaxduration:** Default: 30. This means that by default, at most 3 commands can be run every 30 seconds.

### Spam
* **ImagePressure:** Additional pressure generated by each image, link or attachment in a message. Defaults to (`MaxPressure` - `BasePressure`) / 6, instantly silencing anyone posting 6 or more links at once.
* **PingPressure:** Additional pressure generated by each unique ping in a message. Defaults to (`MaxPressure` - `BasePressure`) / 20, instantly silencing anyone pinging 20 or more people at once.
* **LengthPressure:** Additional pressure generated by each individual character in the message. Discord allows messages up to 2000 characters in length. Defaults to (`MaxPressure` - `BasePressure`) / 8000, silencing anyone posting 3 huge messages at the same time.
* **RepeatPressure:** Additional pressure generated by a message that is identical to the previous message sent (ignores case). Defaults to `BasePressure`, effectively doubling the pressure penalty for repeated messages.
* **BasePressure:** The base pressure generated by sending a message, regardless of length or content. Defaults to 10.
* **PressureDecay:** The number of seconds it takes for a user to lose `Spam.BasePressure` from their pressure amount. Defaults to 2.5, so after sending 3 messages, it will take 7.5 seconds for their pressure to return to 0.
* **MaxPressure:** The maximum pressure allowed. If a user's pressure exceeds this amount, they will be silenced. Defaults to 60, which is intended to ban after a maximum of 6 short messages sent in rapid succession.
* **MaxChannelPressure**: Per-channel pressure override. If a channel's pressure is specified in this map, it will override the global maxpressure setting. This is actually implemented by multiplying the pressure generated by a message by (MaxPressure/That Channel's Pressure) to ensure consistency, so the pressure returned by `!getpressure` may not line up with what you expect.
* **MaxRemoveLookback:** Number of seconds back the bot should delete messages of a silenced user on the channel they spammed on. If set to 0, the bot will only delete the message that caused the user to be silenced. If less than 0, the bot won't delete any messages.
* **SilentRole:** This should be a role with no permissions, so the bot can quarantine potential spammers without banning them.
* **IgnoreRole:** If set, the bot will exclude anyone with this role from spam detection. Use with caution.
* **RaidTime:** In order to trigger a raid alarm, at least `Spam.RaidSize` people must join the chat within this many seconds of each other.
* **RaidSize:** Specifies how many people must have joined the server within the `Spam.RaidTime` period to qualify as a raid.
* **SilenceMessage:** This message will be sent to users that have been silenced by the !silence command.
* **AutoSilence:** Gets the current autosilence state. Use the !autosilence command to set this.
* **LockdownDuration:** Determines how long the server's verification mode will temporarily be increased to tableflip levels after a raid is detected. If set to 0, disables lockdown entirely.

### Bucket
* **MaxItems:** Determines the maximum number of items sweetiebot can carry in her bucket. If set to 0, her bucket is disabled.
* **MaxItemLength:** Determines the maximum length of a string that can be added to her bucket.
* **MaxFightHP:** Maximum HP of the randomly generated enemy for the `!fight` command.
* **MaxFightDamage:** Maximum amount of damage a randomly generated weapon can deal for the `!fight` command.

### Markov
* **MaxPMLines:** This is the maximum number of lines a response can be before sweetiebot automatically sends it as a PM to avoid cluttering the chat. Default: 5
* **Maxlines:** Maximum number of lines the `!episodequote` command can be given.
* **DefaultLines:** Number of lines for the Markov chain to spawn when not given a line count.
* **UseMemberNames:** Use member names instead of random pony names

### Users
* **TimezoneLocation:** Sets the timezone location of the server itself. When no user timezone is available, the bot will use this.
* **WelcomeChannel:** If set to a channel ID, the bot will treat this channel as a "quarantine zone" for silenced members. If autosilence is enabled, new users will be sent to this channel
* **WelcomeMessage:** If autosilence is enabled, this message will be sent to a new user upon joining.
* **Roles**: A list of all user-assignable roles, managed via !addrole and !removerole.

### Bored
* **Cooldown:** The bored cooldown timer, in seconds. This is the length of time a channel must be inactive for sweetiebot to post a bored message in it. Note that Sweetie Bot only checks each channel for inactivity every 30 seconds.
* **Commands [list]:** This determines what commands sweetie will run when she gets bored. She will choose one command from this list at random.

### Help
* **Rules [map]:** Contains a list of numbered rules. The numbers do not need to be contiguous, and can be negative.
* **HideNegativeRules:** If true, `!rules -1` will display a rule at index -1, but `!rules` will not. This is useful for joke rules or additional rules that newcomers don't need to know about.

### Log
* **Channel:** This is the channel where sweetiebot logs her output.
* **Cooldown:** The cooldown time for sweetiebot to display an error message, in seconds, intended to prevent the bot from spamming itself. Default: 4

### Witty
* **Responses [map]:** Stores the replies used by the Witty module and must be configured using `!addwit` or `!removewit`
* **Cooldown:** The cooldown time for the witty module. At least this many seconds must have passed before the bot will make another witty reply.

### Schedule
* **BirthdayRole:** This is the role given to members on their birthday.

### Search
* **MaxResults:** Maximum number of search results that can be requested at once.

### Spoiler
* **Channels [list]:** A list of channels that are exempt from the spoiler rules.

### Status
* **Cooldown:** Number of seconds sweetiebot waits before changing her status to a string picked randomly from the `status` collection

### Quote
* **Quotes [maplist]:** This is a map of quotes, which should be managed via `!addquote` and `!removequote`

## Modules
### Anti-Spam
Tracks all channels it is active on for spammers. Each message someone sends generates "pressure", which decays rapidly. Long messages, messages with links, or messages with pings will generate more pressure. If a user generates too much pressure, they will be silenced and the moderators notified. Also detects groups of people joining at the same time and alerts the moderators of a potential raid.
#### Commands
* **AutoSilence:** Toggle auto silence. `All` will autosilence all new members. `Raid` will turn on autosilence if a raid is detected (not recommended). `Alert` does not auto-silence anyone, but sends an alert to the mod channel whenever anyone joins the server. `Log` sends alerts to the log channel instead. `Off` disables auto-silence and unsilences everyone.
* **WipeWelcome:** Deletes all messages in the welcome channel, if there is one.
* **GetPressure:** [RESTRICTED] Gets user's spam pressure.

### Bored
After the chat is inactive for a given amount of time, chooses a random action from the `Bored.Commands` configuration option to run, such posting a link from the bored collection or throwing an item from her bucket.

### Bucket
Manages Sweetie's bucket functionality.
#### Commands
* **Give:** Gives something to sweetie.
* **Drop:** Drops something from sweetie's bucket.
* **List:** Lists everything sweetie has.
* **Fight:** Fights a random user or keyword.

### Collection
Contains commands for manipulating Sweetie Bot's collections.
#### Commands
* **Add:** Adds a line to a collection.
* **Remove:** Removes a line from a collection.
* **Collections:** Lists all collections.
* **Pick:** Picks a random item.
* **New:** Creates a new collection.
* **Delete:** Deletes a collection.
* **SearchCollection:** Searches a collection.
* **Import:** Imports a collection from another server.

### Configuration
Manages Sweetie Bot's configuration file.
#### Commands
* **SetConfig:** Sets a config value and saves the new configuration.
* **GetConfig:** Returns the current configuration, or a specific option.
* **Setup:** Performs initial setup on Sweetie Bot for a new server.

### Debug
Contains various debugging commands. Some of these commands can only be run by the bot owner.
#### Commands
* **Echo:** Makes Sweetie Bot say something in the given channel.
* **EchoEmbed:** Makes Sweetie Bot echo a rich text embed in a given channel.
* **Disable:** Disables the given module/command, if possible.
* **Enable:** Enables the given module/command.
* **Update:** [RESTRICTED] Updates sweetiebot.
* **DumpTables:** Dumps table row counts.
* **ListGuilds:** Lists servers.
* **Announce:** [RESTRICTED] Announcement command.
* **RemoveAlias:** [RESTRICTED] Removes an alias.

### Emotes
Keeps a list of banned emotes that are either seizure-inducing or way too big, and deletes any messages that use them.

### Roles
Contains commands for manipulating user-assignable roles. Roles created via !addrole are pingable by default, but user-assignable roles do NOT have any restrictions on them, so you can make a user-assignable role that isn't pingable, or gives special permissions, etc.
#### Commands
* **AddRole:** Either creates a new role, or adds an existing role to Sweetie's list of user-assignable roles. To create a new role, simply put in the name of the new role. To set an existing role as user-assignable, ping the role instead, via @role.
* **JoinRole:** Adds you to a user-assignable role.
* **ListRole:** Lists all users in a given role.
* **LeaveRole:** Removes you from a role.
* **RemoveRole:** Removes a role from the list of user-assignable roles, but **does not delete the role**. Use `!deleterole` for that.
* **DeleteRole:** Completely deletes a user-assignable role from the server. To prevent accidents, this cannot be used on roles that aren't user-assignable.

### Help/About
Contains commands for getting information about Sweetie Bot, her commands, or the server she is in.
#### Commands
* **Help:** [PM Only] Generates the list you are looking at right now.
* **About:** Displays information about Sweetie Bot.
* **Rules:** Lists the rules of the server.
* **Changelog:** Retrieves the changelog for Sweetie Bot.

### Markov
Generates content using Markov chains.
#### Commands
* **episodegen:** Randomly generates episodes.
* **EpisodeQuote:** Quotes random or specific lines from the show.
* **ship:** Generates a random ship.
* **BestPony:** Generates a random pony name.

### Miscellaneous
A collection of miscellaneous commands that don't belong to a module.
#### Commands
* **LastSeen:** Returns when a user was last seen.
* **Search:** [Self-Hosted Only] Performs a complex search on the chat history.
* **Roll:** Evaluates a dice expression.

### Polls
Manages polls.
#### Commands
* **Poll:** Displays poll description and options.
* **CreatePoll:** Creates a poll.
* **DeletePoll:** Deletes a poll.
* **Vote:** Votes in a poll.
* **Results:** Displays results of a poll.
* **AddOption:** Appends an option to a poll.

### Quotes
Manages the quoting system.
#### Commands
* **Quote:** Quotes a user.
* **AddQuote:** Adds a quote.
* **RemoveQuote:** Removes a quote.
* **SearchQuote:** Finds a quote.

### Scheduler
Manages the scheduling system, and periodically checks for events that need to be processed.
#### Commands
* **Schedule:** Gets a list of upcoming scheduled events.
* **Next:** Gets time until next event.
* **AddEvent:** Adds an event to the schedule.
* **RemoveEvent:** Removes an event.
Tells sweetiebot to remind you about something.
* **AddBirthday:** Adds a birthday to the schedule.

### Spoiler
Deletes any messages that match a regex created by the spoiler collection, unless a message is in `spoilchannels`.

### Status
Manages Sweetie Bot's status.
#### Commands
* **SetStatus:** [Self-Hosted Only] Sets the status message.

### Users
Contains commands for getting and setting user information.
#### Commands
* **newusers:** [PM Only] Gets a list of the most recent users to join the server.
* **aka:** Lists all known aliases of a user.
* **ban:** Bans a user.
* **BanNewcomers:** Bans any users that have sent their first message in the past 2 minutes.
* **time:** Gets a user's local time.
* **settimezone:** Set your local timezone.
* **UserInfo:** Lists information about a user.
* **DefaultServer:** Sets your default server.
* **Silence:** Silences a user.
* **Unsilence:** Unsilences a user.

### Witty
In response to certain patterns (determined by a regex) will post a response picked randomly from a list of them associated with that trigger. Rate limits itself to make sure it isn't too annoying.
#### Commands
* **AddWit:** Adds a line to wittyremarks.
* **RemoveWit:** Removes a remark from wittyremarks.

## Error Recovery
Sweetiebot can function with no database, but over half her commands will no longer function, and it will be impossible for her to respond to PMs. While in this state, there will be no errors in the log about failed database operations, becuase sweetiebot simply won't attempt the operations in the first place until she can re-establish a connection. After a database failure is detected, she will attempt to reconnect to the database every 30 seconds. She also had a deadlock detector which sends fake !about commands through the pipeline every 20 seconds - if sweetiebot fails to respond for 1 minute and 40 seconds, she will automatically terminate and restart.

## Compiling
**SELF-HOSTING THE BOT IS NOT SUPPORTED! If these instructions don't work, you're on your own.** If you would simply like to add the public instance of the bot to your server, use the link above. Sweetie Bot uses Go and MariaDB for a database backend. Install at least [Go 1.6](https://golang.org/dl/) (required for some language constructs) on your computer and [MariaDB 10.1](https://downloads.mariadb.org/) (required for utf8mb4 support). After cloning the project, `sweetiebot.sql` is included in the main folder directory. Run it from HiediSQL or your command line and it will create the necessary sweetiebot database. Then run the `sweetiebot_tz.sql` script.

Three files are necessary for sweetiebot to run that are never uploaded to the Git repository:

* `db.auth`: Database connection string
* `token`: Bot token used for login. [Create an application](https://discordapp.com/developers/applications/me#top) and turn it into a Bot User to get one.
* [OPTIONAL] `isdebug`: If this file exists and contains the word "true", sweetiebot will start in debug mode, and will only respond to commands on the hardcoded debug channels.

These files must in the root directory of wherever `main.exe` is compiled to. For testing purposes, it is sufficient to navigate to `/sweetiebot/main` in your command line and compile it there by typing `go build`, which will create `/sweetiebot/main/main.exe`. An example `config.json` file is included in `/sweetiebot/main` for testing purposes.

If your MariaDB installation uses default settings, your `db.auth` file should look like this:

`root:PASSWORD@tcp(127.0.0.1:3306)/sweetiebot?parseTime=true&collation=utf8mb4_general_ci`

If you get compiler errors, sweetiebot has two dependencies you should get:
* `go get github.com/go-sql-driver/mysql`
* `go get github.com/blackhole12/discordgo`

Note that sweetiebot requires the **develop** branch of `discordgo`. To switch, you will need to find where Golang downloaded the `discordgo` library, open a terminal in that directory, and use the command `git checkout develop`. Failure to do so will cause random compilation errors due to missing features. You will need to set MainGuildID at [sweetiebot.go:1534](https://github.com/blackhole12/sweetiebot/blob/master/sweetiebot/sweetiebot.go#L1534) to the primary Guild ID that Sweetiebot is attached to. If you don't do this, the deadlock detector will fail and terminate the bot.

### Contributing
Sweetiebot is modular and can easily incorporate additional modules or commands. A command is a struct that satisfies the `Command` interface. 

    type Command interface {
      Name() string
      Process([]string, *discordgo.Message, []int, *GuildInfo) (string, bool, *discordgo.MessageEmbed)
      Usage(*GuildInfo) *CommandUsage
      UsageShort() string
    }
    
`Name()` returns the actual text that invokes the command, `Usage()` is a long, structured explanation of the command and it's parameters, and `UsageShort()` is a much shorter explanation of the command, both used by `!help`. `Process()` is called when Sweetiebot evaluates a command and matches it with this command's name (case-insensitive). The first `[]string` parameter is a list of the arguments to the command, which are seperated by spaces, unless they were surrounded by double-quotes `"`, just how command-line arguments work on all standard operating systems.

Commands belong to Modules, and are automatically added when adding a module. Modules are more complicated and respond to certain events in the chat if they are enabled. At minimum, a module must implement the `Module` interface:

    type Module interface {
      Name() string
      Register(*GuildInfo)
      Commands() []Command
      Description() string
    }
    
`Name()` returns the name of the module, only used for enabling or restricting the module configuration. `Register()` is called whenever a guild is loaded, and that guild's configuration information is passed into the function. `Description()` is called by `!help` and should briefly describe the module's purpose. `Commands()` should return an initialized list of all commands associated with the module. A module must add itself to any hooks that it requires. For example:

    func (w *WittyModule) Register(info *GuildInfo) {
      info.hooks.OnMessageDelete = append(info.hooks.OnMessageDelete, w)
      info.hooks.OnMessageCreate = append(info.hooks.OnMessageCreate, w)
    }
    
A module must satisfy the interface of the hook it is trying to add itself to, which simply means implementing a hook function with the appropriate parameters. You can access the bot database using `sb.db`, but this will only work for server-independent database information (like users or transcripts), or on servers that have permission to write to the database. Additional modules will always be disabled on existing servers until they are explicitely enabled. [Submit a pull request](https://github.com/blackhole12/sweetiebot/pull/new/master) if you'd like to contribute!

******

©2017 Erik McClure
