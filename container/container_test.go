package container

import "testing"

func TestSingletonResolvesOnce(t *testing.T) {
	c := New()
	calls := 0
	c.Singleton("service", func(*Container) (any, error) {
		calls++
		return &struct{ Name string }{Name: "CopyTyGo"}, nil
	})

	a, err := c.Resolve("service")
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.Resolve("service")
	if err != nil {
		t.Fatal(err)
	}
	if a != b || calls != 1 {
		t.Fatalf("singleton should resolve once, calls=%d", calls)
	}
}

func TestResolveAs(t *testing.T) {
	c := New()
	c.Instance("number", 42)
	value, err := ResolveAs[int](c, "number")
	if err != nil {
		t.Fatal(err)
	}
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}
}
