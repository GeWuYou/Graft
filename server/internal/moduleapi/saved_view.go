// Package moduleapi 定义稳定的跨模块能力契约。
package moduleapi

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrSavedViewNotFound 表示请求的已保存视图不属于当前调用方。
	ErrSavedViewNotFound = errors.New("saved view not found")
	// ErrSavedViewConflict 表示该所有者和 surface 下已存在同名已保存视图。
	ErrSavedViewConflict = errors.New("saved view conflict")
	// ErrSavedViewInvalidInput 表示通用已保存视图状态的结构不合法。
	ErrSavedViewInvalidInput = errors.New("saved view invalid input")
)

// SavedView 是与消费者无关的持久化列表页视图；QueryState 对该服务保持不透明。
type SavedView struct {
	ID             uint64
	OwnerUserID    uint64
	SurfaceKey     string
	Name           string
	QueryState     json.RawMessage
	PageSize       int
	VisibleColumns []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SavedViewCreateInput 描述消费者完成 surface 和查询状态校验后创建视图所需的输入。
type SavedViewCreateInput struct {
	OwnerUserID    uint64
	SurfaceKey     string
	Name           string
	QueryState     json.RawMessage
	PageSize       int
	VisibleColumns []string
}

// SavedViewUpdateInput 描述更新 OwnerUserID 所有的已有视图所需的输入。
type SavedViewUpdateInput struct {
	ID             uint64
	OwnerUserID    uint64
	SurfaceKey     string
	Name           string
	QueryState     json.RawMessage
	PageSize       int
	VisibleColumns []string
}

// SavedViewService 是通用持久化边界；授权和载荷语义仍由消费者拥有。
type SavedViewService interface {
	List(ctx context.Context, ownerUserID uint64, surfaceKey string) ([]SavedView, error)
	Create(ctx context.Context, input SavedViewCreateInput) (SavedView, error)
	Update(ctx context.Context, input SavedViewUpdateInput) (SavedView, error)
	Delete(ctx context.Context, ownerUserID uint64, surfaceKey string, id uint64) error
}
