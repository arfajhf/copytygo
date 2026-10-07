package queue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatchAfter(t *testing.T) {
	worker := New(4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx, 1)
	defer worker.Stop()

	var ran atomic.Bool
	if err := worker.DispatchAfter(ctx, 25*time.Millisecond, Item{
		Name: "delayed",
		Run: func(context.Context) error {
			ran.Store(true)
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	if ran.Load() {
		t.Fatal("delayed job ran too early")
	}
	time.Sleep(60 * time.Millisecond)
	if !ran.Load() {
		t.Fatal("delayed job did not run")
	}
}
