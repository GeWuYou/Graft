package logger

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newAppLogSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	schema := `CREATE TABLE app_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		occurred_at TIMESTAMP NOT NULL,
		severity TEXT NOT NULL,
		component TEXT NOT NULL,
		operation TEXT NULL,
		request_id TEXT NULL,
		trace_id TEXT NULL,
		route TEXT NULL,
		method TEXT NULL,
		error TEXT NULL,
		message TEXT NOT NULL,
		fields TEXT NOT NULL DEFAULT '{}'
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create app_logs schema: %v", err)
	}

	return db
}

func newSQLiteAppLogRepository(t *testing.T) AppLogRepository {
	t.Helper()

	repo, err := newAppLogRepositoryWithDialect(newAppLogSQLiteDB(t), appLogSQLDialectSQLite)
	if err != nil {
		t.Fatalf("new app log repository: %v", err)
	}

	return repo
}

func TestAppLogRepositoryCreateAndList(t *testing.T) {
	repo := newSQLiteAppLogRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 4, 8, 0, 0, 0, time.UTC)

	created, err := repo.CreateAppLog(ctx, CreateAppLogInput{
		OccurredAt: base,
		Severity:   AppLogSeverityError,
		Component:  " modules.user.route ",
		Operation:  " map user ",
		RequestID:  " req-1 ",
		TraceID:    " trace-1 ",
		Route:      " /api/users/:id ",
		Method:     " PATCH ",
		Error:      " bad \n response ",
		Message:    " map\tuser response failed ",
		Fields: map[string]string{
			"module name":  " user ",
			"access_token": "secret",
		},
	})
	if err != nil {
		t.Fatalf("create app log: %v", err)
	}

	if created.ID == 0 {
		t.Fatalf("expected generated id, got %#v", created)
	}
	if created.Component != "modules.user.route" || created.Message != "map user response failed" {
		t.Fatalf("expected normalized app log, got %#v", created)
	}
	if got := created.Fields["access_token"]; got != redactedValue {
		t.Fatalf("expected sensitive field redaction, got %q", got)
	}

	result, err := repo.ListAppLogs(ctx, AppLogListQuery{
		Severity:  AppLogSeverityError,
		Component: " modules.user.route ",
		Keyword:   "response",
		Page:      0,
		PageSize:  500,
	})
	if err != nil {
		t.Fatalf("list app logs: %v", err)
	}

	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one app log, got total=%d items=%d", result.Total, len(result.Items))
	}
	if result.Page != 1 || result.PageSize != appLogMaxPageSize {
		t.Fatalf("expected normalized paging, got page=%d pageSize=%d", result.Page, result.PageSize)
	}
	if got := result.Items[0].Fields["module_name"]; got != "user" {
		t.Fatalf("expected decoded fields, got %#v", result.Items[0].Fields)
	}

	detail, err := repo.GetAppLogByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get app log by id: %v", err)
	}
	if detail.ID != created.ID || detail.Message != created.Message {
		t.Fatalf("expected matching app log detail, got %#v", detail)
	}
}

func TestAppLogRepositoryDeleteBefore(t *testing.T) {
	repo := newSQLiteAppLogRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 4, 8, 0, 0, 0, time.UTC)

	for _, input := range []CreateAppLogInput{
		{OccurredAt: base, Severity: AppLogSeverityInfo, Component: "core.app", Message: "old"},
		{OccurredAt: base.Add(time.Hour), Severity: AppLogSeverityInfo, Component: "core.app", Message: "keep"},
	} {
		if _, err := repo.CreateAppLog(ctx, input); err != nil {
			t.Fatalf("seed app log: %v", err)
		}
	}

	deleted, err := repo.DeleteAppLogsBefore(ctx, base.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("delete app logs before: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one deleted row, got %d", deleted)
	}
}

func TestAppLogRepositorySortsByRequestedFields(t *testing.T) {
	repo := newSQLiteAppLogRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 4, 8, 0, 0, 0, time.UTC)

	for _, input := range []CreateAppLogInput{
		{OccurredAt: base, Severity: AppLogSeverityWarn, Component: "modules.user", Message: "third"},
		{OccurredAt: base.Add(time.Minute), Severity: AppLogSeverityError, Component: "core.app", Message: "first"},
		{OccurredAt: base.Add(2 * time.Minute), Severity: AppLogSeverityInfo, Component: "modules.auth", Message: "second"},
	} {
		if _, err := repo.CreateAppLog(ctx, input); err != nil {
			t.Fatalf("seed app log: %v", err)
		}
	}

	result, err := repo.ListAppLogs(ctx, AppLogListQuery{
		Sorters: []AppLogSorter{
			{Field: AppLogSortFieldComponent, Order: AppLogSortOrderAsc},
		},
	})
	if err != nil {
		t.Fatalf("list app logs: %v", err)
	}

	got := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		got = append(got, item.Component)
	}
	expected := []string{"core.app", "modules.auth", "modules.user"}
	for index := range expected {
		if got[index] != expected[index] {
			t.Fatalf("expected component order %#v, got %#v", expected, got)
		}
	}
}

func TestAppLogKeywordFilterUsesPostgresFullTextSearch(t *testing.T) {
	repo := &appLogRepository{dialect: appLogSQLDialectPostgres}

	whereSQL, args := repo.buildAppLogWhereClause(AppLogListQuery{Keyword: " response "})

	if len(args) != 1 || args[0] != "response" {
		t.Fatalf("expected normalized keyword arg, got %#v", args)
	}
	if !strings.Contains(whereSQL, "to_tsvector('simple'") || !strings.Contains(whereSQL, "@@ plainto_tsquery('simple', $1)") {
		t.Fatalf("expected postgres full-text keyword condition, got %s", whereSQL)
	}
	if strings.Contains(whereSQL, "concat_ws") {
		t.Fatalf("expected keyword search to use immutable index expression, got %s", whereSQL)
	}
	if strings.Contains(whereSQL, "LIKE") {
		t.Fatalf("expected keyword search to avoid leading-wildcard LIKE, got %s", whereSQL)
	}
}
