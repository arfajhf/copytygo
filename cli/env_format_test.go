package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatEnvironmentTextPreservesKeysAndValues(t *testing.T) {
	original := "APP_NAME=demo\nAPP_KEY=secret-with=equals\nCOPYTYGO_STUDIO=true\n" +
		"SERVER_READ_TIMEOUT=15\nDB_DATABASE=demo\nDB_PASSWORD='contains spaces # and ='\n" +
		"QUEUE_DRIVER=database\nSTORAGE_PATH=storage/app\nMAIL_FROM=noreply@example.test\n"
	want := "APP_NAME=demo\nAPP_KEY=secret-with=equals\n\nCOPYTYGO_STUDIO=true\n\n" +
		"SERVER_READ_TIMEOUT=15\n\nDB_DATABASE=demo\nDB_PASSWORD='contains spaces # and ='\n\n" +
		"QUEUE_DRIVER=database\n\nSTORAGE_PATH=storage/app\n\nMAIL_FROM=noreply@example.test\n"
	result := formatEnvironmentText([]byte(original))
	if string(result) != want {
		t.Fatalf("unexpected format:\n%s\nwant:\n%s", result, want)
	}
	if !bytes.Equal(formatEnvironmentText(result), result) {
		t.Fatal("format is not idempotent")
	}
}

func TestFormatEnvironmentTextKeepsCommentsAndCRLF(t *testing.T) {
	before := "APP_ENV=local\r\n# The database below is local\r\nDB_DRIVER=mysql\r\n\r\n\r\nQUEUE_DRIVER=memory\r\n"
	after := formatEnvironmentText([]byte(before))
	if !bytes.Contains(after, []byte("# The database below is local")) {
		t.Fatal("custom comment was removed")
	}
	if !bytes.Contains(after, []byte("\r\n\r\n")) || bytes.Contains(after, []byte("\r\n\r\n\r\n")) {
		t.Fatalf("unexpected CRLF separators: %q", after)
	}
	if bytes.Contains(after, []byte("\nDB_DRIVER")) && !bytes.Contains(after, []byte("\r\nDB_DRIVER")) {
		t.Fatal("line-ending format changed")
	}
}

func TestFormatEnvironmentFilePreservesPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	before := []byte("APP_NAME=demo\nDB_PASSWORD=secret-value\n")
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := formatEnvironmentFile(path)
	if err != nil || !changed {
		t.Fatalf("failed to format: changed=%t err=%v", changed, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("dotenv permissions changed to %o", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "DB_PASSWORD=secret-value") {
		t.Fatal("value was changed")
	}
	changed, err = formatEnvironmentFile(path)
	if err != nil || changed {
		t.Fatalf("second format should do nothing: changed=%t err=%v", changed, err)
	}
}

func TestFormatEnvironmentFileRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "actual.env")
	path := filepath.Join(root, ".env")
	if err := os.WriteFile(target, []byte("APP_ENV=local\nDB_DRIVER=mysql\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := formatEnvironmentFile(path)
	if err == nil {
		t.Fatal("env:format should not overwrite symlinked secrets")
	}
}
