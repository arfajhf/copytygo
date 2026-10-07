package database

import (
	"github.com/arfajhf/copytygo/v4/config"
)

type Config struct {
	Driver   string
	Host     string
	Port     string
	Database string
	Username string
	Password string

	MaxOpenConnections int
	MaxIdleConnections int
}

func LoadConfig() Config {
	return Config{
		Driver: config.Get(
			"DB_DRIVER",
			"mysql",
		),

		Host: config.Get(
			"DB_HOST",
			"127.0.0.1",
		),

		Port: config.Get(
			"DB_PORT",
			defaultPort(),
		),

		Database: config.Get(
			"DB_DATABASE",
			"copytygo",
		),

		Username: config.Get(
			"DB_USERNAME",
			"root",
		),

		Password: config.Get(
			"DB_PASSWORD",
			"",
		),

		MaxOpenConnections: config.GetInt(
			"DB_MAX_OPEN_CONNECTIONS",
			25,
		),

		MaxIdleConnections: config.GetInt(
			"DB_MAX_IDLE_CONNECTIONS",
			10,
		),
	}
}

func defaultPort() string {
	driver := config.Get(
		"DB_DRIVER",
		"mysql",
	)

	switch driver {
	case "postgres":
		return "5432"

	default:
		return "3306"
	}
}
