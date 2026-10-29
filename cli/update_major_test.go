package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasLegacyV4Requirement(t *testing.T) {
	tests := []struct{
		name, input string
		legacy bool
	}{
		{"single require", "module app\nrequire github.com/arfajhf/copytygo/v4 v4.0.6\n", true},
		{"block require", "module app\nrequire (\n github.com/arfajhf/copytygo/v4 v4.0.6\n)\n", true},
		{"legacy replace", "replace github.com/arfajhf/copytygo/v4 => ../copytygo\n", true},
		{"v5 only", "require github.com/arfajhf/copytygo/v5 v5.0.0\n", false},
		{"commented v4", "// require github.com/arfajhf/copytygo/v4 v4.0.6\n", false},
		{"v4 text in comments", "module app // github.com/arfajhf/copytygo/v4\n", false},
		{"other module", "require example.com/copytygo/v4 v4.0.6\n", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasLegacyV4Requirement(test.input); got != test.legacy {
				t.Fatalf("unexpected detection: %v", got)
			}
		})
	}
}

func TestUpdateRejectsLegacyProjectBeforeInstall(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	if err := os.Chdir(dir); err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = os.Chdir(old) })
	mod := "module sample\n\nrequire github.com/arfajhf/copytygo/v4 v4.0.6\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0644); err != nil {
		t.Fatal(err)
	}
	err = Update([]string{"v5.0.0"})
	if err == nil || !strings.Contains(err.Error(), "major-version migration") {
		t.Fatalf("expected explicit migration refusal, got: %v", err)
	}
	raw, err := os.ReadFile("go.mod")
	if err != nil || string(raw) != mod {
		t.Fatalf("update changed legacy project: %v %q", err, raw)
	}
}
