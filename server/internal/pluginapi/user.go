// Package pluginapi 定义稳定的跨插件能力契约。
package pluginapi

import "context"

// UserSummary 是跨插件共享的稳定用户摘要 DTO。
type UserSummary struct {
	ID       uint64
	Username string
	Display  string
}

// UserService 暴露其他插件可依赖的最小用户能力接口。
type UserService interface {
	// GetUserByID 按 ID 返回稳定的用户摘要 DTO，而不是内部持久化模型。
	GetUserByID(ctx context.Context, id uint64) (UserSummary, error)
}
