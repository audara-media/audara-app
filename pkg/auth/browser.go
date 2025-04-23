package auth

import (
	"os/exec"
	"runtime"
)

func openBrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "windows":
		err = exec.Command("cmd", "/c", "start", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default: // "linux", "freebsd", "openbsd", "netbsd"
		err = exec.Command("xdg-open", url).Start()
	}

	return err
}
