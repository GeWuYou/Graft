// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"errors"

	"graft/server/internal/permission"
	notificationcontract "graft/server/modules/notification/contract"
)

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
