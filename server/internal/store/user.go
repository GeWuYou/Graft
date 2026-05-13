package store

import (
	"context"
	"errors"
	"time"
)

// ErrUserNotFound 表示请求的用户不存在。
var ErrUserNotFound = errors.New("user not found")

// User 表示用户仓储向上层返回的稳定持久化 DTO。
//
// 该类型刻意不暴露 Ent 等 ORM 细节，保证插件与上层服务不依赖具体存储实现。
type User struct {
	ID        uint64
	Username  string
	Display   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepository 暴露当前 MVP 插件面所需的最小用户持久化操作集合。
//
// 实现方需要把底层“未命中”语义统一收敛为 ErrUserNotFound，避免上层感知具体存储驱动。
type UserRepository interface {
	// GetByID 按 ID 返回单个用户记录，未命中时返回 ErrUserNotFound。
	GetByID(ctx context.Context, id uint64) (User, error)
}
