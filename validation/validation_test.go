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


func TestNumericRejectsNonFiniteValues(t *testing.T) {
	for _, input := range []string{"NaN", "Inf", "+Inf", "-Inf", "1e999"} {
		t.Run(input, func(t *testing.T) {
			v := New(map[string]string{"price": input})
			v.Numeric("price")
			if v.Valid() {
				t.Fatalf("non-finite %q was accepted", input)
			}
		})
	}
	for _, input := range []string{"0", "123.45", "-0.75", "1e3"} {
		t.Run(input, func(t *testing.T) {
			v := New(map[string]string{"price": input})
			v.Numeric("price")
			if !v.Valid() {
				t.Fatalf("valid price %q was rejected: %v", input, v.Errors())
			}
		})
	}
}
