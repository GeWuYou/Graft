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
	notificationDisplayKey                                = "notification.display"
	notificationDisplayShowReadDaysKey                    = "display.showReadDays"
	notificationDisplayPopupLimitKey                      = "display.popupLimit"
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
		notificationDisplayDefinition(),
	}
}

func booleanNotificationDefinition(key string, group string, title string, description string, defaultValue bool) configregistry.Definition {
	return baseNotificationDefinition(key, group, title, description, configregistry.ValueTypeBoolean, mustRawJSON(defaultValue))
}

func numberNotificationDefinition(key string, group string, title string, description string, defaultValue int) configregistry.Definition {
	return baseNotificationDefinition(key, group, title, description, configregistry.ValueTypeInteger, mustRawJSON(defaultValue))
}

func notificationDisplayDefinition() configregistry.Definition {
	definition := baseNotificationDefinition(
		notificationDisplayKey,
		notificationConfigDisplayGroup,
		"Notification display",
		"Notification Center display defaults.",
		configregistry.ValueTypeObject,
		json.RawMessage(fmt.Sprintf(`{"showReadDays":%d,"popupLimit":%d}`, defaultNotificationDisplayShowReadDays, defaultNotificationDisplayPopupLimit)),
	)
	definition.Schema = notificationDisplaySchema()
	return definition
}

func notificationDisplaySchema() json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"type":"object","title":"Notification display","description":"Notification Center display defaults.","properties":{"showReadDays":{"type":"integer","minimum":1,"maximum":365,"default":7,"title":"Show read days","description":"Default time window for showing read notifications.","x-i18n":{"titleKey":%q,"descriptionKey":%q,"unitKey":"systemConfig.units.days"}},"popupLimit":{"type":"integer","minimum":1,"maximum":50,"default":5,"title":"Popup limit","description":"Maximum notifications displayed by the notification bell popup.","x-i18n":{"titleKey":%q,"descriptionKey":%q}}},"required":["showReadDays","popupLimit"],"additionalProperties":false,"x-i18n":{"titleKey":%q,"descriptionKey":%q}}`,
		notificationConfigTitleKey(notificationDisplayShowReadDaysKey),
		notificationConfigDescriptionKey(notificationDisplayShowReadDaysKey),
		notificationConfigTitleKey(notificationDisplayPopupLimitKey),
		notificationConfigDescriptionKey(notificationDisplayPopupLimitKey),
		notificationConfigTitleKey(notificationDisplayKey),
		notificationConfigDescriptionKey(notificationDisplayKey),
	))
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
		GroupDescription:    metadata.description,
		GroupDescriptionKey: metadata.descriptionKey,
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
	descriptionKey string
	description    string
}

func notificationConfigGroupMetadata(group string) notificationConfigGroupInfo {
	switch group {
	case notificationConfigSourcesGroup:
		return notificationConfigGroupInfo{
			key:            notificationConfigSourcesGroupKey,
			label:          "Notification sources",
			descriptionKey: notificationConfigSourcesDescKey,
			description:    "Control which platform events create notifications.",
		}
	case notificationConfigDeliveryGroup:
		return notificationConfigGroupInfo{
			key:            notificationConfigDeliveryGroupKey,
			label:          "Notification delivery",
			descriptionKey: notificationConfigDeliveryDescKey,
			description:    "Control in-app delivery and fan-out limits.",
		}
	case notificationConfigDisplayGroup:
		return notificationConfigGroupInfo{
			key:            notificationConfigDisplayGroupKey,
			label:          "Notification display",
			descriptionKey: notificationConfigDisplayDescKey,
			description:    "Control Notification Center display defaults.",
		}
	default:
		return notificationConfigGroupInfo{
			key:            notificationConfigGeneralGroupKey,
			label:          "Notification general",
			descriptionKey: notificationConfigGeneralDescKey,
			description:    "Control the Notification Center baseline behavior.",
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
	return nil
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
