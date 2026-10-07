package cli

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/arfajhf/copytygo/v4/config"
)

func npmCommand(args ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", append([]string{"/c", "npm"}, args...)...)
	}
	return exec.Command("npm", args...)
}
func ensureAuthFrontend() error {
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("copytygo: TypeScript frontend requires Node.js 22+. Install Node.js, then run ctg dev again")
	}
	if fileExists("frontend/node_modules/vite/bin/vite.js") {
		return nil
	}
	fmt.Println("Installing TypeScript frontend dependencies…")
	cmd := npmCommand("install", "--no-audit", "--no-fund")
	cmd.Dir = "frontend"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copytygo: frontend dependency installation failed: %w", err)
	}
	return nil
}
func startAuthFrontend() ([]string, func(), error) {
	cleanup := func() {}
	if !fileExists("frontend/auth.html") {
		return os.Environ(), cleanup, nil
	}
	if err := config.LoadEnv(".env"); err != nil {
		return nil, cleanup, err
	}
	if err := ensureAuthFrontend(); err != nil {
		return nil, cleanup, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, cleanup, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	backendPort := config.Get("APP_PORT", "8080")
	origin := "http://127.0.0.1:" + strconv.Itoa(port)
	cmd := exec.Command("node", "node_modules/vite/bin/vite.js", "--config", "vite.auth.config.ts", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--strictPort")
	cmd.Dir = "frontend"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "COPYTYGO_BACKEND_ORIGIN=http://127.0.0.1:"+backendPort, "COPYTYGO_BACKEND_PORT="+backendPort)
	if err := cmd.Start(); err != nil {
		return nil, cleanup, fmt.Errorf("copytygo: unable to start TypeScript frontend: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	cleanup = func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	client := &http.Client{Timeout: time.Second}
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); {
		select {
		case err := <-done:
			_ = cmd.Process.Kill()
			return nil, func() {}, fmt.Errorf("copytygo: TypeScript frontend stopped: %v", err)
		default:
		}
		response, err := client.Get(origin + "/auth.html")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == 200 {
				fmt.Println("Frontend    : TypeScript (Vite, served through the Go application URL)")
				return append(os.Environ(), "COPYTYGO_VITE_ORIGIN="+origin), cleanup, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	cleanup()
	return nil, func() {}, fmt.Errorf("copytygo: TypeScript frontend did not start within 30 seconds")
}
func buildAuthFrontend() error {
	if !fileExists("frontend/auth.html") {
		return nil
	}
	if err := ensureAuthFrontend(); err != nil {
		return err
	}
	cmd := npmCommand("run", "build", "--", "--config", "vite.auth.config.ts")
	cmd.Dir = "frontend"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copytygo: TypeScript frontend build failed: %w", err)
	}
	// Production starts from build/, alongside the compiled frontend directory.
	return filepath.WalkDir("frontend/dist", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel("frontend/dist", path)
		if err != nil {
			return err
		}
		target := filepath.Join("build/frontend/dist", relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		source, err := os.Open(path)
		if err != nil {
			return err
		}
		defer source.Close()
		dest, err := os.Create(target)
		if err != nil {
			return err
		}
		_, err = io.Copy(dest, source)
		closeErr := dest.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}
