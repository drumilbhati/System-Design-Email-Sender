package mailer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
)

// getClient retrieves a token, saves the token, then returns the generated client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	// Try to get token from ENV first (Deployment friendly)
	tokenStr := os.Getenv("GMAIL_TOKEN_JSON")
	var tok *oauth2.Token
	var err error

	if tokenStr != "" {
		tok = &oauth2.Token{}
		if err := json.Unmarshal([]byte(tokenStr), tok); err != nil {
			log.Printf("Warning: Could not unmarshal GMAIL_TOKEN_JSON: %v", err)
			tok = nil
		}
	}

	// If not in env, try local file (Local development friendly)
	if tok == nil {
		tok, err = tokenFromFile("token.json")
		if err != nil {
			// If no token, and we are interactive, we could ask for one.
			// But for this setup, we'll assume the user will generate it locally first.
			log.Println("No token found in ENV or token.json. You must generate one locally.")
			tok = getTokenFromWeb(config)
			saveToken("token.json", tok)
		}
	}

	return config.Client(ctx, tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
