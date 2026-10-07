package authorization

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arfajhf/copytygo/v4/core"
)

func TestGateMiddleware(t *testing.T) {
	gate := New()
	gate.Define("edit", func(_ *core.Context, subject any) bool {
		return subject == "owner"
	})

	app := core.New()
	app.Get("/ok", func(ctx *core.Context) error { return ctx.Text("ok") }).
		Middleware(gate.Authorize("edit", func(*core.Context) any { return "owner" }))
	app.Get("/no", func(ctx *core.Context) error { return ctx.Text("no") }).
		Middleware(gate.Authorize("edit", func(*core.Context) any { return "other" }))

	okRec := httptest.NewRecorder()
	app.ServeHTTP(okRec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if okRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", okRec.Code)
	}

	noRec := httptest.NewRecorder()
	app.ServeHTTP(noRec, httptest.NewRequest(http.MethodGet, "/no", nil))
	if noRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", noRec.Code)
	}
}
