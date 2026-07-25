package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go.uber.org/zap"

	"graft/server/internal/event"
	"graft/server/internal/moduleapi"
	rbaccontract "graft/server/modules/rbac/contract"
	rbacstore "graft/server/modules/rbac/store"
)

type recordingPublisher struct {
	published  []event.Event
	publishErr error
}

func (p *recordingPublisher) Publish(_ context.Context, current event.Event, _ event.PublishOptions) (event.Receipt, error) {
	p.published = append(p.published, current)
	return event.Receipt{EventID: current.ID, Delivery: event.DeliveryDurable}, p.publishErr
}

func (p *recordingPublisher) PublishAsync(current event.Event, _ event.PublishOptions) (event.Receipt, error) {
	return p.Publish(context.Background(), current, event.PublishOptions{Delivery: event.DeliveryBestEffort})
}

func (p *recordingPublisher) PublishBatch(context.Context, []event.Event, event.PublishOptions) event.BatchReceipt {
	return event.BatchReceipt{}
}

func decodeRecordedAuditEvent(t *testing.T, current event.Event) moduleapi.AuditEvent {
	t.Helper()
	var decoded moduleapi.AuditEvent
	err := json.Unmarshal(current.Payload, &decoded)
	if err != nil {
		t.Fatalf("decode audit event: %v", err)
	}
	return decoded
}

func TestManagementWriterCreateRolePublishesAuditEvent(t *testing.T) {
	bus := &recordingPublisher{}
	writer := managementWriter{
		users:  testUserService{},
		rbac:   testRBACRepository{},
		events: bus,
		logger: zap.NewNop(),
	}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{
		User: &moduleapi.CurrentUser{ID: 7, Username: "admin", DisplayName: "Admin"},
	})

	role, err := writer.CreateRole(ctx, rbacstore.CreateRoleInput{
		Name:    "editor",
		Display: "Editor",
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if role.Name != "editor" {
		t.Fatalf("unexpected role: %#v", role)
	}
	if len(bus.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(bus.published))
	}

	payload := decodeRecordedAuditEvent(t, bus.published[0])
	if payload.Action != "rbac.role.create" || payload.ResourceID != "1" || payload.ResourceName != "editor" {
		t.Fatalf("unexpected event payload: %#v", payload)
	}
	if payload.Operator == nil || payload.Operator.ID != 7 {
		t.Fatalf("expected operator id 7, got %#v", payload.Operator)
	}
}

func TestManagementWriterRolePermissionMutationsPublishAuditMessageKeys(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mutate     func(managementWriter, context.Context) error
		action     string
		messageKey string
	}{
		{
			name: "add",
			mutate: func(writer managementWriter, ctx context.Context) error {
				return writer.AddPermissionsToRole(ctx, rbacstore.AddPermissionsToRoleInput{RoleID: 3, PermissionIDs: []uint64{9}})
			},
			action:     "rbac.role.permissions.add",
			messageKey: rbaccontract.AuditRolePermissionsAdded.String(),
		},
		{
			name: "remove",
			mutate: func(writer managementWriter, ctx context.Context) error {
				return writer.RemovePermissionsFromRole(ctx, rbacstore.RemovePermissionsFromRoleInput{RoleID: 3, PermissionIDs: []uint64{9}})
			},
			action:     "rbac.role.permissions.remove",
			messageKey: rbaccontract.AuditRolePermissionsRemoved.String(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bus := &recordingPublisher{}
			writer := managementWriter{
				users: testUserService{},
				rbac: testRBACRepository{
					roleByID: map[uint64]rbacstore.Role{
						3: {ID: 3, Name: "operator", Status: rbacstore.RoleStatusEnabled},
					},
					permissions: []rbacstore.Permission{{ID: 9, Code: "system.read"}},
				},
				events: bus,
				logger: zap.NewNop(),
			}

			if err := tc.mutate(writer, context.Background()); err != nil {
				t.Fatalf("mutate role permissions: %v", err)
			}
			if len(bus.published) != 1 {
				t.Fatalf("expected 1 published event, got %d", len(bus.published))
			}
			payload := decodeRecordedAuditEvent(t, bus.published[0])
			if payload.Action != tc.action || payload.MessageKey != tc.messageKey {
				t.Fatalf("unexpected audit event: %#v", payload)
			}
		})
	}
}

func TestManagementWriterReplaceRolesForUserAuditFailureDoesNotBlock(t *testing.T) {
	bus := &recordingPublisher{publishErr: errors.New("audit down")}
	writer := managementWriter{
		users: testUserService{users: map[uint64]moduleapi.UserSummary{
			11: {ID: 11, Username: "alice", Display: "Alice"},
		}},
		rbac: testRBACRepository{
			roles: []rbacstore.Role{
				{ID: 3, Name: "editor", Status: rbacstore.RoleStatusEnabled},
			},
			roleByID: map[uint64]rbacstore.Role{
				3: {ID: 3, Name: "editor", Status: rbacstore.RoleStatusEnabled},
			},
		},
		events: bus,
		logger: zap.NewNop(),
	}

	err := writer.ReplaceRolesForUser(context.Background(), rbacstore.ReplaceRolesForUserInput{
		UserID:  11,
		RoleIDs: []uint64{3},
	})
	if err != nil {
		t.Fatalf("replace roles for user: %v", err)
	}
	if len(bus.published) != 1 {
		t.Fatalf("expected audit publish attempt, got %d", len(bus.published))
	}
}
