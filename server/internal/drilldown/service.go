package drilldown

import (
	"context"
	"fmt"
	"strings"
)

// Service resolves stored drilldown scopes into query patches and UI metadata.
type Service[T any, Q any] struct {
	repo     MetadataRepository
	resolver Resolver[T, Q]
}

// NewService creates a drilldown service with the required metadata repository and resolver.
func NewService[T any, Q any](repo MetadataRepository, resolver Resolver[T, Q]) (*Service[T, Q], error) {
	if repo == nil {
		return nil, fmt.Errorf("new drilldown service: metadata repository is required")
	}
	if resolver == nil {
		return nil, fmt.Errorf("new drilldown service: resolver is required")
	}
	return &Service[T, Q]{repo: repo, resolver: resolver}, nil
}

// ResolveScope loads one stored scope and converts it into a typed query patch.
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
