package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/arfajhf/copytygo/v4/core"
)

type Task func(context.Context) error

type Entry struct {
	Name      string        `json:"name"`
	Interval  time.Duration `json:"interval"`
	Cron      string        `json:"cron,omitempty"`
	Task      Task          `json:"-"`
	LastRun   time.Time     `json:"last_run,omitempty"`
	NextRun   time.Time     `json:"next_run,omitempty"`
	LastError string        `json:"last_error,omitempty"`
	Runs      int64         `json:"runs"`
}

type Failure struct {
	Name string
	Err  error
}

type Scheduler struct {
	mu        sync.RWMutex
	entries   []*Entry
	onFailure func(Failure)
	started   bool
}

func New() *Scheduler {
	return &Scheduler{entries: make([]*Entry, 0)}
}

var Default = New()

func Attach(app *core.Application, scheduler *Scheduler) {
	if app == nil {
		return
	}
	if scheduler == nil {
		scheduler = Default
	}
	app.Background("scheduler", func(ctx context.Context) error {
		scheduler.Start(ctx)
		<-ctx.Done()
		return nil
	})
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
	defer s.mu.Unlock()

	for _, existing := range s.entries {
		if existing.Name == name {
			return errors.New("copytygo scheduler: task name already registered")
		}
	}

	s.entries = append(s.entries, &Entry{
		Name:     name,
		Interval: interval,
	})
	s.entries[len(s.entries)-1].Task = task
	return nil
}

func (s *Scheduler) EveryMinute(name string, task Task) error {
	return s.Every(name, time.Minute, task)
}

func (s *Scheduler) Hourly(name string, task Task) error {
	return s.Every(name, time.Hour, task)
}

func (s *Scheduler) Daily(name string, task Task) error {
	return s.Every(name, 24*time.Hour, task)
}

func (s *Scheduler) Cron(name, expression string, task Task) error {
	if name == "" {
		return errors.New("copytygo scheduler: task name is required")
	}
	if task == nil {
		return errors.New("copytygo scheduler: task function is required")
	}
	parsed, err := parseCron(expression)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.entries {
		if existing.Name == name {
			return errors.New("copytygo scheduler: task name already registered")
		}
	}

	entry := &Entry{
		Name: name,
		Cron: parsed.expression,
		Task: task,
	}
	entry.NextRun = parsed.nextAfter(time.Now())
	s.entries = append(s.entries, entry)
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

	out := make([]Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		if entry != nil {
			out = append(out, *entry)
		}
	}
	return out
}

func (s *Scheduler) Started() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	entries := append([]*Entry(nil), s.entries...)
	now := time.Now()
	for _, entry := range entries {
		if entry.Cron != "" {
			parsed, err := parseCron(entry.Cron)
			if err == nil {
				entry.NextRun = parsed.nextAfter(now)
			}
			continue
		}
		entry.NextRun = now.Add(entry.Interval)
	}
	s.mu.Unlock()

	for _, entry := range entries {
		entry := entry
		if entry.Cron != "" {
			go s.runCronEntry(ctx, entry)
			continue
		}
		go s.runEntry(ctx, entry)
	}
}

func (s *Scheduler) runEntry(ctx context.Context, entry *Entry) {
	ticker := time.NewTicker(entry.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case at := <-ticker.C:
			err := entry.Task(ctx)

			s.mu.Lock()
			entry.LastRun = at
			entry.NextRun = at.Add(entry.Interval)
			entry.Runs++
			entry.LastError = ""
			if err != nil {
				entry.LastError = err.Error()
			}
			fn := s.onFailure
			s.mu.Unlock()

			if err != nil && fn != nil {
				fn(Failure{Name: entry.Name, Err: err})
			}
		}
	}
}


func (s *Scheduler) runCronEntry(ctx context.Context, entry *Entry) {
	parsed, err := parseCron(entry.Cron)
	if err != nil {
		s.mu.Lock()
		entry.LastError = err.Error()
		fn := s.onFailure
		s.mu.Unlock()
		if fn != nil {
			fn(Failure{Name: entry.Name, Err: err})
		}
		return
	}

	for {
		next := parsed.nextAfter(time.Now())
		if next.IsZero() {
			err := errors.New("copytygo scheduler: unable to determine next cron run")
			s.mu.Lock()
			entry.LastError = err.Error()
			fn := s.onFailure
			s.mu.Unlock()
			if fn != nil {
				fn(Failure{Name: entry.Name, Err: err})
			}
			return
		}

		s.mu.Lock()
		entry.NextRun = next
		s.mu.Unlock()

		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case at := <-timer.C:
			runErr := entry.Task(ctx)
			s.mu.Lock()
			entry.LastRun = at
			entry.Runs++
			entry.LastError = ""
			entry.NextRun = parsed.nextAfter(at)
			if runErr != nil {
				entry.LastError = runErr.Error()
			}
			fn := s.onFailure
			s.mu.Unlock()

			if runErr != nil && fn != nil {
				fn(Failure{Name: entry.Name, Err: runErr})
			}
		}
	}
}
