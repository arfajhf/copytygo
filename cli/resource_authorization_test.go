package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resourceTestProject(t *testing.T, role, kind, access string) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{"app/controllers", "routes"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"go.mod": "module example.com/app\n",
		"routes/web.go": "package routes\nimport \"github.com/arfajhf/copytygo/v5/core\"\nfunc Register(app *core.Application) {}\n",
		"app/controllers/product_controller.go": "package controllers\n// copytygo:resource products\n// copytygo:resource-access " + access + "\ntype ProductController struct{}\n",
	}
	if kind != "none" {
		files["app/auth/web.go"] = "package auth\nconst DefaultRole = \"" + role + "\"\nfunc Sessions() {}\n"
		files["routes/auth.go"] = "package routes\nfunc RegisterAuth() {\n"
		if kind == "browser" {
			files["routes/auth.go"] += "appauth.Sessions()\n"
			files["frontend/auth.html"] = "<html></html>"
		}
		files["routes/auth.go"] += "}\n"
	}
	for path, contents := range files {
		target := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestGeneratedResourceAuthorizationModes(t *testing.T) {
	tests := []struct {
		name, role, kind, access string
		want []string
		forbid []string
	}{
		{"no auth locked", "", "none", "default",
			[]string{"core.ResourceAccessUnconfigured()"}, []string{"app.Resource(\"/products\""}},
		{"public explicit", "", "none", "public",
			[]string{"app.Resource(\"/products\", controllers.ProductController{})"}, []string{"ResourceWithMiddleware"}},
		{"multi browser admin", "user", "browser", "default",
			[]string{"sessions.Middleware()", "sessions.RequireAdmin()", "sessions.CSRF()"}, []string{"copyauth.RequireRole"}},
		{"multi browser user", "user", "browser", "user",
			[]string{"sessions.Middleware()", "copyauth.RequireRole(\"user\")", "sessions.CSRF()"}, []string{"sessions.RequireAdmin()"}},
		{"multi bearer admin", "user", "bearer", "default",
			[]string{"copyauth.Middleware()", "copyauth.RequireRole(\"admin\")"}, []string{"sessions.CSRF()"}},
		{"single browser auth", "", "browser", "default",
			[]string{"sessions.Middleware()", "sessions.CSRF()"}, []string{"RequireAdmin()", "copyauth.RequireRole"}},
		{"single role admin blocked", "", "browser", "admin",
			[]string{"core.ResourceAccessUnconfigured()"}, []string{"sessions.RequireAdmin()"}},
		{"multi bearer authenticated", "user", "bearer", "authenticated",
			[]string{"copyauth.Middleware()"}, []string{"copyauth.RequireRole"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := resourceTestProject(t, tc.role, tc.kind, tc.access)
			if err := GenerateResourceRoutes(root); err != nil {
				t.Fatal(err)
			}
			contents, err := os.ReadFile(filepath.Join(root, "routes", "resources.go"))
			if err != nil {
				t.Fatal(err)
			}
			code := string(contents)
			for _, needle := range tc.want {
				if !strings.Contains(code, needle) {
					t.Fatalf("missing %s in generated routes:\n%s", needle, code)
				}
			}
			for _, needle := range tc.forbid {
				if strings.Contains(code, needle) {
					t.Fatalf("unexpected %s in generated routes:\n%s", needle, code)
				}
			}
		})
	}
}

func TestLegacyResourceMarkerDefaultsToProtected(t *testing.T) {
	root := resourceTestProject(t, "user", "browser", "default")
	path := filepath.Join(root, "app", "controllers", "product_controller.go")
	if err := os.WriteFile(path, []byte("package controllers\n// copytygo:resource products\ntype ProductController struct{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := GenerateResourceRoutes(root); err != nil {
		t.Fatal(err)
	}
	code, err := os.ReadFile(filepath.Join(root, "routes", "resources.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(code), "sessions.RequireAdmin()") {
		t.Fatalf("legacy resource remained public after resync: %s", code)
	}
}

func TestGenerateResourceRoutesWithoutControllers(t *testing.T) {
	root := resourceTestProject(t, "", "none", "default")
	if err := os.Remove(filepath.Join(root, "app", "controllers", "product_controller.go")); err != nil {
		t.Fatal(err)
	}
	if err := GenerateResourceRoutes(root); err != nil {
		t.Fatal(err)
	}
	code, err := os.ReadFile(filepath.Join(root, "routes", "resources.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(code), "/app/controllers") {
		t.Fatalf("unused controller import: %s", code)
	}
}
