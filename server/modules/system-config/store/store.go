// Package store 定义 system-config 模块仅持久化用户覆盖值的存储契约。
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ErrOverrideNotFound 表示指定 key 没有用户覆盖值；调用方可据此回退到模块默认值。
var ErrOverrideNotFound = errors.New("system config override not found")

// Override 保存一个配置 key 的用户 JSON 覆盖值及其审计字段；模块默认值不进入该结构。
type Override struct {
	Key       string
	Value     json.RawMessage
	CreatedAt time.Time
	CreatedBy *uint64
	UpdatedAt time.Time
	UpdatedBy *uint64
}

// Repository 只读写用户覆盖值；有效配置值的默认值合并与校验由上层 Service 负责。
type Repository interface {
	ListOverrides(ctx context.Context) ([]Override, error)
	GetOverride(ctx context.Context, key string) (Override, error)
	SetOverride(ctx context.Context, key string, value json.RawMessage, userID *uint64) (Override, error)
	DeleteOverride(ctx context.Context, key string) error
}
