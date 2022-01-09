package server

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dghubble/gologin/v2"
	"github.com/dghubble/gologin/v2/github"
	"github.com/dghubble/sessions"
	"github.com/majst01/metal-dns/pkg/server/templates"
	"golang.org/x/oauth2"
	githubOAuth2 "golang.org/x/oauth2/github"
)

const (
	sessionName     = "metal-dns-github-app"
	sessionSecret   = "example cookie signing secret"
	sessionUserKey  = "githubID"
	sessionUsername = "githubUsername"
)

// sessionStore encodes and decodes session data stored in signed cookies
var sessionStore = sessions.NewCookieStore([]byte(sessionSecret), nil)

// Config configures the main ServeMux.
type LoginConfig struct {
	GithubClientID     string
	GithubClientSecret string
}

// New returns a new ServeMux with app routes.
func newLoginServer(config *LoginConfig) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", profileHandler)
	mux.HandleFunc("/logout", logoutHandler)
	// 1. Register LoginHandler and CallbackHandler
	oauth2Config := &oauth2.Config{
		ClientID:     config.GithubClientID,
		ClientSecret: config.GithubClientSecret,
		RedirectURL:  "http://localhost:8080/github/callback",
		Endpoint:     githubOAuth2.Endpoint,
	}
	// state param cookies require HTTPS by default; disable for localhost development
	stateConfig := gologin.DebugOnlyCookieConfig
	mux.Handle("/github/login", github.StateHandler(stateConfig, github.LoginHandler(oauth2Config, nil)))
	mux.Handle("/github/callback", github.StateHandler(stateConfig, github.CallbackHandler(oauth2Config, issueSession(), nil)))
	return mux
}

// issueSession issues a cookie session after successful Github login
func issueSession() http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		githubUser, err := github.UserFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 2. Implement a success handler to issue some form of session
		session := sessionStore.New(sessionName)
		session.Values[sessionUserKey] = *githubUser.ID
		session.Values[sessionUsername] = *githubUser.Login
		err = session.Save(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, req, "/profile", http.StatusFound)
		if githubUser.Login != nil {
			log.Printf("logged in as: Login:%s\n", *githubUser.Login)
		}
		if githubUser.ID != nil {
			log.Printf("logged in as: ID:%d\n", *githubUser.ID)
		}
		if githubUser.Email != nil {
			log.Printf("logged in as: email:%s\n", *githubUser.Email)
		}
	}
	return http.HandlerFunc(fn)
}

// profileHandler shows a personal profile or a login button (unauthenticated).
func profileHandler(w http.ResponseWriter, req *http.Request) {
	session, err := sessionStore.Get(req, sessionName)
	if err != nil {
		// welcome with login button
		page, _ := templates.Templates.ReadFile("index.html")
		fmt.Fprint(w, string(page))
		return
	}
	// authenticated profile
	overview, _ := templates.Templates.ReadFile("overview.html")
	fmt.Fprintf(w, string(overview), session.Values[sessionUsername])
}

// logoutHandler destroys the session on POSTs and redirects to home.
func logoutHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		sessionStore.Destroy(w, sessionName)
	}
	http.Redirect(w, req, "/", http.StatusFound)
}

// StartLoginService starts a http endpoint to handle social login
func (s *Server) StartLoginService() {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.HTTPPort)
	// read credentials from environment variables if available
	config := &LoginConfig{
		GithubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GithubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
	}
	// allow consumer credential flags to override config fields
	clientID := flag.String("client-id", "", "Github Client ID")
	clientSecret := flag.String("client-secret", "", "Github Client Secret")
	flag.Parse()
	if *clientID != "" {
		config.GithubClientID = *clientID
	}
	if *clientSecret != "" {
		config.GithubClientSecret = *clientSecret
	}
	if config.GithubClientID == "" {
		log.Fatal("Missing Github Client ID")
	}
	if config.GithubClientSecret == "" {
		log.Fatal("Missing Github Client Secret")
	}

	go func() {
		log.Printf("Starting Server listening on %s\n", addr)
		err := http.ListenAndServe(addr, newLoginServer(config))
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}()
}
