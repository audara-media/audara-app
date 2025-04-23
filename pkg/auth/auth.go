package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type Config struct {
	WebappURL string `yaml:"webapp_url"`
	TokenFile string `yaml:"token_file"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

func OpenAuthURL(url string) error {
	// Open default browser with the auth URL
	return openBrowser(url)
}

func WaitForAuthCallback(port int) (*TokenResponse, error) {
	// Create a channel to receive the token
	tokenChan := make(chan *TokenResponse)
	errorChan := make(chan error)

	// Start a local server to receive the callback
	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			errorChan <- fmt.Errorf("no token received")
			return
		}

		tokenChan <- &TokenResponse{Token: token}
		fmt.Fprintf(w, "Authentication successful! You can close this window.")
	})

	// Start the server
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			errorChan <- err
		}
	}()

	// Wait for either the token or an error
	select {
	case token := <-tokenChan:
		return token, nil
	case err := <-errorChan:
		return nil, err
	}
}

func SaveToken(token *TokenResponse, tokenFile string) error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(tokenFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Save the token to file
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}

	return os.WriteFile(tokenFile, data, 0600)
}

func LoadToken(tokenFile string) (*TokenResponse, error) {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	var token TokenResponse
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

func IsAuthenticated(tokenFile string) bool {
	_, err := os.Stat(tokenFile)
	return err == nil
}
