package foundation

import (
	"fmt"

	"github.com/arfajhf/copytygo/v4/cache"
	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/container"
	"github.com/arfajhf/copytygo/v4/events"
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

	return nil
}
