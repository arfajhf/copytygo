package storage

import (
	"path/filepath"
	"testing"
)

func TestLocalDisk(t *testing.T) {
	disk, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := disk.Put("avatars/user.txt", []byte("copytygo")); err != nil {
		t.Fatal(err)
	}
	if !disk.Exists("avatars/user.txt") {
		t.Fatal("expected stored file")
	}
	data, err := disk.Get("avatars/user.txt")
	if err != nil || string(data) != "copytygo" {
		t.Fatalf("unexpected data %q err=%v", data, err)
	}
	if err := disk.Put(filepath.Join("..", "escape.txt"), []byte("bad")); err == nil {
		t.Fatal("expected traversal protection")
	}
}
