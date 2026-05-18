// Package pluginapi 定义稳定的跨插件能力契约。
package pluginapi

import (
	"context"
	"errors"
)

var (
	// ErrUserNotFound 表示跨插件读取的目标用户不存在。
	ErrUserNotFound = errors.New("user not found")
)

// UserSummary 是跨插件共享的稳定用户摘要 DTO。
//
// 该 DTO 只能承载其他插件明确可依赖的字段，避免把用户插件的内部模型直接泄漏出去。
type UserSummary struct {
	ID       uint64
	Username string
	Display  string
}

// UserService 暴露其他插件可依赖的最小用户能力接口。
//
// 该接口的稳定性高于单个插件内部仓储；一旦签名或错误语义发生变化，需要同步评估所有依赖方。
type UserService interface {
	// GetUserByID 按 ID 返回稳定的用户摘要 DTO，而不是内部持久化模型。
	//
	// 未命中时实现应返回 ErrUserNotFound，方便调用方做统一分支处理。
	GetUserByID(ctx context.Context, id uint64) (UserSummary, error)
}
