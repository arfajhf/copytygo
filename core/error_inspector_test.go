package core

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorInspectorCapturesHandlerError(t *testing.T) {
	inspector := NewErrorInspector(10)
	app := New()
	app.Use(RequestID(), inspector.Middleware())
	app.Get("/fail", func(*Context) error {
		return errors.New("boom")
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fail", nil))

	records := inspector.Records()
	if len(records) != 1 {
		t.Fatalf("expected one error, got %d", len(records))
	}
	if records[0].Message != "boom" || records[0].Path != "/fail" {
		t.Fatalf("unexpected error record %#v", records[0])
	}
	if records[0].RequestID == "" {
		t.Fatal("expected request id")
	}
}
