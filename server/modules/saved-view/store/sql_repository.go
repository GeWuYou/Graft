package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

const (
	maxNameLength       = 120
	maxSurfaceKeyLength = 120
	maxColumnKeyLength  = 120
	maxPageSize         = 500
)

// SQLRepository persists saved views in the module-owned table.
type SQLRepository struct{ db *sql.DB }

// NewSQLRepository constructs a repository backed by the platform database pool.
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("saved view repository requires a non-nil sql db")
	}
	return &SQLRepository{db: db}, nil
}

// List returns live views for one owner and one surface.
func (r *SQLRepository) List(ctx context.Context, ownerUserID uint64, surfaceKey string) ([]moduleapi.SavedView, error) {
	if err := validateOwnerAndSurface(ownerUserID, surfaceKey); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, owner_user_id, surface_key, name, query_state_json, page_size, visible_columns_json, created_at, updated_at
		FROM saved_views WHERE owner_user_id = $1 AND surface_key = $2 AND deleted_at = 0 ORDER BY updated_at DESC, id DESC`, ownerUserID, strings.TrimSpace(surfaceKey))
	if err != nil {
		return nil, fmt.Errorf("list saved views: %w", err)
	}
	defer closeRows(rows)
	items := make([]moduleapi.SavedView, 0)
	for rows.Next() {
		item, scanErr := scanView(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate saved views: %w", err)
	}
	return items, nil
}

// Create adds one live view.
func (r *SQLRepository) Create(ctx context.Context, input moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	input, err := normalizeCreate(input)
	if err != nil {
		return moduleapi.SavedView{}, err
	}
	columnsJSON, err := json.Marshal(input.VisibleColumns)
	if err != nil {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	var item moduleapi.SavedView
	err = r.db.QueryRowContext(ctx, `INSERT INTO saved_views (owner_user_id, surface_key, name, query_state_json, page_size, visible_columns_json, created_by, updated_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$1,$1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		RETURNING id, owner_user_id, surface_key, name, query_state_json, page_size, visible_columns_json, created_at, updated_at`, input.OwnerUserID, input.SurfaceKey, input.Name, input.QueryState, input.PageSize, columnsJSON).Scan(
		&item.ID, &item.OwnerUserID, &item.SurfaceKey, &item.Name, &item.QueryState, &item.PageSize, columnScanner(&item.VisibleColumns), &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return moduleapi.SavedView{}, mapWriteError(err)
	}
	return item, nil
}

// Update replaces the persisted user-controlled state of one live view.
func (r *SQLRepository) Update(ctx context.Context, input moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	input, err := normalizeUpdate(input)
	if err != nil {
		return moduleapi.SavedView{}, err
	}
	columnsJSON, err := json.Marshal(input.VisibleColumns)
	if err != nil {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	var item moduleapi.SavedView
	err = r.db.QueryRowContext(ctx, `UPDATE saved_views SET name=$1, query_state_json=$2, page_size=$3, visible_columns_json=$4, updated_at=CURRENT_TIMESTAMP, updated_by=$5
		WHERE id=$6 AND owner_user_id=$5 AND surface_key=$7 AND deleted_at=0
		RETURNING id, owner_user_id, surface_key, name, query_state_json, page_size, visible_columns_json, created_at, updated_at`, input.Name, input.QueryState, input.PageSize, columnsJSON, input.OwnerUserID, input.ID, input.SurfaceKey).Scan(
		&item.ID, &item.OwnerUserID, &item.SurfaceKey, &item.Name, &item.QueryState, &item.PageSize, columnScanner(&item.VisibleColumns), &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return moduleapi.SavedView{}, mapReadError(err)
	}
	return item, nil
}

// Delete soft-deletes one owned view.
func (r *SQLRepository) Delete(ctx context.Context, ownerUserID uint64, surfaceKey string, id uint64) error {
	if err := validateOwnerAndSurface(ownerUserID, surfaceKey); err != nil || id == 0 {
		return moduleapi.ErrSavedViewInvalidInput
	}
	result, err := r.db.ExecContext(ctx, `UPDATE saved_views SET deleted_at=$1, deleted_by=$2, updated_at=CURRENT_TIMESTAMP, updated_by=$2 WHERE id=$3 AND owner_user_id=$2 AND surface_key=$4 AND deleted_at=0`, time.Now().UTC().UnixMilli(), ownerUserID, id, strings.TrimSpace(surfaceKey))
	if err != nil {
		return fmt.Errorf("delete saved view: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted saved view: %w", err)
	}
	if affected == 0 {
		return moduleapi.ErrSavedViewNotFound
	}
	return nil
}

func normalizeCreate(input moduleapi.SavedViewCreateInput) (moduleapi.SavedViewCreateInput, error) {
	if err := validateOwnerAndSurface(input.OwnerUserID, input.SurfaceKey); err != nil {
		return input, err
	}
	input.SurfaceKey = strings.TrimSpace(input.SurfaceKey)
	input.Name = strings.TrimSpace(input.Name)
	if len(input.Name) == 0 || len(input.Name) > maxNameLength {
		return input, moduleapi.ErrSavedViewInvalidInput
	}
	if err := normalizeState(&input.QueryState, &input.PageSize, &input.VisibleColumns); err != nil {
		return input, err
	}
	return input, nil
}

func normalizeUpdate(input moduleapi.SavedViewUpdateInput) (moduleapi.SavedViewUpdateInput, error) {
	if input.ID == 0 {
		return input, moduleapi.ErrSavedViewInvalidInput
	}
	created, err := normalizeCreate(moduleapi.SavedViewCreateInput{OwnerUserID: input.OwnerUserID, SurfaceKey: input.SurfaceKey, Name: input.Name, QueryState: input.QueryState, PageSize: input.PageSize, VisibleColumns: input.VisibleColumns})
	if err != nil {
		return input, err
	}
	input.SurfaceKey, input.Name, input.QueryState, input.PageSize, input.VisibleColumns = created.SurfaceKey, created.Name, created.QueryState, created.PageSize, created.VisibleColumns
	return input, nil
}

func validateOwnerAndSurface(ownerUserID uint64, surfaceKey string) error {
	if ownerUserID == 0 || len(strings.TrimSpace(surfaceKey)) == 0 || len(strings.TrimSpace(surfaceKey)) > maxSurfaceKeyLength {
		return moduleapi.ErrSavedViewInvalidInput
	}
	return nil
}

func normalizeState(queryState *json.RawMessage, pageSize *int, columns *[]string) error {
	if *pageSize < 1 || *pageSize > maxPageSize || !json.Valid(*queryState) {
		return moduleapi.ErrSavedViewInvalidInput
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(*queryState, &object); err != nil || object == nil {
		return moduleapi.ErrSavedViewInvalidInput
	}
	seen := make(map[string]struct{}, len(*columns))
	for index, value := range *columns {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > maxColumnKeyLength {
			return moduleapi.ErrSavedViewInvalidInput
		}
		if _, exists := seen[value]; exists {
			return moduleapi.ErrSavedViewInvalidInput
		}
		seen[value] = struct{}{}
		(*columns)[index] = value
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func closeRows(rows *sql.Rows) {
	if rows != nil {
		_ = rows.Close()
	}
}

func scanView(row rowScanner) (moduleapi.SavedView, error) {
	var item moduleapi.SavedView
	if err := row.Scan(&item.ID, &item.OwnerUserID, &item.SurfaceKey, &item.Name, &item.QueryState, &item.PageSize, columnScanner(&item.VisibleColumns), &item.CreatedAt, &item.UpdatedAt); err != nil {
		return moduleapi.SavedView{}, fmt.Errorf("scan saved view: %w", err)
	}
	return item, nil
}

type columnsValue struct{ target *[]string }

func columnScanner(target *[]string) *columnsValue { return &columnsValue{target: target} }
func (v *columnsValue) Scan(value any) error {
	raw, ok := value.([]byte)
	if !ok {
		return errors.New("saved view visible columns must be json")
	}
	return json.Unmarshal(raw, v.target)
}
func mapWriteError(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
		return moduleapi.ErrSavedViewConflict
	}
	return fmt.Errorf("create saved view: %w", err)
}
func mapReadError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.ErrSavedViewNotFound
	}
	return mapWriteError(err)
}
