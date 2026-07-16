package announcement

import (
	"context"
	"errors"
	"strings"
	"time"

	announcementcontract "graft/server/modules/announcement/contract"
	announcementstore "graft/server/modules/announcement/store"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

var (
	errAnnouncementInvalidInput      = errors.New("announcement invalid input")
	errAnnouncementNotFound          = errors.New("announcement not found")
	errAnnouncementInvalidTransition = errors.New("announcement invalid status transition")
	errAnnouncementPublishedDelete   = errors.New("published announcement must be archived before delete")
)

// AdminListQuery 描述管理端公告筛选条件；分页和排序值由 Service 统一规范化并校验。
type AdminListQuery struct {
	Status   string
	Level    string
	Pinned   *bool
	Keyword  string
	Page     int
	PageSize int
	Sort     string
}

// AdminListResult 保存管理端公告分页结果及规范化后的分页参数。
type AdminListResult struct {
	Items    []announcementstore.Announcement
	Total    int
	Page     int
	PageSize int
}

// UserListQuery 描述当前用户公告筛选条件；可见性由 store 按当前时间和用户读取状态判断。
type UserListQuery struct {
	UserID     uint64
	UnreadOnly bool
	Page       int
	PageSize   int
}

// UserListResult 保存当前用户公告分页结果及规范化后的分页参数。
type UserListResult struct {
	Items    []announcementstore.UserAnnouncement
	Total    int
	Page     int
	PageSize int
}

// Service 拥有公告用例，并通过 Repository 保持状态转换与当前用户可见性边界。
type Service struct {
	repository announcementstore.Repository
}

// NewService 创建公告服务边界；Repository 为空时返回错误，避免模块在无持久化能力下启动。
func NewService(repository announcementstore.Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("announcement repository is unavailable")
	}
	return &Service{repository: repository}, nil
}

// ListAdmin 返回一页管理端公告，并拒绝未知状态、等级或排序值。
func (s *Service) ListAdmin(ctx context.Context, query AdminListQuery) (AdminListResult, error) {
	if err := s.ensureReady(); err != nil {
		return AdminListResult{}, err
	}
	query = normalizeAdminListQuery(query)
	if query.Status != "" && !announcementcontract.ValidAnnouncementStatus(announcementcontract.AnnouncementStatus(query.Status)) {
		return AdminListResult{}, errAnnouncementInvalidInput
	}
	if query.Level != "" && !announcementcontract.ValidAnnouncementLevel(announcementcontract.AnnouncementLevel(query.Level)) {
		return AdminListResult{}, errAnnouncementInvalidInput
	}
	if !validAdminSort(query.Sort) {
		return AdminListResult{}, errAnnouncementInvalidInput
	}
	page, size := normalizePage(query.Page, query.PageSize)
	result, err := s.repository.ListAdmin(ctx, announcementstore.ListQuery{
		Status:  query.Status,
		Level:   query.Level,
		Pinned:  query.Pinned,
		Keyword: query.Keyword,
		Sort:    query.Sort,
		Limit:   size,
		Offset:  (page - 1) * size,
	})
	if err != nil {
		return AdminListResult{}, mapStoreError(err)
	}
	return AdminListResult{Items: result.Items, Total: result.Total, Page: page, PageSize: size}, nil
}

// Create 创建草稿公告；OpenAPI 创建请求没有 status 字段，因此服务端始终强制初始状态为草稿。
func (s *Service) Create(ctx context.Context, input announcementstore.CreateInput) (announcementstore.Announcement, error) {
	if err := s.ensureReady(); err != nil {
		return announcementstore.Announcement{}, err
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	input.Level = strings.TrimSpace(input.Level)
	input.DeliveryMode = normalizeDeliveryMode(input.DeliveryMode)
	input.Status = announcementcontract.AnnouncementStatusDraft.String()
	if err := validateWriteInput(input.Title, input.Content, input.Level, input.DeliveryMode, input.PublishAt, input.ExpireAt); err != nil {
		return announcementstore.Announcement{}, err
	}
	item, err := s.repository.Create(ctx, input)
	return item, mapStoreError(err)
}

// GetAdmin 返回一条未删除的管理端公告。
func (s *Service) GetAdmin(ctx context.Context, id uint64) (announcementstore.Announcement, error) {
	if err := s.ensureReady(); err != nil {
		return announcementstore.Announcement{}, err
	}
	item, err := s.repository.GetAdmin(ctx, id)
	return item, mapStoreError(err)
}

// Update 更新草稿或已发布公告的可编辑字段；已归档公告不能再次编辑。
func (s *Service) Update(ctx context.Context, id uint64, input announcementstore.UpdateInput) (announcementstore.Announcement, error) {
	if err := s.ensureReady(); err != nil {
		return announcementstore.Announcement{}, err
	}
	current, err := s.repository.GetAdmin(ctx, id)
	if err != nil {
		return announcementstore.Announcement{}, mapStoreError(err)
	}
	if current.Status == announcementcontract.AnnouncementStatusArchived.String() {
		return announcementstore.Announcement{}, errAnnouncementInvalidTransition
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	input.Level = strings.TrimSpace(input.Level)
	input.DeliveryMode = normalizeDeliveryMode(input.DeliveryMode)
	if err := validateWriteInput(input.Title, input.Content, input.Level, input.DeliveryMode, input.PublishAt, input.ExpireAt); err != nil {
		return announcementstore.Announcement{}, err
	}
	item, err := s.repository.Update(ctx, id, input)
	return item, mapStoreError(err)
}

// Publish 将公告标记为已发布，并校验过期时间必须晚于本次生效时间。
func (s *Service) Publish(ctx context.Context, id uint64, publishAt *time.Time, actorID *uint64) (announcementstore.Announcement, error) {
	if err := s.ensureReady(); err != nil {
		return announcementstore.Announcement{}, err
	}
	current, err := s.repository.GetAdmin(ctx, id)
	if err != nil {
		return announcementstore.Announcement{}, mapStoreError(err)
	}
	var effectivePublishAt *time.Time
	publishedAt := time.Now().UTC()
	publicationInstant := publishedAt
	if publishAt != nil {
		normalized := publishAt.UTC()
		effectivePublishAt = &normalized
		publicationInstant = normalized
	}
	if current.ExpireAt != nil && !current.ExpireAt.After(publicationInstant) {
		return announcementstore.Announcement{}, errAnnouncementInvalidInput
	}
	item, err := s.repository.Publish(ctx, id, effectivePublishAt, publishedAt, actorID)
	return item, mapStoreError(err)
}

// Archive 将公告归档，使其不再出现在当前用户可见列表中；重复归档保持幂等。
func (s *Service) Archive(ctx context.Context, id uint64, actorID *uint64) (announcementstore.Announcement, error) {
	if err := s.ensureReady(); err != nil {
		return announcementstore.Announcement{}, err
	}
	current, err := s.repository.GetAdmin(ctx, id)
	if err != nil {
		return announcementstore.Announcement{}, mapStoreError(err)
	}
	if current.Status == announcementcontract.AnnouncementStatusArchived.String() {
		return current, nil
	}
	item, err := s.repository.Archive(ctx, id, actorID)
	return item, mapStoreError(err)
}

// Delete 软删除草稿或已归档公告；已发布公告必须先归档，避免直接移除仍可见的内容。
func (s *Service) Delete(ctx context.Context, id uint64, actorID uint64) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	current, err := s.repository.GetAdmin(ctx, id)
	if err != nil {
		return mapStoreError(err)
	}
	if current.Status == announcementcontract.AnnouncementStatusPublished.String() {
		return errAnnouncementPublishedDelete
	}
	return mapStoreError(s.repository.Delete(ctx, id, actorID, time.Now().UTC()))
}

// ListCurrentUser 返回指定用户当前可见的公告及其读取状态。
func (s *Service) ListCurrentUser(ctx context.Context, query UserListQuery) (UserListResult, error) {
	if err := s.ensureReady(); err != nil {
		return UserListResult{}, err
	}
	page, size := normalizePage(query.Page, query.PageSize)
	now := time.Now().UTC()
	result, err := s.repository.ListCurrentUser(ctx, announcementstore.UserListQuery{
		UserID:     query.UserID,
		UnreadOnly: query.UnreadOnly,
		Now:        now,
		Limit:      size,
		Offset:     (page - 1) * size,
	})
	if err != nil {
		return UserListResult{}, mapStoreError(err)
	}
	return UserListResult{Items: result.Items, Total: result.Total, Page: page, PageSize: size}, nil
}

// MarkRead 将指定用户当前可见的一条公告标记为已读。
func (s *Service) MarkRead(ctx context.Context, userID uint64, announcementID uint64) (announcementstore.UserAnnouncement, error) {
	if err := s.ensureReady(); err != nil {
		return announcementstore.UserAnnouncement{}, err
	}
	item, err := s.repository.MarkRead(ctx, userID, announcementID, time.Now().UTC())
	return item, mapStoreError(err)
}

// MarkAllRead 将指定用户当前可见的所有未读公告标记为已读。
func (s *Service) MarkAllRead(ctx context.Context, userID uint64) (int, error) {
	if err := s.ensureReady(); err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	count, err := s.repository.MarkAllRead(ctx, userID, now, now)
	return count, mapStoreError(err)
}

// UnreadCount 返回指定用户当前可见的未读公告数量。
func (s *Service) UnreadCount(ctx context.Context, userID uint64) (int, error) {
	if err := s.ensureReady(); err != nil {
		return 0, err
	}
	count, err := s.repository.UnreadCount(ctx, userID, time.Now().UTC())
	return count, mapStoreError(err)
}

func (s *Service) ensureReady() error {
	if s == nil || s.repository == nil {
		return errors.New("announcement service is unavailable")
	}
	return nil
}

func normalizeAdminListQuery(query AdminListQuery) AdminListQuery {
	query.Status = strings.TrimSpace(query.Status)
	query.Level = strings.TrimSpace(query.Level)
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Sort = strings.TrimSpace(query.Sort)
	return query
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

func validAdminSort(sort string) bool {
	switch sort {
	case "", "updated_desc", "publish_desc", "pinned_publish_desc":
		return true
	default:
		return false
	}
}

func validateWriteInput(
	title string,
	content string,
	level string,
	deliveryMode string,
	publishAt *time.Time,
	expireAt *time.Time,
) error {
	if title == "" || content == "" || level == "" {
		return errAnnouncementInvalidInput
	}
	if !validWriteContracts(level, deliveryMode) {
		return errAnnouncementInvalidInput
	}
	if publishAt != nil {
		normalized := publishAt.UTC()
		publishAt = &normalized
	}
	if expireAt != nil {
		normalized := expireAt.UTC()
		expireAt = &normalized
	}
	if publishAt != nil && expireAt != nil && !expireAt.After(*publishAt) {
		return errAnnouncementInvalidInput
	}
	return nil
}

func validWriteContracts(level string, deliveryMode string) bool {
	return announcementcontract.ValidAnnouncementLevel(announcementcontract.AnnouncementLevel(level)) &&
		announcementcontract.ValidAnnouncementDeliveryMode(announcementcontract.AnnouncementDeliveryMode(deliveryMode))
}

func normalizeDeliveryMode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return announcementcontract.AnnouncementDeliveryModeSilent.String()
	}
	return value
}

func mapStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, announcementstore.ErrInvalidInput):
		return errAnnouncementInvalidInput
	case errors.Is(err, announcementstore.ErrAnnouncementNotFound):
		return errAnnouncementNotFound
	default:
		return err
	}
}
