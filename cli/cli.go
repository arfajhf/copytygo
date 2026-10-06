package cli

import (
	"fmt"
	"os"

	"github.com/arfajhf/copytygo/version"
)

type CommandHandler func(args []string) error

type CLI struct {
	commands map[string]CommandHandler
}

func New() *CLI {
	return &CLI{
		commands: make(
			map[string]CommandHandler,
		),
	}
}

func (cli *CLI) Command(
	name string,
	handler CommandHandler,
) {
	cli.commands[name] = handler
}

func (cli *CLI) Run() error {
	args := os.Args[1:]

	if len(args) == 0 {
		cli.printHelp()
		return nil
	}

	command := args[0]

	switch command {
	case "version", "--version", "-v":
		fmt.Printf(
			"CopyTyGo v%s\n",
			version.Framework,
		)

		return nil

	case "help", "--help", "-h":
		cli.printHelp()
		return nil
	}

	handler, exists :=
		cli.commands[command]

	if !exists {
		return fmt.Errorf(
			"unknown command %q",
			command,
		)
	}

	return handler(args[1:])
}

func (cli *CLI) printHelp() {
	fmt.Printf("\nCopyTyGo v%s\n\n", version.Framework)
	fmt.Println("Usage: ctg <command>")
	fmt.Println("\nProject:")
	fmt.Println("  new <name>                 Create a CopyTyGo project")
	fmt.Println("  dev [--lite]               Run development server with automatic Lite fallback")
	fmt.Println("  doctor                     Check project and Lite Runtime health")
	fmt.Println("  build                      Build application")
	fmt.Println("  route:list                 List project routes")
	fmt.Println("  version                    Show version")
	fmt.Println("\nGenerators:")
	fmt.Println("  make:controller <name> [--resource|--memory-resource]")
	fmt.Println("  make:resource <name>        Create model + DB controller + migration")
	fmt.Println("  make:model <name>")
	fmt.Println("  make:migration <name>")
	fmt.Println("  make:middleware <name>")
	fmt.Println("  make:service <name>")
	fmt.Println("\nDatabase:")
	fmt.Println("  migrate")
	fmt.Println("  migrate:status")
	fmt.Println("  migrate:rollback")
	fmt.Println("  migrate:reset")
	fmt.Println("  migrate:fresh")
	fmt.Println("  db:check                    Check database connection")
	fmt.Println("\nSecurity/Auth:")
	fmt.Println("  key:generate")
	fmt.Println("  install:auth [single|multi]")
	fmt.Println()
}
