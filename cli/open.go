package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/arfajhf/copytygo/v4/config"
)

func OpenApplication(studio bool) error {
	if err := config.LoadEnv(".env"); err != nil {
		return err
	}

	host := strings.TrimSpace(config.Get("APP_HOST", "127.0.0.1"))
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	port := strings.TrimSpace(config.Get("APP_PORT", "8080"))
	url := "http://" + host + ":" + port
	if studio {
		url += "/__copytygo"
	}

	if err := openBrowser(url); err != nil {
		return fmt.Errorf("copytygo: unable to open browser: %w", err)
	}

	fmt.Println("Opened:", url)
	return nil
}

func openBrowser(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		command = exec.Command("open", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	return command.Start()
}
