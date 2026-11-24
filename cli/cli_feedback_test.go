package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDatabaseConnectionErrorGivesSafeGuidance(t *testing.T) {
	tests := []struct {
		name string
		cause string
		hint string
	}{
		{"server down", "dial tcp 127.0.0.1:5432: connect: connection refused", "Start the database server"},
		{"wrong password", "pq: password authentication failed for user admin", "Check DB_USERNAME, DB_PASSWORD"},
		{"mysql access", "Error 1045: Access denied for user 'root'@'localhost'", "Check DB_USERNAME, DB_PASSWORD"},
		{"unknown database", "Unknown database 'shop'", "Check DB_DATABASE"},
		{"unsupported driver", "copytygo: unsupported database driver \"sqlite\"", "Set DB_DRIVER to mysql or postgres"},
		{"dns", "lookup db.example.test: no such host", "Check DB_HOST"},
		{"timeout", "i/o timeout while connecting", "database server is reachable"},
		{"other", "driver failed using password=hunter2", "Check DB_DRIVER, DB_HOST"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := databaseConnectionError(errors.New(tc.cause))
			if !strings.Contains(err.Error(), tc.hint) {
				t.Fatalf("missing hint %q in %q", tc.hint, err)
			}
			if strings.Contains(err.Error(), tc.cause) {
				t.Fatalf("raw driver error exposed: %q", err)
			}
			if strings.Contains(err.Error(), "hunter2") {
				t.Fatal("error contains connection credential")
			}
			if !strings.HasPrefix(err.Error(), "copytygo: database connection failed") {
				t.Fatalf("missing readable summary: %q", err)
			}
		})
	}
	if databaseConnectionError(nil) != nil {
		t.Fatal("nil error should remain nil")
	}
}

func TestPrintProjectCreatedIsCompact(t *testing.T) {
	var out bytes.Buffer
	printProjectCreated(&out, "store-app", ProjectOptions{
		Database: "postgres",
		Frontend: "typescript",
		Auth: "multi",
		Studio: true,
	})

	result := out.String()
	for _, want := range []string{
		"CopyTyGo | Project created",
		"Name      : store-app",
		"Database  : postgres",
		"Frontend  : typescript",
		"Auth      : multi",
		"Studio    : enabled",
		"Next steps:",
		"cd store-app\n  ctg dev",
	} {
		if !strings.Contains(result, want) {
			t.Errorf("output missing %q:\n%s", want, result)
		}
	}
	if strings.Contains(result, "APP_KEY") || strings.Contains(result, "DB_PASSWORD") {
		t.Fatal("project creation output should not reveal credentials")
	}

	out.Reset()
	printProjectCreated(&out, "api-app", ProjectOptions{Database: "mysql", Frontend: "api", Auth: "none"})
	if !strings.Contains(out.String(), "Studio    : disabled") {
		t.Fatal("disabled Studio status not reflected in output")
	}
}

func TestExistingProjectDirectoryHasActionableHint(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.Mkdir("myapp", 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join("myapp", "existing.txt")
	if err := os.WriteFile(marker, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewProjectWithOptions("myapp", DefaultProjectOptions())
	if err == nil || !strings.Contains(err.Error(), "Hint: choose a different project name") {
		t.Fatalf("expected helpful non-destructive error, got: %v", err)
	}
	content, err := os.ReadFile(marker)
	if err != nil || string(content) != "keep me" {
		t.Fatal("existing application was modified")
	}
}

func TestUnknownCommandIncludesHelpHint(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"ctg", "unknown:action"}

	err := New().Run()
	if err == nil || !strings.Contains(err.Error(), "ctg help") {
		t.Fatalf("unknown command should suggest help, got %v", err)
	}
}
