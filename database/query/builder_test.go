package query

import "testing"

func TestIdentifierAndOperatorValidation(t *testing.T) {
	if !validIdentifier("products") || !validIdentifier("created_at") {
		t.Fatal("expected valid identifiers")
	}
	if validIdentifier("products;drop") || validIdentifier("user name") {
		t.Fatal("unsafe identifiers must be rejected")
	}

	for _, operator := range []string{"=", "!=", "<>", ">", ">=", "<", "<=", "LIKE"} {
		if !validOperator(operator) {
			t.Fatalf("expected operator %s to be valid", operator)
		}
	}
	if validOperator("OR 1=1") {
		t.Fatal("unsafe operator must be rejected")
	}
}

func TestWriteValuesRejectEmptyAndUnsafeColumns(t *testing.T) {
	builder := &Builder{}
	if err := builder.validateWriteValues(map[string]any{}); err == nil {
		t.Fatal("expected empty write values error")
	}

	builder = &Builder{}
	if err := builder.validateWriteValues(map[string]any{"name;drop": "x"}); err == nil {
		t.Fatal("expected unsafe column error")
	}
}
