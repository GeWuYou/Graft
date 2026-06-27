package audit

import (
	"context"
	"testing"
	"time"
	"strings"

	"go.uber.org/zap"

	"graft/server/internal/moduleapi"
	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
	notificationcontract "graft/server/modules/notification/contract"
)

func TestPublishAuditNotificationTargetsAuditReaders(t *testing.T) {
	publisher := &auditNotificationPublisherRecorder{}
	publishAuditNotification(context.Background(), zap.NewNop(), publisher, auditstore.AuditLog{
		ID:           12,
		Action:       "auth.permission.denied",
		Visibility:   auditstore.AuditVisibilityStrategyVisible,
		ResourceType: "permission",
		ResourceID:   "rbac.role.delete",
		Success:      false,
		RequestID:    "req-1",
		CreatedAt:    time.Date(2026, 6, 9, 8, 0, 0, 0, time.UTC),
	})

	if len(publisher.inputs) != 1 {
		t.Fatalf("expected one notification, got %d", len(publisher.inputs))
	}
	input := publisher.inputs[0]
	if input.Target.Type != moduleapi.NotificationTargetType(notificationcontract.TargetPermission) ||
		input.Target.Ref != auditcontract.AuditReadPermission.String() {
		t.Fatalf("unexpected audit notification target: %#v", input.Target)
	}
	if input.EventType != "permission_denied" || input.Category != moduleapi.NotificationCategory(notificationcontract.CategorySecurity) {
		t.Fatalf("unexpected audit notification input: %#v", input)
	}
	if input.TitleKey != "notification.title.audit.permissionDenied" || input.MessageKey != "notification.message.audit.permissionDenied" {
		t.Fatalf("expected localized audit copy keys, got %#v", input)
	}
	if input.ActionLabelKey != "notification.action.openAuditLog" {
		t.Fatalf("expected action label key, got %#v", input.ActionLabelKey)
	}
	if input.Navigation.Kind != moduleapi.NotificationNavigationKind(notificationcontract.NavigationAuditLog) {
		t.Fatalf("expected audit log navigation, got %#v", input.Navigation.Kind)
	}
	if string(input.Navigation.Payload) == "" || !strings.Contains(string(input.Navigation.Payload), `"audit_log_id":12`) {
		t.Fatalf("expected navigation payload to carry audit log id, got %s", string(input.Navigation.Payload))
	}
	if string(input.Metadata) == "" || !strings.Contains(string(input.Metadata), `"action":"auth.permission.denied"`) {
		t.Fatalf("expected audit metadata to carry action context, got %s", string(input.Metadata))
	}
}

func TestPublishAuditNotificationSkipsHiddenAuditLogs(t *testing.T) {
	publisher := &auditNotificationPublisherRecorder{}
	publishAuditNotification(context.Background(), zap.NewNop(), publisher, auditstore.AuditLog{
		ID:         22,
		Action:     "auth.permission.denied",
		Visibility: auditstore.AuditVisibilityStrategyHidden,
		Success:    false,
		CreatedAt:  time.Date(2026, 6, 9, 8, 0, 0, 0, time.UTC),
	})

	if len(publisher.inputs) != 0 {
		t.Fatalf("expected hidden audit log to skip notification, got %d", len(publisher.inputs))
	}
}

type auditNotificationPublisherRecorder struct {
	inputs []moduleapi.PublishNotificationInput
}

func (r *auditNotificationPublisherRecorder) Publish(_ context.Context, input moduleapi.PublishNotificationInput) (moduleapi.PublishNotificationResult, error) {
	r.inputs = append(r.inputs, input)
	return moduleapi.PublishNotificationResult{}, nil
}
