package main

import (
	"fmt"
	"github.com/arfajhf/copytygo/cli"
	"github.com/arfajhf/copytygo/example/migrations"
	"os"
)

func main() {
	app := cli.New()
	cli.RegisterDeveloperCommands(app)
	cli.RegisterMigrationCommands(app, "example/.env", migrations.Register)
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
