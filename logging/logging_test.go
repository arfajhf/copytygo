package logging

import "testing"

func TestLoggerRingBuffer(t *testing.T) {
	logger := New(2)
	logger.Info("one", map[string]any{"n":1})
	logger.Warn("two", nil)
	logger.Error("three", nil)

	entries := logger.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Message != "three" || entries[1].Message != "two" {
		t.Fatalf("unexpected order %#v", entries)
	}
}
