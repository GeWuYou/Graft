package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"graft/server/internal/moduleapi"
)

const (
	maxNameLength       = 120
	maxSurfaceKeyLength = 120
	maxColumnKeyLength  = 120
	maxPageSize         = 500
)

// SQLRepository 将保存视图持久化到模块自有表，并在查询中隔离用户、消费界面和软删除状态。
type SQLRepository struct{ db *sql.DB }

// NewSQLRepository 创建一个由平台数据库连接支持的保存视图仓储。
// 如果 db 为 nil，则返回错误。
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("saved view repository requires a non-nil sql db")
	}
	return &SQLRepository{db: db}, nil
}

// List 返回指定用户和消费界面的未删除视图，并按最近更新时间倒序、ID 倒序保持稳定顺序。
func (r *SQLRepository) List(ctx context.Context, ownerUserID uint64, surfaceKey string) ([]moduleapi.SavedView, error) {
	if err := validateOwnerAndSurface(ownerUserID, surfaceKey); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, owner_user_id, surface_key, name, query_state_json, page_size, visible_columns_json, is_default, created_at, updated_at
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

// Create 新增一个未删除视图；唯一约束冲突转换为 ErrSavedViewConflict。
func (r *SQLRepository) Create(ctx context.Context, input moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	input, err := normalizeCreate(input)
	if err != nil {
		return moduleapi.SavedView{}, err
	}
	columnsJSON, err := json.Marshal(input.VisibleColumns)
	if err != nil {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return moduleapi.SavedView{}, fmt.Errorf("begin create saved view transaction: %w", err)
	}
	defer rollback(tx)
	if input.IsDefault {
		if err := clearDefault(ctx, tx, input.OwnerUserID, input.SurfaceKey, 0); err != nil {
			return moduleapi.SavedView{}, err
		}
	}
	var item moduleapi.SavedView
	err = tx.QueryRowContext(ctx, `INSERT INTO saved_views (owner_user_id, surface_key, name, query_state_json, page_size, visible_columns_json, is_default, created_by, updated_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$1,$1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		RETURNING id, owner_user_id, surface_key, name, query_state_json, page_size, visible_columns_json, is_default, created_at, updated_at`, input.OwnerUserID, input.SurfaceKey, input.Name, input.QueryState, input.PageSize, columnsJSON, input.IsDefault).Scan(
		&item.ID, &item.OwnerUserID, &item.SurfaceKey, &item.Name, &item.QueryState, &item.PageSize, columnScanner(&item.VisibleColumns), &item.IsDefault, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return moduleapi.SavedView{}, mapWriteError(err)
	}
	if err := tx.Commit(); err != nil {
		return moduleapi.SavedView{}, fmt.Errorf("commit create saved view: %w", err)
	}
	return item, nil
}

// Update 替换一个未删除视图的用户可控状态，并在所有权或消费界面不匹配时返回 ErrSavedViewNotFound。
func (r *SQLRepository) Update(ctx context.Context, input moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	input, err := normalizeUpdate(input)
	if err != nil {
		return moduleapi.SavedView{}, err
	}
	columnsJSON, err := json.Marshal(input.VisibleColumns)
	if err != nil {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return moduleapi.SavedView{}, fmt.Errorf("begin update saved view transaction: %w", err)
	}
	defer rollback(tx)
	if input.IsDefault {
		if err := clearDefault(ctx, tx, input.OwnerUserID, input.SurfaceKey, input.ID); err != nil {
			return moduleapi.SavedView{}, err
		}
	}
	var item moduleapi.SavedView
	err = tx.QueryRowContext(ctx, `UPDATE saved_views SET name=$1, query_state_json=$2, page_size=$3, visible_columns_json=$4, is_default=$5, updated_at=CURRENT_TIMESTAMP, updated_by=$6
		WHERE id=$7 AND owner_user_id=$6 AND surface_key=$8 AND deleted_at=0
		RETURNING id, owner_user_id, surface_key, name, query_state_json, page_size, visible_columns_json, is_default, created_at, updated_at`, input.Name, input.QueryState, input.PageSize, columnsJSON, input.IsDefault, input.OwnerUserID, input.ID, input.SurfaceKey).Scan(
		&item.ID, &item.OwnerUserID, &item.SurfaceKey, &item.Name, &item.QueryState, &item.PageSize, columnScanner(&item.VisibleColumns), &item.IsDefault, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return moduleapi.SavedView{}, mapReadError(err)
	}
	if err := tx.Commit(); err != nil {
		return moduleapi.SavedView{}, fmt.Errorf("commit update saved view: %w", err)
	}
	return item, nil
}

// clearDefault 在同一写事务内撤销当前用户和列表页面的旧默认视图，确保后续写入可以安全设为默认。
func clearDefault(ctx context.Context, tx *sql.Tx, ownerUserID uint64, surfaceKey string, exceptID uint64) error {
	_, err := tx.ExecContext(ctx, `UPDATE saved_views SET is_default=FALSE, updated_at=CURRENT_TIMESTAMP, updated_by=$1
		WHERE owner_user_id=$1 AND surface_key=$2 AND deleted_at=0 AND is_default=TRUE AND id<>$3`, ownerUserID, surfaceKey, exceptID)
	if err != nil {
		return fmt.Errorf("clear saved view default: %w", err)
	}
	return nil
}

// rollback 仅在提前返回时回滚未提交事务，保留原始错误作为调用方的失败原因。
func rollback(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

// Delete 软删除一个归属指定用户和消费界面的视图；重复删除和越权访问均表现为 ErrSavedViewNotFound。
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

// normalizeCreate 裁剪并校验保存视图创建输入，包括所有者、消费界面、名称、查询状态、页大小和可见列。
// 校验失败时返回 moduleapi.ErrSavedViewInvalidInput，避免把不完整状态写入仓储。
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

// normalizeUpdate 校验并规范化保存视图更新输入；有效时返回规范化输入，否则返回无效输入错误。
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

// validateOwnerAndSurface 校验保存视图的所有者标识和消费界面标识，确保查询范围具有明确归属。
func validateOwnerAndSurface(ownerUserID uint64, surfaceKey string) error {
	if ownerUserID == 0 || len(strings.TrimSpace(surfaceKey)) == 0 || len(strings.TrimSpace(surfaceKey)) > maxSurfaceKeyLength {
		return moduleapi.ErrSavedViewInvalidInput
	}
	return nil
}

// normalizeState 校验并规范化保存视图的查询状态、分页大小和可见列。
// 它会裁剪可见列两端的空白，并拒绝无效查询状态、越界分页大小、空列名、过长列名或重复列名。
// 返回 moduleapi.ErrSavedViewInvalidInput 表示输入无效。
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

// closeRows 在结果集非空时负责关闭数据库行；关闭错误不覆盖已经确定的查询结果错误。
func closeRows(rows *sql.Rows) {
	if rows != nil {
		_ = rows.Close()
	}
}

// scanView 将数据库行扫描为保存视图；扫描失败时返回错误。
func scanView(row rowScanner) (moduleapi.SavedView, error) {
	var item moduleapi.SavedView
	if err := row.Scan(&item.ID, &item.OwnerUserID, &item.SurfaceKey, &item.Name, &item.QueryState, &item.PageSize, columnScanner(&item.VisibleColumns), &item.IsDefault, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return moduleapi.SavedView{}, fmt.Errorf("scan saved view: %w", err)
	}
	return item, nil
}

type columnsValue struct{ target *[]string }

// columnScanner 创建用于将数据库中的可见列 JSON 扫描到目标字符串切片的值。
func columnScanner(target *[]string) *columnsValue { return &columnsValue{target: target} }
func (v *columnsValue) Scan(value any) error {
	switch raw := value.(type) {
	case []byte:
		return json.Unmarshal(raw, v.target)
	case string:
		return json.Unmarshal([]byte(raw), v.target)
	case nil:
		*v.target = nil
		return nil
	default:
		return errors.New("saved view visible columns must be json")
	}
}

// mapWriteError 将重复或唯一约束错误映射为保存视图冲突错误，其它错误保留创建上下文。
func mapWriteError(err error) error {
	if isUniqueViolation(err) {
		return moduleapi.ErrSavedViewConflict
	}
	return fmt.Errorf("create saved view: %w", err)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return isSQLiteUniqueViolation(err)
}

// mapReadError 将缺失行映射为保存视图未找到错误，其它错误保留读取操作上下文。
func mapReadError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.ErrSavedViewNotFound
	}
	return mapWriteError(err)
}
