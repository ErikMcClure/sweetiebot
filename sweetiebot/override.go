package sweetiebot

import (
        "archive/tar"
        "archive/zip"
        "compress/gzip"
        "fmt"
        "io"
        "io/ioutil"
        "os"
        "os/exec"
        "path/filepath"
        "runtime"
        "strings"
)

// CheckForUpdate does a git pull to check for an update in the main instance
func (b *Selfhost) CheckForUpdate(userID DiscordUser, oldversion int) (int8, *UpdateStatus) {
        path, err := GetCurrentDir()
        if err != nil {
                return 0, nil
        }

        // Check if git has an actual update to do
        out, err := RunCommand("git", filepath.Dir(path), "pull")
        if err != nil {
                return 0, nil
        }
        out = strings.ToLower(strings.TrimSpace(out))
        if out == "already up-to-date." || out == "already up to date." {
                return 0, nil
        }
        return 1, &UpdateStatus{Version: BotVersion.Integer()}
}

func WriteMD5(path string) (bytes []byte, err error) {
        bytes, err = calcMD5Hash(path)
        if err == nil {
                err = ioutil.WriteFile(path+".md5", bytes, 0644)
        }
        return
}
func GetMD5(path string) (bytes []byte, err error) {
        bytes, err = ioutil.ReadFile(path + ".md5")
        if err == nil && len(bytes) == 16 {
                return
        }
        bytes, err = WriteMD5(path)
        return
}

// SelfUpdate in the main instance builds a new version of the updater and then all the versions of the main EXE
func (b *Selfhost) SelfUpdate(ownerid DiscordUser) {
        path, err := GetCurrentDir()
        parent := "../docs/"
        if err == nil {
                parent = filepath.Join(filepath.Dir(path), "docs")
        }
        name := "updater" + getExt(runtime.GOOS)
        os.Remove("~" + name)
        os.Rename(name, "~"+name)
        RunCommand("go", path, "build", "-tags", "purego", "../updater")
        if _, err := os.Stat(name); os.IsNotExist(err) {
                os.Rename("~"+name, name)
        }

        // Rewrite MD5 hashes for all .sql files in case we updated them
        WriteMD5(filepath.Join(parent, "sweetiebot.sql"))
        WriteMD5(filepath.Join(parent, "sweetiebot_tz.sql"))
        WriteMD5(filepath.Join(parent, "legacy_migrate.sql"))
        WriteMD5(filepath.Join(parent, "web.css"))
        WriteMD5(filepath.Join(parent, "web.html"))
        WriteMD5(filepath.Join(parent, "sweetiebot.svg"))
        results, _ := ioutil.ReadDir(parent)
        for _, f := range results {
                if !f.IsDir() {
                        if sqlfileregex.MatchString(f.Name()) {
                                WriteMD5(filepath.Join(parent, f.Name()))
                        }
                }
        }

        pairs := [][2]string{{"windows", "386"}, {"windows", "amd64"}, {"linux", "386"}, {"linux", "amd64"}}
        curenv := os.Environ()

        // Compile sweetiebot for all architecture/platform pairs
        for _, pair := range pairs {
                name := filepath.Join(parent, "update", pair[0], pair[1])
                env := append(curenv, "GOOS="+pair[0], "GOARCH="+pair[1])
                if err := os.MkdirAll(name, 0775); err != os.ErrExist && err != nil {
                        fmt.Println(err)
                }
                fmt.Println("Compiling " + pair[0] + "-" + pair[1])
                ExecCommand(exec.Command("go", "build", "-tags", "purego", "../../../../updater"), name, env...)
                ExecCommand(exec.Command("go", "build", "-tags", "purego", "../../../../sweetie"), name, env...)
                WriteMD5(filepath.Join(name, "updater"+getExt(pair[0])))
                target := filepath.Join(name, "sweetie"+getExt(pair[0]))
                WriteMD5(target)

                fmt.Println("Compressing " + pair[0] + "-" + pair[1])
                var f *os.File
                if pair[0] == "windows" {
                        if f, err = os.Create(filepath.Join(name, "sweetie.zip")); err == nil {
                                w := zip.NewWriter(f)
                                if zw, err := w.Create("sweetie.exe"); err == nil {
                                        var buf *os.File
                                        if buf, err = os.Open(target); err == nil {
                                                io.Copy(zw, buf)
                                                buf.Close()
                                        }
                                }
                                w.Close()
                                f.Close()
                        }
                } else {
                        if f, err = os.Create(filepath.Join(name, "sweetie.tar.gz")); err == nil {
                                w := gzip.NewWriter(f)
                                tw := tar.NewWriter(w)

                                var buf *os.File
                                if buf, err = os.Open(target); err == nil {
                                        var stat os.FileInfo
                                        if stat, err = buf.Stat(); err == nil {
                                                header := new(tar.Header)
                                                header.Name = stat.Name()
                                                header.Size = stat.Size()
                                                header.Mode = 0777
                                                header.ModTime = stat.ModTime()
                                                if err = tw.WriteHeader(header); err == nil {
                                                        _, err = io.Copy(tw, buf)
                                                }
                                        }
                                        buf.Close()
                                }
                                tw.Close()
                                w.Close()
                                f.Close()
                        }
                }
                if err != nil {
                        fmt.Println(err)
                }
                fmt.Println("Finished " + pair[0] + "-" + pair[1])
        }

        b.ready.Set(true)
}

// DoUpdate in the main instance does another git pull (just in case) and builds a new version of the main EXE
func (b *Selfhost) DoUpdate(dbauth string, token string) error {
        path, err := GetCurrentDir()
        parent := "../"
        if err == nil {
                parent = filepath.Dir(path)
        }

        RunCommand("git", parent, "pull")

        if b.UpgradeDatabase(parent, dbauth) == nil {
                fmt.Println("Building Sweetie Bot")
                RunCommand("go", path, "build", "-tags", "purego", "../sweetie")
        }
        return nil
}

func (b *Selfhost) GetWebDir() string {
        if dir, err := GetCurrentDir(); err == nil {
                return filepath.Dir(dir) + "/docs"
        }
        return "../docs"
}