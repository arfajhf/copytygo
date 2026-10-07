package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Task func(context.Context) error

type Entry struct {
	Name     string
	Interval time.Duration
	Task     Task
}

type Failure struct {
	Name string
	Err  error
}

type Scheduler struct {
	mu        sync.RWMutex
	entries   []Entry
	onFailure func(Failure)
}

func New() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) Every(name string, interval time.Duration, task Task) error {
	if name == "" {
		return errors.New("copytygo scheduler: task name is required")
	}
	if interval <= 0 {
		return errors.New("copytygo scheduler: interval must be positive")
	}
	if task == nil {
		return errors.New("copytygo scheduler: task function is required")
	}
	s.mu.Lock()
	s.entries = append(s.entries, Entry{Name: name, Interval: interval, Task: task})
	s.mu.Unlock()
	return nil
}

func (s *Scheduler) OnFailure(fn func(Failure)) {
	s.mu.Lock()
	s.onFailure = fn
	s.mu.Unlock()
}

func (s *Scheduler) Entries() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Entry(nil), s.entries...)
}

func (s *Scheduler) Start(ctx context.Context) {
	for _, entry := range s.Entries() {
		entry := entry
		go func() {
			ticker := time.NewTicker(entry.Interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := entry.Task(ctx); err != nil {
						s.mu.RLock()
						fn := s.onFailure
						s.mu.RUnlock()
						if fn != nil {
							fn(Failure{Name: entry.Name, Err: err})
						}
					}
				}
			}
		}()
	}
}
