// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package systemconfig

import (
	"errors"
	"fmt"

	"graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	schedulercore "graft/server/internal/scheduler"
)

const moduleID = "system-config"

// Module owns system configuration user overrides and HTTP management.
type Module struct {
	service *Service
}

// NewModule creates the system configuration module.
func NewModule(service *Service) (*Module, error) {
	if service == nil {
		return nil, errors.New("system config service is unavailable")
	}
	return &Module{service: service}, nil
}

// Register declares permissions, menu, messages, and management routes.
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("system config module service is unavailable")
	}
	userService, err := requiredUserService(ctx.Services)
	if err != nil {
		return fmt.Errorf("resolve user service: %w", err)
	}
	m.service.setUserService(userService)
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	if err := registerSystemConfigPermissions(ctx.PermissionRegistry, moduleID); err != nil {
		return err
	}
	if err := registerSystemConfigMenu(ctx.MenuRegistry, moduleID); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*schedulercore.DefaultConfigResolver)(nil), func(_ container.Resolver) (any, error) {
		return m.service, nil
	}); err != nil {
		return fmt.Errorf("register system-config default resolver: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.SystemConfigResolver)(nil), func(_ container.Resolver) (any, error) {
		return m.service, nil
	}); err != nil {
		return fmt.Errorf("register system-config resolver: %w", err)
	}
	return registerSystemConfigRoutes(ctx, moduleID, m.service)
}

func requiredUserService(resolver container.Resolver) (moduleapi.UserService, error) {
	return module.ResolveService[moduleapi.UserService](resolver, (*moduleapi.UserService)(nil))
}

// Boot does not start extra runtime mechanics because cachex handles shared snapshot storage.
func (m *Module) Boot(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("system config module service is unavailable")
	}
	if ctx == nil {
		return errors.New("system config module context is unavailable")
	}
	return nil
}

// Shutdown has no module-local background resources to release.
func (m *Module) Shutdown(_ *module.Context) error {
	return nil
}
