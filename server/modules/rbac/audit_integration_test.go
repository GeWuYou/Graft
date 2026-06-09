// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
)

type recordingBus struct {
	published  []eventbus.Event
	publishErr error
}

func (b *recordingBus) Subscribe(string, eventbus.Handler) error {
	return nil
}

func (b *recordingBus) Publish(_ context.Context, event eventbus.Event) error {
	b.published = append(b.published, event)
	return b.publishErr
}

func TestManagementWriterCreateRolePublishesAuditEvent(t *testing.T) {
	bus := &recordingBus{}
	writer := managementWriter{
		users:    testUserService{},
		rbac:     testRBACRepository{},
		auditBus: bus,
		logger:   zap.NewNop(),
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

	event, ok := bus.published[0].Payload.(moduleapi.AuditEvent)
	if !ok {
		t.Fatalf("expected audit event payload, got %T", bus.published[0].Payload)
	}
	if event.Action != "rbac.role.create" || event.ResourceID != "1" || event.ResourceName != "editor" {
		t.Fatalf("unexpected event payload: %#v", event)
	}
	if event.Operator == nil || event.Operator.ID != 7 {
		t.Fatalf("expected operator id 7, got %#v", event.Operator)
	}
}

func TestManagementWriterReplaceRolesForUserAuditFailureDoesNotBlock(t *testing.T) {
	bus := &recordingBus{publishErr: errors.New("audit down")}
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
		auditBus: bus,
		logger:   zap.NewNop(),
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
