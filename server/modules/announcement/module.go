// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package announcement

import (
	"errors"
	"fmt"

	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	announcementcontract "graft/server/modules/announcement/contract"
)

// Module owns announcement management and current-user read APIs.
type Module struct {
	service *Service
}

// NewModule creates the announcement module.
func NewModule(service *Service) *Module {
	return &Module{service: service}
}

// Register declares announcement metadata and routes.
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("announcement module service is unavailable")
	}
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	if err := registerAnnouncementPermissions(ctx.PermissionRegistry, moduleID); err != nil {
		return err
	}
	if err := registerAnnouncementMenu(ctx.MenuRegistry, moduleID); err != nil {
		return err
	}
	if ctx.Router == nil {
		return nil
	}
	return m.registerRoutes(ctx)
}

func (m *Module) registerRoutes(ctx *module.Context) error {
	authService, err := resolveAuthService(ctx)
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return fmt.Errorf("resolve authorizer: %w", err)
	}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	return registerAnnouncementRoutes(ctx, m.service, announcementGuards{
		authenticated: httpx.RequirePermission(ctx.I18n, authService, nil, "", publisher),
		read:          httpx.RequirePermission(ctx.I18n, authService, authorizer, announcementcontract.AnnouncementReadPermission.String(), publisher),
		create:        httpx.RequirePermission(ctx.I18n, authService, authorizer, announcementcontract.AnnouncementCreatePermission.String(), publisher),
		update:        httpx.RequirePermission(ctx.I18n, authService, authorizer, announcementcontract.AnnouncementUpdatePermission.String(), publisher),
		publish:       httpx.RequirePermission(ctx.I18n, authService, authorizer, announcementcontract.AnnouncementPublishPermission.String(), publisher),
		delete:        httpx.RequirePermission(ctx.I18n, authService, authorizer, announcementcontract.AnnouncementDeletePermission.String(), publisher),
	})
}

// Boot currently has no runtime work.
func (m *Module) Boot(_ *module.Context) error {
	return nil
}

// Shutdown currently has no runtime resources to release.
func (m *Module) Shutdown(_ *module.Context) error {
	return nil
}

func resolveAuthService(ctx *module.Context) (moduleapi.AuthService, error) {
	return module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
}

func resolveAuthorizer(ctx *module.Context) (moduleapi.Authorizer, error) {
	return module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
}
