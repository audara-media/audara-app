package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Config struct {
	WebappURL string `yaml:"webapp_url"`
	TokenFile string `yaml:"token_file"`
}

type Profile struct {
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	ImageURL       string   `json:"imageUrl"`
	EmailAddresses []string `json:"emailAddresses"`
	Username       *string  `json:"username"`
}

type TokenResponse struct {
	SessionToken string  `json:"sessionToken"`
	UserID       string  `json:"userId"`
	SessionID    string  `json:"sessionId"`
	Profile      Profile `json:"profile"`
}

type UserData struct {
	Username string  `json:"username"`
	Profile  Profile `json:"profile"`
}

type AuthResult struct {
	Token *TokenResponse
	Error error
}

var (
	callbackRegistered bool
	callbackMutex      sync.Mutex
	rnd                = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// generateShortCode creates a random 6-character string for request matching
func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[rnd.Intn(len(charset))]
	}
	return string(b)
}

func OpenAuthURL(url string) (string, error) {
	// Generate a short code for request matching
	defcode := generateShortCode()

	// Add the defcode to the URL
	authURL := url + "?defcode=" + defcode

	// Open default browser with the auth URL
	return defcode, openBrowser(authURL)
}

func WaitForAuthCallback(port int, webappURL string, defcode string) (*TokenResponse, error) {
	callbackMutex.Lock()
	if callbackRegistered {
		callbackMutex.Unlock()
		return nil, fmt.Errorf("callback handler already registered")
	}
	callbackRegistered = true
	callbackMutex.Unlock()

	// Create channels for token and error
	tokenChan := make(chan *TokenResponse)
	errorChan := make(chan error)

	// Start long-polling for token
	go func() {
		client := &http.Client{
			Timeout: 60 * time.Second,
		}

		req, err := http.NewRequest("GET", webappURL+"/api/gettoken?defcode="+defcode, nil)
		if err != nil {
			errorChan <- fmt.Errorf("error creating request: %v", err)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			errorChan <- fmt.Errorf("error waiting for token: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errorChan <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			return
		}

		var token TokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
			errorChan <- fmt.Errorf("error decoding token response: %v", err)
			return
		}

		tokenChan <- &token
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

	req.Header.Set("Authorization", "Bearer "+token.SessionToken)
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

// resetCallbackRegistration resets the callback registration flag
func resetCallbackRegistration() {
	callbackMutex.Lock()
	callbackRegistered = false
	callbackMutex.Unlock()
}

func StartAuthProcess(webappURL string, port int) (chan AuthResult, func()) {
	resultChan := make(chan AuthResult)
	done := make(chan struct{})

	go func() {
		// Open the auth URL in browser and get the defcode
		defcode, err := OpenAuthURL(webappURL + "/auth/callback")
		if err != nil {
			resetCallbackRegistration()
			resultChan <- AuthResult{Error: fmt.Errorf("error opening auth URL: %v", err)}
			return
		}

		// Wait for auth callback
		token, err := WaitForAuthCallback(port, webappURL, defcode)
		if err != nil {
			resetCallbackRegistration()
			resultChan <- AuthResult{Error: err}
			return
		}

		// Check if token is nil
		if token == nil {
			resetCallbackRegistration()
			resultChan <- AuthResult{Error: fmt.Errorf("received nil token")}
			return
		}

		// Verify the token
		if _, err := VerifyToken(token, webappURL); err != nil {
			// If we get a 403, don't write the token file
			if strings.Contains(err.Error(), "403") {
				resetCallbackRegistration()
				resultChan <- AuthResult{Error: fmt.Errorf("authentication failed: %v", err)}
				return
			}
			resetCallbackRegistration()
			resultChan <- AuthResult{Error: err}
			return
		}

		// Save the token
		if err := SaveToken(token, "auth_token.json"); err != nil {
			resetCallbackRegistration()
			resultChan <- AuthResult{Error: err}
			return
		}

		resetCallbackRegistration()
		resultChan <- AuthResult{Token: token}
	}()

	// Return the result channel and a cancel function
	return resultChan, func() {
		resetCallbackRegistration()
		close(done)
	}
}
