package events

import (
	"context"
	"fmt"
	"sync"
)

type Listener func(context.Context, any) error

type Bus struct {
	mu        sync.RWMutex
	listeners map[string][]Listener
}

func New() *Bus {
	return &Bus{listeners: make(map[string][]Listener)}
}

func (b *Bus) On(event string, listener Listener) {
	b.mu.Lock()
	b.listeners[event] = append(b.listeners[event], listener)
	b.mu.Unlock()
}

func (b *Bus) Emit(ctx context.Context, event string, payload any) error {
	b.mu.RLock()
	listeners := append([]Listener(nil), b.listeners[event]...)
	b.mu.RUnlock()

	for i, listener := range listeners {
		if err := listener(ctx, payload); err != nil {
			return fmt.Errorf("copytygo events: %s listener %d: %w", event, i+1, err)
		}
	}
	return nil
}

func (b *Bus) Count(event string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.listeners[event])
}


type Registration struct {
	Event     string `json:"event"`
	Listeners int    `json:"listeners"`
}

func (b *Bus) Registrations() []Registration {
	b.mu.RLock()
	defer b.mu.RUnlock()

	items := make([]Registration, 0, len(b.listeners))
	for event, listeners := range b.listeners {
		items = append(items, Registration{
			Event:     event,
			Listeners: len(listeners),
		})
	}
	return items
}
