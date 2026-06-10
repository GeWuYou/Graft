// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/configregistry"
	"graft/server/internal/i18n"
)

const (
	notificationConfigDomain            = "notification"
	notificationConfigDomainKey         = "systemConfig.domains.notification"
	notificationConfigGeneralGroup      = "notification.general"
	notificationConfigGeneralGroupKey   = "systemConfig.groups.notification.general"
	notificationConfigSourcesGroup      = "notification.sources"
	notificationConfigSourcesGroupKey   = "systemConfig.groups.notification.sources"
	notificationConfigDeliveryGroup     = "notification.delivery"
	notificationConfigDeliveryGroupKey  = "systemConfig.groups.notification.delivery"
	notificationConfigDisplayGroup      = "notification.display"
	notificationConfigDisplayGroupKey   = "systemConfig.groups.notification.display"
	notificationConfigGeneralDescKey    = "systemConfig.groups.notification.general.description"
	notificationConfigSourcesDescKey    = "systemConfig.groups.notification.sources.description"
	notificationConfigDeliveryDescKey   = "systemConfig.groups.notification.delivery.description"
	notificationConfigDisplayDescKey    = "systemConfig.groups.notification.display.description"
	notificationConfigDefinitionBaseOrd = 5100
)

const (
	defaultNotificationRetentionDays       = 30
	defaultNotificationPageSize            = 20
	defaultNotificationDedupeWindowSeconds = 300
	defaultNotificationMaxBatchRecipients  = 1000
	defaultNotificationDisplayShowReadDays = 7
	defaultNotificationDisplayPopupLimit   = 5
)

const (
	notificationEnabledKey                                = "notification.enabled"
	notificationRetentionDaysKey                          = "notification.retention_days"
	notificationDefaultPageSizeKey                        = "notification.default_page_size"
	notificationSourceSystemAnnouncementEnabledKey        = "notification.source.system_announcement.enabled"
	notificationSourceScheduledTaskFailureEnabledKey      = "notification.source.scheduled_task_failure.enabled"
	notificationSourceScheduledTaskSuccessEnabledKey      = "notification.source.scheduled_task_success.enabled"
	notificationSourceAuditIncidentEnabledKey             = "notification.source.audit_incident.enabled"
	notificationSourceSystemConfigChangeEnabledKey        = "notification.source.system_config_change.enabled"
	notificationSourceAccessLogRetentionFailureEnabledKey = "notification.source.access_log_retention_failure.enabled"
	notificationDeliveryInAppEnabledKey                   = "notification.delivery.in_app.enabled"
	notificationDeliveryDedupeWindowSecondsKey            = "notification.delivery.dedupe_window_seconds"
	notificationDeliveryMaxBatchRecipientsKey             = "notification.delivery.max_batch_recipients"
	notificationDisplayShowReadDaysKey                    = "notification.display.show_read_days"
	notificationDisplayPopupLimitKey                      = "notification.display.popup_limit"
)

// ConfigResolver resolves effective notification configuration values.
type ConfigResolver interface {
	Boolean(ctx context.Context, key string, fallback bool) bool
}

func registerNotificationConfig(localizer *i18n.Service, registry *configregistry.Registry) error {
	if err := registerNotificationConfigMessages(localizer); err != nil {
		return err
	}
	return registerNotificationConfigDefinitions(registry)
}

func registerNotificationConfigDefinitions(registry *configregistry.Registry) error {
	if registry == nil {
		return errors.New("config registry is required")
	}
	for index, definition := range notificationConfigDefinitions() {
		definition.Order = notificationConfigDefinitionBaseOrd + index
		if err := registry.Register(definition); err != nil {
			return fmt.Errorf("register notification config definition %s: %w", definition.Key, err)
		}
	}
	return nil
}

func notificationConfigDefinitions() []configregistry.Definition {
	return []configregistry.Definition{
		booleanNotificationDefinition(notificationEnabledKey, notificationConfigGeneralGroup, "Notification enabled", "Whether in-app notifications are enabled.", true),
		numberNotificationDefinition(notificationRetentionDaysKey, notificationConfigGeneralGroup, "Notification retention days", "Number of days notification records should be retained.", defaultNotificationRetentionDays),
		numberNotificationDefinition(notificationDefaultPageSizeKey, notificationConfigGeneralGroup, "Default page size", "Default page size used by Notification Center.", defaultNotificationPageSize),
		booleanNotificationDefinition(notificationSourceSystemAnnouncementEnabledKey, notificationConfigSourcesGroup, "System announcement notifications", "Enables notifications created from published system announcements.", true),
		booleanNotificationDefinition(notificationSourceScheduledTaskFailureEnabledKey, notificationConfigSourcesGroup, "Scheduled task failure notifications", "Enables notifications for failed scheduled task runs.", true),
		booleanNotificationDefinition(notificationSourceScheduledTaskSuccessEnabledKey, notificationConfigSourcesGroup, "Scheduled task success notifications", "Enables notifications for successful manual scheduled task runs.", false),
		booleanNotificationDefinition(notificationSourceAuditIncidentEnabledKey, notificationConfigSourcesGroup, "Audit incident notifications", "Enables notifications for audit events that require security review.", true),
		booleanNotificationDefinition(notificationSourceSystemConfigChangeEnabledKey, notificationConfigSourcesGroup, "System config change notifications", "Enables notifications for system configuration changes, resets, and validation failures.", true),
		booleanNotificationDefinition(notificationSourceAccessLogRetentionFailureEnabledKey, notificationConfigSourcesGroup, "Access log retention failure notifications", "Enables notifications for access-log retention cleanup failures.", true),
		booleanNotificationDefinition(notificationDeliveryInAppEnabledKey, notificationConfigDeliveryGroup, "In-app delivery enabled", "Whether Notification Center should create in-app delivery records.", true),
		numberNotificationDefinition(notificationDeliveryDedupeWindowSecondsKey, notificationConfigDeliveryGroup, "Dedupe window seconds", "Dedupe window for same source, target, and business object notifications.", defaultNotificationDedupeWindowSeconds),
		numberNotificationDefinition(notificationDeliveryMaxBatchRecipientsKey, notificationConfigDeliveryGroup, "Max batch recipients", "Maximum recipients in one notification delivery batch.", defaultNotificationMaxBatchRecipients),
		numberNotificationDefinition(notificationDisplayShowReadDaysKey, notificationConfigDisplayGroup, "Show read days", "Default time window for showing read notifications.", defaultNotificationDisplayShowReadDays),
		numberNotificationDefinition(notificationDisplayPopupLimitKey, notificationConfigDisplayGroup, "Popup limit", "Maximum notifications displayed by the notification bell popup.", defaultNotificationDisplayPopupLimit),
	}
}

func booleanNotificationDefinition(key string, group string, title string, description string, defaultValue bool) configregistry.Definition {
	return baseNotificationDefinition(key, group, title, description, configregistry.ValueTypeBoolean, mustRawJSON(defaultValue))
}

func numberNotificationDefinition(key string, group string, title string, description string, defaultValue int) configregistry.Definition {
	return baseNotificationDefinition(key, group, title, description, configregistry.ValueTypeInteger, mustRawJSON(defaultValue))
}

func baseNotificationDefinition(
	key string,
	group string,
	title string,
	description string,
	valueType configregistry.ValueType,
	defaultValue json.RawMessage,
) configregistry.Definition {
	metadata := notificationConfigGroupMetadata(group)
	return configregistry.Definition{
		Key:                 key,
		Module:              moduleID,
		Domain:              notificationConfigDomain,
		DomainKey:           notificationConfigDomainKey,
		DomainLabel:         "Notification",
		Group:               group,
		GroupKey:            metadata.key,
		GroupLabel:          metadata.label,
		GroupDescription:    metadata.Description,
		GroupDescriptionKey: metadata.DescriptionKey,
		Title:               title,
		TitleKey:            notificationConfigTitleKey(key),
		Description:         description,
		DescriptionKey:      notificationConfigDescriptionKey(key),
		Tags:                []string{"notification", group},
		Type:                valueType,
		DefaultValue:        defaultValue,
	}
}

type notificationConfigGroupInfo struct {
	key            string
	label          string
	DescriptionKey string
	Description    string
}

func notificationConfigGroupMetadata(group string) notificationConfigGroupInfo {
	switch group {
	case notificationConfigSourcesGroup:
		return notificationConfigGroupInfo{
			key:            notificationConfigSourcesGroupKey,
			label:          "Notification sources",
			DescriptionKey: notificationConfigSourcesDescKey,
			Description:    "Control which platform events create notifications.",
		}
	case notificationConfigDeliveryGroup:
		return notificationConfigGroupInfo{
			key:            notificationConfigDeliveryGroupKey,
			label:          "Notification delivery",
			DescriptionKey: notificationConfigDeliveryDescKey,
			Description:    "Control in-app delivery and fan-out limits.",
		}
	case notificationConfigDisplayGroup:
		return notificationConfigGroupInfo{
			key:            notificationConfigDisplayGroupKey,
			label:          "Notification display",
			DescriptionKey: notificationConfigDisplayDescKey,
			Description:    "Control Notification Center display defaults.",
		}
	default:
		return notificationConfigGroupInfo{
			key:            notificationConfigGeneralGroupKey,
			label:          "Notification general",
			DescriptionKey: notificationConfigGeneralDescKey,
			Description:    "Control the Notification Center baseline behavior.",
		}
	}
}

func notificationConfigTitleKey(key string) string {
	return "systemConfig.notification." + key + ".title"
}

func notificationConfigDescriptionKey(key string) string {
	return "systemConfig.notification." + key + ".description"
}

func mustRawJSON(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func registerNotificationConfigMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is required")
	}
	for _, registration := range notificationConfigMessageRegistrations() {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register notification config messages: %w", err)
		}
	}
	return nil
}

func notificationConfigMessageRegistrations() []i18n.Registration {
	return []i18n.Registration{
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleZHCN,
			Messages: notificationConfigMessages(map[string]string{
				notificationConfigDomainKey:        "站内通知",
				notificationConfigGeneralGroupKey:  "通用",
				notificationConfigGeneralDescKey:   "控制通知中心的基础行为。",
				notificationConfigSourcesGroupKey:  "通知来源",
				notificationConfigSourcesDescKey:   "控制哪些平台事件会产生通知。",
				notificationConfigDeliveryGroupKey: "投递",
				notificationConfigDeliveryDescKey:  "控制站内投递与批量 fan-out 限制。",
				notificationConfigDisplayGroupKey:  "展示",
				notificationConfigDisplayDescKey:   "控制通知中心的展示默认值。",
			}, zhCNNotificationConfigCopy()),
		},
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleENUS,
			Messages: notificationConfigMessages(map[string]string{
				notificationConfigDomainKey:        "Notification",
				notificationConfigGeneralGroupKey:  "General",
				notificationConfigGeneralDescKey:   "Control the Notification Center baseline behavior.",
				notificationConfigSourcesGroupKey:  "Sources",
				notificationConfigSourcesDescKey:   "Control which platform events create notifications.",
				notificationConfigDeliveryGroupKey: "Delivery",
				notificationConfigDeliveryDescKey:  "Control in-app delivery and fan-out limits.",
				notificationConfigDisplayGroupKey:  "Display",
				notificationConfigDisplayDescKey:   "Control Notification Center display defaults.",
			}, enUSNotificationConfigCopy()),
		},
	}
}

func notificationConfigMessages(prefix map[string]string, definitions map[string][2]string) []i18n.MessageResource {
	messages := make([]i18n.MessageResource, 0, len(prefix)+len(definitions)*2)
	for key, text := range prefix {
		messages = append(messages, i18n.MessageResource{Key: i18n.MessageKey(key), Text: text})
	}
	for key, copy := range definitions {
		messages = append(messages,
			i18n.MessageResource{Key: i18n.MessageKey(notificationConfigTitleKey(key)), Text: copy[0]},
			i18n.MessageResource{Key: i18n.MessageKey(notificationConfigDescriptionKey(key)), Text: copy[1]},
		)
	}
	return messages
}

func zhCNNotificationConfigCopy() map[string][2]string {
	return map[string][2]string{
		notificationEnabledKey:                                {"启用通知", "是否启用站内通知功能。"},
		notificationRetentionDaysKey:                          {"通知保留天数", "通知记录的默认保留天数。"},
		notificationDefaultPageSizeKey:                        {"默认分页大小", "通知中心列表默认分页大小。"},
		notificationSourceSystemAnnouncementEnabledKey:        {"系统公告通知", "发布系统公告时是否产生通知。"},
		notificationSourceScheduledTaskFailureEnabledKey:      {"定时任务失败通知", "定时任务执行失败时是否产生通知。"},
		notificationSourceScheduledTaskSuccessEnabledKey:      {"定时任务成功通知", "手动执行定时任务完成时是否产生通知。"},
		notificationSourceAuditIncidentEnabledKey:             {"安全审计事件通知", "审计事件升级为需要关注的安全事件时是否产生通知。"},
		notificationSourceSystemConfigChangeEnabledKey:        {"系统配置变更通知", "系统配置被修改、重置或校验失败时是否产生通知。"},
		notificationSourceAccessLogRetentionFailureEnabledKey: {"访问日志清理失败通知", "访问日志保留清理任务失败时是否产生通知。"},
		notificationDeliveryInAppEnabledKey:                   {"站内投递", "是否创建站内通知投递记录。"},
		notificationDeliveryDedupeWindowSecondsKey:            {"去重窗口秒数", "相同来源、目标和业务对象通知的去重窗口。"},
		notificationDeliveryMaxBatchRecipientsKey:             {"批量投递人数上限", "单批通知投递的最大接收人数。"},
		notificationDisplayShowReadDaysKey:                    {"已读展示天数", "已读通知的默认展示时间范围。"},
		notificationDisplayPopupLimitKey:                      {"铃铛弹层数量", "通知铃铛弹层最多展示条数。"},
	}
}

func enUSNotificationConfigCopy() map[string][2]string {
	return map[string][2]string{
		notificationEnabledKey:                                {"Notification Enabled", "Whether in-app notifications are enabled."},
		notificationRetentionDaysKey:                          {"Notification Retention Days", "Number of days notification records should be retained."},
		notificationDefaultPageSizeKey:                        {"Default Page Size", "Default page size used by Notification Center."},
		notificationSourceSystemAnnouncementEnabledKey:        {"System Announcement Notifications", "Enables notifications created from published system announcements."},
		notificationSourceScheduledTaskFailureEnabledKey:      {"Scheduled Task Failure Notifications", "Enables notifications for failed scheduled task runs."},
		notificationSourceScheduledTaskSuccessEnabledKey:      {"Scheduled Task Success Notifications", "Enables notifications for successful manual scheduled task runs."},
		notificationSourceAuditIncidentEnabledKey:             {"Audit Incident Notifications", "Enables notifications for audit events that require security review."},
		notificationSourceSystemConfigChangeEnabledKey:        {"System Config Change Notifications", "Enables notifications for system configuration changes, resets, and validation failures."},
		notificationSourceAccessLogRetentionFailureEnabledKey: {"Access Log Retention Failure Notifications", "Enables notifications for access-log retention cleanup failures."},
		notificationDeliveryInAppEnabledKey:                   {"In-App Delivery Enabled", "Whether Notification Center should create in-app delivery records."},
		notificationDeliveryDedupeWindowSecondsKey:            {"Dedupe Window Seconds", "Dedupe window for same source, target, and business object notifications."},
		notificationDeliveryMaxBatchRecipientsKey:             {"Max Batch Recipients", "Maximum recipients in one notification delivery batch."},
		notificationDisplayShowReadDaysKey:                    {"Show Read Days", "Default time window for showing read notifications."},
		notificationDisplayPopupLimitKey:                      {"Popup Limit", "Maximum notifications displayed by the notification bell popup."},
	}
}

func notificationSourceEnabledKey(sourceModule string, eventType string) string {
	switch strings.TrimSpace(sourceModule) {
	case "scheduler":
		if strings.TrimSpace(eventType) == "task_succeeded" {
			return notificationSourceScheduledTaskSuccessEnabledKey
		}
		return notificationSourceScheduledTaskFailureEnabledKey
	case "audit":
		return notificationSourceAuditIncidentEnabledKey
	case "system-config":
		return notificationSourceSystemConfigChangeEnabledKey
	case "access-log":
		return notificationSourceAccessLogRetentionFailureEnabledKey
	case moduleID:
		return notificationSourceSystemAnnouncementEnabledKey
	default:
		return ""
	}
}
