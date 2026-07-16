package drilldown

import (
	"context"
	"fmt"
	"strings"
)

// Service 将已存储的 drilldown scope 解析为查询补丁和 UI 元数据。
type Service[T any, Q any] struct {
	repo     MetadataRepository
	resolver Resolver[T, Q]
}

// NewService 使用必需的元数据仓储和 resolver 创建 drilldown service。
func NewService[T any, Q any](repo MetadataRepository, resolver Resolver[T, Q]) (*Service[T, Q], error) {
	if repo == nil {
		return nil, fmt.Errorf("new drilldown service: metadata repository is required")
	}
	if resolver == nil {
		return nil, fmt.Errorf("new drilldown service: resolver is required")
	}
	return &Service[T, Q]{repo: repo, resolver: resolver}, nil
}

// ResolveScope 读取一个已存储 scope，校验其启用状态和目标页面后转换为类型化查询补丁。
func (s *Service[T, Q]) ResolveScope(
	ctx context.Context,
	module string,
	page string,
	scope string,
	currentQuery Q,
) (ResolvedScope[T], error) {
	if s == nil || s.repo == nil || s.resolver == nil {
		return ResolvedScope[T]{}, ErrResolverNotFound
	}

	metadata, err := s.repo.GetScope(ctx, module, scope)
	if err != nil {
		return ResolvedScope[T]{}, err
	}
	if !metadata.Enabled {
		return ResolvedScope[T]{}, ErrScopeDisabled
	}
	if metadata.TargetType != "log_query" ||
		!strings.EqualFold(strings.TrimSpace(metadata.TargetModule), strings.TrimSpace(module)) ||
		!strings.EqualFold(strings.TrimSpace(metadata.TargetPage), strings.TrimSpace(page)) {
		return ResolvedScope[T]{}, ErrTargetMismatch
	}

	resolved, err := s.resolver.Resolve(ctx, metadata, currentQuery)
	if err != nil {
		return ResolvedScope[T]{}, err
	}
	return resolved, nil
}
