package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FormatEnvironmentFiles adds visual separators without changing any key,
// value or comment in the current project's existing dotenv files.
func FormatEnvironmentFiles() error {
	found := false

	for _, path := range []string{".env", ".env.example"} {
		changed, err := formatEnvironmentFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("copytygo: format %s: %w", path, err)
		}
		found = true
		if changed {
			fmt.Printf("Formatted %s\n", path)
		} else {
			fmt.Printf("Already formatted %s\n", path)
		}
	}

	if !found {
		return fmt.Errorf("copytygo: no .env or .env.example found")
	}
	return nil
}

func formatEnvironmentFile(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	// Do not rewrite a symlink to a secret stored outside the project.
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("not a regular file")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	after := formatEnvironmentText(before)
	if bytes.Equal(before, after) {
		return false, nil
	}

	// Atomic replacement avoids leaving a partially written .env on failure.
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".copytygo-env-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(tmp.Name())

	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		tmp.Close()
		return false, err
	}
	if _, err := tmp.Write(after); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return false, err
	}
	return true, nil
}

func formatEnvironmentText(source []byte) []byte {
	eol := "\n"
	if bytes.Contains(source, []byte("\r\n")) {
		eol = "\r\n"
	}
	normalized := strings.ReplaceAll(string(source), "\r\n", "\n")
	if normalized == "" {
		return source
	}

	lines := strings.Split(normalized, "\n")
	output := make([]string, 0, len(lines)+8)
	lastSection := ""

	for _, line := range lines {
		section := environmentSection(line)
		if section != "" && lastSection != "" && section != lastSection {
			// Normalize only blank lines between known sections. Preserve
			// all other whitespace, comments and potential multiline values.
			for len(output) > 0 && output[len(output)-1] == "" {
				output = output[:len(output)-1]
			}
			output = append(output, "")
		}

		output = append(output, line)
		if section != "" {
			lastSection = section
		} else if line != "" && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			lastSection = ""
		}
	}

	return []byte(strings.Join(output, eol))
}

func environmentSection(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "export ") {
		return ""
	}

	name, _, ok := strings.Cut(trimmed, "=")
	if !ok {
		return ""
	}
	for _, section := range []string{
		"APP", "COPYTYGO", "SERVER", "DB", "QUEUE", "STORAGE", "MAIL",
	} {
		if strings.HasPrefix(strings.TrimSpace(name), section+"_") {
			return section
		}
	}
	return ""
}
