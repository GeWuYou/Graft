package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

const (
	composeUpdateTaskType moduleapi.TaskType          = "platform.update.compose"
	composeUpdateExecutor moduleapi.StageExecutorType = "platform.update.compose-runner"
)

// ComposeUpdateOperation 是 Update 编排层持有的冻结关联；Task 和 Backup 的持久事实仍分别由各自模块拥有。
type ComposeUpdateOperation struct {
	OperationID            string           `json:"operation_id"`
	SourceVersion          string           `json:"source_version"`
	TargetVersion          string           `json:"target_version"`
	TaskID                 uint64           `json:"task_id"`
	BackupID               uint64           `json:"backup_id,omitempty"`
	RequestedBy            uint64           `json:"requested_by,omitempty"`
	Outcome                ExecutionOutcome `json:"status"`
	ReceiptIntegritySHA256 string           `json:"-"`
	FailureCode            string           `json:"failure_code,omitempty"`
	RecoveryCompleted      bool             `json:"recovery_completed"`
	CreatedAt              time.Time        `json:"created_at"`
	StartedAt              time.Time        `json:"started_at"`
	FinishedAt             *time.Time       `json:"finished_at,omitempty"`
}

// ComposeExecutionCoordinator 只消费 Task 与 Backup capability，避免 Update 直接访问其它模块的仓储。
type ComposeExecutionCoordinator struct {
	tasks   moduleapi.TaskService
	backups moduleapi.BackupService
}

// NewComposeExecutionCoordinator 创建 Compose receipt 编排器。
func NewComposeExecutionCoordinator(tasks moduleapi.TaskService, backups moduleapi.BackupService) *ComposeExecutionCoordinator {
	return &ComposeExecutionCoordinator{tasks: tasks, backups: backups}
}

// Start 冻结目标版本、提交外部 receipt Task，并在 runner 启动前创建 Backup handoff。
func (c *ComposeExecutionCoordinator) Start(ctx context.Context, operation ComposeUpdateOperation, requestedBy uint64, handoff moduleapi.BackupRunnerHandoffPlan) (ComposeUpdateOperation, RunnerInput, error) {
	if c == nil || c.tasks == nil || c.backups == nil || !validOperation(operation) || strings.TrimSpace(handoff.OperationID) != operation.OperationID {
		return ComposeUpdateOperation{}, RunnerInput{}, errors.New("compose update operation is invalid")
	}
	input, err := json.Marshal(struct {
		SourceVersion string `json:"source_version"`
		TargetVersion string `json:"target_version"`
	}{SourceVersion: operation.SourceVersion, TargetVersion: operation.TargetVersion})
	if err != nil {
		return ComposeUpdateOperation{}, RunnerInput{}, fmt.Errorf("encode compose update task input: %w", err)
	}
	task, err := c.tasks.Submit(ctx, moduleapi.SubmitTaskInput{
		Type: composeUpdateTaskType, Owner: moduleapi.TaskOwner{Type: "platform_update", ID: operation.OperationID}, RequestedBy: requestedBy, Input: input,
		Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{Key: "compose_runner", ExecutorType: composeUpdateExecutor, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile, ExternalReceipt: &moduleapi.ExternalReceiptExpectation{Protocol: "compose-runner/v1", OperationID: operation.OperationID}}}},
	})
	if err != nil {
		return ComposeUpdateOperation{}, RunnerInput{}, fmt.Errorf("submit compose update task: %w", err)
	}
	handoff.TaskID = task.TaskID
	prepared, err := c.backups.PrepareRunnerHandoff(ctx, handoff)
	if err != nil {
		_ = c.tasks.Cancel(ctx, task.TaskID)
		return ComposeUpdateOperation{}, RunnerInput{}, fmt.Errorf("prepare update backup handoff: %w", err)
	}
	operation.TaskID = task.TaskID
	return operation, RunnerInput{ProtocolVersion: runnerProtocolVersion, OperationID: operation.OperationID, TaskID: task.TaskID}, validatePreparedHandoff(prepared, operation)
}

// CancelBeforeLaunch 通过各 owner capability 清理 runner 尚未启动时的 Task 与 Backup handoff，避免 Update 写入其它模块的事实表。
func (c *ComposeExecutionCoordinator) CancelBeforeLaunch(ctx context.Context, operation ComposeUpdateOperation) error {
	if c == nil || c.tasks == nil || c.backups == nil || operation.TaskID == 0 || !runnerOperationID.MatchString(operation.OperationID) {
		return errors.New("compose update cancellation is unavailable")
	}
	backupErr := c.backups.CancelRunnerHandoff(ctx, operation.OperationID, operation.TaskID)
	taskErr := c.tasks.Cancel(ctx, operation.TaskID)
	if backupErr != nil || taskErr != nil {
		return errors.Join(backupErr, taskErr)
	}
	return nil
}

// SettleReceipt consumes runner evidence after recreation. Migration-started failures become NEEDS_ATTENTION and never request database restore.
func (c *ComposeExecutionCoordinator) SettleReceipt(ctx context.Context, operation ComposeUpdateOperation, receipt RunnerReceipt) (ComposeUpdateOperation, error) {
	if err := validateReceiptSettlement(c, operation, receipt); err != nil {
		return ComposeUpdateOperation{}, errors.New("compose runner receipt does not match update operation")
	}
	completed, err := c.completeBackupHandoff(ctx, operation, receipt.BackupCompletion)
	if err != nil {
		return ComposeUpdateOperation{}, err
	}
	operation.BackupID = completed
	outcome := ClassifyRunnerReceipt(receipt)
	externalOutcome := taskReceiptOutcome(outcome)
	integrity, err := RunnerReceiptIntegrity(receipt)
	if err != nil {
		return ComposeUpdateOperation{}, err
	}
	settlement, err := c.tasks.SettleExternalReceipt(ctx, moduleapi.ExternalTaskReceipt{TaskID: operation.TaskID, ExecutorType: composeUpdateExecutor, Protocol: "compose-runner/v1", OperationID: operation.OperationID, Outcome: externalOutcome, FailureCode: receipt.FailureCode, IntegritySHA256: integrity})
	if err != nil {
		return ComposeUpdateOperation{}, fmt.Errorf("settle compose runner receipt: %w", err)
	}
	operation.Outcome = outcome
	operation.ReceiptIntegritySHA256 = integrity
	operation.FailureCode = receipt.FailureCode
	operation.RecoveryCompleted = receipt.RecoveryCompleted
	if settlement.TaskID != operation.TaskID {
		return ComposeUpdateOperation{}, errors.New("settled task does not match update operation")
	}
	return operation, nil
}

func validateReceiptSettlement(c *ComposeExecutionCoordinator, operation ComposeUpdateOperation, receipt RunnerReceipt) error {
	if c == nil || c.tasks == nil || c.backups == nil || operation.TaskID == 0 || receipt.ProtocolVersion != runnerProtocolVersion || receipt.OperationID != operation.OperationID {
		return errors.New("invalid")
	}
	return nil
}

func (c *ComposeExecutionCoordinator) completeBackupHandoff(ctx context.Context, operation ComposeUpdateOperation, input *moduleapi.CompleteBackupRunnerHandoffInput) (uint64, error) {
	if input == nil {
		return operation.BackupID, nil
	}
	completion, err := c.backups.CompleteRunnerHandoff(ctx, *input)
	if err != nil {
		return 0, fmt.Errorf("complete update backup handoff: %w", err)
	}
	if completion.OperationID != operation.OperationID || completion.TaskID != operation.TaskID {
		return 0, errors.New("backup handoff completion does not match update operation")
	}
	return completion.BackupID, nil
}

func taskReceiptOutcome(outcome ExecutionOutcome) moduleapi.ExternalReceiptOutcome {
	switch outcome {
	case ExecutionOutcomeSuccess:
		return moduleapi.ExternalReceiptOutcomeSuccess
	case ExecutionOutcomeNeedsAttention:
		return moduleapi.ExternalReceiptOutcomeNeedsAttention
	default:
		return moduleapi.ExternalReceiptOutcomeFailed
	}
}

func validOperation(operation ComposeUpdateOperation) bool {
	return runnerOperationID.MatchString(operation.OperationID) && strings.TrimSpace(operation.SourceVersion) != "" && strings.TrimSpace(operation.TargetVersion) != "" && operation.SourceVersion != operation.TargetVersion
}

func validatePreparedHandoff(plan moduleapi.BackupRunnerHandoffPlan, operation ComposeUpdateOperation) error {
	if plan.OperationID != operation.OperationID || plan.TaskID != operation.TaskID {
		return errors.New("prepared backup handoff does not match update operation")
	}
	return nil
}
