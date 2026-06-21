package audit

import (
	"context"
	"testing"
	"time"

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
}

type auditNotificationPublisherRecorder struct {
	inputs []moduleapi.PublishNotificationInput
}

func (r *auditNotificationPublisherRecorder) Publish(_ context.Context, input moduleapi.PublishNotificationInput) (moduleapi.PublishNotificationResult, error) {
	r.inputs = append(r.inputs, input)
	return moduleapi.PublishNotificationResult{}, nil
}
