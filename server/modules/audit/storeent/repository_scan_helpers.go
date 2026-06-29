package storeent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"graft/server/internal/i18n"
	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
)

// scanAuditLog 解析一条审计日志记录并补充派生字段。
// 它会从扫描器中读取基础列，转换 actor_user_id，并完成元数据克隆与记录增强。
// scanAuditLog 扫描一条审计日志记录，并补充派生字段。
// 解析成功时返回完整的审计日志；扫描失败时返回错误。
func scanAuditLog(
	ctx context.Context,
	localizer *i18n.Service,
	scanner interface {
		Scan(dest ...any) error
	},
) (auditstore.AuditLog, error) {
	var (
		record      auditstore.AuditLog
		actorUserID sql.NullInt64
		metadata    []byte
	)
	if err := scanner.Scan(
		&record.ID,
		&record.Source,
		&record.Visibility,
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
	record.Visibility = normalizeStoredAuditVisibility(record.Visibility)
	record.Metadata = cloneRawMessage(metadata)
	enrichAuditLog(ctx, &record, localizer)

	return record, nil
}

func enrichAuditLog(ctx context.Context, record *auditstore.AuditLog, localizer *i18n.Service) {
	if record == nil {
		return
	}

	metadata := decodeAuditMetadata(record.Metadata)
	record.Source = normalizeAuditSource(metadataTextFirst(metadata, "auditSource", "audit_source"))
	record.TraceID = stringMetadataValue(metadata, "trace_id")
	if record.TraceID == "" {
		record.TraceID = record.RequestID
	}
	record.SessionID = stringMetadataValue(metadata, "session_id")
	record.RequestMethod = stringMetadataValue(metadata, "request_method")
	record.RequestPath = stringMetadataValue(metadata, "request_path")
	record.StatusCode = intMetadataValue(metadata, "status_code")
	record.Result = classifyAuditResult(*record, metadata)
	record.RiskLevel = classifyAuditRiskLevel(*record)
	record.TargetType = normalizeAuditTargetType(record.ResourceType)
	record.TargetLabel = firstNonEmpty(record.ResourceName, displayTargetLabel(ctx, localizer, record.TargetType), record.ResourceID)
	record.Target = buildAuditTarget(*record)
}

func buildAuditTarget(record auditstore.AuditLog) auditstore.AuditTarget {
	targetType := firstNonEmpty(record.TargetType, record.ResourceType)
	label := firstNonEmpty(record.TargetLabel, record.ResourceName, record.ResourceID, record.Action)
	target := auditstore.AuditTarget{
		Kind:  "resource",
		Type:  targetType,
		ID:    record.ResourceID,
		Label: label,
	}

	switch {
	case record.RequestID != "":
		target.Kind = "request"
		target.Type = firstNonEmpty(target.Type, "request")
		target.ID = record.RequestID
		target.Label = firstNonEmpty(label, record.RequestID)
	case record.SessionID != "":
		target.Kind = "session"
		target.Type = firstNonEmpty(target.Type, "session")
		target.ID = record.SessionID
		target.Label = firstNonEmpty(label, record.SessionID)
	case record.ActorUserID != nil || record.ActorUsername != "" || record.ActorDisplayName != "":
		target.Kind = "actor"
		target.Type = firstNonEmpty(target.Type, "user")
		if target.ID == "" && record.ActorUserID != nil {
			target.ID = strconv.FormatUint(*record.ActorUserID, 10)
		}
		target.Label = firstNonEmpty(record.ActorDisplayName, record.ActorUsername, target.Label)
	}

	if shouldLinkAuditIncident(record) {
		target.Kind = "incident"
		target.Type = firstNonEmpty(target.Type, "incident")
		target.ID = strconv.FormatUint(record.ID, 10)
		target.Label = firstNonEmpty(target.Label, label, record.Action, target.ID)
		target.RouteRef = strings.Replace(auditcontract.AuditIncidentItem, ":"+auditcontract.AuditIncidentParam, target.ID, 1)
	}

	if target.Label == "" {
		target.Label = firstNonEmpty(target.Type, target.Kind, record.Action)
	}

	return target
}

func shouldLinkAuditIncident(record auditstore.AuditLog) bool {
	switch record.Result {
	case auditstore.AuditResultDenied, auditstore.AuditResultError:
		return true
	}

	switch record.Source {
	case auditstore.AuditSourceSecurityEvent:
		return true
	}

	switch record.RiskLevel {
	case auditstore.AuditRiskLevelHigh, auditstore.AuditRiskLevelCritical:
		return true
	}

	return false
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

func metadataTextFirst(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringMetadataValue(metadata, key); value != "" {
			return value
		}
	}
	return ""
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
	action := normalizedAuditClassifierValue(record.Action)
	resourceType := normalizedAuditClassifierValue(record.ResourceType)

	if record.Result == auditstore.AuditResultError || record.Result == auditstore.AuditResultDenied {
		return auditstore.AuditRiskLevelCritical
	}
	if isContainerDangerousAction(resourceType, action) {
		return auditstore.AuditRiskLevelHigh
	}
	if containsAny(action, []string{"reset_password", "update_permission", "update_role", "assign_role", "token_revoke"}) {
		return auditstore.AuditRiskLevelCritical
	}
	if record.Result == auditstore.AuditResultFailed || sensitiveOperationMatch(action) {
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

func sensitiveOperationAuthorityKeywords() []string {
	return []string{"delete", "reset", "grant", "assign", "revoke", "remove", "replace"}
}

func sensitiveOperationMatch(action string) bool {
	return containsAny(action, sensitiveOperationAuthorityKeywords())
}

func normalizedAuditClassifierValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizedAuditClassifierColumn(column string) string {
	return "LOWER(TRIM(" + column + "))"
}

func isContainerDangerousAction(resourceType string, action string) bool {
	return (resourceType == "container" || resourceType == "container_batch") &&
		strings.HasPrefix(action, "ops.container.action.")
}

func containerDangerousActionExpression(actionColumn string, resourceTypeColumn string) string {
	normalizedAction := normalizedAuditClassifierColumn(actionColumn)
	normalizedResourceType := normalizedAuditClassifierColumn(resourceTypeColumn)
	return `((
		` + normalizedResourceType + ` = 'container'
		OR ` + normalizedResourceType + ` = 'container_batch'
	) AND ` + normalizedAction + ` LIKE 'ops.container.action.%')`
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

func displayTargetLabel(ctx context.Context, localizer *i18n.Service, targetType string) string {
	key := targetLabelMessageKey(targetType)
	if key == "" || localizer == nil {
		return ""
	}

	return localizer.Lookup(i18n.LookupRequest{
		Namespace: "audit",
		Locale:    i18n.LocaleTag(auditLocaleFromContext(ctx)),
		Key:       i18n.MessageKey(key),
	})
}

func targetLabelMessageKey(targetType string) string {
	switch targetType {
	case "USER":
		return auditcontract.AuditTargetLabelUser.String()
	case "ROLE":
		return auditcontract.AuditTargetLabelRole.String()
	case "PERMISSION":
		return auditcontract.AuditTargetLabelPermission.String()
	case "AUDIT":
		return auditcontract.AuditTargetLabelAudit.String()
	case "SERVER_STATUS":
		return auditcontract.AuditTargetLabelServerStatus.String()
	case "AUTH":
		return auditcontract.AuditTargetLabelAuth.String()
	default:
		return ""
	}
}

// WithAuditLocale attaches the resolved request locale to one audit read path.
func WithAuditLocale(ctx context.Context, locale string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, auditLocaleContextKey{}, strings.TrimSpace(locale))
}

func auditLocaleFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	locale, _ := ctx.Value(auditLocaleContextKey{}).(string)
	return strings.TrimSpace(locale)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeAuditSource(value string) auditstore.AuditSource {
	switch auditstore.AuditSource(strings.ToUpper(strings.TrimSpace(value))) {
	case auditstore.AuditSourceRequest:
		return auditstore.AuditSourceRequest
	case auditstore.AuditSourceSecurityEvent:
		return auditstore.AuditSourceSecurityEvent
	case auditstore.AuditSourceDomainEvent:
		return auditstore.AuditSourceDomainEvent
	default:
		return ""
	}
}
