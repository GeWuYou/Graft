// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package announcement

import (
	"errors"
	"fmt"

	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	announcementcontract "graft/server/modules/announcement/contract"
)

const announcementMenuOrder = 106

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}
	for _, registration := range []i18n.Registration{
		{
			Namespace: "announcement",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(announcementcontract.AnnouncementMenuTitle.String()), Text: "公告管理"},
			},
		},
		{
			Namespace: "announcement",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(announcementcontract.AnnouncementMenuTitle.String()), Text: "Announcements"},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register announcement module messages: %w", err)
		}
	}
	return nil
}

func registerAnnouncementPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}
	for _, item := range []permission.Item{
		{
			Code:           announcementcontract.AnnouncementReadPermission.String(),
			Name:           "Read Announcements",
			DisplayKey:     "rbac.permissionCatalog.announcementRead.display",
			Description:    "Allows reading announcement management records.",
			DescriptionKey: "rbac.permissionCatalog.announcementRead.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           announcementcontract.AnnouncementCreatePermission.String(),
			Name:           "Create Announcements",
			DisplayKey:     "rbac.permissionCatalog.announcementCreate.display",
			Description:    "Allows creating announcement drafts.",
			DescriptionKey: "rbac.permissionCatalog.announcementCreate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           announcementcontract.AnnouncementUpdatePermission.String(),
			Name:           "Update Announcements",
			DisplayKey:     "rbac.permissionCatalog.announcementUpdate.display",
			Description:    "Allows updating announcement drafts and management metadata.",
			DescriptionKey: "rbac.permissionCatalog.announcementUpdate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           announcementcontract.AnnouncementPublishPermission.String(),
			Name:           "Publish Announcements",
			DisplayKey:     "rbac.permissionCatalog.announcementPublish.display",
			Description:    "Allows publishing and archiving announcements.",
			DescriptionKey: "rbac.permissionCatalog.announcementPublish.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           announcementcontract.AnnouncementDeletePermission.String(),
			Name:           "Delete Announcements",
			DisplayKey:     "rbac.permissionCatalog.announcementDelete.display",
			Description:    "Allows soft-deleting announcement records.",
			DescriptionKey: "rbac.permissionCatalog.announcementDelete.description",
			Category:       "api",
			Module:         moduleName,
		},
	} {
		registry.Register(item)
	}
	return nil
}

func registerAnnouncementMenu(registry *menu.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}
	// Title 仅作为旧消费方的展示兜底，长期标题真相由 TitleKey 对接 i18n。
	registry.Register(menu.Item{
		Code:       "announcement.list",
		Title:      "公告管理",
		TitleKey:   announcementcontract.AnnouncementMenuTitle.String(),
		Path:       announcementcontract.AnnouncementMenuPath,
		Icon:       "notification",
		Order:      announcementMenuOrder,
		Permission: announcementcontract.AnnouncementReadPermission.String(),
		Module:     moduleName,
	})
	return nil
}
