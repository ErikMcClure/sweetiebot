package sweetiebot

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/erikmcclure/discordgo"
)

const buySelfhosting = "Whoops, it seems like you haven't paid for selfhosting support! To buy selfhosting support, go to " + PatreonURL + " and choose the Selfhost reward, then make sure you link your Patreon account with your Discord account. If you're already paying for selfhosting support but this installation failed to detect it, please contact Erik McClure#9999 on the sweetiebot support channel: https://discord.gg/t2gVQvN"

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

// Install sweetiebot in the given directory
func Install(path string, selfhoster *Selfhost) {
	var input string
	var dbauth string
	var token string
	var userid uint64
	var appid uint64
	var mainguild uint64
	var mainguildname string
	dl := make(chan error)
	fmt.Print("This looks like your first time running Sweetie Bot! To begin, do you already have MariaDB (or compatible SQL server) installed? (Y/N)\n> ")
	fmt.Scanln(&input)
	switch strings.ToLower(input) {
	case "y":
		fallthrough
	case "yes":
		fmt.Print("In that case, please paste the database connection string I should use to connect to the database. The connection string should look like: root:<YOUR PASSWORD>@tcp(127.0.0.1:3306)/sweetiebot?parseTime=true&collation=utf8mb4_general_ci\nIf you are connecting with a socket instead of an IP, use the standard socket format instead.\n> ")
		fmt.Scanln(&dbauth)
		go func() {
			dl <- nil
		}()
	default:

		switch runtime.GOOS {
		case "windows":
			fmt.Println("In that case, I'll start downloading mariadb for you. While that's downloading...")
			go func() {
				dl <- DownloadFile(fmt.Sprintf("https://sweetiebot.io/update/%s/%s/%s", runtime.GOOS, runtime.GOARCH, "mariadb-10.msi"), "mariadb.msi", false)
			}()
		case "linux":
			url := "https://downloads.mariadb.org/mariadb/repositories/#mirror=nodesdirect"
			fmt.Println("I can't install packages for you on linux. Please follow the directions on this website to add the mariadb 10.2 repository, install it, and restart the installation:", url)
			openBrowser(url)
			fmt.Scanln(&input)
			os.Exit(1)
		}
	}

	fmt.Print("To connect to discord, I will need a bot token. You can get a token to connect with by creating an app here: https://discordapp.com/developers/applications/me\nMake your new app a 'Bot User' and copy+paste the token it gives you here. Don't give me your client secret! A token should look like this: Mjk0MjUwBAADF00DMjQ1NDUw.Cfake.tockenNiWM6l0TtHBlaBla0PZl4\n> ")
	fmt.Scanln(&token)

dgretry:
	dg, err := discordgo.New("Bot " + token)
	if err == nil {
		dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
			app, err := s.Application("@me")
			if err == nil {
				appid = SBatoi(app.ID)
				atomic.StoreUint64(&userid, SBatoi(app.Owner.ID))
			} else {
				panic("Failed to get application data")
			}
		})
		dg.AddHandler(func(s *discordgo.Session, m *discordgo.GuildCreate) {
			if atomic.LoadUint64(&mainguild) == 0 {
				atomic.StoreUint64(&mainguild, SBatoi(m.Guild.ID))
				mainguildname = m.Guild.Name
			}
		})
		err = dg.Open()
	}
	if err != nil {
		fmt.Print("Failed to create discordgo session. Are you sure you gave me the user bot token and NOT the client secret? Enter a new token, or just press enter to try again. Error:", err.Error(), "\n> ")
		fmt.Scanln(&input)
		if len(input) > 0 {
			token = input
		}
		goto dgretry
	}

	for atomic.LoadUint64(&userid) == 0 {
		time.Sleep(100 * time.Millisecond)
	}
	if update, _ := selfhoster.CheckForUpdate(NewDiscordUser(userid), 0); update < 0 {
		fmt.Print(buySelfhosting + "\n\nPress any key to exit.")
		fmt.Scanln(&token)
		os.Exit(0)
	}

	time.Sleep(500 * time.Millisecond)
	if atomic.LoadUint64(&mainguild) == 0 {
		url := fmt.Sprintf("https://discordapp.com/oauth2/authorize?client_id=%v&scope=bot&permissions=535948390", appid)
		fmt.Println("I've launched a webpage that will let you add the bot to a server. The first server you add it to will be set as the main server. I'll wait here until you've added a server. URL:", url)
		openBrowser(url)

		for atomic.LoadUint64(&mainguild) == 0 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	fmt.Println("The main server was set to", mainguildname, "(", mainguild, ")")
	fmt.Println("Waiting for all downloads to finish...")

	if err = <-dl; err == nil {
		files := []string{"updater" + getExt(runtime.GOOS), "sweetiebot.sql", "sweetiebot_tz.sql", "legacy_migrate.sql", "web.html", "web.css", "sweetiebot.svg"}
		for _, v := range files {
			if err = DownloadFile(UpdateEndpoint(v, NewDiscordUser(userid), 0), v, true); err != nil {
				break
			}
		}
	}

	os.Chmod("updater"+getExt(runtime.GOOS), 0777)
	if err != nil {
		panic("Download error: " + err.Error())
	}

	DumpScript(path)
	if len(dbauth) == 0 {
		// Wait for mariadb to finish
		var password string
		var location string
		fmt.Print("Installing MariaDB... What should the root password be? If you don't care, leave it blank to generate a random one.\n> ")
		fmt.Scanln(&password)

		if len(password) == 0 {
			const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			buf := make([]byte, 16)
			for i := range buf {
				buf[i] = chars[rand.Intn(len(chars))]
			}
			password = string(buf)
			fmt.Println("Your root password is: ", password)
		}

		switch runtime.GOOS {
		case "windows":
			fmt.Print("\nWhere should I install to? If you don't care, leave it blank. The path cannot have spaces.\n> ")
			fmt.Scanln(&location)
			properties := []string{"/i", "mariadb.msi", "SERVICENAME=MySQL", "UTF8=1", "PASSWORD=" + password}
			if len(location) > 0 {
				properties = append(properties, "INSTALLDIR="+location)
			}
			properties = append(properties, "/passive")
			RunCommand("msiexec", path, properties...)
		default:
			panic("Unknown Operating System!")
		}
		dbauth = fmt.Sprintf("root:%s@tcp(127.0.0.1:3306)/sweetiebot?parseTime=true&collation=utf8mb4_general_ci", password)
	}

	// Remove the database specifier from the connection string because it probably doesn't exist yet
dbretry:
	halfway := strings.Index(dbauth, "@")
	if halfway < 0 {
		panic("Invalid database authentication string format.")
	}
	dbauthalt := dbauth[halfway:]
	dbauthalt = dbauth[:strings.Index(dbauthalt, ")")+2+halfway] + dbauth[strings.Index(dbauthalt, "?")+halfway:]

	db, err := sql.Open("mysql", strings.TrimSpace(dbauthalt))
	if err == nil {
		err = db.Ping()
	}
	if err != nil {
		fmt.Println("Cannot connect to database, are you sure the connection string is correct? Enter a new connection string or simply press enter to try again. Error:", err.Error())
		fmt.Print("\n> ")
		var conn string
		fmt.Scanln(&conn)
		if len(conn) > 0 {
			dbauth = conn
		}
		goto dbretry
	}
	defer db.Close()

	row := db.QueryRow("SHOW DATABASES LIKE 'sweetiebot'")
	var dbname string
	err = row.Scan(&dbname)
	if err != nil || len(dbname) == 0 {
		fmt.Println("Initializing database...")
		_, err = db.Exec("CREATE DATABASE sweetiebot CHARACTER SET = 'utf8mb4' COLLATE = 'utf8mb4_general_ci'")
		if err == nil {
			_, err = db.Exec("USE sweetiebot")
		}
		if err == nil {
			err = ExecuteSQLFile(db, "sweetiebot.sql")
		}
		if err == nil {
			err = ExecuteSQLFile(db, "sweetiebot_tz.sql")
		}
		if err != nil {
			panic("FATAL ERROR: Failed to initialize database! " + err.Error())
		}
	} else {
		fmt.Println("Existing sweetiebot database found, attempting upgrade...")
		_, err = db.Exec("USE sweetiebot")
		if err != nil {
			panic(err)
		}
		if err = ExecuteSQLFile(db, "legacy_migrate.sql"); err != nil {
			panic("FATAL ERROR: Failed to migrate database! " + err.Error())
		}
	}

	// Only create the selfhost file if the install succeeded
	ioutil.WriteFile("selfhost.json", []byte(fmt.Sprintf(`{"token": "%s", "dbauth": "%s", "mainguildid": "%v"}`, token, dbauth, mainguild)), 0644)

	script := "run.sh"
	switch runtime.GOOS {
	case "windows":
		script = "run.bat"
	}
	fmt.Println("Sweetie Bot has been installed! To run her in an update loop, run the", script, "shell script that was created in this directory.\n\nPress enter to exit.")
	fmt.Scanln(&input)
	os.Exit(1)
}
