package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLocalDiskRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOOS == "plan9" {
		t.Skip("os.Root platform caveats")
	}
	root := t.TempDir()
	outside := t.TempDir()
	disk, err := NewLocal(root)
	if err != nil { t.Fatal(err) }

	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := disk.Put("escape/created.txt", []byte("exfiltrate")); err == nil {
		t.Fatal("storage wrote through symlink outside root")
	}
	if _, err := os.Stat(filepath.Join(outside, "created.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside directory modified by Put: %v", err)
	}
	if _, err := disk.Get("escape/secret.txt"); err == nil {
		t.Fatal("storage read through symlink outside root")
	}
	if disk.Exists("escape/secret.txt") {
		t.Fatal("storage exposed outside path via Exists")
	}
	if err := disk.Delete("escape/secret.txt"); err == nil {
		t.Fatal("storage deleted through symlink outside root")
	}
	if data, err := os.ReadFile(filepath.Join(outside, "secret.txt")); err != nil || string(data) != "private" {
		t.Fatalf("outside file changed: %q, %v", data, err)
	}
}

func TestLocalDiskAllowsInternalSymlinks(t *testing.T) {
	root := t.TempDir()
	disk, err := NewLocal(root)
	if err != nil { t.Fatal(err) }
	if err := disk.Put("legit/file.txt", []byte("inside")); err != nil { t.Fatal(err) }
	if err := os.Symlink("legit", filepath.Join(root, "alias")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	data, err := disk.Get("alias/file.txt")
	if err != nil || string(data) != "inside" {
		t.Fatalf("internal symlink should be permitted: %q, %v", data, err)
	}
}
