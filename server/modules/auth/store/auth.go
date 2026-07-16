// Package store 定义 auth 与持久层之间的窄化 credential、密码变更和 refresh session 契约。
package store

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCredentialNotFound 表示 profile 存在但尚未配置 auth credential；调用方可据此执行首次 credential provisioning。
	ErrCredentialNotFound = errors.New("credential not found")

	// ErrRefreshSessionNotFound 表示 token 对应的 refresh session 不存在；auth 将其映射为无效 refresh token。
	ErrRefreshSessionNotFound = errors.New("refresh session not found")
)

// UserCredential 是 auth runtime 使用的 credential 快照；PasswordHash 可为空表示尚未配置可验证的密码。
type UserCredential struct {
	UserID             uint64
	Username           string
	PasswordHash       *string
	MustChangePassword bool
	PasswordChangedAt  *time.Time
}

// SetPasswordHashInput 描述 credential 更新及强制改密状态变更；ChangedAt 为空表示由 store 使用默认更新时间语义。
type SetPasswordHashInput struct {
	UserID             uint64
	PasswordHash       string
	MustChangePassword bool
	ChangedAt          *time.Time
}

// ResetPasswordAndRevokeSessionsInput 描述管理员重置密码的原子操作输入，要求同时吊销用户现有 refresh session。
type ResetPasswordAndRevokeSessionsInput struct {
	UserID             uint64
	PasswordHash       string
	MustChangePassword bool
	ChangedAt          time.Time
}

// ChangePasswordAndRevokeOtherRefreshSessionsInput 描述自助改密的原子操作输入；CurrentTokenID 对应 session 会被保留。
type ChangePasswordAndRevokeOtherRefreshSessionsInput struct {
	UserID             uint64
	PasswordHash       string
	MustChangePassword bool
	ChangedAt          time.Time
	CurrentTokenID     string
}

// EnsureUserCredentialInput 描述首次创建 credential 的输入；user profile 必须已由 identity owner 创建。
type EnsureUserCredentialInput struct {
	Username           string
	Display            string
	PasswordHash       string
	MustChangePassword bool
}

// RevokeOtherRefreshSessionsInput 描述保留 CurrentTokenID、吊销同一用户其它 refresh session 的输入。
type RevokeOtherRefreshSessionsInput struct {
	UserID         uint64
	CurrentTokenID string
	RevokedAt      time.Time
}

// RefreshSession 是 auth runtime 的 session 状态快照；RevokedAt 和 ReplacedByTokenID 表达轮换或主动吊销结果。
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

// ListActiveRefreshSessionsByUserIDInput 描述按用户和当前时间读取未过期、未吊销 session 的输入。
type ListActiveRefreshSessionsByUserIDInput struct {
	UserID uint64
	Now    time.Time
}

// CreateRefreshSessionInput 描述服务端 refresh session 持久化输入；TokenID 必须与签发 token 的 claims 一致。
type CreateRefreshSessionInput struct {
	UserID    uint64
	TokenID   string
	ExpiresAt time.Time
}

// RevokeRefreshSessionInput 描述单个 session 的吊销输入；轮换时同时记录替代 token 标识。
type RevokeRefreshSessionInput struct {
	TokenID           string
	RevokedAt         time.Time
	ReplacedByTokenID *string
}

// RevokeRefreshSessionsByUserIDInput 描述吊销用户全部 refresh session 的输入。
type RevokeRefreshSessionsByUserIDInput struct {
	UserID    uint64
	RevokedAt time.Time
}

// RevokeRefreshSessionByUserIDInput 描述校验用户归属后定向吊销 session 的输入。
type RevokeRefreshSessionByUserIDInput struct {
	UserID    uint64
	TokenID   string
	RevokedAt time.Time
}

// RotateRefreshSessionInput 描述将当前 session 标记为已替代并创建新 session 的原子轮换输入。
type RotateRefreshSessionInput struct {
	CurrentTokenID string
	NewTokenID     string
	Now            time.Time
	RevokedAt      time.Time
	NewExpiresAt   time.Time
}

// PasswordChangeRepository 暴露自助改密所需的原子写入契约，避免密码已更新而旧 session 仍可用。
type PasswordChangeRepository interface {
	ChangePasswordAndRevokeOtherRefreshSessions(
		ctx context.Context,
		input ChangePasswordAndRevokeOtherRefreshSessionsInput,
	) error
}

// AuthRepository 暴露开发重置等完整 auth 持久化操作；运行时优先依赖更窄的 CredentialStore 或 SessionStore。
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

// CredentialStore 负责 auth credential 持久化；它与 user profile identity 分离，使 auth runtime 不依赖 user 的存储实现。
type CredentialStore interface {
	GetUserCredentialByUsername(ctx context.Context, username string) (UserCredential, error)
	SetPasswordHash(ctx context.Context, input SetPasswordHashInput) error
	EnsureUserCredential(ctx context.Context, input EnsureUserCredentialInput) (UserCredential, error)
	ResetPasswordAndRevokeRefreshSessions(ctx context.Context, input ResetPasswordAndRevokeSessionsInput) error
}

// SessionStore 负责 refresh session 的创建、轮换、查询和吊销，令 token 生命周期与 JWT 本身分离。
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
