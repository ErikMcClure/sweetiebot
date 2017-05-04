How to properly set up Sweetiebot for use on your server

**1.** Install [Golang](https://golang.org/dl/) (at least v1.5) and [MariaDB](https://downloads.mariadb.org/) (at least v10.1) to your computer. Take note of your credentials for MariaDB, as you'll need to know that for later.

**2.** Verify that Go was properly installed to your PATH variable by typing `go version` in your terminal / command prompt. If you aren't prompted with something Go related, restart your computer and try again.

**3.** Clone the repository in its entirety

**4.** Install the required libraries with the commands `go get github.com/go-sql-driver/mysql` and `go get github.com/bwmarrin/discordgo`.

**5.** Open your command prompt / terminal in the `/main` folder and type `go build`. You should see `main.exe` appear in the folder.

**6a.** Create a file with the literal name of `token`. No extensions. There, you will put the bot token you get from [this](https://discordapp.com/developers/applications/me) page. Make sure to make your application a Bot User! A valid, but fake, token file would have a single line that looks like `Mjk0MjUwBAADF00DMjQ1NDUw.Cfake.tockenNiWM6l0TtHBlaBla0PZl4`. Make sure this file is saved in `/main`.

**6b.** If the application stops doing anything after "Connection established" or exits with an authentication error: Check that you dont have a `isuser`-file in the `/main` folder (The `isuser`-file is only needed if you want to start the bot in user mode, which is rarely the case).

**7.** Create a file called `db.auth` and put this inside: `root:<YOUR PASSWORD>@tcp(127.0.0.1:3306)/sweetiebot?parseTime=true&collation=utf8mb4_general_ci`. Save the file in `/main`.

**8.** Open the MySQL GUI under MariaDB called "HeidiSQL". Enter your password in the password box and then hit *Open*.

**9.** Top Bar: File > Run SQL File... then navigate to your cloned Sweetiebot folder and select "sweetiebot.sql". Afterwards, do the same with "sweetiebot_tz.sql".

**10.** Run `main.exe`.

You may have noticed that your bot isn't in your server yet. You will be able to connect your bot to your server by using this URL.

`https://discordapp.com/oauth2/authorize?client_id=<YOUR_BOT_CLIENT_ID>&scope=bot`

Then add it to a server you have administration rights to. Be sure to run `!quickconfig` before attempting to use the bot.
