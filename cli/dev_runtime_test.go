package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevRemembersPolicyFallbackAfterInstallingAuth(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := NewProject("app"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(root, "app")); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "user-cache", "runtime.json")
	blocked := errors.New(`fork/exec C:\temp\app.exe: An Application Control policy has blocked this file.`)
	nativeCalls, liteCalls := 0, 0
	native := func() error { nativeCalls++; return blocked }
	lite := func() error { liteCalls++; return nil }
	var output bytes.Buffer
	if err := devWithRuntimes(nil, "windows", cache, native, lite, &output); err != nil {
		t.Fatal(err)
	}
	if nativeCalls != 1 || liteCalls != 1 || !rememberedLite(cache, "windows") {
		t.Fatal("first blocked startup did not select and remember Lite")
	}
	if strings.Contains(output.String(), "fork/exec") {
		t.Fatal("policy error was presented as a startup failure")
	}
	if err := InstallAuth("multi", "."); err != nil {
		t.Fatal(err)
	}
	lite = func() error {
		liteCalls++
		routes, err := discoverLiteRoutes("routes")
		if err != nil {
			return err
		}
		found := map[string]bool{}
		for _, route := range routes {
			if route.AuthAction != "" {
				found[route.AuthAction] = true
			}
		}
		for _, action := range []string{"register", "login", "me"} {
			if !found[action] {
				return errors.New("auth action missing: " + action)
			}
		}
		return nil
	}
	output.Reset()
	if err := devWithRuntimes(nil, "windows", cache, native, lite, &output); err != nil {
		t.Fatal(err)
	}
	if nativeCalls != 1 || liteCalls != 2 {
		t.Fatal("ctg dev retried the blocked executable after auth changed")
	}
	if !strings.Contains(output.String(), "Using Lite Runtime automatically") {
		t.Fatal("runtime choice was not explained")
	}
	if _, err := os.Stat(".copytygo"); !os.IsNotExist(err) {
		t.Fatal("machine runtime state leaked into the project")
	}
	if err := devWithRuntimes([]string{"--native"}, "windows", cache, func() error { nativeCalls++; return nil }, lite, &output); err != nil {
		t.Fatal(err)
	}
	if nativeCalls != 2 || rememberedLite(cache, "windows") {
		t.Fatal("explicit Native retry did not reset the cached decision")
	}
}

func TestDevDoesNotMaskCompilationOrPermissionErrors(t *testing.T) {
	for _, message := range []string{"undefined: RegisterAuth", "open routes/web.go: Access is denied."} {
		t.Run(message, func(t *testing.T) {
			cache := filepath.Join(t.TempDir(), "runtime.json")
			problem := errors.New(message)
			liteCalled := false
			err := devWithRuntimes(nil, "windows", cache, func() error { return problem }, func() error { liteCalled = true; return nil }, &bytes.Buffer{})
			if !errors.Is(err, problem) || liteCalled || rememberedLite(cache, "windows") {
				t.Fatal("ordinary error was hidden by runtime fallback")
			}
		})
	}
}

func TestDevIgnoresInvalidOrForeignRuntimeCache(t *testing.T) {
	for _, state := range []string{"invalid json", `{"os":"linux","runtime":"lite"}`, `{"os":"windows","runtime":"unknown"}`} {
		cache := filepath.Join(t.TempDir(), "runtime.json")
		if err := os.WriteFile(cache, []byte(state), 0600); err != nil {
			t.Fatal(err)
		}
		called := false
		if err := devWithRuntimes(nil, "windows", cache, func() error { called = true; return nil }, func() error { t.Fatal("unexpected Lite startup"); return nil }, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("invalid cache prevented Native startup")
		}
	}
}

func TestDevLiteFailureIsReturnedEvenWhenRemembered(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "runtime.json")
	if err := rememberLite(cache, "windows"); err != nil {
		t.Fatal(err)
	}
	problem := errors.New("invalid route source")
	err := devWithRuntimes(nil, "windows", cache, func() error { t.Fatal("blocked Native was retried"); return nil }, func() error { return problem }, &bytes.Buffer{})
	if !errors.Is(err, problem) {
		t.Fatalf("lost Lite startup error: %v", err)
	}
	if rememberedLite(cache, "linux") {
		t.Fatal("Windows fallback affected another OS")
	}
}
