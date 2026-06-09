// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"context"
	"errors"
	"time"

	"graft/server/internal/moduleapi"
	notificationstore "graft/server/modules/notification/store"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// ListQuery describes current-user notification filters.
type ListQuery struct {
	RecipientUserID uint64
	Status          string
	Severity        string
	Category        string
	SourceModule    string
	OccurredFrom    *time.Time
	OccurredTo      *time.Time
	Page            int
	PageSize        int
}

// ListResult returns a current-user notification page.
type ListResult struct {
	Items []notificationstore.Notification
	Total int
	Page  int
	Size  int
}

// Service owns current-user notification read and delivery-state mutations.
type Service struct {
	repository notificationstore.Repository
}

// NewService creates the Notification Center service boundary.
func NewService(repository notificationstore.Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("notification repository is unavailable")
	}
	return &Service{repository: repository}, nil
}

// List returns one page of current-user notifications.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	if s == nil || s.repository == nil {
		return ListResult{}, errors.New("notification service is unavailable")
	}
	page, size := normalizePage(query.Page, query.PageSize)
	result, err := s.repository.List(ctx, notificationstore.ListQuery{
		RecipientUserID: query.RecipientUserID,
		Status:          query.Status,
		Severity:        query.Severity,
		Category:        query.Category,
		SourceModule:    query.SourceModule,
		OccurredFrom:    query.OccurredFrom,
		OccurredTo:      query.OccurredTo,
		Limit:           size,
		Offset:          (page - 1) * size,
	})
	if err != nil {
		return ListResult{}, mapStoreError(err)
	}
	return ListResult{Items: result.Items, Total: result.Total, Page: page, Size: size}, nil
}

// Get returns one current-user notification by delivery id.
func (s *Service) Get(ctx context.Context, recipientUserID uint64, deliveryID uint64) (notificationstore.Notification, error) {
	if s == nil || s.repository == nil {
		return notificationstore.Notification{}, errors.New("notification service is unavailable")
	}
	item, err := s.repository.Get(ctx, recipientUserID, deliveryID)
	return item, mapStoreError(err)
}

// UnreadCount returns the current user's unread notification count.
func (s *Service) UnreadCount(ctx context.Context, recipientUserID uint64) (int, error) {
	if s == nil || s.repository == nil {
		return 0, errors.New("notification service is unavailable")
	}
	count, err := s.repository.UnreadCount(ctx, recipientUserID)
	return count, mapStoreError(err)
}

// MarkRead marks one current-user delivery as read.
func (s *Service) MarkRead(ctx context.Context, recipientUserID uint64, deliveryID uint64, readAt time.Time) (notificationstore.Delivery, error) {
	if s == nil || s.repository == nil {
		return notificationstore.Delivery{}, errors.New("notification service is unavailable")
	}
	if readAt.IsZero() {
		readAt = time.Now().UTC()
	}
	delivery, err := s.repository.MarkRead(ctx, recipientUserID, deliveryID, readAt)
	return delivery, mapStoreError(err)
}

// MarkAllRead marks all current-user unread deliveries as read.
func (s *Service) MarkAllRead(ctx context.Context, recipientUserID uint64, readAt time.Time) (int, error) {
	if s == nil || s.repository == nil {
		return 0, errors.New("notification service is unavailable")
	}
	if readAt.IsZero() {
		readAt = time.Now().UTC()
	}
	count, err := s.repository.MarkAllRead(ctx, recipientUserID, readAt)
	return count, mapStoreError(err)
}

// MarkAllReadMatching marks all current-user unread deliveries matching the optional filters as read.
func (s *Service) MarkAllReadMatching(ctx context.Context, query ListQuery, readAt time.Time) (int, error) {
	if s == nil || s.repository == nil {
		return 0, errors.New("notification service is unavailable")
	}
	if readAt.IsZero() {
		readAt = time.Now().UTC()
	}
	count, err := s.repository.MarkAllReadMatching(ctx, notificationstore.ListQuery{
		RecipientUserID: query.RecipientUserID,
		Status:          "unread",
		Severity:        query.Severity,
		Category:        query.Category,
		SourceModule:    query.SourceModule,
		OccurredFrom:    query.OccurredFrom,
		OccurredTo:      query.OccurredTo,
	}, readAt)
	return count, mapStoreError(err)
}

// DeleteDelivery soft-deletes one current-user delivery.
func (s *Service) DeleteDelivery(ctx context.Context, recipientUserID uint64, deliveryID uint64, deletedAt time.Time) error {
	if s == nil || s.repository == nil {
		return errors.New("notification service is unavailable")
	}
	if deletedAt.IsZero() {
		deletedAt = time.Now().UTC()
	}
	return mapStoreError(s.repository.DeleteDelivery(ctx, recipientUserID, deliveryID, deletedAt))
}

func normalizePage(page int, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return page, size
}

func mapStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, notificationstore.ErrInvalidInput):
		return moduleapi.ErrNotificationInvalidInput
	case errors.Is(err, notificationstore.ErrDeliveryNotFound):
		return moduleapi.ErrNotificationDeliveryNotFound
	default:
		return err
	}
}
