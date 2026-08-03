// Package storeent 提供 system-config 用户覆盖值的 SQL 持久化实现。
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

// NewRepository 创建 SQL 覆盖值仓储；数据库连接为空时返回错误，避免后续请求路径出现隐式失效。
func NewRepository(db *sql.DB) (systemconfigstore.Repository, error) {
	if db == nil {
		return nil, errors.New("system config repository requires a non-nil sql db")
	}
	return &repository{db: db}, nil
}

func (r *repository) ListOverrides(ctx context.Context) (overrides []systemconfigstore.Override, err error) {
	if r == nil || r.db == nil {
		return nil, errors.New("system config repository is unavailable")
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT key, override_value, version, created_at, created_by, updated_at, updated_by
		 FROM system_config_values`,
	)
	if err != nil {
		return nil, fmt.Errorf("list system config overrides: %w", err)
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("list system config overrides: close rows: %w", closeErr)
		}
	}()

	overrides = make([]systemconfigstore.Override, 0)
	for rows.Next() {
		override, scanErr := scanOverride(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("list system config overrides: %w", scanErr)
		}
		overrides = append(overrides, override)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list system config overrides: %w", err)
	}
	return overrides, nil
}

func (r *repository) GetOverride(ctx context.Context, key string) (systemconfigstore.Override, error) {
	if r == nil || r.db == nil {
		return systemconfigstore.Override{}, errors.New("system config repository is unavailable")
	}

	row := r.db.QueryRowContext(
		ctx,
		`SELECT key, override_value, version, created_at, created_by, updated_at, updated_by
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

func (r *repository) CompareAndSwapOverride(ctx context.Context, key string, value json.RawMessage, userID *uint64, expectedVersion int64) (systemconfigstore.Override, error) {
	if r == nil || r.db == nil {
		return systemconfigstore.Override{}, errors.New("system config repository is unavailable")
	}
	userIDValue, err := nullableInt64(userID)
	if err != nil {
		return systemconfigstore.Override{}, fmt.Errorf("compare and swap system config override: %w", err)
	}
	if expectedVersion < 0 {
		return systemconfigstore.Override{}, fmt.Errorf("compare and swap system config override: invalid expected version %d", expectedVersion)
	}

	var row *sql.Row
	if expectedVersion == 0 {
		row = r.db.QueryRowContext(
			ctx,
			`INSERT INTO system_config_values (key, override_value, version, created_at, created_by, updated_at, updated_by)
			 VALUES ($1, $2, 1, NOW(), $3, NOW(), $3)
			 ON CONFLICT (key) DO NOTHING
			 RETURNING key, override_value, version, created_at, created_by, updated_at, updated_by`,
			strings.TrimSpace(key), value, userIDValue,
		)
	} else {
		row = r.db.QueryRowContext(
			ctx,
			`UPDATE system_config_values
			 SET override_value = $2, version = version + 1, updated_at = NOW(), updated_by = $3
			 WHERE key = $1 AND version = $4
			 RETURNING key, override_value, version, created_at, created_by, updated_at, updated_by`,
			strings.TrimSpace(key), value, userIDValue, expectedVersion,
		)
	}
	override, err := scanOverride(row)
	if errors.Is(err, sql.ErrNoRows) {
		return systemconfigstore.Override{}, systemconfigstore.ErrVersionConflict
	}
	if err != nil {
		return systemconfigstore.Override{}, fmt.Errorf("compare and swap system config override: %w", err)
	}
	return override, nil
}

func (r *repository) ResetOverride(ctx context.Context, key string, userID *uint64, expectedVersion int64) (systemconfigstore.Override, error) {
	if r == nil || r.db == nil {
		return systemconfigstore.Override{}, errors.New("system config repository is unavailable")
	}
	userIDValue, err := nullableInt64(userID)
	if err != nil {
		return systemconfigstore.Override{}, fmt.Errorf("reset system config override: %w", err)
	}
	if expectedVersion < 0 {
		return systemconfigstore.Override{}, fmt.Errorf("reset system config override: invalid expected version %d", expectedVersion)
	}
	var row *sql.Row
	if expectedVersion == 0 {
		row = r.db.QueryRowContext(ctx,
			`INSERT INTO system_config_values (key, override_value, version, created_at, created_by, updated_at, updated_by)
			 VALUES ($1, NULL, 1, NOW(), $2, NOW(), $2)
			 ON CONFLICT (key) DO NOTHING
			 RETURNING key, override_value, version, created_at, created_by, updated_at, updated_by`,
			strings.TrimSpace(key), userIDValue)
	} else {
		row = r.db.QueryRowContext(ctx,
			`UPDATE system_config_values
			 SET override_value = NULL, version = version + 1, updated_at = NOW(), updated_by = $2
			 WHERE key = $1 AND version = $3
			 RETURNING key, override_value, version, created_at, created_by, updated_at, updated_by`,
			strings.TrimSpace(key), userIDValue, expectedVersion)
	}
	override, err := scanOverride(row)
	if errors.Is(err, sql.ErrNoRows) {
		return systemconfigstore.Override{}, systemconfigstore.ErrVersionConflict
	}
	if err != nil {
		return systemconfigstore.Override{}, fmt.Errorf("reset system config override: %w", err)
	}
	return override, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOverride(row rowScanner) (systemconfigstore.Override, error) {
	var override systemconfigstore.Override
	var value []byte
	var createdAt time.Time
	var createdBy sql.NullInt64
	var updatedAt time.Time
	var updatedBy sql.NullInt64
	if err := row.Scan(&override.Key, &value, &override.Version, &createdAt, &createdBy, &updatedAt, &updatedBy); err != nil {
		return systemconfigstore.Override{}, err
	}
	override.Value = append(json.RawMessage(nil), value...)
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
