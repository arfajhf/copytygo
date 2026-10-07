package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Status string

const (
	Healthy   Status = "healthy"
	Unhealthy Status = "unhealthy"
)

type Check func(context.Context) error

type Result struct {
	Name       string `json:"name"`
	Status     Status `json:"status"`
	Message    string `json:"message,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

type registration struct {
	name  string
	check Check
}

type Registry struct {
	mu     sync.RWMutex
	checks []registration
}

func New() *Registry {
	return &Registry{checks: make([]registration, 0)}
}

func (r *Registry) Add(name string, check Check) error {
	if name == "" {
		return fmt.Errorf("copytygo health: check name is required")
	}
	if check == nil {
		return fmt.Errorf("copytygo health: check %q has no handler", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.checks {
		if existing.name == name {
			return fmt.Errorf("copytygo health: check %q already exists", name)
		}
	}
	r.checks = append(r.checks, registration{name: name, check: check})
	return nil
}

func (r *Registry) Run(ctx context.Context, timeout time.Duration) []Result {
	r.mu.RLock()
	checks := append([]registration(nil), r.checks...)
	r.mu.RUnlock()

	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	results := make([]Result, 0, len(checks))
	for _, item := range checks {
		start := time.Now()
		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		err := item.check(checkCtx)
		cancel()

		result := Result{
			Name:       item.name,
			Status:     Healthy,
			DurationMS: time.Since(start).Milliseconds(),
		}
		if err != nil {
			result.Status = Unhealthy
			result.Message = err.Error()
		}
		results = append(results, result)
	}
	return results
}

func Overall(results []Result) Status {
	for _, result := range results {
		if result.Status != Healthy {
			return Unhealthy
		}
	}
	return Healthy
}
