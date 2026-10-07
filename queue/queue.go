package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arfajhf/copytygo/v4/core"
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
	Name     string    `json:"name"`
	Attempts int       `json:"attempts"`
	Error    string    `json:"error"`
	At       time.Time `json:"at"`
	Err      error     `json:"-"`
}

type Stats struct {
	Pending     int   `json:"pending"`
	Running     int64 `json:"running"`
	Completed   int64 `json:"completed"`
	Failed      int64 `json:"failed"`
	Started     bool  `json:"started"`
	Concurrency int   `json:"concurrency"`
}

type Worker struct {
	jobs      chan Item
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	closed    bool
	started   bool
	concurrency int
	onFailure func(Failure)
	failures  []Failure
	running   atomic.Int64
	completed atomic.Int64
	failed    atomic.Int64
}

func New(buffer int) *Worker {
	if buffer < 1 {
		buffer = 64
	}
	return &Worker{jobs: make(chan Item, buffer), failures: make([]Failure, 0)}
}

var Default = New(256)

func Attach(app *core.Application, worker *Worker, concurrency int) {
	if app == nil {
		return
	}
	if worker == nil {
		worker = Default
	}
	app.Background("queue", func(ctx context.Context) error {
		worker.Start(ctx, concurrency)
		<-ctx.Done()
		worker.Stop()
		return nil
	})
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

	w.mu.Lock()
	if w.started || w.closed {
		w.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	w.started = true
	w.concurrency = concurrency
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

func (w *Worker) Stats() Stats {
	w.mu.RLock()
	started := w.started && !w.closed
	concurrency := w.concurrency
	w.mu.RUnlock()

	return Stats{
		Pending:     len(w.jobs),
		Running:     w.running.Load(),
		Completed:   w.completed.Load(),
		Failed:      w.failed.Load(),
		Started:     started,
		Concurrency: concurrency,
	}
}

func (w *Worker) Failures() []Failure {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]Failure, len(w.failures))
	for i := range w.failures {
		out[len(w.failures)-1-i] = w.failures[i]
	}
	return out
}

func (w *Worker) loop(ctx context.Context) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-w.jobs:
			w.running.Add(1)
			success := w.run(ctx, item)
			w.running.Add(-1)
			if success {
				w.completed.Add(1)
			}
		}
	}
}

func (w *Worker) run(ctx context.Context, item Item) bool {
	var err error
	for attempt := 1; attempt <= item.MaxAttempts; attempt++ {
		err = item.Run(ctx)
		if err == nil {
			return true
		}
		if attempt < item.MaxAttempts && item.Backoff > 0 {
			timer := time.NewTimer(item.Backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return false
			case <-timer.C:
			}
		}
	}

	failure := Failure{
		Name:     item.Name,
		Attempts: item.MaxAttempts,
		Error:    err.Error(),
		Err:      fmt.Errorf("%w", err),
		At:       time.Now().UTC(),
	}
	w.failed.Add(1)

	w.mu.Lock()
	w.failures = append(w.failures, failure)
	if len(w.failures) > 100 {
		w.failures = append([]Failure(nil), w.failures[len(w.failures)-100:]...)
	}
	fn := w.onFailure
	w.mu.Unlock()

	if fn != nil {
		fn(failure)
	}
	return false
}
