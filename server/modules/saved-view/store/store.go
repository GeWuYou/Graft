// Package store persists generic saved-view state without interpreting consumer filters.
package store

import (
	"context"

	"graft/server/internal/moduleapi"
)

// Repository 持久化通用保存视图，但不解释消费者保存的查询状态。
type Repository interface {
	// List 返回指定用户和消费界面的未删除视图。
	List(context.Context, uint64, string) ([]moduleapi.SavedView, error)
	// Create 创建一个视图；所有权和消费者状态约束由实现校验。
	Create(context.Context, moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error)
	// Update 更新一个仍由指定用户和消费界面拥有的视图。
	Update(context.Context, moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error)
	// Delete 软删除一个仍由指定用户和消费界面拥有的视图。
	Delete(context.Context, uint64, string, uint64) error
}
