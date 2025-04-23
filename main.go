package main

import (
	"log"
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

func main() {
	// Create a new Fyne application
	a := app.New()
	w := a.NewWindow("Media Control")

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

	// Create a label and button
	label := widget.NewLabel("Media Control 0.1 Testing")
	playButton := widget.NewButton("Play", func() {
		keyPressOnce(vk.VK_MEDIA_PLAY_PAUSE)
	})

	// Create a container with the label and button
	content := container.NewVBox(
		label,
		playButton,
	)

	// Set the window content
	w.SetContent(content)

	// Set window size
	w.Resize(fyne.NewSize(300, 110))

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
