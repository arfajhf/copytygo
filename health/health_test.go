package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestHealthRegistry(t *testing.T) {
	registry := New()
	if err := registry.Add("ok", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := registry.Add("bad", func(context.Context) error { return errors.New("down") }); err != nil {
		t.Fatal(err)
	}

	results := registry.Run(context.Background(), time.Second)
	if len(results) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(results))
	}
	if Overall(results) != Unhealthy {
		t.Fatal("expected unhealthy overall status")
	}
}
