package factory

import "testing"

func TestFactorySequence(t *testing.T) {
	f := New(func(n uint64) uint64 { return n })
	items, err := f.Count(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0] != 1 || items[2] != 3 {
		t.Fatalf("unexpected sequence %#v", items)
	}
}
