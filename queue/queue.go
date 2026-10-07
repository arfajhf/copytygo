package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrClosed = errors.New("copytygo queue: worker is closed")

type Job func(context.Context) error

type Item struct {
	Name        string
	Run         Job
	MaxAttempts int
	Backoff     time.Duration
}

type Failure struct {
	Name     string
	Attempts int
	Err      error
}

type Worker struct {
	jobs      chan Item
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	closed    bool
	onFailure func(Failure)
}

func New(buffer int) *Worker {
	if buffer < 1 {
		buffer = 64
	}
	return &Worker{jobs: make(chan Item, buffer)}
}

func (w *Worker) OnFailure(fn func(Failure)) {
	w.mu.Lock()
	w.onFailure = fn
	w.mu.Unlock()
}

func (w *Worker) Start(parent context.Context, concurrency int) {
	if concurrency < 1 {
		concurrency = 1
	}
	ctx, cancel := context.WithCancel(parent)
	w.mu.Lock()
	w.cancel = cancel
	w.mu.Unlock()

	for i := 0; i < concurrency; i++ {
		w.wg.Add(1)
		go w.loop(ctx)
	}
}

func (w *Worker) Dispatch(item Item) error {
	if item.Run == nil {
		return errors.New("copytygo queue: job function is required")
	}
	if item.MaxAttempts < 1 {
		item.MaxAttempts = 1
	}
	w.mu.RLock()
	closed := w.closed
	w.mu.RUnlock()
	if closed {
		return ErrClosed
	}
	w.jobs <- item
	return nil
}

func (w *Worker) Stop() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	cancel := w.cancel
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	w.wg.Wait()
}

func (w *Worker) loop(ctx context.Context) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-w.jobs:
			w.run(ctx, item)
		}
	}
}

func (w *Worker) run(ctx context.Context, item Item) {
	var err error
	for attempt := 1; attempt <= item.MaxAttempts; attempt++ {
		err = item.Run(ctx)
		if err == nil {
			return
		}
		if attempt < item.MaxAttempts && item.Backoff > 0 {
			timer := time.NewTimer(item.Backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}
	w.mu.RLock()
	fn := w.onFailure
	w.mu.RUnlock()
	if fn != nil {
		fn(Failure{Name: item.Name, Attempts: item.MaxAttempts, Err: fmt.Errorf("%w", err)})
	}
}
