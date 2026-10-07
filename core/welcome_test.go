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
