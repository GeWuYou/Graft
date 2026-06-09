// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"errors"

	"graft/server/internal/container"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	notificationcontract "graft/server/modules/notification/contract"
)

// Module is the Notification Center backend module.
type Module struct {
	service   *Service
	publisher moduleapi.NotificationPublisher
}

// NewModule creates a Notification Center module instance.
func NewModule(service *Service, publisher moduleapi.NotificationPublisher) *Module {
	return &Module{service: service, publisher: publisher}
}

// Register declares notification permissions and the cross-module publisher capability.
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil || m.publisher == nil {
		return errors.New("notification module dependencies are unavailable")
	}
	if ctx == nil || ctx.Services == nil {
		return errors.New("notification register context is required")
	}

	if err := registerNotificationPermissions(ctx.PermissionRegistry, moduleID); err != nil {
		return err
	}
	if ctx.Router != nil {
		authService, err := resolveAuthService(ctx)
		if err != nil {
			return err
		}
		authorizer, err := resolveAuthorizer(ctx)
		if err != nil {
			return err
		}
		publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
		registerNotificationRoutes(ctx, m.service, notificationGuards{
			view: httpx.RequirePermission(
				ctx.I18n,
				authService,
				authorizer,
				notificationcontract.NotificationViewPermission.String(),
				publisher,
			),
			read: httpx.RequirePermission(
				ctx.I18n,
				authService,
				authorizer,
				notificationcontract.NotificationReadPermission.String(),
				publisher,
			),
		})
	}
	return ctx.Services.RegisterSingleton((*moduleapi.NotificationPublisher)(nil), func(_ container.Resolver) (any, error) {
		return m.publisher, nil
	})
}

// Boot currently has no background behavior to start.
func (m *Module) Boot(_ *module.Context) error {
	return nil
}

// Shutdown currently has no runtime resources to release.
func (m *Module) Shutdown(_ *module.Context) error {
	return nil
}

func resolveAuthService(ctx *module.Context) (moduleapi.AuthService, error) {
	resolved, err := ctx.Services.Resolve((*moduleapi.AuthService)(nil))
	if err != nil {
		return nil, err
	}
	authService, ok := resolved.(moduleapi.AuthService)
	if !ok || authService == nil {
		return nil, errors.New("notification auth service has unexpected type")
	}
	return authService, nil
}

func resolveAuthorizer(ctx *module.Context) (moduleapi.Authorizer, error) {
	resolved, err := ctx.Services.Resolve((*moduleapi.Authorizer)(nil))
	if err != nil {
		return nil, err
	}
	authorizer, ok := resolved.(moduleapi.Authorizer)
	if !ok || authorizer == nil {
		return nil, errors.New("notification authorizer has unexpected type")
	}
	return authorizer, nil
}
