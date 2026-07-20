// Package store 负责 runtime-target 的持久化查询。
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// ErrNotFound 表示查询不到未软删除的运行时目标。
var ErrNotFound = errors.New("runtime target not found")

// LocalDockerProbe 记录一次受限的本机 Docker 探测结果；调用方应保留探测时间和失败原因，供运行时目标状态查询使用。
type LocalDockerProbe struct {
	Endpoint  string
	Available bool
	Error     string
	CheckedAt time.Time
}

// SQLRepository 持久化运行时目标记录，并将已软删除记录排除在公开查询之外。
type SQLRepository struct{ db *sql.DB }

// NewSQLRepository 创建由模块拥有的 SQL 仓储。
func NewSQLRepository(db *sql.DB) *SQLRepository { return &SQLRepository{db: db} }

// Target 是运行时目标的持久化读取投影；Capabilities 是 provider-neutral 的能力集合，供上层筛选可执行能力。
type Target struct {
	ID             uint64     `json:"id"`
	Provider       string     `json:"provider"`
	DisplayName    string     `json:"displayName"`
	EndpointLabel  string     `json:"endpointLabel"`
	ConnectionKind string     `json:"connectionKind"`
	Capabilities   []string   `json:"capabilities"`
	Availability   bool       `json:"availability"`
	LastError      string     `json:"lastError"`
	CheckedAt      *time.Time `json:"checkedAt"`
}

// Page 表示一个稳定的运行时目标分页窗口；Total 与 Items 使用同一份未软删除数据集计算。
type Page struct {
	Items   []Target
	Total   int64
	Summary Summary
}

// Summary 是未软删除运行时目标的全量健康聚合，不受当前分页窗口影响。
type Summary struct {
	Total       int64
	Healthy     int64
	Unavailable int64
}

// List 返回全部未软删除的运行时目标，并按 provider、显示名称和 ID 排序以保证跨请求顺序稳定。
func (r *SQLRepository) List(ctx context.Context) ([]Target, error) {
	if r == nil || r.db == nil {
		return []Target{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE deleted_at = 0 ORDER BY provider, display_name, id`)
	if err != nil {
		return nil, err
	}
	return scanTargets(rows)
}

// ListPage 返回一个未软删除的运行时目标分页窗口及总数；limit 和 offset 由调用方负责先行校验。
func (r *SQLRepository) ListPage(ctx context.Context, limit, offset int) (Page, error) {
	if r == nil || r.db == nil {
		return Page{Items: []Target{}}, nil
	}
	var summary Summary
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE availability = true), COUNT(*) FILTER (WHERE availability = false) FROM runtime_targets WHERE deleted_at = 0`).Scan(&summary.Total, &summary.Healthy, &summary.Unavailable); err != nil {
		return Page{}, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE deleted_at = 0 ORDER BY provider, display_name, id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return Page{}, err
	}
	items, err := scanTargets(rows)
	if err != nil {
		return Page{}, err
	}
	return Page{Items: items, Total: summary.Total, Summary: summary}, nil
}

// scanTargets 读取 runtime-target 列表投影，并始终关闭结果集以释放数据库资源。
func scanTargets(rows *sql.Rows) ([]Target, error) {
	defer func() { _ = rows.Close() }()
	items := []Target{}
	for rows.Next() {
		var item Target
		var capabilities []byte
		if err := rows.Scan(&item.ID, &item.Provider, &item.DisplayName, &item.EndpointLabel, &item.ConnectionKind, &capabilities, &item.Availability, &item.LastError, &item.CheckedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(capabilities, &item.Capabilities); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// FindSystemLocalDocker 返回此前发现的系统托管本机 Docker 记录；未发现时统一返回 ErrNotFound。
func (r *SQLRepository) FindSystemLocalDocker(ctx context.Context) (Target, error) {
	if r == nil || r.db == nil {
		return Target{}, ErrNotFound
	}
	var item Target
	var capabilities []byte
	err := r.db.QueryRowContext(ctx, `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE provider = 'docker' AND endpoint = 'unix:///var/run/docker.sock' AND system_managed = true AND deleted_at = 0`).Scan(&item.ID, &item.Provider, &item.DisplayName, &item.EndpointLabel, &item.ConnectionKind, &capabilities, &item.Availability, &item.LastError, &item.CheckedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, ErrNotFound
	}
	if err != nil {
		return Target{}, err
	}
	if err := json.Unmarshal(capabilities, &item.Capabilities); err != nil {
		return Target{}, err
	}
	return item, nil
}

// Get 按 ID 返回一个未软删除的运行时目标；记录不存在时统一返回 ErrNotFound。
func (r *SQLRepository) Get(ctx context.Context, id uint64) (Target, error) {
	if r == nil || r.db == nil {
		return Target{}, ErrNotFound
	}
	var item Target
	var capabilities []byte
	err := r.db.QueryRowContext(ctx, `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE id = $1 AND deleted_at = 0`, id).Scan(&item.ID, &item.Provider, &item.DisplayName, &item.EndpointLabel, &item.ConnectionKind, &capabilities, &item.Availability, &item.LastError, &item.CheckedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, ErrNotFound
	}
	if err != nil {
		return Target{}, err
	}
	if err := json.Unmarshal(capabilities, &item.Capabilities); err != nil {
		return Target{}, err
	}
	return item, nil
}

// UpsertLocalDocker 写入系统托管的本机 Docker 探测结果，并通过 provider 与 endpoint 的活跃记录唯一键更新已有记录。
func (r *SQLRepository) UpsertLocalDocker(ctx context.Context, probe LocalDockerProbe) error {
	if r == nil || r.db == nil {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO runtime_targets (provider, endpoint, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at, system_managed, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by) VALUES ('docker', $1, 'Local Docker', 'unix:///var/run/docker.sock', 'unix_socket', '["containers","compose_execution","workspace_access"]'::jsonb, $2, $3, $4, true, NOW(), 0, NOW(), 0, 0, 0) ON CONFLICT (provider, endpoint) WHERE deleted_at = 0 DO UPDATE SET capabilities_json = EXCLUDED.capabilities_json, availability = EXCLUDED.availability, last_error = EXCLUDED.last_error, checked_at = EXCLUDED.checked_at, updated_at = NOW(), updated_by = 0`, probe.Endpoint, probe.Available, probe.Error, probe.CheckedAt)
	return err
}
