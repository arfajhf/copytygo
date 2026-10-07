package foundation

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/arfajhf/copytygo/v4/cache"
	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/container"
	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
	"github.com/arfajhf/copytygo/v4/events"
	"github.com/arfajhf/copytygo/v4/health"
	"github.com/arfajhf/copytygo/v4/mail"
	"github.com/arfajhf/copytygo/v4/queue"
	"github.com/arfajhf/copytygo/v4/scheduler"
	"github.com/arfajhf/copytygo/v4/storage"
)

const (
	CacheService     = "cache"
	EventBusService  = "events"
	MailService      = "mail"
	StorageService   = "storage"
	QueueService     = "queue"
	SchedulerService = "scheduler"
	HealthService    = "health"
)

func RegisterDefaults(services *container.Container) error {
	if services == nil {
		return fmt.Errorf("copytygo foundation: service container is required")
	}

	services.Singleton(CacheService, func(*container.Container) (any, error) {
		return cache.NewMemory(), nil
	})

	services.Singleton(EventBusService, func(*container.Container) (any, error) {
		return events.New(), nil
	})

	services.Singleton(StorageService, func(*container.Container) (any, error) {
		root := config.Get("STORAGE_PATH", "storage/app")
		return storage.NewLocal(root)
	})

	services.Singleton(MailService, func(*container.Container) (any, error) {
		return mail.NewSMTP(mail.SMTPConfig{
			Host:     config.Get("MAIL_HOST", "127.0.0.1"),
			Port:     config.GetInt("MAIL_PORT", 1025),
			Username: config.Get("MAIL_USERNAME"),
			Password: config.Get("MAIL_PASSWORD"),
			From:     config.Get("MAIL_FROM", "noreply@copytygo.local"),
		}), nil
	})

	services.Instance(QueueService, queue.Default)
	services.Instance(SchedulerService, scheduler.Default)

	services.Singleton(HealthService, func(services *container.Container) (any, error) {
		registry := health.New()

		if err := registry.Add("database", func(ctx context.Context) error {
			drivers.Register()
			db, err := database.Connect()
			if err != nil {
				return err
			}
			checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			return db.PingContext(checkCtx)
		}); err != nil {
			return nil, err
		}

		if err := registry.Add("cache", func(context.Context) error {
			service, err := services.Resolve(CacheService)
			if err != nil {
				return err
			}
			store, ok := service.(cache.Store)
			if !ok {
				return fmt.Errorf("cache service does not implement cache.Store")
			}
			const key = "__copytygo_health"
			store.Set(key, "ok", time.Second)
			value, ok := store.Get(key)
			store.Delete(key)
			if !ok || value != "ok" {
				return fmt.Errorf("cache read/write check failed")
			}
			return nil
		}); err != nil {
			return nil, err
		}

		if err := registry.Add("storage", func(context.Context) error {
			service, err := services.Resolve(StorageService)
			if err != nil {
				return err
			}
			local, ok := service.(*storage.Local)
			if !ok {
				return nil
			}
			info, err := os.Stat(local.Root())
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return fmt.Errorf("storage root is not a directory")
			}
			return nil
		}); err != nil {
			return nil, err
		}

		return registry, nil
	})

	return nil
}


func RunHealth(ctx context.Context, services *container.Container) (health.Status, []health.Result, error) {
	if services == nil {
		return health.Unhealthy, nil, fmt.Errorf("copytygo foundation: service container is required")
	}

	service, err := services.Resolve(HealthService)
	if err != nil {
		return health.Unhealthy, nil, err
	}

	registry, ok := service.(*health.Registry)
	if !ok {
		return health.Unhealthy, nil, fmt.Errorf("copytygo foundation: health service has unexpected type %T", service)
	}

	results := registry.Run(ctx, 5*time.Second)
	return health.Overall(results), results, nil
}
