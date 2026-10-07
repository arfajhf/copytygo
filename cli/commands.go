package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func RegisterDeveloperCommands(app *CLI) {
	app.Command("new", func(args []string) error {
		if len(args) == 0 {
			return InteractiveNewProject("")
		}

		name := args[0]
		for _, arg := range args[1:] {
			if arg == "--interactive" || arg == "-i" {
				return InteractiveNewProject(name)
			}
		}

		return NewProject(name)
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
	app.Command("make:resource", func(args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("usage: ctg make:resource <name> [field:type ...]")
		}
		return MakeResourceWithFields(args[0], args[1:])
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
	app.Command("make:job", func(args []string) error {
		if len(args) < 1 { return fmt.Errorf("usage: ctg make:job <name>") }
		return MakeJob(args[0], filepath.Join("app", "jobs"))
	})
	app.Command("make:listener", func(args []string) error {
		if len(args) < 1 { return fmt.Errorf("usage: ctg make:listener <name>") }
		return MakeListener(args[0], filepath.Join("app", "listeners"))
	})
	app.Command("make:seeder", func(args []string) error {
		if len(args) < 1 { return fmt.Errorf("usage: ctg make:seeder <name>") }
		return MakeSeeder(args[0], filepath.Join("database", "seeders"))
	})
	app.Command("make:factory", func(args []string) error {
		if len(args) < 1 { return fmt.Errorf("usage: ctg make:factory <name>") }
		return MakeFactory(args[0], filepath.Join("database", "factories"))
	})
	app.Command("make:mail", func(args []string) error {
		if len(args) < 1 { return fmt.Errorf("usage: ctg make:mail <name>") }
		return MakeMail(args[0], filepath.Join("app", "mails"))
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
	app.Command("db:check", DatabaseCheck)
	app.Command("update", Update)
	app.Command("dev", Dev)
	app.Command("open", func(_ []string) error { return OpenApplication(false) })
	app.Command("studio", func(_ []string) error { return OpenApplication(true) })
	app.Command("doctor", Doctor)
	app.Command("route:list", func(_ []string) error { return RouteList("routes") })
	app.Command("build", func(_ []string) error {
		if err := os.MkdirAll("build", 0755); err != nil {
			return err
		}

		output := filepath.Join("build", "app")
		if runtime.GOOS == "windows" {
			output += ".exe"
		}

		cmd := exec.Command("go", "build", "-o", output, "./cmd/app")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		fmt.Println("Built:", output)
		return nil
	})
}
