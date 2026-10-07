package queue

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/core"
	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
)

type DispatchOptions struct {
	MaxAttempts int
	Delay       time.Duration
	Backoff     time.Duration
}

type Manager struct {
	Driver         string
	Memory         *Worker
	Database       *DatabaseQueue
	DatabaseWorker *DatabaseWorker
	Registry       *Registry
}

var configured = struct {
	sync.RWMutex
	manager *Manager
}{
	manager: &Manager{
		Driver:   "memory",
		Memory:   Default,
		Registry: DefaultRegistry,
	},
}

func Current() *Manager {
	configured.RLock()
	defer configured.RUnlock()
	copy := *configured.manager
	return &copy
}

func AttachConfigured(app *core.Application) error {
	if app == nil {
		return fmt.Errorf("copytygo queue: application is required")
	}

	driver := strings.ToLower(strings.TrimSpace(config.Get("QUEUE_DRIVER", "memory")))
	workers := config.GetInt("QUEUE_WORKERS", 1)
	if workers < 1 {
		workers = 1
	}

	manager := &Manager{
		Driver:   driver,
		Memory:   Default,
		Registry: DefaultRegistry,
	}

	switch driver {
	case "sync":
		// Jobs run immediately when dispatched.

	case "memory":
		Attach(app, Default, workers)

	case "database":
		drivers.Register()
		db, err := database.Connect()
		if err != nil {
			return err
		}
		queue, err := NewDatabaseQueue(db, config.Get("DB_DRIVER", "mysql"))
		if err != nil {
			return err
		}
		worker := NewDatabaseWorker(queue, DefaultRegistry)
		worker.PollInterval = time.Duration(config.GetInt("QUEUE_POLL_SECONDS", 1)) * time.Second
		worker.Backoff = time.Duration(config.GetInt("QUEUE_BACKOFF_SECONDS", 1)) * time.Second

		manager.Database = queue
		manager.DatabaseWorker = worker

		app.Background("queue:database", func(ctx context.Context) error {
			return worker.Run(ctx, workers)
		})

	default:
		return fmt.Errorf("copytygo queue: unsupported QUEUE_DRIVER %q", driver)
	}

	configured.Lock()
	configured.manager = manager
	configured.Unlock()
	return nil
}

func DispatchNamed(
	ctx context.Context,
	name string,
	payload Payload,
	options ...DispatchOptions,
) (string, error) {
	manager := Current()
	opts := DispatchOptions{MaxAttempts: 3, Backoff: time.Second}
	if len(options) > 0 {
		opts = options[0]
	}
	if opts.MaxAttempts < 1 {
		opts.MaxAttempts = 3
	}
	if opts.Backoff < 0 {
		opts.Backoff = 0
	}
	if payload == nil {
		payload = Payload{}
	}

	handler, exists := manager.Registry.Handler(name)
	if !exists {
		return "", fmt.Errorf("copytygo queue: job handler %q is not registered", name)
	}

	switch manager.Driver {
	case "sync":
		id, err := queueID()
		if err != nil {
			return "", err
		}
		if err := handler(ctx, payload); err != nil {
			return id, err
		}
		return id, nil

	case "memory":
		id, err := queueID()
		if err != nil {
			return "", err
		}
		item := Item{
			Name:        name,
			MaxAttempts: opts.MaxAttempts,
			Backoff:     opts.Backoff,
			Run: func(jobCtx context.Context) error {
				return handler(jobCtx, payload)
			},
		}
		if opts.Delay > 0 {
			if err := manager.Memory.DispatchAfter(ctx, opts.Delay, item); err != nil {
				return "", err
			}
		} else if err := manager.Memory.Dispatch(item); err != nil {
			return "", err
		}
		return id, nil

	case "database":
		if manager.Database == nil {
			return "", fmt.Errorf("copytygo queue: database driver is not initialized")
		}
		return manager.Database.Dispatch(ctx, name, payload, opts.MaxAttempts, opts.Delay)

	default:
		return "", fmt.Errorf("copytygo queue: unsupported driver %q", manager.Driver)
	}
}
