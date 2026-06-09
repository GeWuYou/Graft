// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package storeent provides SQL persistence for system-config overrides.
package storeent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	systemconfigstore "graft/server/modules/system-config/store"
)

type repository struct {
	db *sql.DB
}

// NewRepository builds a SQL-backed system config override repository.
func NewRepository(db *sql.DB) (systemconfigstore.Repository, error) {
	if db == nil {
		return nil, errors.New("system config repository requires a non-nil sql db")
	}
	return &repository{db: db}, nil
}

func (r *repository) GetOverride(ctx context.Context, key string) (systemconfigstore.Override, error) {
	if r == nil || r.db == nil {
		return systemconfigstore.Override{}, errors.New("system config repository is unavailable")
	}

	row := r.db.QueryRowContext(
		ctx,
		`SELECT key, override_value, created_at, created_by, updated_at, updated_by
		 FROM system_config_values WHERE key = $1`,
		strings.TrimSpace(key),
	)
	override, err := scanOverride(row)
	if errors.Is(err, sql.ErrNoRows) {
		return systemconfigstore.Override{}, systemconfigstore.ErrOverrideNotFound
	}
	if err != nil {
		return systemconfigstore.Override{}, fmt.Errorf("get system config override: %w", err)
	}
	return override, nil
}

func (r *repository) SetOverride(ctx context.Context, key string, value json.RawMessage, userID *uint64) (systemconfigstore.Override, error) {
	if r == nil || r.db == nil {
		return systemconfigstore.Override{}, errors.New("system config repository is unavailable")
	}
	userIDValue, err := nullableInt64(userID)
	if err != nil {
		return systemconfigstore.Override{}, fmt.Errorf("set system config override: %w", err)
	}

	row := r.db.QueryRowContext(
		ctx,
		`INSERT INTO system_config_values (key, override_value, created_at, created_by, updated_at, updated_by)
		 VALUES ($1, $2, NOW(), $3, NOW(), $3)
		 ON CONFLICT (key)
		 DO UPDATE SET override_value = EXCLUDED.override_value, updated_at = NOW(), updated_by = EXCLUDED.updated_by
		 RETURNING key, override_value, created_at, created_by, updated_at, updated_by`,
		strings.TrimSpace(key),
		value,
		userIDValue,
	)
	override, err := scanOverride(row)
	if err != nil {
		return systemconfigstore.Override{}, fmt.Errorf("set system config override: %w", err)
	}
	return override, nil
}

func (r *repository) DeleteOverride(ctx context.Context, key string) error {
	if r == nil || r.db == nil {
		return errors.New("system config repository is unavailable")
	}

	if _, err := r.db.ExecContext(ctx, `DELETE FROM system_config_values WHERE key = $1`, strings.TrimSpace(key)); err != nil {
		return fmt.Errorf("delete system config override: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOverride(row rowScanner) (systemconfigstore.Override, error) {
	var override systemconfigstore.Override
	var createdAt time.Time
	var createdBy sql.NullInt64
	var updatedAt time.Time
	var updatedBy sql.NullInt64
	if err := row.Scan(&override.Key, &override.Value, &createdAt, &createdBy, &updatedAt, &updatedBy); err != nil {
		return systemconfigstore.Override{}, err
	}
	override.CreatedAt = createdAt.UTC()
	override.CreatedBy = uint64FromNullInt64(createdBy)
	override.UpdatedAt = updatedAt.UTC()
	override.UpdatedBy = uint64FromNullInt64(updatedBy)
	return override, nil
}

func nullableInt64(value *uint64) (sql.NullInt64, error) {
	if value == nil {
		return sql.NullInt64{}, nil
	}
	if *value > math.MaxInt64 {
		return sql.NullInt64{}, fmt.Errorf("system config override user id exceeds database range")
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}, nil
}

func uint64FromNullInt64(value sql.NullInt64) *uint64 {
	if !value.Valid || value.Int64 < 0 {
		return nil
	}
	converted := uint64(value.Int64)
	return &converted
}
