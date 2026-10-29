package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestWSLNativeDevArgsPreserveWorkingDirectory(t *testing.T) {
	directory := `C:\Users\Example User\copytygo project`
	args := wslNativeDevArgs(directory)
	if len(args) != 6 {
		t.Fatalf("unexpected WSL argument count: %d", len(args))
	}
	if !reflect.DeepEqual(args[:5], []string{"--cd", directory, "--exec", "/bin/sh", "-c"}) {
		t.Fatalf("unexpected WSL launch arguments: %q", args[:5])
	}
	if !strings.Contains(args[5], "exec ctg dev") {
		t.Fatal("WSL does not launch full native runtime")
	}
	if strings.Contains(args[5], "go.exe") || strings.Contains(args[5], "--lite") {
		t.Fatal("WSL launch unexpectedly invokes Windows Go or Lite")
	}
	other := wslNativeDevArgs(`C:\A "quoted" folder`)
	if args[5] != other[5] {
		t.Fatal("user-derived project path appeared in the Linux shell")
	}
}
