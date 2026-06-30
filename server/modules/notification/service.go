package notification

import (
	"context"
	"errors"
	"math"
	"time"

	"graft/server/internal/moduleapi"
	notificationstore "graft/server/modules/notification/store"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

var errNotificationServiceUnavailable = errors.New("notification service is unavailable")

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
	page, size := normalizePage(query.Page, query.PageSize)
	result, err := withNotificationRepository(s, func(repository notificationstore.Repository) (notificationstore.ListResult, error) {
		return repository.List(ctx, query.toStoreListQuery(page, size))
	})
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: result.Items, Total: result.Total, Page: page, Size: size}, nil
}

// Get returns one current-user notification by delivery id.
func (s *Service) Get(ctx context.Context, recipientUserID uint64, deliveryID uint64) (notificationstore.Notification, error) {
	return withNotificationRepository(s, func(repository notificationstore.Repository) (notificationstore.Notification, error) {
		return repository.Get(ctx, recipientUserID, deliveryID)
	})
}

// UnreadCount returns the current user's unread notification count.
func (s *Service) UnreadCount(ctx context.Context, recipientUserID uint64) (int, error) {
	return withNotificationRepository(s, func(repository notificationstore.Repository) (int, error) {
		return repository.UnreadCount(ctx, recipientUserID)
	})
}

// MarkRead marks one current-user delivery as read.
func (s *Service) MarkRead(ctx context.Context, recipientUserID uint64, deliveryID uint64, readAt time.Time) (notificationstore.Delivery, error) {
	return withNotificationRepository(s, func(repository notificationstore.Repository) (notificationstore.Delivery, error) {
		return repository.MarkRead(ctx, recipientUserID, deliveryID, defaultUTCTimestamp(readAt))
	})
}

// MarkAllRead marks all current-user unread deliveries as read.
func (s *Service) MarkAllRead(ctx context.Context, recipientUserID uint64, readAt time.Time) (int, error) {
	return withNotificationRepository(s, func(repository notificationstore.Repository) (int, error) {
		return repository.MarkAllRead(ctx, recipientUserID, defaultUTCTimestamp(readAt))
	})
}

// MarkAllReadMatching marks all current-user unread deliveries matching the optional filters as read.
func (s *Service) MarkAllReadMatching(ctx context.Context, query ListQuery, readAt time.Time) (int, error) {
	return withNotificationRepository(s, func(repository notificationstore.Repository) (int, error) {
		return repository.MarkAllReadMatching(ctx, query.toStoreFilter("unread"), defaultUTCTimestamp(readAt))
	})
}

// DeleteDelivery soft-deletes one current-user delivery.
func (s *Service) DeleteDelivery(ctx context.Context, recipientUserID uint64, deliveryID uint64, deletedAt time.Time) error {
	return runNotificationRepository(s, func(repository notificationstore.Repository) error {
		return repository.DeleteDelivery(ctx, recipientUserID, deliveryID, defaultUTCTimestamp(deletedAt))
	})
}

func (s *Service) repositoryOrErr() (notificationstore.Repository, error) {
	if s == nil || s.repository == nil {
		return nil, errNotificationServiceUnavailable
	}
	return s.repository, nil
}

func (q ListQuery) toStoreListQuery(page int, size int) notificationstore.ListQuery {
	query := q.toStoreFilter("")
	query.Limit = size
	query.Offset = (page - 1) * size
	return query
}

func (q ListQuery) toStoreFilter(status string) notificationstore.ListQuery {
	if status == "" {
		status = q.Status
	}

	return notificationstore.ListQuery{
		RecipientUserID: q.RecipientUserID,
		Status:          status,
		Severity:        q.Severity,
		Category:        q.Category,
		SourceModule:    q.SourceModule,
		OccurredFrom:    q.OccurredFrom,
		OccurredTo:      q.OccurredTo,
	}
}

// defaultUTCTimestamp 返回零值时刻对应的当前 UTC 时间，非零值则原样返回。
func defaultUTCTimestamp(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}

// normalizePage 规范分页参数。
// 它确保页码至少为 1，页大小在默认值与最大值范围内。
func normalizePage(page int, size int) (int, int) {
	if size <= 0 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	if page <= 0 {
		page = 1
	}
	maxPage := math.MaxInt / size
	if maxPage < math.MaxInt {
		maxPage++
	}
	if page > maxPage {
		page = maxPage
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
