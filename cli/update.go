package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const frameworkModule = "github.com/arfajhf/copytygo/v2"

func Update(args []string) error {
	target := "latest"
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		target = strings.TrimSpace(args[0])
	}

	fmt.Println("CopyTyGo Update")
	fmt.Println("---------------")
	fmt.Println("Target :", target)
	fmt.Println()

	if err := runUpdateCommand("go", "install", frameworkModule+"/cmd/ctg@"+target); err != nil {
		return fmt.Errorf("copytygo: unable to update CLI: %w", err)
	}

	if _, err := os.Stat("go.mod"); err == nil {
		if err := runUpdateCommand("go", "get", frameworkModule+"@"+target); err != nil {
			return fmt.Errorf("copytygo: CLI updated, but project dependency update failed: %w", err)
		}
		if err := runUpdateCommand("go", "mod", "tidy"); err != nil {
			return fmt.Errorf("copytygo: dependency updated, but go mod tidy failed: %w", err)
		}
		fmt.Println()
		fmt.Println("CLI and project dependency updated.")
		return nil
	}

	fmt.Println()
	fmt.Println("CLI updated. No go.mod found, so no project dependency was changed.")
	return nil
}

func runUpdateCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
