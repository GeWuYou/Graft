// Package store 定义 Notification Center 模块的持久化契约。
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrInvalidInput 表示通知 store 输入违反模块持久化契约。
	ErrInvalidInput = errors.New("notification invalid input")
	// ErrDeliveryNotFound 表示指定用户范围内不存在未删除的投递记录。
	ErrDeliveryNotFound = errors.New("notification delivery not found")
)

// Event 保存不可变的通知事实。
type Event struct {
	ID                uint64
	TitleKey          string
	Title             string
	MessageKey        string
	Message           string
	CategoryKey       string
	SourceKey         string
	LevelKey          string
	EventTypeKey      string
	ResourceTypeKey   string
	ActionLabelKey    string
	ActionLabel       string
	Severity          string
	Category          string
	SourceModule      string
	EventType         string
	ResourceType      string
	ResourceID        string
	ResourceName      string
	NavigationKind    string
	NavigationPayload json.RawMessage
	Metadata          json.RawMessage
	DedupeKey         string
	OccurredAt        time.Time
	ExpiresAt         *time.Time
	CreatedAt         time.Time
}

// Delivery 保存按用户划分的通知投递状态。
type Delivery struct {
	ID              uint64
	EventID         uint64
	RecipientUserID uint64
	TargetType      string
	TargetRef       string
	ReadAt          *time.Time
	DeletedAt       int64
	CreatedAt       time.Time
}

// Notification 将一次投递与其通知事实合并，供当前用户读取。
type Notification struct {
	Event    Event
	Delivery Delivery
}

// CreateEventInput 描述一次通知事件写入。
type CreateEventInput struct {
	TitleKey          string
	Title             string
	MessageKey        string
	Message           string
	CategoryKey       string
	SourceKey         string
	LevelKey          string
	EventTypeKey      string
	ResourceTypeKey   string
	ActionLabelKey    string
	ActionLabel       string
	Severity          string
	Category          string
	SourceModule      string
	EventType         string
	ResourceType      string
	ResourceID        string
	ResourceName      string
	NavigationKind    string
	NavigationPayload json.RawMessage
	Metadata          json.RawMessage
	DedupeKey         string
	OccurredAt        time.Time
	ExpiresAt         *time.Time
}

// CreateDeliveryInput 描述一次用户投递写入。
type CreateDeliveryInput struct {
	EventID         uint64
	RecipientUserID uint64
	TargetType      string
	TargetRef       string
}

// ListQuery 描述当前用户通知列表支持的过滤条件。
type ListQuery struct {
	RecipientUserID uint64
	Status          string
	Severity        string
	Category        string
	SourceModule    string
	OccurredFrom    *time.Time
	OccurredTo      *time.Time
	Limit           int
	Offset          int
}

// ListResult 返回分页后的当前用户通知页面。
type ListResult struct {
	Items []Notification
	Total int
}

// Repository 负责通知事件与投递记录的持久化，并保持当前用户读取范围约束。
type Repository interface {
	CreateEvent(ctx context.Context, input CreateEventInput) (Event, bool, error)
	CreateDeliveries(ctx context.Context, inputs []CreateDeliveryInput) ([]Delivery, error)
	List(ctx context.Context, query ListQuery) (ListResult, error)
	Get(ctx context.Context, recipientUserID uint64, deliveryID uint64) (Notification, error)
	UnreadCount(ctx context.Context, recipientUserID uint64) (int, error)
	MarkRead(ctx context.Context, recipientUserID uint64, deliveryID uint64, readAt time.Time) (Delivery, error)
	MarkAllRead(ctx context.Context, recipientUserID uint64, readAt time.Time) (int, error)
	MarkAllReadMatching(ctx context.Context, query ListQuery, readAt time.Time) (int, error)
	DeleteDelivery(ctx context.Context, recipientUserID uint64, deliveryID uint64, deletedAt time.Time) error
}
