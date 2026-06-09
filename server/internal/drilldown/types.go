// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package drilldown

import (
	"context"
	"errors"
)

var (
	// ErrScopeNotFound indicates the requested scope key does not exist.
	ErrScopeNotFound = errors.New("drilldown scope not found")
	// ErrScopeDisabled indicates the stored scope exists but is disabled.
	ErrScopeDisabled = errors.New("drilldown scope disabled")
	// ErrTargetMismatch indicates the stored scope does not belong to the requested page target.
	ErrTargetMismatch = errors.New("drilldown scope target mismatch")
	// ErrScopeConflict indicates the caller supplied filters that conflict with a locked scope.
	ErrScopeConflict = errors.New("drilldown scope conflict")
	// ErrResolverNotFound indicates the drilldown service was used without a resolver.
	ErrResolverNotFound = errors.New("drilldown resolver not found")
	// ErrResolverBadPayload indicates the resolver payload or metadata cannot be interpreted.
	ErrResolverBadPayload = errors.New("drilldown resolver payload is invalid")
)

// ScopeMetadata stores the persisted definition for one drilldown scope.
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

// AppliedScope describes the currently active scope returned to API consumers.
type AppliedScope struct {
	Module      string   `json:"module"`
	Scope       string   `json:"scope"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	OwnedFields []string `json:"owned_fields,omitempty"`
}

// ScopeProjectionItem describes one locked field projection entry for the UI.
type ScopeProjectionItem struct {
	Key      string   `json:"key"`
	LabelKey string   `json:"label_key"`
	Kind     string   `json:"kind"`
	Values   []string `json:"values,omitempty"`
	Locked   bool     `json:"locked"`
}

// ScopeProjection describes how a locked scope should be displayed in the UI.
type ScopeProjection struct {
	Title       string                `json:"title"`
	Description string                `json:"description,omitempty"`
	Items       []ScopeProjectionItem `json:"items,omitempty"`
}

// ConvertibleFilters lists drilldown constraints that can be turned into editable filters.
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

// ResolvedScope contains the resolved scope metadata, display projection, and typed query patch.
type ResolvedScope[T any] struct {
	Metadata           ScopeMetadata
	Applied            AppliedScope
	Projection         ScopeProjection
	ConvertibleFilters ConvertibleFilters
	QueryPatch         T
}

// MetadataRepository loads persisted drilldown scope metadata.
type MetadataRepository interface {
	GetScope(ctx context.Context, module, scope string) (ScopeMetadata, error)
}

// Resolver converts scope metadata into a typed query patch for a target page.
type Resolver[T any, Q any] interface {
	Resolve(ctx context.Context, metadata ScopeMetadata, currentQuery Q) (ResolvedScope[T], error)
}
