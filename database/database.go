package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

var (
	connection *sql.DB

	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

func RegisterDriver(driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()

	drivers[driver.Name()] = driver
}

func Connect() (*sql.DB, error) {
	if connection != nil {
		return connection, nil
	}

	cfg := LoadConfig()

	driversMu.RLock()
	driver, exists := drivers[cfg.Driver]
	driversMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf(
			"copytygo: unsupported database driver %q",
			cfg.Driver,
		)
	}

	db, err := sql.Open(
		driver.SQLDriver(),
		driver.DSN(cfg),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"copytygo: unable to open %s database: %w",
			cfg.Driver,
			err,
		)
	}

	db.SetMaxOpenConns(
		cfg.MaxOpenConnections,
	)

	db.SetMaxIdleConns(
		cfg.MaxIdleConnections,
	)

	db.SetConnMaxLifetime(
		30 * time.Minute,
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf(
			"copytygo: unable to connect to %s database: %w",
			cfg.Driver,
			err,
		)
	}

	connection = db

	return connection, nil
}

func Connection() (*sql.DB, error) {
	return Connect()
}

func Close() error {
	if connection == nil {
		return nil
	}

	err := connection.Close()

	connection = nil

	return err
}
