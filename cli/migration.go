package cli

import (
	"fmt"
	"strings"

	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
	"github.com/arfajhf/copytygo/v4/database/migration"
)

type MigrationRegistrar func() error

func RegisterMigrationCommands(
	app *CLI,
	envPath string,
	register MigrationRegistrar,
) {

	app.Command(
		"migrate",
		func(args []string) error {

			migrator, cleanup, err :=
				prepareMigrator(
					envPath,
					register,
				)

			if err != nil {
				return err
			}

			defer cleanup()

			fmt.Println()
			fmt.Println("CopyTyGo Migration")
			fmt.Println("----------------------------")

			return migrator.Migrate()
		},
	)

	app.Command(
		"migrate:status",
		func(args []string) error {

			migrator, cleanup, err :=
				prepareMigrator(
					envPath,
					register,
				)

			if err != nil {
				return err
			}

			defer cleanup()

			return migrator.Status()
		},
	)

	app.Command(
		"migrate:rollback",
		func(args []string) error {
			if err := requireProductionForce(args, "migrate:rollback"); err != nil {
				return err
			}

			migrator, cleanup, err :=
				prepareMigrator(
					envPath,
					register,
				)

			if err != nil {
				return err
			}

			defer cleanup()

			fmt.Println()
			fmt.Println("CopyTyGo Rollback")
			fmt.Println("----------------------------")

			return migrator.Rollback()
		},
	)

	app.Command("migrate:reset", func(args []string) error {
		if err := requireProductionForce(args, "migrate:reset"); err != nil {
			return err
		}
		migrator, cleanup, err := prepareMigrator(envPath, register)
		if err != nil {
			return err
		}
		defer cleanup()
		return migrator.Reset()
	})
	app.Command("migrate:fresh", func(args []string) error {
		if err := requireProductionForce(args, "migrate:fresh"); err != nil {
			return err
		}
		migrator, cleanup, err := prepareMigrator(envPath, register)
		if err != nil {
			return err
		}
		defer cleanup()
		return migrator.Fresh()
	})

}

func prepareMigrator(
	envPath string,
	register MigrationRegistrar,
) (*migration.Migrator, func(), error) {

	if err := config.LoadEnv(
		envPath,
	); err != nil {

		return nil, func() {}, fmt.Errorf(
			"copytygo: unable to load environment: %w",
			err,
		)
	}

	drivers.Register()

	db, err := database.Connect()

	if err != nil {
		return nil, func() {}, err
	}

	if register != nil {
		if err := register(); err != nil {

			_ = database.Close()

			return nil, func() {}, err
		}
	}

	migrator := migration.NewMigrator(
		db,
		config.Get(
			"DB_DRIVER",
			"mysql",
		),
	)

	cleanup := func() {
		_ = database.Close()
	}

	return migrator, cleanup, nil
}


func requireProductionForce(args []string, action string) error {
	if err := config.LoadEnv(".env"); err != nil {
		return err
	}

	if strings.ToLower(strings.TrimSpace(config.Get("APP_ENV", "local"))) != "production" {
		return nil
	}

	for _, arg := range args {
		if arg == "--force" {
			return nil
		}
	}

	return fmt.Errorf("copytygo: %s is destructive in production; rerun with --force", action)
}
