package queue

import (
	"context"
	"testing"
)

func TestPayloadRegistry(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register("SendReport", func(context.Context, Payload) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Handler("SendReport"); !ok {
		t.Fatal("expected registered handler")
	}
	if err := registry.Register("SendReport", func(context.Context, Payload) error { return nil }); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}

func TestDatabaseQueuePlaceholders(t *testing.T) {
	mysql := &DatabaseQueue{Driver:"mysql"}
	if got := mysql.placeholder(3); got != "?" {
		t.Fatalf("mysql placeholder = %q", got)
	}
	postgres := &DatabaseQueue{Driver:"postgres"}
	if got := postgres.placeholder(3); got != "$3" {
		t.Fatalf("postgres placeholder = %q", got)
	}
}
