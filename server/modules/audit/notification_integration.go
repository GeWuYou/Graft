// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
			zap.String("action", record.Action),
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
	title, message := auditNotificationCopy(kind, record)
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
	})

	return moduleapi.PublishNotificationInput{
		Title:        title,
		Message:      message,
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
		Metadata:   metadata,
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

func auditNotificationCopy(kind string, record auditstore.AuditLog) (string, string) {
	target := firstNonEmptyTrimmed(record.ResourceName, record.ResourceID, record.Action, "Audit event")
	switch kind {
	case "login_failed":
		return "Login failed",
			"A failed login attempt needs review."
	case "permission_denied":
		return "Permission denied",
			"Permission was denied for " + target + "."
	default:
		return "High-risk audit event",
			"High-risk audit activity needs review."
	}
}

func auditRecordRiskLevel(record auditstore.AuditLog) auditstore.AuditRiskLevel {
	if record.RiskLevel != "" {
		return record.RiskLevel
	}
	return classifyCandidateAuditRiskLevel(record)
}
