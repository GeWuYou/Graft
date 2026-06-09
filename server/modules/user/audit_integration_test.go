// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/moduleapi"
	userstore "graft/server/modules/user/store"
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

func TestUserServiceCreateUserPublishesAuditEvent(t *testing.T) {
	bus := &recordingBus{}
	svc := userService{
		users: moduleTestUserRepository{
			create: func(_ context.Context, input userstore.CreateUserInput) (userstore.User, error) {
				return userstore.User{ID: 42, Username: input.Username, Display: input.Display, Status: input.Status}, nil
			},
		},
		auditBus: bus,
		logger:   zap.NewNop(),
	}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{
		User: &moduleapi.CurrentUser{ID: 7, Username: "admin", DisplayName: "Admin"},
	})

	created, err := svc.CreateUser(ctx, passwordHasher{cost: 4}, passwordPolicy{}, CreateUserCommand{
		Username: "alice",
		Display:  "Alice",
		Password: "Password1234",
		ActorID:  7,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.ID != 42 {
		t.Fatalf("expected created user id 42, got %d", created.ID)
	}
	if len(bus.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(bus.published))
	}

	event, ok := bus.published[0].Payload.(moduleapi.AuditEvent)
	if !ok {
		t.Fatalf("expected audit event payload, got %T", bus.published[0].Payload)
	}
	if event.Action != "user.create" || event.ResourceType != "user" || event.ResourceID != "42" {
		t.Fatalf("unexpected event payload: %#v", event)
	}
	if event.Operator == nil || event.Operator.ID != 7 {
		t.Fatalf("expected operator id 7, got %#v", event.Operator)
	}
}

func TestUserServiceResetUserPasswordAuditFailureDoesNotBlock(t *testing.T) {
	bus := &recordingBus{publishErr: errors.New("audit down")}
	svc := userService{
		auditBus: bus,
		logger:   zap.NewNop(),
	}
	authRepo := &moduleTestAuthRepository{}

	err := svc.ResetUserPassword(context.Background(), authRepo, passwordHasher{cost: 4}, passwordPolicy{}, 9, "Password1234")
	if err != nil {
		t.Fatalf("reset user password: %v", err)
	}
	if len(bus.published) != 1 {
		t.Fatalf("expected audit publish attempt, got %d", len(bus.published))
	}
}
