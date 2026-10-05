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
    "net/http"
    "github.com/arfajhf/copytygo/config"
    "github.com/arfajhf/copytygo/core"
    "github.com/arfajhf/copytygo/security"
)

func Middleware() core.Middleware { return func(next core.Handler) core.Handler { return func(ctx *core.Context) error { token:=ctx.BearerToken(); if token=="" { return core.NewHTTPError(http.StatusUnauthorized,"Authentication required") }; if _,err:=security.VerifyToken(config.Get("APP_KEY"),token);err!=nil{return core.NewHTTPError(http.StatusUnauthorized,"Invalid authentication token")}; return next(ctx) } } }
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
