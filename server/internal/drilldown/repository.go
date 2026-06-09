// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package drilldown

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type sqlRepository struct {
	db *sql.DB
}

// NewRepository creates the SQL-backed drilldown metadata repository.
func NewRepository(db *sql.DB) (MetadataRepository, error) {
	if db == nil {
		return nil, errors.New("drilldown repository requires a non-nil sql db")
	}
	return &sqlRepository{db: db}, nil
}

func (r *sqlRepository) GetScope(ctx context.Context, module, scope string) (ScopeMetadata, error) {
	if r == nil || r.db == nil {
		return ScopeMetadata{}, errors.New("drilldown repository is unavailable")
	}

	module = strings.TrimSpace(module)
	scope = strings.TrimSpace(scope)
	if module == "" || scope == "" {
		return ScopeMetadata{}, ErrScopeNotFound
	}

	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, module, scope, name, description, target_type, target_module, target_page, enabled, sort_order
		FROM system_drilldown_scope
		WHERE module = $1 AND scope = $2
		LIMIT 1`,
		module,
		scope,
	)

	var metadata ScopeMetadata
	if err := row.Scan(
		&metadata.ID,
		&metadata.Module,
		&metadata.Scope,
		&metadata.Name,
		&metadata.Description,
		&metadata.TargetType,
		&metadata.TargetModule,
		&metadata.TargetPage,
		&metadata.Enabled,
		&metadata.SortOrder,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ScopeMetadata{}, ErrScopeNotFound
		}
		return ScopeMetadata{}, fmt.Errorf("read drilldown scope metadata: %w", err)
	}

	return metadata, nil
}
