// Package store 定义 auth 所有的 credential 与 refresh-session 持久化契约。
package store

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCredentialNotFound 表示用户 profile 尚未配置 auth credential。
	ErrCredentialNotFound = errors.New("credential not found")

	// ErrRefreshSessionNotFound 表示请求的 refresh session 不存在。
	ErrRefreshSessionNotFound = errors.New("refresh session not found")
)

// UserCredential 是 auth runtime 使用的最小 credential DTO。
type UserCredential struct {
	UserID             uint64
	Username           string
	PasswordHash       *string
	MustChangePassword bool
	PasswordChangedAt  *time.Time
}

// SetPasswordHashInput 描述最小密码哈希更新输入。
type SetPasswordHashInput struct {
	UserID             uint64
	PasswordHash       string
	MustChangePassword bool
	ChangedAt          *time.Time
}

// ResetPasswordAndRevokeSessionsInput 描述管理员重置密码所需的最小输入。
type ResetPasswordAndRevokeSessionsInput struct {
	UserID             uint64
	PasswordHash       string
	MustChangePassword bool
	ChangedAt          time.Time
}

// ChangePasswordAndRevokeOtherRefreshSessionsInput 描述保留当前 session、
// 同时吊销其它 refresh session 的最小密码变更输入。
type ChangePasswordAndRevokeOtherRefreshSessionsInput struct {
	UserID             uint64
	PasswordHash       string
	MustChangePassword bool
	ChangedAt          time.Time
	CurrentTokenID     string
}

// EnsureUserCredentialInput 描述确保 credential 存在所需的最小输入。
type EnsureUserCredentialInput struct {
	Username           string
	Display            string
	PasswordHash       string
	MustChangePassword bool
}

// RevokeOtherRefreshSessionsInput 描述吊销其它 refresh session 的最小输入。
type RevokeOtherRefreshSessionsInput struct {
	UserID         uint64
	CurrentTokenID string
	RevokedAt      time.Time
}

// RefreshSession 是 auth runtime 使用的稳定 refresh-session DTO。
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

// ListActiveRefreshSessionsByUserIDInput 描述读取活跃 session 的最小查询输入。
type ListActiveRefreshSessionsByUserIDInput struct {
	UserID uint64
	Now    time.Time
}

// CreateRefreshSessionInput 描述创建 refresh session 的最小输入。
type CreateRefreshSessionInput struct {
	UserID    uint64
	TokenID   string
	ExpiresAt time.Time
}

// RevokeRefreshSessionInput 描述吊销单个 refresh session 的最小输入。
type RevokeRefreshSessionInput struct {
	TokenID           string
	RevokedAt         time.Time
	ReplacedByTokenID *string
}

// RevokeRefreshSessionsByUserIDInput 描述按用户批量吊销 refresh session 的最小输入。
type RevokeRefreshSessionsByUserIDInput struct {
	UserID    uint64
	RevokedAt time.Time
}

// RevokeRefreshSessionByUserIDInput 描述按用户和 token 定向吊销 session 的最小输入。
type RevokeRefreshSessionByUserIDInput struct {
	UserID    uint64
	TokenID   string
	RevokedAt time.Time
}

// RotateRefreshSessionInput 描述一次 refresh session 轮换操作。
type RotateRefreshSessionInput struct {
	CurrentTokenID string
	NewTokenID     string
	Now            time.Time
	RevokedAt      time.Time
	NewExpiresAt   time.Time
}

// PasswordChangeRepository 暴露原子密码变更写入契约。
type PasswordChangeRepository interface {
	ChangePasswordAndRevokeOtherRefreshSessions(
		ctx context.Context,
		input ChangePasswordAndRevokeOtherRefreshSessionsInput,
	) error
}

// AuthRepository 暴露 auth 所有的 credential 与 session 持久化能力。
type AuthRepository interface {
	GetUserCredentialByUsername(ctx context.Context, username string) (UserCredential, error)
	SetPasswordHash(ctx context.Context, input SetPasswordHashInput) error
	EnsureUserCredential(ctx context.Context, input EnsureUserCredentialInput) (UserCredential, error)
	CreateRefreshSession(ctx context.Context, input CreateRefreshSessionInput) (RefreshSession, error)
	GetRefreshSessionByTokenID(ctx context.Context, tokenID string) (RefreshSession, error)
	RevokeRefreshSession(ctx context.Context, input RevokeRefreshSessionInput) error
	RevokeRefreshSessionsByUserID(ctx context.Context, input RevokeRefreshSessionsByUserIDInput) error
	RevokeOtherRefreshSessionsByUserID(ctx context.Context, input RevokeOtherRefreshSessionsInput) error
	RevokeRefreshSessionByUserID(ctx context.Context, input RevokeRefreshSessionByUserIDInput) error
	ListActiveRefreshSessionsByUserID(ctx context.Context, input ListActiveRefreshSessionsByUserIDInput) ([]RefreshSession, error)
	RotateRefreshSession(ctx context.Context, input RotateRefreshSessionInput) (RefreshSession, error)
	ResetPasswordAndRevokeRefreshSessions(ctx context.Context, input ResetPasswordAndRevokeSessionsInput) error
}

// CredentialStore 负责密码 credential 持久化。
// 它有意与 user-profile identity 分离，使底层 schema 迁移时无需改变 auth runtime 依赖。
type CredentialStore interface {
	GetUserCredentialByUsername(ctx context.Context, username string) (UserCredential, error)
	SetPasswordHash(ctx context.Context, input SetPasswordHashInput) error
	EnsureUserCredential(ctx context.Context, input EnsureUserCredentialInput) (UserCredential, error)
	ResetPasswordAndRevokeRefreshSessions(ctx context.Context, input ResetPasswordAndRevokeSessionsInput) error
}

// SessionStore 负责 refresh-session 生命周期持久化。
type SessionStore interface {
	CreateRefreshSession(ctx context.Context, input CreateRefreshSessionInput) (RefreshSession, error)
	GetRefreshSessionByTokenID(ctx context.Context, tokenID string) (RefreshSession, error)
	RevokeRefreshSession(ctx context.Context, input RevokeRefreshSessionInput) error
	RevokeRefreshSessionsByUserID(ctx context.Context, input RevokeRefreshSessionsByUserIDInput) error
	RevokeOtherRefreshSessionsByUserID(ctx context.Context, input RevokeOtherRefreshSessionsInput) error
	RevokeRefreshSessionByUserID(ctx context.Context, input RevokeRefreshSessionByUserIDInput) error
	ListActiveRefreshSessionsByUserID(ctx context.Context, input ListActiveRefreshSessionsByUserIDInput) ([]RefreshSession, error)
	RotateRefreshSession(ctx context.Context, input RotateRefreshSessionInput) (RefreshSession, error)
}
