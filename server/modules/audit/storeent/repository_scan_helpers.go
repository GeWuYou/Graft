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
// scanAuditLog 扫描并补全一条审计日志记录。
//
// @param ctx 审计标签本地化使用的上下文。
// @param scanner 提供行数据的扫描器。
// @returns 成功时返回完整的审计日志记录，扫描失败时返回错误。
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

// enrichAuditLog 基于元数据补全审计日志的派生字段。
// 它会解析记录中的元数据，填充来源、TraceID、会话与请求信息，计算结果与风险等级，归一化目标类型和标签，并构建目标信息。
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

// buildAuditTarget 根据审计记录构建用于展示和跳转的目标信息。
// 目标类型、标识和标签会按请求、会话、操作者和资源信息择优填充；当记录满足告警关联条件时，会改写为 incident 目标并生成对应路由引用。
// @param record 审计记录。
// @returns 构建后的审计目标。
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

// shouldLinkAuditIncident 判断审计记录是否应链接到事故详情。
// 当结果为拒绝或错误、来源为安全事件，或风险等级为高或严重时，返回 true。
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

// decodeAuditMetadata 解析审计元数据的 JSON 内容。
//
// 当输入为空或解析失败时，返回空映射。
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

// stringMetadataValue 返回元数据中指定键对应的字符串值。
// 当值为字符串时会去除首尾空白；当值为数值时会将其按整数形式格式化后返回。
// 其他类型或缺失键返回空字符串。
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

// metadataTextFirst 按顺序返回元数据中第一个非空文本值。
// 依次检查给定键，返回首个可用的字符串值；如果都为空，则返回空字符串。
func metadataTextFirst(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringMetadataValue(metadata, key); value != "" {
			return value
		}
	}
	return ""
}

// intMetadataValue 返回元数据中指定键的整数值。
// 支持 float64、int 和可解析的字符串；其余类型或解析失败时返回 0。
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

// classifyAuditResult 根据记录状态和元数据归类审计结果。
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

// classifyAuditRiskLevel 根据审计记录的结果、资源类型和操作名称计算风险等级。
// 当结果为错误或拒绝时返回严重风险；当操作属于容器危险操作、权限重置类操作或其他敏感操作时返回较高风险；登录、权限和认证相关操作返回中等风险。
// @returns 审计记录对应的风险等级。
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

// containsAny reports whether source contains any keyword in keywords.
// It returns true if at least one keyword is found in source, false otherwise.
func containsAny(source string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(source, keyword) {
			return true
		}
	}
	return false
}

// sensitiveOperationAuthorityKeywords 返回用于识别敏感操作的权限相关关键字列表。
func sensitiveOperationAuthorityKeywords() []string {
	return []string{"delete", "reset", "grant", "assign", "revoke", "remove", "replace"}
}

// sensitiveOperationMatch 判断操作名称是否包含敏感操作关键字。
func sensitiveOperationMatch(action string) bool {
	return containsAny(action, sensitiveOperationAuthorityKeywords())
}

// normalizedAuditClassifierValue 规范化审计分类器值，去除首尾空白并转换为小写。
func normalizedAuditClassifierValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// normalizedAuditClassifierColumn 生成将列值转为小写并去除首尾空白的 SQL 表达式。
func normalizedAuditClassifierColumn(column string) string {
	return "LOWER(TRIM(" + column + "))"
}

// isContainerDangerousAction 判断容器类型资源上的高风险操作。
// 当资源类型为 "container" 或 "container_batch"，且操作以 "ops.container.action." 开头时返回 true。
func isContainerDangerousAction(resourceType string, action string) bool {
	return (resourceType == "container" || resourceType == "container_batch") &&
		strings.HasPrefix(action, "ops.container.action.")
}

// containerDangerousActionExpression 返回用于匹配容器高风险操作的 SQL 条件表达式。
// 表达式要求资源类型为 `container` 或 `container_batch`，且操作名以 `ops.container.action.` 开头。
func containerDangerousActionExpression(actionColumn string, resourceTypeColumn string) string {
	normalizedAction := normalizedAuditClassifierColumn(actionColumn)
	normalizedResourceType := normalizedAuditClassifierColumn(resourceTypeColumn)
	return `((
		` + normalizedResourceType + ` = 'container'
		OR ` + normalizedResourceType + ` = 'container_batch'
	) AND ` + normalizedAction + ` LIKE 'ops.container.action.%')`
}

// normalizeAuditTargetType 将资源类型规范化为审计目标类型。
// 支持常见别名映射，并在无法识别时保留原值的标准化形式。
func normalizeAuditTargetType(resourceType string) string {
	trimmed := strings.TrimSpace(resourceType)
	switch strings.ToLower(trimmed) {
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
		if trimmed == "" {
			return "AUDIT"
		}
		return strings.ToUpper(strings.ReplaceAll(trimmed, "-", "_"))
	}
}

// displayTargetLabel 根据目标类型和审计语言环境返回本地化的目标标签。
// 当未找到对应消息键、localizer 为空或无法匹配翻译时，返回空字符串。
// @param targetType 目标类型标识。
// @returns 本地化后的目标标签。
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

// targetLabelMessageKey 返回与目标类型对应的审计目标标签消息键。
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

// WithAuditLocale 将审计读取流程使用的语言环境写入上下文。
// 当 ctx 为空时，使用 context.Background() 作为基础上下文。
func WithAuditLocale(ctx context.Context, locale string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, auditLocaleContextKey{}, strings.TrimSpace(locale))
}

// auditLocaleFromContext 从上下文中读取审计语言环境并返回修剪后的值。
// 如果上下文中未设置语言环境或值类型不匹配，则返回空字符串。
func auditLocaleFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	locale, _ := ctx.Value(auditLocaleContextKey{}).(string)
	return strings.TrimSpace(locale)
}

// firstNonEmpty 返回第一个去除首尾空白后仍非空的字符串。
// 如果所有入参都为空，则返回空字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// normalizeAuditSource 将字符串转换为受支持的审计来源枚举值。
// 返回对应的审计来源值；如果输入无法识别，则返回空值。
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
