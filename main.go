package main

import (
	"log"
	vk "mediacontrol/pkg/winVirtualKeyCodes"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32        = syscall.NewLazyDLL("user32.dll")
	sendInputProc = user32.NewProc("SendInput")
)

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
	keyPressOnce(vk.VK_MEDIA_PLAY_PAUSE)
	time.Sleep(time.Second)
	keyPressOnce(vk.VK_MEDIA_NEXT_TRACK)
}
