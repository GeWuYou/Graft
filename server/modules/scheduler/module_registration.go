// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"errors"
	"fmt"

	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	schedulercontract "graft/server/modules/scheduler/contract"
)

const scheduledTaskMenuOrder = 104

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, registration := range []i18n.Registration{
		{
			Namespace: "scheduler",
			Locale:    i18n.LocaleZHCN,
			Messages: schedulerMessageResources([]string{
				"定时任务",
				"定时任务不存在",
				"定时任务正在运行",
				"定时任务请求无效",
				"定时任务失败",
				"定时任务执行失败。",
			}),
		},
		{
			Namespace: "scheduler",
			Locale:    i18n.LocaleENUS,
			Messages: schedulerMessageResources([]string{
				"Scheduled Tasks",
				"Scheduled task not found",
				"Scheduled task is already running",
				"Invalid scheduled task request",
				"Scheduled Task Failed",
				"Scheduled task failed.",
			}),
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register scheduler module messages: %w", err)
		}
	}

	return nil
}

func schedulerMessageResources(texts []string) []i18n.MessageResource {
	keys := []schedulercontract.MessageKey{
		schedulercontract.ScheduledTaskMenuTitle,
		schedulercontract.ScheduledTaskNotFound,
		schedulercontract.ScheduledTaskAlreadyRunning,
		schedulercontract.ScheduledTaskInvalidRequest,
		schedulercontract.ScheduledTaskRunFailedNotificationTitle,
		schedulercontract.ScheduledTaskRunFailedNotificationMessage,
	}
	resources := make([]i18n.MessageResource, 0, len(keys))
	for index, key := range keys {
		resources = append(resources, i18n.MessageResource{
			Key:  i18n.MessageKey(key.String()),
			Text: texts[index],
		})
	}
	return resources
}

func registerSchedulerPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}

	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskReadPermission.String(),
		Name:           "Read Scheduled Tasks",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskRead.display",
		Description:    "Allows reading scheduled task runtime data and run history.",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskRead.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskCreatePermission.String(),
		Name:           "Create Scheduled Tasks",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskCreate.display",
		Description:    "Allows creating user-managed scheduled task instances.",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskCreate.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskUpdatePermission.String(),
		Name:           "Update Scheduled Tasks",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskUpdate.display",
		Description:    "Allows updating scheduled task definitions.",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskUpdate.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskDeletePermission.String(),
		Name:           "Delete Scheduled Tasks",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskDelete.display",
		Description:    "Allows deleting user-managed scheduled task instances.",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskDelete.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskRunPermission.String(),
		Name:           "Run Scheduled Tasks",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskRun.display",
		Description:    "Allows manually running scheduled task runtime jobs.",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskRun.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskEnablePermission.String(),
		Name:           "Enable Scheduled Tasks",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskEnable.display",
		Description:    "Allows enabling and disabling scheduled tasks.",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskEnable.description",
		Category:       "api",
		Module:         moduleName,
	})
	return nil
}

func registerSchedulerMenu(registry *menu.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}

	registry.Register(menu.Item{
		Code:       "scheduled-task.list",
		Title:      "定时任务",
		TitleKey:   schedulercontract.ScheduledTaskMenuTitle.String(),
		Path:       schedulercontract.ScheduledTaskMenuPath,
		Icon:       "time",
		Order:      scheduledTaskMenuOrder,
		Permission: schedulercontract.ScheduledTaskReadPermission.String(),
		Module:     moduleName,
	})
	return nil
}
