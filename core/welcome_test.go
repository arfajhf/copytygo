package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWelcomePage(t *testing.T) {
	t.Setenv("APP_NAME", "Demo App")
	t.Setenv("APP_ENV", "local")

	app := New()
	app.Get("/", Welcome())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected html content type, got %q", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	for _, want := range []string{"Demo App", "Documentation", "Open Studio", "CopyTyGo"} {
		if !strings.Contains(body, want) {
			t.Fatalf("welcome page missing %q", want)
		}
	}
}

func TestWelcomeHidesStudioInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	app := New()
	app.Get("/", Welcome())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if strings.Contains(rec.Body.String(), "Open Studio") {
		t.Fatal("welcome page must not expose Studio in production")
	}
}

func TestWelcomeOnlyLinksInstalledAuthPages(t *testing.T) {
	for _, installed := range []bool{false, true} {
		app := New()
		app.Get("/", Welcome())
		if installed {
			app.Get("/login", func(ctx *Context) error { return ctx.Text("login") })
			app.Get("/register", func(ctx *Context) error { return ctx.Text("register") })
		}
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		for _, path := range []string{"/login", "/register"} {
			if strings.Contains(rec.Body.String(), `href="`+path+`"`) != installed {
				t.Fatalf("auth link %s has wrong visibility", path)
			}
		}
	}
}

func TestWelcomeNavbarHasProjectIdentityAndUsefulLinks(t *testing.T) {
	t.Setenv("APP_NAME", "My Store")
	t.Setenv("APP_ENV", "local")
	t.Setenv("COPYTYGO_STUDIO", "true")

	app := New()
	app.Get("/", Welcome())
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	page := rec.Body.String()
	for _, want := range []string{
		`aria-label="My Store home"`,
		`class="brand-name">My Store</span>`,
		"Powered by CopyTyGo",
		`aria-label="Main navigation"`,
		`class="nav-link" href="`,
		">Docs</a>",
		">GitHub</a>",
		`class="nav-action" href="/__copytygo">Open Studio</a>`,
		"@media(max-width:520px)",
		".navbar a:focus-visible",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("welcome navigation missing %q", want)
		}
	}
	if strings.Contains(page, `href="/login"`) || strings.Contains(page, `href="/register"`) {
		t.Fatal("welcome navigation linked to authentication routes that are not installed")
	}
}

func TestWelcomeNavbarUsesInstalledAuthRoutes(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	app := New()
	app.Get("/", Welcome())
	app.Get("/login", func(ctx *Context) error { return ctx.Text("login") })
	app.Get("/register", func(ctx *Context) error { return ctx.Text("register") })

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	page := rec.Body.String()

	for _, want := range []string{
		`class="nav-link" href="/login">Log in</a>`,
		`class="nav-action nav-action-primary" href="/register">Register</a>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("missing installed auth navigation: %q", want)
		}
	}
}

func TestWelcomeHidesStudioWhenDisabled(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("COPYTYGO_STUDIO", "false")

	app := New()
	app.Get("/", Welcome())
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	page := rec.Body.String()
	if strings.Contains(page, `href="/__copytygo"`) || strings.Contains(page, "Open Studio") {
		t.Fatal("disabled Studio must not be linked from the welcome page")
	}
}

func TestWelcomeEscapesApplicationNameInNavbar(t *testing.T) {
	t.Setenv("APP_NAME", `<script>alert("unsafe")</script>`)

	app := New()
	app.Get("/", Welcome())
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if strings.Contains(rec.Body.String(), `<script>alert("unsafe")</script>`) {
		t.Fatal("navbar rendered unescaped application name")
	}
	if !strings.Contains(rec.Body.String(), "&lt;script&gt;") {
		t.Fatal("escaped project name missing")
	}
}
