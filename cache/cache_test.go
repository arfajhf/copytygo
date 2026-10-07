package cache

import (
	"testing"
	"time"
)

func TestMemoryTTL(t *testing.T) {
	store := NewMemory()
	store.Set("name", "copytygo", 10*time.Millisecond)
	if value, ok := store.Get("name"); !ok || value != "copytygo" {
		t.Fatal("expected cached value")
	}
	time.Sleep(20 * time.Millisecond)
	if _, ok := store.Get("name"); ok {
		t.Fatal("expected expired value")
	}
}

func TestRemember(t *testing.T) {
	store := NewMemory()
	calls := 0
	load := func() (string, error) {
		calls++
		return "value", nil
	}
	_, _ = Remember(store, "key", time.Minute, load)
	_, _ = Remember(store, "key", time.Minute, load)
	if calls != 1 {
		t.Fatalf("expected loader once, got %d", calls)
	}
}
