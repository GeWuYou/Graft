// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package moduleapi 定义稳定的跨模块能力契约。
package moduleapi

import (
	"context"
	"errors"
)

var (
	// ErrUserNotFound 表示跨模块读取的目标用户不存在。
	ErrUserNotFound = errors.New("user not found")
)

// UserSummary 是跨模块共享的稳定用户摘要 DTO。
//
// 该 DTO 只能承载其他模块明确可依赖的字段，避免把用户模块的内部模型直接泄漏出去。
type UserSummary struct {
	ID       uint64
	Username string
	Display  string
}

// UserService 暴露其他模块可依赖的最小用户能力接口。
//
// 该接口的稳定性高于单个模块内部仓储；一旦签名或错误语义发生变化，需要同步评估所有依赖方。
type UserService interface {
	// GetUserByID 按 ID 返回稳定的用户摘要 DTO，而不是内部持久化模型。
	//
	// 未命中时实现应返回 ErrUserNotFound，方便调用方做统一分支处理。
	GetUserByID(ctx context.Context, id uint64) (UserSummary, error)
	// CountUsers 返回当前可管理用户总数，供跨模块摘要类只读能力使用。
	CountUsers(ctx context.Context) (int, error)
}
