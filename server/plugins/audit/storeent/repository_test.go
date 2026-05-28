package storeent

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	auditstore "graft/server/plugins/audit/store"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file:audit-plugin-storeent?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	schema := `CREATE TABLE audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		actor_user_id INTEGER NULL,
		actor_username TEXT NOT NULL DEFAULT '',
		actor_display_name TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL DEFAULT '',
		resource_id TEXT NOT NULL DEFAULT '',
		resource_name TEXT NOT NULL DEFAULT '',
		success BOOLEAN NOT NULL DEFAULT 0,
		request_id TEXT NOT NULL DEFAULT '',
		ip TEXT NOT NULL DEFAULT '',
		user_agent TEXT NOT NULL DEFAULT '',
		message TEXT NOT NULL DEFAULT '',
		metadata TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create audit schema: %v", err)
	}

	return db
}

func TestRepositoryCreateAndListAuditLogs(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	actorID := uint64(7)
	firstCreatedAt := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	secondCreatedAt := firstCreatedAt.Add(time.Minute)
	_, err = repo.CreateAuditLog(ctx, auditstore.CreateAuditLogInput{
		ActorUserID:      &actorID,
		ActorUsername:    "alice",
		ActorDisplayName: "Alice",
		Action:           "user.update",
		ResourceType:     "user",
		ResourceID:       "7",
		ResourceName:     "Alice",
		Success:          true,
		RequestID:        "req-1",
		IP:               "127.0.0.1",
		UserAgent:        "curl/8",
		Message:          "profile updated",
		Metadata:         json.RawMessage(`{"field":"display"}`),
		CreatedAt:        firstCreatedAt,
	})
	if err != nil {
		t.Fatalf("create first audit log: %v", err)
	}

	_, err = repo.CreateAuditLog(ctx, auditstore.CreateAuditLogInput{
		Action:       "user.delete",
		ResourceType: "user",
		ResourceID:   "8",
		Success:      false,
		RequestID:    "req-2",
		Message:      "delete failed",
		Metadata:     json.RawMessage(`{"reason":"conflict"}`),
		CreatedAt:    secondCreatedAt,
	})
	if err != nil {
		t.Fatalf("create second audit log: %v", err)
	}

	result, err := repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		ResourceType: "user",
		Limit:        10,
		Offset:       0,
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("expected two audit logs, got %#v", result)
	}
	if result.Items[0].Action != "user.delete" || result.Items[1].Action != "user.update" {
		t.Fatalf("expected descending created_at order, got %#v", result.Items)
	}
	if result.Items[1].ActorUserID == nil || *result.Items[1].ActorUserID != actorID {
		t.Fatalf("expected actor user id to round-trip, got %#v", result.Items[1].ActorUserID)
	}
	if string(result.Items[1].Metadata) != `{"field":"display"}` {
		t.Fatalf("expected metadata to round-trip, got %s", result.Items[1].Metadata)
	}
}

func TestRepositoryListAuditLogsAppliesFilters(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	success := true
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	for _, entry := range []auditstore.CreateAuditLogInput{
		{
			Action:       "user.update",
			ResourceType: "user",
			ResourceID:   "7",
			ResourceName: "Alice",
			Success:      true,
			RequestID:    "req-keep",
			CreatedAt:    now,
		},
		{
			Action:       "user.update",
			ResourceType: "user",
			ResourceID:   "8",
			ResourceName: "Bob",
			Success:      false,
			RequestID:    "req-drop",
			CreatedAt:    now.Add(time.Minute),
		},
	} {
		if _, err := repo.CreateAuditLog(ctx, entry); err != nil {
			t.Fatalf("seed audit log: %v", err)
		}
	}

	result, err := repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		Action:    "user.update",
		Success:   &success,
		RequestID: "req-keep",
		Limit:     10,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("list filtered audit logs: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one filtered audit log, got %#v", result)
	}
	if result.Items[0].RequestID != "req-keep" || !result.Items[0].Success {
		t.Fatalf("unexpected filtered record %#v", result.Items[0])
	}
}

func TestRepositoryListAuditLogsRejectsInvalidPagination(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()

	_, err = repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{Limit: 0, Offset: 0})
	if err == nil || !strings.Contains(err.Error(), "invalid limit") {
		t.Fatalf("expected invalid limit error, got %v", err)
	}

	_, err = repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{Limit: 10, Offset: -1})
	if err == nil || !strings.Contains(err.Error(), "invalid offset") {
		t.Fatalf("expected invalid offset error, got %v", err)
	}
}
