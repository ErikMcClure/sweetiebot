These instructions are for **self-hosting only**. SELF-HOSTING IS NOT SUPPORTED. If you would simply like to add the public instance of the bot to your server, use [this link](https://discordapp.com/oauth2/authorize?client_id=171790139712864257&scope=bot&permissions=535948390).

**1.** Install at least [Go 1.8](https://golang.org/dl/) (required for json serialization). Verify that Go was properly installed to your PATH variable by typing `go version` in your terminal / command prompt. If you aren't prompted with something Go related, restart your computer and try again.

**2.** Install at least [MariaDB 10.1](https://downloads.mariadb.org/) (required for utf8mb4 support). If you get database errors, your MariaDB version is too old. Some repos ship very old versions of MariaDB, so don't trust them.

**3.** Run the `sweetiebot.sql` script, either in HeidiSQL (via `File > Run SQL File...`) or by using the `mysql` command line. Afterwards, run the `sweetiebot_tz.sql` script.

**4.** Get the required dependencies by running `go get github.com/go-sql-driver/mysql`, and `go get github.com/blackhole12/discordgo`. Note that sweetiebot relies on the **develop** branch of discordgo. To switch, you navigate to the `discordgo` library in your %GOPATH% folder, open a terminal in that directory, and use the command `git checkout develop`. Failure to do so will cause [random compilation errors](https://github.com/blackhole12/sweetiebot/issues/61) due to missing features.

**5.** Navigate to `sweetiebot/main` (where `main.go` is located) and open a console. Type `go build`, and verify that `main.exe` is now located in `sweetiebot/main/main.exe`.

**6.** Create a file called `db.auth` in `sweetiebot/main`. Open the file and paste a database connection string in the following format: `root:<YOUR PASSWORD>@tcp(127.0.0.1:3306)/sweetiebot?parseTime=true&collation=utf8mb4_general_ci`. If you are using unix sockets, remember to replace the IP address with the socket.

**7.** Create a file called `token` (no extension) in `sweetiebot/main`. Put the token you get from [this page](https://discordapp.com/developers/applications/me). Make sure your application is a Bot User! An example (fake) token file looks like: `Mjk0MjUwBAADF00DMjQ1NDUw.Cfake.tockenNiWM6l0TtHBlaBla0PZl4`.

**8.** Create a file called `mainguild` (no extension) in `sweetiebot/main`. Copy and paste the primary server ID you will be using Sweetie Bot on. [You may need to enable Developer Mode](https://support.discordapp.com/hc/en-us/articles/206346498-Where-can-I-find-my-User-Server-Message-ID-) to get the server ID.

**9.** Replace <YOUR_BOT_CLIENT_ID> with your bot ID in this link: `https://discordapp.com/oauth2/authorize?client_id=<YOUR_BOT_CLIENT_ID>&scope=bot&permissions=535948390`, then navigate to it in your browser to add your instance of sweetiebot to your server.

**10.** Run main.exe to start sweetiebot. If she doesn't message you with further instructions, you have not added her to your main guild. Remember that only the *server owner* can run `!setup`.