package cli

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiteStudioScaffolds(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("COPYTYGO_STUDIO", "true")
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	for _, tc := range []struct{ kind, path string }{
		{"model", "app/models/sample.go"},
		{"middleware", "app/middleware/sample.go"},
		{"service", "app/services/sample.go"},
		{"job", "app/jobs/sample.go"},
		{"listener", "app/listeners/sample.go"},
		{"seeder", "database/seeders/sample.go"},
		{"factory", "database/factories/sample.go"},
		{"mail", "app/mails/sample.go"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			form := url.Values{"type": {tc.kind}, "name": {"Sample"}}
			req := httptest.NewRequest(http.MethodPost, "/__copytygo/generator/scaffold", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			if !handleLiteStudio(rec, req) {
				t.Fatal("scaffold route not handled")
			}
			if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "created=") {
				t.Fatalf("scaffold failed: status=%d location=%s", rec.Code, rec.Header().Get("Location"))
			}
			if _, err := os.Stat(filepath.Join(root, tc.path)); err != nil {
				t.Fatal(err)
			}
		})
	}
	raw, err := os.ReadFile(filepath.Join(root, "app/jobs/registry.go"))
	if err != nil || !strings.Contains(string(raw), "Sample{}.Handle") {
		t.Fatalf("generated job not registered: %v", err)
	}
	for _, kind := range []string{"unknown", "job"} {
		form := url.Values{"type": {kind}, "name": {"../Outside"}}
		req := httptest.NewRequest(http.MethodPost, "/__copytygo/generator/scaffold", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		handleLiteStudio(rec, req)
		if !strings.Contains(rec.Header().Get("Location"), "error=") {
			t.Fatal("invalid scaffold accepted")
		}
	}
	t.Setenv("APP_ENV", "production")
	req := httptest.NewRequest(http.MethodPost, "/__copytygo/generator/scaffold", strings.NewReader("type=job&name=Blocked"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handleLiteStudio(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("production scaffold status=%d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(root, "app/jobs/blocked.go")); !os.IsNotExist(err) {
		t.Fatal("production wrote a scaffold")
	}
}

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

func TestLiteStudioInspectorPages(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("COPYTYGO_STUDIO", "true")
	t.Setenv("APP_KEY", "12345678901234567890123456789012")

	for _, path := range []string{
		"/__copytygo/models",
		"/__copytygo/auth",
		"/__copytygo/services",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if !handleLiteStudio(rec, req) {
			t.Fatalf("expected %s to be handled", path)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Lite Runtime") {
			t.Fatalf("%s did not render Lite Studio", path)
		}
	}
}
