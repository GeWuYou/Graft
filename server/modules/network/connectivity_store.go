package network

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

const maxConnectivityHistoryLimit = 100

// ConnectivityCheck 是批量和单目标执行共享的轻量查询投影。
type ConnectivityCheck struct {
	ID        int64
	TargetID  moduleapi.ConnectivityTargetID
	Status    moduleapi.ConnectivityReportStatus
	Latency   time.Duration
	CheckedAt time.Time
}

// ConnectivityAggregate 是最近一轮已完成检查的有界健康摘要。
type ConnectivityAggregate struct {
	LastRunAt      *time.Time
	TargetCount    int
	HealthyCount   int
	DegradedCount  int
	FailedCount    int
	AverageLatency time.Duration
	WorstTargetID  moduleapi.ConnectivityTargetID
	WorstLatency   time.Duration
}

// ConnectivityStore 是 Network 模块唯一的检查、报告与历史持久化边界。
type ConnectivityStore interface {
	Append(context.Context, moduleapi.ConnectivityReport) (ConnectivityCheck, error)
	Latest(context.Context) ([]ConnectivityCheck, error)
	History(context.Context, moduleapi.ConnectivityTargetID, int) ([]ConnectivityCheck, error)
	Report(context.Context, moduleapi.ConnectivityTargetID, int64) (moduleapi.ConnectivityReport, error)
	Aggregate(context.Context) (ConnectivityAggregate, error)
}

// SQLConnectivityStore 使用 Network 自有表保存已净化的可扩展报告。
type SQLConnectivityStore struct{ db *sql.DB }

// NewSQLConnectivityStore 创建 Network 连通性持久化仓储。
func NewSQLConnectivityStore(db *sql.DB) (*SQLConnectivityStore, error) {
	if db == nil {
		return nil, errors.New("connectivity store requires a non-nil sql db")
	}
	return &SQLConnectivityStore{db: db}, nil
}

// Append 写入一个检查和其结构化净化报告，并在同一事务中执行每目标的有界保留。
func (s *SQLConnectivityStore) Append(ctx context.Context, report moduleapi.ConnectivityReport) (ConnectivityCheck, error) {
	if s == nil || s.db == nil {
		return ConnectivityCheck{}, errors.New("connectivity store is unavailable")
	}
	report = report.Snapshot()
	if err := validateConnectivityReport(report); err != nil {
		return ConnectivityCheck{}, err
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return ConnectivityCheck{}, fmt.Errorf("encode connectivity report: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ConnectivityCheck{}, fmt.Errorf("begin connectivity append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var check ConnectivityCheck
	check.TargetID, check.Status, check.Latency, check.CheckedAt = report.TargetID, report.Status, report.TotalLatency, report.CheckedAt
	err = tx.QueryRowContext(ctx, `INSERT INTO platform_connectivity_checks (target_id, status, latency_ms, checked_at)
		VALUES ($1, $2, $3, $4) RETURNING id`, string(report.TargetID), string(report.Status), report.TotalLatency.Milliseconds(), report.CheckedAt).Scan(&check.ID)
	if err != nil {
		return ConnectivityCheck{}, fmt.Errorf("append connectivity check: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO platform_connectivity_reports (check_id, schema_version, report) VALUES ($1, $2, $3)`, check.ID, report.SchemaVersion, payload); err != nil {
		return ConnectivityCheck{}, fmt.Errorf("append connectivity report: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM platform_connectivity_checks WHERE id IN (
		SELECT id FROM platform_connectivity_checks WHERE target_id = $1 ORDER BY checked_at DESC, id DESC OFFSET $2
	)`, string(report.TargetID), maxConnectivityHistoryLimit); err != nil {
		return ConnectivityCheck{}, fmt.Errorf("retain connectivity history: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return ConnectivityCheck{}, fmt.Errorf("commit connectivity append: %w", err)
	}
	return check, nil
}

// Latest 返回每个目标最近一次检查，按目标稳定排序。
func (s *SQLConnectivityStore) Latest(ctx context.Context) ([]ConnectivityCheck, error) {
	return s.list(ctx, `SELECT DISTINCT ON (target_id) id, target_id, status, latency_ms, checked_at
		FROM platform_connectivity_checks ORDER BY target_id, checked_at DESC, id DESC`, nil)
}

// History 返回一个目标最近的有界检查摘要。
func (s *SQLConnectivityStore) History(ctx context.Context, targetID moduleapi.ConnectivityTargetID, limit int) ([]ConnectivityCheck, error) {
	if strings.TrimSpace(string(targetID)) == "" || limit < 1 || limit > maxConnectivityHistoryLimit {
		return nil, errors.New("connectivity history query is invalid")
	}
	return s.list(ctx, `SELECT id, target_id, status, latency_ms, checked_at FROM platform_connectivity_checks
		WHERE target_id = $1 ORDER BY checked_at DESC, id DESC LIMIT $2`, []any{string(targetID), limit})
}

func (s *SQLConnectivityStore) list(ctx context.Context, query string, args []any) ([]ConnectivityCheck, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("connectivity store is unavailable")
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query connectivity checks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := []ConnectivityCheck{}
	for rows.Next() {
		var item ConnectivityCheck
		var targetID, status string
		var latencyMS int64
		if err := rows.Scan(&item.ID, &targetID, &status, &latencyMS, &item.CheckedAt); err != nil {
			return nil, fmt.Errorf("scan connectivity check: %w", err)
		}
		item.TargetID, item.Status, item.Latency, item.CheckedAt = moduleapi.ConnectivityTargetID(targetID), moduleapi.ConnectivityReportStatus(status), time.Duration(latencyMS)*time.Millisecond, item.CheckedAt.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connectivity checks: %w", err)
	}
	return items, nil
}

// Report 返回一个持久化报告。报告中没有完整出口 IP 或原始网络数据。
func (s *SQLConnectivityStore) Report(ctx context.Context, targetID moduleapi.ConnectivityTargetID, checkID int64) (moduleapi.ConnectivityReport, error) {
	if s == nil || s.db == nil || strings.TrimSpace(string(targetID)) == "" || checkID < 1 {
		return moduleapi.ConnectivityReport{}, errors.New("connectivity report query is invalid")
	}
	var payload []byte
	if err := s.db.QueryRowContext(ctx, `SELECT reports.report FROM platform_connectivity_reports AS reports
		JOIN platform_connectivity_checks AS checks ON checks.id = reports.check_id
		WHERE reports.check_id = $1 AND checks.target_id = $2`, checkID, string(targetID)).Scan(&payload); err != nil {
		return moduleapi.ConnectivityReport{}, fmt.Errorf("read connectivity report: %w", err)
	}
	var report moduleapi.ConnectivityReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return moduleapi.ConnectivityReport{}, fmt.Errorf("decode connectivity report: %w", err)
	}
	return report.Snapshot(), nil
}

// Aggregate 计算所有目标最新状态的实时摘要，不引入缓存和第二份聚合真相。
func (s *SQLConnectivityStore) Aggregate(ctx context.Context) (ConnectivityAggregate, error) {
	latest, err := s.Latest(ctx)
	if err != nil {
		return ConnectivityAggregate{}, err
	}
	result := ConnectivityAggregate{TargetCount: len(latest)}
	for _, item := range latest {
		checked := item.CheckedAt
		if result.LastRunAt == nil || checked.After(*result.LastRunAt) {
			result.LastRunAt = &checked
		}
		switch item.Status {
		case moduleapi.ConnectivityReportStatusHealthy:
			result.HealthyCount++
		case moduleapi.ConnectivityReportStatusDegraded:
			result.DegradedCount++
		default:
			result.FailedCount++
		}
		result.AverageLatency += item.Latency
		if item.Latency > result.WorstLatency {
			result.WorstLatency, result.WorstTargetID = item.Latency, item.TargetID
		}
	}
	if result.TargetCount > 0 {
		result.AverageLatency /= time.Duration(result.TargetCount)
	}
	return result, nil
}

// CreateCustomTarget 仅在 service 应用 SSRF 输入策略后持久化目标。
func (s *SQLConnectivityStore) CreateCustomTarget(ctx context.Context, target CustomConnectivityTarget, actorID uint64) (CustomConnectivityTarget, error) {
	if s == nil || s.db == nil || target.ID == "" || target.DisplayName == "" || target.Endpoint == "" {
		return CustomConnectivityTarget{}, errors.New("custom connectivity target is invalid")
	}
	var created CustomConnectivityTarget
	err := s.db.QueryRowContext(ctx, `INSERT INTO platform_connectivity_custom_targets (target_id, display_name, endpoint, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $4) RETURNING target_id, display_name, endpoint, enabled, created_at`, string(target.ID), target.DisplayName, target.Endpoint, actorID).
		Scan(&created.ID, &created.DisplayName, &created.Endpoint, &created.Enabled, &created.CreatedAt)
	if err != nil {
		return CustomConnectivityTarget{}, fmt.Errorf("create custom connectivity target: %w", err)
	}
	created.CreatedAt = created.CreatedAt.UTC()
	return created, nil
}

// ListCustomTargets 仅返回有效的管理员管理目标元数据。
func (s *SQLConnectivityStore) ListCustomTargets(ctx context.Context) ([]CustomConnectivityTarget, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("connectivity store is unavailable")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT target_id, display_name, endpoint, enabled, created_at FROM platform_connectivity_custom_targets WHERE deleted_at = 0 ORDER BY target_id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list custom connectivity targets: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := []CustomConnectivityTarget{}
	for rows.Next() {
		var target CustomConnectivityTarget
		if err := rows.Scan(&target.ID, &target.DisplayName, &target.Endpoint, &target.Enabled, &target.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan custom connectivity target: %w", err)
		}
		target.CreatedAt = target.CreatedAt.UTC()
		items = append(items, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate custom connectivity targets: %w", err)
	}
	return items, nil
}

// CustomTarget 按稳定目标标识读取一个有效自定义目标。
func (s *SQLConnectivityStore) CustomTarget(ctx context.Context, targetID moduleapi.ConnectivityTargetID) (CustomConnectivityTarget, error) {
	if s == nil || s.db == nil || strings.TrimSpace(string(targetID)) == "" {
		return CustomConnectivityTarget{}, errors.New("custom connectivity target query is invalid")
	}
	var target CustomConnectivityTarget
	err := s.db.QueryRowContext(ctx, `SELECT target_id, display_name, endpoint, enabled, created_at FROM platform_connectivity_custom_targets WHERE target_id = $1 AND deleted_at = 0`, string(targetID)).Scan(&target.ID, &target.DisplayName, &target.Endpoint, &target.Enabled, &target.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return CustomConnectivityTarget{}, errCustomConnectivityTargetNotFound
	}
	if err != nil {
		return CustomConnectivityTarget{}, fmt.Errorf("read custom connectivity target: %w", err)
	}
	target.CreatedAt = target.CreatedAt.UTC()
	return target, nil
}

// DeleteCustomTarget 软删除目标，使报告历史仍可归属其稳定标识。
func (s *SQLConnectivityStore) DeleteCustomTarget(ctx context.Context, targetID moduleapi.ConnectivityTargetID, actorID uint64) error {
	if s == nil || s.db == nil || strings.TrimSpace(string(targetID)) == "" {
		return errors.New("custom connectivity target delete is invalid")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE platform_connectivity_custom_targets SET deleted_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT, deleted_by = $2, updated_at = CURRENT_TIMESTAMP, updated_by = $2 WHERE target_id = $1 AND deleted_at = 0`, string(targetID), actorID)
	if err != nil {
		return fmt.Errorf("delete custom connectivity target: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read custom connectivity target delete result: %w", err)
	}
	if changed == 0 {
		return errCustomConnectivityTargetNotFound
	}
	return nil
}

func validateConnectivityReport(report moduleapi.ConnectivityReport) error {
	if strings.TrimSpace(string(report.TargetID)) == "" || report.SchemaVersion < 1 || report.CheckedAt.IsZero() || report.TotalLatency < 0 {
		return errors.New("connectivity report is invalid")
	}
	switch report.Status {
	case moduleapi.ConnectivityReportStatusHealthy, moduleapi.ConnectivityReportStatusDegraded, moduleapi.ConnectivityReportStatusFailed:
	default:
		return errors.New("connectivity report status is invalid")
	}
	if report.ExitIP != nil && !strings.ContainsAny(report.ExitIP.Masked, "*•") {
		return errors.New("connectivity report contains unmasked exit IP")
	}
	return nil
}
