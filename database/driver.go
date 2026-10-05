package database

type Driver interface {
	Name() string

	SQLDriver() string

	DSN(config Config) string
}
