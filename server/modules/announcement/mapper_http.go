// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package announcement

import (
	"errors"
	"math"
	"time"

	announcementopenapi "graft/server/internal/contract/openapi/announcement"
	announcementstore "graft/server/modules/announcement/store"
)

type announcementListResponse struct {
	Items    []announcementItemResponse `json:"items"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"page_size"`
}

type announcementItemResponse struct {
	ID           int64                                                                        `json:"id"`
	Title        string                                                                       `json:"title"`
	Content      string                                                                       `json:"content"`
	Level        announcementopenapi.GetAnnouncements200JSONResponseBodyDataItemsLevel        `json:"level"`
	Status       announcementopenapi.GetAnnouncements200JSONResponseBodyDataItemsStatus       `json:"status"`
	DeliveryMode announcementopenapi.GetAnnouncements200JSONResponseBodyDataItemsDeliveryMode `json:"delivery_mode"`
	Pinned       bool                                                                         `json:"pinned"`
	PublishAt    *time.Time                                                                   `json:"publish_at"`
	PublishedBy  *int64                                                                       `json:"published_by"`
	ArchivedAt   *time.Time                                                                   `json:"archived_at"`
	ExpireAt     *time.Time                                                                   `json:"expire_at"`
	CreatedBy    *int64                                                                       `json:"created_by"`
	UpdatedBy    *int64                                                                       `json:"updated_by"`
	DeletedBy    *int64                                                                       `json:"deleted_by"`
	CreatedAt    time.Time                                                                    `json:"created_at"`
	UpdatedAt    time.Time                                                                    `json:"updated_at"`
}

type myAnnouncementListResponse struct {
	Items    []myAnnouncementItemResponse `json:"items"`
	Total    int                          `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}

type myAnnouncementItemResponse struct {
	ID           int64                                                                          `json:"id"`
	Title        string                                                                         `json:"title"`
	Content      string                                                                         `json:"content"`
	Level        announcementopenapi.GetMyAnnouncements200JSONResponseBodyDataItemsLevel        `json:"level"`
	Status       announcementopenapi.GetMyAnnouncements200JSONResponseBodyDataItemsStatus       `json:"status"`
	DeliveryMode announcementopenapi.GetMyAnnouncements200JSONResponseBodyDataItemsDeliveryMode `json:"delivery_mode"`
	Pinned       bool                                                                           `json:"pinned"`
	PublishAt    *time.Time                                                                     `json:"publish_at"`
	PublishedBy  *int64                                                                         `json:"published_by"`
	ArchivedAt   *time.Time                                                                     `json:"archived_at"`
	ExpireAt     *time.Time                                                                     `json:"expire_at"`
	ReadAt       *time.Time                                                                     `json:"read_at"`
	Unread       bool                                                                           `json:"unread"`
	CreatedAt    time.Time                                                                      `json:"created_at"`
	UpdatedAt    time.Time                                                                      `json:"updated_at"`
}

type announcementReadAllResponse struct {
	UpdatedCount int `json:"updated_count"`
}

type announcementUnreadCountResponse struct {
	Count int `json:"count"`
}

type emptyResponse struct{}

var errAnnouncementIDOverflow = errors.New("announcement id exceeds OpenAPI int64 range")

func toAnnouncementListResponse(result AdminListResult) (announcementListResponse, error) {
	items := make([]announcementItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		response, err := toAnnouncementItem(item)
		if err != nil {
			return announcementListResponse{}, err
		}
		items = append(items, response)
	}
	return announcementListResponse{Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize}, nil
}

func toAnnouncementItem(item announcementstore.Announcement) (announcementItemResponse, error) {
	id, err := safeInt64(item.ID)
	if err != nil {
		return announcementItemResponse{}, err
	}
	createdBy, err := safeOptionalInt64(item.CreatedBy)
	if err != nil {
		return announcementItemResponse{}, err
	}
	updatedBy, err := safeOptionalInt64(item.UpdatedBy)
	if err != nil {
		return announcementItemResponse{}, err
	}
	publishedBy, err := safeOptionalInt64(item.PublishedBy)
	if err != nil {
		return announcementItemResponse{}, err
	}
	deletedBy, err := safeOptionalInt64(item.DeletedBy)
	if err != nil {
		return announcementItemResponse{}, err
	}
	return announcementItemResponse{
		ID:           id,
		Title:        item.Title,
		Content:      item.Content,
		Level:        announcementopenapi.GetAnnouncements200JSONResponseBodyDataItemsLevel(item.Level),
		Status:       announcementopenapi.GetAnnouncements200JSONResponseBodyDataItemsStatus(item.Status),
		DeliveryMode: announcementopenapi.GetAnnouncements200JSONResponseBodyDataItemsDeliveryMode(item.DeliveryMode),
		Pinned:       item.Pinned,
		PublishAt:    item.PublishAt,
		PublishedBy:  publishedBy,
		ArchivedAt:   item.ArchivedAt,
		ExpireAt:     item.ExpireAt,
		CreatedBy:    createdBy,
		UpdatedBy:    updatedBy,
		DeletedBy:    deletedBy,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}, nil
}

func toMyAnnouncementListResponse(result UserListResult) (myAnnouncementListResponse, error) {
	items := make([]myAnnouncementItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		response, err := toMyAnnouncementItem(item)
		if err != nil {
			return myAnnouncementListResponse{}, err
		}
		items = append(items, response)
	}
	return myAnnouncementListResponse{Items: items, Total: result.Total, Page: result.Page, PageSize: result.PageSize}, nil
}

func toMyAnnouncementItem(item announcementstore.UserAnnouncement) (myAnnouncementItemResponse, error) {
	announcement := item.Announcement
	id, err := safeInt64(announcement.ID)
	if err != nil {
		return myAnnouncementItemResponse{}, err
	}
	publishedBy, err := safeOptionalInt64(announcement.PublishedBy)
	if err != nil {
		return myAnnouncementItemResponse{}, err
	}
	return myAnnouncementItemResponse{
		ID:           id,
		Title:        announcement.Title,
		Content:      announcement.Content,
		Level:        announcementopenapi.GetMyAnnouncements200JSONResponseBodyDataItemsLevel(announcement.Level),
		Status:       announcementopenapi.GetMyAnnouncements200JSONResponseBodyDataItemsStatus(announcement.Status),
		DeliveryMode: announcementopenapi.GetMyAnnouncements200JSONResponseBodyDataItemsDeliveryMode(announcement.DeliveryMode),
		Pinned:       announcement.Pinned,
		PublishAt:    announcement.PublishAt,
		PublishedBy:  publishedBy,
		ArchivedAt:   announcement.ArchivedAt,
		ExpireAt:     announcement.ExpireAt,
		ReadAt:       item.ReadAt,
		Unread:       item.ReadAt == nil,
		CreatedAt:    announcement.CreatedAt,
		UpdatedAt:    announcement.UpdatedAt,
	}, nil
}

func safeInt64(value uint64) (int64, error) {
	if value > uint64(math.MaxInt64) {
		return 0, errAnnouncementIDOverflow
	}
	return int64(value), nil
}

func safeOptionalInt64(value *uint64) (*int64, error) {
	if value == nil {
		return nil, nil
	}
	converted, err := safeInt64(*value)
	if err != nil {
		return nil, err
	}
	return &converted, nil
}
