package cli

import (
	"os"
	"strings"
	"path/filepath"
	"testing"
)

func TestDiscoverModels(t *testing.T) {
	dir := t.TempDir()
	source := "package models\n\ntype Product struct {\n\tID int64 @@BT@@json:\"id\"@@BT@@\n\tName string @@BT@@json:\"name\"@@BT@@\n\tDeletedAt *string @@BT@@json:\"deleted_at,omitempty\"@@BT@@\n}\n"
	source = strings.ReplaceAll(source, "@@BT@@", string(rune(96)))
	if err := os.WriteFile(filepath.Join(dir, "product.go"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	models, err := DiscoverModels(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].Name != "Product" {
		t.Fatalf("unexpected models %#v", models)
	}
	if len(models[0].Fields) != 3 {
		t.Fatalf("unexpected fields %#v", models[0].Fields)
	}
	if models[0].Fields[2].Type != "*string" || models[0].Fields[2].JSON != "deleted_at" {
		t.Fatalf("unexpected field %#v", models[0].Fields[2])
	}
}
