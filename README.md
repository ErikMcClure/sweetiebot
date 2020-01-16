# Sweetie Bot
[![GoDoc](https://godoc.org/github.com/erikmcclure/sweetiebot?status.svg)](https://godoc.org/github.com/erikmcclure/sweetiebot/sweetiebot) [![Go report](http://goreportcard.com/badge/erikmcclure/sweetiebot)](http://goreportcard.com/report/erikmcclure/sweetiebot)

Sweetie Bot was an administration bot for Discord servers. **Sweetie Bot is no longer under active development.** Feature requests will be denied, only bugfixes will be provided.

## Documentation

Please visit the [official website](https://sweetiebot.io/help) for help with commands and configuration.

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
