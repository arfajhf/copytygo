package cli

import (
	"fmt"
	copyauth "github.com/arfajhf/copytygo/v4/auth"
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

func TestAuthStarterUsesTypeScriptAndExplicitJSONRoutes(t *testing.T) {
	for _, mode := range []string{"single", "multi"} {
		t.Run(mode, func(t *testing.T) {
			root := authProject(t)
			if err := InstallAuth(mode, root); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"app/auth/web.go", "app/auth/README.md", "frontend/src/auth/views.ts", "frontend/src/auth/main.ts", "frontend/src/auth/api.ts", "frontend/src/auth/style.css", "frontend/auth.html"} {
				if _, err := os.Stat(filepath.Join(root, name)); err != nil {
					t.Fatal(err)
				}
			}
			routes, err := os.ReadFile(filepath.Join(root, "routes/auth.go"))
			if err != nil {
				t.Fatal(err)
			}
			for _, expected := range []string{"appauth.Sessions()", `"/login"`, `app.Post("/api/session/register"`, `app.Post("/api/session/logout"`, "copyauth.RegisterHandler(appauth.DefaultRole)", "sessions.Middleware()", "sessions.CSRF()", "frontend.Page()"} {
				if !strings.Contains(string(routes), expected) {
					t.Fatalf("auth routes missing %s", expected)
				}
			}
			if _, err := os.Stat(filepath.Join(root, "app/auth/views")); !os.IsNotExist(err) {
				t.Fatal("fresh auth generated Go HTML views")
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
	files := []string{"app/auth/user.go", "app/auth/middleware.go", "app/auth/web.go", "routes/auth.go", "frontend/src/auth/views.ts"}
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
	if !strings.Contains(string(source), "frontend.Page()") {
		t.Fatal("old API-only auth did not gain browser pages")
	}
	if _, err := os.Stat(filepath.Join(root, "frontend/src/auth/views.ts")); err != nil {
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

func TestAuthUpgradesUnchangedV403Starter(t *testing.T) {
	root := authProject(t)
	if err := InstallAuth("multi", root); err != nil {
		t.Fatal(err)
	}
	module, err := projectModule(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	route, _ := legacyAuthStarter.ReadFile("auth_legacy_v403/routes.txt")
	raw, err := format.Source([]byte(fmt.Sprintf(string(route), module)))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "routes/auth.go"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(root, "app/auth/views"), 0755)
	for _, name := range []string{"layout", "account"} {
		old, _ := legacyAuthStarter.ReadFile("auth_legacy_v403/" + name + ".html")
		if err := os.WriteFile(filepath.Join(root, "app/auth/views/"+name+".html"), old, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := InstallAuth("multi", root); err != nil {
		t.Fatal(err)
	}
	routes, _ := os.ReadFile(filepath.Join(root, "routes/auth.go"))
	if !strings.Contains(string(routes), "sessions.Dashboard") || !strings.Contains(string(routes), "sessions.RequireAdmin") {
		t.Fatal("v4.0.3 routes did not upgrade")
	}
	if _, err := os.Stat(filepath.Join(root, "app/auth/views")); !os.IsNotExist(err) {
		t.Fatal("unchanged Go HTML starter should be removed")
	}
}

func TestAuthMigratesV404AndPreservesCustomLegacyView(t *testing.T) {
	root := authProject(t)
	module, _ := projectModule(filepath.Join(root, "go.mod"))
	os.MkdirAll(filepath.Join(root, "app/auth/views"), 0755)
	for name, arg := range map[string]string{"routes.txt": module, "web.txt": "user"} {
		old, _ := legacyAuthStarter.ReadFile("auth_legacy_v404/" + name)
		raw, err := format.Source([]byte(fmt.Sprintf(string(old), arg)))
		if err != nil {
			t.Fatal(err)
		}
		path := "routes/auth.go"
		if name == "web.txt" {
			path = "app/auth/web.go"
		}
		os.WriteFile(filepath.Join(root, path), raw, 0644)
	}
	old, _ := legacyAuthStarter.ReadFile("auth_legacy_v404/README.txt")
	os.WriteFile(filepath.Join(root, "app/auth/README.md"), old, 0644)
	views, err := copyauth.StarterViews()
	if err != nil {
		t.Fatal(err)
	}
	for name, raw := range views {
		os.WriteFile(filepath.Join(root, "app/auth/views", name), []byte(raw), 0644)
	}
	custom := views["login.html"] + "\n<!-- my customized login -->"
	os.WriteFile(filepath.Join(root, "app/auth/views/login.html"), []byte(custom), 0644)
	if err := InstallAuth("multi", root); err != nil {
		t.Fatal(err)
	}
	web, _ := os.ReadFile(filepath.Join(root, "app/auth/web.go"))
	if strings.Contains(string(web), "embed") || !strings.Contains(string(web), "NewBrowserAuth") {
		t.Fatal("old Go view controller was not migrated")
	}
	if _, err := os.Stat(filepath.Join(root, "app/auth/views/register.html")); !os.IsNotExist(err) {
		t.Fatal("default HTML not removed")
	}
	got, _ := os.ReadFile(filepath.Join(root, "app/auth/views/login.html"))
	if string(got) != custom {
		t.Fatal("custom legacy file lost")
	}
	welcome, _ := os.ReadFile(filepath.Join(root, "routes/web.go"))
	if !strings.Contains(string(welcome), "frontend.Page()") {
		t.Fatal("welcome did not migrate to TS")
	}
}

func TestAuthAPIOnlyDoesNotAddFrontend(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	options := DefaultProjectOptions()
	options.Frontend = "api"
	if err := NewProjectWithOptions("app", options); err != nil {
		t.Fatal(err)
	}
	root = filepath.Join(root, "app")
	for i := 0; i < 2; i++ {
		if err := InstallAuth("multi", root); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "frontend")); !os.IsNotExist(err) {
		t.Fatal("API-only gained frontend")
	}
	if _, err := os.Stat(filepath.Join(root, "app/auth/views")); !os.IsNotExist(err) {
		t.Fatal("API-only gained Go views")
	}
	routes, _ := os.ReadFile(filepath.Join(root, "routes/auth.go"))
	if strings.Contains(string(routes), "frontend.") {
		t.Fatal("API-only gained frontend routes")
	}
}
