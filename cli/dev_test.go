package cli

import (
    "os"
    "path/filepath"
    "testing"
)

func TestDiscoverLiteRoutes(t *testing.T) {
    dir := t.TempDir()
    source := `package routes
import "github.com/arfajhf/copytygo/core"
func Register(app *core.Application) {
    app.Get("/api/health", func(ctx *core.Context) error {
        return ctx.JSON(core.Map{"framework":"CopyTyGo","status":"ok","ready":true})
    })
    app.Get("/hello", func(ctx *core.Context) error { return ctx.Text("hello") })
}`
    if err := os.WriteFile(filepath.Join(dir, "web.go"), []byte(source), 0644); err != nil {
        t.Fatal(err)
    }
    routes, err := discoverLiteRoutes(dir)
    if err != nil { t.Fatal(err) }
    if len(routes) != 2 { t.Fatalf("expected 2 routes, got %d", len(routes)) }
    if routes[0].Path != "/api/health" { t.Fatalf("unexpected path %s", routes[0].Path) }
    if routes[0].Body != `{"framework":"CopyTyGo","status":"ok","ready":true}`+"\n" { t.Fatalf("unexpected body %q", routes[0].Body) }
    if routes[1].Body != "hello" { t.Fatalf("unexpected text %q", routes[1].Body) }
}
