package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLiteStudioDashboard(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("COPYTYGO_STUDIO", "true")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/__copytygo", nil)

	if !handleLiteStudio(rec, req) {
		t.Fatal("expected Lite Studio request to be handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"CopyTyGo", "Lite Runtime", "Dashboard"} {
		if !strings.Contains(body, want) {
			t.Fatalf("Lite Studio dashboard missing %q", want)
		}
	}
}

func TestLiteStudioDisabledInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("COPYTYGO_STUDIO", "true")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/__copytygo", nil)

	if !handleLiteStudio(rec, req) {
		t.Fatal("expected Studio prefix to be handled")
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestLiteWelcomeLinksStudio(t *testing.T) {
	t.Setenv("APP_NAME", "Lite Demo")
	page := liteWelcomePage()

	for _, want := range []string{"Lite Demo", "/__copytygo", "Documentation"} {
		if !strings.Contains(page, want) {
			t.Fatalf("Lite welcome missing %q", want)
		}
	}
}
