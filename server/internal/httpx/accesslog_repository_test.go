package httpx

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newAccessLogSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	schema := `CREATE TABLE access_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		request_id TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		route TEXT NULL,
		status_code INTEGER NOT NULL,
		duration_ms BIGINT NOT NULL,
		client_ip TEXT NULL,
		user_agent TEXT NULL,
		user_id BIGINT NULL,
		username TEXT NULL,
		request_size BIGINT NULL,
		response_size BIGINT NULL,
		occurred_at TIMESTAMP NOT NULL
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create access_logs schema: %v", err)
	}

	return db
}

func newSQLiteAccessLogRepository(t *testing.T) AccessLogRepository {
	t.Helper()

	repo, err := newAccessLogRepositoryWithDialect(newAccessLogSQLiteDB(t), accessLogSQLDialectSQLite)
	if err != nil {
		t.Fatalf("new access log repository: %v", err)
	}

	return repo
}

func TestAccessLogRepositoryCreateAndBatchCreate(t *testing.T) {
	repo := newSQLiteAccessLogRepository(t)
	ctx := context.Background()
	userID := uint64(7)
	requestSize := int64(128)
	responseSize := int64(512)
	occurredAt := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)

	first, err := repo.CreateAccessLog(ctx, CreateAccessLogInput{
		RequestID:    "req-1",
		Method:       "POST",
		Path:         "/api/login?token=secret",
		Route:        "/api/login",
		StatusCode:   201,
		DurationMS:   42,
		ClientIP:     "203.0.113.10",
		UserAgent:    "curl authorization=Bearer secret-token",
		UserID:       &userID,
		Username:     "alice",
		RequestSize:  &requestSize,
		ResponseSize: &responseSize,
		OccurredAt:   occurredAt,
	})
	if err != nil {
		t.Fatalf("create access log: %v", err)
	}

	if first.ID == 0 {
		t.Fatalf("expected generated id, got %#v", first)
	}
	if first.Path != "/api/login?token=[REDACTED]" {
		t.Fatalf("expected sanitized repository path, got %q", first.Path)
	}
	if first.UserAgent != "curl authorization=[REDACTED]" {
		t.Fatalf("expected sanitized user agent, got %q", first.UserAgent)
	}

	batch, err := repo.CreateAccessLogs(ctx, []CreateAccessLogInput{
		{
			RequestID:  "req-2",
			Method:     "GET",
			Path:       "/healthz",
			Route:      "/healthz",
			StatusCode: 204,
			DurationMS: 3,
			OccurredAt: occurredAt.Add(time.Second),
		},
		{
			RequestID:  "req-3",
			Method:     "GET",
			Path:       "/api/users/password=guessme",
			Route:      "/api/users/:id",
			StatusCode: 200,
			DurationMS: 9,
			OccurredAt: occurredAt.Add(2 * time.Second),
		},
	})
	if err != nil {
		t.Fatalf("batch create access logs: %v", err)
	}

	if len(batch) != 2 {
		t.Fatalf("expected two batch items, got %d", len(batch))
	}
	if batch[1].Path != "/api/users/password=[REDACTED]" {
		t.Fatalf("expected batch sanitization, got %q", batch[1].Path)
	}
}

func TestAccessLogRepositoryDeleteBefore(t *testing.T) {
	repo := newSQLiteAccessLogRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)

	_, err := repo.CreateAccessLogs(ctx, []CreateAccessLogInput{
		{RequestID: "req-old", Method: "GET", Path: "/old", StatusCode: 200, DurationMS: 1, OccurredAt: base},
		{RequestID: "req-keep", Method: "GET", Path: "/keep", StatusCode: 200, DurationMS: 1, OccurredAt: base.Add(time.Hour)},
	})
	if err != nil {
		t.Fatalf("seed access logs: %v", err)
	}

	deleted, err := repo.DeleteAccessLogsBefore(ctx, base.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("delete access logs before: %v", err)
	}

	if deleted != 1 {
		t.Fatalf("expected one deleted row, got %d", deleted)
	}
}
