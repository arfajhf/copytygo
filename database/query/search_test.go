package query

import "testing"

func TestWhereAnyLikeBuildsSafeSearch(t *testing.T) {
	builder := &Builder{
		driver: "mysql",
		table: "products",
		columns: []string{"*"},
	}

	builder.WhereAnyLike([]string{"name", "description"}, "key")

	sql, args := builder.SQL()
	expected := "SELECT * FROM products WHERE (name LIKE ? OR description LIKE ?)"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
	if len(args) != 2 || args[0] != "%key%" || args[1] != "%key%" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
