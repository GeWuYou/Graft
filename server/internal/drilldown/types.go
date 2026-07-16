package drilldown

import (
	"context"
	"errors"
)

var (
	// ErrScopeNotFound 表示请求的 scope 键不存在。
	ErrScopeNotFound = errors.New("drilldown scope not found")
	// ErrScopeDisabled 表示已存储的 scope 存在但已禁用。
	ErrScopeDisabled = errors.New("drilldown scope disabled")
	// ErrTargetMismatch 表示已存储的 scope 不属于请求的页面目标。
	ErrTargetMismatch = errors.New("drilldown scope target mismatch")
	// ErrScopeConflict 表示调用方提供的筛选条件与锁定 scope 冲突。
	ErrScopeConflict = errors.New("drilldown scope conflict")
	// ErrResolverNotFound 表示 drilldown service 缺少 resolver 依赖。
	ErrResolverNotFound = errors.New("drilldown resolver not found")
	// ErrResolverBadPayload 表示 resolver 的 payload 或 metadata 无法解释。
	ErrResolverBadPayload = errors.New("drilldown resolver payload is invalid")
)

// ScopeMetadata 保存一个 drilldown scope 的持久化定义及其目标页面约束。
type ScopeMetadata struct {
	ID           uint64
	Module       string
	Scope        string
	Name         string
	Description  string
	TargetType   string
	TargetModule string
	TargetPage   string
	Enabled      bool
	SortOrder    int
}

// AppliedScope 描述返回给 API 消费者的当前生效 scope。
type AppliedScope struct {
	Module      string   `json:"module"`
	Scope       string   `json:"scope"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	OwnedFields []string `json:"owned_fields,omitempty"`
}

// ScopeProjectionItem 描述 UI 展示的一个锁定字段投影项。
type ScopeProjectionItem struct {
	Key      string   `json:"key"`
	LabelKey string   `json:"label_key"`
	Kind     string   `json:"kind"`
	Values   []string `json:"values,omitempty"`
	Locked   bool     `json:"locked"`
}

// ScopeProjection 描述 UI 应如何展示锁定 scope。
type ScopeProjection struct {
	Title       string                `json:"title"`
	Description string                `json:"description,omitempty"`
	Items       []ScopeProjectionItem `json:"items,omitempty"`
}

// ConvertibleFilters 列出可以转换为可编辑筛选条件的 drilldown 约束。
type ConvertibleFilters struct {
	ActionKeywords      []string `json:"action_keywords,omitempty"`
	ActionPrefixes      []string `json:"action_prefixes,omitempty"`
	ResourceTypes       []string `json:"resource_types,omitempty"`
	RequestPathPrefixes []string `json:"request_path_prefixes,omitempty"`
	Results             []string `json:"results,omitempty"`
	RiskLevels          []string `json:"risk_levels,omitempty"`
	Preset              string   `json:"preset,omitempty"`
	Source              string   `json:"source,omitempty"`
	BusinessCategory    string   `json:"business_category,omitempty"`
	Success             *bool    `json:"success,omitempty"`
}

// ResolvedScope 包含解析后的 scope 元数据、展示投影和类型化查询补丁。
type ResolvedScope[T any] struct {
	Metadata           ScopeMetadata
	Applied            AppliedScope
	Projection         ScopeProjection
	ConvertibleFilters ConvertibleFilters
	QueryPatch         T
}

// MetadataRepository 读取持久化的 drilldown scope 元数据。
type MetadataRepository interface {
	GetScope(ctx context.Context, module, scope string) (ScopeMetadata, error)
}

// Resolver 将 scope 元数据转换为目标页面使用的类型化查询补丁。
type Resolver[T any, Q any] interface {
	Resolve(ctx context.Context, metadata ScopeMetadata, currentQuery Q) (ResolvedScope[T], error)
}
