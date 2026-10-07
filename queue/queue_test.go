package queue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerRetries(t *testing.T) {
	worker := New(2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx, 1)
	defer worker.Stop()

	var attempts atomic.Int32
	done := make(chan struct{})
	err := worker.Dispatch(Item{
		Name: "retry",
		MaxAttempts: 3,
		Run: func(context.Context) error {
			if attempts.Add(1) < 3 {
				return errors.New("temporary")
			}
			close(done)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("job did not complete")
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}
