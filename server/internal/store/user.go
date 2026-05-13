package store

import (
	"context"
	"errors"
	"time"
)

// ErrUserNotFound 表示请求的用户不存在。
var ErrUserNotFound = errors.New("user not found")

// User 表示用户仓储向上层返回的持久化记录。
type User struct {
	ID        uint64
	Username  string
	Display   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepository 暴露当前 MVP 插件面所需的最小用户持久化操作集合。
type UserRepository interface {
	// GetByID 按 ID 返回单个用户记录，未命中时返回 ErrUserNotFound。
	GetByID(ctx context.Context, id uint64) (User, error)
}
