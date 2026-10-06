package cli

import "testing"

func TestParseResourceFields(t *testing.T) {
	fields, err := ParseResourceFields([]string{
		"name:string",
		"price:decimal",
		"stock:integer",
		"description:text?",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fields))
	}
	if fields[3].Name != "description" || !fields[3].Nullable {
		t.Fatalf("nullable field not parsed correctly: %#v", fields[3])
	}
	if got := migrationColumnForResourceField(fields[1]); got != `table.Decimal("price", 12, 2)` {
		t.Fatalf("unexpected decimal migration: %s", got)
	}
	if got := validationRuleForResourceField(fields[2]); got != "required|integer" {
		t.Fatalf("unexpected integer validation: %s", got)
	}
}

func TestParseResourceFieldsRejectsDuplicates(t *testing.T) {
	_, err := ParseResourceFields([]string{"name:string", "name:text"})
	if err == nil {
		t.Fatal("expected duplicate field error")
	}
}

func TestDefaultResourceField(t *testing.T) {
	fields, err := ParseResourceFields(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 || fields[0].Name != "name" || fields[0].Type != "string" {
		t.Fatalf("unexpected default fields: %#v", fields)
	}
}
