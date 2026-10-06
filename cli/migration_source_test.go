package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arfajhf/copytygo/v2/database/migration"
)

func TestRegisterProjectMigrationsFromSource(t *testing.T) {
	dir := t.TempDir()

	source := `package migrations

import (
	"github.com/arfajhf/copytygo/v2/database/migration"
	"github.com/arfajhf/copytygo/v2/database/schema"
)

func Register20261006120000() error {
	return migration.Register(
		"20261006_120000_create_products_table",
		func() *schema.Blueprint {
			return schema.Create("products", func(table *schema.Table) {
				table.ID()
				table.String("name", 120).Unique()
				table.Decimal("price", 12, 2)
				table.Text("description").Nullable()
				table.Boolean("active").DefaultValue(true)
				table.Timestamps()
			})
		},
		func() *schema.Blueprint {
			return schema.Drop("products")
		},
	)
}
`

	if err := os.WriteFile(
		filepath.Join(dir, "20261006_120000_create_products_table.go"),
		[]byte(source),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := RegisterProjectMigrations(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(migration.ResetRegistry)

	items := migration.All()
	if len(items) != 1 {
		t.Fatalf("expected 1 migration, got %d", len(items))
	}

	up := items[0].Up()
	if up == nil || up.Table == nil || up.Table.Name != "products" {
		t.Fatalf("unexpected up blueprint: %#v", up)
	}
	if len(up.Table.Columns) != 8 {
		t.Fatalf("expected 8 columns including timestamps, got %d", len(up.Table.Columns))
	}

	name := up.Table.Columns[1]
	if name.Name != "name" || name.Length != 120 || !name.IsUnique {
		t.Fatalf("unexpected name column: %#v", name)
	}

	description := up.Table.Columns[3]
	if !description.IsNullable {
		t.Fatalf("expected description nullable: %#v", description)
	}

	active := up.Table.Columns[4]
	if !active.HasDefault || active.Default != true {
		t.Fatalf("expected active default true: %#v", active)
	}

	down := items[0].Down()
	if down == nil || down.Table == nil || down.Table.Name != "products" {
		t.Fatalf("unexpected down blueprint: %#v", down)
	}
}
