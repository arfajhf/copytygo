package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRuns(t *testing.T) {
	s := New()
	var calls atomic.Int32
	if err := s.Every("heartbeat", 10*time.Millisecond, func(context.Context) error {
		calls.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	time.Sleep(35 * time.Millisecond)
	cancel()
	if calls.Load() < 2 {
		t.Fatalf("expected task to run, got %d", calls.Load())
	}
}
