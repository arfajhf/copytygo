package cli

import (
	"fmt"
	"strings"
)

// databaseConnectionError keeps driver messages and connection strings out of CLI output.
// Help is based on the failure category, not on values read from .env.
func databaseConnectionError(err error) error {
	if err == nil {
		return nil
	}

	failure := strings.ToLower(err.Error())
	hint := "Check DB_DRIVER, DB_HOST, DB_PORT, DB_DATABASE, DB_USERNAME, and DB_PASSWORD in .env, then run ctg db:check."

	switch {
	case strings.Contains(failure, "unsupported database driver"),
		strings.Contains(failure, "unknown driver"):
		hint = "Set DB_DRIVER to mysql or postgres in .env."
	case strings.Contains(failure, "connection refused"),
		strings.Contains(failure, "actively refused"):
		hint = "Start the database server and verify DB_HOST and DB_PORT in .env."
	case strings.Contains(failure, "password authentication failed"),
		strings.Contains(failure, "access denied for user"),
		strings.Contains(failure, "authentication failed"):
		hint = "Check DB_USERNAME, DB_PASSWORD, and database user permissions in .env."
	case strings.Contains(failure, "unknown database"),
		strings.Contains(failure, "does not exist"):
		hint = "Check DB_DATABASE in .env and create the database if needed."
	case strings.Contains(failure, "no such host"),
		strings.Contains(failure, "name resolution"):
		hint = "Check DB_HOST in .env and your database server hostname."
	case strings.Contains(failure, "deadline exceeded"),
		strings.Contains(failure, "i/o timeout"),
		strings.Contains(failure, "connection timed out"):
		hint = "Check that the database server is reachable and DB_HOST and DB_PORT are correct."
	}

	return fmt.Errorf("copytygo: database connection failed\nHint: %s", hint)
}
