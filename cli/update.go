package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const frameworkModule = "github.com/arfajhf/copytygo/v5"

func Update(args []string) error {
	// A major-version change is not a routine dependency update. Abort before
	// installing a new CLI, so a v4 application cannot be left half-upgraded.
	if raw, err := os.ReadFile("go.mod"); err == nil {
		if hasLegacyV4Requirement(string(raw)) {
			return fmt.Errorf("copytygo: v4 to v5 requires a reviewed major-version migration; no changes made. See docs/UPGRADE_V5.md")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("copytygo: cannot inspect go.mod before update: %w", err)
	}
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

func hasLegacyV4Requirement(goMod string) bool {
	const legacy = "github.com/arfajhf/copytygo/v4"
	for _, line := range strings.Split(goMod, "\n") {
		line, _, _ = strings.Cut(line, "//")
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == legacy {
			return true // within require(...) or replace(...)
		}
		if len(fields) >= 3 && fields[0] == "require" && fields[1] == legacy {
			return true
		}
		if len(fields) >= 3 && fields[0] == "replace" && fields[1] == legacy {
			return true
		}
	}
	return false
}
