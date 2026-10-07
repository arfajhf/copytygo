package core

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextFile(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "avatar.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("copytygo"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	app := New()
	app.Post("/upload", func(ctx *Context) error {
		file, err := ctx.File("avatar")
		if err != nil {
			return err
		}
		if file.Filename != "avatar.txt" {
			t.Fatalf("unexpected filename %q", file.Filename)
		}
		return ctx.NoContent(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
