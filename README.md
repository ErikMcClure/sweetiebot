# Sweetie Bot
Sweetie Bot is an administration bot for the /r/mylittlepony Discord chat. Her primary function is anti-spam, by detecting potential spammers, silencing them, and deleting their messages. This helps immunize the chat against bot raids. She also keeps a log of the chat and it's users, and provides a command to find the last message that pinged a given user.

## Compiling
Sweetie Bot uses Go and MariaDB for a database backend. Install at least [Go 1.5](https://golang.org/dl/) (required for some language constructs) on your computer and [MariaDB 10.1](https://downloads.mariadb.org/) (required for utf8mb4 support). After cloning the project, `sweetiebot.sql` is included in the main folder directory. Run it from HiediSQL or your command line and it will create the necessary sweetiebot database. 

Three files are necessary for sweetiebot to run that are never uploaded to the git repo:

* `db.auth`: Database connection string
* `username`: Username that sweetiebot should log in with
* `passwd`: Password for the account sweetiebot should use

These files must in the root directory of wherever `main.exe` is compiled to. For testing purposes, it is sufficient to navigate to `/sweetiebot/main` in your command line and compile it there by typing `go build`, which will create `/sweetiebot/main/main.exe`. An example `config.json` file is included in `/sweetiebot/main` for testing purposes.

If your MariaDB installation uses default settings, your `db.auth` file should look like this:

`root:PASSWORD@tcp(127.0.0.1:3306)/sweetiebot?parseTime=true&collation=utf8mb4_general_ci`

If you get compiler errors, sweetiebot has two dependences you should get:
* `go get github.com/go-sql-driver/mysql`
* `go get github.com/bwmarrin/discordgo`

## Functionality
### Modules
#### Anti-Spam
Tracks all channels for spammers. If someone posts more than *n* messages in *m* seconds, they will be silenced, their messages deleted, and the moderators will be notified.

#### Emotes
Keeps a list of banned emotes that are either siezure inducing or way too big, and deletes any messages that use them.

#### Pings
Tracks any messages that ping a user, including @everyone. This information can be used by the !lastping command to get the last message that pinged a user and any surrounding context.

******

©2016 Erik McClure
