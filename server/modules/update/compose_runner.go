package update

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	BootstrapMigrate(context.Context, RunnerInput) error
	Recreate(context.Context, RunnerInput) error
	DockerHealth(context.Context, RunnerInput) error
	Healthz(context.Context, RunnerInput) error
}

const (
	runnerFailureInvalidInput  = "invalid_input"
	runnerFailureBackup        = "backup_failed"
	runnerFailurePull          = "pull_failed"
	runnerFailureMigration     = "migration_failed"
	runnerFailureRecreate      = "recreate_failed"
	runnerFailureDockerHealth  = "docker_health_failed"
	runnerFailureHealthz       = "healthz_failed"
	runnerFailureReceiptWrite  = "receipt_write_failed"
	runnerReceiptDirectory     = ".graft-update/receipts"
	runnerReceiptDirectoryMode = 0o700
	runnerReceiptFileMode      = 0o600
)

// ExecuteComposeRunner 执行唯一允许的 Compose 更新顺序，并在每个终态写入受限 receipt。
// Backup 成功后必须先校验 BackupReceipt；migration 动作开始前即记录边界，确保后续失败永远不会被解释为可自动恢复数据库的失败。
func ExecuteComposeRunner(ctx context.Context, input RunnerInput, actions ComposeRunnerActions) (RunnerReceipt, error) {
	receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: input.OperationID}
	if err := validateRunnerExecution(input, actions); err != nil {
		receipt.FailureCode = runnerFailureInvalidInput
		return writeRunnerReceipt(input, receipt, err)
	}
	if err := actions.Backup(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureBackup
		return writeRunnerReceipt(input, receipt, err)
	}
	completion := actions.BackupReceipt()
	if err := validateBackupReceipt(completion, input); err != nil {
		receipt.FailureCode = runnerFailureBackup
		return writeRunnerReceipt(input, receipt, err)
	}
	receipt.BackupCompletion = &completion
	if err := actions.Pull(ctx, input); err != nil {
		receipt.FailureCode = runnerFailurePull
		receipt.RecoveryCompleted = recoverPreMigration(ctx, input, actions)
		return writeRunnerReceipt(input, receipt, err)
	}

	// Atlas 是 forward-only：调用 bootstrap 前便跨过自动数据库恢复边界。
	receipt.MigrationStarted = true
	if err := actions.BootstrapMigrate(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureMigration
		return writeRunnerReceipt(input, receipt, err)
	}
	if err := actions.Recreate(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureRecreate
		return writeRunnerReceipt(input, receipt, err)
	}
	if err := actions.DockerHealth(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureDockerHealth
		return writeRunnerReceipt(input, receipt, err)
	}
	if err := actions.Healthz(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureHealthz
		return writeRunnerReceipt(input, receipt, err)
	}
	receipt.Succeeded = true
	return writeRunnerReceipt(input, receipt, nil)
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

func writeRunnerReceipt(input RunnerInput, receipt RunnerReceipt, runErr error) (RunnerReceipt, error) {
	if err := persistRunnerReceipt(input, receipt); err != nil {
		return RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: input.OperationID, MigrationStarted: receipt.MigrationStarted, FailureCode: runnerFailureReceiptWrite}, fmt.Errorf("persist compose runner receipt: %w", err)
	}
	if runErr != nil {
		return receipt, fmt.Errorf("compose runner %s: %w", receipt.FailureCode, runErr)
	}
	return receipt, nil
}

func persistRunnerReceipt(input RunnerInput, receipt RunnerReceipt) error {
	path, err := runnerReceiptPath(input)
	if err != nil {
		return err
	}
	contents, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("encode runner receipt: %w", err)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, runnerReceiptDirectoryMode); err != nil {
		return fmt.Errorf("create runner receipt directory: %w", err)
	}
	temporaryPath, err := writeSyncedRunnerReceipt(directory, contents)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace runner receipt: %w", err)
	}
	return syncRunnerReceiptDirectory(directory)
}

func writeSyncedRunnerReceipt(directory string, contents []byte) (string, error) {
	file, err := os.CreateTemp(directory, ".receipt-*")
	if err != nil {
		return "", fmt.Errorf("create temporary runner receipt: %w", err)
	}
	temporaryPath := file.Name()
	cleanup := func(cause error) (string, error) {
		_ = file.Close()
		_ = os.Remove(temporaryPath)
		return "", cause
	}
	if err := file.Chmod(runnerReceiptFileMode); err != nil {
		return cleanup(fmt.Errorf("set runner receipt permissions: %w", err))
	}
	if _, err := file.Write(append(contents, '\n')); err != nil {
		return cleanup(fmt.Errorf("write temporary runner receipt: %w", err))
	}
	if err := file.Sync(); err != nil {
		return cleanup(fmt.Errorf("sync temporary runner receipt: %w", err))
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return "", fmt.Errorf("close temporary runner receipt: %w", err)
	}
	return temporaryPath, nil
}

func syncRunnerReceiptDirectory(directory string) error {
	// #nosec G304 -- directory derives from the validated absolute Compose root and fixed receipt subdirectory.
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open runner receipt directory: %w", err)
	}
	defer func() { _ = directoryHandle.Close() }()
	if err := directoryHandle.Sync(); err != nil {
		return fmt.Errorf("sync runner receipt directory: %w", err)
	}
	return nil
}

func runnerReceiptPath(input RunnerInput) (string, error) {
	if !runnerOperationID.MatchString(input.OperationID) || !filepath.IsAbs(input.Preflight.ComposeRoot) {
		return "", errors.New("runner receipt path is invalid")
	}
	return filepath.Join(input.Preflight.ComposeRoot, runnerReceiptDirectory, input.OperationID+".json"), nil
}
