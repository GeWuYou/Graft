// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"errors"
	"fmt"

	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/permission"
	notificationcontract "graft/server/modules/notification/contract"
)

func registerNotificationMetadata(ctx *module.Context) error {
	if err := registerNotificationMessages(ctx.I18n); err != nil {
		return err
	}
	return registerNotificationPermissions(ctx.PermissionRegistry, moduleID)
}

func registerNotificationMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, registration := range []i18n.Registration{
		{
			Namespace: "notification",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(notificationcontract.NotificationMenuTitle.String()), Text: "通知中心"},
			},
		},
		{
			Namespace: "notification",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(notificationcontract.NotificationMenuTitle.String()), Text: "Notification Center"},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register notification module messages: %w", err)
		}
	}

	return nil
}

func registerNotificationPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}

	registry.Register(permission.Item{
		Code:           notificationcontract.NotificationViewPermission.String(),
		Name:           "View Notifications",
		DisplayKey:     "rbac.permissionCatalog.notificationView.display",
		Description:    "Allows reading current-user notifications and unread counts.",
		DescriptionKey: "rbac.permissionCatalog.notificationView.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           notificationcontract.NotificationReadPermission.String(),
		Name:           "Read Notifications",
		DisplayKey:     "rbac.permissionCatalog.notificationRead.display",
		Description:    "Allows marking current-user notifications as read or deleting current-user deliveries.",
		DescriptionKey: "rbac.permissionCatalog.notificationRead.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           notificationcontract.NotificationManagePermission.String(),
		Name:           "Manage Notifications",
		DisplayKey:     "rbac.permissionCatalog.notificationManage.display",
		Description:    "Reserved for future global notification delivery management.",
		DescriptionKey: "rbac.permissionCatalog.notificationManage.description",
		Category:       "api",
		Module:         moduleName,
	})
	return nil
}
