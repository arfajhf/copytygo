package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
	"github.com/arfajhf/copytygo/v4/queue"
)

func TestDurableQueueIntegration(t *testing.T) {
	if os.Getenv("COPYTYGO_INTEGRATION_DB") != "1" {
		t.Skip("set COPYTYGO_INTEGRATION_DB=1 to run database integration tests")
	}

	drivers.Register()
	db, err := database.Connect()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	q, err := queue.NewDatabaseQueue(db, os.Getenv("DB_DRIVER"))
	if err != nil {
		t.Fatal(err)
	}
	if err := q.Ensure(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(ctx, "DELETE FROM "+q.Table); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM "+q.FailedTable); err != nil {
		t.Fatal(err)
	}

	id, err := q.Dispatch(ctx, "SendReport", queue.Payload{"report_id": 42}, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("expected durable job id")
	}

	pending, err := q.Pending(ctx)
	if err != nil || pending != 1 {
		t.Fatalf("expected 1 pending job, got %d err=%v", pending, err)
	}

	job, err := q.Reserve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if job == nil || job.ID != id || job.Name != "SendReport" {
		t.Fatalf("unexpected reserved job %#v", job)
	}
	if got := job.Payload["report_id"]; got == nil {
		t.Fatalf("expected payload, got %#v", job.Payload)
	}

	if err := q.Release(ctx, job.ID, 1, 0); err != nil {
		t.Fatal(err)
	}

	job, err = q.Reserve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if job == nil || job.Attempts != 1 {
		t.Fatalf("expected released job with one attempt, got %#v", job)
	}

	if err := q.Complete(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	pending, err = q.Pending(ctx)
	if err != nil || pending != 0 {
		t.Fatalf("expected empty queue, got %d err=%v", pending, err)
	}

	_, err = q.Dispatch(ctx, "AlwaysFails", queue.Payload{"value": "x"}, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	failedJob, err := q.Reserve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if failedJob == nil {
		t.Fatal("expected job for failure path")
	}

	if err := q.Fail(ctx, failedJob, 1, errors.New("intentional failure")); err != nil {
		t.Fatal(err)
	}
	failed, err := q.Failed(ctx)
	if err != nil || failed != 1 {
		t.Fatalf("expected one failed job, got %d err=%v", failed, err)
	}

	pending, err = q.Pending(ctx)
	if err != nil || pending != 0 {
		t.Fatalf("expected no pending jobs after failure, got %d err=%v", pending, err)
	}
}
