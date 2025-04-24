package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	WebappURL string `yaml:"webapp_url"`
	TokenFile string `yaml:"token_file"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

type UserData struct {
	Username string `json:"username"`
	// Add other user fields as needed
}

type AuthResult struct {
	Token *TokenResponse
	Error error
}

var (
	callbackRegistered bool
	callbackMutex      sync.Mutex
)

func OpenAuthURL(url string) error {
	// Open default browser with the auth URL
	return openBrowser(url)
}

func WaitForAuthCallback(port int) (*TokenResponse, error) {
	callbackMutex.Lock()
	if callbackRegistered {
		callbackMutex.Unlock()
		return nil, fmt.Errorf("callback handler already registered")
	}
	callbackRegistered = true
	callbackMutex.Unlock()

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

func VerifyToken(token *TokenResponse, webappURL string) (*UserData, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", webappURL+"/api/checktoken", nil)
	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.Token)
	log.Printf("Making HTTP request to %s", req.URL.String())

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making HTTP request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("Received response with status code: %d", resp.StatusCode)

	if resp.StatusCode == http.StatusForbidden {
		log.Printf("Token verification failed: Invalid token (403 Forbidden)")
		return nil, fmt.Errorf("invalid token")
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Token verification failed: Unexpected status code %d", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userData UserData
	if err := json.NewDecoder(resp.Body).Decode(&userData); err != nil {
		log.Printf("Error decoding response body: %v", err)
		return nil, err
	}

	log.Printf("Token verification successful. Username: %s", userData.Username)
	return &userData, nil
}

func StartAuthProcess(webappURL string, port int) (chan AuthResult, func()) {
	resultChan := make(chan AuthResult)
	done := make(chan struct{})

	go func() {
		// Open the auth URL in browser
		if err := OpenAuthURL(webappURL + "/auth/callback"); err != nil {
			resultChan <- AuthResult{Error: fmt.Errorf("error opening auth URL: %v", err)}
			return
		}

		// Wait for auth callback
		token, err := WaitForAuthCallback(port)
		if err != nil {
			resultChan <- AuthResult{Error: err}
			return
		}

		resultChan <- AuthResult{Token: token}
	}()

	// Return the result channel and a cancel function
	return resultChan, func() {
		close(done)
	}
}
