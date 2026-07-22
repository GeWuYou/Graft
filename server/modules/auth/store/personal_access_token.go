package store

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrPersonalAccessTokenNotFound 表示没有匹配当前查询条件的个人 API Token。
	ErrPersonalAccessTokenNotFound = errors.New("personal access token not found")
)

// PersonalAccessToken 是 auth 持久层返回的个人 API Token 生命周期快照。
//
// SecretHash 只用于服务端验证，禁止直接映射到 HTTP 或 MCP 响应。
type PersonalAccessToken struct {
	ID          uint64
	UserID      uint64
	Name        string
	TokenPrefix string
	SecretHash  string
	Scopes      []string
	ExpiresAt   time.Time
	RevokedAt   *time.Time
	LastUsedAt  *time.Time
	CreatedAt   time.Time
}

// CreatePersonalAccessTokenInput 描述写入个人 API Token 所需的完整持久化字段。
type CreatePersonalAccessTokenInput struct {
	UserID      uint64
	Name        string
	TokenPrefix string
	SecretHash  string
	Scopes      []string
	ExpiresAt   time.Time
}

// ListPersonalAccessTokensByUserIDInput 描述按 Token 所有者读取可展示生命周期记录的条件。
type ListPersonalAccessTokensByUserIDInput struct {
	UserID uint64
	Limit  int
}

// RevokePersonalAccessTokenByUserIDInput 描述按所有者定向撤销 Token 的写入条件。
type RevokePersonalAccessTokenByUserIDInput struct {
	UserID    uint64
	TokenID   uint64
	RevokedAt time.Time
}

// PersonalAccessTokenStore 负责 auth 模块拥有的个人 API Token 持久化。
//
// 验证查询必须只返回未软删除的记录；有效期与撤销的业务语义由 auth service 统一判定，
// 以便 HTTP 管理接口与 MCP 认证路径保持同一规则。
type PersonalAccessTokenStore interface {
	CreatePersonalAccessToken(ctx context.Context, input CreatePersonalAccessTokenInput) (PersonalAccessToken, error)
	GetPersonalAccessTokenBySecretHash(ctx context.Context, secretHash string) (PersonalAccessToken, error)
	ListPersonalAccessTokensByUserID(ctx context.Context, input ListPersonalAccessTokensByUserIDInput) ([]PersonalAccessToken, error)
	RevokePersonalAccessTokenByUserID(ctx context.Context, input RevokePersonalAccessTokenByUserIDInput) error
	MarkPersonalAccessTokenUsed(ctx context.Context, tokenID uint64, usedAt time.Time) error
}
