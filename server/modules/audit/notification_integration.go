package audit

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"graft/server/internal/moduleapi"
	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
	notificationcontract "graft/server/modules/notification/contract"
)

// publishAuditNotification 在满足条件时发布审计通知，并在发布失败时记录警告日志。
// 当 publisher 为空或审计记录不符合通知条件时，函数直接返回。否则会构建通知输入并尝试发布；
// 发布失败时会使用非空 logger 记录包含模块、事件类型、严重级别、审计日志 ID 和错误信息的警告日志。
func publishAuditNotification(
	ctx context.Context,
	logger *zap.Logger,
	publisher moduleapi.NotificationPublisher,
	record auditstore.AuditLog,
) {
	if publisher == nil || !shouldNotifyAuditRecord(record) {
		return
	}
	input := auditNotificationInput(record)
	if _, err := publisher.Publish(ctx, input); err != nil {
		if logger == nil {
			logger = zap.NewNop()
		}
		logger.Warn("publish audit notification failed",
			zap.String("module", moduleID),
			zap.String("notificationEventType", input.EventType),
			zap.String("notificationSeverity", string(input.Severity)),
			zap.Uint64("auditLogID", record.ID),
			zap.Error(err),
		)
	}
}

// 仅对可见的记录进行判断；当操作包含权限拒绝、登录失败，或登录类操作且执行失败时返回 true。风险级别为高或严重时也返回 true。
func shouldNotifyAuditRecord(record auditstore.AuditLog) bool {
	if record.Visibility != auditstore.AuditVisibilityStrategyVisible {
		return false
	}
	action := strings.ToLower(strings.TrimSpace(record.Action))
	if strings.Contains(action, "permission.denied") || strings.Contains(action, "permission_denied") {
		return true
	}
	if strings.Contains(action, "login_failed") || (strings.Contains(action, "login") && !record.Success) {
		return true
	}
	return auditRecordRiskLevel(record) == auditstore.AuditRiskLevelHigh ||
		auditRecordRiskLevel(record) == auditstore.AuditRiskLevelCritical
}

// auditNotificationInput 根据审计日志构建通知发布输入。
// 它会选择通知文案与严重级别，合并审计上下文与现有元数据，并填充导航、资源和接收目标信息。
// 返回用于发布审计通知的输入。
func auditNotificationInput(record auditstore.AuditLog) moduleapi.PublishNotificationInput {
	kind := auditNotificationKind(record)
	copyParts := auditNotificationCopy(kind, record)
	titleKey, title := copyParts.titleKey, copyParts.title
	messageKey, message := copyParts.messageKey, copyParts.message
	actionLabelKey, actionLabel := copyParts.actionLabelKey, copyParts.actionLabel
	severity := moduleapi.NotificationSeverity(notificationcontract.SeverityWarning)
	if auditRecordRiskLevel(record) == auditstore.AuditRiskLevelCritical {
		severity = moduleapi.NotificationSeverity(notificationcontract.SeverityCritical)
	}
	metadata := json.RawMessage(record.Metadata)
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	navigationPayload, _ := json.Marshal(map[string]any{
		"audit_log_id": record.ID,
		"request_id":   record.RequestID,
		"trace_id":     record.TraceID,
	})
	auditContext := map[string]any{
		"auditLogId":   record.ID,
		"action":       record.Action,
		"eventType":    kind,
		"resourceType":  firstNonEmptyTrimmed(record.ResourceType, "audit_log"),
		"resourceId":   firstNonEmptyTrimmed(record.ResourceID, strconv.FormatUint(record.ID, 10)),
		"resourceName":  firstNonEmptyTrimmed(record.ResourceName, record.Action),
		"result":       firstNonEmptyTrimmed(string(record.Result), resultFallback(record.Success)),
		"riskLevel":    string(record.RiskLevel),
		"requestId":    record.RequestID,
		"traceId":      record.TraceID,
		"reason":       firstNonEmptyTrimmed(record.Message, string(record.Result)),
		"actionLabel":   actionLabel,
		"title":        title,
		"message":      message,
	}
	if record.StatusCode > 0 {
		auditContext["statusCode"] = record.StatusCode
	}

	return moduleapi.PublishNotificationInput{
		TitleKey:     titleKey,
		Title:        title,
		MessageKey:   messageKey,
		Message:      message,
		ActionLabelKey: actionLabelKey,
		ActionLabel:    actionLabel,
		Severity:     severity,
		Category:     moduleapi.NotificationCategory(notificationcontract.CategorySecurity),
		SourceModule: moduleID,
		EventType:    kind,
		ResourceType: firstNonEmptyTrimmed(record.ResourceType, "audit_log"),
		ResourceID:   firstNonEmptyTrimmed(record.ResourceID, strconv.FormatUint(record.ID, 10)),
		ResourceName: firstNonEmptyTrimmed(record.ResourceName, record.Action),
		Navigation: moduleapi.NotificationNavigation{
			Kind:    moduleapi.NotificationNavigationKind(notificationcontract.NavigationAuditLog),
			Payload: navigationPayload,
		},
		Metadata:   mustMarshalJSON(auditContext, metadata),
		DedupeKey:  "audit:" + strconv.FormatUint(record.ID, 10),
		OccurredAt: record.CreatedAt,
		Target: moduleapi.NotificationTarget{
			Type: moduleapi.NotificationTargetType(notificationcontract.TargetPermission),
			Ref:  auditcontract.AuditReadPermission.String(),
		},
	}
}

// auditNotificationKind 根据审计记录的动作和结果确定通知类型。
// 当动作包含权限拒绝或登录失败相关标记时返回对应类型；否则返回高风险类型。
func auditNotificationKind(record auditstore.AuditLog) string {
	action := strings.ToLower(strings.TrimSpace(record.Action))
	switch {
	case strings.Contains(action, "permission.denied") || strings.Contains(action, "permission_denied"):
		return "permission_denied"
	case strings.Contains(action, "login_failed") || (strings.Contains(action, "login") && !record.Success):
		return "login_failed"
	default:
		return "high_risk"
	}
}

type auditNotificationCopyParts struct {
	titleKey       string
	title          string
	messageKey     string
	message        string
	actionLabelKey string
	actionLabel    string
}

// auditNotificationCopy 根据通知类型和审计记录生成标题、消息与操作按钮的本地化键和值。
//
// kind 决定返回登录失败、权限拒绝或高风险审计事件的文案；record 用于生成权限拒绝消息中的目标名称。
// 返回对应的标题、消息和操作按钮文案及其本地化键。
func auditNotificationCopy(kind string, record auditstore.AuditLog) auditNotificationCopyParts {
	target := firstNonEmptyTrimmed(record.ResourceName, record.ResourceID, record.Action, "Audit event")
	switch kind {
	case "login_failed":
		return auditNotificationCopyParts{
		titleKey:       "notification.title.audit.loginFailed",
		title:          "Login failed",
		messageKey:     "notification.message.audit.loginFailed",
		message:        "A failed login attempt needs review.",
		actionLabelKey: "notification.action.openAuditLog",
		actionLabel:    "View audit log",
	}
	case "permission_denied":
		return auditNotificationCopyParts{
		titleKey:       "notification.title.audit.permissionDenied",
		title:          "Permission denied",
		messageKey:     "notification.message.audit.permissionDenied",
		message:        "Permission was denied for " + target + ".",
		actionLabelKey: "notification.action.openAuditLog",
		actionLabel:    "View audit log",
	}
	default:
		return auditNotificationCopyParts{
		titleKey:       "notification.title.audit.highRisk",
		title:          "High-risk audit event",
		messageKey:     "notification.message.audit.highRisk",
		message:        "High-risk audit activity needs review.",
		actionLabelKey: "notification.action.openAuditLog",
		actionLabel:    "View audit log",
	}
	}
}

// mustMarshalJSON 合并审计上下文字段与已有元数据，并将结果编码为 JSON。
// 同名字段以审计上下文中的值为准；如果编码失败，则返回空对象 `{}`。
func mustMarshalJSON(auditContext map[string]any, existing json.RawMessage) json.RawMessage {
	const auditMetadataExtraCapacity = 4

	metadata := make(map[string]any, len(auditContext)+auditMetadataExtraCapacity)
	for key, value := range auditContext {
		metadata[key] = value
	}
	if len(existing) > 0 {
		var extra map[string]any
		if err := json.Unmarshal(existing, &extra); err == nil {
			for key, value := range extra {
				if _, exists := metadata[key]; !exists {
					metadata[key] = value
				}
			}
		}
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(payload)
}

// resultFallback 返回与成功状态对应的结果字符串。
// 返回 "SUCCESS" 表示成功，返回 "FAILED" 表示失败。
func resultFallback(success bool) string {
	if success {
		return "SUCCESS"
	}
	return "FAILED"
}

// auditRecordRiskLevel 返回审计日志的风险等级；如果记录已显式指定，则优先使用该值，否则根据记录内容推断。
func auditRecordRiskLevel(record auditstore.AuditLog) auditstore.AuditRiskLevel {
	if record.RiskLevel != "" {
		return record.RiskLevel
	}
	return classifyCandidateAuditRiskLevel(record)
}
