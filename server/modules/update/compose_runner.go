package update

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"

	"graft/server/internal/moduleapi"
)

var runnerOperationID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ComposeRunnerActions 是一次性 runner 可调用的固定动作集合。
// 它没有任意命令入口；BackupReceipt 必须返回紧接 Backup 成功后生成的无秘密完整性证据，
// runner 才能继续执行后续阶段并写入可结算的 receipt。
type ComposeRunnerActions interface {
	Backup(context.Context, RunnerInput) error
	BackupReceipt() moduleapi.CompleteBackupRunnerHandoffInput
	Pull(context.Context, RunnerInput) error
	VerifyImages(context.Context, RunnerInput) error
	BootstrapMigrate(context.Context, RunnerInput) error
	Recreate(context.Context, RunnerInput) error
	DockerHealth(context.Context, RunnerInput) error
	Healthz(context.Context, RunnerInput) error
}

const (
	runnerFailureInvalidInput = "invalid_input"
	runnerFailureBackup       = "backup_failed"
	runnerFailurePull         = "pull_failed"
	runnerFailureImageVerify  = "image_verification_failed"
	runnerFailureMigration    = "migration_failed"
	runnerFailureRecreate     = "recreate_failed"
	runnerFailureDockerHealth = "docker_health_failed"
	runnerFailureHealthz      = "healthz_failed"
)

// ExecuteComposeRunner 执行唯一允许的 Compose 更新顺序，并把每个终态交给入口写入容器日志回执。
// Backup 成功后必须先校验 BackupReceipt；migration 动作开始前即记录边界，确保后续失败永远不会被解释为可自动恢复数据库的失败。
func ExecuteComposeRunner(ctx context.Context, input RunnerInput, actions ComposeRunnerActions) (RunnerReceipt, error) {
	receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: input.OperationID}
	if err := validateRunnerExecution(input, actions); err != nil {
		receipt.FailureCode = runnerFailureInvalidInput
		return finalizeRunnerReceipt(receipt, err)
	}
	if err := actions.Backup(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureBackup
		return finalizeRunnerReceipt(receipt, err)
	}
	completion := actions.BackupReceipt()
	if err := validateBackupReceipt(completion, input); err != nil {
		receipt.FailureCode = runnerFailureBackup
		return finalizeRunnerReceipt(receipt, err)
	}
	receipt.BackupCompletion = &completion
	if err := actions.Pull(ctx, input); err != nil {
		receipt.FailureCode = runnerFailurePull
		receipt.RecoveryCompleted = recoverPreMigration(ctx, input, actions)
		return finalizeRunnerReceipt(receipt, err)
	}
	if err := actions.VerifyImages(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureImageVerify
		receipt.RecoveryCompleted = recoverPreMigration(ctx, input, actions)
		return finalizeRunnerReceipt(receipt, err)
	}

	// Atlas 是 forward-only：调用 bootstrap 前便跨过自动数据库恢复边界。
	receipt.MigrationStarted = true
	if err := actions.BootstrapMigrate(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureMigration
		return finalizeRunnerReceipt(receipt, err)
	}
	if err := actions.Recreate(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureRecreate
		return finalizeRunnerReceipt(receipt, err)
	}
	if err := actions.DockerHealth(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureDockerHealth
		return finalizeRunnerReceipt(receipt, err)
	}
	if err := actions.Healthz(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureHealthz
		return finalizeRunnerReceipt(receipt, err)
	}
	receipt.Succeeded = true
	return finalizeRunnerReceipt(receipt, nil)
}

func validateBackupReceipt(completion moduleapi.CompleteBackupRunnerHandoffInput, input RunnerInput) error {
	return validateBackupCompletion(completion, input.OperationID, input.TaskID)
}

func validateBackupCompletion(completion moduleapi.CompleteBackupRunnerHandoffInput, operationID string, taskID uint64) error {
	if completion.OperationID != operationID || completion.TaskID != taskID || completion.ConfigSnapshotBytes < 0 || completion.DatabaseDumpBytes < 0 || len(completion.ConfigSnapshotSHA256) != 64 || len(completion.DatabaseDumpSHA256) != 64 {
		return errors.New("backup completion does not match runner input")
	}
	if _, err := hex.DecodeString(completion.ConfigSnapshotSHA256); err != nil {
		return errors.New("backup config snapshot digest is invalid")
	}
	if _, err := hex.DecodeString(completion.DatabaseDumpSHA256); err != nil {
		return errors.New("backup database dump digest is invalid")
	}
	return nil
}

func recoverPreMigration(ctx context.Context, input RunnerInput, actions ComposeRunnerActions) bool {
	recovery, ok := actions.(interface {
		RecoverPreMigration(context.Context, RunnerInput) error
	})
	return ok && recovery.RecoverPreMigration(ctx, input) == nil
}

func validateRunnerExecution(input RunnerInput, actions ComposeRunnerActions) error {
	if actions == nil {
		return errors.New("compose runner actions are required")
	}
	if !runnerOperationID.MatchString(input.OperationID) {
		return errors.New("runner operation identity is unsafe for receipt path")
	}
	return ValidateRunnerInput(input)
}

func finalizeRunnerReceipt(receipt RunnerReceipt, runErr error) (RunnerReceipt, error) {
	if runErr != nil {
		return receipt, fmt.Errorf("compose runner %s: %w", receipt.FailureCode, runErr)
	}
	return receipt, nil
}
