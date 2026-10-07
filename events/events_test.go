package events

import (
	"context"
	"testing"
)

func TestEmit(t *testing.T) {
	bus := New()
	got := ""
	bus.On("user.created", func(_ context.Context, payload any) error {
		got = payload.(string)
		return nil
	})
	if err := bus.Emit(context.Background(), "user.created", "42"); err != nil {
		t.Fatal(err)
	}
	if got != "42" {
		t.Fatalf("unexpected payload %q", got)
	}
}
