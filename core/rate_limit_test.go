package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitHeadersAndBlock(t *testing.T) {
	app := New()
	app.Get("/limited", func(ctx *Context) error {
		return ctx.Text("ok")
	}).Middleware(RateLimitWith(RateLimitOptions{
		Max:    2,
		Window: time.Minute,
		Key: func(*Context) string {
			return "test-client"
		},
	}))

	for i := 1; i <= 3; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/limited", nil))

		if rec.Header().Get("X-RateLimit-Limit") != "2" {
			t.Fatal("missing rate limit header")
		}

		if i < 3 && rec.Code != http.StatusOK {
			t.Fatalf("request %d expected 200, got %d", i, rec.Code)
		}
		if i == 3 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d expected 429, got %d", i, rec.Code)
		}
	}
}
