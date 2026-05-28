// Package storeent 提供 audit 插件基于 SQL 的 repository 实现。
package storeent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	auditstore "graft/server/plugins/audit/store"
)

type repository struct {
	db *sql.DB
}

const defaultFilterCapacity = 8
const paginationParamCount = 2
const overviewRecentLimit = 3
const httpStatusForbidden = 403

var sensitiveAuditActionKeywords = []string{"delete", "reset", "grant", "assign", "revoke", "remove", "replace", "update_role", "update_permission"}

// NewRepository 基于共享连接池构建 audit 插件的 SQL repository。
func NewRepository(db *sql.DB) (auditstore.AuditRepository, error) {
	if db == nil {
		return nil, errors.New("audit repository requires a non-nil sql db")
	}

	return &repository{db: db}, nil
}

// CreateAuditLog 持久化一条审计日志记录。
func (r *repository) CreateAuditLog(ctx context.Context, input auditstore.CreateAuditLogInput) (auditstore.AuditLog, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditLog{}, errors.New("audit repository is unavailable")
	}

	metadata := cloneRawMessage(input.Metadata)
	record := auditstore.AuditLog{
		ActorUserID:      input.ActorUserID,
		ActorUsername:    input.ActorUsername,
		ActorDisplayName: input.ActorDisplayName,
		Action:           input.Action,
		ResourceType:     input.ResourceType,
		ResourceID:       input.ResourceID,
		ResourceName:     input.ResourceName,
		Success:          input.Success,
		RequestID:        input.RequestID,
		IP:               input.IP,
		UserAgent:        input.UserAgent,
		Message:          input.Message,
		Metadata:         metadata,
		CreatedAt:        input.CreatedAt,
	}

	row := r.db.QueryRowContext(
		ctx,
		`INSERT INTO audit_logs (
			actor_user_id,
			actor_username,
			actor_display_name,
			action,
			resource_type,
			resource_id,
			resource_name,
			success,
			request_id,
			ip,
			user_agent,
			message,
			metadata,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`,
		nullableUint64(input.ActorUserID),
		input.ActorUsername,
		input.ActorDisplayName,
		input.Action,
		input.ResourceType,
		input.ResourceID,
		input.ResourceName,
		input.Success,
		input.RequestID,
		input.IP,
		input.UserAgent,
		input.Message,
		metadata,
		input.CreatedAt,
	)
	var id int64
	if err := row.Scan(&id); err != nil {
		return auditstore.AuditLog{}, fmt.Errorf("create audit log: %w", err)
	}
	record.ID = toStoreID(id)

	return record, nil
}

// ListAuditLogs returns a stable page of audit records plus total count.
func (r *repository) ListAuditLogs(ctx context.Context, query auditstore.ListAuditLogsQuery) (auditstore.ListAuditLogsResult, error) {
	if r == nil || r.db == nil {
		return auditstore.ListAuditLogsResult{}, errors.New("audit repository is unavailable")
	}
	if query.Limit <= 0 {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("list audit logs: invalid limit %d", query.Limit)
	}
	if query.Offset < 0 {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("list audit logs: invalid offset %d", query.Offset)
	}

	whereSQL, args := buildAuditLogFilters(query)

	countSQL := `SELECT COUNT(*) FROM audit_logs` + whereSQL
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("count audit logs: %w", err)
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, query.Limit, query.Offset)

	//nolint:gosec // Query text is assembled from fixed SQL fragments; all dynamic values stay parameterized.
	selectSQL := `SELECT
		id,
		actor_user_id,
		actor_username,
		actor_display_name,
		action,
		resource_type,
		resource_id,
		resource_name,
		success,
		request_id,
		ip,
		user_agent,
		message,
		metadata,
		created_at
	FROM audit_logs` + whereSQL + fmt.Sprintf(
		" ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d",
		len(args)+1,
		len(args)+paginationParamCount,
	)

	rows, err := r.db.QueryContext(ctx, selectSQL, queryArgs...)
	if err != nil {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("list audit logs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]auditstore.AuditLog, 0, query.Limit)
	for rows.Next() {
		record, err := scanAuditLog(rows)
		if err != nil {
			return auditstore.ListAuditLogsResult{}, err
		}
		enrichAuditLog(&record)
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("iterate audit logs: %w", err)
	}

	return auditstore.ListAuditLogsResult{Items: items, Total: total}, nil
}

// ReadAuditOverview aggregates real overview data from the settled audit log table.
func (r *repository) ReadAuditOverview(ctx context.Context, window auditstore.OverviewWindow) (auditstore.AuditOverview, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditOverview{}, errors.New("audit repository is unavailable")
	}

	now := time.Now().UTC()
	startedAt := overviewWindowStart(now, window)
	args := []any{startedAt}

	summary, err := r.readAuditOverviewSummary(ctx, args)
	if err != nil {
		return auditstore.AuditOverview{}, err
	}

	failedAuth, err := r.readAuditOverviewItems(ctx, args, overviewFailedAuthWhere)
	if err != nil {
		return auditstore.AuditOverview{}, err
	}
	permissionDenied, err := r.readAuditOverviewItems(ctx, args, overviewPermissionDeniedWhere)
	if err != nil {
		return auditstore.AuditOverview{}, err
	}
	sensitiveOps, err := r.readAuditOverviewItems(ctx, args, overviewSensitiveOpsWhere)
	if err != nil {
		return auditstore.AuditOverview{}, err
	}

	return auditstore.AuditOverview{
		Window:           window,
		Summary:          summary,
		FailedAuth:       failedAuth,
		PermissionDenied: failedAuthUniqueByRequest(failedAuth, permissionDenied),
		SensitiveOps:     sensitiveOps,
	}, nil
}

func buildAuditLogFilters(query auditstore.ListAuditLogsQuery) (string, []any) {
	clauses := make([]string, 0, defaultFilterCapacity)
	args := make([]any, 0, defaultFilterCapacity)

	add := func(format string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(format, len(args)))
	}

	addUint64Filter(&clauses, &args, "actor_user_id = $%d", query.ActorUserID)
	addScalarFilter(add, "action = $%d", query.Action)
	addScalarFilter(add, "resource_type = $%d", query.ResourceType)
	addScalarFilter(add, "resource_id = $%d", query.ResourceID)
	addScalarFilter(add, "resource_name = $%d", query.ResourceName)
	addBoolFilter(&clauses, &args, "success = $%d", query.Success)
	addScalarFilter(add, "request_id = $%d", query.RequestID)
	addScalarFilter(add, auditResultWhereClause(), string(query.Result))
	addScalarFilter(add, riskLevelWhereClause(), string(query.RiskLevel))
	addTimeFilter(&clauses, &args, "created_at >= $%d", query.CreatedFrom)
	addTimeFilter(&clauses, &args, "created_at <= $%d", query.CreatedTo)
	if len(clauses) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}

func addScalarFilter(add func(string, any), format string, value string) {
	if value == "" {
		return
	}
	add(format, value)
}

func addUint64Filter(clauses *[]string, args *[]any, format string, value *uint64) {
	if value == nil {
		return
	}
	*args = append(*args, *value)
	*clauses = append(*clauses, fmt.Sprintf(format, len(*args)))
}

func addBoolFilter(clauses *[]string, args *[]any, format string, value *bool) {
	if value == nil {
		return
	}
	*args = append(*args, *value)
	*clauses = append(*clauses, fmt.Sprintf(format, len(*args)))
}

func addTimeFilter(clauses *[]string, args *[]any, format string, value *time.Time) {
	if value == nil {
		return
	}
	*args = append(*args, value.UTC())
	*clauses = append(*clauses, fmt.Sprintf(format, len(*args)))
}

func scanAuditLog(scanner interface {
	Scan(dest ...any) error
}) (auditstore.AuditLog, error) {
	var (
		record      auditstore.AuditLog
		actorUserID sql.NullInt64
		metadata    []byte
	)
	if err := scanner.Scan(
		&record.ID,
		&actorUserID,
		&record.ActorUsername,
		&record.ActorDisplayName,
		&record.Action,
		&record.ResourceType,
		&record.ResourceID,
		&record.ResourceName,
		&record.Success,
		&record.RequestID,
		&record.IP,
		&record.UserAgent,
		&record.Message,
		&metadata,
		&record.CreatedAt,
	); err != nil {
		return auditstore.AuditLog{}, fmt.Errorf("scan audit log: %w", err)
	}

	if actorUserID.Valid {
		value := toStoreID(actorUserID.Int64)
		record.ActorUserID = &value
	}
	record.Metadata = cloneRawMessage(metadata)
	enrichAuditLog(&record)

	return record, nil
}

func enrichAuditLog(record *auditstore.AuditLog) {
	if record == nil {
		return
	}

	metadata := decodeAuditMetadata(record.Metadata)
	record.TraceID = stringMetadataValue(metadata, "trace_id")
	if record.TraceID == "" {
		record.TraceID = record.RequestID
	}
	record.SessionID = stringMetadataValue(metadata, "session_id")
	record.RequestMethod = stringMetadataValue(metadata, "request_method")
	record.RequestPath = stringMetadataValue(metadata, "request_path")
	record.StatusCode = intMetadataValue(metadata, "status_code")
	record.TargetType = normalizeAuditTargetType(record.ResourceType)
	record.TargetLabel = firstNonEmpty(record.ResourceName, displayTargetLabel(record.TargetType), record.ResourceID)
	record.Result = classifyAuditResult(*record, metadata)
	record.RiskLevel = classifyAuditRiskLevel(*record)
}

func decodeAuditMetadata(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return map[string]any{}
	}

	return metadata
}

func stringMetadataValue(metadata map[string]any, key string) string {
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", typed))
	default:
		return ""
	}
}

func intMetadataValue(metadata map[string]any, key string) int {
	value, ok := metadata[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return 0
}

func classifyAuditResult(record auditstore.AuditLog, metadata map[string]any) auditstore.AuditResult {
	if record.Success {
		return auditstore.AuditResultSuccess
	}

	statusCode := record.StatusCode
	if statusCode == 0 {
		statusCode = intMetadataValue(metadata, "status_code")
	}
	if statusCode == httpStatusForbidden {
		return auditstore.AuditResultDenied
	}
	if statusCode >= 500 || stringMetadataValue(metadata, "error_kind") == "system" || stringMetadataValue(metadata, "error") != "" {
		return auditstore.AuditResultError
	}

	return auditstore.AuditResultFailed
}

func classifyAuditRiskLevel(record auditstore.AuditLog) auditstore.AuditRiskLevel {
	action := strings.ToLower(strings.TrimSpace(record.Action))

	if record.Result == auditstore.AuditResultError || record.Result == auditstore.AuditResultDenied {
		return auditstore.AuditRiskLevelCritical
	}
	if containsAny(action, []string{"reset_password", "update_permission", "update_role", "assign_role", "token_revoke"}) {
		return auditstore.AuditRiskLevelCritical
	}
	if record.Result == auditstore.AuditResultFailed || containsAny(action, sensitiveAuditActionKeywords) {
		return auditstore.AuditRiskLevelHigh
	}
	if containsAny(action, []string{"login_failed", "login", "permission", "role", "auth"}) {
		return auditstore.AuditRiskLevelMedium
	}
	return auditstore.AuditRiskLevelLow
}

func containsAny(source string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(source, keyword) {
			return true
		}
	}
	return false
}

func normalizeAuditTargetType(resourceType string) string {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "user", "users":
		return "USER"
	case "role", "roles":
		return "ROLE"
	case "permission", "permissions":
		return "PERMISSION"
	case "audit":
		return "AUDIT"
	case "monitor", "server-status", "server_status":
		return "SERVER_STATUS"
	case "auth", "session", "sessions", "login":
		return "AUTH"
	default:
		if resourceType == "" {
			return "AUDIT"
		}
		return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(resourceType), "-", "_"))
	}
}

func displayTargetLabel(targetType string) string {
	switch targetType {
	case "USER":
		return "用户"
	case "ROLE":
		return "角色"
	case "PERMISSION":
		return "权限"
	case "AUDIT":
		return "审计"
	case "SERVER_STATUS":
		return "服务器状态"
	case "AUTH":
		return "认证"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func auditResultWhereClause() string {
	return `CASE
		WHEN success THEN 'SUCCESS'
		ELSE CASE
			WHEN (metadata ->> 'status_code') = '403' THEN 'DENIED'
			WHEN (
				COALESCE(metadata ->> 'status_code', '') ~ '^[0-9]+$'
				AND (metadata ->> 'status_code')::int >= 500
			) OR COALESCE(metadata ->> 'error_kind', '') = 'system'
			  OR COALESCE(metadata ->> 'error', '') <> '' THEN 'ERROR'
			ELSE 'FAILED'
		END
	END = $%d`
}

func riskLevelWhereClause() string {
	return `CASE
		WHEN success = false AND (
			(metadata ->> 'status_code') = '403'
			OR (
				COALESCE(metadata ->> 'status_code', '') ~ '^[0-9]+$'
				AND (metadata ->> 'status_code')::int >= 500
			)
			OR COALESCE(metadata ->> 'error_kind', '') = 'system'
			OR COALESCE(metadata ->> 'error', '') <> ''
		) THEN 'CRITICAL'
		WHEN LOWER(action) LIKE '%%reset_password%%' OR LOWER(action) LIKE '%%update_permission%%' OR LOWER(action) LIKE '%%update_role%%' OR LOWER(action) LIKE '%%assign_role%%' OR LOWER(action) LIKE '%%token_revoke%%' THEN 'CRITICAL'
		WHEN success = false OR LOWER(action) LIKE '%%delete%%' OR LOWER(action) LIKE '%%reset%%' OR LOWER(action) LIKE '%%grant%%' OR LOWER(action) LIKE '%%assign%%' OR LOWER(action) LIKE '%%revoke%%' OR LOWER(action) LIKE '%%remove%%' OR LOWER(action) LIKE '%%replace%%' THEN 'HIGH'
		WHEN LOWER(action) LIKE '%%login_failed%%' OR LOWER(action) LIKE '%%login%%' OR LOWER(action) LIKE '%%permission%%' OR LOWER(action) LIKE '%%role%%' OR LOWER(action) LIKE '%%auth%%' THEN 'MEDIUM'
		ELSE 'LOW'
	END = $%d`
}

const overviewSummarySQL = `
SELECT
	COUNT(*) AS total_logs,
	COUNT(*) FILTER (WHERE success = false) AS failed_operations,
	COUNT(*) FILTER (
		WHERE success = false
		   OR LOWER(action) LIKE '%delete%'
		   OR LOWER(action) LIKE '%reset%'
		   OR LOWER(action) LIKE '%grant%'
		   OR LOWER(action) LIKE '%assign%'
		   OR LOWER(action) LIKE '%revoke%'
		   OR LOWER(action) LIKE '%remove%'
		   OR LOWER(action) LIKE '%replace%'
	) AS high_risk_events,
	COUNT(*) FILTER (
		WHERE LOWER(action) LIKE '%delete%'
		   OR LOWER(action) LIKE '%reset%'
		   OR LOWER(action) LIKE '%grant%'
		   OR LOWER(action) LIKE '%assign%'
		   OR LOWER(action) LIKE '%revoke%'
		   OR LOWER(action) LIKE '%remove%'
		   OR LOWER(action) LIKE '%replace%'
	) AS sensitive_operations
FROM audit_logs
WHERE created_at >= $1
`

const overviewRecentBaseSQL = `
SELECT
	id,
	actor_user_id,
	actor_username,
	actor_display_name,
	action,
	resource_type,
	resource_id,
	resource_name,
	success,
	request_id,
	message,
	metadata,
	created_at
FROM audit_logs
WHERE created_at >= $1 AND %s
ORDER BY created_at DESC, id DESC
LIMIT 3
`

func metadataTextValueSQL(column string, key string) string {
	return fmt.Sprintf("COALESCE(%s ->> '%s', '')", column, key)
}

var (
	overviewMetadataRequestPathSQL = metadataTextValueSQL("metadata", "request_path")
	overviewMetadataStatusCodeSQL  = metadataTextValueSQL("metadata", "status_code")
)

const overviewSensitiveOpsWhere = `
	LOWER(action) LIKE '%delete%'
	OR LOWER(action) LIKE '%reset%'
	OR LOWER(action) LIKE '%grant%'
	OR LOWER(action) LIKE '%assign%'
	OR LOWER(action) LIKE '%revoke%'
	OR LOWER(action) LIKE '%remove%'
	OR LOWER(action) LIKE '%replace%'
`

var overviewFailedAuthWhere = `
	success = false AND (
		LOWER(action) LIKE '%auth%'
		OR resource_type = 'auth'
		OR resource_type = 'session'
		OR LOWER(` + overviewMetadataRequestPathSQL + `) LIKE '/api/auth%'
	)
`

var overviewPermissionDeniedWhere = `
	success = false AND (
		` + overviewMetadataStatusCodeSQL + ` = '403'
		OR message = 'common.forbidden'
		OR LOWER(message) LIKE '%forbidden%'
		OR LOWER(message) LIKE '%permission%'
	)
`

func overviewWindowStart(now time.Time, window auditstore.OverviewWindow) time.Time {
	switch window {
	case auditstore.OverviewWindow7Days:
		return now.Add(-7 * 24 * time.Hour)
	case auditstore.OverviewWindow30Days:
		return now.Add(-30 * 24 * time.Hour)
	default:
		return now.Add(-24 * time.Hour)
	}
}

func (r *repository) readAuditOverviewSummary(ctx context.Context, args []any) (auditstore.OverviewSummary, error) {
	var summary auditstore.OverviewSummary
	if err := r.db.QueryRowContext(ctx, overviewSummarySQL, args...).Scan(
		&summary.TotalLogs,
		&summary.FailedOperations,
		&summary.HighRiskEvents,
		&summary.SensitiveOperations,
	); err != nil {
		return auditstore.OverviewSummary{}, fmt.Errorf("read audit overview summary: %w", err)
	}
	return summary, nil
}

func (r *repository) readAuditOverviewItems(ctx context.Context, args []any, where string) ([]auditstore.OverviewItem, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(overviewRecentBaseSQL, where), args...)
	if err != nil {
		return nil, fmt.Errorf("read audit overview items: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]auditstore.OverviewItem, 0, overviewRecentLimit)
	for rows.Next() {
		item, scanErr := scanAuditOverviewItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit overview items: %w", err)
	}

	return items, nil
}

func scanAuditOverviewItem(scanner interface {
	Scan(dest ...any) error
}) (auditstore.OverviewItem, error) {
	var (
		item        auditstore.OverviewItem
		actorUserID sql.NullInt64
		metadata    []byte
	)
	if err := scanner.Scan(
		&item.ID,
		&actorUserID,
		&item.ActorUsername,
		&item.ActorDisplayName,
		&item.Action,
		&item.ResourceType,
		&item.ResourceID,
		&item.ResourceName,
		&item.Success,
		&item.RequestID,
		&item.Message,
		&metadata,
		&item.CreatedAt,
	); err != nil {
		return auditstore.OverviewItem{}, fmt.Errorf("scan audit overview item: %w", err)
	}

	if actorUserID.Valid {
		value := toStoreID(actorUserID.Int64)
		item.ActorUserID = &value
	}
	item.Metadata = cloneRawMessage(metadata)
	return item, nil
}

func failedAuthUniqueByRequest(primary []auditstore.OverviewItem, fallback []auditstore.OverviewItem) []auditstore.OverviewItem {
	items := append([]auditstore.OverviewItem(nil), fallback...)
	slices.SortFunc(items, func(a, b auditstore.OverviewItem) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	if len(items) > overviewRecentLimit {
		items = items[:overviewRecentLimit]
	}
	if len(items) > 0 {
		return items
	}
	return primary
}

func nullableUint64(value *uint64) any {
	if value == nil {
		return nil
	}

	return *value
}

func toStoreID(id int64) uint64 {
	//nolint:gosec // 数据库 ID 来自受控 schema，并保持为正数。
	return uint64(id)
}

func cloneRawMessage(value []byte) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage([]byte("{}"))
	}

	cloned := make([]byte, len(value))
	copy(cloned, value)
	return json.RawMessage(cloned)
}
