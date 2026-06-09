// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"context"
	"testing"
	"time"

	"graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	notificationcontract "graft/server/modules/notification/contract"
	notificationstore "graft/server/modules/notification/store"
)

func TestModuleRegistersPermissionsAndPublisher(t *testing.T) {
	repository := &moduleTestRepository{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	publisher, err := NewPublisher(repository)
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}

	services := container.New()
	ctx := &module.Context{
		Services:           services,
		PermissionRegistry: permission.NewRegistry(),
	}
	if err := NewModule(service, publisher).Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}

	registered := make(map[string]struct{}, len(ctx.PermissionRegistry.Items()))
	for _, item := range ctx.PermissionRegistry.Items() {
		registered[item.Code] = struct{}{}
	}
	for _, code := range []string{
		notificationcontract.NotificationViewPermission.String(),
		notificationcontract.NotificationReadPermission.String(),
		notificationcontract.NotificationManagePermission.String(),
	} {
		if _, ok := registered[code]; !ok {
			t.Fatalf("expected permission %s to be registered", code)
		}
	}

	resolved, err := services.Resolve((*moduleapi.NotificationPublisher)(nil))
	if err != nil {
		t.Fatalf("resolve notification publisher: %v", err)
	}
	if _, ok := resolved.(moduleapi.NotificationPublisher); !ok {
		t.Fatalf("unexpected publisher service type %T", resolved)
	}
}

type moduleTestRepository struct{}

func (r *moduleTestRepository) CreateEvent(context.Context, notificationstore.CreateEventInput) (notificationstore.Event, bool, error) {
	return notificationstore.Event{}, false, nil
}

func (r *moduleTestRepository) CreateDeliveries(context.Context, []notificationstore.CreateDeliveryInput) ([]notificationstore.Delivery, error) {
	return nil, nil
}

func (r *moduleTestRepository) List(context.Context, notificationstore.ListQuery) (notificationstore.ListResult, error) {
	return notificationstore.ListResult{}, nil
}

func (r *moduleTestRepository) Get(context.Context, uint64, uint64) (notificationstore.Notification, error) {
	return notificationstore.Notification{}, nil
}

func (r *moduleTestRepository) UnreadCount(context.Context, uint64) (int, error) {
	return 0, nil
}

func (r *moduleTestRepository) MarkRead(context.Context, uint64, uint64, time.Time) (notificationstore.Delivery, error) {
	return notificationstore.Delivery{}, nil
}

func (r *moduleTestRepository) MarkAllRead(context.Context, uint64, time.Time) (int, error) {
	return 0, nil
}

func (r *moduleTestRepository) MarkAllReadMatching(context.Context, notificationstore.ListQuery, time.Time) (int, error) {
	return 0, nil
}

func (r *moduleTestRepository) DeleteDelivery(context.Context, uint64, uint64, time.Time) error {
	return nil
}
