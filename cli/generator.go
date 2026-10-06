package cli

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var safeName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

func writeGenerated(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("copytygo: %s already exists", path)
	}
	return os.WriteFile(path, []byte(content), 0644)
}
func MakeController(name, dir string) error {
	return MakeControllerWithMode(name, dir, "")
}

func MakeControllerWithMode(name, dir, mode string) error {
	if !safeName.MatchString(name) {
		return fmt.Errorf("invalid controller name")
	}
	if !strings.HasSuffix(name, "Controller") {
		name += "Controller"
	}

	var body string
	resourceName := strings.ToLower(strings.TrimSuffix(name, "Controller")) + "s"

	switch mode {
	case "resource":
		body = fmt.Sprintf(`package controllers

import "github.com/arfajhf/copytygo/v3/core"

// copytygo:resource %s

type %s struct{}

func (%s) Index(ctx *core.Context) error {
	return ctx.DBIndex("%s")
}

func (%s) Show(ctx *core.Context) error {
	return ctx.DBShow("%s", ctx.Param("id"))
}

func (%s) Store(ctx *core.Context) error {
	if err := ctx.Validate(map[string]string{
		"name": "required|min:3",
	}); err != nil {
		return err
	}

	return ctx.DBStore("%s", core.Map{
		"name": ctx.Input("name"),
	})
}

func (%s) Update(ctx *core.Context) error {
	if err := ctx.Validate(map[string]string{
		"name": "required|min:3",
	}); err != nil {
		return err
	}

	return ctx.DBUpdate("%s", ctx.Param("id"), core.Map{
		"name": ctx.Input("name"),
	})
}

func (%s) Destroy(ctx *core.Context) error {
	return ctx.DBDestroy("%s", ctx.Param("id"))
}
`, resourceName, name, name, resourceName, name, resourceName, name, resourceName, name, resourceName, name, resourceName)

	case "memory":
		body = fmt.Sprintf(`package controllers

import "github.com/arfajhf/copytygo/v3/core"

type %s struct{}

func (%s) Index(ctx *core.Context) error {
	return ctx.MemoryIndex("%s")
}

func (%s) Show(ctx *core.Context) error {
	return ctx.MemoryShow("%s", ctx.Param("id"))
}

func (%s) Store(ctx *core.Context) error {
	if err := ctx.Validate(map[string]string{
		"name": "required|min:3",
	}); err != nil {
		return err
	}
	return ctx.MemoryStore("%s", core.Map{"name": ctx.Input("name")})
}

func (%s) Update(ctx *core.Context) error {
	if err := ctx.Validate(map[string]string{
		"name": "required|min:3",
	}); err != nil {
		return err
	}
	return ctx.MemoryUpdate("%s", ctx.Param("id"), core.Map{"name": ctx.Input("name")})
}

func (%s) Destroy(ctx *core.Context) error {
	return ctx.MemoryDestroy("%s", ctx.Param("id"))
}
`, name, name, resourceName, name, resourceName, name, resourceName, name, resourceName, name, resourceName)

	default:
		body = fmt.Sprintf("package controllers\n\nimport \"github.com/arfajhf/copytygo/v3/core\"\n\ntype %s struct{}\n\nfunc (%s) Index(ctx *core.Context) error {\n\treturn ctx.JSON(core.Map{\"data\": []any{}})\n}\n", name, name)
	}

	return writeGenerated(filepath.Join(dir, strings.ToLower(strings.TrimSuffix(name, "Controller"))+"_controller.go"), body)
}

func MakeResource(name string) error {
	return MakeResourceWithFields(name, nil)
}

func MakeModel(name, dir string) error {
	if !safeName.MatchString(name) {
		return fmt.Errorf("invalid model name")
	}
	body := fmt.Sprintf("package models\n\ntype %s struct {\n\tID int64 `json:\"id\"`\n\tCreatedAt string `json:\"created_at\"`\n\tUpdatedAt string `json:\"updated_at\"`\n}\n", name)
	return writeGenerated(filepath.Join(dir, strings.ToLower(name)+".go"), body)
}
func MakeMiddleware(name, dir string) error {
	if !safeName.MatchString(name) {
		return fmt.Errorf("invalid middleware name")
	}
	body := fmt.Sprintf("package middleware\n\nimport \"github.com/arfajhf/copytygo/v3/core\"\n\nfunc %s() core.Middleware {\n\treturn func(next core.Handler) core.Handler {\n\t\treturn func(ctx *core.Context) error { return next(ctx) }\n\t}\n}\n", name)
	return writeGenerated(filepath.Join(dir, strings.ToLower(name)+".go"), body)
}
func MakeService(name, dir string) error {
	if !safeName.MatchString(name) {
		return fmt.Errorf("invalid service name")
	}
	body := fmt.Sprintf("package services\n\ntype %s struct{}\n\nfunc New%s() *%s { return &%s{} }\n", name, name, name, name)
	return writeGenerated(filepath.Join(dir, strings.ToLower(name)+".go"), body)
}
func GenerateKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
