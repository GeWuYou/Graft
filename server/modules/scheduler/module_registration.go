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

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range schedulerMessageKeys() {
			matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key.String()))
			if len(matches) == 0 {
				return fmt.Errorf("register scheduler module messages: locale resource %s missing key %s", locale, key)
			}
		}
	}

	return nil
}

func schedulerMessageKeys() []schedulercontract.MessageKey {
	return []schedulercontract.MessageKey{
		schedulercontract.ScheduledTaskNotFound,
		schedulercontract.ScheduledTaskAlreadyRunning,
		schedulercontract.ScheduledTaskInvalidRequest,
		schedulercontract.ScheduledTaskRunFailedNotificationTitle,
		schedulercontract.ScheduledTaskRunFailedNotificationMessage,
		schedulercontract.ScheduledTaskRunSucceededNotificationTitle,
		schedulercontract.ScheduledTaskRunSucceededNotificationMessage,
	}
}

func registerSchedulerPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}

	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskReadPermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskRead.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskRead.description",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskCreatePermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskCreate.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskCreate.description",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskUpdatePermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskUpdate.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskUpdate.description",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskDeletePermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskDelete.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskDelete.description",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskRunPermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskRun.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskRun.description",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           schedulercontract.ScheduledTaskEnablePermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.scheduledTaskEnable.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.scheduledTaskEnable.description",
		Module:         moduleName,
	})
	return nil
}

// registerSchedulerMenu 为定时任务注册菜单项；registry 为 nil 时返回错误。
func registerSchedulerMenu(registry *menu.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}

	registry.Register(menu.Item{
		Code:       "scheduled-task.list",
		ParentCode: "domain.platform",
		Kind:       menu.NodeKindEntry,
		Title:      "",
		TitleKey:   schedulercontract.ScheduledTaskMenuTitle.String(),
		Path:       schedulercontract.ScheduledTaskMenuPath,
		Icon:       "scheduled-tasks",
		Order:      scheduledTaskMenuOrder,
		Permission: schedulercontract.ScheduledTaskReadPermission.String(),
		Module:     moduleName,
	})
	return nil
}
