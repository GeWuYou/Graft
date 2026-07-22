package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var runnerOperationID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ComposeRunnerActions 是一次性 runner 可调用的固定动作集合。
// 它没有任意命令入口；Backup 的工件事实仍由 Backup capability 在 runner 交接后结算。
type ComposeRunnerActions interface {
	Backup(context.Context, RunnerInput) error
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
// migration 动作开始前即记录边界，确保后续失败永远不会被解释为可自动恢复数据库的失败。
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
	if err := actions.Pull(ctx, input); err != nil {
		receipt.FailureCode = runnerFailurePull
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
	if err := os.MkdirAll(filepath.Dir(path), runnerReceiptDirectoryMode); err != nil {
		return fmt.Errorf("create runner receipt directory: %w", err)
	}
	return os.WriteFile(path, append(contents, '\n'), runnerReceiptFileMode)
}

func runnerReceiptPath(input RunnerInput) (string, error) {
	if !runnerOperationID.MatchString(input.OperationID) || !filepath.IsAbs(input.Preflight.ComposeRoot) {
		return "", errors.New("runner receipt path is invalid")
	}
	return filepath.Join(input.Preflight.ComposeRoot, runnerReceiptDirectory, input.OperationID+".json"), nil
}
