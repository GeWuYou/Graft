package update

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	updateFailureDiagnosticSummary = "platform update rollout start failed"
	maxFailureDiagnosticDetailSize = 32 * 1024
)

var errUpdateFailureDiagnosticNotFound = errors.New("update failure diagnostic not found")

// FailureDiagnostic 保存一次更新启动失败的受控排障事实。
type FailureDiagnostic struct {
	RequestID     string    `json:"request_id"`
	OperationID   string    `json:"operation_id,omitempty"`
	TaskID        uint64    `json:"task_id,omitempty"`
	TargetVersion string    `json:"target_version"`
	FailureCode   string    `json:"failure_code"`
	FailureStage  string    `json:"failure_stage"`
	Summary       string    `json:"summary"`
	Detail        string    `json:"detail"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// FailureDiagnosticStore 保存更新启动失败的持久化诊断记录。
type FailureDiagnosticStore interface {
	CreateFailureDiagnostic(context.Context, FailureDiagnostic, uint64) error
	GetFailureDiagnostic(context.Context, string) (FailureDiagnostic, error)
}

type sqlFailureDiagnosticStore struct{ db *sql.DB }

func newSQLFailureDiagnosticStore(db *sql.DB) (FailureDiagnosticStore, error) {
	if db == nil {
		return nil, errors.New("update failure diagnostic database is unavailable")
	}
	return &sqlFailureDiagnosticStore{db: db}, nil
}

func (s *sqlFailureDiagnosticStore) CreateFailureDiagnostic(ctx context.Context, value FailureDiagnostic, requestedBy uint64) error {
	if s == nil || s.db == nil || !validFailureDiagnostic(value) || requestedBy == 0 {
		return errors.New("update failure diagnostic is invalid")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO update_failure_diagnostics
 (request_id, operation_id, task_id, requested_by, target_version, failure_code, failure_stage, summary, detail, occurred_at)
 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
 ON CONFLICT (request_id) DO NOTHING`, value.RequestID, nullableString(value.OperationID), nullableUint64(value.TaskID), requestedBy,
		value.TargetVersion, value.FailureCode, value.FailureStage, value.Summary, value.Detail, value.OccurredAt.UTC())
	if err != nil {
		return fmt.Errorf("create update failure diagnostic: %w", err)
	}
	return nil
}

func (s *sqlFailureDiagnosticStore) GetFailureDiagnostic(ctx context.Context, requestID string) (FailureDiagnostic, error) {
	if s == nil || s.db == nil || strings.TrimSpace(requestID) == "" {
		return FailureDiagnostic{}, errors.New("update failure diagnostic identity is invalid")
	}
	var value FailureDiagnostic
	var operationID sql.NullString
	var taskID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT request_id, operation_id, task_id, target_version, failure_code,
 failure_stage, summary, detail, occurred_at FROM update_failure_diagnostics WHERE request_id = $1`, strings.TrimSpace(requestID)).Scan(
		&value.RequestID, &operationID, &taskID, &value.TargetVersion, &value.FailureCode, &value.FailureStage,
		&value.Summary, &value.Detail, &value.OccurredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return FailureDiagnostic{}, errUpdateFailureDiagnosticNotFound
	}
	if err != nil {
		return FailureDiagnostic{}, fmt.Errorf("get update failure diagnostic: %w", err)
	}
	if operationID.Valid {
		value.OperationID = operationID.String
	}
	if taskID.Valid && taskID.Int64 > 0 {
		value.TaskID = uint64(taskID.Int64)
	}
	value.OccurredAt = value.OccurredAt.UTC()
	return value, nil
}

func newFailureDiagnostic(requestID, operationID, targetVersion, failureCode, failureStage string, err error) FailureDiagnostic {
	return FailureDiagnostic{
		RequestID:     strings.TrimSpace(requestID),
		OperationID:   strings.TrimSpace(operationID),
		TargetVersion: strings.TrimSpace(targetVersion),
		FailureCode:   strings.TrimSpace(failureCode),
		FailureStage:  strings.TrimSpace(failureStage),
		Summary:       updateFailureDiagnosticSummary,
		Detail:        truncateFailureDiagnosticDetail(sanitizeRolloutError(err)),
		OccurredAt:    time.Now().UTC(),
	}
}

func validFailureDiagnostic(value FailureDiagnostic) bool {
	return value.RequestID != "" && value.TargetVersion != "" && value.FailureCode != "" && value.FailureStage != "" && value.Summary != "" && value.Detail != "" && !value.OccurredAt.IsZero()
}

func truncateFailureDiagnosticDetail(value string) string {
	if len(value) <= maxFailureDiagnosticDetailSize {
		return value
	}
	return value[:maxFailureDiagnosticDetailSize] + "\n[TRUNCATED]"
}
