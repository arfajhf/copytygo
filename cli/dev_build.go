package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// nativeDevBinaryPath uses a predictable project-local location rather than
// go run's temporary directory. This does not override Windows security policy.
func nativeDevBinaryPath(goos string) string {
	name := "app"
	if goos == "windows" {
		name += ".exe"
	}
	return filepath.Join("build", "dev", name)
}

// runNativeBuildDev builds a real native application binary and runs it from
// the project directory. The TypeScript dev server is still started by the
// regular frontend integration and continues to use the Go app origin.
func runNativeBuildDev() error {
	output := nativeDevBinaryPath(runtime.GOOS)
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return fmt.Errorf("copytygo native dev build directory: %w", err)
	}
	fmt.Println("Building native development application:", output)
	build := exec.Command("go", "build", "-o", output, "./cmd/app")
	build.Stdout = os.Stdout
	var buildErr bytes.Buffer
	build.Stderr = &buildErr
	if err := build.Run(); err != nil {
		return fmt.Errorf("copytygo native dev compilation: %w: %s", err, strings.TrimSpace(buildErr.String()))
	}
	env, cleanup, err := startAuthFrontend()
	if err != nil {
		return err
	}
	defer cleanup()

	absolute, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	cmd := exec.Command(absolute)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nativeExecutionError(err, stderr.String(), runtime.GOOS)
	}
	return nil
}

// nativeExecutionError keeps the original error and adds an actionable
// diagnosis only when there is evidence of a Windows application-policy block.
func nativeExecutionError(err error, stderr, goos string) error {
	details := strings.TrimSpace(stderr)
	message := fmt.Errorf("copytygo native runtime: %w: %s", err, details)
	if goos != "windows" || !isApplicationPolicyBlock(err.Error()+" "+details) {
		return message
	}
	return fmt.Errorf("%w\nWindows appears to have blocked the generated executable. Check Windows Security > Protection history and the Microsoft-Windows-CodeIntegrity/Operational log (event 3077). See docs/WINDOWS_DEVELOPMENT.md. A fixed build location does not bypass Smart App Control or managed policy. Use an approved signing or allow policy if required; do not disable protection globally", message)
}

func isApplicationPolicyBlock(message string) bool {
	s := strings.ToLower(message)
	for _, fragment := range []string{
		"application control policy has blocked",
		"blocked by your system administrator",
		"smart app control has blocked",
		"windows defender application control",
		"this app has been blocked for your protection",
	} {
		if strings.Contains(s, fragment) {
			return true
		}
	}
	return false
}
