package store

import (
	"context"
	"errors"
	"time"
)

// ErrRefreshSessionNotFound 表示请求的刷新会话不存在。
var ErrRefreshSessionNotFound = errors.New("refresh session not found")

// UserCredential 表示认证路径可见的最小用户口令 DTO。
//
// 该 DTO 与 User 分离，避免普通用户资料读取能力意外获得密码散列等敏感字段。
type UserCredential struct {
	UserID            uint64
	Username          string
	PasswordHash      *string
	PasswordChangedAt *time.Time
}

// SetPasswordHashInput 描述一次密码散列更新所需的最小输入。
type SetPasswordHashInput struct {
	UserID       uint64
	PasswordHash string
	ChangedAt    time.Time
}

// RefreshSession 表示 refresh token 生命周期对应的稳定持久化 DTO。
type RefreshSession struct {
	ID                uint64
	UserID            uint64
	TokenID           string
	ExpiresAt         time.Time
	RevokedAt         *time.Time
	ReplacedByTokenID *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ListActiveRefreshSessionsByUserIDInput 描述一次按用户读取当前有效刷新会话列表所需的最小输入。
type ListActiveRefreshSessionsByUserIDInput struct {
	UserID uint64
	Now    time.Time
}

// CreateRefreshSessionInput 描述一次刷新会话创建所需的最小输入。
type CreateRefreshSessionInput struct {
	UserID    uint64
	TokenID   string
	ExpiresAt time.Time
}

// RevokeRefreshSessionInput 描述一次刷新会话吊销所需的最小输入。
type RevokeRefreshSessionInput struct {
	TokenID           string
	RevokedAt         time.Time
	ReplacedByTokenID *string
}

// RevokeRefreshSessionsByUserIDInput 描述一次按用户吊销全部刷新会话所需的最小输入。
type RevokeRefreshSessionsByUserIDInput struct {
	UserID    uint64
	RevokedAt time.Time
}

// RevokeRefreshSessionByUserIDInput 描述一次按用户定向吊销单个刷新会话所需的最小输入。
type RevokeRefreshSessionByUserIDInput struct {
	UserID    uint64
	TokenID   string
	RevokedAt time.Time
}

// RotateRefreshSessionInput 描述一次 refresh session 轮换所需的最小输入。
//
// 该输入把“吊销旧会话并创建新会话”收敛为一个显式仓储操作，避免插件层在并发
// refresh 时通过多次独立调用暴露双消费窗口。
type RotateRefreshSessionInput struct {
	CurrentTokenID string
	NewTokenID     string
	Now            time.Time
	RevokedAt      time.Time
	NewExpiresAt   time.Time
}

// AuthRepository 暴露未来认证插件所需的最小持久化操作集合。
//
// 该接口只提供口令与 refresh session 的存储能力，不承载登录、签发或授权决策。
type AuthRepository interface {
	// GetUserCredentialByUsername 按用户名读取口令校验所需的最小用户信息。
	//
	// 未命中时统一返回 ErrUserNotFound。
	GetUserCredentialByUsername(ctx context.Context, username string) (UserCredential, error)

	// SetPasswordHash 为指定用户写入口令散列及其最近变更时间。
	//
	// 当用户不存在时统一返回 ErrUserNotFound。
	SetPasswordHash(ctx context.Context, input SetPasswordHashInput) error

	// CreateRefreshSession 持久化一条新的刷新会话记录。
	CreateRefreshSession(ctx context.Context, input CreateRefreshSessionInput) (RefreshSession, error)

	// GetRefreshSessionByTokenID 按 token 标识读取刷新会话状态。
	//
	// 未命中时统一返回 ErrRefreshSessionNotFound。
	GetRefreshSessionByTokenID(ctx context.Context, tokenID string) (RefreshSession, error)

	// RevokeRefreshSession 吊销一条刷新会话，并可选记录轮换后的新 token 标识。
	//
	// 未命中时统一返回 ErrRefreshSessionNotFound。
	RevokeRefreshSession(ctx context.Context, input RevokeRefreshSessionInput) error

	// RevokeRefreshSessionsByUserID 吊销某个用户名下全部尚未吊销的刷新会话。
	//
	// 该操作应保持幂等，允许同一用户在没有可吊销会话时直接成功返回。
	RevokeRefreshSessionsByUserID(ctx context.Context, input RevokeRefreshSessionsByUserIDInput) error

	// RevokeRefreshSessionByUserID 按用户定向吊销一条当前有效的刷新会话。
	//
	// 该操作只允许命中指定用户、未吊销且未过期的会话；未命中时统一返回
	// ErrRefreshSessionNotFound。
	RevokeRefreshSessionByUserID(ctx context.Context, input RevokeRefreshSessionByUserIDInput) error

	// ListActiveRefreshSessionsByUserID 按用户读取当前有效的 refresh session 列表。
	//
	// 返回值只包含稳定 session DTO，不暴露底层 ORM 结构；实现应过滤已吊销或
	// 已过期记录，并保持显式且稳定的排序，便于插件层构造一致的会话治理视图。
	ListActiveRefreshSessionsByUserID(ctx context.Context, input ListActiveRefreshSessionsByUserIDInput) ([]RefreshSession, error)

	// RotateRefreshSession 以事务方式完成一次 refresh session 轮换。
	//
	// 实现必须保证：只有当前 token 仍未吊销时才允许轮换，并在同一持久化操作中
	// 写入旧会话吊销状态与新会话记录；未命中或已不可用时统一返回
	// ErrRefreshSessionNotFound。
	RotateRefreshSession(ctx context.Context, input RotateRefreshSessionInput) (RefreshSession, error)
}
