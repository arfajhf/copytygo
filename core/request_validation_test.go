package core

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateRejectsMalformedJSONBeforeHandler(t *testing.T) {
	for _, testcase := range []struct{ name, json string }{
		{"truncated", `{"name":`},
		{"array", `[{"name":"a"}]`},
		{"null", `null`},
		{"multiple objects", `{"name":"a"} {"name":"b"}`},
		{"primitive", `42`},
		{"empty", ""},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			app := New()
			called := false
			app.Post("/products", func(ctx *Context) error {
				if err := ctx.Validate(map[string]string{"name": "required"}); err != nil {
					return err
				}
				called = true
				return ctx.NoContent(204)
			})
			req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(testcase.json))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			if rec.Code != 400 || called {
				t.Fatalf("invalid JSON %q: status=%d called=%v body=%s", testcase.json, rec.Code, called, rec.Body.String())
			}
		})
	}
}

func TestValidateCachesJSONAndRetainsIntegerPrecision(t *testing.T) {
	app := New()
	app.Post("/products", func(ctx *Context) error {
		if err := ctx.Validate(map[string]string{"stock": "required|integer", "name": "required", "active": "boolean"}); err != nil {
			return err
		}
		stock, err := ctx.InputWithError("stock")
		if err != nil || stock != "9007199254740993" {
			t.Fatalf("lost precise integer: %q err=%v", stock, err)
		}
		if ctx.Input("name") != "Unicode ✓" || ctx.Input("active") != "true" {
			t.Fatalf("bad input values name=%q active=%q", ctx.Input("name"), ctx.Input("active"))
		}
		if !strings.Contains(ctx.Body(), `"stock":9007199254740993`) {
			t.Fatal("Input consumed request body unexpectedly")
		}
		return ctx.NoContent(204)
	})
	req := httptest.NewRequest("POST", "/products", strings.NewReader(`{"stock":9007199254740993,"name":"Unicode ✓","active":true}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Fatalf("request failed: %d %s", rec.Code, rec.Body.String())
	}
}

func TestValidateOversizedJSONReturns413(t *testing.T) {
	app := New()
	app.Use(BodyLimit(256))
	app.Post("/products", func(ctx *Context) error {
		return ctx.Validate(map[string]string{"name": "required"})
	})
	req := httptest.NewRequest("POST", "/products", strings.NewReader(`{"name":"`+strings.Repeat("a", 1000)+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 413 {
		t.Fatalf("expected 413, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestValidatePreservesFormInput(t *testing.T) {
	app := New()
	app.Post("/products", func(ctx *Context) error {
		if err := ctx.Validate(map[string]string{"name": "required"}); err != nil {
			return err
		}
		if ctx.Input("name") != "Example Item" {
			t.Fatalf("unexpected form field %q", ctx.Input("name"))
		}
		return ctx.NoContent(204)
	})
	req := httptest.NewRequest("POST", "/products", strings.NewReader("name=Example+Item"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Fatalf("form data failed: %d %s", rec.Code, rec.Body.String())
	}
}

func TestValidateReadsMultipartFormValues(t *testing.T) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if err := form.WriteField("name", "Profile Avatar"); err != nil {
		t.Fatal(err)
	}
	upload, err := form.CreateFormFile("avatar", "profile.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := upload.Write([]byte("test-image")); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	app := New()
	app.Post("/upload", func(ctx *Context) error {
		if err := ctx.Validate(map[string]string{"name": "required"}); err != nil {
			return err
		}
		if ctx.Input("name") != "Profile Avatar" {
			t.Fatalf("multipart field lost: %q", ctx.Input("name"))
		}
		header, err := ctx.File("avatar")
		if err != nil || header.Filename != "profile.png" {
			t.Fatalf("multipart file unavailable: %v", err)
		}
		return ctx.NoContent(204)
	})
	req := httptest.NewRequest("POST", "/upload", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Fatalf("multipart request failed: %d %s", rec.Code, rec.Body.String())
	}
}
