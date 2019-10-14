package sweetiebot

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func fwdhttps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST", "PUT", "PATCH":
		http.Error(w, "HTTPS access required", 400)
		return
	default:
		http.RedirectHandler(fmt.Sprintf("https://%s%s", r.Host, r.RequestURI), http.StatusPermanentRedirect).ServeHTTP(w, r)
	}
}

// Serves documentation pages
func (selfhost *Selfhost) helpHandler(w http.ResponseWriter, r *http.Request) {
	parts := []string{}
	if r.URL != nil {
		parts = splitURL(strings.ToLower(r.URL.Path))
	}
	if r.Method == "GET" {
		if len(parts) == 0 || (len(parts) == 1 && strings.ToLower(parts[0]) == "help") {
			if f, err := os.Open(filepath.Join(selfhost.GetWebDir(), "help", "home", "index.html")); err != nil {
				http.Error(w, "File cache error", http.StatusInternalServerError)
			} else {
				defer f.Close()
				io.Copy(w, f)
			}
		} else if len(parts) == 1 {
			fpath := strings.ToLower(parts[0])
			switch fpath {
			case "web.css":
			case "favicon.ico":
			case "sweetiebot.svg":
				w.Header().Set("Content-Type", "image/svg+xml")
			default:
				http.Error(w, "Page not found", http.StatusNotFound)
				return
			}
			if f, err := os.Open(filepath.Join(selfhost.GetWebDir(), fpath)); err != nil {
				http.Error(w, "Page not found", http.StatusNotFound)
			} else {
				defer f.Close()
				io.Copy(w, f)
			}
		} else if len(parts) == 2 {
			if f, err := os.Open(filepath.Join(selfhost.GetWebDir(), "help", strings.ToLower(parts[1]), "index.html")); err != nil {
				http.Error(w, "Page not found", http.StatusNotFound)
			} else {
				defer f.Close()
				io.Copy(w, f)
			}
		}
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func splitURL(urlpath string) []string {
	paths := strings.Split(urlpath, "/")
	r := []string{}
	for _, v := range paths {
		if len(v) > 0 {
			r = append(r, v)
		}
	}
	return r
}

type webCommand struct {
	URL   string
	Usage CommandUsage
	Info  CommandInfo
}
type webModule struct {
	Name        string
	URL         string
	Description string
	Commands    []webCommand
	Config      map[string]string
}
type webData struct {
	Name    string
	Title   string
	Index   int
	Year    int
	Modules []webModule
}

var codeBlockRegex = regexp.MustCompile("`[^`]+`")
var preBlockRegex = regexp.MustCompile("```[^`]+```")
var boldRegex = regexp.MustCompile("\\*\\*[^*]+\\*\\*")
var newLineRegex = regexp.MustCompile("\n\n")

func (sb *SweetieBot) generateCache(webdir string) *template.Template {
	t := template.Must(template.New("web.html").Funcs(template.FuncMap{
		"parsemarkup": func(str string) template.HTML {
			str = strings.Replace(str, "\n\n", "<p>", -1)
			str = preBlockRegex.ReplaceAllStringFunc(str, func(s string) string { return "<pre>" + s[3:len(s)-3] + "</pre>" })
			str = codeBlockRegex.ReplaceAllStringFunc(str, func(s string) string { return "<code>" + s[1:len(s)-1] + "</code>" })
			str = boldRegex.ReplaceAllStringFunc(str, func(s string) string { return "<b>" + s[2:len(s)-2] + "</b>" })
			return template.HTML(str)
		},
	}).ParseFiles(filepath.Join(webdir, "web.html")))

	data := webData{
		Name:  "Sweetie Bot",
		Title: "",
		Index: -1,
		Year:  time.Now().Year(),
	}

	configonly := []string{"Basic", "Modules"}
	for _, v := range configonly {
		data.Modules = append(data.Modules, webModule{
			Name:        v,
			Description: "",
			URL:         strings.ToLower(v),
			Config:      ConfigHelp[strings.ToLower(v)],
		})
	}

	modules := sb.loader(sb.EmptyGuild)
	for _, m := range modules {
		config, _ := ConfigHelp[strings.ToLower(m.Name())]
		module := webModule{
			Name:        m.Name(),
			Description: m.Description(sb.EmptyGuild),
			URL:         strings.ToLower(m.Name()),
			Config:      config,
		}
		for _, c := range m.Commands() {
			command := webCommand{
				URL:   strings.ToLower(c.Info().Name),
				Usage: *c.Usage(sb.EmptyGuild),
				Info:  *c.Info(),
			}

			module.Commands = append(module.Commands, command)
		}
		data.Modules = append(data.Modules, module)
	}

	os.MkdirAll(webdir+"/help/home", 0775)
	if home, err := os.Create(webdir + "/help/home/index.html"); err == nil {
		defer home.Close()
		if err = t.ExecuteTemplate(home, "home", data); err != nil {
			fmt.Println(err)
		}
	}
	{
		src, _ := os.Open(webdir + "/help/home/index.html")
		dest, _ := os.Create(webdir + "/help/index.html")
		defer src.Close()
		defer dest.Close()
		io.Copy(dest, src)
	}
	{
		src, _ := os.Open(webdir + "/help/home/index.html")
		dest, _ := os.Create(webdir + "/index.html")
		defer src.Close()
		defer dest.Close()
		io.Copy(dest, src)
	}

	for k, m := range data.Modules {
		os.MkdirAll(webdir+"/help/"+m.URL, 0775)
		if cache, err := os.Create(webdir + "/help/" + m.URL + "/index.html"); err == nil {
			defer cache.Close()
			data.Title = m.Name
			data.Index = k
			if err = t.ExecuteTemplate(cache, "module", data); err != nil {
				fmt.Println(err)
			}
		}
	}
	return t
}

// ServeWeb starts a webserver on :80 and optionally on :443. If you're doing a reverse-proxy via nginx, SSL terminates at nginx, so use insecure mode.
func (sb *SweetieBot) ServeWeb() error {
	sb.generateCache(sb.Selfhoster.GetWebDir())

	mux := http.NewServeMux()
	mux.HandleFunc("/", sb.Selfhoster.helpHandler)
	mux.HandleFunc("/help", sb.Selfhoster.helpHandler)
	mux.HandleFunc("/help/", sb.Selfhoster.helpHandler)
	sb.Selfhoster.ConfigureMux(mux)
	if sb.WebSecure {
		go http.ListenAndServe(":80", http.HandlerFunc(fwdhttps))
		m := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(sb.WebDomain),
			Cache:      autocert.DirCache("./.relayd"),
		}

		s := &http.Server{
			IdleTimeout: 5 * time.Minute,
			Addr:        ":433",
			TLSConfig:   &tls.Config{GetCertificate: m.GetCertificate},
			Handler:     mux,
		}
		return s.ListenAndServeTLS("", "")
	}

	s := &http.Server{
		IdleTimeout: 5 * time.Minute,
		Addr:        sb.WebPort,
		Handler:     mux,
	}
	return s.ListenAndServe()
}
