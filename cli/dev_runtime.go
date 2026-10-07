package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type devRuntimeState struct {
	OS      string `json:"os"`
	Runtime string `json:"runtime"`
}

// Keep machine-specific runtime decisions outside the application's source tree.
func devRuntimeCachePath() string {
	root, err := os.Getwd()
	if err != nil {
		return ""
	}
	if realRoot, err := filepath.EvalSymlinks(root); err == nil {
		root = realRoot
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	key := sha256.Sum256([]byte(root))
	return filepath.Join(cache, "copytygo", "runtime", hex.EncodeToString(key[:])+".json")
}

func rememberedLite(path, goos string) bool {
	if path == "" || goos != "windows" {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var state devRuntimeState
	return json.Unmarshal(raw, &state) == nil && state.OS == goos && state.Runtime == "lite"
}

func rememberLite(path, goos string) error {
	if path == "" {
		return fmt.Errorf("user cache directory is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	raw, err := json.Marshal(devRuntimeState{OS: goos, Runtime: "lite"})
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0600)
}

func devWithRuntimes(args []string, goos, cachePath string, native, lite func() error, output io.Writer) error {
	forceLite, retryNative := false, false
	for _, arg := range args {
		switch arg {
		case "--lite":
			forceLite = true
		case "--native":
			retryNative = true
		default:
			return fmt.Errorf("usage: ctg dev [--lite|--native]")
		}
	}
	if forceLite && retryNative {
		return fmt.Errorf("copytygo: --lite and --native cannot be combined")
	}
	if forceLite {
		return lite()
	}
	if !retryNative && rememberedLite(cachePath, goos) {
		fmt.Fprintln(output, "Using Lite Runtime automatically: Windows previously blocked Native Runtime in this project.")
		return lite()
	}
	if retryNative && cachePath != "" {
		if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("copytygo: unable to reset runtime decision: %w", err)
		}
	}
	fmt.Fprintln(output, "CopyTyGo Dev\n-------------\nRuntime : native")
	fmt.Fprintln(output)
	err := native()
	if err == nil || !executionPolicyBlock(err, goos) {
		return err
	}
	fmt.Fprintln(output, "Windows blocked Native Runtime. Starting Lite Runtime automatically.")
	if err := rememberLite(cachePath, goos); err != nil {
		fmt.Fprintf(output, "Unable to remember the runtime decision: %v\n", err)
	}
	return lite()
}
