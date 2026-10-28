package cli

import (
	"fmt"
	"strings"
)

func validResourceAccess(access string) bool {
	switch access {
	case "default", "authenticated", "admin", "user", "public":
		return true
	default:
		return false
	}
}

// parseResourceArguments keeps policy flags separate from schema fields.
// A resource is never public merely because auth is not installed.
func parseResourceArguments(args []string) ([]string, string, error) {
	fields := make([]string, 0, len(args))
	access := "default"
	seen := false
	for _, arg := range args {
		switch {
		case arg == "--public":
			if seen {
				return nil, "", fmt.Errorf("copytygo: resource access set more than once")
			}
			access, seen = "public", true
		case strings.HasPrefix(arg, "--access="):
			if seen {
				return nil, "", fmt.Errorf("copytygo: resource access set more than once")
			}
			access, seen = strings.TrimPrefix(arg, "--access="), true
			if !validResourceAccess(access) {
				return nil, "", fmt.Errorf("copytygo: invalid resource access %q", access)
			}
		case strings.HasPrefix(arg, "-"):
			return nil, "", fmt.Errorf("copytygo: unknown resource flag %q", arg)
		default:
			fields = append(fields, arg)
		}
	}
	return fields, access, nil
}
