package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// wslNativeDevArgs keeps all Go compilation inside WSL. No Windows-native Go
// executable is invoked by this command. The argument to sh is constant,
// never derived from user input.
func wslNativeDevArgs(projectDirectory string) []string {
	return []string{
		"--cd", projectDirectory,
		"--exec", "/bin/sh", "-c",
		`PATH="$HOME/go/bin:$PATH"; export PATH
if ! command -v ctg >/dev/null 2>&1; then
  echo "CopyTyGo is not installed inside WSL. Install the Linux CLI and Go toolchain first. See docs/WSL_DEVELOPMENT.md" >&2
  exit 127
fi
exec ctg dev`,
	}
}

// runWSLDev starts the complete native Go application inside the user's
// configured default WSL distribution. It does not alter Windows application
// control policies or degrade to the limited Lite Runtime.
func runWSLDev() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("copytygo: --wsl is only for Windows; run ctg dev directly inside Linux")
	}
	if _, err := exec.LookPath("wsl.exe"); err != nil {
		return fmt.Errorf("copytygo: WSL is unavailable: %w. See docs/WSL_DEVELOPMENT.md", err)
	}
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("copytygo: cannot locate project directory: %w", err)
	}
	fmt.Println("CopyTyGo Dev")
	fmt.Println("-------------")
	fmt.Println("Runtime : wsl-native (Linux Go + Vite, no Windows go.exe)")
	fmt.Println("Run directly in WSL for best performance. See docs/WSL_DEVELOPMENT.md.")
	fmt.Println()
	cmd := exec.Command("wsl.exe", wslNativeDevArgs(dir)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copytygo WSL Native Runtime failed: %w. Check your WSL distro and its Go/Node installation (docs/WSL_DEVELOPMENT.md)", err)
	}
	return nil
}
