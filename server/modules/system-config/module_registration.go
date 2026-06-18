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

	for _, registration := range []i18n.Registration{
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleZHCN,
			Messages: systemConfigBaseMessages("系统配置", "系统配置不存在", "系统配置请求无效"),
		},
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleENUS,
			Messages: systemConfigBaseMessages(
				"System Configuration",
				"System Configuration Not Found",
				"Invalid System Configuration Request",
			),
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register system-config module messages: %w", err)
		}
	}
	return nil
}

func systemConfigBaseMessages(menuTitle, notFound, invalidRequest string) []i18n.MessageResource {
	return []i18n.MessageResource{
		{Key: i18n.MessageKey(systemconfigcontract.SystemConfigMenuTitle.String()), Text: menuTitle},
		{Key: i18n.MessageKey(systemconfigcontract.SystemConfigNotFound.String()), Text: notFound},
		{Key: i18n.MessageKey(systemconfigcontract.SystemConfigInvalidRequest.String()), Text: invalidRequest},
	}
}

func registerSystemConfigPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}

	registry.Register(permission.Item{
		Code:           systemconfigcontract.SystemConfigReadPermission.String(),
		Name:           "Read System Configuration",
		DisplayKey:     "rbac.permissionCatalog.systemConfigRead.display",
		Description:    "Allows reading registered system configuration definitions and effective values.",
		DescriptionKey: "rbac.permissionCatalog.systemConfigRead.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           systemconfigcontract.SystemConfigWritePermission.String(),
		Name:           "Update System Configuration",
		DisplayKey:     "rbac.permissionCatalog.systemConfigWrite.display",
		Description:    "Allows writing and resetting user configuration overrides.",
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
		Title:      "系统配置",
		TitleKey:   systemconfigcontract.SystemConfigMenuTitle.String(),
		Path:       systemconfigcontract.SystemConfigMenuPath,
		Icon:       "setting",
		Order:      systemConfigMenuOrder,
		Permission: systemconfigcontract.SystemConfigReadPermission.String(),
		Module:     moduleName,
	})
	return nil
}
