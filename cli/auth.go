package cli

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"time"

	copyauth "github.com/arfajhf/copytygo/v4/auth"
)

//go:embed auth_legacy_v403/* auth_legacy_v404/* auth_frontend/* auth_frontend/src/auth/*
var legacyAuthStarter embed.FS

func InstallAuth(mode, root string) error {
	if mode != "single" && mode != "multi" {
		return fmt.Errorf("copytygo: auth mode must be single or multi")
	}
	module, err := projectModule(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	// Check existing routes before adding files. Customized routes remain yours.
	routePath := filepath.Join(root, "routes", "auth.go")
	current, err := os.ReadFile(routePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	updateRoutes := os.IsNotExist(err) || unchangedCurrentAuthRoutes(current, module) || legacyAuthRoutes(current) || unchangedBrowserRoutes(current, module) || matchesAuthSnapshot(current, "routes.txt", module)
	if !updateRoutes && !strings.Contains(string(current), "appauth.Sessions()") {
		return fmt.Errorf("copytygo: routes/auth.go has custom auth routes; integrate the TypeScript starter routes from docs/AUTHENTICATION.md manually")
	}
	webPath := filepath.Join(root, "app/auth/web.go")
	if raw, err := os.ReadFile(webPath); err == nil && strings.Contains(string(raw), "func Pages()") && !unchangedAuthWeb(raw) {
		return fmt.Errorf("copytygo: app/auth/web.go has custom HTML controllers; migrate it to Sessions() using docs/AUTHENTICATION.md before reinstalling")
	}
	frontendEnabled := fileExists(filepath.Join(root, "frontend/package.json"))
	roleField, defaultRole := "", ""
	if mode == "multi" {
		roleField = "\tRole string `json:\"role\"`\n"
		defaultRole = "user"
	}
	model := fmt.Sprintf("package auth\n\n// User describes the application user. Password hashes are never serialized.\ntype User struct {\n\tID int64 `json:\"id\"`\n\tName string `json:\"name\"`\n\tEmail string `json:\"email\"`\n\tPassword string `json:\"-\"`\n%s}\n", roleField)
	files := map[string]string{
		"app/auth/user.go":       model,
		"app/auth/middleware.go": authMiddlewareScaffold,
		"app/auth/web.go":        fmt.Sprintf(authWebScaffold, defaultRole),
		"app/auth/README.md":     authScaffoldReadme,
	}
	if frontendEnabled {
		for _, path := range []string{"auth.html", "vite.auth.config.ts", "src/auth/api.ts", "src/auth/views.ts", "src/auth/main.ts", "src/auth/style.css"} {
			raw, err := legacyAuthStarter.ReadFile("auth_frontend/" + path)
			if err != nil {
				return err
			}
			files["frontend/"+path] = string(raw)
		}
	}
	for path, content := range files {
		if err := writeAuthStarterFile(root, path, content); err != nil {
			return err
		}
	}
	if err := makeAuthMigration(mode, filepath.Join(root, "database", "migrations")); err != nil {
		return err
	}
	if updateRoutes {
		routes := fmt.Sprintf(authRoutesScaffold, module)
		if !frontendEnabled {
			routes = fmt.Sprintf(authAPIRoutesScaffold, module)
		}
		source, err := format.Source([]byte(routes))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(routePath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(routePath, source, 0644); err != nil {
			return err
		}
	}
	webPath = filepath.Join(root, "routes", "web.go")
	raw, err := os.ReadFile(webPath)
	if err != nil {
		return err
	}
	web := string(raw)
	if !strings.Contains(web, "RegisterAuth(app)") {
		needle := "func Register(app *core.Application) {"
		index := strings.Index(web, needle)
		if index < 0 {
			return fmt.Errorf("copytygo: routes.Register function not found")
		}
		at := index + len(needle)
		web = web[:at] + "\n RegisterAuth(app)" + web[at:]
	}
	if frontendEnabled && strings.Contains(web, `app.Get("/", core.Welcome()).Name("welcome")`) {
		web = strings.Replace(web, `app.Get("/", core.Welcome()).Name("welcome")`, `app.Get("/", frontend.Page()).Name("welcome")`, 1)
		web = strings.Replace(web, `"github.com/arfajhf/copytygo/v4/core"`, `"github.com/arfajhf/copytygo/v4/core"`+"\n"+`"github.com/arfajhf/copytygo/v4/frontend"`, 1)
	}
	source, err := format.Source([]byte(web))
	if err != nil {
		return err
	}
	if err := os.WriteFile(webPath, source, 0644); err != nil {
		return err
	}
	if err := removeUnchangedHTMLViews(root); err != nil {
		return err
	}
	fmt.Printf("Auth starter installed (%s role).\n", mode)
	fmt.Println("1. Set your database connection in .env, then run: ctg migrate")
	fmt.Println("2. Start the application: ctg dev")
	if frontendEnabled {
		fmt.Println("3. Open the application URL. Login/register, dashboard and Users are TypeScript UI.")
		fmt.Println("Customize: frontend/src/auth/*.ts, frontend/src/auth/style.css, routes/auth.go")
		fmt.Println("ctg dev starts Go + Vite; ctg build compiles TypeScript + Go. Node.js is required to develop/build the frontend.")
	} else {
		fmt.Println("API-only project: use /api/auth/register, /api/auth/login and /api/auth/me.")
	}
	if mode == "multi" {
		fmt.Println("First admin: register your account, then run ctg auth:admin <email> locally.")
	}
	fmt.Println("Guide: app/auth/README.md. Customized source files are preserved.")
	return nil
}

func writeAuthStarterFile(root, path, content string) error {
	target := filepath.Join(root, filepath.FromSlash(path))
	if current, err := os.ReadFile(target); err == nil {
		legacyName := map[string]string{"app/auth/views/layout.html": "layout.html", "app/auth/views/account.html": "account.html", "app/auth/README.md": "README.txt"}[path]
		legacy, readErr := legacyAuthStarter.ReadFile("auth_legacy_v403/" + legacyName)
		upgrade := legacyName != "" && readErr == nil && bytes.Equal(current, legacy)
		if path == "app/auth/web.go" {
			upgrade = unchangedAuthWeb(current)
		}
		if path == "app/auth/README.md" {
			old, _ := legacyAuthStarter.ReadFile("auth_legacy_v404/README.txt")
			upgrade = upgrade || bytes.Equal(current, old)
		}
		if !upgrade {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	data := []byte(content)
	if filepath.Ext(path) == ".go" {
		formatted, err := format.Source(data)
		if err != nil {
			return fmt.Errorf("copytygo auth scaffold %s: %w", path, err)
		}
		data = formatted
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0644)
}

func unchangedBrowserRoutes(raw []byte, module string) bool {
	expected, err := legacyAuthStarter.ReadFile("auth_legacy_v403/routes.txt")
	if err != nil {
		return false
	}
	old, err := format.Source([]byte(fmt.Sprintf(string(expected), module)))
	if err != nil {
		return false
	}
	actual, err := format.Source(raw)
	return err == nil && bytes.Equal(bytes.TrimSpace(old), bytes.TrimSpace(actual))
}

func legacyAuthRoutes(raw []byte) bool {
	actual, err := format.Source(raw)
	if err != nil {
		return false
	}
	for _, role := range []string{"", "user"} {
		expected, _ := format.Source([]byte(fmt.Sprintf(legacyAuthRoutesScaffold, role)))
		if bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
			return true
		}
	}
	return false
}

func makeAuthMigration(mode, directory string) error {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.Contains(entry.Name(), "create_users_table") && filepath.Ext(entry.Name()) == ".go" {
			return nil
		}
	}

	now := time.Now()
	timestamp := now.Format("20060102_150405")
	functionID := migrationFunctionID(now.Format("20060102150405"), "create_users_table")
	filename := timestamp + "_create_users_table.go"
	path := filepath.Join(directory, filename)

	roleColumn := ""
	if mode == "multi" {
		roleColumn = "\t\t\t\t\ttable.String(\"role\").DefaultValue(\"user\")\n"
	}

	content := fmt.Sprintf(`package migrations

import (
	"github.com/arfajhf/copytygo/v4/database/migration"
	"github.com/arfajhf/copytygo/v4/database/schema"
)

func Register%s() error {
	return migration.Register(
		%q,
		func() *schema.Blueprint {
			return schema.Create("users", func(table *schema.Table) {
				table.ID()
				table.String("name")
				table.String("email").Unique()
				table.String("password")
%s				table.Timestamps()
			})
		},
		func() *schema.Blueprint {
			return schema.Drop("users")
		},
	)
}
`, functionID, timestamp+"_create_users_table", roleColumn)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	return GenerateMigrationRegistry(directory)
}

const authMiddlewareScaffold = `package auth

import (
 copyauth "github.com/arfajhf/copytygo/v4/auth"
 "github.com/arfajhf/copytygo/v4/core"
)

// Middleware protects bearer-token API routes. Browser JSON APIs use Sessions().Middleware().
func Middleware() core.Middleware { return copyauth.Middleware() }

// RequireRole must run after authentication middleware.
func RequireRole(roles ...string) core.Middleware { return copyauth.RequireRole(roles...) }
`

const authWebScaffold = `package auth

import copyauth "github.com/arfajhf/copytygo/v4/auth"

// DefaultRole is assigned by Go during public registration, never by the client.
// Multi-role accounts start as "user"; create the first admin with ctg auth:admin.
const DefaultRole = %q

// Sessions supplies JSON controllers. All UI lives in frontend/src/auth/.
func Sessions() *copyauth.BrowserAuth { return copyauth.NewBrowserAuth(DefaultRole) }
`

const authRoutesScaffold = `package routes

import (
 appauth "%s/app/auth"
 copyauth "github.com/arfajhf/copytygo/v4/auth"
 "github.com/arfajhf/copytygo/v4/core"
 "github.com/arfajhf/copytygo/v4/frontend"
)

// Go defines API routes; TypeScript renders the pages in frontend/src/auth/.
func RegisterAuth(app *core.Application) {
 sessions := appauth.Sessions()
 frontend.Assets(app)
 for _, path := range []string{"/login", "/register", "/dashboard", "/account", "/users", "/users/create", "/users/:id/edit"} {
  app.Get(path, frontend.Page())
 }

 // Browser JSON APIs use HttpOnly cookie sessions and X-CSRF-Token.
 app.Get("/api/session", sessions.Session)
 app.Post("/api/session/login", sessions.Login).Middleware(sessions.CSRF())
 app.Post("/api/session/register", sessions.Register).Middleware(sessions.CSRF())
 app.Post("/api/session/logout", sessions.Logout).Middleware(sessions.Middleware(), sessions.CSRF())
 app.Get("/api/dashboard", sessions.Dashboard).Middleware(sessions.Middleware())

 // Always enforce fresh database roles on the backend, including every write.
 if appauth.DefaultRole != "" {
  app.Get("/api/users", sessions.Users).Middleware(sessions.Middleware(), sessions.RequireAdmin())
  app.Get("/api/users/:id", sessions.User).Middleware(sessions.Middleware(), sessions.RequireAdmin())
  app.Post("/api/users", sessions.CreateUser).Middleware(sessions.Middleware(), sessions.RequireAdmin(), sessions.CSRF())
  app.Put("/api/users/:id", sessions.UpdateUser).Middleware(sessions.Middleware(), sessions.RequireAdmin(), sessions.CSRF())
  app.Delete("/api/users/:id", sessions.DeleteUser).Middleware(sessions.Middleware(), sessions.RequireAdmin(), sessions.CSRF())
 }

 // Bearer APIs remain available for external/mobile clients.
 app.Post("/api/auth/register", copyauth.RegisterHandler(appauth.DefaultRole))
 app.Post("/api/auth/login", copyauth.LoginHandler())
 app.Get("/api/auth/me", copyauth.MeHandler()).Middleware(copyauth.Middleware())
}
`

const authAPIRoutesScaffold = `package routes
import (
 appauth "%s/app/auth"
 copyauth "github.com/arfajhf/copytygo/v4/auth"
 "github.com/arfajhf/copytygo/v4/core"
)
func RegisterAuth(app *core.Application) {
 app.Post("/api/auth/register", copyauth.RegisterHandler(appauth.DefaultRole))
 app.Post("/api/auth/login", copyauth.LoginHandler())
 app.Get("/api/auth/me", copyauth.MeHandler()).Middleware(copyauth.Middleware())
}
`

const legacyAuthRoutesScaffold = `// Code generated by CopyTyGo. DO NOT EDIT.

package routes

import (
 copyauth "github.com/arfajhf/copytygo/v4/auth"
 "github.com/arfajhf/copytygo/v4/core"
)

func RegisterAuth(app *core.Application) {
 copyauth.Routes(app, %q)
}
`

const authScaffoldReadme = `# Your TypeScript + Go authentication starter

1. Configure MySQL/PostgreSQL in .env and run ctg migrate.
2. Install Node.js 22+ and Go. Run ctg dev from the project root.
3. Open the application URL. Vite runs automatically alongside the native Go app.
4. Register or log in. You arrive at /dashboard.

Edit the frontend:
- frontend/src/auth/views.ts: welcome/navbar, login/register, dashboard, account, Users and user form.
- frontend/src/auth/main.ts: page routing, form submission and interactions.
- frontend/src/auth/api.ts: typed JSON client and API response types.
- frontend/src/auth/style.css: responsive styling.
- frontend/vite.auth.config.ts: auth entry plus your existing frontend preset.

Edit the backend:
- routes/auth.go: JSON endpoints and frontend page routes.
- app/auth/web.go: default role and JSON controller configuration.
- app/auth/user.go: application user shape.
- app/auth/middleware.go: bearer API helpers.

TypeScript changes reload through Vite. Restart ctg dev after Go changes.
ctg build type-checks/builds the frontend and compiles the Go application.
Deploy build/app (app.exe on Windows) with build/frontend/. Start from the build
folder so frontend/dist is available. Node.js is not needed on production servers.
A custom location can be set with COPYTYGO_FRONTEND_DIST.

Browser JSON endpoints live at /api/session, /api/dashboard and /api/users.
Cookie sessions are encrypted and HttpOnly; do not store auth tokens in localStorage.
Every write includes X-CSRF-Token obtained from GET /api/session. Go checks it.
Bearer /api/auth endpoints remain available for external/mobile clients.

Public multi-role registration starts as user. Register the first admin normally,
then run ctg auth:admin your-email@example.com locally inside the project.
Only admins can manage Users. Go reads roles from the database for each request
and checks write permissions within a transaction. Passwords never appear in UI.
You cannot delete or demote your own administrator account from Users.

Reinstall using the same mode. Unchanged v4.0.3/v4.0.4 Go HTML starter files are
migrated to TypeScript. Customized old routes/controllers require manual migration;
customized HTML files are preserved on disk but are no longer starter UI.
API-only projects keep bearer endpoints and do not add a frontend or Node dependency.
`

func fileExists(path string) bool { info, err := os.Stat(path); return err == nil && !info.IsDir() }

func matchesAuthSnapshot(raw []byte, name, argument string) bool {
	expected, err := legacyAuthStarter.ReadFile("auth_legacy_v404/" + name)
	if err != nil {
		return false
	}
	old, err := format.Source([]byte(fmt.Sprintf(string(expected), argument)))
	if err != nil {
		return false
	}
	actual, err := format.Source(raw)
	return err == nil && bytes.Equal(bytes.TrimSpace(old), bytes.TrimSpace(actual))
}
func unchangedAuthWeb(raw []byte) bool {
	return matchesAuthSnapshot(raw, "web.txt", "") || matchesAuthSnapshot(raw, "web.txt", "user")
}
func removeUnchangedHTMLViews(root string) error {
	views, err := copyauth.StarterViews()
	if err != nil {
		return err
	}
	for name, expected := range views {
		path := filepath.Join(root, "app/auth/views", name)
		current, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		legacy, _ := legacyAuthStarter.ReadFile("auth_legacy_v403/" + name)
		if bytes.Equal(current, []byte(expected)) || len(legacy) > 0 && bytes.Equal(current, legacy) {
			if err := os.Remove(path); err != nil {
				return err
			}
		} else {
			fmt.Println("Preserved customized legacy view:", path, "(move its UI to frontend/src/auth/)")
		}
	}
	_ = os.Remove(filepath.Join(root, "app/auth/views")) // only removes an empty directory
	return nil
}

func unchangedCurrentAuthRoutes(raw []byte, module string) bool {
	actual, err := format.Source(raw)
	if err != nil {
		return false
	}
	for _, scaffold := range []string{authRoutesScaffold, authAPIRoutesScaffold} {
		expected, err := format.Source([]byte(fmt.Sprintf(scaffold, module)))
		if err == nil && bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
			return true
		}
	}
	return false
}
