package cli

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeDevBinaryPath(t *testing.T) {
	for _, tc := range []struct{ goos, filename string }{
		{"windows", "app.exe"},
		{"linux", "app"},
		{"darwin", "app"},
	} {
		got := nativeDevBinaryPath(tc.goos)
		want := filepath.Join("build", "dev", tc.filename)
		if got != want {
			t.Fatalf("%s: got %q, want %q", tc.goos, got, want)
		}
	}
}

func TestNativeExecutionErrorProvidesPolicyDiagnostics(t *testing.T) {
	blocked := errors.New("fork/exec C:\\Windows\\Temp\\app.exe: An Application Control policy has blocked this file.")
	err := nativeExecutionError(blocked, "", "windows")
	if !errors.Is(err, blocked) {
		t.Fatalf("original execution error lost: %v", err)
	}
	if !strings.Contains(err.Error(), "event 3077") || !strings.Contains(err.Error(), "WINDOWS_DEVELOPMENT.md") {
		t.Fatalf("missing Windows diagnostic guidance: %v", err)
	}

	for _, message := range []string{"undefined: app", "listen tcp: address already in use", "open routes/web.go: Access is denied"} {
		err = nativeExecutionError(errors.New(message), "", "windows")
		if strings.Contains(err.Error(), "event 3077") {
			t.Fatalf("false positive for %q", message)
		}
	}
	err = nativeExecutionError(blocked, "", "linux")
	if strings.Contains(err.Error(), "event 3077") {
		t.Fatal("Windows-specific diagnosis on Linux")
	}
}

func TestRecognizesWindowsPolicyBlock(t *testing.T) {
	for _, msg := range []string{
		"An Application Control policy has blocked this file.",
		"This app has been blocked by your system administrator.",
		"Smart App Control has blocked an app.",
	} {
		if !isApplicationPolicyBlock(msg) {
			t.Fatalf("did not recognize %q", msg)
		}
	}
	if isApplicationPolicyBlock("The build failed: undefined: service") {
		t.Fatal("build failure mistaken for application policy")
	}
}

func TestNativeStderrWriterStreamsAndCaptures(t *testing.T) {
	var visible, captured bytes.Buffer
	writer := nativeStderrWriter(&visible, &captured)
	for _, message := range []string{"scheduler registered\n", "SmokeJob executed\n"} {
		if _, err := writer.Write([]byte(message)); err != nil {
			t.Fatal(err)
		}
	}
	expected := "scheduler registered\nSmokeJob executed\n"
	if visible.String() != expected {
		t.Fatalf("live stderr missing logs: %q", visible.String())
	}
	if captured.String() != expected {
		t.Fatalf("diagnostic capture missing logs: %q", captured.String())
	}
}
