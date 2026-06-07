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

type systemConfigFieldMessages struct {
	retentionDaysTitle       string
	retentionDaysDescription string
	batchSizeTitle           string
	batchSizeDescription     string
	daysUnit                 string
	rowsUnit                 string
}

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, registration := range []i18n.Registration{
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleZHCN,
			Messages: systemConfigMessages(
				systemConfigBaseMessages("系统配置", "系统配置不存在", "系统配置请求无效"),
				systemConfigFieldMessages{
					retentionDaysTitle:       "日志保留时间",
					retentionDaysDescription: "删除早于指定天数的日志。",
					batchSizeTitle:           "批量大小",
					batchSizeDescription:     "单次清理最多删除的日志行数。",
					daysUnit:                 "天",
					rowsUnit:                 "行",
				},
			),
		},
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleENUS,
			Messages: systemConfigMessages(
				systemConfigBaseMessages(
					"System Configuration",
					"System Configuration Not Found",
					"Invalid System Configuration Request",
				),
				systemConfigFieldMessages{
					retentionDaysTitle:       "Log Retention Days",
					retentionDaysDescription: "Delete logs older than this many days.",
					batchSizeTitle:           "Batch Size",
					batchSizeDescription:     "Maximum rows deleted per cleanup batch.",
					daysUnit:                 "days",
					rowsUnit:                 "rows",
				},
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

func systemConfigMessages(
	base []i18n.MessageResource,
	fieldMessages systemConfigFieldMessages,
) []i18n.MessageResource {
	return append(base,
		i18n.MessageResource{Key: i18n.MessageKey("systemConfig.fields.retentionDays.title"), Text: fieldMessages.retentionDaysTitle},
		i18n.MessageResource{
			Key:  i18n.MessageKey("systemConfig.fields.retentionDays.description"),
			Text: fieldMessages.retentionDaysDescription,
		},
		i18n.MessageResource{Key: i18n.MessageKey("systemConfig.fields.batchSize.title"), Text: fieldMessages.batchSizeTitle},
		i18n.MessageResource{
			Key:  i18n.MessageKey("systemConfig.fields.batchSize.description"),
			Text: fieldMessages.batchSizeDescription,
		},
		i18n.MessageResource{Key: i18n.MessageKey("systemConfig.units.days"), Text: fieldMessages.daysUnit},
		i18n.MessageResource{Key: i18n.MessageKey("systemConfig.units.rows"), Text: fieldMessages.rowsUnit},
	)
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
		Description:    "Allows writing and resetting administrator configuration overrides.",
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
