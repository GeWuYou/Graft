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
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(schedulercontract.ScheduledTaskMenuTitle.String()), Text: "定时任务"},
				{Key: i18n.MessageKey(schedulercontract.ScheduledTaskNotFound.String()), Text: "定时任务不存在"},
				{Key: i18n.MessageKey(schedulercontract.ScheduledTaskAlreadyRunning.String()), Text: "定时任务正在运行"},
				{Key: i18n.MessageKey(schedulercontract.ScheduledTaskInvalidRequest.String()), Text: "定时任务请求无效"},
			},
		},
		{
			Namespace: "scheduler",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(schedulercontract.ScheduledTaskMenuTitle.String()), Text: "Scheduled Tasks"},
				{Key: i18n.MessageKey(schedulercontract.ScheduledTaskNotFound.String()), Text: "Scheduled task not found"},
				{Key: i18n.MessageKey(schedulercontract.ScheduledTaskAlreadyRunning.String()), Text: "Scheduled task is already running"},
				{Key: i18n.MessageKey(schedulercontract.ScheduledTaskInvalidRequest.String()), Text: "Invalid scheduled task request"},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register scheduler module messages: %w", err)
		}
	}

	return nil
}

func registerSchedulerPermissions(registry *permission.Registry, moduleName string) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:        schedulercontract.ScheduledTaskReadPermission.String(),
		Name:        "Read Scheduled Tasks",
		Description: "Allows reading scheduled task runtime data and run history.",
		Category:    "api",
		Module:      moduleName,
	})
	registry.Register(permission.Item{
		Code:        schedulercontract.ScheduledTaskCreatePermission.String(),
		Name:        "Create Scheduled Tasks",
		Description: "Allows creating user-managed HTTP scheduled tasks.",
		Category:    "api",
		Module:      moduleName,
	})
	registry.Register(permission.Item{
		Code:        schedulercontract.ScheduledTaskUpdatePermission.String(),
		Name:        "Update Scheduled Tasks",
		Description: "Allows updating scheduled task definitions.",
		Category:    "api",
		Module:      moduleName,
	})
	registry.Register(permission.Item{
		Code:        schedulercontract.ScheduledTaskDeletePermission.String(),
		Name:        "Delete Scheduled Tasks",
		Description: "Allows deleting user-managed HTTP scheduled tasks.",
		Category:    "api",
		Module:      moduleName,
	})
	registry.Register(permission.Item{
		Code:        schedulercontract.ScheduledTaskRunPermission.String(),
		Name:        "Run Scheduled Tasks",
		Description: "Allows manually running scheduled task runtime jobs.",
		Category:    "api",
		Module:      moduleName,
	})
	registry.Register(permission.Item{
		Code:        schedulercontract.ScheduledTaskEnablePermission.String(),
		Name:        "Enable Scheduled Tasks",
		Description: "Allows enabling and disabling scheduled tasks.",
		Category:    "api",
		Module:      moduleName,
	})
}

func registerSchedulerMenu(registry *menu.Registry, moduleName string) {
	if registry == nil {
		return
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
}
