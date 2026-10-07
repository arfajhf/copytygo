package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestInspectorRecordsRequests(t *testing.T) {
	inspector := NewRequestInspector(2)
	app := New()
	app.Use(RequestID(), inspector.Middleware())
	app.Get("/ping", func(ctx *Context) error { return ctx.Text("pong") })

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	}

	records := inspector.Records()
	if len(records) != 2 {
		t.Fatalf("expected capacity 2, got %d", len(records))
	}
	if records[0].Path != "/ping" || records[0].Status != http.StatusOK {
		t.Fatalf("unexpected latest record %#v", records[0])
	}
	if records[0].RequestID == "" {
		t.Fatal("expected request id")
	}
}
