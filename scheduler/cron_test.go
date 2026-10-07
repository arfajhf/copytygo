package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestCronParserMatchesAndFindsNext(t *testing.T) {
	cron, err := parseCron("*/15 9-17 * * 1-5")
	if err != nil {
		t.Fatal(err)
	}

	monday := time.Date(2026, 10, 5, 9, 30, 0, 0, time.Local)
	if !cron.matches(monday) {
		t.Fatal("expected cron to match Monday 09:30")
	}

	sunday := time.Date(2026, 10, 4, 9, 30, 0, 0, time.Local)
	if cron.matches(sunday) {
		t.Fatal("cron must not match Sunday")
	}

	next := cron.nextAfter(time.Date(2026, 10, 5, 9, 31, 0, 0, time.Local))
	expected := time.Date(2026, 10, 5, 9, 45, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, next)
	}
}

func TestSchedulerRegistersCron(t *testing.T) {
	s := New()
	if err := s.Cron("report", "0 8 * * 1-5", func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	entries := s.Entries()
	if len(entries) != 1 || entries[0].Cron != "0 8 * * 1-5" {
		t.Fatalf("unexpected entries %#v", entries)
	}
}
