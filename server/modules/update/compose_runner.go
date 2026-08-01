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
	StopServices(context.Context, RunnerInput) error
	BootstrapMigrate(context.Context, RunnerInput) error
	Recreate(context.Context, RunnerInput) error
	DockerHealth(context.Context, RunnerInput) error
	Healthz(context.Context, RunnerInput) error
}

// RunnerStateReporter 由 runner 入口实现；执行核心只报告受控生命周期阶段，不直接持有状态卷写入能力。
type RunnerStateReporter interface {
	Report(RunnerPhase, int, string, string) error
}

const (
	runnerFailureInvalidInput = "invalid_input"
	runnerFailureBackup       = "backup_failed"
	runnerFailurePull         = "pull_failed"
	runnerFailureImageVerify  = "image_verification_failed"
	runnerFailureStopServices = "stop_services_failed"
	runnerFailureMigration    = "migration_failed"
	runnerFailureRecreate     = "recreate_failed"
	runnerFailureDockerHealth = "docker_health_failed"
	runnerFailureHealthz      = "healthz_failed"
	runnerProgressPreflight   = 5
	runnerProgressBackup      = 15
	runnerProgressPullImages  = 30
	runnerProgressStop        = 45
	runnerProgressApply       = 60
	runnerProgressMigration   = 70
	runnerProgressStart       = 82
	runnerProgressHealth      = 92
	runnerProgressTerminal    = 100
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

// ExecuteComposeRunner 执行唯一允许的 Compose 更新顺序，并把每个终态交给入口写入状态卷 receipt。
// Backup 成功后必须先校验 BackupReceipt；migration 动作开始前即记录边界，确保后续失败永远不会被解释为可自动恢复数据库的失败。
//
//nolint:cyclop,funlen // 每一个失败分支都对应不可折叠的升级安全边界和受控终态。
func ExecuteComposeRunner(ctx context.Context, input RunnerInput, actions ComposeRunnerActions) (RunnerReceipt, error) {
	receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: input.OperationID, RunnerID: input.RunnerID}
	reporter, ok := actions.(RunnerStateReporter)
	if !ok || reporter == nil {
		return receipt, errors.New("compose runner state reporter is required")
	}
	if err := reportRunnerState(reporter, RunnerPhasePreflight, runnerProgressPreflight, "checking_environment", ""); err != nil {
		return receipt, fmt.Errorf("persist runner preflight state: %w", err)
	}
	if err := runRunnerBackup(ctx, input, actions, reporter, &receipt); err != nil {
		return receipt, err
	}
	if err := runRunnerPreMigration(ctx, input, actions, reporter, &receipt); err != nil {
		return receipt, err
	}
	if err := runRunnerMigrationAndHealth(ctx, input, actions, reporter, &receipt); err != nil {
		return receipt, err
	}
	receipt.Succeeded = true
	return finalizeRunnerReceipt(receipt, reportRunnerState(reporter, RunnerPhaseSuccess, runnerProgressTerminal, "update_completed", ""))
}

func runRunnerBackup(ctx context.Context, input RunnerInput, actions ComposeRunnerActions, reporter RunnerStateReporter, receipt *RunnerReceipt) error {
	if err := validateRunnerExecution(input, actions); err != nil {
		receipt.FailureCode = runnerFailureInvalidInput
		return finalizeRunnerError(*receipt, errors.Join(err, reportRunnerState(reporter, RunnerPhaseFailed, runnerProgressTerminal, "update_failed", receipt.FailureCode)))
	}
	emitRunnerProgress(RunnerProgressBackingUp)
	if err := reportRunnerState(reporter, RunnerPhaseBackup, runnerProgressBackup, "creating_backup", ""); err != nil {
		return fmt.Errorf("persist runner backup state: %w", err)
	}
	if err := actions.Backup(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureBackup
		receipt.FailureStage, receipt.FailureDetail = runnerBackupFailureDiagnostic(err)
		return finalizeRunnerError(*receipt, errors.Join(err, reportRunnerState(reporter, RunnerPhaseFailed, runnerProgressTerminal, "update_failed", receipt.FailureCode)))
	}
	completion := actions.BackupReceipt()
	if err := validateBackupReceipt(completion, input); err != nil {
		receipt.FailureCode = runnerFailureBackup
		receipt.FailureStage = string(RunnerBackupFailureStageArtifactDigest)
		receipt.FailureDetail = string(RunnerBackupFailureDetailIOFailed)
		return finalizeRunnerError(*receipt, errors.Join(err, reportRunnerState(reporter, RunnerPhaseFailed, runnerProgressTerminal, "update_failed", receipt.FailureCode)))
	}
	receipt.BackupCompletion = &completion
	return nil
}

func runRunnerPreMigration(ctx context.Context, input RunnerInput, actions ComposeRunnerActions, reporter RunnerStateReporter, receipt *RunnerReceipt) error {
	emitRunnerProgress(RunnerProgressPulling)
	if err := reportRunnerState(reporter, RunnerPhasePullImages, runnerProgressPullImages, "pulling_images", ""); err != nil {
		return fmt.Errorf("persist runner image pull state: %w", err)
	}
	if err := actions.Pull(ctx, input); err != nil {
		receipt.FailureCode = runnerFailurePull
		receipt.RecoveryCompleted = recoverPreMigration(ctx, input, actions)
		return finalizeRunnerError(*receipt, errors.Join(err, reportReceiptFailure(reporter, *receipt)))
	}
	if err := reportRunnerState(reporter, RunnerPhasePullImages, runnerProgressPullImages, "verifying_images", ""); err != nil {
		return fmt.Errorf("persist runner image verification state: %w", err)
	}
	if err := actions.VerifyImages(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureImageVerify
		receipt.RecoveryCompleted = recoverPreMigration(ctx, input, actions)
		return finalizeRunnerError(*receipt, errors.Join(err, reportReceiptFailure(reporter, *receipt)))
	}

	// Atlas 是 forward-only：调用 bootstrap 前便跨过自动数据库恢复边界。
	// Compose bootstrap 可能替换服务容器，故控制器在迁移前持久化停止和应用边界。
	if err := reportRunnerState(reporter, RunnerPhaseStopServices, runnerProgressStop, "stopping_services", ""); err != nil {
		return fmt.Errorf("persist runner stop-services state: %w", err)
	}
	if err := actions.StopServices(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureStopServices
		receipt.RecoveryCompleted = recoverPreMigration(ctx, input, actions)
		return finalizeRunnerError(*receipt, errors.Join(err, reportReceiptFailure(reporter, *receipt)))
	}
	return nil
}

func runRunnerMigrationAndHealth(ctx context.Context, input RunnerInput, actions ComposeRunnerActions, reporter RunnerStateReporter, receipt *RunnerReceipt) error {
	if err := reportRunnerState(reporter, RunnerPhaseApplyUpdate, runnerProgressApply, "applying_update", ""); err != nil {
		return fmt.Errorf("persist runner apply-update state: %w", err)
	}
	emitRunnerProgress(RunnerProgressMigrating)
	if err := reportRunnerState(reporter, RunnerPhaseMigration, runnerProgressMigration, "running_migrations", ""); err != nil {
		return fmt.Errorf("persist runner migration state: %w", err)
	}
	receipt.MigrationStarted = true
	if err := actions.BootstrapMigrate(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureMigration
		return finalizeRunnerError(*receipt, errors.Join(err, reportReceiptFailure(reporter, *receipt)))
	}
	emitRunnerProgress(RunnerProgressRecreating)
	if err := reportRunnerState(reporter, RunnerPhaseStartServices, runnerProgressStart, "starting_services", ""); err != nil {
		return fmt.Errorf("persist runner start-services state: %w", err)
	}
	if err := actions.Recreate(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureRecreate
		return finalizeRunnerError(*receipt, errors.Join(err, reportReceiptFailure(reporter, *receipt)))
	}
	emitRunnerProgress(RunnerProgressVerifying)
	if err := reportRunnerState(reporter, RunnerPhaseHealthCheck, runnerProgressHealth, "checking_health", ""); err != nil {
		return fmt.Errorf("persist runner health-check state: %w", err)
	}
	if err := actions.DockerHealth(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureDockerHealth
		return finalizeRunnerError(*receipt, errors.Join(err, reportReceiptFailure(reporter, *receipt)))
	}
	if err := actions.Healthz(ctx, input); err != nil {
		receipt.FailureCode = runnerFailureHealthz
		return finalizeRunnerError(*receipt, errors.Join(err, reportReceiptFailure(reporter, *receipt)))
	}
	return nil
}

func reportReceiptFailure(reporter RunnerStateReporter, receipt RunnerReceipt) error {
	if receipt.RecoveryCompleted {
		return reportRunnerState(reporter, RunnerPhaseRollback, runnerProgressTerminal, "rollback_completed", receipt.FailureCode)
	}
	return reportRunnerState(reporter, RunnerPhaseFailed, runnerProgressTerminal, "update_failed", receipt.FailureCode)
}

func reportRunnerState(reporter RunnerStateReporter, phase RunnerPhase, progress int, message, failure string) error {
	if reporter == nil {
		return errors.New("compose runner state reporter is required")
	}
	return reporter.Report(phase, progress, message, failure)
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

func finalizeRunnerError(receipt RunnerReceipt, runErr error) error {
	_, err := finalizeRunnerReceipt(receipt, runErr)
	return err
}
