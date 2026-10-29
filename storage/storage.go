package storage

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Disk interface {
	Put(path string, data []byte) error
	Get(path string) ([]byte, error)
	Delete(path string) error
	Exists(path string) bool
}

type Local struct {
	root string
}

func NewLocal(root string) (*Local, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("copytygo storage: root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, err
	}
	return &Local{root: abs}, nil
}

// relativeName rejects lexical traversal. Individual operations below use
// os.Root, so a symlink created within the storage tree cannot redirect reads,
// writes or deletes outside its configured boundary.
func (l *Local) relativeName(name string) (string, error) {
	if l == nil {
		return "", errors.New("copytygo storage: disk is unavailable")
	}
	name = filepath.Clean(strings.TrimSpace(name))
	if name == "." || filepath.IsAbs(name) || name == ".." ||
		strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return "", errors.New("copytygo storage: invalid path")
	}
	return name, nil
}

func (l *Local) Put(name string, data []byte) error {
	name, err := l.relativeName(name)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(l.root)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	return root.WriteFile(name, data, 0644)
}

func (l *Local) Get(name string) ([]byte, error) {
	name, err := l.relativeName(name)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(l.root)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(name)
}

func (l *Local) Delete(name string) error {
	name, err := l.relativeName(name)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(l.root)
	if err != nil {
		return err
	}
	defer root.Close()
	err = root.Remove(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (l *Local) Exists(name string) bool {
	name, err := l.relativeName(name)
	if err != nil {
		return false
	}
	root, err := os.OpenRoot(l.root)
	if err != nil {
		return false
	}
	defer root.Close()
	info, err := root.Stat(name)
	return err == nil && !info.IsDir()
}

func (l *Local) Root() string {
	if l == nil {
		return ""
	}
	return l.root
}
