# Sweetie Bot
[![GoDoc](https://godoc.org/github.com/blackhole12/sweetiebot?status.svg)](https://godoc.org/github.com/blackhole12/sweetiebot/sweetiebot) [![Go report](http://goreportcard.com/badge/blackhole12/sweetiebot)](http://goreportcard.com/report/blackhole12/sweetiebot)

Sweetie Bot is an administration bot for Discord servers whose primary function is anti-spam, by detecting potential spammers, silencing them, and deleting their messages. Many users joining at the same time will trigger a lockdown to help immunize the chat against raids. [Patreon supporters](https://www.patreon.com/erikmcclure) also have access to a discord support channel and a chat log that allows moderators to track deleted messages.

**Sweetie Bot is no longer under active development.** Feature requests will be denied, only bugfixes will be provided.

## Installing Sweetie Bot

A limited version of Sweetie Bot can be added to any server via [this link](https://discordapp.com/oauth2/authorize?client_id=171790139712864257&scope=bot&permissions=535948390). For [$5 a month](https://www.patreon.com/erikmcclure), you can install your own instance of sweetie bot anywhere you want, with all features unlocked, which will automatically keep itself up-to-date. Download the selfhost executable for your operating system and run it from the directory you want the bot to be in. Follow the installation instructions provided and the bot will be up and running in no time.

[Windows (64-bit)](https://sweetiebot.io/update/windows/amd64/sweetie.zip)

[Windows (32-bit)](https://sweetiebot.io/update/windows/386/sweetie.zip)

[Linux (64-bit)](https://sweetiebot.io/update/linux/amd64/sweetie.tar.gz)

[Linux (32-bit)](https://sweetiebot.io/update/linux/386/sweetie.tar.gz)

## Sweetiebot Silver

Donating at least $1 a month to the [Patreon](https://www.patreon.com/erikmcclure) will automatically enable chat logging for your server, which is only accessible by moderators via the `!search` command. It will also enable much higher limits for the total number of unique items you can store in tagged item collections. In order to recieve these benefits, you must [link your Patreon and Discord accounts](https://patreon.zendesk.com/hc/en-us/articles/212052266-How-do-I-get-my-Discord-Rewards-), and you must join the [Sweetie Bot Support Channel](https://discord.gg/t2gVQvN). The bot cannot detect your donation level unless you are on the server and your account has been linked.

## Documentation

Please visit the [official website](https://sweetiebot.io/help) for help with commands and configuration.

## Setup

Upon being added to a server, Sweetie Bot will begin with all commands and modules disabled. **Only users with admin rights can setup a server.** Sweetie Bot will send the owner of the server a PM when she is first added with instructions on how to run the `!setup` command. In case you missed it, `!setup` takes the following parameters, in order:

* **Mod Role** should be set to a role shared by all moderators. It is used to alert moderators and also allows the moderators to bypass command restrictions imposed by certain modules.
* **Mod Channel** should be set to whatever channel the moderators would like to recieve notifications on, such as potential raids, spammers being silenced, etc.
* **Log Channel** [OPTIONAL] should be set to a channel that recieves log messages about errors and initialization. Usually this channel is only visible to the bot and the moderators.
* **Member Role** [OPTIONAL] if you already have a role that you've assigned to all members, specify it here. Otherwise, if this argument is omitted, Sweetie Bot will generate a new "Member" role and add everyone on the server to it, then disable all permissions on the everyone role. This ensures that if a massive raid happens, the new users won't have the ability to speak, even if the bot is overwhelmed or nonfunctional.

For example: `!setup @Mods #staff-lounge #bot-log @Member`

`!setup` will automatically restrict all sensitive commands to `modrole` and enable a default set of modules. 
**Running the setup twice will *delete everything* and reset all configuration values.** Specify an additional `OVERRIDE` parameter if this is your intent.

**DO NOT GIVE SWEETIE BOT ADMINISTRATIVE PERMISSIONS OR THE ABILITY TO PING EVERYONE!** Sweetie bot does not and will never attempt to filter `@everyone` pings. Sweetie bot only requires the following permissions: `Manage Server`, `Manage Roles`, `Ban Members`, `Manage Messages`, `Mute Members`, plus all the default read/write permissions given to everyone.

Additional configuration is optional via `!setconfig` but usually isn't necessary. **DO NOT SET PRESSURE VALUES UNLESS YOU NEED TO CHANGE THEM.** The pressure values are *already set up for you* and setting them incorrectly will result in Sweetie Bot silencing everyone instantly.

### Configuration
Basic configuration parameters can be set with `!setconfig <parameter name> <value>`. To get a list of configuration parameters, use `!getconfig`. To output the current value of a parameter, use `!getconfig <paramater name>`. Do not use quotes on these values if they have spaces.

#### Common Scenarios
* **Changing the prefix:** `!setconfig commandprefix [prefix]`
* **Make the anti-spam module ignore `#channelname`:** `!setconfig modules.channels spam ! #channelname`
* **Make birthday announcements show up in `#channelname`:** `!setconfig modules.channels scheduler #channelname`
* **Change the channel the bored module activates on:** `!setconfig modules.channels bored #yourchannel`
* **Sweetiebot is silencing everyone?!** You messed up the spam module configuration. Either run `!setup` again to wipe your settings, or reset all her spam module values to the defaults listed here.
* **Prevent Sweetiebot from saying "That's an invalid command":** `!setconfig IgnoreInvalidCommands true`
* **Set the bored command list:** `!setconfig bored.commands "!command1" "!command2 arg"`
* **Set up a basic word filter:** Check the [help page for the filter module](https://sweetiebot.io/help/filter/), which includes an example filter regex for this.

#### Advanced Configuration
Certain configuration parameters are more complex. They can either be maps, lists, or maps of lists. This type information is listed when using `!getconfig`. Parameters that are lists simply take multiple values instead of one. Setting a list parameter to a set of values will *replace* the current list of values. In list parameters, *all values* must use quotes if they have spaces in them.

    !setconfig <list parameter> <value 1> <value 2> <value 3> <etc...>
    !setconfig bored.commands !drop "!pick cute"

You may pass no values to a list, which will simply set the list to nothing:

    !setconfig bored.commands

Maps are a set of key-value pairs. Unlike lists, each invocation of `!setconfig` will set just a single key-value pair and won't affect any others. If a key already exists, the value of that key will be overwritten.

    !setconfig <map parameter> <key> <value>
    !setconfig basic.aliases listbucket list

If no value is given, the key will be deleted:

    !setconfig basic.aliases listbucket

Maps of lists match keys to entire lists of values instead of just one value. The syntax is similar to setting a single map value:

    !setconfig <maplist parameter> <key> <value 1> <value 2> <value 3> <etc...>
    !setconfig modules.commandchannels roll #channel1 #channel2

To delete a value, simply provide only the key and no values:

    !setconfig modules.commandchannels roll
	
Some maplists are whitelists of channels or roles. To change them into a blacklist, add `!` anywhere in the maplist:

    !setconfig modules.commandchannels roll ! #excludedchannel1 #excludedchannel2

## Error Recovery
Sweetie Bot can function with no database, but most commands will no longer function, and it will be impossible to respond to PMs. While in this state, there will be no errors in the log about failed database operations, because Sweetie Bot simply won't attempt the operations in the first place until she can re-establish a connection. After a database failure is detected, she will attempt to reconnect to the database every 30 seconds. She also has a deadlock detector which sends fake !about commands through the pipeline every 20 seconds - if Sweetie Bot fails to respond for 1 minute and 40 seconds, she will automatically terminate and restart.

******

©2018 Erik McClure
