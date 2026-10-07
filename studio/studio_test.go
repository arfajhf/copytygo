package studio

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arfajhf/copytygo/v4/core"
)

func TestRegisterStudioLocally(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("COPYTYGO_STUDIO", "true")

	app := core.New()
	app.Get("/ping", func(ctx *core.Context) error { return ctx.Text("pong") })

	if !Register(app) {
		t.Fatal("expected Studio to register in local environment")
	}

	req := httptest.NewRequest(http.MethodGet, DefaultPath, nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "CopyTyGo Studio") {
		t.Fatal("expected Studio dashboard")
	}
}

func TestStudioDisabledInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("COPYTYGO_STUDIO", "true")

	app := core.New()
	if Register(app) {
		t.Fatal("Studio must not register in production")
	}

	req := httptest.NewRequest(http.MethodGet, DefaultPath, nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}


func TestStudioRouteExplorer(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("COPYTYGO_STUDIO", "true")

	app := core.New()
	app.Get("/users/:id", func(ctx *core.Context) error { return ctx.Text("user") }).Name("users.show")
	Register(app)

	htmlRec := httptest.NewRecorder()
	app.ServeHTTP(htmlRec, httptest.NewRequest(http.MethodGet, DefaultPath+"/routes", nil))
	if htmlRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", htmlRec.Code)
	}
	body := htmlRec.Body.String()
	for _, want := range []string{"Route Explorer", "/users/:id", "users.show"} {
		if !strings.Contains(body, want) {
			t.Fatalf("route explorer missing %q", want)
		}
	}

	apiRec := httptest.NewRecorder()
	app.ServeHTTP(apiRec, httptest.NewRequest(http.MethodGet, DefaultPath+"/api/routes", nil))
	if apiRec.Code != http.StatusOK {
		t.Fatalf("expected API 200, got %d", apiRec.Code)
	}
	if !strings.Contains(apiRec.Body.String(), "/users/:id") {
		t.Fatal("route API missing registered route")
	}
}
