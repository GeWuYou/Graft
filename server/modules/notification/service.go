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

// ListQuery 描述当前用户通知筛选条件；结果范围由当前用户投递记录约束。
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

// ListResult 保存当前用户通知分页结果及规范化后的分页参数。
type ListResult struct {
	Items []notificationstore.Notification
	Total int
	Page  int
	Size  int
}

// Service 拥有当前用户通知读取和投递状态变更用例，不暴露跨用户查询能力。
type Service struct {
	repository notificationstore.Repository
}

// NewService 创建通知中心服务边界；Repository 为空时返回错误。
func NewService(repository notificationstore.Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("notification repository is unavailable")
	}
	return &Service{repository: repository}, nil
}

// List 返回一页当前用户通知，并将筛选范围限制在该用户的投递记录内。
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

// Get 按投递 ID 返回当前用户可见的一条通知。
func (s *Service) Get(ctx context.Context, recipientUserID uint64, deliveryID uint64) (notificationstore.Notification, error) {
	return withNotificationRepository(s, func(repository notificationstore.Repository) (notificationstore.Notification, error) {
		return repository.Get(ctx, recipientUserID, deliveryID)
	})
}

// UnreadCount 返回当前用户未读通知数量。
func (s *Service) UnreadCount(ctx context.Context, recipientUserID uint64) (int, error) {
	return withNotificationRepository(s, func(repository notificationstore.Repository) (int, error) {
		return repository.UnreadCount(ctx, recipientUserID)
	})
}

// MarkRead 将当前用户的一条投递记录标记为已读。
func (s *Service) MarkRead(ctx context.Context, recipientUserID uint64, deliveryID uint64, readAt time.Time) (notificationstore.Delivery, error) {
	return withNotificationRepository(s, func(repository notificationstore.Repository) (notificationstore.Delivery, error) {
		return repository.MarkRead(ctx, recipientUserID, deliveryID, defaultUTCTimestamp(readAt))
	})
}

// MarkAllRead 将当前用户所有符合条件的未读投递记录标记为已读。
func (s *Service) MarkAllRead(ctx context.Context, recipientUserID uint64, readAt time.Time) (int, error) {
	return withNotificationRepository(s, func(repository notificationstore.Repository) (int, error) {
		return repository.MarkAllRead(ctx, recipientUserID, defaultUTCTimestamp(readAt))
	})
}

// MarkAllReadMatching 将当前用户符合可选筛选条件的未读投递记录全部标记为已读。
func (s *Service) MarkAllReadMatching(ctx context.Context, query ListQuery, readAt time.Time) (int, error) {
	return withNotificationRepository(s, func(repository notificationstore.Repository) (int, error) {
		return repository.MarkAllReadMatching(ctx, query.toStoreFilter("unread"), defaultUTCTimestamp(readAt))
	})
}

// DeleteDelivery 软删除当前用户的一条投递记录，不删除不可变通知事实。
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
