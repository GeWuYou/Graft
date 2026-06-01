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
		trace_id TEXT NULL,
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
		started_at TIMESTAMP NOT NULL,
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
	startedAt := occurredAt.Add(-42 * time.Millisecond)

	first, err := repo.CreateAccessLog(ctx, CreateAccessLogInput{
		RequestID:    "req-1",
		TraceID:      "req-1",
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
		StartedAt:    startedAt,
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
	if first.TraceID != "" {
		t.Fatalf("expected trace id identical to request id to normalize to empty, got %q", first.TraceID)
	}

	batch, err := repo.CreateAccessLogs(ctx, []CreateAccessLogInput{
		{
			RequestID:  "req-2",
			TraceID:    "req-2",
			Method:     "GET",
			Path:       "/healthz",
			Route:      "/healthz",
			StatusCode: 204,
			DurationMS: 3,
			OccurredAt: occurredAt.Add(time.Second),
		},
		{
			RequestID:  "req-3",
			TraceID:    "trace-3",
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
	if batch[0].TraceID != "" {
		t.Fatalf("expected batch trace id alias to normalize to empty, got %q", batch[0].TraceID)
	}
}

func TestAccessLogRepositoryDeleteBefore(t *testing.T) {
	repo := newSQLiteAccessLogRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)

	_, err := repo.CreateAccessLogs(ctx, []CreateAccessLogInput{
		{RequestID: "req-old", TraceID: "trace-old", Method: "GET", Path: "/old", StatusCode: 200, DurationMS: 1, OccurredAt: base},
		{RequestID: "req-keep", TraceID: "trace-keep", Method: "GET", Path: "/keep", StatusCode: 200, DurationMS: 1, OccurredAt: base.Add(time.Hour)},
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

func TestAccessLogRepositoryListAccessLogsSortNormalization(t *testing.T) {
	repo := newSQLiteAccessLogRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)

	seedAccessLogSortNormalizationCases(ctx, t, repo, base)

	testCases := accessLogSortNormalizationTestCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, listErr := repo.ListAccessLogs(ctx, testCase.query)
			if listErr != nil {
				t.Fatalf("list access logs: %v", listErr)
			}
			assertAccessLogRequestOrder(t, result, testCase.wantOrder)
		})
	}
}

type accessLogSortNormalizationCase struct {
	name      string
	query     AccessLogListQuery
	wantOrder []string
}

func accessLogSortNormalizationTestCases() []accessLogSortNormalizationCase {
	return []accessLogSortNormalizationCase{
		{
			name: "occurred at desc",
			query: AccessLogListQuery{
				Page:     1,
				PageSize: 10,
				Sorts: []AccessLogSort{{
					Field: AccessLogSortOccurredAt,
					Order: AccessLogSortOrderDesc,
				}},
			},
			wantOrder: []string{"req-b", "req-c", "req-a"},
		},
		{
			name: "started at desc",
			query: AccessLogListQuery{
				Page:     1,
				PageSize: 10,
				Sorts: []AccessLogSort{{
					Field: AccessLogSortStartedAt,
					Order: AccessLogSortOrderDesc,
				}},
			},
			wantOrder: []string{"req-b", "req-c", "req-a"},
		},
		{
			name: "duration asc",
			query: AccessLogListQuery{
				Page:     1,
				PageSize: 10,
				Sorts: []AccessLogSort{{
					Field: AccessLogSortDurationMS,
					Order: AccessLogSortOrderAsc,
				}},
			},
			wantOrder: []string{"req-b", "req-a", "req-c"},
		},
		{
			name: "status desc",
			query: AccessLogListQuery{
				Page:     1,
				PageSize: 10,
				Sorts: []AccessLogSort{{
					Field: AccessLogSortStatusCode,
					Order: AccessLogSortOrderDesc,
				}},
			},
			wantOrder: []string{"req-c", "req-a", "req-b"},
		},
		{
			name: "invalid sort field falls back to repository default order",
			query: AccessLogListQuery{
				Page:     1,
				PageSize: 10,
				Sorts: []AccessLogSort{{
					Field: AccessLogSortField("occurred_at; DROP TABLE access_logs; --"),
					Order: AccessLogSortOrderAsc,
				}},
			},
			wantOrder: []string{"req-b", "req-c", "req-a"},
		},
		{
			name: "invalid sort order falls back to desc",
			query: AccessLogListQuery{
				Page:     1,
				PageSize: 10,
				Sorts: []AccessLogSort{{
					Field: AccessLogSortOccurredAt,
					Order: AccessLogSortOrder("asc; DROP TABLE access_logs; --"),
				}},
			},
			wantOrder: []string{"req-b", "req-c", "req-a"},
		},
		{
			name: "invalid sort field and order still list safely",
			query: AccessLogListQuery{
				Page:     1,
				PageSize: 10,
				Sorts: []AccessLogSort{{
					Field: AccessLogSortField("status_code desc; DROP TABLE access_logs; --"),
					Order: AccessLogSortOrder("desc NULLS LAST"),
				}},
			},
			wantOrder: []string{"req-b", "req-c", "req-a"},
		},
		{
			name: "no explicit sort keeps backend default pagination order",
			query: AccessLogListQuery{
				Page:     1,
				PageSize: 10,
			},
			wantOrder: []string{"req-b", "req-c", "req-a"},
		},
	}
}

func TestAccessLogRepositoryListAccessLogsFiltersByTraceIDColumn(t *testing.T) {
	repo := newSQLiteAccessLogRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)

	_, err := repo.CreateAccessLogs(ctx, []CreateAccessLogInput{
		{RequestID: "req-1", TraceID: "req-1", Method: "GET", Path: "/a", StatusCode: 200, DurationMS: 1, OccurredAt: base},
		{RequestID: "req-2", TraceID: "trace-kept", Method: "POST", Path: "/b", StatusCode: 201, DurationMS: 2, OccurredAt: base.Add(time.Minute)},
		{RequestID: "req-3", TraceID: "trace-other", Method: "GET", Path: "/c", StatusCode: 500, DurationMS: 3, OccurredAt: base.Add(2 * time.Minute)},
	})
	if err != nil {
		t.Fatalf("seed access logs: %v", err)
	}

	result, err := repo.ListAccessLogs(ctx, AccessLogListQuery{
		Page:     1,
		PageSize: 10,
		TraceID:  "trace-kept",
		Sorts: []AccessLogSort{{
			Field: AccessLogSortOccurredAt,
			Order: AccessLogSortOrderAsc,
		}},
	})
	if err != nil {
		t.Fatalf("list access logs by trace id: %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	assertAccessLogRequestOrder(t, result, []string{"req-2"})
	if result.Items[0].TraceID != "trace-kept" {
		t.Fatalf("expected stored independent trace id to remain readable, got %#v", result.Items[0])
	}
}

func TestAccessLogRepositoryListAccessLogsEscapesPrefixPathWildcards(t *testing.T) {
	repo := newSQLiteAccessLogRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)

	_, err := repo.CreateAccessLogs(ctx, []CreateAccessLogInput{
		{RequestID: "req-underscore-match", Method: "GET", Path: "/api/a_b", StatusCode: 200, DurationMS: 1, OccurredAt: base},
		{RequestID: "req-underscore-miss", Method: "GET", Path: "/api/axb", StatusCode: 200, DurationMS: 1, OccurredAt: base.Add(time.Minute)},
		{RequestID: "req-percent-match", Method: "GET", Path: "/api/100%/detail", StatusCode: 200, DurationMS: 1, OccurredAt: base.Add(2 * time.Minute)},
		{RequestID: "req-backslash-match", Method: "GET", Path: "/files/a\\b", StatusCode: 200, DurationMS: 1, OccurredAt: base.Add(3 * time.Minute)},
		{RequestID: "req-backslash-miss", Method: "GET", Path: "/files/ab", StatusCode: 200, DurationMS: 1, OccurredAt: base.Add(4 * time.Minute)},
	})
	if err != nil {
		t.Fatalf("seed access logs: %v", err)
	}

	testCases := []struct {
		name      string
		path      string
		wantOrder []string
	}{
		{
			name:      "underscore remains literal",
			path:      "/api/a_b",
			wantOrder: []string{"req-underscore-match"},
		},
		{
			name:      "percent remains literal",
			path:      "/api/100%",
			wantOrder: []string{"req-percent-match"},
		},
		{
			name:      "backslash remains literal",
			path:      "/files/a\\",
			wantOrder: []string{"req-backslash-match"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, listErr := repo.ListAccessLogs(ctx, AccessLogListQuery{
				Page:          1,
				PageSize:      10,
				Path:          testCase.path,
				PathMatchMode: AccessLogPathMatchPrefix,
				Sorts: []AccessLogSort{{
					Field: AccessLogSortOccurredAt,
					Order: AccessLogSortOrderAsc,
				}},
			})
			if listErr != nil {
				t.Fatalf("list access logs by prefix path: %v", listErr)
			}
			assertAccessLogRequestOrder(t, result, testCase.wantOrder)
		})
	}
}

func TestAccessLogRepositoryListAccessLogsFiltersStartedAtAndOccurredAtIndependently(t *testing.T) {
	repo := newSQLiteAccessLogRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)

	_, err := repo.CreateAccessLogs(ctx, []CreateAccessLogInput{
		{
			RequestID:  "req-a",
			Method:     "GET",
			Path:       "/a",
			StatusCode: 200,
			DurationMS: 10,
			StartedAt:  base.Add(-90 * time.Minute),
			OccurredAt: base.Add(-60 * time.Minute),
		},
		{
			RequestID:  "req-b",
			Method:     "GET",
			Path:       "/b",
			StatusCode: 200,
			DurationMS: 10,
			StartedAt:  base.Add(-30 * time.Minute),
			OccurredAt: base.Add(-5 * time.Minute),
		},
		{
			RequestID:  "req-c",
			Method:     "GET",
			Path:       "/c",
			StatusCode: 200,
			DurationMS: 10,
			StartedAt:  base.Add(-15 * time.Minute),
			OccurredAt: base.Add(15 * time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("seed access logs: %v", err)
	}

	result, err := repo.ListAccessLogs(ctx, AccessLogListQuery{
		Page:         1,
		PageSize:     10,
		StartedFrom:  timePointer(base.Add(-40 * time.Minute)),
		StartedTo:    timePointer(base),
		OccurredFrom: timePointer(base.Add(-10 * time.Minute)),
		OccurredTo:   timePointer(base),
		Sorts: []AccessLogSort{{
			Field: AccessLogSortStartedAt,
			Order: AccessLogSortOrderAsc,
		}},
	})
	if err != nil {
		t.Fatalf("list access logs with independent time filters: %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	assertAccessLogRequestOrder(t, result, []string{"req-b"})
}

func TestAccessLogRepositoryListAccessLogsSupportsKeywordAndStatusGroupFilters(t *testing.T) {
	repo := newSQLiteAccessLogRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)

	_, err := repo.CreateAccessLogs(ctx, []CreateAccessLogInput{
		{RequestID: "req-404", Method: "GET", Path: "/api/users", Username: "alice", StatusCode: 404, DurationMS: 1, OccurredAt: base},
		{RequestID: "req-500", Method: "GET", Path: "/api/orders", Username: "bob", StatusCode: 500, DurationMS: 1, OccurredAt: base.Add(time.Minute)},
		{RequestID: "req-200", Method: "GET", Path: "/healthz", Username: "ops", StatusCode: 200, DurationMS: 1, OccurredAt: base.Add(2 * time.Minute)},
	})
	if err != nil {
		t.Fatalf("seed access logs: %v", err)
	}

	result, err := repo.ListAccessLogs(ctx, AccessLogListQuery{
		Page:         1,
		PageSize:     10,
		Keyword:      "alice",
		StatusGroups: []AccessLogStatusGroup{AccessLogStatusGroup4xx},
	})
	if err != nil {
		t.Fatalf("list access logs with keyword and status_group: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	assertAccessLogRequestOrder(t, result, []string{"req-404"})
}

func seedAccessLogSortNormalizationCases(
	ctx context.Context,
	t *testing.T,
	repo AccessLogRepository,
	base time.Time,
) {
	t.Helper()

	_, err := repo.CreateAccessLogs(ctx, []CreateAccessLogInput{
		{
			RequestID:  "req-a",
			TraceID:    "trace-a",
			Method:     "GET",
			Path:       "/a",
			StatusCode: 404,
			DurationMS: 15,
			OccurredAt: base,
		},
		{
			RequestID:  "req-b",
			TraceID:    "trace-b",
			Method:     "GET",
			Path:       "/b",
			StatusCode: 200,
			DurationMS: 5,
			OccurredAt: base.Add(2 * time.Minute),
		},
		{
			RequestID:  "req-c",
			TraceID:    "trace-c",
			Method:     "GET",
			Path:       "/c",
			StatusCode: 500,
			DurationMS: 30,
			OccurredAt: base.Add(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("seed access logs: %v", err)
	}
}

func assertAccessLogRequestOrder(t *testing.T, result AccessLogListResult, wantOrder []string) {
	t.Helper()

	if len(result.Items) != len(wantOrder) {
		t.Fatalf("expected %d items, got %d", len(wantOrder), len(result.Items))
	}
	for index, wantRequestID := range wantOrder {
		if result.Items[index].RequestID != wantRequestID {
			t.Fatalf("item %d: expected request id %q, got %q", index, wantRequestID, result.Items[index].RequestID)
		}
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}
