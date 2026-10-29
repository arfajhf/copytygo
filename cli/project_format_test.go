package cli

import (
	"encoding/json"
	"go/format"
	"strings"
	"testing"
)

func TestProjectEnvironmentSectionsStayConsistent(t *testing.T) {
	local := projectEnvironment(
		"authv5", "temporary-test-key", "authv5",
		"mysql", "3306", "ctgdev", "true",
	)
	example := projectEnvironment(
		"CopyTyGo", "", "copytygo",
		"mysql", "3306", "ctgdev", "true",
	)
	for name, env := range map[string]string{
		".env": local,
		".env.example": example,
	} {
		if !strings.HasSuffix(env, "\n") {
			t.Fatalf("%s has no trailing newline", name)
		}
		if strings.Contains(env, "\n\n\n") {
			t.Fatalf("%s contains extra blank lines", name)
		}
		groups := []string{
			"APP_KEY=", "COPYTYGO_STUDIO=", "SERVER_READ_HEADER_TIMEOUT=",
			"DB_DRIVER=", "QUEUE_DRIVER=", "STORAGE_PATH=", "MAIL_HOST=",
		}
		for index := 1; index < len(groups); index++ {
			previous := strings.Index(env, groups[index-1])
			next := strings.Index(env, groups[index])
			if previous < 0 || next <= previous {
				t.Fatalf("%s section order invalid: %q then %q", name, groups[index-1], groups[index])
			}
			if !strings.Contains(env[previous:next], "\n\n") {
				t.Fatalf("%s sections %q / %q not separated", name, groups[index-1], groups[index])
			}
		}
		if strings.Contains(env, "\n\n\n") {
			t.Fatalf("%s has more than one separator line", name)
		}
	}
	localLines := strings.Split(local, "\n")
	exampleLines := strings.Split(example, "\n")
	if len(localLines) != len(exampleLines) {
		t.Fatalf("local and example env layouts differ: %d vs %d", len(localLines), len(exampleLines))
	}
	for i := range localLines {
		if (localLines[i] == "") != (exampleLines[i] == "") {
			t.Fatalf("env layout mismatch on line %d", i+1)
		}
	}
}

func TestFrontendPresetJSONIsReadableAndValid(t *testing.T) {
	for _, preset := range []string{"typescript", "react", "vue"} {
		t.Run(preset, func(t *testing.T) {
			files := frontendFiles(preset)
			for _, name := range []string{"frontend/package.json", "frontend/tsconfig.json"} {
				source := files[name]
				if !strings.Contains(source, "\n") {
					t.Fatalf("%s emitted compressed JSON", name)
				}
				var value map[string]any
				if err := json.Unmarshal([]byte(source), &value); err != nil {
					t.Fatalf("%s invalid JSON: %v", name, err)
				}
			}
		})
	}
}

func TestGoStarterTemplatesAreGofmtCompatible(t *testing.T) {
	for name, source := range map[string]string{
		"main.go": strings.ReplaceAll(projectMain, "{{MODULE}}", "demo"),
		"web.go": projectRoutes,
	} {
		if _, err := format.Source([]byte(source)); err != nil {
			t.Fatalf("%s generated Go invalid: %v", name, err)
		}
	}
}
