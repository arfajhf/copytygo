package main

import (
	"fmt"
	"os"

	"github.com/arfajhf/copytygo/v3/cli"
)

func main() {
	app := cli.New()
	cli.RegisterDeveloperCommands(app)
	cli.RegisterMigrationCommands(
		app,
		".env",
		func() error {
			return cli.RegisterProjectMigrations("database/migrations")
		},
	)

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
