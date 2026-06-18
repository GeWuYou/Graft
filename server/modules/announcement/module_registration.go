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
	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		matches := localizer.RegisteredMessageResources(
			locale,
			i18n.MessageKey(announcementcontract.AnnouncementPublishedDeleteForbidden.String()),
		)
		if len(matches) == 0 {
			return fmt.Errorf(
				"register announcement module messages: locale resource %s missing key %s",
				locale,
				announcementcontract.AnnouncementPublishedDeleteForbidden.String(),
			)
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
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.announcementRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.announcementRead.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           announcementcontract.AnnouncementCreatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.announcementCreate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.announcementCreate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           announcementcontract.AnnouncementUpdatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.announcementUpdate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.announcementUpdate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           announcementcontract.AnnouncementPublishPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.announcementPublish.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.announcementPublish.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           announcementcontract.AnnouncementDeletePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.announcementDelete.display",
			Description:    "",
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
	registry.Register(menu.Item{
		Code:       "announcement.list",
		Title:      "",
		TitleKey:   announcementcontract.AnnouncementMenuTitle.String(),
		Path:       announcementcontract.AnnouncementMenuPath,
		Icon:       "notification",
		Order:      announcementMenuOrder,
		Permission: announcementcontract.AnnouncementReadPermission.String(),
		Module:     moduleName,
	})
	return nil
}
