package sweetiebot

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/erikmcclure/discordgo"
)

var sqlfileregex = regexp.MustCompile("^sql_([0-9]+)[.]sql$")
var ErrMD5Error = errors.New("MD5 mismatch, file corrupt")

type SelfhostBase struct {
	Version int
}

// Selfhost stores selfhost update state
type Selfhost struct {
	SelfhostBase
	ready  AtomicBool
	Donors sync.Map //map[DiscordUser]bool
}

// UpdateStatus stores the version and additional files to download
type UpdateStatus struct {
	Version int      `json:"version"`
	Files   []string `json:"files"`
}

// UpdateEndpoint returns the update endpoint for the given id and current architecture
func UpdateEndpoint(request string, userID DiscordUser, oldversion int) string {
	if oldversion > 0 {
		return fmt.Sprintf("https://sweetiebot.io/update/%s/%s/%s?user=%v&version=%v", runtime.GOOS, runtime.GOARCH, request, userID, oldversion)
	}
	return fmt.Sprintf("https://sweetiebot.io/update/%s/%s/%s?user=%v", runtime.GOOS, runtime.GOARCH, request, userID)
}

// CheckForUpdate returns 0 if no update is needed, -1 if you haven't bought selfhosting, and 1 plus a list of files that are needed if oldversion is nonzero.
func (b *SelfhostBase) CheckForUpdate(userID DiscordUser, oldversion int) (int8, *UpdateStatus) {
	resp, err := HTTPRequestData(UpdateEndpoint("", userID, oldversion))

	if err != nil {
		return 0, nil
	}

	status := &UpdateStatus{}
	err = json.Unmarshal(resp, status)
	if err == nil && status.Version == 0 {
		return -1, nil
	}
	if err != nil || b.Version >= status.Version {
		return 0, nil
	}
	return 1, status
}

func getExt(sys string) string {
	if sys == "windows" {
		return ".exe"
	}
	return ""
}

// SelfUpdate is run by sweetie.exe to to get a new version of updater.exe
func (b *SelfhostBase) SelfUpdate(ownerid DiscordUser) error {
	path, _ := GetCurrentDir()
	v, err := RunCommand("./updater"+getExt(runtime.GOOS), path, "version")
	if err != nil {
		return err
	}
	version, err := strconv.Atoi(v)
	if err != nil {
		return err
	}
	if version < BotVersion.Integer() {
		name := "updater" + getExt(runtime.GOOS)
		if err = DownloadFile(UpdateEndpoint(name, ownerid, 0), name, true); err == nil {
			err = os.Chmod(name, 0777)
		}
		return err
	}
	return nil
}

// DoUpdate is the function run by updater.exe to perform the actual update
func (b *SelfhostBase) DoUpdate(dbauth string, token string) error {
	c := make(chan DiscordUser)
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		app, err := s.Application("@me")
		if err == nil {
			c <- DiscordUser(app.Owner.ID)
		} else {
			c <- UserEmpty
		}
	})
	if err = dg.Open(); err != nil {
		return err
	}
	defer dg.Close()
	defer close(c)

	ownerid := <-c
	if ownerid == UserEmpty {
		return errors.New("failed to get App info")
	}

	if check, update := b.CheckForUpdate(ownerid, BotVersion.Integer()); check > 0 {
		fmt.Println("Update found, downloading files:", update.Files)
		dir, _ := GetCurrentDir()
		for _, file := range update.Files {
			url := UpdateEndpoint(file, ownerid, 0)
			if err := DownloadFile(url, filepath.Join(dir, "~"+file), true); err != nil {
				return err
			}

			switch file {
			case "sweetie":
			case "sweetie.exe":
			default:
				if err := os.Rename(filepath.Join(dir, "~"+file), filepath.Join(dir, file)); err != nil {
					fmt.Println(err)
				}
			}
		}

		if err = b.UpgradeDatabase(dir, dbauth); err != nil { // Only replace the EXE if the upgrade succeeded and the replacement EXE exists
			return err
		}
		if _, err := os.Stat("~sweetie.exe"); err == nil {
			os.Remove("sweetie.exe")
		}
		if _, err := os.Stat("~sweetie"); err == nil {
			os.Remove("sweetie")
		}
		os.Rename("~sweetie.exe", "sweetie.exe")
		os.Rename("~sweetie", "sweetie")
		os.Chmod("sweetie", 0777)
	}
	return nil
}

// ConfigureMux has nothing to configure for selfhosters
func (b *SelfhostBase) ConfigureMux(mux *http.ServeMux) {}

// CheckDonor always returns false because donor status never changes when selfhosting
func (b *SelfhostBase) CheckDonor(m *discordgo.Member) bool { return false }

// CheckGuilds sets all guilds as silver unless there's more than 30
func (b *SelfhostBase) CheckGuilds(guilds map[DiscordGuild]*GuildInfo) {
	count := 0
	for _, g := range guilds {
		g.Silver.Set(count < 30)
		count++
	}
}

// FindUpgradeFiles finds the .sql upgrade files that should be run
func FindUpgradeFiles(scriptdir string, version int) (files []int) {
	results, _ := ioutil.ReadDir(scriptdir)
	for _, f := range results {
		if !f.IsDir() {
			matches := sqlfileregex.FindStringSubmatch(f.Name())
			if len(matches) > 1 {
				v, err := strconv.Atoi(matches[1])
				if err == nil && v > version {
					files = append(files, v)
				}
			}
		}
	}
	return
}

// UpgradeDatabase does the database migration portion of the upgrade
func (b *SelfhostBase) UpgradeDatabase(scriptdir string, dbauth string) error {
	db, err := sql.Open("mysql", strings.TrimSpace(dbauth))
	if err != nil {
		return err
	}
	defer db.Close()

	// Find all SQL migration scripts with a version above our current one
	files := FindUpgradeFiles(scriptdir, b.Version)

	fmt.Println(files)
	// If there are any migration scripts we need to run, sort them and then execute them
	if len(files) > 0 {
		sort.Ints(files)
		for _, v := range files {
			if err := ExecuteSQLFile(db, filepath.Join(scriptdir, fmt.Sprintf("sql_%v.sql", v))); err != nil {
				return fmt.Errorf("Error executing SQL file: %s", err.Error())
			}
			fmt.Printf("Applied sql_%v.sql\n", v)
		}
	}

	return nil
}

// ExecuteSQLFile splits a file into statements and executes it
func ExecuteSQLFile(db *sql.DB, file string) error {
	script, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	// Split up the file into individual statements
	statements := strings.Split(string(script), "//")
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if len(statement) > 0 && strings.ToLower(statement) != "delimiter" {
			_, err = db.Exec(statement)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// RunCommand runs the given command and prints any output
func RunCommand(path string, dir string, args ...string) (string, error) {
	c := exec.Command(path, args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(out))
	return string(out), err
}

// ExecCommand adds a directory and environment to a command created by exec.Command()
func ExecCommand(c *exec.Cmd, dir string, env ...string) (string, error) {
	c.Env = env
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(out))
	return string(out), err
}

// StartCommand starts the given command but doesn't wait for it to complete
func StartCommand(path string, dir string, args ...string) error {
	c := exec.Command(path, args...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Start()
}

func DumpScript(dir string) {
	switch runtime.GOOS {
	case "windows":
		ioutil.WriteFile("run.bat", []byte(`
:loop
.\sweetie.exe
.\updater.exe
goto loop
`), 0744)
	case "linux":
		ioutil.WriteFile("run.sh", []byte(`
#!/bin/bash
while :
do
./sweetie
./updater
done
`), 0744)
	}
}

func calcMD5Hash(file string) ([]byte, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
func checkHash(file string, hash []byte) bool {
	if result, err := calcMD5Hash(file); err == nil {
		return bytes.Compare(result, hash) == 0
	}
	return false
}
func getMD5Hash(url string) (md5body []byte, err error) {
	i := strings.Index(url, "?")
	md5url := url + ".md5"
	if i >= 0 {
		md5url = url[:i] + ".md5" + url[i:]
	}
	var resp *http.Response
	if resp, err = http.Get(md5url); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			err = errors.New(resp.Status)
			return
		}
		md5body, err = ioutil.ReadAll(resp.Body)
	}
	return
}

// DownloadFile downloads a file from the url and attempts to get a *.md5 to check it against if checkMD5 is true
func DownloadFile(url string, file string, checkMD5 bool) error {
	var md5body []byte
	if checkMD5 {
		md5body, _ = getMD5Hash(url)
		if len(md5body) > 0 && checkHash(file, md5body) {
			return nil // If the hash is nonzero and matches the file, we don't need to download it
		}
	}

	os.Remove(file) // Otherwise, delete any existing file
	out, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}
	_, err = io.Copy(out, resp.Body)

	out.Close()
	if err == nil {
		out, err = os.OpenFile(file, os.O_RDONLY, 0666)
	}
	if err == nil && len(md5body) > 0 {
		h := md5.New()
		if _, err = io.Copy(h, out); err != nil {
			return err
		}
		if bytes.Compare(md5body, h.Sum(nil)) != 0 {
			return ErrMD5Error
		}
	}
	return err
}

// HTTPRequestData returns the result of an HTTP request as a byte array
func HTTPRequestData(url string) (body []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = errors.New(resp.Status)
		return
	}

	body, err = ioutil.ReadAll(resp.Body)
	return
}

// GetWebDir returns the directory where the website is
func (b *SelfhostBase) GetWebDir() string {
	if dir, err := GetCurrentDir(); err == nil {
		return dir
	}
	return ""
}
