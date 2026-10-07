package cli

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"time"

	copyauth "github.com/arfajhf/copytygo/v4/auth"
)

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
	updateRoutes := os.IsNotExist(err) || legacyAuthRoutes(current)
	if !updateRoutes && !strings.Contains(string(current), "appauth.Pages()") {
		return fmt.Errorf("copytygo: routes/auth.go has custom auth routes; add the browser routes from docs/AUTHENTICATION.md manually")
	}
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
	views, err := copyauth.StarterViews()
	if err != nil {
		return err
	}
	for name, content := range views {
		files["app/auth/views/"+name] = content
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
		source, err := format.Source([]byte(fmt.Sprintf(authRoutesScaffold, module)))
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
	webPath := filepath.Join(root, "routes", "web.go")
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
		source, err := format.Source([]byte(web[:at] + "\n RegisterAuth(app)" + web[at:]))
		if err != nil {
			return err
		}
		if err := os.WriteFile(webPath, source, 0644); err != nil {
			return err
		}
	}
	fmt.Printf("Auth starter installed (%s role).\n", mode)
	fmt.Println("1. Set your database connection in .env, then run: ctg migrate")
	fmt.Println("2. Start the application: ctg dev")
	fmt.Println("3. Open /login or /register, or use the buttons on the welcome page.")
	fmt.Println("Customize: routes/auth.go, app/auth/web.go, app/auth/views/*.html")
	fmt.Println("Guide: app/auth/README.md. Existing starter files are preserved.")
	return nil
}

func writeAuthStarterFile(root, path, content string) error {
	target := filepath.Join(root, filepath.FromSlash(path))
	if _, err := os.Stat(target); err == nil {
		return nil
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

// Middleware protects bearer-token API routes. Browser pages use Pages().Middleware().
func Middleware() core.Middleware { return copyauth.Middleware() }

// RequireRole must run after authentication middleware.
func RequireRole(roles ...string) core.Middleware { return copyauth.RequireRole(roles...) }
`

const authWebScaffold = `package auth

import (
 "embed"
 copyauth "github.com/arfajhf/copytygo/v4/auth"
)

// DefaultRole is assigned by the server during public registration.
// Use an empty role for single-role auth; multi-role auth starts with "user".
const DefaultRole = %q

//go:embed views/*.html
var pageViews embed.FS

// Pages connects your editable HTML views to the browser auth controller.
// Views compile into the application binary. Restart ctg dev after editing them.
func Pages() *copyauth.WebAuth {
 return copyauth.NewWebAuth(copyauth.WebOptions{
  DefaultRole: DefaultRole,
  Views: pageViews,
 })
}
`

const authRoutesScaffold = `package routes

import (
 appauth "%s/app/auth"
 copyauth "github.com/arfajhf/copytygo/v4/auth"
 "github.com/arfajhf/copytygo/v4/core"
)

// RegisterAuth defines browser pages and JSON endpoints in one place.
// Edit these routes to customize your application.
func RegisterAuth(app *core.Application) {
 pages := appauth.Pages()

 // Browser: HTML forms, encrypted cookie sessions, and CSRF protection.
 app.Get("/login", pages.LoginPage).Name("auth.login")
 app.Post("/login", pages.Login)
 app.Get("/register", pages.RegisterPage).Name("auth.register")
 app.Post("/register", pages.Register)
 app.Get("/account", pages.Account).Middleware(pages.Middleware()).Name("auth.account")
 app.Post("/logout", pages.Logout).Middleware(pages.Middleware())

 // API: JSON responses and Authorization: Bearer <token>.
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

const authScaffoldReadme = `# Your authentication starter

Start here:

1. Configure your MySQL or PostgreSQL connection in .env.
2. Run ctg migrate to create the users table.
3. Run ctg dev, then open the welcome page, /login, or /register.
4. Registration signs you in automatically and redirects to /account.

Files you can edit:

- routes/auth.go: browser and API route definitions.
- app/auth/web.go: default registration role and HTML view binding.
- app/auth/views/layout.html: shared layout, responsive CSS, and form helpers.
- app/auth/views/login.html: login form.
- app/auth/views/register.html: registration form and password confirmation.
- app/auth/views/account.html: signed-in page and logout form.
- app/auth/user.go: user structure.
- app/auth/middleware.go: API authentication and role helpers.

Restart ctg dev after editing Go code or embedded HTML views. Production builds
include the views and do not require Node.js for these pages.

Browser pages use encrypted HttpOnly cookies. Every POST form includes a CSRF
field. Keep that field when changing templates. API endpoints still use bearer
tokens and do not accept the browser cookie as API authentication.

Public multi-role registration assigns DefaultRole on the server. It never reads
a role field submitted by the user. Assign privileged roles through trusted
administrative code.

Running install:auth again preserves existing starter files and custom routes.
An unchanged older CopyTyGo API-only routes file is upgraded automatically.
`
