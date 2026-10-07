package frontend

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arfajhf/copytygo/v4/core"
)

func TestProductionFrontendAndAssetIsolation(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	dir := t.TempDir()
	t.Setenv("COPYTYGO_FRONTEND_DIST", dir)
	os.MkdirAll(filepath.Join(dir, "assets"), 0755)
	os.WriteFile(filepath.Join(dir, "auth.html"), []byte(`<div id="app"></div><script type="module" src="/assets/auth.js"></script>`), 0644)
	os.WriteFile(filepath.Join(dir, "assets/auth.js"), []byte(`document.title="TypeScript UI"`), 0644)
	os.WriteFile(filepath.Join(dir, "logo.png"), []byte("logo"), 0644)
	t.Setenv("COPYTYGO_VITE_ORIGIN", "http://127.0.0.1:1")
	app := core.New()
	app.Get("/login", Page())
	Assets(app)
	for _, test := range []struct {
		path     string
		status   int
		contains string
	}{{"/login", 200, "assets/auth.js"}, {"/assets/auth.js", 200, "TypeScript UI"}, {"/logo.png", 200, "logo"}, {"/src/auth/main.ts", 404, ""}, {"/assets/../auth.html", 404, ""}, {"/assets/../../.env", 404, ""}, {"/api/missing", 404, ""}} {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest("GET", test.path, nil))
		if rec.Code != test.status || !strings.Contains(rec.Body.String(), test.contains) {
			t.Fatalf("%s: %d %s", test.path, rec.Code, rec.Body.String())
		}
	}
}
func TestDevProxiesTypeScriptAndPageEntry(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	vite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.Path)) }))
	defer vite.Close()
	t.Setenv("COPYTYGO_VITE_ORIGIN", vite.URL)
	app := core.New()
	app.Get("/users/:id/edit", Page())
	Assets(app)
	for path, want := range map[string]string{"/users/7/edit": "/auth.html", "/src/auth/main.ts": "/src/auth/main.ts", "/node_modules/.vite/deps/react.js": "/node_modules/.vite/deps/react.js"} {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 200 || rec.Body.String() != want {
			t.Fatalf("%s proxy: %d %s", path, rec.Code, rec.Body.String())
		}
	}
}
