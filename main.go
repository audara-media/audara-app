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
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	logFile, err := os.OpenFile(filepath.Join(wd, "mediacontrol.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

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
	config, err := loadConfig()
	if err != nil {
		log.Printf("Error loading config: %v", err)
		return
	}

	a := app.New()
	w := a.NewWindow(config.App.Name)

	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %v", err)
	}

	iconPath := filepath.Join(wd, "temp-play.png")
	icon, err := fyne.LoadResourceFromPath(iconPath)
	if err != nil {
		log.Printf("Error loading icon: %v", err)
	} else {
		w.SetIcon(icon)
	}

	label := widget.NewLabel("Audara Pre-MVP baby")
	playButton := widget.NewButton("Play", func() {
		keyPressOnce(vk.VK_MEDIA_PLAY_PAUSE)
	})
	playButton.Disable()

	userInfo := widget.NewLabel("")
	authButton := widget.NewButton("Login", nil)
	loadingLabel := widget.NewLabel("")
	loadingLabel.Hide()

	var loginHandler func()
	var cancelAuth func()

	updateUI := func(userData *auth.UserData) {
		fyne.Do(func() {
			if userData != nil {
				var displayName string
				if userData.Profile.FirstName != "" {
					displayName = userData.Profile.FirstName
				} else if userData.Username != "" {
					displayName = userData.Username
				} else if len(userData.Profile.EmailAddresses) > 0 {
					displayName = userData.Profile.EmailAddresses[0]
				} else {
					displayName = "Logged in"
				}
				userInfo.SetText(displayName)
				authButton.SetText("Logout")
				playButton.Enable()
				authButton.OnTapped = func() {
					if err := os.Remove(config.App.Auth.TokenFile); err != nil {
						log.Printf("Error removing token file: %v", err)
					}
					userInfo.SetText("")
					authButton.SetText("Login")
					playButton.Disable()
					authButton.OnTapped = loginHandler
				}
			} else {
				userInfo.SetText("")
				authButton.SetText("Login")
				playButton.Disable()
				authButton.OnTapped = loginHandler
			}
		})
	}

	loginHandler = func() {
		resultChan, cancel := auth.StartAuthProcess(config.App.Auth.WebappURL, 3001)
		cancelAuth = cancel

		loadingLabel.SetText("Opening browser...")
		loadingLabel.Show()
		authButton.SetText("Cancel")
		authButton.OnTapped = func() {
			if cancelAuth != nil {
				cancelAuth()
				cancelAuth = nil
			}

			loadingLabel.Hide()
			authButton.SetText("Login")
			authButton.OnTapped = loginHandler
		}

		go func() {
			result := <-resultChan

			fyne.Do(func() {
				loadingLabel.Hide()
				authButton.SetText("Login")
				authButton.OnTapped = loginHandler
			})

			if result.Error != nil {
				log.Printf("Authentication error: %v", result.Error)
				return
			}

			if err := auth.SaveToken(result.Token, config.App.Auth.TokenFile); err != nil {
				log.Printf("Error saving token: %v", err)
				return
			}

			userData, err := auth.VerifyToken(result.Token, config.App.Auth.WebappURL)
			if err != nil {
				log.Printf("Error verifying token: %v", err)
				return
			}

			updateUI(userData)
			log.Printf("Successfully authenticated")
		}()
	}

	authButton.OnTapped = loginHandler

	if token, err := auth.LoadToken(config.App.Auth.TokenFile); err == nil {
		var username string
		if token.Profile.Username != nil {
			username = *token.Profile.Username
		}
		userData := &auth.UserData{
			Username: username,
			Profile:  token.Profile,
		}
		fyne.Do(func() {
			updateUI(userData)
			playButton.Enable()
		})
	}

	content := container.NewVBox(
		label,
		container.NewHBox(userInfo, loadingLabel, authButton),
		playButton,
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(300, 150))

	if desk, ok := a.(desktop.App); ok {
		menu := fyne.NewMenu("Audara",
			fyne.NewMenuItem("Open App", func() {
				w.Show()
			}),
			fyne.NewMenuItem("Exit", func() {
				a.Quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)
		if icon != nil {
			desk.SetSystemTrayIcon(icon)
		}
	}

	w.Hide()
	w.ShowAndRun()
}
