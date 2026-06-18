// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"errors"
	"fmt"

	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	containercontract "graft/server/modules/container/contract"
)

const (
	operationsMenuOrderRoot = 50
	containerMenuOrderList  = 51
)

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range containerLocaleBackedMessageKeys() {
			matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
			if len(matches) == 0 {
				return fmt.Errorf("register container module messages: locale resource %s missing key %s", locale, key)
			}
		}
	}
	return nil
}

func containerLocaleBackedMessageKeys() []string {
	keys := make([]string, 0, len(containerMessageKeys))
	for _, key := range containerMessageKeys {
		if key == containercontract.OperationsMenuTitle.String() || key == containercontract.ContainerMenuTitle.String() {
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

var containerMessageKeys = []string{
	containercontract.OperationsMenuTitle.String(),
	containercontract.ContainerMenuTitle.String(),
	containercontract.ContainerRuntimeDisabled.String(),
	containercontract.ContainerRuntimeSocketMissing.String(),
	containercontract.ContainerRuntimePermissionDenied.String(),
	containercontract.ContainerRuntimeUnavailable.String(),
	containercontract.ContainerNotFound.String(),
	containercontract.ContainerMountNotFound.String(),
	containercontract.ContainerInvalidRef.String(),
	containercontract.ContainerInvalidListQuery.String(),
	containercontract.ContainerInvalidBatchAction.String(),
	containercontract.ContainerInvalidState.String(),
	containercontract.ContainerLogsTooLarge.String(),
	containercontract.ContainerInvalidLogQuery.String(),
	containercontract.ContainerTimeout.String(),
	containercontract.ContainerMountUsageUnsupported.String(),
	containercontract.ContainerDangerousActionsDisabled.String(),
	containercontract.ContainerActionStartCompleted.String(),
	containercontract.ContainerActionStopCompleted.String(),
	containercontract.ContainerActionRestartCompleted.String(),
	containercontract.ContainerActionRemoveCompleted.String(),
	containercontract.ContainerBatchActionCompleted.String(),
	containercontract.ContainerBatchActionPartial.String(),
	containercontract.ContainerBatchActionFailed.String(),
}

func registerPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}

	for _, item := range permissionItems(moduleName) {
		registry.Register(item)
	}
	return nil
}

func permissionItems(moduleName string) []permission.Item {
	return []permission.Item{
		{
			Code:           containercontract.ContainerViewPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerView.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerView.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerDetailPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerDetail.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerDetail.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerEnvironmentPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerEnvironment.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerEnvironment.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerLogsPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerLogs.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerLogs.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerStartPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerStart.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerStart.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerStopPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerStop.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerStop.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerRestartPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerRestart.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerRestart.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerRemovePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerRemove.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerRemove.description",
			Category:       "api",
			Module:         moduleName,
		},
	}
}

func registerMenu(registry *menu.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}

	registry.Register(menu.Item{
		Code:       "ops.root",
		Title:      "",
		TitleKey:   containercontract.OperationsMenuTitle.String(),
		Path:       containercontract.ContainerMenuRootPath,
		Icon:       "tools",
		Order:      operationsMenuOrderRoot,
		Permission: "",
		Module:     moduleName,
	})
	registry.Register(menu.Item{
		Code:                     "container.list",
		Title:                    "",
		TitleKey:                 containercontract.ContainerMenuTitle.String(),
		Path:                     containercontract.ContainerMenuPath,
		Icon:                     "server",
		Order:                    containerMenuOrderList,
		Permission:               containercontract.ContainerViewPermission.String(),
		VisibleWhenConfigEnabled: containercontract.ContainerRuntimeEnabledConfig.String(),
		Module:                   moduleName,
	})
	return nil
}
