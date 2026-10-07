package testing

import (
	"net/http"
	stdtesting "testing"

	"github.com/arfajhf/copytygo/v4/core"
)

func TestHTTPClient(t *stdtesting.T) {
	app := core.New()
	app.Post("/echo", func(ctx *core.Context) error {
		return ctx.JSON(core.Map{"name": ctx.Input("name")})
	})

	client := NewClient(app)
	response, err := client.PostJSON("/echo", map[string]any{"name":"CopyTyGo"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != http.StatusOK || !response.OK() {
		t.Fatalf("unexpected status %d", response.Status)
	}

	var body map[string]any
	if err := response.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["name"] != "CopyTyGo" {
		t.Fatalf("unexpected body %#v", body)
	}
}
