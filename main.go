package main

import (
	"log"
	"mediacontrol/pkg/auth"
	vk "mediacontrol/pkg/winVirtualKeyCodes"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v3"
)

var (
	user32        = syscall.NewLazyDLL("user32.dll")
	sendInputProc = user32.NewProc("SendInput")
)

func init() {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Create log file
	logFile, err := os.OpenFile(filepath.Join(wd, "mediacontrol.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	// Set log output to file
	log.SetOutput(logFile)
}

func keyPressOnce(keyCode uint16) {
	type keyboardInput struct {
		wVk         uint16
		wScan       uint16
		dwFlags     uint32
		time        uint32
		dwExtraInfo uint64
	}

	type input struct {
		inputType uint32
		ki        keyboardInput
		padding   uint64
	}

	var i input
	i.inputType = 1 //INPUT_KEYBOARD
	i.ki.wVk = keyCode
	ret, _, err := sendInputProc.Call(
		uintptr(1),
		uintptr(unsafe.Pointer(&i)),
		uintptr(unsafe.Sizeof(i)),
	)
	log.Printf("ret: %v error: %v", ret, err)
}

type AppConfig struct {
	App struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		Auth    struct {
			WebappURL string `yaml:"webapp_url"`
			TokenFile string `yaml:"token_file"`
		} `yaml:"auth"`
	} `yaml:"app"`
}

func loadConfig() (*AppConfig, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(wd, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Printf("Error loading config: %v", err)
		return
	}

	// Create a new Fyne application
	a := app.New()
	w := a.NewWindow(config.App.Name)

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %v", err)
	}

	// Load and set the icon
	iconPath := filepath.Join(wd, "icon.png")
	icon, err := fyne.LoadResourceFromPath(iconPath)
	if err != nil {
		log.Printf("Error loading icon: %v", err)
	} else {
		w.SetIcon(icon)
	}

	// Create a label and buttons
	label := widget.NewLabel("Media Control 0.1 Testing")
	playButton := widget.NewButton("Play", func() {
		keyPressOnce(vk.VK_MEDIA_PLAY_PAUSE)
	})

	// Create a container for user info and auth button
	userInfo := widget.NewLabel("")
	authButton := widget.NewButton("Login", nil)
	loadingLabel := widget.NewLabel("")
	loadingLabel.Hide()

	// Declare loginHandler variable
	var loginHandler func()
	var cancelAuth func()

	// Function to update UI based on auth state
	updateUI := func(userData *auth.UserData) {
		if userData != nil {
			userInfo.SetText(userData.Username)
			authButton.SetText("Logout")
			authButton.OnTapped = func() {
				// Delete token file
				if err := os.Remove(config.App.Auth.TokenFile); err != nil {
					log.Printf("Error removing token file: %v", err)
				}
				// Reset UI
				userInfo.SetText("")
				authButton.SetText("Login")
				authButton.OnTapped = loginHandler
			}
		} else {
			userInfo.SetText("")
			authButton.SetText("Login")
			authButton.OnTapped = loginHandler
		}
	}

	// Define login handler function
	loginHandler = func() {
		// Start the auth process
		resultChan, cancel := auth.StartAuthProcess(config.App.Auth.WebappURL, 3001)
		cancelAuth = cancel

		// Update UI to show loading state
		loadingLabel.SetText("Opening browser...")
		loadingLabel.Show()
		authButton.SetText("Cancel")
		authButton.OnTapped = func() {
			if cancelAuth != nil {
				cancelAuth()
				cancelAuth = nil
			}
			// Reset UI
			loadingLabel.Hide()
			authButton.SetText("Login")
			authButton.OnTapped = loginHandler
		}

		// Handle auth result in a goroutine
		go func() {
			result := <-resultChan

			// Reset UI first
			loadingLabel.Hide()
			authButton.SetText("Login")
			authButton.OnTapped = loginHandler

			if result.Error != nil {
				log.Printf("Authentication error: %v", result.Error)
				return
			}

			// Save the token
			if err := auth.SaveToken(result.Token, config.App.Auth.TokenFile); err != nil {
				log.Printf("Error saving token: %v", err)
				return
			}

			// Verify token and get user data
			userData, err := auth.VerifyToken(result.Token, config.App.Auth.WebappURL)
			if err != nil {
				log.Printf("Error verifying token: %v", err)
				return
			}

			// Update UI with user data
			updateUI(userData)
			log.Printf("Successfully authenticated")
		}()
	}

	// Set initial auth button handler
	authButton.OnTapped = loginHandler

	// Check for existing token on startup
	if token, err := auth.LoadToken(config.App.Auth.TokenFile); err == nil {
		userData, err := auth.VerifyToken(token, config.App.Auth.WebappURL)
		if err != nil {
			log.Printf("Error verifying existing token: %v", err)
			// Delete invalid token
			if err := os.Remove(config.App.Auth.TokenFile); err != nil {
				log.Printf("Error removing invalid token file: %v", err)
			}
		} else {
			updateUI(userData)
		}
	}

	// Create a container with the label and buttons
	content := container.NewVBox(
		label,
		playButton,
		container.NewHBox(userInfo, loadingLabel, authButton),
	)

	// Set the window content
	w.SetContent(content)

	// Set window size
	w.Resize(fyne.NewSize(300, 150))

	// Create system tray icon
	if desk, ok := a.(desktop.App); ok {
		menu := fyne.NewMenu("Media Control",
			fyne.NewMenuItem("Show", func() {
				w.Show()
			}),
			fyne.NewMenuItem("Quit", func() {
				a.Quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)
		if icon != nil {
			desk.SetSystemTrayIcon(icon)
		}
	}

	// Hide the window initially
	w.Hide()

	// Show and run the application
	w.ShowAndRun()
}
