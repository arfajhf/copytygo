package cli

import (
    "net/http/httptest"
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
    app.Get("/users/:id", func(ctx *core.Context) error {
        return ctx.JSON(core.Map{"id":ctx.Param("id"),"search":ctx.Query("q")})
    })
    app.Post("/created", func(ctx *core.Context) error {
        return ctx.Status(201).Text("created")
    })
}`
    if err := os.WriteFile(filepath.Join(dir, "web.go"), []byte(source), 0644); err != nil {
        t.Fatal(err)
    }
    routes, err := discoverLiteRoutes(dir)
    if err != nil { t.Fatal(err) }
    if len(routes) != 4 { t.Fatalf("expected 4 routes, got %d", len(routes)) }
    if routes[0].Path != "/api/health" { t.Fatalf("unexpected path %s", routes[0].Path) }
    if routes[0].Body != `{"framework":"CopyTyGo","status":"ok","ready":true}`+"\n" { t.Fatalf("unexpected body %q", routes[0].Body) }
    if routes[1].Body != "hello" { t.Fatalf("unexpected text %q", routes[1].Body) }
    if routes[3].Status != 201 { t.Fatalf("expected 201, got %d", routes[3].Status) }
}

func TestLiteParamsAndQuery(t *testing.T) {
    params, ok := matchLitePath("/users/:id", "/users/42")
    if !ok || params["id"] != "42" {
        t.Fatalf("unexpected params: %#v matched=%v", params, ok)
    }
    req := httptest.NewRequest("GET", "/users/42?q=book", nil)
    body := renderLiteBody(`{"id":"{{param:id}}","search":"{{query:q}}"}`, req, params)
    expected := `{"id":"42","search":"book"}`
    if body != expected {
        t.Fatalf("expected %s, got %s", expected, body)
    }
}
