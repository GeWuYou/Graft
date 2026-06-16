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

	for _, registration := range []i18n.Registration{
		containerMessageRegistration(i18n.LocaleZHCN, zhCNCopyIndex),
		containerMessageRegistration(i18n.LocaleENUS, enUSCopyIndex),
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register container module messages: %w", err)
		}
	}
	return nil
}

func containerMessageRegistration(locale i18n.LocaleTag, copyIndex int) i18n.Registration {
	return i18n.Registration{
		Namespace: "container",
		Locale:    locale,
		Messages:  containerMessageResources(copyIndex),
	}
}

func containerMessageResources(copyIndex int) []i18n.MessageResource {
	messages := make([]i18n.MessageResource, 0, len(containerMessageCopyRows))
	for _, row := range containerMessageCopyRows {
		messages = append(messages, i18n.MessageResource{Key: i18n.MessageKey(row.key), Text: row.copy[copyIndex]})
	}
	return messages
}

type containerMessageCopyRow struct {
	key  string
	copy [2]string
}

const (
	zhCNCopyIndex = iota
	enUSCopyIndex
)

var containerMessageCopyRows = []containerMessageCopyRow{
	{key: containercontract.OperationsMenuTitle.String(), copy: [2]string{"运维管理", "Operations"}},
	{key: containercontract.ContainerMenuTitle.String(), copy: [2]string{"容器管理", "Container Management"}},
	{key: containercontract.ContainerRuntimeDisabled.String(), copy: [2]string{"容器运行时访问未启用", "Container runtime access is not enabled"}},
	{key: containercontract.ContainerRuntimeSocketMissing.String(), copy: [2]string{"容器运行时 socket 不存在", "Container runtime socket is missing"}},
	{key: containercontract.ContainerRuntimePermissionDenied.String(), copy: [2]string{"当前进程无权访问容器运行时", "The current process cannot access the container runtime"}},
	{key: containercontract.ContainerRuntimeUnavailable.String(), copy: [2]string{"容器运行时不可用", "Container runtime is unavailable"}},
	{key: containercontract.ContainerNotFound.String(), copy: [2]string{"容器不存在", "Container not found"}},
	{key: containercontract.ContainerInvalidRef.String(), copy: [2]string{"容器标识不合法", "Invalid container reference"}},
	{key: containercontract.ContainerInvalidListQuery.String(), copy: [2]string{"容器列表查询参数不合法", "Invalid container list query parameter"}},
	{key: containercontract.ContainerInvalidBatchAction.String(), copy: [2]string{"批量容器操作请求不合法", "Invalid container batch action request"}},
	{key: containercontract.ContainerInvalidState.String(), copy: [2]string{"容器当前状态不允许执行该操作", "The container state does not allow this action"}},
	{key: containercontract.ContainerLogsTooLarge.String(), copy: [2]string{"日志读取数量超过限制", "Requested log tail exceeds the configured limit"}},
	{key: containercontract.ContainerInvalidLogQuery.String(), copy: [2]string{"日志查询参数不合法", "Invalid container log query parameter"}},
	{key: containercontract.ContainerTimeout.String(), copy: [2]string{"容器运行时操作超时", "Container runtime operation timed out"}},
	{key: containercontract.ContainerDangerousActionsDisabled.String(), copy: [2]string{"高危容器操作未启用", "Dangerous container actions are disabled"}},
	{key: containercontract.ContainerActionStartCompleted.String(), copy: [2]string{"容器启动操作已完成", "Container start action completed"}},
	{key: containercontract.ContainerActionStopCompleted.String(), copy: [2]string{"容器停止操作已完成", "Container stop action completed"}},
	{key: containercontract.ContainerActionRestartCompleted.String(), copy: [2]string{"容器重启操作已完成", "Container restart action completed"}},
	{key: containercontract.ContainerActionRemoveCompleted.String(), copy: [2]string{"容器删除操作已完成", "Container remove action completed"}},
	{key: containercontract.ContainerBatchActionCompleted.String(), copy: [2]string{"批量容器操作已完成", "Container batch action completed"}},
	{key: containercontract.ContainerBatchActionPartial.String(), copy: [2]string{"批量容器操作部分完成", "Container batch action partially completed"}},
	{key: containercontract.ContainerBatchActionFailed.String(), copy: [2]string{"批量容器操作全部失败", "Container batch action failed"}},
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
			Name:           "View Containers",
			DisplayKey:     "rbac.permissionCatalog.containerView.display",
			Description:    "Allows reading the container list.",
			DescriptionKey: "rbac.permissionCatalog.containerView.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerDetailPermission.String(),
			Name:           "View Container Details",
			DisplayKey:     "rbac.permissionCatalog.containerDetail.display",
			Description:    "Allows reading container details.",
			DescriptionKey: "rbac.permissionCatalog.containerDetail.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerEnvironmentPermission.String(),
			Name:           "Read Container Environment Values",
			DisplayKey:     "rbac.permissionCatalog.containerEnvironment.display",
			Description:    "Allows reading container environment variable values when the display policy permits them.",
			DescriptionKey: "rbac.permissionCatalog.containerEnvironment.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerLogsPermission.String(),
			Name:           "Read Container Logs",
			DisplayKey:     "rbac.permissionCatalog.containerLogs.display",
			Description:    "Allows reading bounded container logs.",
			DescriptionKey: "rbac.permissionCatalog.containerLogs.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerStartPermission.String(),
			Name:           "Start Containers",
			DisplayKey:     "rbac.permissionCatalog.containerStart.display",
			Description:    "Allows starting containers when dangerous actions are enabled.",
			DescriptionKey: "rbac.permissionCatalog.containerStart.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerStopPermission.String(),
			Name:           "Stop Containers",
			DisplayKey:     "rbac.permissionCatalog.containerStop.display",
			Description:    "Allows stopping containers when dangerous actions are enabled.",
			DescriptionKey: "rbac.permissionCatalog.containerStop.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerRestartPermission.String(),
			Name:           "Restart Containers",
			DisplayKey:     "rbac.permissionCatalog.containerRestart.display",
			Description:    "Allows restarting containers when dangerous actions are enabled.",
			DescriptionKey: "rbac.permissionCatalog.containerRestart.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerRemovePermission.String(),
			Name:           "Remove Containers",
			DisplayKey:     "rbac.permissionCatalog.containerRemove.display",
			Description:    "Allows removing containers when dangerous actions are enabled.",
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
		Title:      "运维管理",
		TitleKey:   containercontract.OperationsMenuTitle.String(),
		Path:       containercontract.ContainerMenuRootPath,
		Icon:       "tools",
		Order:      operationsMenuOrderRoot,
		Permission: "",
		Module:     moduleName,
	})
	registry.Register(menu.Item{
		Code:                     "container.list",
		Title:                    "容器管理",
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
