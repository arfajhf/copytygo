package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

func InstallAuth(mode, root string) error {
	if mode != "single" && mode != "multi" {
		return fmt.Errorf("copytygo: auth mode must be single or multi")
	}
	if err := os.MkdirAll(filepath.Join(root, "app", "auth"), 0755); err != nil {
		return err
	}
	roleField := ""
	if mode == "multi" {
		roleField = "\tRole string `json:\"role\"`\n"
	}
	model := fmt.Sprintf("package auth\n\ntype User struct {\n\tID int64 `json:\"id\"`\n\tName string `json:\"name\"`\n\tEmail string `json:\"email\"`\n\tPassword string `json:\"-\"`\n%s}\n", roleField)
	middleware := `package auth

import (
    copyauth "github.com/arfajhf/copytygo/auth"
    "github.com/arfajhf/copytygo/core"
)

func Middleware() core.Middleware {
    return copyauth.Middleware()
}

func RequireRole(roles ...string) core.Middleware {
    return copyauth.RequireRole(roles...)
}
`
	if err := os.WriteFile(filepath.Join(root, "app", "auth", "user.go"), []byte(model), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "app", "auth", "middleware.go"), []byte(middleware), 0644); err != nil {
		return err
	}
	fmt.Printf("Auth scaffold installed (%s role).\n", mode)
	return nil
}
