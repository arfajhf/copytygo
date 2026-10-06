package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func RegisterDeveloperCommands(app *CLI) {
	app.Command("new", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("usage: ctg new <project>")
		}
		return NewProject(args[0])
	})
	app.Command("make:migration", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("usage: ctg make:migration <name>")
		}
		return MakeMigration(args[0], filepath.Join("database", "migrations"))
	})
	app.Command("make:controller", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("usage: ctg make:controller <name> [--resource|--memory-resource]")
		}
		mode := ""
		for _, arg := range args[1:] {
			switch arg {
			case "--resource":
				mode = "resource"
			case "--memory-resource":
				mode = "memory"
			}
		}
		return MakeControllerWithMode(args[0], filepath.Join("app", "controllers"), mode)
	})
	app.Command("make:model", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("usage: ctg make:model <name>")
		}
		return MakeModel(args[0], filepath.Join("app", "models"))
	})
	app.Command("make:middleware", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("usage: ctg make:middleware <name>")
		}
		return MakeMiddleware(args[0], filepath.Join("app", "middleware"))
	})
	app.Command("make:service", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("usage: ctg make:service <name>")
		}
		return MakeService(args[0], filepath.Join("app", "services"))
	})
	app.Command("key:generate", func(_ []string) error {
		key, err := GenerateKey()
		if err != nil {
			return err
		}
		fmt.Println(key)
		return nil
	})
	app.Command("install:auth", func(args []string) error {
		mode := "single"
		if len(args) > 0 {
			mode = args[0]
		}
		return InstallAuth(mode, ".")
	})
	app.Command("dev", Dev)
	app.Command("doctor", Doctor)
	app.Command("route:list", func(_ []string) error { return RouteList("routes") })
	app.Command("build", func(_ []string) error {
		cmd := exec.Command("go", "build", "-o", "build/app", "./cmd/app")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
}
