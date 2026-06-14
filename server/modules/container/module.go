// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"errors"

	"graft/server/internal/module"
)

// Module declares the container management module foundation.
type Module struct {
	service *service
}

// NewModule creates a container management module instance.
func NewModule() *Module {
	return &Module{}
}

// Register declares container menu, permissions, messages, config definitions, and routes.
func (m *Module) Register(ctx *module.Context) error {
	if m == nil {
		return errors.New("container module is unavailable")
	}
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	if err := registerPermissions(ctx.PermissionRegistry, moduleID); err != nil {
		return err
	}
	if err := registerMenu(ctx.MenuRegistry, moduleID); err != nil {
		return err
	}
	if err := registerConfig(ctx.I18n, ctx.ConfigRegistry); err != nil {
		return err
	}
	service, err := newContainerService(ctx, moduleID)
	if err != nil {
		return err
	}
	m.service = service
	return registerRoutes(ctx, moduleID, service)
}

// Boot currently has no background runtime work; container reads are request-driven.
func (m *Module) Boot(_ *module.Context) error {
	return nil
}

// Shutdown releases the runtime client owned by this module.
func (m *Module) Shutdown(_ *module.Context) error {
	if m == nil || m.service == nil {
		return nil
	}
	if err := m.service.Close(); err != nil {
		return err
	}
	return nil
}
