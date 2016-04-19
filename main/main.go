package main

import (
  "../sweetiebot"
  "golang.org/x/oauth2"
  "github.com/gorilla/handlers"
  "github.com/bwmarrin/discordgo"
  "net/http"
  "fmt"
  "io/ioutil"
  "os"
  "encoding/json"
  "time"
  "math/rand"
  "bytes"
)

var (
	// Permission Constants
	READ_MESSAGES = 1024
	SEND_MESSAGES = 2048
	CONNECT       = 1048576
	SPEAK         = 2097152
  
	oauthConf *oauth2.Config
	apiBaseUrl = "https://discordapp.com/api"
  oauthState string
  letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

// Return a random character sequence of n length
func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	oauthState = randSeq(32)
	perms := READ_MESSAGES | SEND_MESSAGES | CONNECT | SPEAK

	// Return a redirect to the ouath provider
	url := oauthConf.AuthCodeURL(oauthState, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url+fmt.Sprintf("&permissions=%v", perms), http.StatusTemporaryRedirect)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	// Check the state string is correct
	state := r.FormValue("state")
	if state != oauthState {
		fmt.Println("Invalid OAuth state")
		return
	}

	errorMsg := r.FormValue("error")
	if errorMsg != "" {
		fmt.Println("Received OAuth error from provider")
		return
	}

	token, err := oauthConf.Exchange(oauth2.NoContext, r.FormValue("code"))
	if err != nil {
		fmt.Println("Failed to exchange token with provider")
		return
	}

	body, _ := json.Marshal(map[interface{}]interface{}{})
	req, err := http.NewRequest("GET", apiBaseUrl+"/users/@me", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Failed to create @me request")
		return
	}

	req.Header.Set("Authorization", token.Type()+" "+token.AccessToken)
	client := &http.Client{Timeout: (20 * time.Second)}
	resp, err := client.Do(req)
	if err != nil {
    fmt.Println("Failed to request @me data");
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
    fmt.Println("Failed to read data from HTTP response");
		return
	}

	user := discordgo.User{}
	err = json.Unmarshal(respBody, &user)
	if err != nil {
    fmt.Println("Failed to parse JSON payload from HTTP response");
		return
	}

	// Finally write some information to the session store
  fmt.Println("Token: ", token.AccessToken);
  fmt.Println("username: ", user.Username);
  fmt.Println("tag: ", user.Discriminator);
}

func handleMe(w http.ResponseWriter, r *http.Request) {
}

func setUpOAuth() {
  clientid, _ := ioutil.ReadFile("client_id")  
  clientsecret, _ := ioutil.ReadFile("client_secret")
  
  // Setup the OAuth2 Configuration
	endpoint := oauth2.Endpoint{
		AuthURL:  apiBaseUrl + "/oauth2/authorize",
		TokenURL: apiBaseUrl + "/oauth2/token",
	}

	oauthConf = &oauth2.Config{
		ClientID:     string(clientid),
		ClientSecret: string(clientsecret),
		Scopes:       []string{"bot", "identify"},
		Endpoint:     endpoint,
    RedirectURL:  "http://localhost:5000/callback",
		//RedirectURL:  "https://airhornbot.com/callback",
	}
}

func server() {
	server := http.NewServeMux()
	server.HandleFunc("/me", handleMe)
	server.HandleFunc("/login", handleLogin)
	server.HandleFunc("/callback", handleCallback)
  
  fmt.Println("Starting http server on port :5000");

	// If the requests log doesnt exist, make it
	if _, err := os.Stat("requests.log"); os.IsNotExist(err) {
		ioutil.WriteFile("requests.log", []byte{}, 0600)
	}

	// Open the log file in append mode
	logFile, err := os.OpenFile("requests.log", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
    fmt.Println("Failed to open requests log file");
		return
	}
	defer logFile.Close()

	// Actually start the server
	loggedRouter := handlers.LoggingHandler(logFile, server)
	http.ListenAndServe(":5000", loggedRouter)
}
func main() {
  setUpOAuth()
  server()
  sweetiebot.Initialize("")
}