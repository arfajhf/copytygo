package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONNotFoundAndMethodNotAllowed(t *testing.T) {
	app := New()
	app.Get("/users", func(ctx *Context) error {
		return ctx.JSON(Map{"ok": true})
	})

	notFound := httptest.NewRecorder()
	app.ServeHTTP(notFound, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if notFound.Code != http.StatusNotFound || !strings.Contains(notFound.Body.String(), ""status":404") {
		t.Fatalf("unexpected 404 response: %d %s", notFound.Code, notFound.Body.String())
	}

	notAllowed := httptest.NewRecorder()
	app.ServeHTTP(notAllowed, httptest.NewRequest(http.MethodPost, "/users", nil))
	if notAllowed.Code != http.StatusMethodNotAllowed || !strings.Contains(notAllowed.Body.String(), ""status":405") {
		t.Fatalf("unexpected 405 response: %d %s", notAllowed.Code, notAllowed.Body.String())
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	app := New()
	app.Use(RequestID())
	app.Get("/request", func(ctx *Context) error {
		return ctx.JSON(Map{"request_id": RequestIDValue(ctx)})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/request", nil)
	app.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("expected X-Request-ID response header")
	}
	if !strings.Contains(rec.Body.String(), id) {
		t.Fatalf("response body does not contain request id: %s", rec.Body.String())
	}
}

func TestRequestIDPreservesValidIncomingID(t *testing.T) {
	app := New()
	app.Use(RequestID())
	app.Get("/request", func(ctx *Context) error {
		return ctx.NoContent(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/request", nil)
	req.Header.Set("X-Request-ID", "edge-request-123")
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != "edge-request-123" {
		t.Fatalf("unexpected request id %q", rec.Header().Get("X-Request-ID"))
	}
}
