// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package systemconfig

import (
	"errors"
	"fmt"

	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	systemconfigcontract "graft/server/modules/system-config/contract"
)

const systemConfigMenuOrder = 105

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range []systemconfigcontract.MessageKey{
			systemconfigcontract.SystemConfigMenuTitle,
			systemconfigcontract.SystemConfigNotFound,
			systemconfigcontract.SystemConfigInvalidRequest,
		} {
			matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key.String()))
			if len(matches) == 0 {
				return fmt.Errorf("register system-config module messages: locale resource %s missing key %s", locale, key)
			}
		}
	}
	return nil
}

func registerSystemConfigPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}

	registry.Register(permission.Item{
		Code:           systemconfigcontract.SystemConfigReadPermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.systemConfigRead.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.systemConfigRead.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           systemconfigcontract.SystemConfigWritePermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.systemConfigWrite.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.systemConfigWrite.description",
		Category:       "api",
		Module:         moduleName,
	})
	return nil
}

func registerSystemConfigMenu(registry *menu.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}

	registry.Register(menu.Item{
		Code:       "system-config.list",
		Title:      "",
		TitleKey:   systemconfigcontract.SystemConfigMenuTitle.String(),
		Path:       systemconfigcontract.SystemConfigMenuPath,
		Icon:       "setting",
		Order:      systemConfigMenuOrder,
		Permission: systemconfigcontract.SystemConfigReadPermission.String(),
		Module:     moduleName,
	})
	return nil
}
