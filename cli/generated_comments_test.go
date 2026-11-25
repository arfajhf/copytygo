package cli

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedProjectHasConciseFileHeaders(t *testing.T) {
	t.Chdir(t.TempDir())
	options := DefaultProjectOptions()
	options.Auth = "multi"
	if err := NewProjectWithOptions("readable-app", options); err != nil {
		t.Fatal(err)
	}

	root := "readable-app"
	for path, want := range map[string]string{
		".env":                   "# CopyTyGo Application Configuration",
		".env.example":           "# CopyTyGo Application Configuration",
		"cmd/app/main.go":        "// CopyTyGo Application Entry Point",
		"routes/web.go":          "// Application Routes",
		"frontend/src/api.ts":    "// API client for Go backend requests.",
		"frontend/src/forms.ts":  "// Form helpers for submitting",
		"frontend/src/main.ts":   "// Application UI entry point.",
		"frontend/src/auth/api.ts":   "// Session and API helpers",
		"frontend/src/auth/main.ts":  "// Browser authentication pages",
		"frontend/src/auth/views.ts": "// Edit these templates",
		"app/auth/web.go":           "// Authentication configuration:",
	} {
		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(content, []byte(want)) {
			t.Errorf("%s missing expected orientation comment %q", path, want)
		}
		if strings.Count(string(content), "CopyTyGo Application Configuration") > 1 {
			t.Errorf("%s contains duplicate config header", path)
		}
	}

	// Go source still compiles through gofmt after comments are inserted.
	for _, path := range []string{
		"cmd/app/main.go",
		"routes/web.go",
		"app/auth/web.go",
		"app/auth/user.go",
		"routes/auth.go",
	} {
		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := format.Source(content); err != nil {
			t.Fatalf("generated %s is invalid Go: %v", path, err)
		}
	}

	authRoute, err := os.ReadFile(filepath.Join(root, "routes/auth.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(authRoute), "sessions.CSRF()") ||
		!strings.Contains(string(authRoute), "sessions.RequireAdmin()") {
		t.Fatal("generated auth routes lost CSRF or admin protection")
	}
}

func TestScaffoldCommentsPreserveResourceAndJobMarkers(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := NewProject("example"); err != nil {
		t.Fatal(err)
	}
	t.Chdir("example")

	if err := MakeResourceWithOptions("Product", []string{
		"name:string", "price:decimal", "active:boolean",
	}, "default"); err != nil {
		t.Fatal(err)
	}
	if err := MakeJob("SendReceipt", filepath.Join("app", "jobs")); err != nil {
		t.Fatal(err)
	}

	for path, required := range map[string][]string{
		"app/controllers/product_controller.go": {
			"// Application Controller",
			"// copytygo:resource products",
			"// copytygo:resource-access default",
		},
		"app/models/product.go": {
			"// Model fields define the data",
		},
		"app/jobs/sendreceipt.go": {
			"// A job handles background work",
			"// copytygo:job SendReceipt",
		},
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range required {
			if !strings.Contains(string(raw), fragment) {
				t.Errorf("%s missing %q", path, fragment)
			}
		}
		if _, err := format.Source(raw); err != nil {
			t.Fatalf("generated %s is invalid Go: %v", path, err)
		}
	}

	registry, err := os.ReadFile("app/jobs/registry.go")
	if err != nil || !strings.Contains(string(registry), "SendReceipt{}.Handle") {
		t.Fatalf("job registry lost generated job: %v", err)
	}
	resources, err := os.ReadFile("routes/resources.go")
	if err != nil || !strings.Contains(string(resources), "ResourceAccessUnconfigured") {
		t.Fatalf("resource route lost default protection: %v", err)
	}
}

func TestReadableFrontendPresets(t *testing.T) {
	for _, preset := range []string{"typescript", "react", "vue"} {
		t.Run(preset, func(t *testing.T) {
			files := frontendFiles(preset)
			switch preset {
			case "typescript":
				if !strings.Contains(files["frontend/src/main.ts"], "Application UI entry point") {
					t.Fatal("TypeScript entry point missing comment")
				}
			case "react":
				if !strings.Contains(files["frontend/src/main.tsx"], "React entry point") {
					t.Fatal("React entry point missing comment")
				}
			case "vue":
				if !strings.Contains(files["frontend/src/main.ts"], "Vue entry point") {
					t.Fatal("Vue entry point missing comment")
				}
			}
		})
	}
}
