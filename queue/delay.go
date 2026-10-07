package queue

import (
	"context"
	"time"
)

func (w *Worker) DispatchAfter(ctx context.Context, delay time.Duration, item Item) error {
	if delay <= 0 {
		return w.Dispatch(item)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	timer := time.NewTimer(delay)
	go func() {
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			_ = w.Dispatch(item)
		}
	}()
	return nil
}
