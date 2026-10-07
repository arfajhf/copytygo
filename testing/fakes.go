package testing

import (
	"sync"

	"github.com/arfajhf/copytygo/v4/events"
	"github.com/arfajhf/copytygo/v4/mail"
	"github.com/arfajhf/copytygo/v4/storage"
)

type Mail = mail.Fake

type EventBus struct {
	mu     sync.Mutex
	Events []Event
}

type Event struct {
	Name    string
	Payload any
}

func (f *EventBus) Emit(name string, payload any) {
	f.mu.Lock()
	f.Events = append(f.Events, Event{Name: name, Payload: payload})
	f.mu.Unlock()
}

func (f *EventBus) Reset() {
	f.mu.Lock()
	f.Events = nil
	f.mu.Unlock()
}

type Storage struct {
	mu    sync.RWMutex
	Files map[string][]byte
}

func NewStorage() *Storage {
	return &Storage{Files: make(map[string][]byte)}
}

func (s *Storage) Put(path string, data []byte) error {
	s.mu.Lock()
	s.Files[path] = append([]byte(nil), data...)
	s.mu.Unlock()
	return nil
}

func (s *Storage) Get(path string) ([]byte, error) {
	s.mu.RLock()
	data, ok := s.Files[path]
	s.mu.RUnlock()
	if !ok {
		return nil, &storageError{path: path}
	}
	return append([]byte(nil), data...), nil
}

func (s *Storage) Delete(path string) error {
	s.mu.Lock()
	delete(s.Files, path)
	s.mu.Unlock()
	return nil
}

func (s *Storage) Exists(path string) bool {
	s.mu.RLock()
	_, ok := s.Files[path]
	s.mu.RUnlock()
	return ok
}

type storageError struct{ path string }

func (e *storageError) Error() string { return "copytygo testing storage: file not found: " + e.path }

var _ storage.Disk = (*Storage)(nil)
var _ = events.New
