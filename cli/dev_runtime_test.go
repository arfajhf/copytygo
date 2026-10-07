package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevRetriesNativeAfterPolicyBlockAndAuthInstallation(t *testing.T) {
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

	blocked := errors.New(`fork/exec C:\temp\app.exe: An Application Control policy has blocked this file.`)
	nativeCalls := 0
	native := func() error {
		nativeCalls++
		if nativeCalls == 1 {
			return blocked
		}
		if _, err := os.Stat("routes/auth.go"); err != nil {
			return err
		}
		return nil
	}
	lite := func() error { t.Fatal("Native startup switched to Lite"); return nil }
	var output bytes.Buffer
	if err := devWithRuntimes(nil, native, lite, &output); !errors.Is(err, blocked) {
		t.Fatalf("lost Native policy error: %v", err)
	}
	if err := InstallAuth("multi", "."); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := devWithRuntimes(nil, native, lite, &output); err != nil {
		t.Fatal(err)
	}
	if nativeCalls != 2 || !strings.Contains(output.String(), "Runtime : native") {
		t.Fatal("plain ctg dev did not retry Native after the earlier refusal")
	}
}

func TestDevReturnsNativeErrorsWithoutChangingRuntime(t *testing.T) {
	for _, message := range []string{
		"undefined: RegisterAuth",
		"open routes/web.go: Access is denied.",
		"fork/exec app.exe: An Application Control policy has blocked this file.",
		"listen tcp: address already in use",
	} {
		t.Run(message, func(t *testing.T) {
			problem := errors.New(message)
			for attempt := 0; attempt < 2; attempt++ {
				err := devWithRuntimes(nil, func() error { return problem }, func() error {
					t.Fatal("Native error triggered Lite")
					return nil
				}, &bytes.Buffer{})
				if !errors.Is(err, problem) {
					t.Fatalf("lost Native error: %v", err)
				}
			}
		})
	}
}

func TestDevExplicitRuntimeFlags(t *testing.T) {
	problem := errors.New("startup error")
	for _, test := range []struct {
		name       string
		args       []string
		wantNative bool
	}{
		{"default", nil, true},
		{"native alias", []string{"--native"}, true},
		{"explicit lite", []string{"--lite"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			nativeCalls, liteCalls := 0, 0
			var output bytes.Buffer
			err := devWithRuntimes(test.args, func() error { nativeCalls++; return problem }, func() error { liteCalls++; return problem }, &output)
			if !errors.Is(err, problem) {
				t.Fatalf("lost startup error: %v", err)
			}
			if test.wantNative {
				if nativeCalls != 1 || liteCalls != 0 || !strings.Contains(output.String(), "Runtime : native") {
					t.Fatal("Native was not selected")
				}
			} else if nativeCalls != 0 || liteCalls != 1 || strings.Contains(output.String(), "Runtime : native") {
				t.Fatal("explicit Lite incorrectly started Native")
			}
		})
	}
}

func TestDevInvalidFlagsDoNotStartRuntime(t *testing.T) {
	for _, args := range [][]string{{"--unknown"}, {"--lite", "--native"}, {"--native", "--lite"}} {
		startup := func() error { t.Fatal("invalid flags started a runtime"); return nil }
		if err := devWithRuntimes(args, startup, startup, &bytes.Buffer{}); err == nil {
			t.Fatalf("expected usage error for %v", args)
		}
	}
}
