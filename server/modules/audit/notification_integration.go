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

func shouldNotifyAuditRecord(record auditstore.AuditLog) bool {
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

func auditNotificationInput(record auditstore.AuditLog) moduleapi.PublishNotificationInput {
	kind := auditNotificationKind(record)
	titleKey, title, messageKey, message, actionLabelKey, actionLabel := auditNotificationCopy(kind, record)
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

func auditNotificationCopy(kind string, record auditstore.AuditLog) (string, string, string, string, string, string) {
	target := firstNonEmptyTrimmed(record.ResourceName, record.ResourceID, record.Action, "Audit event")
	switch kind {
	case "login_failed":
		return "notification.title.audit.loginFailed",
			"Login failed",
			"notification.message.audit.loginFailed",
			"A failed login attempt needs review.",
			"notification.action.openAuditLog",
			"View audit log"
	case "permission_denied":
		return "notification.title.audit.permissionDenied",
			"Permission denied",
			"notification.message.audit.permissionDenied",
			"Permission was denied for " + target + ".",
			"notification.action.openAuditLog",
			"View audit log"
	default:
		return "notification.title.audit.highRisk",
			"High-risk audit event",
			"notification.message.audit.highRisk",
			"High-risk audit activity needs review.",
			"notification.action.openAuditLog",
			"View audit log"
	}
}

func mustMarshalJSON(auditContext map[string]any, existing json.RawMessage) json.RawMessage {
	metadata := make(map[string]any, len(auditContext)+4)
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

func resultFallback(success bool) string {
	if success {
		return "SUCCESS"
	}
	return "FAILED"
}

func auditRecordRiskLevel(record auditstore.AuditLog) auditstore.AuditRiskLevel {
	if record.RiskLevel != "" {
		return record.RiskLevel
	}
	return classifyCandidateAuditRiskLevel(record)
}
