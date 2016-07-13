# Sweetie Bot
Sweetie Bot is an administration bot for the /r/mylittlepony Discord chat. Her primary function is anti-spam, by detecting potential spammers, silencing them, and deleting their messages. This helps immunize the chat against bot raids. She also keeps a log of the chat and it's users, and provides a command to find the last message that pinged a given user.

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

Upon being added to a server, Sweetiebot will begin with all her commands and modules disabled, pending configuration. This is to ensure that members of the server cannot abuse the bot during the configuration process - the owner of the server can run any command, even if it's disabled (except for !update and !announce, which can only be run by the bot owner). It is highly recommended the following settings are configured at a bare minimum:

* **logchannel** should be set to a channel that recieves log messages about errors and initialization. Usually this channel is only visible to the bot and the moderators.
* **alertrole** should be set to a role shared by all moderators. It is used to alert moderators and also allows the moderators to bypass command restrictions imposed by certain modules.
* **modchannel** should be set to whatever channel the moderators would like to recieve notifications on, such as potential raids, spammers being silenced, etc.
* **freechannels** should be set to a list of channel IDs that are excluded from rate limiting. If you have a #bot channel for spamming the bot, add it here.
* **silentrole** should be set to a role with all permissions disabled. This is the role assigned to spammers, which allows the moderation team to review what happened and ban them if necessary.
* **boredchannel** should either be set to the channel that sweetiebot will post bored messages on, or to **0**, which will disable the bored module.

Once this has been done, all sensitive commands (such as setconfig) should be restricted to the appropriate moderator role. You can use sweetiebot's !quickconfig command to do all of this automatically and re-enable all modules.

Additional configuration is optional, depending on what features of the bot are being used:

* **spoilchannels** is a list of channels where spoilers are allowed. Only applicable to the Spoiler module.
* **aliases** can be used to redirect commands, such as making !listgroup call the !listgroups command. Useful for making shortcuts.
* **witty** stores the replies used by the Witty module and must be configured using !addwit or !removewit
* **maxbored** specifies the duration of inactivity that will trigger Sweetiebot's bored module.
* **maxraidtime** specifies the time period sweetiebot should search for a potential raid
* **raidsize** specifies how many people must have joined the server within the **maxraidtime** period to qualify as a raid.
* **maxbucket** determines the maximum number of items sweetiebot can carry in her bucket.
* **maxbucketlength** determines the maximum length of a string that can be added to her bucket.

### Using !setconfig
Basic configuration parameters can be set with `!setconfig <parameter name> <value>`. To get a list of configuration parameters, use `!getconfig`. To output the current value of a paramter, use `!getconfig <paramater name>`.

Certain configuration parameters are more complex. They can either be maps, lists, or maps of lists. This type information is listed when using `!getconfig`. Parameters that are lists simply take multiple values instead of one. Setting a list parameter to a set of values will *replace* the current list of values.

`!setconfig <list parameter> <value 1> <value 2> <value 3> <etc...>`
`!setconfig boredchannels #channel1 #channel2`

Maps are a set of key-value pairs. Unlike lists, each invocation of `!setconfig` will set just a single key-value pair and won't affect any others. If a key already exists, the value of that key will be overwritten. If the value is set to "", the key will be deleted.

`!setconfig <map parameter> <key> <value>`
`!setconfig aliases listbucket list`
`!setconfig aliases listbucket ""`

Maps of lists simply map their keys to entire lists of values instead of just one value. The syntax is similar to setting a single map value:

`!setconfig <maplist parameter> <key> <value 1> <value 2> <value 3> <etc...>`
`!setconfig command_channels roll #channel1 #channel2`

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

******

©2016 Erik McClure
