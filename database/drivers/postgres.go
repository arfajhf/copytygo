package drivers

import (
	"fmt"
	"net/url"

	"github.com/arfajhf/copytygo/v3/database"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgreSQL struct{}

func (PostgreSQL) Name() string {
	return "postgres"
}

func (PostgreSQL) SQLDriver() string {
	return "pgx"
}

func (PostgreSQL) DSN(config database.Config) string {
	connection := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			config.Username,
			config.Password,
		),
		Host: fmt.Sprintf(
			"%s:%s",
			config.Host,
			config.Port,
		),
		Path: config.Database,
	}

	query := connection.Query()

	query.Set(
		"sslmode",
		"disable",
	)

	connection.RawQuery = query.Encode()

	return connection.String()
}
