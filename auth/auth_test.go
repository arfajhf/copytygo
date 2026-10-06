package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arfajhf/copytygo/v3/core"
)

func TestRoutesRegisterAuthEndpoints(t *testing.T) {
	app := core.New()
	Routes(app, "")

	routes := app.Routes()
	if len(routes) != 3 {
		t.Fatalf("expected 3 auth routes, got %d", len(routes))
	}

	expected := []struct {
		method string
		path   string
	}{
		{"POST", "/api/auth/register"},
		{"POST", "/api/auth/login"},
		{"GET", "/api/auth/me"},
	}

	for i, want := range expected {
		if routes[i].Method != want.method || routes[i].Path != want.path {
			t.Fatalf("route %d expected %s %s, got %s %s", i, want.method, want.path, routes[i].Method, routes[i].Path)
		}
	}
}

func TestAuthMiddlewareAndRoleGuard(t *testing.T) {
	t.Setenv("APP_KEY", "copytygo-test-secret-0123456789abcdef")

	token, err := Issue("7", "admin", map[string]string{
		"name": "Test Admin",
	}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	app := core.New()
	app.Get("/admin", func(ctx *core.Context) error {
		return ctx.JSON(core.Map{
			"id":   UserID(ctx),
			"role": Role(ctx),
		})
	}).Middleware(Middleware(), RequireRole("admin"))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRoleGuardRejectsWrongRole(t *testing.T) {
	t.Setenv("APP_KEY", "copytygo-test-secret-0123456789abcdef")

	token, err := Issue("7", "user", nil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	app := core.New()
	app.Get("/admin", func(ctx *core.Context) error {
		return ctx.Text("ok")
	}).Middleware(Middleware(), RequireRole("admin"))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}
