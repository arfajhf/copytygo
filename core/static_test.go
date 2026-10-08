package core

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStaticFilesFollowRoutesAndApplyMiddleware(t *testing.T) {
	directory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, "images"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(directory, "images", "banner.txt"), []byte("public banner"), 0644)
	os.WriteFile(filepath.Join(directory, "health"), []byte("static should not win"), 0644)
	os.WriteFile(filepath.Join(directory, ".env"), []byte("private value"), 0644)
	os.WriteFile(filepath.Join(directory, "write-only"), []byte("static should not win"), 0644)
	app := New()
	app.Use(RequestID())
	app.Static("/", directory)
	app.Get("/health", func(ctx *Context) error { return ctx.Text("backend health") })
	app.Post("/write-only", func(ctx *Context) error { return ctx.Text("backend write") })
	for _, test := range []struct {
		method, path string
		status       int
		body         string
	}{
		{"GET", "/images/banner.txt", 200, "public banner"},
		{"HEAD", "/images/banner.txt", 200, ""},
		{"GET", "/health", 200, "backend health"},
		{"GET", "/write-only", 405, ""},
		{"POST", "/images/banner.txt", 404, ""},
		{"GET", "/images", 404, ""},
		{"GET", "/.env", 404, ""},
		{"GET", "/images/../.env", 404, ""},
		{"GET", "/images/%2e%2e/.env", 404, ""},
		{"GET", "/images%5c..%5c.env", 404, ""},
		{"GET", "/missing.png", 404, ""},
	} {
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
		if response.Code != test.status {
			t.Fatalf("%s %s: %d %s", test.method, test.path, response.Code, response.Body.String())
		}
		if test.body != "" && response.Body.String() != test.body {
			t.Fatalf("%s: %q", test.path, response.Body.String())
		}
		if test.method == "HEAD" && response.Body.Len() != 0 {
			t.Fatal("HEAD included body")
		}
		if test.status == 200 && response.Header().Get("X-Request-ID") == "" {
			t.Fatal("public files bypassed middleware")
		}
	}
	// Files added while ctg dev is running do not need a route or server restart.
	os.WriteFile(filepath.Join(directory, "new.txt"), []byte("added later"), 0644)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest("GET", "/new.txt", nil))
	if response.Code != 200 || response.Body.String() != "added later" {
		t.Fatal("new public file unavailable")
	}
}
func TestStaticPrefixExclusionsAndSymlinkIsolation(t *testing.T) {
	directory := t.TempDir()
	outside := t.TempDir()
	os.MkdirAll(filepath.Join(directory, "reserved"), 0755)
	os.WriteFile(filepath.Join(directory, "logo.txt"), []byte("logo"), 0644)
	os.WriteFile(filepath.Join(directory, "reserved", "secret.txt"), []byte("reserved"), 0644)
	os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("outside"), 0644)
	app := New()
	app.Static("/public", directory, StaticOptions{Exclude: []string{"/public/reserved"}})
	for _, test := range []struct {
		path   string
		status int
	}{{"/public/logo.txt", 200}, {"/logo.txt", 404}, {"/public/reserved/secret.txt", 404}, {"/public/../secret.txt", 404}} {
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest("GET", test.path, nil))
		if response.Code != test.status {
			t.Fatalf("%s: %d", test.path, response.Code)
		}
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(directory, "linked.txt")); err != nil {
		t.Skip("symlinks unavailable")
	}
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest("GET", "/public/linked.txt", nil))
	if response.Code != 404 {
		t.Fatal("public symlink exposed file outside root")
	}
}
