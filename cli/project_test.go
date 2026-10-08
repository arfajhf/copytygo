package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProjectPresets(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	tests := []struct {
		name     string
		options  ProjectOptions
		frontend bool
		authFile bool
	}{
		{
			name:     "typescript-app",
			options:  DefaultProjectOptions(),
			frontend: true,
		},
		{
			name: "vue-postgres-auth",
			options: ProjectOptions{
				Database: "postgres",
				Auth:     "multi",
				Studio:   true,
				Frontend: "vue",
			},
			frontend: true,
			authFile: true,
		},
		{
			name: "api-only",
			options: ProjectOptions{
				Database: "mysql",
				Auth:     "none",
				Studio:   false,
				Frontend: "api",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := NewProjectWithOptions(tc.name, tc.options); err != nil {
				t.Fatal(err)
			}

			envRaw, err := os.ReadFile(filepath.Join(tc.name, ".env"))
			if err != nil {
				t.Fatal(err)
			}
			env := string(envRaw)
			if strings.Count(env, "QUEUE_DRIVER=") != 1 {
				t.Fatalf("expected one QUEUE_DRIVER, got env:\n%s", env)
			}
			for _, want := range []string{
				"QUEUE_DRIVER=memory",
				"QUEUE_WORKERS=1",
				"QUEUE_POLL_SECONDS=1",
				"QUEUE_BACKOFF_SECONDS=1",
				"STORAGE_PATH=storage/app",
				"MAIL_HOST=127.0.0.1",
			} {
				if !strings.Contains(env, want) {
					t.Fatalf("env missing %q", want)
				}
			}

			mainRaw, err := os.ReadFile(filepath.Join(tc.name, "cmd", "app", "main.go"))
			if err != nil {
				t.Fatal(err)
			}
			main := string(mainRaw)
			for _, want := range []string{
				"foundation.RegisterDefaults(app.Services)",
				"jobs.Register(queue.DefaultRegistry)",
				"queue.AttachConfigured(app)",
				"studio.Register(app)",
			} {
				if !strings.Contains(main, want) {
					t.Fatalf("generated main missing %q", want)
				}
			}

			registryRaw, err := os.ReadFile(filepath.Join(tc.name, "app", "jobs", "registry.go"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(registryRaw), "func Register(registry *queue.Registry) error") {
				t.Fatal("job registry not generated")
			}

			if tc.frontend {
				routes, err := os.ReadFile(filepath.Join(tc.name, "routes/web.go"))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(routes), "frontend.Assets(app)") && tc.options.Auth == "none" {
					t.Fatal("non-auth frontend project lacks public asset serving")
				}
				for _, path := range []string{"frontend/public/logo.png", "frontend/public/images/.gitkeep", "frontend/ASSETS.md"} {
					if _, err := os.Stat(filepath.Join(tc.name, path)); err != nil {
						t.Fatal(err)
					}
				}
			}
			_, frontendErr := os.Stat(filepath.Join(tc.name, "frontend", "package.json"))
			if tc.frontend && frontendErr != nil {
				t.Fatalf("expected frontend package: %v", frontendErr)
			}
			if !tc.frontend && !os.IsNotExist(frontendErr) {
				t.Fatal("API-only project must not include frontend package")
			}

			_, authErr := os.Stat(filepath.Join(tc.name, "app", "auth", "user.go"))
			if tc.authFile && authErr != nil {
				t.Fatalf("expected auth scaffold: %v", authErr)
			}
			if !tc.authFile && !os.IsNotExist(authErr) {
				t.Fatal("auth scaffold should not exist")
			}
		})
	}
}
