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
	);
	CREATE TABLE audit_policy_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL DEFAULT '',
		enabled BOOLEAN NOT NULL DEFAULT 1,
		priority INTEGER NOT NULL DEFAULT 100,
		effect TEXT NOT NULL,
		match_type TEXT NOT NULL DEFAULT 'exact',
		method TEXT NOT NULL DEFAULT '',
		path_pattern TEXT NOT NULL DEFAULT '',
		event_type TEXT NOT NULL DEFAULT '',
		risk_level TEXT NOT NULL DEFAULT '',
		target_type TEXT NOT NULL DEFAULT '',
		condition_expr TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
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

func TestRepositoryReadAuditOverview(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	seed := []auditstore.CreateAuditLogInput{
		{
			Action:       "POST /api/auth/login",
			ResourceType: "auth",
			Success:      false,
			RequestID:    "req-auth",
			Message:      "common.invalid_argument",
			Metadata:     json.RawMessage(`{"request_path":"/api/auth/login","status_code":400}`),
			CreatedAt:    now.Add(-2 * time.Hour),
		},
		{
			Action:       "rbac.role.delete",
			ResourceType: "role",
			ResourceID:   "12",
			ResourceName: "Ops Admin",
			Success:      false,
			RequestID:    "req-role",
			Message:      "common.forbidden",
			Metadata:     json.RawMessage(`{"request_path":"/api/roles/12/delete","status_code":403}`),
			CreatedAt:    now.Add(-time.Hour),
		},
		{
			Action:       "user.password.reset",
			ResourceType: "user",
			ResourceID:   "42",
			ResourceName: "alice",
			Success:      true,
			RequestID:    "req-user",
			Message:      "",
			Metadata:     json.RawMessage(`{"request_path":"/api/users/42/reset-password","status_code":200}`),
			CreatedAt:    now.Add(-30 * time.Minute),
		},
	}
	for _, item := range seed {
		if _, err := repo.CreateAuditLog(ctx, item); err != nil {
			t.Fatalf("seed audit log: %v", err)
		}
	}

	overview, err := repo.ReadAuditOverview(ctx, auditstore.OverviewWindow24Hours)
	if err != nil {
		t.Fatalf("read audit overview: %v", err)
	}

	if overview.Window != auditstore.OverviewWindow24Hours {
		t.Fatalf("expected 24h window, got %q", overview.Window)
	}
	if overview.Summary.TotalLogs != 3 || overview.Summary.FailedOperations != 2 {
		t.Fatalf("unexpected overview summary: %#v", overview.Summary)
	}
	if overview.Summary.HighRiskEvents != 3 || overview.Summary.SensitiveOperations != 2 {
		t.Fatalf("unexpected risk counters: %#v", overview.Summary)
	}
	if len(overview.FailedAuth) != 1 || overview.FailedAuth[0].RequestID != "req-auth" {
		t.Fatalf("unexpected failed auth items: %#v", overview.FailedAuth)
	}
	if len(overview.PermissionDenied) != 1 || overview.PermissionDenied[0].RequestID != "req-role" {
		t.Fatalf("unexpected permission denied items: %#v", overview.PermissionDenied)
	}
	if len(overview.SensitiveOps) != 2 {
		t.Fatalf("unexpected sensitive ops items: %#v", overview.SensitiveOps)
	}
}

func TestRepositoryListAuditPolicyRulesOrdersByPriority(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	now := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	if _, err := db.Exec(`
		INSERT INTO audit_policy_rules (
			name, description, source, enabled, priority, effect, match_type, method, path_pattern, event_type, risk_level, target_type, condition_expr, created_at, updated_at
		) VALUES
			('later', '', 'REQUEST', 1, 20, 'include', 'exact', 'GET', '/api/z', '', '', '', '', ?, ?),
			('first', '', 'DOMAIN_EVENT', 1, 10, 'include', 'exact', '', '', 'user.create', '', '', '', ?, ?)
	`, now, now, now, now); err != nil {
		t.Fatalf("seed policy rules: %v", err)
	}

	rules, err := repo.ListAuditPolicyRules(context.Background())
	if err != nil {
		t.Fatalf("list audit policy rules: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Name != "first" || rules[0].Source != auditstore.AuditSourceDomainEvent {
		t.Fatalf("unexpected first rule %#v", rules[0])
	}
	if rules[1].Method != "GET" || rules[1].PathPattern != "/api/z" {
		t.Fatalf("unexpected second rule %#v", rules[1])
	}
}

func TestOverviewSQLUsesPostgresJSONBExtraction(t *testing.T) {
	for name, clause := range map[string]string{
		"failed auth":       overviewFailedAuthWhere,
		"permission denied": overviewPermissionDeniedWhere,
	} {
		if strings.Contains(clause, "json_extract(") {
			t.Fatalf("%s clause should not use sqlite json_extract: %s", name, clause)
		}
		if !strings.Contains(clause, "metadata ->>") {
			t.Fatalf("%s clause should use postgres jsonb text extraction: %s", name, clause)
		}
	}
}

func TestAuditResultWhereClauseHandlesServerErrors(t *testing.T) {
	clause := auditResultWhereClause()
	if !strings.Contains(clause, "(metadata ->> 'status_code')::int >= 500") {
		t.Fatalf("expected 5xx branch in audit result clause, got %s", clause)
	}
}

func TestRiskLevelWhereClauseKeepsEscapedLikePatterns(t *testing.T) {
	clause := riskLevelWhereClause()
	if !strings.Contains(clause, "LIKE '%%delete%%'") {
		t.Fatalf("expected escaped LIKE wildcard in risk level clause, got %s", clause)
	}
	if !strings.Contains(clause, "(metadata ->> 'status_code')::int >= 500") {
		t.Fatalf("expected 5xx branch in risk level clause, got %s", clause)
	}
}
