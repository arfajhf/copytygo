package logging

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
	At      time.Time      `json:"at"`
}

type Logger struct {
	mu       sync.RWMutex
	capacity int
	entries  []Entry
}

func New(capacity int) *Logger {
	if capacity < 1 {
		capacity = 500
	}
	return &Logger{capacity: capacity}
}

var Default = New(500)

func (l *Logger) Log(level, message string, fields map[string]any) {
	if l == nil {
		return
	}

	entry := Entry{
		Level:   strings.ToLower(strings.TrimSpace(level)),
		Message: message,
		Fields:  cloneFields(fields),
		At:      time.Now().UTC(),
	}
	if entry.Level == "" {
		entry.Level = "info"
	}

	l.mu.Lock()
	l.entries = append(l.entries, entry)
	if len(l.entries) > l.capacity {
		l.entries = append([]Entry(nil), l.entries[len(l.entries)-l.capacity:]...)
	}
	l.mu.Unlock()

	payload := map[string]any{
		"level":   entry.Level,
		"message": entry.Message,
		"at":      entry.At,
	}
	for key, value := range entry.Fields {
		payload[key] = value
	}
	if raw, err := json.Marshal(payload); err == nil {
		log.Printf("%s", raw)
	}
}

func (l *Logger) Debug(message string, fields map[string]any) { l.Log("debug", message, fields) }
func (l *Logger) Info(message string, fields map[string]any)  { l.Log("info", message, fields) }
func (l *Logger) Warn(message string, fields map[string]any)  { l.Log("warn", message, fields) }
func (l *Logger) Error(message string, fields map[string]any) { l.Log("error", message, fields) }

func (l *Logger) Entries() []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	out := make([]Entry, len(l.entries))
	for index := range l.entries {
		entry := l.entries[len(l.entries)-1-index]
		entry.Fields = cloneFields(entry.Fields)
		out[index] = entry
	}
	return out
}

func (l *Logger) Clear() {
	l.mu.Lock()
	l.entries = nil
	l.mu.Unlock()
}

func cloneFields(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
