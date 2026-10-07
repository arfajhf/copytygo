package studio

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arfajhf/copytygo/v3/core"
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
