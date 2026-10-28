package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type securityTestResource struct{ hits *int }

func (r securityTestResource) call(ctx *Context) error {
	*r.hits++
	return ctx.JSON(Map{"ok": true})
}
func (r securityTestResource) Index(ctx *Context) error   { return r.call(ctx) }
func (r securityTestResource) Show(ctx *Context) error    { return r.call(ctx) }
func (r securityTestResource) Store(ctx *Context) error   { return r.call(ctx) }
func (r securityTestResource) Update(ctx *Context) error  { return r.call(ctx) }
func (r securityTestResource) Destroy(ctx *Context) error { return r.call(ctx) }

func TestResourceWithMiddlewareSecuresEveryVerb(t *testing.T) {
	app := New()
	hits := 0
	auth := func(next Handler) Handler {
		return func(ctx *Context) error {
			if ctx.Request.Header.Get("Authorization") != "Bearer valid" {
				return NewHTTPError(http.StatusUnauthorized, "Authentication required")
			}
			if ctx.Request.Header.Get("X-Role") != "admin" {
				return NewHTTPError(http.StatusForbidden, "Insufficient role")
			}
			return next(ctx)
		}
	}
	csrf := func(next Handler) Handler {
		return func(ctx *Context) error {
			if ctx.Request.Header.Get("X-CSRF-Token") != "valid" {
				return NewHTTPError(http.StatusForbidden, "CSRF required")
			}
			return next(ctx)
		}
	}
	app.ResourceWithMiddleware("/products", securityTestResource{hits: &hits}, []Middleware{auth}, []Middleware{csrf})

	for _, tc := range []struct{ method, path string }{
		{"GET", "/products"},
		{"GET", "/products/5"},
		{"POST", "/products"},
		{"PUT", "/products/5"},
		{"DELETE", "/products/5"},
	} {
		request := httptest.NewRequest(tc.method, tc.path, nil)
		response := httptest.NewRecorder()
		app.ServeHTTP(response, request)
		if response.Code != 401 || hits != 0 {
			t.Fatalf("%s %s guest: status=%d hits=%d", tc.method, tc.path, response.Code, hits)
		}
		request.Header.Set("Authorization", "Bearer valid")
		response = httptest.NewRecorder()
		app.ServeHTTP(response, request)
		if response.Code != 403 || hits != 0 {
			t.Fatalf("%s %s non-admin: status=%d hits=%d", tc.method, tc.path, response.Code, hits)
		}
		request.Header.Set("X-Role", "admin")
		response = httptest.NewRecorder()
		app.ServeHTTP(response, request)
		if tc.method != "GET" && (response.Code != 403 || hits != 0) {
			t.Fatalf("%s %s missing CSRF: status=%d hits=%d", tc.method, tc.path, response.Code, hits)
		}
		if tc.method != "GET" {
			request.Header.Set("X-CSRF-Token", "valid")
			response = httptest.NewRecorder()
			app.ServeHTTP(response, request)
		}
		if response.Code != 200 {
			t.Fatalf("%s %s authorized: status=%d body=%s", tc.method, tc.path, response.Code, response.Body.String())
		}
	}
	if hits != 5 {
		t.Fatalf("expected five authorized handlers, got %d", hits)
	}
}

func TestResourceUnconfiguredIsFailClosed(t *testing.T) {
	app := New()
	hits := 0
	app.ResourceWithMiddleware("/locked", securityTestResource{hits: &hits}, []Middleware{ResourceAccessUnconfigured()}, nil)
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		path := "/locked"
		if method == "PUT" || method == "DELETE" {
			path += "/1"
		}
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest(method, path, nil))
		if response.Code != 401 || hits != 0 {
			t.Fatalf("%s %s: status=%d hits=%d", method, path, response.Code, hits)
		}
	}
}
