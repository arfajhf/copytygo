package cli

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func authProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	if err := NewProject("app"); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, "app")
}

func TestAuthStarterHasEditablePagesAndExplicitRoutes(t *testing.T) {
	for _, mode := range []string{"single", "multi"} {
		t.Run(mode, func(t *testing.T) {
			root := authProject(t)
			if err := InstallAuth(mode, root); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"app/auth/web.go", "app/auth/README.md", "app/auth/views/layout.html", "app/auth/views/login.html", "app/auth/views/register.html", "app/auth/views/account.html"} {
				if _, err := os.Stat(filepath.Join(root, name)); err != nil {
					t.Fatal(err)
				}
			}
			routes, err := os.ReadFile(filepath.Join(root, "routes/auth.go"))
			if err != nil {
				t.Fatal(err)
			}
			for _, expected := range []string{"appauth.Pages()", `app.Get("/login"`, `app.Post("/register"`, `app.Post("/logout"`, "copyauth.RegisterHandler(appauth.DefaultRole)", "pages.Middleware()"} {
				if !strings.Contains(string(routes), expected) {
					t.Fatalf("auth routes missing %s", expected)
				}
			}
			if strings.Contains(string(routes), "DO NOT EDIT") {
				t.Fatal("editable auth routes marked generated-only")
			}
			t.Chdir(root)
			liteRoutes, err := parseLiteRouteFile("routes/auth.go")
			if err != nil {
				t.Fatal(err)
			}
			if len(liteRoutes) != 3 {
				t.Fatalf("explicit API handlers not discovered in Lite: %#v", liteRoutes)
			}
			role := ""
			if mode == "multi" {
				role = "user"
			}
			if liteRoutes[0].AuthRole != role {
				t.Fatal("Lite registration lost configured default role")
			}
		})
	}
}

func TestAuthReinstallPreservesCustomizedFiles(t *testing.T) {
	root := authProject(t)
	if err := InstallAuth("multi", root); err != nil {
		t.Fatal(err)
	}
	files := []string{"app/auth/user.go", "app/auth/middleware.go", "app/auth/web.go", "routes/auth.go", "app/auth/views/login.html"}
	before := map[string]string{}
	for _, file := range files {
		path := filepath.Join(root, file)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		content := string(raw) + "\n// Customized by the application\n"
		if strings.HasSuffix(file, ".html") {
			content = string(raw) + "\n<!-- Custom login -->\n"
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		before[file] = content
	}
	if err := InstallAuth("multi", root); err != nil {
		t.Fatal(err)
	}
	for file, want := range before {
		got, err := os.ReadFile(filepath.Join(root, file))
		if err != nil || string(got) != want {
			t.Fatalf("reinstall changed %s", file)
		}
	}
	migrations, _ := filepath.Glob(filepath.Join(root, "database/migrations/*create_users_table.go"))
	if len(migrations) != 1 {
		t.Fatal("reinstall duplicated users migration")
	}
}

func TestAuthUpgradesUnchangedLegacyAPIRoutes(t *testing.T) {
	root := authProject(t)
	raw, err := format.Source([]byte(fmt.Sprintf(legacyAuthRoutesScaffold, "user")))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "routes/auth.go"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	if err := InstallAuth("multi", root); err != nil {
		t.Fatal(err)
	}
	source, _ := os.ReadFile(filepath.Join(root, "routes/auth.go"))
	if !strings.Contains(string(source), "pages.RegisterPage") {
		t.Fatal("old API-only auth did not gain browser pages")
	}
	if _, err := os.Stat(filepath.Join(root, "app/auth/views/register.html")); err != nil {
		t.Fatal(err)
	}
}

func TestAuthDoesNotReplaceCustomLegacyRoutes(t *testing.T) {
	root := authProject(t)
	custom := fmt.Sprintf(legacyAuthRoutesScaffold, "user") + "\n// custom application behavior\n"
	path := filepath.Join(root, "routes/auth.go")
	if err := os.WriteFile(path, []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}
	if err := InstallAuth("multi", root); err == nil {
		t.Fatal("custom routes should require manual integration")
	}
	got, _ := os.ReadFile(path)
	if string(got) != custom {
		t.Fatal("custom routes changed")
	}
	if _, err := os.Stat(filepath.Join(root, "app/auth")); !os.IsNotExist(err) {
		t.Fatal("failed install added partial auth files")
	}
}
