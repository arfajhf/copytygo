package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/arfajhf/copytygo/config"
	"github.com/arfajhf/copytygo/database"
	"github.com/arfajhf/copytygo/database/drivers"
)

func DatabaseCheck(_ []string) error {
	if err := databaseHealthCheck(); err != nil {
		return err
	}

	fmt.Println("CopyTyGo Database")
	fmt.Println("-----------------")
	fmt.Println("Driver   :", config.Get("DB_DRIVER", "mysql"))
	fmt.Println("Host     :", config.Get("DB_HOST", "127.0.0.1"))
	fmt.Println("Database :", config.Get("DB_DATABASE", ""))
	fmt.Println("Status   : connected")
	return nil
}

func databaseHealthCheck() error {
	if err := config.LoadEnv(".env"); err != nil {
		return fmt.Errorf("copytygo: unable to load .env: %w", err)
	}

	drivers.Register()

	db, err := database.Connect()
	if err != nil {
		return err
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("copytygo: database ping failed: %w", err)
	}

	return nil
}
