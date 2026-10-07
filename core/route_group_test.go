package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type testResourceController struct{}

func (testResourceController) Index(ctx *Context) error   { return ctx.Text("index") }
func (testResourceController) Show(ctx *Context) error    { return ctx.Text("show") }
func (testResourceController) Store(ctx *Context) error   { return ctx.Text("store") }
func (testResourceController) Update(ctx *Context) error  { return ctx.Text("update") }
func (testResourceController) Destroy(ctx *Context) error { return ctx.Text("destroy") }

func TestNestedRouteGroupsAndNames(t *testing.T) {
	app := New()

	api := app.APIVersion("v1")
	admin := api.Group("/admin").Name("admin.")
	admin.Get("/users/:id", func(ctx *Context) error {
		return ctx.Text(ctx.Param("id"))
	}).Name("users.show")

	url, ok := app.URL("api.v1.admin.users.show", map[string]string{"id":"42"})
	if !ok || url != "/api/v1/admin/users/42" {
		t.Fatalf("unexpected named URL %q ok=%v", url, ok)
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "42" {
		t.Fatalf("unexpected response %d %q", rec.Code, rec.Body.String())
	}
}

func TestGroupedResourceRoutes(t *testing.T) {
	app := New()
	app.APIVersion("v2").Resource("/products", testResourceController{})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v2/products/7", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != "show" {
		t.Fatalf("unexpected response %d %q", rec.Code, rec.Body.String())
	}
}
