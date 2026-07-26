package backup

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

// ManualRetention 是用户主动创建 Backup 可选的保留期限。
type ManualRetention string

const (
	// ManualRetentionOneDay 保留用户主动创建 Backup 一天。
	ManualRetentionOneDay ManualRetention = "1d"
	// ManualRetentionSevenDays 保留用户主动创建 Backup 七天。
	ManualRetentionSevenDays ManualRetention = "7d"
	// ManualRetentionThirtyDays 保留用户主动创建 Backup 三十天。
	ManualRetentionThirtyDays ManualRetention = "30d"

	manualRetentionOneDayDuration     = 24 * time.Hour
	manualRetentionSevenDaysDuration  = 7 * manualRetentionOneDayDuration
	manualRetentionThirtyDaysDuration = 30 * manualRetentionOneDayDuration
)

// ManualRetentionDeadline 将公开保留期转换为从提交时刻起算的冻结截止时间。
func ManualRetentionDeadline(retention ManualRetention, now time.Time) (time.Time, error) {
	var duration time.Duration
	switch retention {
	case ManualRetentionOneDay:
		duration = manualRetentionOneDayDuration
	case ManualRetentionSevenDays:
		duration = manualRetentionSevenDaysDuration
	case ManualRetentionThirtyDays:
		duration = manualRetentionThirtyDaysDuration
	default:
		return time.Time{}, moduleapi.ErrBackupInvalidInput
	}
	return now.UTC().Add(duration), nil
}

func manualBackupOperationID(idempotencyKey string, now time.Time) string {
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return fmt.Sprintf("backup-%d", now.UTC().UnixNano())
	}
	digest := sha256.Sum256([]byte(key))
	return fmt.Sprintf("backup-%x", digest[:16])
}

const (
	backupTaskType         = moduleapi.TaskType("platform.backup.create.v1")
	backupArtifactExecutor = moduleapi.StageExecutorType("platform.backup.create-artifacts.v1")
	backupRecordExecutor   = moduleapi.StageExecutorType("platform.backup.record-artifacts.v1")
	backupTaskOwnerType    = "platform_backup"
	backupTaskOwnerID      = "manual"
	backupTaskPurpose      = "platform_manual"
)

type backupTaskInput struct {
	OperationID string    `json:"operation_id"`
	RetainUntil time.Time `json:"retain_until"`
	RequestedBy uint64    `json:"requested_by"`
}

// SubmitManualBackup creates the frozen, two-stage Backup plan. The artifact
// root and database connection remain module-private executor configuration.
func (s *Service) SubmitManualBackup(ctx context.Context, operationID string, requestedBy uint64, retainUntil time.Time, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if s == nil || s.tasks == nil || s.writer == nil || operationID == "" || requestedBy == 0 || !retainUntil.After(time.Now().UTC()) {
		return moduleapi.TaskReceipt{}, moduleapi.ErrBackupInvalidInput
	}
	input, err := json.Marshal(backupTaskInput{OperationID: operationID, RetainUntil: retainUntil.UTC(), RequestedBy: requestedBy})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal backup task input: %w", err)
	}
	return s.tasks.Submit(ctx, moduleapi.SubmitTaskInput{Type: backupTaskType, Owner: moduleapi.TaskOwner{Type: backupTaskOwnerType, ID: backupTaskOwnerID}, RequestedBy: requestedBy, IdempotencyKey: idempotencyKey, Input: input, Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{
		{Key: "create-artifacts", ExecutorType: backupArtifactExecutor, Input: input, RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile},
		{Key: "record-artifacts", ExecutorType: backupRecordExecutor, Input: input, RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile},
	}}})
}

type backupArtifactWriter interface {
	Create(context.Context, backupTaskInput) (moduleapi.CreateBackupInput, error)
	Verify(backupTaskInput) (moduleapi.CreateBackupInput, error)
}

type backupArtifactTaskExecutor struct{ service *Service }

func (e backupArtifactTaskExecutor) Type() moduleapi.StageExecutorType { return backupArtifactExecutor }
func (e backupArtifactTaskExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	if e.service == nil || e.service.writer == nil {
		return errors.New("backup artifact writer is unavailable")
	}
	input, err := decodeBackupTaskInput(run.Input())
	if err != nil {
		return err
	}
	if _, err = e.service.writer.Create(ctx, input); err != nil {
		return fmt.Errorf("create backup artifacts: %w", err)
	}
	return run.AppendLog(ctx, moduleapi.TaskLogEntry{Stream: "system", Level: "info", Line: "backup artifacts created"})
}
func (backupArtifactTaskExecutor) Cancel(context.Context, moduleapi.StageRun) error { return nil }

type backupRecordTaskExecutor struct{ service *Service }

func (e backupRecordTaskExecutor) Type() moduleapi.StageExecutorType { return backupRecordExecutor }
func (e backupRecordTaskExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	if e.service == nil || e.service.writer == nil {
		return errors.New("backup artifact writer is unavailable")
	}
	input, err := decodeBackupTaskInput(run.Input())
	if err != nil {
		return err
	}
	artifacts, err := e.service.writer.Verify(input)
	if err != nil {
		return fmt.Errorf("verify backup artifacts: %w", err)
	}
	artifacts.TaskID = run.TaskID()
	if _, err = e.service.Create(ctx, artifacts); err != nil {
		return fmt.Errorf("record backup artifacts: %w", err)
	}
	return run.AppendLog(ctx, moduleapi.TaskLogEntry{Stream: "system", Level: "info", Line: "backup metadata recorded"})
}
func (backupRecordTaskExecutor) Cancel(context.Context, moduleapi.StageRun) error { return nil }

func decodeBackupTaskInput(raw json.RawMessage) (backupTaskInput, error) {
	var input backupTaskInput
	if err := json.Unmarshal(raw, &input); err != nil || input.OperationID == "" || input.RequestedBy == 0 || input.RetainUntil.IsZero() {
		return backupTaskInput{}, moduleapi.ErrBackupInvalidInput
	}
	return input, nil
}
