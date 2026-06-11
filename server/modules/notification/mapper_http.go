// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"encoding/json"

	generated "graft/server/internal/contract/openapi/generated"
	notificationopenapi "graft/server/internal/contract/openapi/notification"
	notificationstore "graft/server/modules/notification/store"
)

func toNotificationListResponse(result ListResult) generated.NotificationListResponse {
	items := make([]generated.NotificationItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toNotificationItem(item))
	}
	return generated.NotificationListResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.Size,
		Total:    result.Total,
	}
}

func toNotificationUnreadCountResponse(count int) generated.NotificationUnreadCountResponse {
	return generated.NotificationUnreadCountResponse{Count: count}
}

func toNotificationItem(item notificationstore.Notification) generated.NotificationItem {
	status := generated.Unread
	if item.Delivery.ReadAt != nil {
		status = generated.Read
	}

	return generated.NotificationItem{
		Category:          generated.NotificationCategory(item.Event.Category),
		DeliveryCreatedAt: item.Delivery.CreatedAt,
		DeliveryId:        responseID(item.Delivery.ID),
		EventCreatedAt:    &item.Event.CreatedAt,
		EventId:           responseID(item.Event.ID),
		EventType:         item.Event.EventType,
		ExpiresAt:         item.Event.ExpiresAt,
		Message:           item.Event.Message,
		MessageKey:        optionalString(item.Event.MessageKey),
		CategoryKey:       optionalString(item.Event.CategoryKey),
		SourceKey:         optionalString(item.Event.SourceKey),
		LevelKey:          optionalString(item.Event.LevelKey),
		EventTypeKey:      optionalString(item.Event.EventTypeKey),
		ResourceTypeKey:   optionalString(item.Event.ResourceTypeKey),
		ActionLabelKey:    optionalString(item.Event.ActionLabelKey),
		ActionLabel:       optionalString(item.Event.ActionLabel),
		Context:           optionalRawJSONMap(item.Event.Metadata),
		Navigation: generated.NotificationNavigation{
			Kind:    generated.NotificationNavigationKind(item.Event.NavigationKind),
			Payload: rawJSONMap(item.Event.NavigationPayload),
		},
		OccurredAt:   item.Event.OccurredAt,
		ReadAt:       item.Delivery.ReadAt,
		ResourceId:   optionalString(item.Event.ResourceID),
		ResourceName: optionalString(item.Event.ResourceName),
		ResourceType: optionalString(item.Event.ResourceType),
		Severity:     generated.NotificationSeverity(item.Event.Severity),
		SourceModule: item.Event.SourceModule,
		Status:       status,
		TargetRef:    item.Delivery.TargetRef,
		TargetType:   generated.NotificationTargetType(item.Delivery.TargetType),
		Title:        item.Event.Title,
		TitleKey:     optionalString(item.Event.TitleKey),
	}
}

func responseID(id uint64) int64 {
	if id > uint64(^uint64(0)>>1) {
		return 0
	}
	return int64(id)
}

func readAllQueryFromBody(body notificationopenapi.PostNotificationsReadAllJSONRequestBody) ListQuery {
	return ListQuery{
		Severity:     stringFromPointer(body.Severity),
		Category:     stringFromPointer(body.Category),
		SourceModule: stringFromPointer(body.SourceModule),
		OccurredFrom: body.OccurredFrom,
		OccurredTo:   body.OccurredTo,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringFromPointer[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func rawJSONMap(raw json.RawMessage) map[string]interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil || payload == nil {
		return map[string]interface{}{}
	}
	return payload
}

func optionalRawJSONMap(raw json.RawMessage) *map[string]interface{} {
	payload := rawJSONMap(raw)
	if len(payload) == 0 {
		return nil
	}
	return &payload
}
