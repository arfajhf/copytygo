package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateResourceRoutes(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "app", "controllers"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "routes"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(root, "go.mod"),
		[]byte("module example.com/store\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	controller := "package controllers\n\n// copytygo:resource products\n\ntype ProductController struct{}\n"
	if err := os.WriteFile(
		filepath.Join(root, "app", "controllers", "product_controller.go"),
		[]byte(controller),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	web := "package routes\n\nimport \"github.com/arfajhf/copytygo/v2/core\"\n\nfunc Register(app *core.Application) {\n}\n"
	if err := os.WriteFile(
		filepath.Join(root, "routes", "web.go"),
		[]byte(web),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := GenerateResourceRoutes(root); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(root, "routes", "resources.go"))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(raw)

	if !strings.Contains(generated, "\"example.com/store/app/controllers\"") {
		t.Fatalf("missing controller import: %s", generated)
	}
	if !strings.Contains(generated, "app.Resource(\"/products\", controllers.ProductController{})") {
		t.Fatalf("missing generated resource route: %s", generated)
	}

	raw, err = os.ReadFile(filepath.Join(root, "routes", "web.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "RegisterGeneratedResources(app)") {
		t.Fatal("web routes did not call generated resources")
	}
}
