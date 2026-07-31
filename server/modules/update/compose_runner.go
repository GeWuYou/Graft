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

// RunnerBackupFailureStage 标识备份期间可安全暴露给操作者的固定失败阶段。
type RunnerBackupFailureStage string

const (
	// RunnerBackupFailureStageArtifactDirectory 表示创建暂存或 server 工件目录失败。
	RunnerBackupFailureStageArtifactDirectory RunnerBackupFailureStage = "artifact_directory"
	// RunnerBackupFailureStageConfigSnapshot 表示复制私有部署环境快照失败。
	RunnerBackupFailureStageConfigSnapshot RunnerBackupFailureStage = "env_snapshot"
	// RunnerBackupFailureStageDatabaseDump 表示导出 PostgreSQL 数据失败。
	RunnerBackupFailureStageDatabaseDump RunnerBackupFailureStage = "postgres_dump"
	// RunnerBackupFailureStageArtifactDigest 表示计算备份产物摘要失败。
	RunnerBackupFailureStageArtifactDigest RunnerBackupFailureStage = "artifact_digest"
)

// RunnerBackupFailureDetail 标识备份失败的无秘密原因类别。
type RunnerBackupFailureDetail string

const (
	// RunnerBackupFailureDetailPermissionDenied 表示固定操作被文件系统权限拒绝。
	RunnerBackupFailureDetailPermissionDenied RunnerBackupFailureDetail = "permission_denied"
	// RunnerBackupFailureDetailCommandFailed 表示固定 Docker 或 Compose 命令执行失败。
	RunnerBackupFailureDetailCommandFailed RunnerBackupFailureDetail = "command_failed"
	// RunnerBackupFailureDetailIOFailed 表示无法分类为权限或命令的 I/O 失败。
	RunnerBackupFailureDetailIOFailed RunnerBackupFailureDetail = "io_failed"
)

type runnerBackupFailure struct {
	stage  RunnerBackupFailureStage
	detail RunnerBackupFailureDetail
	cause  error
}

func (e *runnerBackupFailure) Error() string {
	return string(e.stage) + ": " + string(e.detail)
}

func (e *runnerBackupFailure) Unwrap() error { return e.cause }

// NewRunnerBackupFailure 将底层备份错误包装为可写入 receipt 的无秘密诊断。
// 调用方必须只传入本包声明的固定阶段与详情，避免路径、stderr 或凭证进入回执。
func NewRunnerBackupFailure(stage RunnerBackupFailureStage, detail RunnerBackupFailureDetail, cause error) error {
	if !validRunnerBackupFailureStage(stage) {
		stage = RunnerBackupFailureStageArtifactDirectory
	}
	if !validRunnerBackupFailureDetail(detail) {
		detail = RunnerBackupFailureDetailIOFailed
	}
	return &runnerBackupFailure{stage: stage, detail: detail, cause: cause}
}

// ExecuteComposeRunner 执行唯一允许的 Compose 更新顺序，并把每个终态交给入口写入容器日志回执。
// Backup 成功后必须先校验 BackupReceipt；migration 动作开始前即记录边界，确保后续失败永远不会被解释为可自动恢复数据库的失败。
func ExecuteComposeRunner(ctx context.Context, input RunnerInput, actions ComposeRunnerActions) (RunnerReceipt, error) {
	receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: input.OperationID}
	if err := validateRunnerExecution(input, actions); err != nil {
		receipt.FailureCode = runnerFailureInvalidInput
		return finalizeRunnerReceipt(receipt, err)
	}
	emitRunnerProgress(RunnerProgressBackingUp)
	if err := actions.Backup(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureBackup
		receipt.FailureStage, receipt.FailureDetail = runnerBackupFailureDiagnostic(err)
		return finalizeRunnerReceipt(receipt, err)
	}
	completion := actions.BackupReceipt()
	if err := validateBackupReceipt(completion, input); err != nil {
		receipt.FailureCode = runnerFailureBackup
		receipt.FailureStage = string(RunnerBackupFailureStageArtifactDigest)
		receipt.FailureDetail = string(RunnerBackupFailureDetailIOFailed)
		return finalizeRunnerReceipt(receipt, err)
	}
	receipt.BackupCompletion = &completion
	emitRunnerProgress(RunnerProgressPulling)
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
	emitRunnerProgress(RunnerProgressMigrating)
	receipt.MigrationStarted = true
	if err := actions.BootstrapMigrate(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureMigration
		return finalizeRunnerReceipt(receipt, err)
	}
	emitRunnerProgress(RunnerProgressRecreating)
	if err := actions.Recreate(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureRecreate
		return finalizeRunnerReceipt(receipt, err)
	}
	emitRunnerProgress(RunnerProgressVerifying)
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

func emitRunnerProgress(progress RunnerProgress) {
	_, _ = fmt.Println(RunnerProgressLogMarker + string(progress))
}

func runnerBackupFailureDiagnostic(err error) (string, string) {
	var failure *runnerBackupFailure
	if !errors.As(err, &failure) {
		return string(RunnerBackupFailureStageArtifactDirectory), string(RunnerBackupFailureDetailIOFailed)
	}
	if !validRunnerBackupFailureStage(failure.stage) || !validRunnerBackupFailureDetail(failure.detail) {
		return string(RunnerBackupFailureStageArtifactDirectory), string(RunnerBackupFailureDetailIOFailed)
	}
	return string(failure.stage), string(failure.detail)
}

func validRunnerBackupFailureStage(value RunnerBackupFailureStage) bool {
	switch value {
	case RunnerBackupFailureStageArtifactDirectory, RunnerBackupFailureStageConfigSnapshot, RunnerBackupFailureStageDatabaseDump, RunnerBackupFailureStageArtifactDigest:
		return true
	default:
		return false
	}
}

func validRunnerBackupFailureDetail(value RunnerBackupFailureDetail) bool {
	switch value {
	case RunnerBackupFailureDetailPermissionDenied, RunnerBackupFailureDetailCommandFailed, RunnerBackupFailureDetailIOFailed:
		return true
	default:
		return false
	}
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
