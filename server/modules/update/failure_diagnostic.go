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
	runnerFailureDiagnosticSummary = "platform update runner reported a terminal failure"
	maxFailureDiagnosticDetailSize = 32 * 1024
	runnerFailureStageReceipt      = "runner_receipt"
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
	GetFailureDiagnosticByOperation(context.Context, string) (FailureDiagnostic, error)
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
	return scanFailureDiagnostic(s.db.QueryRowContext(ctx, `SELECT request_id, operation_id, task_id, target_version, failure_code,
 failure_stage, summary, detail, occurred_at FROM update_failure_diagnostics WHERE request_id = $1`, strings.TrimSpace(requestID)))
}

// GetFailureDiagnosticByOperation 返回 runner 回执终态关联的受控诊断，避免调用方猜测原始 HTTP request ID。
func (s *sqlFailureDiagnosticStore) GetFailureDiagnosticByOperation(ctx context.Context, operationID string) (FailureDiagnostic, error) {
	if s == nil || s.db == nil || !runnerOperationID.MatchString(operationID) {
		return FailureDiagnostic{}, errors.New("update operation identity is invalid")
	}
	return scanFailureDiagnostic(s.db.QueryRowContext(ctx, `SELECT request_id, operation_id, task_id, target_version, failure_code,
 failure_stage, summary, detail, occurred_at FROM update_failure_diagnostics WHERE operation_id = $1`, operationID))
}

func scanFailureDiagnostic(row interface{ Scan(...any) error }) (FailureDiagnostic, error) {
	var value FailureDiagnostic
	var operationID sql.NullString
	var taskID sql.NullInt64
	err := row.Scan(&value.RequestID, &operationID, &taskID, &value.TargetVersion, &value.FailureCode, &value.FailureStage,
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

func runnerTerminalFailureDiagnostic(operation ComposeUpdateOperation, receipt RunnerReceipt) FailureDiagnostic {
	stage, detail := runnerReceiptFailureDiagnostic(receipt)
	if receipt.MigrationStarted {
		detail += "; migration had started and operator attention is required"
	}
	return FailureDiagnostic{
		RequestID:     operation.RequestID,
		OperationID:   operation.OperationID,
		TaskID:        operation.TaskID,
		TargetVersion: operation.TargetVersion,
		FailureCode:   rolloutFailureRunnerTerminal,
		FailureStage:  stage,
		Summary:       runnerFailureDiagnosticSummary,
		Detail:        detail,
		OccurredAt:    time.Now().UTC(),
	}
}

// runnerReceiptFailureDiagnostic 只接受 runner 的固定无秘密备份事实，未知字段降级为通用诊断，避免透传 runner 输出。
func runnerReceiptFailureDiagnostic(receipt RunnerReceipt) (string, string) {
	stage := runnerFailureStageReceipt
	if receipt.FailureCode == runnerFailureBackup {
		switch strings.TrimSpace(receipt.FailureStage) {
		case string(RunnerBackupFailureStageArtifactDirectory), string(RunnerBackupFailureStageConfigSnapshot), string(RunnerBackupFailureStageDatabaseDump), string(RunnerBackupFailureStageArtifactDigest):
			stage = strings.TrimSpace(receipt.FailureStage)
		}
	}

	detail := "runner reported a terminal failure"
	if receipt.FailureCode == runnerFailureBackup {
		detail = backupFailureDiagnosticDetail(stage, receipt.FailureDetail)
	} else if code := strings.TrimSpace(receipt.FailureCode); code != "" {
		detail += " (" + code + ")"
	}
	return stage, detail
}

func backupFailureDiagnosticDetail(stage, reason string) string {
	step := "backup step"
	switch stage {
	case string(RunnerBackupFailureStageArtifactDirectory):
		step = "backup artifact directory creation"
	case string(RunnerBackupFailureStageConfigSnapshot):
		step = "deployment environment snapshot"
	case string(RunnerBackupFailureStageDatabaseDump):
		step = "PostgreSQL dump"
	case string(RunnerBackupFailureStageArtifactDigest):
		step = "backup artifact digest"
	}
	switch strings.TrimSpace(reason) {
	case "permission_denied":
		return step + " was denied by deployment filesystem permissions"
	case "command_failed":
		return step + " failed"
	case "io_failed":
		return step + " could not access required backup artifacts"
	default:
		return step + " failed"
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
