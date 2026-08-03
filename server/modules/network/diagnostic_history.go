package network

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

const (
	maxDiagnosticHistoryLimit        = 100
	maxDiagnosticHistoryMessageRunes = 512
)

// DiagnosticHistoryStore 持久化并按目标读取经净化的出站诊断结果。
type DiagnosticHistoryStore interface {
	Append(context.Context, string, moduleapi.OutboundDiagnosticResult) error
	List(context.Context, string, int) ([]moduleapi.OutboundDiagnosticResult, error)
}

// SQLDiagnosticHistoryStore 使用 Network 模块自有表保存受限诊断历史。
type SQLDiagnosticHistoryStore struct{ db *sql.DB }

// NewSQLDiagnosticHistoryStore 创建诊断历史仓储；nil 数据库连接会被拒绝。
func NewSQLDiagnosticHistoryStore(db *sql.DB) (*SQLDiagnosticHistoryStore, error) {
	if db == nil {
		return nil, errors.New("outbound diagnostic history store requires a non-nil sql db")
	}
	return &SQLDiagnosticHistoryStore{db: db}, nil
}

// Append 保存单次已净化的诊断结果，不保存目标 URL、代理地址、凭据或响应体。
func (s *SQLDiagnosticHistoryStore) Append(ctx context.Context, targetID string, result moduleapi.OutboundDiagnosticResult) error {
	if s == nil || s.db == nil {
		return errors.New("outbound diagnostic history store is unavailable")
	}
	targetID = strings.TrimSpace(targetID)
	if targetID == "" || len(targetID) > 128 || result.TestedAt.IsZero() {
		return errors.New("outbound diagnostic history entry is invalid")
	}
	var httpStatus any
	if result.HTTPStatus > 0 {
		httpStatus = result.HTTPStatus
	}
	var message any
	if result.Message != "" {
		message = truncateDiagnosticHistoryMessage(result.Message)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO platform_network_diagnostic_history
		(target_id, connected, latency_ms, http_status, error_message, tested_at)
		VALUES ($1, $2, $3, $4, $5, $6)`, targetID, result.Connected, result.Latency.Milliseconds(), httpStatus, message, result.TestedAt.UTC())
	if err != nil {
		return fmt.Errorf("append outbound diagnostic history: %w", err)
	}
	return nil
}

func truncateDiagnosticHistoryMessage(value string) string {
	// 数据库列按字符数限制，使用 rune 截断可避免把多字节 UTF-8 字符切成非法序列。
	runes := []rune(value)
	if len(runes) <= maxDiagnosticHistoryMessageRunes {
		return value
	}
	return string(runes[:maxDiagnosticHistoryMessageRunes])
}

// List 返回目标的最近诊断历史，按完成时间与主键倒序稳定排列。
func (s *SQLDiagnosticHistoryStore) List(ctx context.Context, targetID string, limit int) ([]moduleapi.OutboundDiagnosticResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("outbound diagnostic history store is unavailable")
	}
	var err error
	targetID, err = validDiagnosticHistoryQuery(targetID, limit)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT connected, latency_ms, http_status, error_message, tested_at
		FROM platform_network_diagnostic_history
		WHERE target_id = $1
		ORDER BY tested_at DESC, id DESC
		LIMIT $2`, targetID, limit)
	if err != nil {
		return nil, fmt.Errorf("list outbound diagnostic history: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]moduleapi.OutboundDiagnosticResult, 0)
	for rows.Next() {
		item, err := scanDiagnosticHistoryResult(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbound diagnostic history: %w", err)
	}
	return items, nil
}

func validDiagnosticHistoryQuery(targetID string, limit int) (string, error) {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" || len(targetID) > 128 || limit < 1 || limit > maxDiagnosticHistoryLimit {
		return "", errors.New("outbound diagnostic history query is invalid")
	}
	return targetID, nil
}

type diagnosticHistoryRow interface{ Scan(...any) error }

func scanDiagnosticHistoryResult(row diagnosticHistoryRow) (moduleapi.OutboundDiagnosticResult, error) {
	var item moduleapi.OutboundDiagnosticResult
	var latencyMS int64
	var httpStatus sql.NullInt32
	var message sql.NullString
	if err := row.Scan(&item.Connected, &latencyMS, &httpStatus, &message, &item.TestedAt); err != nil {
		return moduleapi.OutboundDiagnosticResult{}, fmt.Errorf("scan outbound diagnostic history: %w", err)
	}
	item.Latency = time.Duration(latencyMS) * time.Millisecond
	if httpStatus.Valid {
		item.HTTPStatus = int(httpStatus.Int32)
	}
	if message.Valid {
		item.Message = message.String
	}
	item.TestedAt = item.TestedAt.UTC()
	return item, nil
}
