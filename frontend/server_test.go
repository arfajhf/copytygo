package frontend

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestDevWebSocketUpgradeThroughRequestMiddleware(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, buffer, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		fmt.Fprint(buffer, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		buffer.Flush()
		input, err := buffer.ReadString('\n')
		if err != nil {
			return
		}
		fmt.Fprint(buffer, "echo:"+input)
		buffer.Flush()
	}))
	defer upstream.Close()
	t.Setenv("COPYTYGO_VITE_ORIGIN", upstream.URL)
	app := core.New()
	app.Use(core.RequestLogger(), core.NewRequestInspector(10).Middleware())
	app.Get("/", Page())
	server := httptest.NewServer(app)
	defer server.Close()
	conn, err := net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n", strings.TrimPrefix(server.URL, "http://"))
	buffer := bufio.NewReader(conn)
	response, err := http.ReadResponse(buffer, nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 101 {
		t.Fatalf("HMR upgrade blocked: %d", response.StatusCode)
	}
	fmt.Fprint(conn, "hot reload\n")
	received, err := buffer.ReadString('\n')
	if err != nil || received != "echo:hot reload\n" {
		t.Fatalf("upgrade tunnel failed: %q %v", received, err)
	}
}
