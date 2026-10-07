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

func (l *Local) resolve(name string) (string, error) {
	name = filepath.Clean(strings.TrimSpace(name))
	if name == "." || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return "", errors.New("copytygo storage: invalid path")
	}
	full := filepath.Join(l.root, name)
	rel, err := filepath.Rel(l.root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("copytygo storage: path escapes disk root")
	}
	return full, nil
}

func (l *Local) Put(name string, data []byte) error {
	full, err := l.resolve(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0644)
}

func (l *Local) Get(name string) ([]byte, error) {
	full, err := l.resolve(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(full)
}

func (l *Local) Delete(name string) error {
	full, err := l.resolve(name)
	if err != nil {
		return err
	}
	err = os.Remove(full)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (l *Local) Exists(name string) bool {
	full, err := l.resolve(name)
	if err != nil {
		return false
	}
	info, err := os.Stat(full)
	return err == nil && !info.IsDir()
}


func (l *Local) Root() string {
	if l == nil {
		return ""
	}
	return l.root
}
