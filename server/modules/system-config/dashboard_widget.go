// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package systemconfig

import (
	"fmt"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	systemconfigcontract "graft/server/modules/system-config/contract"
)

const (
	systemConfigQuickLinkID    = "system-config.settings"
	systemConfigQuickLinkOrder = 115
)

func registerSystemConfigDashboardQuickLink(ctx *module.Context, moduleName string) error {
	if ctx == nil || ctx.DashboardRegistry == nil {
		return nil
	}

	if err := ctx.DashboardRegistry.RegisterQuickLink(dashboard.QuickLinkDefinition{
		ID:                  systemConfigQuickLinkID,
		ModuleKey:           moduleName,
		TitleKey:            systemconfigcontract.SystemConfigMenuTitle.String(),
		Title:               "System Configuration",
		Icon:                "setting",
		RouteLocation:       systemconfigcontract.SystemConfigMenuPath,
		RequiredPermissions: []string{systemconfigcontract.SystemConfigReadPermission.String()},
		Order:               systemConfigQuickLinkOrder,
	}); err != nil {
		return fmt.Errorf("register system-config dashboard quick link: %w", err)
	}

	return nil
}
