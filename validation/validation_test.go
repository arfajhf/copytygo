package validation

import "testing"

func TestExtendedRules(t *testing.T) {
	v := New(map[string]string{
		"price": "12.50",
		"active": "true",
		"uuid": "550e8400-e29b-41d4-a716-446655440000",
	})

	v.Numeric("price").Boolean("active").UUID("uuid")

	if !v.Valid() {
		t.Fatalf("unexpected errors: %#v", v.Errors())
	}
}

func TestExtendedRulesRejectInvalidValues(t *testing.T) {
	v := New(map[string]string{
		"price": "abc",
		"active": "maybe",
		"uuid": "not-a-uuid",
	})

	v.Numeric("price").Boolean("active").UUID("uuid")

	if v.Valid() {
		t.Fatal("expected validation errors")
	}
}
