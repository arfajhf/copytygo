package queue

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Payload map[string]any

type PayloadHandler func(context.Context, Payload) error

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]PayloadHandler
}

func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]PayloadHandler)}
}

var DefaultRegistry = NewRegistry()

func (r *Registry) Register(name string, handler PayloadHandler) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("copytygo queue: job name is required")
	}
	if handler == nil {
		return fmt.Errorf("copytygo queue: handler for %q is required", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("copytygo queue: job %q already registered", name)
	}
	r.handlers[name] = handler
	return nil
}

func (r *Registry) Handler(name string) (PayloadHandler, bool) {
	r.mu.RLock()
	handler, ok := r.handlers[name]
	r.mu.RUnlock()
	return handler, ok
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}

type DatabaseJob struct {
	ID          string
	Name        string
	Payload     Payload
	Attempts    int
	MaxAttempts int
	AvailableAt time.Time
	CreatedAt   time.Time
}

type DatabaseQueue struct {
	DB          *sql.DB
	Driver      string
	Table       string
	FailedTable string
}

var queueIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func NewDatabaseQueue(db *sql.DB, driver string) (*DatabaseQueue, error) {
	if db == nil {
		return nil, fmt.Errorf("copytygo queue: database connection is required")
	}
	driver = strings.ToLower(strings.TrimSpace(driver))
	if driver != "mysql" && driver != "postgres" {
		return nil, fmt.Errorf("copytygo queue: unsupported database driver %q", driver)
	}
	return &DatabaseQueue{
		DB: db,
		Driver: driver,
		Table: "_copytygo_jobs",
		FailedTable: "_copytygo_failed_jobs",
	}, nil
}

func (q *DatabaseQueue) Ensure(ctx context.Context) error {
	if !queueIdentifier.MatchString(q.Table) || !queueIdentifier.MatchString(q.FailedTable) {
		return fmt.Errorf("copytygo queue: invalid queue table name")
	}

	jobs := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id VARCHAR(64) NOT NULL,
	name VARCHAR(255) NOT NULL,
	payload TEXT NOT NULL,
	attempts INTEGER NOT NULL DEFAULT 0,
	max_attempts INTEGER NOT NULL DEFAULT 3,
	available_at TIMESTAMP(6) NOT NULL,
	reserved_at TIMESTAMP(6) NULL,
	created_at TIMESTAMP(6) NOT NULL,
	PRIMARY KEY (id)
)`, q.Table)

	failed := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id VARCHAR(64) NOT NULL,
	name VARCHAR(255) NOT NULL,
	payload TEXT NOT NULL,
	attempts INTEGER NOT NULL,
	error TEXT NOT NULL,
	failed_at TIMESTAMP(6) NOT NULL,
	PRIMARY KEY (id)
)`, q.FailedTable)

	if _, err := q.DB.ExecContext(ctx, jobs); err != nil {
		return fmt.Errorf("copytygo queue: create jobs table: %w", err)
	}
	if _, err := q.DB.ExecContext(ctx, failed); err != nil {
		return fmt.Errorf("copytygo queue: create failed jobs table: %w", err)
	}
	return nil
}

func (q *DatabaseQueue) Dispatch(
	ctx context.Context,
	name string,
	payload Payload,
	maxAttempts int,
	delay time.Duration,
) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("copytygo queue: job name is required")
	}
	if maxAttempts < 1 {
		maxAttempts = 3
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("copytygo queue: encode payload: %w", err)
	}

	id, err := queueID()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	available := now.Add(delay)

	query := fmt.Sprintf(
		"INSERT INTO %s (id, name, payload, attempts, max_attempts, available_at, reserved_at, created_at) VALUES (%s,%s,%s,%s,%s,%s,NULL,%s)",
		q.Table,
		q.placeholder(1), q.placeholder(2), q.placeholder(3), q.placeholder(4),
		q.placeholder(5), q.placeholder(6), q.placeholder(7),
	)

	_, err = q.DB.ExecContext(ctx, query, id, name, string(raw), 0, maxAttempts, available, now)
	if err != nil {
		return "", fmt.Errorf("copytygo queue: dispatch database job: %w", err)
	}
	return id, nil
}

func (q *DatabaseQueue) Reserve(ctx context.Context) (*DatabaseJob, error) {
	for try := 0; try < 5; try++ {
		query := fmt.Sprintf(
			"SELECT id, name, payload, attempts, max_attempts, available_at, created_at FROM %s WHERE reserved_at IS NULL AND available_at <= %s ORDER BY created_at ASC LIMIT 1",
			q.Table,
			q.placeholder(1),
		)

		var job DatabaseJob
		var raw string
		err := q.DB.QueryRowContext(ctx, query, time.Now().UTC()).Scan(
			&job.ID,
			&job.Name,
			&raw,
			&job.Attempts,
			&job.MaxAttempts,
			&job.AvailableAt,
			&job.CreatedAt,
		)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("copytygo queue: reserve job: %w", err)
		}

		if err := json.Unmarshal([]byte(raw), &job.Payload); err != nil {
			return nil, fmt.Errorf("copytygo queue: decode job payload: %w", err)
		}

		update := fmt.Sprintf(
			"UPDATE %s SET reserved_at = %s WHERE id = %s AND reserved_at IS NULL",
			q.Table,
			q.placeholder(1),
			q.placeholder(2),
		)
		result, err := q.DB.ExecContext(ctx, update, time.Now().UTC(), job.ID)
		if err != nil {
			return nil, fmt.Errorf("copytygo queue: mark job reserved: %w", err)
		}
		affected, err := result.RowsAffected()
		if err == nil && affected == 1 {
			return &job, nil
		}
	}
	return nil, nil
}

func (q *DatabaseQueue) Complete(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = %s", q.Table, q.placeholder(1))
	if _, err := q.DB.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("copytygo queue: complete job: %w", err)
	}
	return nil
}

func (q *DatabaseQueue) Release(ctx context.Context, id string, attempts int, delay time.Duration) error {
	query := fmt.Sprintf(
		"UPDATE %s SET attempts = %s, available_at = %s, reserved_at = NULL WHERE id = %s",
		q.Table,
		q.placeholder(1),
		q.placeholder(2),
		q.placeholder(3),
	)
	_, err := q.DB.ExecContext(ctx, query, attempts, time.Now().UTC().Add(delay), id)
	if err != nil {
		return fmt.Errorf("copytygo queue: release job: %w", err)
	}
	return nil
}

func (q *DatabaseQueue) Fail(ctx context.Context, job *DatabaseJob, attempts int, failure error) error {
	raw, err := json.Marshal(job.Payload)
	if err != nil {
		return err
	}

	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	insert := fmt.Sprintf(
		"INSERT INTO %s (id, name, payload, attempts, error, failed_at) VALUES (%s,%s,%s,%s,%s,%s)",
		q.FailedTable,
		q.placeholder(1), q.placeholder(2), q.placeholder(3),
		q.placeholder(4), q.placeholder(5), q.placeholder(6),
	)
	if _, err := tx.ExecContext(
		ctx,
		insert,
		job.ID,
		job.Name,
		string(raw),
		attempts,
		failure.Error(),
		time.Now().UTC(),
	); err != nil {
		return fmt.Errorf("copytygo queue: record failed job: %w", err)
	}

	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE id = %s", q.Table, q.placeholder(1))
	if _, err := tx.ExecContext(ctx, deleteQuery, job.ID); err != nil {
		return fmt.Errorf("copytygo queue: delete failed job from queue: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("copytygo queue: commit failed job: %w", err)
	}
	return nil
}

func (q *DatabaseQueue) Pending(ctx context.Context) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", q.Table)
	var count int64
	if err := q.DB.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *DatabaseQueue) Failed(ctx context.Context) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", q.FailedTable)
	var count int64
	if err := q.DB.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *DatabaseQueue) placeholder(position int) string {
	if q.Driver == "postgres" {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

func queueID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("copytygo queue: generate job id: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

type DurableStats struct {
	Running   int64 `json:"running"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
}

type DatabaseWorker struct {
	Queue        *DatabaseQueue
	Registry     *Registry
	PollInterval time.Duration
	Backoff      time.Duration

	running   atomic.Int64
	completed atomic.Int64
	failed    atomic.Int64
}

func NewDatabaseWorker(queue *DatabaseQueue, registry *Registry) *DatabaseWorker {
	if registry == nil {
		registry = DefaultRegistry
	}
	return &DatabaseWorker{
		Queue: queue,
		Registry: registry,
		PollInterval: time.Second,
		Backoff: time.Second,
	}
}

func (w *DatabaseWorker) Run(ctx context.Context, concurrency int) error {
	if w == nil || w.Queue == nil {
		return fmt.Errorf("copytygo queue: database worker requires a queue")
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if w.PollInterval <= 0 {
		w.PollInterval = time.Second
	}
	if w.Backoff < 0 {
		w.Backoff = 0
	}

	if err := w.Queue.Ensure(ctx); err != nil {
		return err
	}

	var wg sync.WaitGroup
	for index := 0; index < concurrency; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.loop(ctx)
		}()
	}

	<-ctx.Done()
	wg.Wait()
	return nil
}

func (w *DatabaseWorker) Stats() DurableStats {
	return DurableStats{
		Running: w.running.Load(),
		Completed: w.completed.Load(),
		Failed: w.failed.Load(),
	}
}

func (w *DatabaseWorker) loop(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}

		job, err := w.Queue.Reserve(ctx)
		if err != nil {
			if !sleepContext(ctx, w.PollInterval) {
				return
			}
			continue
		}
		if job == nil {
			if !sleepContext(ctx, w.PollInterval) {
				return
			}
			continue
		}

		w.running.Add(1)
		w.process(ctx, job)
		w.running.Add(-1)
	}
}

func (w *DatabaseWorker) process(ctx context.Context, job *DatabaseJob) {
	handler, ok := w.Registry.Handler(job.Name)
	if !ok {
		w.failed.Add(1)
		_ = w.Queue.Fail(ctx, job, job.Attempts+1, fmt.Errorf("job handler %q is not registered", job.Name))
		return
	}

	attempt := job.Attempts + 1
	if err := handler(ctx, job.Payload); err != nil {
		if attempt >= job.MaxAttempts {
			w.failed.Add(1)
			_ = w.Queue.Fail(ctx, job, attempt, err)
			return
		}
		_ = w.Queue.Release(ctx, job.ID, attempt, w.Backoff*time.Duration(attempt))
		return
	}

	if err := w.Queue.Complete(ctx, job.ID); err != nil {
		w.failed.Add(1)
		return
	}
	w.completed.Add(1)
}

func sleepContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
