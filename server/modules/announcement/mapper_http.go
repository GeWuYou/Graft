// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package announcement

import (
	"math"

	announcementopenapi "graft/server/internal/contract/openapi/announcement"
	announcementstore "graft/server/modules/announcement/store"
)

func toAnnouncementListResponse(result AdminListResult) map[string]any {
	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toAnnouncementItem(item))
	}
	return map[string]any{
		"items":     items,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	}
}

func toAnnouncementItem(item announcementstore.Announcement) map[string]any {
	return map[string]any{
		"id":            safeInt64(item.ID),
		"title":         item.Title,
		"content":       item.Content,
		"level":         item.Level,
		"status":        item.Status,
		"delivery_mode": item.DeliveryMode,
		"pinned":        item.Pinned,
		"publish_at":    item.PublishAt,
		"expire_at":     item.ExpireAt,
		"created_by":    safeOptionalInt64(item.CreatedBy),
		"updated_by":    safeOptionalInt64(item.UpdatedBy),
		"deleted_by":    safeOptionalInt64(item.DeletedBy),
		"created_at":    item.CreatedAt,
		"updated_at":    item.UpdatedAt,
	}
}

func toMyAnnouncementListResponse(result UserListResult) map[string]any {
	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toMyAnnouncementItem(item))
	}
	return map[string]any{
		"items":     items,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	}
}

func toMyAnnouncementItem(item announcementstore.UserAnnouncement) map[string]any {
	announcement := item.Announcement
	return map[string]any{
		"id":            safeInt64(announcement.ID),
		"title":         announcement.Title,
		"content":       announcement.Content,
		"level":         announcementopenapi.GetMyAnnouncements200JSONResponseBodyDataItemsLevel(announcement.Level),
		"status":        announcementopenapi.GetMyAnnouncements200JSONResponseBodyDataItemsStatus(announcement.Status),
		"delivery_mode": announcement.DeliveryMode,
		"pinned":        announcement.Pinned,
		"publish_at":    announcement.PublishAt,
		"expire_at":     announcement.ExpireAt,
		"read_at":       item.ReadAt,
		"created_at":    announcement.CreatedAt,
		"updated_at":    announcement.UpdatedAt,
	}
}

func safeInt64(value uint64) int64 {
	if value > uint64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(value)
}

func safeOptionalInt64(value *uint64) *int64 {
	if value == nil {
		return nil
	}
	converted := safeInt64(*value)
	return &converted
}
