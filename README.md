# Sweetie Bot
Sweetie Bot is an administration bot for the /r/mylittlepony Discord chat. Her primary function is anti-spam, by detecting potential spammers, silencing them, and deleting their messages. This helps immunize the chat against bot raids. She also keeps a log of the chat and it's users, and provides a command to find the last message that pinged a given user.

**If you use Sweetie Bot, consider [contributing to it's Patreon](https://www.patreon.com/erikmcclure) to help pay for hosting and maintenence costs.**

## Compiling
Sweetie Bot uses Go and MariaDB for a database backend. Install at least [Go 1.5](https://golang.org/dl/) (required for some language constructs) on your computer and [MariaDB 10.1](https://downloads.mariadb.org/) (required for utf8mb4 support). After cloning the project, `sweetiebot.sql` is included in the main folder directory. Run it from HiediSQL or your command line and it will create the necessary sweetiebot database. 

Three files are necessary for sweetiebot to run that are never uploaded to the git repo:

* `db.auth`: Database connection string
* `token`: Bot token used for login. [Create an application](https://discordapp.com/developers/applications/me#top) and turn it into a Bot User to get one.

These files must in the root directory of wherever `main.exe` is compiled to. For testing purposes, it is sufficient to navigate to `/sweetiebot/main` in your command line and compile it there by typing `go build`, which will create `/sweetiebot/main/main.exe`. An example `config.json` file is included in `/sweetiebot/main` for testing purposes.

If your MariaDB installation uses default settings, your `db.auth` file should look like this:

`root:PASSWORD@tcp(127.0.0.1:3306)/sweetiebot?parseTime=true&collation=utf8mb4_general_ci`

If you get compiler errors, sweetiebot has two dependences you should get:
* `go get github.com/go-sql-driver/mysql`
* `go get github.com/bwmarrin/discordgo`

## Adding Sweetiebot To Your Server

A limited version of sweetiebot can be added to any server. Simply follow [this link](https://discordapp.com/oauth2/authorize?client_id=171790139712864257&scope=bot&permissions=0) to add her to your server. The limited version of sweetiebot does not have a chatlog, which means !search and !lastping are unavailable. The status change loop and !setstatus are also disabled. All other commands and modules still function, however. 

### Configuration

Upon being added to a server, Sweetiebot will begin with all her commands and modules disabled, pending configuration. This is to ensure that members of the server cannot abuse the bot during the configuration process - the owner of the server can run any command, even if it's disabled (except for !update and !announce, which can only be run by the bot owner). **You must run `!quickconfig` to configure Sweetie Bot for your server!** `!quickconfig` takes the following parameters, in order:

* **logchannel** should be set to a channel that recieves log messages about errors and initialization. Usually this channel is only visible to the bot and the moderators.
* **alertrole** should be set to a role shared by all moderators. It is used to alert moderators and also allows the moderators to bypass command restrictions imposed by certain modules.
* **modchannel** should be set to whatever channel the moderators would like to recieve notifications on, such as potential raids, spammers being silenced, etc.
* **freechannels** should be set to a list of channel IDs that are excluded from rate limiting. If you have a #bot channel for spamming the bot, add it here.
* **silentrole** should be set to a role with all permissions disabled. This is the role assigned to spammers, which allows the moderation team to review what happened and ban them if necessary.
* **boredchannel** should either be set to the channel that sweetiebot will post bored messages on, or to `0`, which will disable the bored module. **This is not a real config option**, it only exists as a shortcut inside `!quickconfig`. To manually set this option, use `!setconfig module_channels bored #channelname`

Use `!help quickconfig` for an example of how to use the command. `!quickconfig` will automatically restrict all sensitive commands to `alertrole` and re-enable all modules. **You must PING the role or channel that you are adding to the bot!** For example, `!quickconfig #botlog @Server Moderator #modchat #bots @Silence #bots` would be a valid configuration. If you are manually setting a configuration option and you have a moderator role called "Server Moderator", you would use `!setconfig AlertRole @Server Moderator`, so that the bot recieves the actual role ID. You can go to your discord server configuration to make a specific role mentionable.

Additional configuration is optional, depending on what features of the bot are being used:

* **maxerror** is the cooldown time for sweetiebot to display an error message, intended to prevent the bot from spamming itself. Default: 4
* **maxwit** is the cooldown time for the witty module. At least this many seconds must have passed before the bot will make another witty reply.
* **maxbored** is the bored cooldown timer. This is the length of time a channel must be inactive for sweetiebot to post a bored message in it.
* **boredcommands** This determines what commands sweetie will run when she gets bored. She will choose one command from this list at random.
* **maxPMlines** This is the maximum number of lines a response can be before sweetiebot automatically sends it as a PM to avoid cluttering the chat. Default: 5
* **maxquotelines** Maximum number of lines the `!quote` command can be given.
* **maxsearchresults** Maximum number of search results that can be requested
* **defaultmarkovlines** Number of lines for the markov chain to spawn when not given a line count.
* **commandperduration** Maximum number of commands that can be run within `commandmaxduration` seconds. Default: 3
* **commandmaxduration** Default: 30. This means that by default, at most 3 commands can be run every 30 seconds.
* **statusdelaytime** Number of seconds sweetiebot waits before changing her status to a string picked randomly from the `status` collection
* **maxraidtime** specifies the time period sweetiebot should search for a potential raid
* **raidsize** specifies how many people must have joined the server within the `maxraidtime` period to qualify as a raid.
* **witty** stores the replies used by the Witty module and must be configured using `!addwit` or `!removewit`
* **aliases** can be used to redirect commands, such as making `!listgroup` call the `!listgroups` command. Useful for making shortcuts.
* **maxbucket** determines the maximum number of items sweetiebot can carry in her bucket. If set to 0, her bucket is disabled.
* **maxbucketlength** determines the maximum length of a string that can be added to her bucket.
* **maxfightHP** Maximum HP of the randomly generated enemy for the `!fight` command.
* **maxfightdamage** Maximum amount of damage a randomly generated weapon can deal for the `!fight` command.

### Using !setconfig
Basic configuration parameters can be set with `!setconfig <parameter name> <value>`. To get a list of configuration parameters, use `!getconfig`. To output the current value of a paramter, use `!getconfig <paramater name>`.

Certain configuration parameters are more complex. They can either be maps, lists, or maps of lists. This type information is listed when using `!getconfig`. Parameters that are lists simply take multiple values instead of one. Setting a list parameter to a set of values will *replace* the current list of values.

    !setconfig <list parameter> <value 1> <value 2> <value 3> <etc...>
    !setconfig boredchannels #channel1 #channel2

Maps are a set of key-value pairs. Unlike lists, each invocation of `!setconfig` will set just a single key-value pair and won't affect any others. If a key already exists, the value of that key will be overwritten. If the value is set to "", the key will be deleted.

    !setconfig <map parameter> <key> <value>
    !setconfig aliases listbucket list
    !setconfig aliases listbucket ""

Maps of lists simply map their keys to entire lists of values instead of just one value. The syntax is similar to setting a single map value:

    !setconfig <maplist parameter> <key> <value 1> <value 2> <value 3> <etc...>
    !setconfig command_channels roll #channel1 #channel2

## Functionality
### Modules
#### Anti-Spam
Tracks all channels for spammers. If someone posts more than *n* messages in *m* seconds, they will be silenced, their messages deleted, and the moderators will be notified. Detects groups of people joining at the same time and alerts the moderators of a potential raid.

#### Emotes
Keeps a list of banned emotes that are either siezure inducing or way too big, and deletes any messages that use them.

#### Pings
Tracks any messages that ping a user, including @everyone. This information can be used by the !lastping command to get the last message that pinged a user and any surrounding context.

#### Bored
After the chat is inactive for a given amount of time, randomly chooses various actions to perform, such as picking a random interesting link from the bored collection, throwing an item from her bucket, quoting the show, or generating a random sentence using markov chains.

#### Witty
In response to certain patterns (determined by a regex) will post a response picked randomly from a list of them associated with that trigger. Rate limits itself to make sure it isn't too annoying.

#### Spoiler
Deletes any messages that match a regex created by the spoiler collection, unless a message is in `spoilchannels`.

## Contributing
Sweetiebot is modular and can easily incorporate additional modules or commands. A command is a struct that satisfies the `Command` interface.

    type Command interface {
      Name() string
      Process([]string, *discordgo.Message, *GuildInfo) (string, bool)
      Usage(*GuildInfo) string
      UsageShort() string
    }
    
`Name()` returns the actual text that invokes the command, `Usage()` is a long explanation of the command, and `UsageShort()` is a much shorter explanation of the command, both used by `!help`. `Process()` is called when Sweetiebot evaluates a command and matches it with this command's name (case-insensitive). The first `[]string` parameter is a list of the arguments to the command, which are seperated by spaces, unless they were surrounded by double-quotes `"`, just how command-line arguments work on all standard operating systems.

Modules are more complicated and respond to certain events in the chat if they are enabled. At minimum, a module must implement the `Module` interface:

    type Module interface {
      Name() string
      Register(*GuildInfo)
    }
    
`Name()` returns the name of the module, only used for enabling or restricting the module configuration. `Register()` is called whenever a guild is loaded, and that guild's configuration information is passed into the function. A module must add itself to any hooks that it requires. For example:

    func (w *WittyModule) Register(info *GuildInfo) {
      info.hooks.OnMessageDelete = append(info.hooks.OnMessageDelete, w)
      info.hooks.OnMessageCreate = append(info.hooks.OnMessageCreate, w)
    }
    
A module must satisfy the interface of the hook it is trying to add itself to, which simply means implementing a hook function with the appropriate parameters. You can access the bot database using `sb.db`, but this will only work for server-independent database information (like users or transcripts), or on servers that have permission to write to the database. Additional modules will always be disabled on existing servers until they are explicitely enabled. [Submit a pull request](https://github.com/blackhole12/sweetiebot/pull/new/master) if you'd like to contribute!

******

©2016 Erik McClure
