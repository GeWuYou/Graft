// Package store 定义公告中心模块的持久化契约；仓储负责保留公告生命周期和按用户隔离的阅读事实，服务层负责用例规则。
package store

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrInvalidInput 表示公告 store 输入违反模块持久化契约。
	ErrInvalidInput = errors.New("announcement invalid input")
	// ErrAnnouncementNotFound 表示请求的 ID 不存在未删除的公告记录。
	ErrAnnouncementNotFound = errors.New("announcement not found")
)

// Announcement 保存管理端公告记录；DeletedAt 非空时记录仍可被审计但不再参与正常读取。
type Announcement struct {
	ID           uint64
	Title        string
	Content      string
	Level        string
	Status       string
	DeliveryMode string
	Pinned       bool
	PublishAt    *time.Time
	PublishedAt  *time.Time
	PublishedBy  *uint64
	ArchivedAt   *time.Time
	ExpireAt     *time.Time
	CreatedBy    *uint64
	UpdatedBy    *uint64
	DeletedBy    *uint64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    int64
}

// AnnouncementRead 保存一个用户对一条公告的阅读事实；该事实按用户和公告唯一归属。
type AnnouncementRead struct {
	ID             uint64
	AnnouncementID uint64
	UserID         uint64
	ReadAt         time.Time
	CreatedAt      time.Time
}

// UserAnnouncement 将当前用户可见的公告与该用户的阅读状态合并，避免调用方自行拼接跨用户数据。
type UserAnnouncement struct {
	Announcement Announcement
	ReadAt       *time.Time
}

// CreateInput 描述一次公告写入；创建入口应将状态固定为草稿，发布由独立生命周期操作完成。
type CreateInput struct {
	Title        string
	Content      string
	Level        string
	Status       string
	DeliveryMode string
	Pinned       bool
	PublishAt    *time.Time
	ExpireAt     *time.Time
	ActorID      *uint64
}

// UpdateInput 描述公告可编辑字段；状态和删除事实不由普通更新覆盖。
type UpdateInput struct {
	Title        string
	Content      string
	Level        string
	DeliveryMode string
	Pinned       bool
	PublishAt    *time.Time
	ExpireAt     *time.Time
	ActorID      *uint64
}

// ListQuery 描述管理端公告筛选条件；Limit 和 Offset 由服务层规范化后才交给仓储。
type ListQuery struct {
	Status  string
	Level   string
	Pinned  *bool
	Keyword string
	Sort    string
	Limit   int
	Offset  int
}

// ListResult 返回管理端公告分页结果及总数。
type ListResult struct {
	Items []Announcement
	Total int
}

// UserListQuery 描述当前用户公告筛选条件；仓储必须同时应用发布时间、过期时间、删除状态和用户范围。
type UserListQuery struct {
	UserID     uint64
	UnreadOnly bool
	Now        time.Time
	Limit      int
	Offset     int
}

// UserListResult 返回当前用户可见公告分页结果及总数。
type UserListResult struct {
	Items []UserAnnouncement
	Total int
}

// Repository 持久化公告记录和按用户划分的阅读事实；用户查询不得退化为跨用户的公告全量读取。
type Repository interface {
	Ping(ctx context.Context) error
	ListAdmin(ctx context.Context, query ListQuery) (ListResult, error)
	ListCurrentUser(ctx context.Context, query UserListQuery) (UserListResult, error)
	Create(ctx context.Context, input CreateInput) (Announcement, error)
	GetAdmin(ctx context.Context, id uint64) (Announcement, error)
	Update(ctx context.Context, id uint64, input UpdateInput) (Announcement, error)
	Publish(ctx context.Context, id uint64, publishAt *time.Time, publishedAt time.Time, actorID *uint64) (Announcement, error)
	Archive(ctx context.Context, id uint64, actorID *uint64) (Announcement, error)
	Delete(ctx context.Context, id uint64, actorID uint64, deletedAt time.Time) error
	MarkRead(ctx context.Context, userID uint64, announcementID uint64, readAt time.Time) (UserAnnouncement, error)
	MarkAllRead(ctx context.Context, userID uint64, readAt time.Time, now time.Time) (int, error)
	UnreadCount(ctx context.Context, userID uint64, now time.Time) (int, error)
}
