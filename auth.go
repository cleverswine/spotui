package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789_"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// SpotifyClientBuilderConfig is the configuration that ClientBuilder uses to initialize a Spotify client
type SpotifyClientBuilderConfig struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
	LocalPort    string
	TokenFile    string
}

// SpotifyClientBuilder builds an authenticated Spotify client
type SpotifyClientBuilder struct {
	Config *SpotifyClientBuilderConfig
	auth   spotify.Authenticator
	state  string
	ch     chan *spotify.Client
}

// NewSpotifyClientBuilder creates a new ClientBuilder with optional config
func NewSpotifyClientBuilder(config *SpotifyClientBuilderConfig) *SpotifyClientBuilder {
	c := &SpotifyClientBuilder{}
	if config == nil {
		c.Config = &SpotifyClientBuilderConfig{
			Scopes: []string{spotify.ScopeUserFollowRead,
				spotify.ScopeUserLibraryRead, spotify.ScopeUserLibraryModify,
				spotify.ScopePlaylistReadPrivate, spotify.ScopePlaylistModifyPrivate,
				spotify.ScopePlaylistReadCollaborative, spotify.ScopePlaylistModifyPublic,
				spotify.ScopeUserReadPrivate,
			},
			LocalPort: "8080",
			TokenFile: "token.json",
		}
	} else {
		c.Config = config
	}
	c.auth = spotify.NewAuthenticator(fmt.Sprintf("http://localhost:%s/callback", c.Config.LocalPort), c.Config.Scopes...)
	// ClientID:     os.Getenv("SPOTIFY_ID"),
	// ClientSecret: os.Getenv("SPOTIFY_SECRET")
	if c.Config.ClientID != "" && c.Config.ClientSecret != "" {
		c.auth.SetAuthInfo(c.Config.ClientID, c.Config.ClientSecret)
	}
	c.state = randStringBytes(40)
	c.ch = make(chan *spotify.Client)
	return c
}

// GetClientWithJSONToken uses a serialized oauth2 token to get an authenticated Spotify client
func (c *SpotifyClientBuilder) GetClientWithJSONToken(jsonToken []byte) (*spotify.Client, error) {
	tok := oauth2.Token{}
	err := json.Unmarshal(jsonToken, &tok)
	if err != nil {
		return nil, err
	}
	client := c.auth.NewClient(&tok)
	return &client, nil
}

// GetClient uses the oauth2 flow to get an authenticated Spotify client
func (c *SpotifyClientBuilder) GetClient() (*spotify.Client, error) {
	// try to get from file first
	client, err := c.getClientWithTokenFile()
	if err != nil {
		return nil, err
	}
	if client != nil {
		return client, nil
	}
	// start an HTTP server and initiate an oidc flow
	http.HandleFunc("/callback", c.completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)
	url := c.auth.AuthURL(c.state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)
	// wait for auth to complete
	client = <-c.ch
	return client, nil
}

// SaveToken serializes the oauth2 token to file
func (c *SpotifyClientBuilder) SaveToken(client *spotify.Client) error {
	tok, err := client.Token()
	if err != nil {
		return fmt.Errorf("unable to get token from client: %v", err)
	}
	tokb, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("unable to serialize token: %v", err)
	}
	err = ioutil.WriteFile(c.Config.TokenFile, tokb, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to save token: %v", err)
	}
	return nil
}

func (c *SpotifyClientBuilder) completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := c.auth.Token(c.state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != c.state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, c.state)
	}
	// use the token to get an authenticated client
	client := c.auth.NewClient(tok)
	c.SaveToken(&client)
	fmt.Fprintf(w, "Login Completed!")
	c.ch <- &client
}

func (c *SpotifyClientBuilder) getClientWithTokenFile() (*spotify.Client, error) {
	_, err := os.Stat(c.Config.TokenFile)
	if os.IsNotExist(err) {
		log.Printf("Token file %s does not exist\n", c.Config.TokenFile)
		return nil, nil
	}
	f, err := os.Open(c.Config.TokenFile)
	if err != nil {
		return nil, err
	}
	jsonToken, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return c.GetClientWithJSONToken(jsonToken)
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
