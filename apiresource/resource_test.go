package apiresource

import "testing"

type user struct {
	ID int
	Name string
	Secret string
}

func TestCollection(t *testing.T) {
	items := []user{{ID:1,Name:"A",Secret:"x"},{ID:2,Name:"B",Secret:"y"}}
	out := Collection(items, func(u user) any {
		return map[string]any{"id":u.ID,"name":u.Name}
	})
	if len(out) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(out))
	}
	first := out[0].(map[string]any)
	if _, ok := first["Secret"]; ok {
		t.Fatal("resource leaked secret")
	}
}

func TestPage(t *testing.T) {
	p := Page([]int{1,2}, 2, 2, 5)
	if p.TotalPages != 3 {
		t.Fatalf("expected 3 pages, got %d", p.TotalPages)
	}
}
