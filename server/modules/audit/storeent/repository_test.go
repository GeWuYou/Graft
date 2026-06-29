package storeent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/config"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
	auditcontract "graft/server/modules/audit/contract"
	auditlocales "graft/server/modules/audit/locales"
	auditstore "graft/server/modules/audit/store"
)

type stubMonitorIncidentEvidenceService struct {
	resolved moduleapi.ResolvedAuditIncidentMonitorEvidence
	err      error
}

func (s stubMonitorIncidentEvidenceService) ResolveAuditIncidentMonitorEvidence(
	context.Context,
	moduleapi.ResolveAuditIncidentMonitorEvidenceInput,
) (moduleapi.ResolvedAuditIncidentMonitorEvidence, error) {
	return s.resolved, s.err
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file:audit-module-storeent?mode=memory&cache=shared")
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
		visibility TEXT NOT NULL DEFAULT 'visible',
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
	);
	CREATE TABLE audit_visibility_defaults (
		key TEXT PRIMARY KEY,
		strategy TEXT NOT NULL DEFAULT 'visible',
		updated_at DATETIME NOT NULL,
		updated_by INTEGER NULL,
		updated_by_name TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE audit_visibility_overrides (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL,
		action_key TEXT NOT NULL,
		strategy TEXT NOT NULL DEFAULT 'visible',
		description TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		created_by INTEGER NULL,
		created_by_name TEXT NOT NULL DEFAULT '',
		updated_at DATETIME NOT NULL,
		updated_by INTEGER NULL,
		updated_by_name TEXT NOT NULL DEFAULT '',
		UNIQUE(source, action_key)
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create audit schema: %v", err)
	}

	return db
}

func newTestLocalizer() *i18n.Service {
	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	resources, err := auditlocales.EmbeddedLocaleResources()
	if err != nil {
		panic(err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		panic(err)
	}
	return localizer
}

func TestRepositoryCreateAndListAuditLogs(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	actorID := uint64(7)
	firstCreatedAt := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
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
		TimePreset:   auditstore.AuditTimePresetLast30Days,
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

func TestRepositoryListAuditLogsDoesNotApplyImplicitTimePreset(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	oldCreatedAt := time.Now().UTC().Add(-45 * 24 * time.Hour)
	recentCreatedAt := time.Now().UTC().Add(-2 * time.Hour)
	for _, item := range []auditstore.CreateAuditLogInput{
		{
			Action:       "audit.old",
			ResourceType: "user",
			ResourceID:   "1",
			Success:      true,
			RequestID:    "req-old",
			CreatedAt:    oldCreatedAt,
		},
		{
			Action:       "audit.recent",
			ResourceType: "user",
			ResourceID:   "2",
			Success:      true,
			RequestID:    "req-recent",
			CreatedAt:    recentCreatedAt,
		},
	} {
		if _, err := repo.CreateAuditLog(ctx, item); err != nil {
			t.Fatalf("seed audit log: %v", err)
		}
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
		t.Fatalf("expected both old and recent logs without implicit preset, got %#v", result)
	}
}

func TestRepositoryDeleteAuditLogsBeforeDeletesOnlyOlderRecords(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	cutoff := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	for _, item := range []auditstore.CreateAuditLogInput{
		{Action: "audit.old", RequestID: "req-old", CreatedAt: cutoff.Add(-time.Second)},
		{Action: "audit.equal", RequestID: "req-equal", CreatedAt: cutoff},
		{Action: "audit.recent", RequestID: "req-recent", CreatedAt: cutoff.Add(time.Second)},
	} {
		if _, err := repo.CreateAuditLog(ctx, item); err != nil {
			t.Fatalf("seed audit log: %v", err)
		}
	}

	deleted, err := repo.DeleteAuditLogsBefore(ctx, cutoff)
	if err != nil {
		t.Fatalf("delete audit logs before cutoff: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one deleted row, got %d", deleted)
	}

	result, err := repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		Limit: 10,
		Sorts: []string{"created_at:asc"},
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("expected two remaining audit logs, got %#v", result)
	}
	if result.Items[0].RequestID != "req-equal" || result.Items[1].RequestID != "req-recent" {
		t.Fatalf("expected cutoff and recent records to remain, got %#v", result.Items)
	}
}

func TestRepositoryHighRiskOperationsIncludesContainerDangerousActions(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	for _, item := range []auditstore.CreateAuditLogInput{
		{
			Action:       "ops.container.action.restart",
			ResourceType: "container",
			ResourceID:   "abc123",
			ResourceName: "web",
			Success:      true,
			RequestID:    "req-container-danger",
			Metadata:     json.RawMessage(`{"status_code":200}`),
			CreatedAt:    now.Add(-10 * time.Minute),
		},
		{
			Action:       "ops.container.shell.session.started",
			ResourceType: "container",
			ResourceID:   "shell-1",
			ResourceName: "web",
			Success:      true,
			RequestID:    "req-shell",
			Metadata:     json.RawMessage(`{"status_code":200}`),
			CreatedAt:    now.Add(-5 * time.Minute),
		},
	} {
		if _, err := repo.CreateAuditLog(ctx, item); err != nil {
			t.Fatalf("create audit log: %v", err)
		}
	}

	assertAuditBusinessCategoryResult(
		ctx,
		t,
		repo,
		auditstore.AuditBusinessCategoryHighRiskOperations,
		"req-container-danger",
	)
}

func TestRepositoryReadAuditLogReturnsOneRecord(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	createdAt := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	created, err := repo.CreateAuditLog(ctx, auditstore.CreateAuditLogInput{
		Action:       "auth.token.expired",
		ResourceType: "auth",
		ResourceID:   "token",
		ResourceName: "access token",
		Success:      false,
		RequestID:    "req-detail",
		IP:           "127.0.0.1",
		UserAgent:    "curl/8",
		Message:      "Token expired",
		Metadata:     json.RawMessage(`{"auditSource":"SECURITY_EVENT","trace_id":"trace-detail"}`),
		CreatedAt:    createdAt,
	})
	if err != nil {
		t.Fatalf("create audit log: %v", err)
	}

	detail, err := repo.ReadAuditLog(ctx, created.ID)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	if detail.ID != created.ID || detail.RequestID != "req-detail" || detail.TraceID != "trace-detail" {
		t.Fatalf("expected detail to round-trip and enrich metadata, got %#v", detail)
	}
}

func TestDisplayTargetLabelUsesLocaleResources(t *testing.T) {
	localizer := newTestLocalizer()

	zhLabel := displayTargetLabel(WithAuditLocale(context.Background(), "zh-CN"), localizer, "AUTH")
	if zhLabel != "认证" {
		t.Fatalf("expected zh-CN AUTH label from locale resource, got %q", zhLabel)
	}

	enLabel := displayTargetLabel(WithAuditLocale(context.Background(), "en-US"), localizer, "AUTH")
	if enLabel != "Authentication" {
		t.Fatalf("expected en-US AUTH label from locale resource, got %q", enLabel)
	}
}

func TestDisplayTargetLabelUnknownTypeDoesNotFallbackToChinese(t *testing.T) {
	localizer := newTestLocalizer()

	label := displayTargetLabel(WithAuditLocale(context.Background(), "zh-CN"), localizer, "LOG_QUERY")
	if label != "" {
		t.Fatalf("expected unknown target type to keep empty localized label, got %q", label)
	}
}

func TestAuditTargetLabelKeysRegisteredInEmbeddedLocales(t *testing.T) {
	localizer := newTestLocalizer()

	for _, tc := range []struct {
		locale   i18n.LocaleTag
		key      string
		expected string
	}{
		{locale: i18n.LocaleZHCN, key: auditcontract.AuditTargetLabelUser.String(), expected: "用户"},
		{locale: i18n.LocaleENUS, key: auditcontract.AuditTargetLabelUser.String(), expected: "User"},
		{locale: i18n.LocaleZHCN, key: auditcontract.AuditTargetLabelServerStatus.String(), expected: "服务器状态"},
		{locale: i18n.LocaleENUS, key: auditcontract.AuditTargetLabelServerStatus.String(), expected: "Server Status"},
	} {
		matches := localizer.RegisteredMessageResources(tc.locale, i18n.MessageKey(tc.key))
		if len(matches) != 1 {
			t.Fatalf("expected one registered audit target label for %s %q, got %#v", tc.locale, tc.key, matches)
		}
		if matches[0].Text != tc.expected {
			t.Fatalf("expected registered audit target label %q for %s %q, got %#v", tc.expected, tc.locale, tc.key, matches[0])
		}
	}
}

func TestRepositoryReadAuditLogMapsMissingRow(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	_, err = repo.ReadAuditLog(context.Background(), 404)
	if !errors.Is(err, auditstore.ErrAuditLogNotFound) {
		t.Fatalf("expected audit log not found, got %v", err)
	}
}

func TestRepositoryListAuditLogsSupportsExplicitAscendingSort(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	base := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	for index, input := range []auditstore.CreateAuditLogInput{
		{Action: "audit.second", RequestID: "req-2", CreatedAt: base.Add(time.Minute)},
		{Action: "audit.first", RequestID: "req-1", CreatedAt: base},
	} {
		if _, err := repo.CreateAuditLog(ctx, input); err != nil {
			t.Fatalf("seed audit log %d: %v", index, err)
		}
	}

	result, err := repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		Limit: 10,
		Sorts: []string{"created_at:asc"},
	})
	if err != nil {
		t.Fatalf("list audit logs with asc sort: %v", err)
	}
	if len(result.Items) != 2 || result.Items[0].RequestID != "req-1" || result.Items[1].RequestID != "req-2" {
		t.Fatalf("expected ascending created_at order, got %#v", result.Items)
	}
}

func TestRepositoryCreateAuditLogRejectsActorUserIDOutsideBigIntRange(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	overflow := uint64(math.MaxInt64) + 1
	_, err = repo.CreateAuditLog(context.Background(), auditstore.CreateAuditLogInput{
		ActorUserID: &overflow,
		Action:      "audit.test",
		Success:     true,
		CreatedAt:   time.Now().UTC(),
	})
	if err == nil {
		t.Fatalf("expected bigint range error")
	}
	if !strings.Contains(err.Error(), "exceeds bigint range") {
		t.Fatalf("expected bigint range error, got %v", err)
	}
}

func TestRepositoryListAuditLogsRejectsActorUserIDOutsideInt64Range(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	overflow := uint64(math.MaxInt64) + 1
	_, err = repo.ListAuditLogs(context.Background(), auditstore.ListAuditLogsQuery{
		ActorUserID: &overflow,
		Limit:       10,
		Offset:      0,
	})
	if err == nil {
		t.Fatalf("expected int64 range error")
	}
	if !strings.Contains(err.Error(), "exceeds int64 range") {
		t.Fatalf("expected int64 range error, got %v", err)
	}
}

func TestRepositoryListAuditLogsSupportsActionPrefix(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	seed := []auditstore.CreateAuditLogInput{
		{
			Action:       "rbac.role.create",
			ResourceType: "role",
			ResourceID:   "1",
			ResourceName: "ops-admin",
			Success:      true,
			RequestID:    "req-rbac-role",
			Message:      "role created",
			Metadata:     json.RawMessage(`{"request_path":"/api/roles","status_code":200}`),
			CreatedAt:    now.Add(-2 * time.Hour),
		},
		{
			Action:       "rbac.user.roles.add",
			ResourceType: "user",
			ResourceID:   "9",
			ResourceName: "alice",
			Success:      true,
			RequestID:    "req-rbac-user-role",
			Message:      "user roles added",
			Metadata:     json.RawMessage(`{"request_path":"/api/users/9/roles/add","status_code":200}`),
			CreatedAt:    now.Add(-time.Hour),
		},
		{
			Action:       "auth.permission.denied",
			ResourceType: "role",
			ResourceID:   "12",
			ResourceName: "Ops Admin",
			Success:      false,
			RequestID:    "req-authz",
			Message:      "common.forbidden",
			Metadata:     json.RawMessage(`{"request_path":"/api/roles/12/delete","status_code":403}`),
			CreatedAt:    now.Add(-30 * time.Minute),
		},
	}
	for _, item := range seed {
		if _, err := repo.CreateAuditLog(ctx, item); err != nil {
			t.Fatalf("seed audit log: %v", err)
		}
	}

	result, err := repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		ActionPrefix: "rbac.",
		TimePreset:   auditstore.AuditTimePresetLast30Days,
		Limit:        10,
		Offset:       0,
	})
	if err != nil {
		t.Fatalf("list prefixed audit logs: %v", err)
	}

	if result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("expected two rbac audit logs, got %#v", result)
	}
	if !strings.HasPrefix(result.Items[0].Action, "rbac.") || !strings.HasPrefix(result.Items[1].Action, "rbac.") {
		t.Fatalf("expected rbac-prefixed actions, got %#v", result.Items)
	}
}

func TestRepositoryListAuditLogsAppliesFilters(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	success := true
	now := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
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
		Action:     "user.update",
		Success:    &success,
		RequestID:  "req-keep",
		TimePreset: auditstore.AuditTimePresetLast30Days,
		Limit:      10,
		Offset:     0,
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
	repo, err := NewRepository(db, newTestLocalizer(), nil)
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

func TestRepositoryListAuditLogsSupportsCanonicalFilters(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	seedAuditOverviewDrilldownLogs(ctx, t, repo, now)

	result, err := repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		TimePreset: auditstore.AuditTimePresetLast24Hours,
		Keyword:    "ops-admin",
		Limit:      20,
		Offset:     0,
	})
	if err != nil {
		t.Fatalf("keyword filter: list audit logs: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].RequestID != "req-denied" {
		t.Fatalf("keyword filter: unexpected result %#v", result)
	}

	result, err = repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		TimePreset: auditstore.AuditTimePresetLast24Hours,
		Actor:      "alice",
		Limit:      20,
		Offset:     0,
	})
	if err != nil {
		t.Fatalf("actor filter: list audit logs: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].RequestID != "req-reset" {
		t.Fatalf("actor filter: unexpected result %#v", result)
	}

	result, err = repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		TimePreset: auditstore.AuditTimePresetLast24Hours,
		SessionID:  "session-1",
		Limit:      20,
		Offset:     0,
	})
	if err != nil {
		t.Fatalf("session filter: list audit logs: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].RequestID != "req-auth" {
		t.Fatalf("session filter: unexpected result %#v", result)
	}

}

func TestRepositoryListAuditLogsSupportsOverviewBusinessCategories(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	seedAuditOverviewDrilldownLogs(ctx, t, repo, now)

	assertAuditBusinessCategoryResult(
		ctx,
		t,
		repo,
		auditstore.AuditBusinessCategoryPermissionDenials,
		"req-denied",
	)
	assertAuditBusinessCategoryResult(ctx, t, repo, auditstore.AuditBusinessCategoryAuthFailures, "req-auth")
}

func assertAuditBusinessCategoryResult(
	ctx context.Context,
	t *testing.T,
	repo auditstore.AuditRepository,
	category auditstore.AuditBusinessCategory,
	wantRequestID string,
) {
	t.Helper()

	result, err := repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		TimePreset:       auditstore.AuditTimePresetLast24Hours,
		BusinessCategory: category,
		Limit:            20,
		Offset:           0,
	})
	if err != nil {
		t.Fatalf("%s business category: list audit logs: %v", category, err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].RequestID != wantRequestID {
		t.Fatalf("%s business category: unexpected result %#v", category, result)
	}
}

func seedAuditOverviewDrilldownLogs(
	ctx context.Context,
	t *testing.T,
	repo auditstore.AuditRepository,
	now time.Time,
) {
	t.Helper()

	seed := []auditstore.CreateAuditLogInput{
		{
			ActorUsername:    "alice",
			ActorDisplayName: "Alice",
			Action:           "user.password.reset",
			ResourceType:     "user",
			ResourceID:       "7",
			ResourceName:     "Alice",
			Success:          true,
			RequestID:        "req-reset",
			Message:          "password reset",
			Metadata:         json.RawMessage(`{"request_path":"/api/users/7/reset-password","status_code":200}`),
			CreatedAt:        now.Add(-20 * time.Minute),
		},
		{
			Action:       "auth.login_failed",
			ResourceType: "auth",
			ResourceID:   "session-1",
			ResourceName: "login",
			Success:      false,
			RequestID:    "req-auth",
			Message:      "common.invalid_argument",
			Metadata:     json.RawMessage(`{"request_path":"/api/auth/login","status_code":401,"session_id":"session-1"}`),
			CreatedAt:    now.Add(-15 * time.Minute),
		},
		{
			Action:       "rbac.role.delete",
			ResourceType: "role",
			ResourceID:   "12",
			ResourceName: "ops-admin",
			Success:      false,
			RequestID:    "req-denied",
			Message:      "common.forbidden",
			Metadata:     json.RawMessage(`{"request_path":"/api/roles/12/delete","status_code":403}`),
			CreatedAt:    now.Add(-10 * time.Minute),
		},
	}

	for _, item := range seed {
		if _, err := repo.CreateAuditLog(ctx, item); err != nil {
			t.Fatalf("seed audit log: %v", err)
		}
	}
}

func TestRepositoryReadIncidentCorrelatesBoundedContext(t *testing.T) {
	db := openTestDB(t)
	base := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	repo, err := NewRepository(db, newTestLocalizer(), stubMonitorIncidentEvidenceService{
		resolved: moduleapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: moduleapi.MonitorEvidenceAvailable,
			Summary:      "CPU pressure matched the bounded incident window.",
			AnomalyKey:   "resource_cpu_pressure",
			ScopeKind:    "resource",
			ScopeRef:     "runtime.cpu",
			ObservedAt:   timePointer(base.Add(4 * time.Minute)),
			EvidenceLinks: []moduleapi.MonitorEvidenceLink{
				{
					TargetKind: "audit_context",
					LinkState:  "available",
					Title:      "Review related audit activity",
					TimeWindow: &moduleapi.MonitorEvidenceLinkTimeWindow{
						CreatedFrom: base.Add(-5 * time.Minute),
						CreatedTo:   base.Add(4 * time.Minute),
					},
					AuditContext: &moduleapi.MonitorAuditEvidenceContext{
						RequestID:    "req-incident",
						ResourceType: "runtime",
						CreatedFrom:  timePointer(base.Add(-5 * time.Minute)),
						CreatedTo:    timePointer(base.Add(4 * time.Minute)),
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	actorID := uint64(7)
	seed, err := repo.CreateAuditLog(ctx, auditstore.CreateAuditLogInput{
		ActorUserID:      &actorID,
		ActorUsername:    "alice",
		ActorDisplayName: "Alice",
		Action:           "auth.permission.denied",
		ResourceType:     "role",
		ResourceID:       "9",
		ResourceName:     "ops-admin",
		Success:          false,
		RequestID:        "req-incident",
		Message:          "common.forbidden",
		Metadata:         json.RawMessage(`{"auditSource":"SECURITY_EVENT","status_code":403,"session_id":"sess-1","trace_id":"trace-1"}`),
		CreatedAt:        base,
	})
	if err != nil {
		t.Fatalf("create seed log: %v", err)
	}

	for _, item := range []auditstore.CreateAuditLogInput{
		{
			ActorUserID:      &actorID,
			ActorUsername:    "alice",
			ActorDisplayName: "Alice",
			Action:           "rbac.role.delete",
			ResourceType:     "role",
			ResourceID:       "9",
			ResourceName:     "ops-admin",
			Success:          false,
			RequestID:        "req-incident",
			Message:          "delete denied",
			Metadata:         json.RawMessage(`{"status_code":403,"session_id":"sess-1","trace_id":"trace-1"}`),
			CreatedAt:        base.Add(2 * time.Minute),
		},
		{
			Action:       "user.update",
			ResourceType: "user",
			ResourceID:   "20",
			Success:      true,
			RequestID:    "req-other",
			Message:      "outside correlation",
			Metadata:     json.RawMessage(`{"status_code":200}`),
			CreatedAt:    base.Add(90 * time.Minute),
		},
	} {
		if _, err := repo.CreateAuditLog(ctx, item); err != nil {
			t.Fatalf("seed incident context log: %v", err)
		}
	}

	incident, err := repo.ReadIncident(ctx, seed.ID)
	if err != nil {
		t.Fatalf("read incident: %v", err)
	}
	assertIncidentCorrelation(t, incident, seed.ID)
}

func assertIncidentCorrelation(t *testing.T, incident auditstore.AuditIncident, seedID uint64) {
	t.Helper()

	if incident.SeedEvent.ID != seedID {
		t.Fatalf("expected seed event id %d, got %d", seedID, incident.SeedEvent.ID)
	}
	if incident.Incident.IncidentKey != "incident:req:req-incident" {
		t.Fatalf("unexpected incident key %q", incident.Incident.IncidentKey)
	}
	if len(incident.RelatedEvents) != 2 {
		t.Fatalf("expected two bounded related events, got %d", len(incident.RelatedEvents))
	}
	if len(incident.RelatedActors) != 1 || incident.RelatedActors[0].EventCount != 2 {
		t.Fatalf("expected one correlated actor summary, got %#v", incident.RelatedActors)
	}
	if len(incident.RelatedResources) != 1 || incident.RelatedResources[0].ResourceID != "9" {
		t.Fatalf("expected one correlated resource summary, got %#v", incident.RelatedResources)
	}
	if len(incident.RelatedRequests) != 1 || incident.RelatedRequests[0].RequestID != "req-incident" {
		t.Fatalf("expected one correlated request summary, got %#v", incident.RelatedRequests)
	}
	if incident.MonitorContext.State != auditstore.MonitorContextStateAvailable {
		t.Fatalf("expected monitor context to be available, got %#v", incident.MonitorContext)
	}
	if incident.MonitorContext.AnomalyKey != "resource_cpu_pressure" {
		t.Fatalf("unexpected monitor anomaly key %#v", incident.MonitorContext)
	}
	if incident.MonitorContext.ObservedAt == nil {
		t.Fatalf("expected observed_at to be attached, got %#v", incident.MonitorContext)
	}
}

func TestBuildAuditTargetPromotesIncidentTargets(t *testing.T) {
	record := auditstore.AuditLog{
		ID:           42,
		Source:       auditstore.AuditSourceSecurityEvent,
		Action:       "auth.failed",
		ResourceType: "AUTH",
		ResourceID:   "console",
		ResourceName: "Console",
		Result:       auditstore.AuditResultFailed,
		RiskLevel:    auditstore.AuditRiskLevelHigh,
		TargetType:   "AUTH",
		TargetLabel:  "Console",
	}

	target := buildAuditTarget(record)

	if target.Kind != "incident" {
		t.Fatalf("expected incident target kind, got %#v", target)
	}
	if target.ID != "42" {
		t.Fatalf("expected incident target id 42, got %#v", target)
	}
	if target.RouteRef != "/incidents/42" {
		t.Fatalf("expected canonical incident route ref, got %#v", target)
	}
}

func TestRepositoryReadAuditOverview(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
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
		{
			Action:       "POST /api/auth/refresh",
			ResourceType: "auth",
			Success:      false,
			RequestID:    "req-malformed",
			Message:      "refresh failed",
			Metadata:     json.RawMessage(`{"request_path":"/api/auth/refresh","status_code":"oops","error":"token refresh failed"}`),
			CreatedAt:    now.Add(-15 * time.Minute),
		},
	}
	for _, item := range seed {
		if _, err := repo.CreateAuditLog(ctx, item); err != nil {
			t.Fatalf("seed audit log: %v", err)
		}
	}

	overview, err := repo.ReadAuditOverview(ctx, auditstore.AuditTimePresetLast24Hours)
	if err != nil {
		t.Fatalf("read audit overview: %v", err)
	}

	assertOverviewSummary(t, overview)
}

func TestRepositoryReadAuditOverviewDoesNotFallbackPermissionDeniedToFailedAuth(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := repo.CreateAuditLog(ctx, auditstore.CreateAuditLogInput{
		Action:       "auth.login_failed",
		ResourceType: "auth",
		ResourceID:   "session-1",
		ResourceName: "login",
		Success:      false,
		RequestID:    "req-auth",
		Message:      "auth.token_expired",
		Metadata:     json.RawMessage(`{"request_path":"/api/auth/login","status_code":401,"session_id":"session-1"}`),
		CreatedAt:    now.Add(-15 * time.Minute),
	}); err != nil {
		t.Fatalf("seed auth audit log: %v", err)
	}

	overview, err := repo.ReadAuditOverview(ctx, auditstore.AuditTimePresetLast24Hours)
	if err != nil {
		t.Fatalf("read audit overview: %v", err)
	}
	if len(overview.FailedAuth) != 1 {
		t.Fatalf("expected failed auth item, got %#v", overview.FailedAuth)
	}
	if len(overview.PermissionDenied) != 0 {
		t.Fatalf("permission denied items must not fallback to failed auth, got %#v", overview.PermissionDenied)
	}
}

func assertOverviewSummary(t *testing.T, overview auditstore.AuditOverview) {
	t.Helper()

	if overview.TimePreset != auditstore.AuditTimePresetLast24Hours {
		t.Fatalf("expected last_24h preset, got %q", overview.TimePreset)
	}
	if overview.Summary.TotalLogs != 4 || overview.Summary.FailedOperations != 3 {
		t.Fatalf("unexpected overview summary: %#v", overview.Summary)
	}
	if overview.Summary.HighRiskEvents != 4 || overview.Summary.SensitiveOperations != 2 {
		t.Fatalf("unexpected risk counters: %#v", overview.Summary)
	}
	if len(overview.FailedAuth) != 2 || overview.FailedAuth[0].RequestID != "req-malformed" || overview.FailedAuth[1].RequestID != "req-auth" {
		t.Fatalf("unexpected failed auth items: %#v", overview.FailedAuth)
	}
	if len(overview.PermissionDenied) != 1 || overview.PermissionDenied[0].RequestID != "req-role" {
		t.Fatalf("unexpected permission denied items: %#v", overview.PermissionDenied)
	}
	if len(overview.SensitiveOps) != 2 {
		t.Fatalf("unexpected sensitive ops items: %#v", overview.SensitiveOps)
	}
	riskCounts := map[string]int{}
	for _, item := range overview.RiskGroups {
		riskCounts[item.Key] = item.Count
	}
	if riskCounts["critical_security"] != 1 || riskCounts["auth_failures"] != 2 {
		t.Fatalf("unexpected risk groups: %#v", overview.RiskGroups)
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func TestRepositoryListAuditPolicyRulesOrdersByPriority(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewRepository(db, newTestLocalizer(), nil)
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
		"failed auth":       authFailuresWhereClause(),
		"permission denied": permissionDenialsWhereClause(),
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
	if !strings.Contains(clause, "{1,5}") {
		t.Fatalf("expected bounded status_code digit guard in audit result clause, got %s", clause)
	}
}

func TestAuditResultExpressionsUseSharedClassificationBranches(t *testing.T) {
	for _, tc := range []struct {
		name       string
		expression string
	}{
		{name: "portable", expression: auditResultExpression()},
		{name: "postgres", expression: auditResultPostgresExpression()},
	} {
		if !strings.Contains(tc.expression, "WHEN success THEN 'SUCCESS'") {
			t.Fatalf("%s expression missing success branch: %s", tc.name, tc.expression)
		}
		for _, branch := range []string{
			"= '403' THEN 'DENIED'",
			"OR COALESCE(metadata ->> 'error_kind', '') = 'system'",
			"OR COALESCE(metadata ->> 'error', '') <> '' THEN 'ERROR'",
			"ELSE 'FAILED'",
		} {
			if !strings.Contains(tc.expression, branch) {
				t.Fatalf("%s expression missing shared branch %q: %s", tc.name, branch, tc.expression)
			}
		}
	}
	if !strings.Contains(auditResultExpression(), "THEN CAST(metadata ->> 'status_code' AS INTEGER)") {
		t.Fatalf("expected portable result expression to keep guarded cast branch, got %s", auditResultExpression())
	}
	if !strings.Contains(auditResultExpression(), "LENGTH(metadata ->> 'status_code') <= 5") {
		t.Fatalf("expected portable result expression to bound digit length, got %s", auditResultExpression())
	}
	if !strings.Contains(auditResultPostgresExpression(), "(metadata ->> 'status_code')::int >= 500") {
		t.Fatalf("expected postgres result expression to keep postgres cast branch, got %s", auditResultPostgresExpression())
	}
	if !strings.Contains(auditResultPostgresExpression(), "{1,5}") {
		t.Fatalf("expected postgres result expression to bound digit length, got %s", auditResultPostgresExpression())
	}
}

func TestAuditRiskLevelExpressionsUseSharedClassificationBranches(t *testing.T) {
	for _, tc := range []struct {
		name       string
		expression string
	}{
		{name: "portable", expression: auditRiskLevelExpression()},
		{name: "postgres", expression: auditRiskLevelPostgresExpression()},
	} {
		for _, branch := range []string{
			"= '403'",
			"OR COALESCE(metadata ->> 'error_kind', '') = 'system'",
			"OR COALESCE(metadata ->> 'error', '') <> ''",
			"THEN 'CRITICAL'",
			"LOWER(TRIM(action)) LIKE '%%reset_password%%'",
			"LOWER(TRIM(resource_type)) = 'container'",
			"LOWER(TRIM(action)) LIKE 'ops.container.action.%'",
			"THEN 'HIGH'",
			"LOWER(TRIM(action)) LIKE '%%login_failed%%'",
			"THEN 'MEDIUM'",
			"ELSE 'LOW'",
		} {
			if !strings.Contains(tc.expression, branch) {
				t.Fatalf("%s expression missing shared branch %q: %s", tc.name, branch, tc.expression)
			}
		}
	}
	if !strings.Contains(auditRiskLevelExpression(), "THEN CAST(metadata ->> 'status_code' AS INTEGER)") {
		t.Fatalf("expected portable risk expression to keep guarded cast branch, got %s", auditRiskLevelExpression())
	}
	if !strings.Contains(auditRiskLevelExpression(), "LENGTH(metadata ->> 'status_code') <= 5") {
		t.Fatalf("expected portable risk expression to bound digit length, got %s", auditRiskLevelExpression())
	}
	if !strings.Contains(auditRiskLevelPostgresExpression(), "(metadata ->> 'status_code')::int >= 500") {
		t.Fatalf("expected postgres risk expression to keep postgres cast branch, got %s", auditRiskLevelPostgresExpression())
	}
	if !strings.Contains(auditRiskLevelPostgresExpression(), "{1,5}") {
		t.Fatalf("expected postgres risk expression to bound digit length, got %s", auditRiskLevelPostgresExpression())
	}
}

func TestAuditResultAndRiskClassifiersCoverRepresentativeInputs(t *testing.T) {
	tests := []struct {
		name         string
		success      bool
		action       string
		resourceType string
		metadata     map[string]string
		wantResult   string
		wantRisk     string
	}{
		{
			name:         "success",
			success:      true,
			action:       "user.update",
			resourceType: "user",
			metadata:     map[string]string{"status_code": "200"},
			wantResult:   "SUCCESS",
			wantRisk:     "LOW",
		},
		{
			name:         "denied",
			success:      false,
			action:       "rbac.role.delete",
			resourceType: "role",
			metadata:     map[string]string{"status_code": "403"},
			wantResult:   "DENIED",
			wantRisk:     "CRITICAL",
		},
		{
			name:         "server error",
			success:      false,
			action:       "audit.export",
			resourceType: "audit",
			metadata:     map[string]string{"status_code": "500"},
			wantResult:   "ERROR",
			wantRisk:     "CRITICAL",
		},
		{
			name:         "system error kind",
			success:      false,
			action:       "audit.export",
			resourceType: "audit",
			metadata:     map[string]string{"error_kind": "system"},
			wantResult:   "ERROR",
			wantRisk:     "CRITICAL",
		},
		{
			name:         "failed without error metadata",
			success:      false,
			action:       "audit.export",
			resourceType: "audit",
			metadata:     map[string]string{"status_code": "400"},
			wantResult:   "FAILED",
			wantRisk:     "HIGH",
		},
		{
			name:         "critical action",
			success:      true,
			action:       "user.reset_password",
			resourceType: "user",
			metadata:     map[string]string{"status_code": "200"},
			wantResult:   "SUCCESS",
			wantRisk:     "CRITICAL",
		},
		{
			name:         "medium auth action",
			success:      true,
			action:       "auth.login",
			resourceType: "auth",
			metadata:     map[string]string{"status_code": "200"},
			wantResult:   "SUCCESS",
			wantRisk:     "MEDIUM",
		},
		{
			name:         "container dangerous action trims runtime values",
			success:      true,
			action:       "  OPS.CONTAINER.ACTION.START  ",
			resourceType: "  Container_Batch  ",
			metadata:     map[string]string{"status_code": "200"},
			wantResult:   "SUCCESS",
			wantRisk:     "HIGH",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertAuditClassifierExpressions(t, tc)
			assertAuditClassifierRuntime(t, tc)
		})
	}
}

func TestOverviewSummaryUsesCanonicalResultAndRiskExpressions(t *testing.T) {
	if !strings.Contains(overviewSummarySQL, "IN ('FAILED', 'DENIED', 'ERROR')") {
		t.Fatalf("expected failed operations to use normalized non-success results, got %s", overviewSummarySQL)
	}
	if !strings.Contains(overviewSummarySQL, "IN ('HIGH', 'CRITICAL')") {
		t.Fatalf("expected high-risk events to use normalized risk levels, got %s", overviewSummarySQL)
	}
	if strings.Contains(overviewSummarySQL, "WHERE success = false) AS failed_operations") {
		t.Fatalf("failed operations must not collapse to raw success=false, got %s", overviewSummarySQL)
	}
}

func literalMetadataExpressionBuilderForTest(metadata map[string]string) auditMetadataExpressionBuilder {
	return auditMetadataExpressionBuilder{
		textValue: func(_ string, key string) string {
			return sqlStringLiteralForTest(metadata[key])
		},
		numericAtLeast: func(_ string, key string, threshold int) string {
			value, err := strconv.Atoi(metadata[key])
			if err != nil || value < threshold {
				return "0 = 1"
			}
			return "1 = 1"
		},
	}
}

func assertAuditClassifierExpressions(
	t *testing.T,
	tc struct {
		name         string
		success      bool
		action       string
		resourceType string
		metadata     map[string]string
		wantResult   string
		wantRisk     string
	},
) {
	t.Helper()

	for _, builder := range []struct {
		name    string
		builder auditMetadataExpressionBuilder
	}{
		{name: "portable", builder: literalMetadataExpressionBuilderForTest(tc.metadata)},
		{name: "postgres", builder: literalMetadataExpressionBuilderForTest(tc.metadata)},
	} {
		resultExpression := auditResultExpressionWith(
			sqlBoolLiteralForTest(tc.success),
			"metadata",
			builder.builder,
		)
		if got := evalStringExpressionForTest(t, resultExpression); got != tc.wantResult {
			t.Fatalf("%s result classifier mismatch: got %s want %s", builder.name, got, tc.wantResult)
		}

		riskExpression := auditRiskLevelExpressionWith(
			sqlBoolLiteralForTest(tc.success),
			sqlStringLiteralForTest(tc.action),
			sqlStringLiteralForTest(tc.resourceType),
			"metadata",
			builder.builder,
		)
		if got := evalStringExpressionForTest(t, riskExpression); got != tc.wantRisk {
			t.Fatalf("%s risk classifier mismatch: got %s want %s", builder.name, got, tc.wantRisk)
		}
	}
}

func assertAuditClassifierRuntime(
	t *testing.T,
	tc struct {
		name         string
		success      bool
		action       string
		resourceType string
		metadata     map[string]string
		wantResult   string
		wantRisk     string
	},
) {
	t.Helper()

	metadataAny := make(map[string]any, len(tc.metadata))
	for key, value := range tc.metadata {
		metadataAny[key] = value
	}
	record := auditstore.AuditLog{
		Success:      tc.success,
		Action:       tc.action,
		ResourceType: tc.resourceType,
	}
	record.Result = classifyAuditResult(record, metadataAny)
	if got := string(record.Result); got != tc.wantResult {
		t.Fatalf("runtime result classifier mismatch: got %s want %s", got, tc.wantResult)
	}
	if got := string(classifyAuditRiskLevel(record)); got != tc.wantRisk {
		t.Fatalf("runtime risk classifier mismatch: got %s want %s", got, tc.wantRisk)
	}
}

func evalStringExpressionForTest(t *testing.T, expression string) string {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open expression db: %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close expression db: %v", closeErr)
		}
	}()

	var got string
	if err := db.QueryRow("SELECT " + expression).Scan(&got); err != nil {
		t.Fatalf("evaluate expression %s: %v", expression, err)
	}
	return got
}

func sqlBoolLiteralForTest(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func sqlStringLiteralForTest(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func TestOverviewTrendUsesCanonicalResultAndRiskExpressions(t *testing.T) {
	resultExpression := auditOverviewTrendResultExpression()
	riskExpression := auditOverviewTrendRiskLevelExpression()
	if !strings.Contains(resultExpression, "logs.metadata ->> 'status_code'") ||
		!strings.Contains(riskExpression, "LOWER(TRIM(logs.action))") ||
		!strings.Contains(riskExpression, "LOWER(TRIM(logs.resource_type))") {
		t.Fatalf("expected trend expressions to use table-qualified canonical fields, result=%s risk=%s", resultExpression, riskExpression)
	}
	trendSQL := overviewTrendSeriesSQL("1 hour")
	if !strings.Contains(trendSQL, "logs.id IS NOT NULL") {
		t.Fatalf("expected trend counters to ignore empty left-join buckets")
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
	if !strings.Contains(clause, "{1,5}") {
		t.Fatalf("expected bounded status_code digit guard in risk level clause, got %s", clause)
	}
}

func TestOverviewRiskGroupsSQLBoundsStatusCodeDigits(t *testing.T) {
	if !strings.Contains(overviewRiskGroupsSQL, "LENGTH(metadata ->> 'status_code') <= 5") {
		t.Fatalf("expected overview risk groups SQL to bound digit length, got %s", overviewRiskGroupsSQL)
	}
}

func TestOverviewRecentWhereClauseUsesAllowlist(t *testing.T) {
	if got := overviewRecentWhereClause(overviewRecentKindAuthFailures); got != authFailuresWhereClause() {
		t.Fatalf("unexpected auth failures clause %q", got)
	}
	if got := overviewRecentWhereClause(overviewRecentKindPermissionDenials); got != permissionDenialsWhereClause() {
		t.Fatalf("unexpected permission denials clause %q", got)
	}
	if got := overviewRecentWhereClause(overviewRecentKindSensitiveOperations); got != sensitiveOperationsWhereClause() {
		t.Fatalf("unexpected sensitive operations clause %q", got)
	}
}

func TestNormalizeAuditTargetTypeTreatsWhitespaceAsAudit(t *testing.T) {
	if got := normalizeAuditTargetType("   "); got != "AUDIT" {
		t.Fatalf("expected whitespace resource type to fall back to AUDIT, got %q", got)
	}
}

func TestSummarizeIncidentActorsUsesStableUniqueTieBreaker(t *testing.T) {
	actorIDSeven := uint64(7)
	actorIDNine := uint64(9)
	actors := summarizeIncidentActors([]auditstore.AuditLog{
		{ActorUserID: &actorIDNine, ActorUsername: "same", ActorDisplayName: "same"},
		{ActorUserID: &actorIDSeven, ActorUsername: "same", ActorDisplayName: "same"},
	})
	if len(actors) != 2 {
		t.Fatalf("expected two actor summaries, got %#v", actors)
	}
	if actors[0].ActorUserID == nil || *actors[0].ActorUserID != actorIDSeven {
		t.Fatalf("expected stable tie-breaker to sort by unique actor identity, got %#v", actors)
	}
}

func TestBuildAuditLogFiltersUsesSingleBackslashLikeEscape(t *testing.T) {
	whereSQL, args := buildAuditLogFilters(auditstore.ListAuditLogsQuery{
		ActionPrefix:        `audit\prefix`,
		ActionKeywords:      []string{`grant_%`},
		RequestPathPrefixes: []string{`/api/a_b%`},
		Limit:               20,
		Offset:              0,
	})

	if strings.Contains(whereSQL, `ESCAPE '\\\\'`) {
		t.Fatalf("unexpected doubled escape clause in where SQL: %s", whereSQL)
	}
	if count := strings.Count(whereSQL, sqlLikeEscapeClause); count != 3 {
		t.Fatalf("expected three single-backslash escape clauses, got %d in %s", count, whereSQL)
	}

	wantArgs := []string{
		string(auditstore.AuditVisibilityStrategyVisible),
		`audit\\prefix%`,
		`%grant\_\%%`,
		`/api/a\_b\%%`,
	}
	if len(args) < len(wantArgs) {
		t.Fatalf("expected at least %d args, got %d (%#v)", len(wantArgs), len(args), args)
	}
	for index, want := range wantArgs {
		got, ok := args[index].(string)
		if !ok {
			t.Fatalf("expected string arg at index %d, got %#v", index, args[index])
		}
		if got != want {
			t.Fatalf("unexpected arg at index %d: got %q want %q", index, got, want)
		}
	}
}
