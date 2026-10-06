package cli

import (
    "net/http/httptest"
    "net/http"
    "strings"
    "os"
    "path/filepath"
    "testing"
)

func TestDiscoverLiteRoutes(t *testing.T) {
    dir := t.TempDir()
    source := `package routes
import "github.com/arfajhf/copytygo/v2/core"
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


func TestLiteControllerHandler(t *testing.T) {
    dir := t.TempDir()
    source := `package controllers
import "github.com/arfajhf/copytygo/v2/core"

type UserController struct{}

func (UserController) Show(ctx *core.Context) error {
    return ctx.JSON(core.Map{
        "id": ctx.Param("id"),
        "search": ctx.Query("q"),
    })
}
`
    if err := os.WriteFile(filepath.Join(dir, "user_controller.go"), []byte(source), 0644); err != nil {
        t.Fatal(err)
    }
    route, ok, err := parseLiteControllerHandler(dir, "UserController", "Show", "GET", "/users/:id")
    if err != nil {
        t.Fatal(err)
    }
    if !ok {
        t.Fatal("expected controller handler to be supported")
    }
    req := httptest.NewRequest("GET", "/users/9?q=book", nil)
    params, matched := matchLitePath(route.Path, "/users/9")
    if !matched {
        t.Fatal("expected controller route to match")
    }
    body := renderLiteBody(route.Body, req, params)
    expected := `{"id":"9","search":"book"}` + "\n"
    if body != expected {
        t.Fatalf("expected %q, got %q", expected, body)
    }
}


func TestLiteInputAndValidation(t *testing.T) {
    source := `package controllers
import "github.com/arfajhf/copytygo/v2/core"

type UserController struct{}

func (UserController) Store(ctx *core.Context) error {
    if err := ctx.Validate(map[string]string{
        "name": "required|min:3",
        "email": "required|email",
    }); err != nil {
        return err
    }

    return ctx.Status(201).JSON(core.Map{
        "name": ctx.Input("name"),
        "email": ctx.Input("email"),
    })
}
`

    dir := t.TempDir()
    if err := os.WriteFile(filepath.Join(dir, "user_controller.go"), []byte(source), 0644); err != nil {
        t.Fatal(err)
    }

    route, ok, err := parseLiteControllerHandler(dir, "UserController", "Store", "POST", "/users")
    if err != nil {
        t.Fatal(err)
    }
    if !ok {
        t.Fatal("expected POST controller to be supported")
    }
    if route.Status != 201 {
        t.Fatalf("expected 201, got %d", route.Status)
    }
    if route.Validation["name"] != "required|min:3" {
        t.Fatalf("missing validation rules: %#v", route.Validation)
    }

    req := httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"Fajar","email":"fajar@example.com"}`))
    req.Header.Set("Content-Type", "application/json")

    if fields := liteValidateRequest(req, route.Validation); len(fields) != 0 {
        t.Fatalf("unexpected validation errors: %#v", fields)
    }

    body := renderLiteBody(route.Body, req, nil)
    expected := `{"name":"Fajar","email":"fajar@example.com"}` + "\n"
    if body != expected {
        t.Fatalf("expected %q, got %q", expected, body)
    }

    bad := httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"A","email":"wrong"}`))
    bad.Header.Set("Content-Type", "application/json")
    fields := liteValidateRequest(bad, route.Validation)
    if len(fields["name"]) == 0 || len(fields["email"]) == 0 {
        t.Fatalf("expected name and email errors, got %#v", fields)
    }
}


func TestLiteMemoryCRUD(t *testing.T) {
    liteMemoryStore.Lock()
    liteMemoryStore.Buckets = make(map[string]*liteMemoryBucket)
    liteMemoryStore.Unlock()

    store := liteRoute{
        Method: "POST",
        Path: "/products",
        ResourceAction: "store",
        Resource: "products",
        ResourceData: `{"name":"{{input:name}}"}`,
    }

    createReq := httptest.NewRequest("POST", "/products", strings.NewReader(`{"name":"Keyboard"}`))
    createReq.Header.Set("Content-Type", "application/json")
    createRec := httptest.NewRecorder()
    handleLiteMemoryRoute(createRec, createReq, store, nil)

    if createRec.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d body=%s", createRec.Code, createRec.Body.String())
    }

    indexRec := httptest.NewRecorder()
    handleLiteMemoryRoute(indexRec, httptest.NewRequest("GET", "/products", nil), liteRoute{
        ResourceAction: "index",
        Resource: "products",
    }, nil)
    if indexRec.Code != http.StatusOK || !strings.Contains(indexRec.Body.String(), "Keyboard") {
        t.Fatalf("unexpected index response: %d %s", indexRec.Code, indexRec.Body.String())
    }

    showRec := httptest.NewRecorder()
    handleLiteMemoryRoute(showRec, httptest.NewRequest("GET", "/products/1", nil), liteRoute{
        ResourceAction: "show",
        Resource: "products",
        ResourceID: "{{param:id}}",
    }, map[string]string{"id":"1"})
    if showRec.Code != http.StatusOK || !strings.Contains(showRec.Body.String(), "Keyboard") {
        t.Fatalf("unexpected show response: %d %s", showRec.Code, showRec.Body.String())
    }

    updateReq := httptest.NewRequest("PUT", "/products/1", strings.NewReader(`{"name":"Mouse"}`))
    updateReq.Header.Set("Content-Type", "application/json")
    updateRec := httptest.NewRecorder()
    handleLiteMemoryRoute(updateRec, updateReq, liteRoute{
        ResourceAction: "update",
        Resource: "products",
        ResourceID: "{{param:id}}",
        ResourceData: `{"name":"{{input:name}}"}`,
    }, map[string]string{"id":"1"})
    if updateRec.Code != http.StatusOK || !strings.Contains(updateRec.Body.String(), "Mouse") {
        t.Fatalf("unexpected update response: %d %s", updateRec.Code, updateRec.Body.String())
    }

    deleteRec := httptest.NewRecorder()
    handleLiteMemoryRoute(deleteRec, httptest.NewRequest("DELETE", "/products/1", nil), liteRoute{
        ResourceAction: "destroy",
        Resource: "products",
        ResourceID: "{{param:id}}",
    }, map[string]string{"id":"1"})
    if deleteRec.Code != http.StatusNoContent {
        t.Fatalf("expected 204, got %d", deleteRec.Code)
    }

    missingRec := httptest.NewRecorder()
    handleLiteMemoryRoute(missingRec, httptest.NewRequest("GET", "/products/1", nil), liteRoute{
        ResourceAction: "show",
        Resource: "products",
        ResourceID: "{{param:id}}",
    }, map[string]string{"id":"1"})
    if missingRec.Code != http.StatusNotFound {
        t.Fatalf("expected 404 after delete, got %d", missingRec.Code)
    }
}
