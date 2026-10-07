package cli

import "github.com/arfajhf/copytygo/v4/database/migration"

func ProjectMigrationStatus() ([]migration.StatusRecord, error) {
	migrator, cleanup, err := prepareMigrator(
		".env",
		func() error { return RegisterProjectMigrations("database/migrations") },
	)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return migrator.StatusRecords()
}

func ProjectMigrate() error {
	migrator, cleanup, err := prepareMigrator(
		".env",
		func() error { return RegisterProjectMigrations("database/migrations") },
	)
	if err != nil {
		return err
	}
	defer cleanup()
	return migrator.Migrate()
}

func ProjectRollback() error {
	migrator, cleanup, err := prepareMigrator(
		".env",
		func() error { return RegisterProjectMigrations("database/migrations") },
	)
	if err != nil {
		return err
	}
	defer cleanup()
	return migrator.Rollback()
}
