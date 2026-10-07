package cache

import (
	"sync"
	"time"
)

type Store interface {
	Get(key string) (any, bool)
	Set(key string, value any, ttl time.Duration)
	Delete(key string)
	Clear()
}

type entry struct {
	value     any
	expiresAt time.Time
}

type Memory struct {
	mu    sync.RWMutex
	items map[string]entry
}

func NewMemory() *Memory {
	return &Memory{items: make(map[string]entry)}
}

func (m *Memory) Get(key string) (any, bool) {
	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		m.Delete(key)
		return nil, false
	}
	return item.value, true
}

func (m *Memory) Set(key string, value any, ttl time.Duration) {
	item := entry{value: value}
	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}
	m.mu.Lock()
	m.items[key] = item
	m.mu.Unlock()
}

func (m *Memory) Delete(key string) {
	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()
}

func (m *Memory) Clear() {
	m.mu.Lock()
	m.items = make(map[string]entry)
	m.mu.Unlock()
}

func Remember[T any](store Store, key string, ttl time.Duration, load func() (T, error)) (T, error) {
	var zero T
	if value, ok := store.Get(key); ok {
		if typed, ok := value.(T); ok {
			return typed, nil
		}
	}
	value, err := load()
	if err != nil {
		return zero, err
	}
	store.Set(key, value, ttl)
	return value, nil
}
