// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"errors"

	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/permission"
	notificationcontract "graft/server/modules/notification/contract"
)

func registerNotificationMetadata(ctx *module.Context) error {
	if err := registerNotificationMessages(ctx.I18n); err != nil {
		return err
	}
	if err := registerNotificationPermissions(ctx.PermissionRegistry, moduleID); err != nil {
		return err
	}
	return registerNotificationConfig(ctx.I18n, ctx.ConfigRegistry)
}

func registerNotificationMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}
	return nil
}

func registerNotificationPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}

	registry.Register(permission.Item{
		Code:           notificationcontract.NotificationViewPermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.notificationView.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.notificationView.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           notificationcontract.NotificationReadPermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.notificationRead.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.notificationRead.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           notificationcontract.NotificationManagePermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.notificationManage.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.notificationManage.description",
		Category:       "api",
		Module:         moduleName,
	})
	return nil
}
