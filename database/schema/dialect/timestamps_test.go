package dialect

import (
	"strings"
	"testing"

	"github.com/arfajhf/copytygo/v3/database/schema"
)

func TestTimestampsUseCurrentTimestampDefaults(t *testing.T) {
	blueprint := schema.Create("products", func(table *schema.Table) {
		table.ID()
		table.String("name")
		table.Timestamps()
	})

	mysqlSQL, err := (MySQL{}).Compile(blueprint)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(mysqlSQL, "DEFAULT CURRENT_TIMESTAMP") != 2 {
		t.Fatalf("expected MySQL timestamp defaults, got: %s", mysqlSQL)
	}

	postgresSQL, err := (PostgreSQL{}).Compile(blueprint)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(postgresSQL, "DEFAULT CURRENT_TIMESTAMP") != 2 {
		t.Fatalf("expected PostgreSQL timestamp defaults, got: %s", postgresSQL)
	}
}
