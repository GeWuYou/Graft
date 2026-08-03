// Package main provides the short-lived, no-HTTP Compose update runner entrypoint.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"graft/server/internal/moduleapi"
	"graft/server/modules/update"
)

var imageTagPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)

const (
	directoryPermission                   os.FileMode = 0o700
	privateFilePermission                 os.FileMode = 0o600
	composeFileArgumentCapacityMultiplier             = 2
	runnerExecutionTimeout                            = 15 * time.Minute
	runnerLeaseHeartbeatInterval                      = 30 * time.Second
	runnerLeaseHeartbeatFailureLimit                  = 3
	healthzCurlTimeoutSeconds                         = "30"
	runnerIDRandomBytes                               = 12
	runnerProtocolVersion                             = 2
)

// main 只执行一次性 Compose runner 协议，不启动 HTTP、数据库连接或业务状态。
//
//nolint:cyclop // 正常执行与受保护恢复是两个互斥入口，各自保留显式失败输出边界。
func main() {
	if encoded := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_RUNNER_RECOVERY_STATE_B64")); encoded != "" {
		if err := recoverTerminatedRunner(encoded); err != nil {
			fatal(err)
		}
		return
	}
	input, err := readRunnerInput()
	if err != nil {
		fatal(err)
	}
	runnerCtx, cancel := context.WithTimeout(context.Background(), runnerExecutionTimeout)
	defer cancel()
	reporter, err := newStateReporter(input)
	if err != nil {
		_ = writeRunnerFailureLog(os.Stdout, runnerStateFailureEvidence(input, err))
		fatal(err)
	}
	if err := reporter.Initialize(); err != nil {
		_ = writeRunnerFailureLog(os.Stdout, runnerStateFailureEvidence(input, err))
		fatal(err)
	}
	reporter.cancelOnHeartbeatFailure = cancel
	reporter.StartHeartbeat()
	defer reporter.StopHeartbeat()
	receipt, executionErr := update.ExecuteComposeRunner(runnerCtx, input, &actions{reporter: reporter})
	finalizeErr := reporter.Finalize(receipt)
	if cleanupErr := cleanupBackupStaging(input); cleanupErr != nil {
		// 回执不能泄露宿主路径或备份内容，保留原终态以便 server 结算。
		_, _ = fmt.Fprintln(os.Stderr, "remove backup staging directory: failed")
	}
	if err := writeRunnerReceiptLog(os.Stdout, receipt); err != nil {
		if finalizeErr != nil {
			fatal(fmt.Errorf("persist terminal update runner state: %w; write runner receipt log: %v", finalizeErr, err))
		}
		fatal(fmt.Errorf("write runner receipt log: %w", err))
	}
	if finalizeErr != nil {
		_ = writeRunnerFailureLog(os.Stdout, runnerStateFailureEvidence(input, finalizeErr))
		fatal(fmt.Errorf("persist terminal update runner state: %w", finalizeErr))
	}
	if executionErr != nil {
		fatal(executionErr)
	}
}

// recoverTerminatedRunner 仅将已验证的中断快照结算为失败，绝不恢复或继续升级执行。
func recoverTerminatedRunner(encoded string) error {
	store, err := update.NewFileRunnerStateStore(update.RunnerStateRoot)
	if err != nil {
		return err
	}
	return recoverTerminatedRunnerWithStore(encoded, store, os.Stdout)
}

//nolint:cyclop,gocyclo // 两种恢复输入需要在同一原子状态边界内完成严格绑定校验。
func recoverTerminatedRunnerWithStore(encoded string, store update.RunnerStateWriter, writer io.Writer) error {
	contents, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("decode recovery runner state: %w", err)
	}
	var input update.RunnerRecoveryInput
	if err := json.Unmarshal(contents, &input); err != nil {
		return fmt.Errorf("decode recovery runner state: %w", err)
	}
	if input.State != nil && (input.State.OperationID != input.OperationID || input.State.RunnerID != input.RunnerID) {
		return errors.New("recovery runner state binding changed")
	}
	persisted, err := store.Read()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read persisted recovery runner state: %w", err)
	}
	if input.State != nil && (errors.Is(err, os.ErrNotExist) || persisted.OperationID != input.OperationID || persisted.RunnerID != input.RunnerID || persisted.Revision != input.State.Revision || persisted.Digest != input.State.Digest) {
		return errors.New("recovery runner state binding changed")
	}
	if input.State == nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("recovery runner unexpectedly found a state snapshot")
	}
	if input.State == nil {
		persisted = update.RunnerState{}
	}
	recoveryRunnerID, err := newRunnerID()
	if err != nil {
		return err
	}
	reporter := &stateReporter{store: store, input: update.RunnerInput{ProtocolVersion: runnerProtocolVersion, OperationID: input.OperationID, RunnerID: recoveryRunnerID, SourceVersion: input.SourceVersion, TargetVersion: input.TargetVersion, Preflight: update.ComposePreflight{DeploymentStrategy: update.DeploymentStrategy(input.Strategy)}}, runnerID: recoveryRunnerID, current: persisted}
	receipt := update.RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: input.OperationID, RunnerID: recoveryRunnerID, FailureCode: update.RunnerFailureCodeInvalidInput, FailureStage: "runner_recovery", FailureDetail: "interrupted_before_migration"}
	if err := reporter.Finalize(receipt); err != nil {
		return fmt.Errorf("persist recovered terminal runner state: %w", err)
	}
	return writeRunnerReceiptLog(writer, receipt)
}

func cleanupBackupStaging(in update.RunnerInput) error {
	if err := update.ValidateRunnerInput(in); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(in.Preflight.ComposeRoot, ".graft-update", "backups", in.OperationID))
}

func readRunnerInput() (update.RunnerInput, error) {
	encoded := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_RUNNER_INPUT_B64"))
	if encoded != "" {
		contents, err := base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			return update.RunnerInput{}, fmt.Errorf("decode inline runner input: %w", err)
		}
		return decodeRunnerInput(contents)
	}

	path := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_RUNNER_INPUT"))
	if path == "" {
		return update.RunnerInput{}, errors.New("runner input is missing")
	}
	// #nosec G304,G703 -- the server supplies a validated, host-mounted runner input path.
	contents, err := os.ReadFile(path)
	if err != nil {
		return update.RunnerInput{}, fmt.Errorf("read runner input: %w", err)
	}
	return decodeRunnerInput(contents)
}

func decodeRunnerInput(contents []byte) (update.RunnerInput, error) {
	var input update.RunnerInput
	if err := json.Unmarshal(contents, &input); err != nil {
		return update.RunnerInput{}, fmt.Errorf("decode runner input: %w", err)
	}
	return input, nil
}

func writeRunnerReceiptLog(writer io.Writer, receipt update.RunnerReceipt) error {
	contents, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("encode runner receipt: %w", err)
	}
	encoded := base64.RawStdEncoding.EncodeToString(contents)
	if _, err := fmt.Fprintln(writer, update.RunnerReceiptLogMarker+encoded); err != nil {
		return err
	}
	return nil
}

func writeRunnerFailureLog(writer io.Writer, evidence update.RunnerFailureEvidence) error {
	contents, err := json.Marshal(evidence)
	if err != nil {
		return fmt.Errorf("encode runner failure evidence: %w", err)
	}
	if _, err := fmt.Fprintln(writer, update.RunnerFailureLogMarker+base64.RawStdEncoding.EncodeToString(contents)); err != nil {
		return err
	}
	return nil
}

func runnerStateFailureEvidence(input update.RunnerInput, err error) update.RunnerFailureEvidence {
	stage := update.RunnerFailureStageIOFailed
	if errors.Is(err, os.ErrPermission) {
		stage = update.RunnerFailureStagePermissionDenied
	}
	return update.RunnerFailureEvidence{ProtocolVersion: runnerProtocolVersion, OperationID: input.OperationID, RunnerID: input.RunnerID, FailureCode: update.RunnerFailureCodeStateWriteFailed, FailureStage: stage}
}

func fatal(err error) { _, _ = fmt.Fprintln(os.Stderr, err); os.Exit(1) }

type actions struct {
	backup   moduleapi.CompleteBackupRunnerHandoffInput
	reporter *stateReporter
}

// Report 让执行核心报告受限生命周期阶段，状态卷协议仍由入口独占。
func (a *actions) Report(phase update.RunnerPhase, progress int, message, failure string) error {
	if a != nil && a.reporter != nil {
		return a.reporter.Report(phase, progress, message, failure)
	}
	return errors.New("runner state reporter is unavailable")
}

type stateReporter struct {
	store                    update.RunnerStateWriter
	input                    update.RunnerInput
	runnerID                 string
	current                  update.RunnerState
	mu                       sync.Mutex
	stop                     chan struct{}
	done                     chan struct{}
	heartbeatFailures        int
	cancelOnHeartbeatFailure context.CancelFunc
}

func newStateReporter(input update.RunnerInput) (*stateReporter, error) {
	store, err := update.NewFileRunnerStateStore(update.RunnerStateRoot)
	if err != nil {
		return nil, err
	}
	identifier := input.RunnerID
	if identifier == "" {
		identifier, err = newRunnerID()
		if err != nil {
			return nil, err
		}
	}
	return &stateReporter{store: store, input: input, runnerID: identifier}, nil
}

func newRunnerID() (string, error) {
	bytes := make([]byte, runnerIDRandomBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate runner identity: %w", err)
	}
	return "runner-" + hex.EncodeToString(bytes), nil
}

func (r *stateReporter) Report(phase update.RunnerPhase, progress int, message, failure string) error {
	return r.write(phase, progress, message, failure)
}

// Initialize 先持久化 READY；状态卷不可用时不得执行可能停止 server 的升级动作。
func (r *stateReporter) Initialize() error {
	return r.write(update.RunnerPhaseReady, 0, "runner_accepted", "")
}

func (r *stateReporter) write(phase update.RunnerPhase, progress int, message, failure string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writeLocked(phase, progress, message, failure)
}

func (r *stateReporter) writeLocked(phase update.RunnerPhase, progress int, message, failure string) error {
	if r == nil || r.store == nil {
		return errors.New("runner state reporter is unavailable")
	}
	next := update.NewRunnerState(r.input, r.runnerID, phase, progress, message, failure, r.current)
	if err := r.store.Write(next); err != nil {
		return err
	}
	r.current = next
	return nil
}

// StartHeartbeat 独立于稀疏业务阶段上报续租持久执行权。
func (r *stateReporter) StartHeartbeat() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.stop != nil || r.current.OperationID == "" {
		r.mu.Unlock()
		return
	}
	r.stop, r.done = make(chan struct{}), make(chan struct{})
	stop, done := r.stop, r.done
	r.mu.Unlock()
	go func() {
		defer close(done)
		ticker := time.NewTicker(runnerLeaseHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if r.heartbeat() {
					return
				}
			}
		}
	}()
}

// heartbeat 只在连续续租失败达到阈值后取消 runner 执行，避免一次短暂 I/O 故障中断升级。
func (r *stateReporter) heartbeat() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	if r.current.OperationID == "" || isTerminalPhase(r.current.Phase) {
		r.mu.Unlock()
		return false
	}
	cancel := context.CancelFunc(nil)
	next, err := r.store.Heartbeat(r.current)
	if err == nil {
		r.current = next
		r.heartbeatFailures = 0
	} else {
		r.heartbeatFailures++
		if r.heartbeatFailures >= runnerLeaseHeartbeatFailureLimit {
			cancel = r.cancelOnHeartbeatFailure
		}
	}
	r.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

// StopHeartbeat 停止续租 goroutine 并等待退出，确保 Finalize 不会与后台 heartbeat 争用状态快照。
func (r *stateReporter) StopHeartbeat() {
	if r == nil {
		return
	}
	r.mu.Lock()
	stop, done := r.stop, r.done
	r.stop, r.done = nil, nil
	r.mu.Unlock()
	if stop != nil {
		close(stop)
		<-done
	}
}

func isTerminalPhase(phase update.RunnerPhase) bool {
	return phase == update.RunnerPhaseSuccess || phase == update.RunnerPhaseFailed || phase == update.RunnerPhaseRollback
}

// Finalize 将受控 receipt 附加到最终快照，供新 server 只读结算终态历史。
func (r *stateReporter) Finalize(receipt update.RunnerReceipt) error {
	if r == nil || r.store == nil || r.input.OperationID == "" {
		return errors.New("runner state reporter is unavailable")
	}
	r.StopHeartbeat()
	r.mu.Lock()
	defer r.mu.Unlock()
	receipt.RunnerID = r.runnerID
	phase, progress, message, failure := r.current.Phase, r.current.Progress, r.current.Message, r.current.Error
	if receipt.Succeeded {
		phase, progress, message, failure = update.RunnerPhaseSuccess, 100, "update_completed", ""
	} else if receipt.RecoveryCompleted {
		phase, progress, message, failure = update.RunnerPhaseRollback, 100, "rollback_completed", receipt.FailureCode
	} else if receipt.FailureCode != "" {
		phase, progress, message, failure = update.RunnerPhaseFailed, 100, "update_failed", receipt.FailureCode
	}
	next := update.NewRunnerState(r.input, r.runnerID, phase, progress, message, failure, r.current)
	next.Receipt = &receipt
	if err := r.store.Write(next); err != nil {
		return err
	}
	r.current = next
	return nil
}

func (a *actions) Backup(ctx context.Context, in update.RunnerInput) error {
	stagingRoot := filepath.Join(in.Preflight.ComposeRoot, ".graft-update", "backups", in.OperationID)
	if err := os.MkdirAll(stagingRoot, directoryPermission); err != nil {
		return backupFailure(update.RunnerBackupFailureStageArtifactDirectory, err)
	}
	if err := copyFile(filepath.Join(in.Preflight.ComposeRoot, ".env"), filepath.Join(stagingRoot, "config.snapshot")); err != nil {
		return backupFailure(update.RunnerBackupFailureStageConfigSnapshot, err)
	}
	// #nosec G304 -- root derives from a preflight-validated host compose root and operation ID.
	dump, err := os.OpenFile(filepath.Join(stagingRoot, "database.dump"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, privateFilePermission)
	if err != nil {
		return backupFailure(update.RunnerBackupFailureStageDatabaseDump, err)
	}
	args := append([]string{"compose", "--env-file", ".env"}, composeFileArgs(in.Preflight.ComposeFiles)...)
	args = append(args, "exec", "-T", "postgres", "sh", "-ec", "pg_dump -U \"$POSTGRES_USER\" \"$POSTGRES_DB\"")
	// #nosec G204 -- this fixed command has no caller-provided executable or arguments.
	command := exec.CommandContext(ctx, "docker", args...)
	command.Dir, command.Stdout, command.Stderr = in.Preflight.ComposeRoot, dump, os.Stderr
	err = command.Run()
	closeErr := dump.Close()
	if err != nil {
		return update.NewRunnerBackupFailure(update.RunnerBackupFailureStageDatabaseDump, update.RunnerBackupFailureDetailCommandFailed, err)
	}
	if closeErr != nil {
		return backupFailure(update.RunnerBackupFailureStageDatabaseDump, closeErr)
	}
	configHash, configSize, err := digest(filepath.Join(stagingRoot, "config.snapshot"))
	if err != nil {
		return backupFailure(update.RunnerBackupFailureStageArtifactDigest, err)
	}
	dumpHash, dumpSize, err := digest(filepath.Join(stagingRoot, "database.dump"))
	if err != nil {
		return backupFailure(update.RunnerBackupFailureStageArtifactDigest, err)
	}
	if err := transferBackupArtifacts(ctx, in, stagingRoot); err != nil {
		return err
	}
	a.backup = moduleapi.CompleteBackupRunnerHandoffInput{OperationID: in.OperationID, TaskID: in.TaskID, ConfigSnapshotSHA256: configHash, ConfigSnapshotBytes: configSize, DatabaseDumpSHA256: dumpHash, DatabaseDumpBytes: dumpSize}
	return nil
}

// transferBackupArtifacts 将临时快照送入 server 容器的受控工件挂载；目标由服务端配置和 RunnerInput 冻结，
// runner 只执行固定 Compose 命令，不能接受调用方命令或任意目标路径。
func transferBackupArtifacts(ctx context.Context, in update.RunnerInput, stagingRoot string) error {
	if err := compose(ctx, in, "exec", "-T", "--user", "0:0", "server", "mkdir", "-p", in.BackupArtifactRoot); err != nil {
		return update.NewRunnerBackupFailure(update.RunnerBackupFailureStageArtifactDirectory, update.RunnerBackupFailureDetailCommandFailed, err)
	}
	for _, name := range []string{"config.snapshot", "database.dump"} {
		if err := compose(ctx, in, "cp", filepath.Join(stagingRoot, name), "server:"+filepath.Join(in.BackupArtifactRoot, name)); err != nil {
			return update.NewRunnerBackupFailure(update.RunnerBackupFailureStageArtifactDirectory, update.RunnerBackupFailureDetailCommandFailed, err)
		}
	}
	if err := compose(ctx, in, "exec", "-T", "--user", "0:0", "server", "chown", "-R", "10001:10001", in.BackupArtifactRoot); err != nil {
		return update.NewRunnerBackupFailure(update.RunnerBackupFailureStageArtifactDirectory, update.RunnerBackupFailureDetailCommandFailed, err)
	}
	if err := compose(ctx, in, "exec", "-T", "--user", "0:0", "server", "chmod", "0700", in.BackupArtifactRoot); err != nil {
		return update.NewRunnerBackupFailure(update.RunnerBackupFailureStageArtifactDirectory, update.RunnerBackupFailureDetailCommandFailed, err)
	}
	if err := compose(ctx, in, "exec", "-T", "--user", "0:0", "server", "chmod", "0600", filepath.Join(in.BackupArtifactRoot, "config.snapshot"), filepath.Join(in.BackupArtifactRoot, "database.dump")); err != nil {
		return update.NewRunnerBackupFailure(update.RunnerBackupFailureStageArtifactDirectory, update.RunnerBackupFailureDetailCommandFailed, err)
	}
	return nil
}

func backupFailure(stage update.RunnerBackupFailureStage, err error) error {
	detail := update.RunnerBackupFailureDetailIOFailed
	if errors.Is(err, os.ErrPermission) {
		detail = update.RunnerBackupFailureDetailPermissionDenied
	}
	return update.NewRunnerBackupFailure(stage, detail, err)
}
func (a *actions) BackupReceipt() moduleapi.CompleteBackupRunnerHandoffInput { return a.backup }
func (a *actions) Pull(ctx context.Context, in update.RunnerInput) error {
	if err := replaceRefs(filepath.Join(in.Preflight.ComposeRoot, ".env"), in.Preflight); err != nil {
		return err
	}
	return compose(ctx, in, "pull")
}
func (a *actions) StopServices(ctx context.Context, in update.RunnerInput) error {
	return compose(ctx, in, "stop", "server", "web")
}
func (a *actions) VerifyImages(ctx context.Context, in update.RunnerInput) error {
	if err := verifyImageDigest(ctx, in.Preflight.ServerReference, in.Preflight.ServerDigest); err != nil {
		return fmt.Errorf("verify server image: %w", err)
	}
	if err := verifyImageDigest(ctx, in.Preflight.WebReference, in.Preflight.WebDigest); err != nil {
		return fmt.Errorf("verify web image: %w", err)
	}
	return nil
}
func (a *actions) BootstrapMigrate(ctx context.Context, in update.RunnerInput) error {
	return compose(ctx, in, "run", "--rm", "bootstrap")
}
func (a *actions) Recreate(ctx context.Context, in update.RunnerInput) error {
	return compose(ctx, in, "up", "-d", "--no-deps", "--force-recreate", "server", "web")
}
func (a *actions) DockerHealth(ctx context.Context, in update.RunnerInput) error {
	return compose(ctx, in, "ps", "--status", "running", "--services", "server", "web")
}
func (a *actions) Healthz(ctx context.Context, in update.RunnerInput) error {
	return compose(ctx, in, healthzArgs()...)
}

func healthzArgs() []string {
	return []string{"exec", "-T", "server", "curl", "--fail", "--silent", "--max-time", healthzCurlTimeoutSeconds, "http://127.0.0.1:8080/healthz"}
}
func (a *actions) RecoverPreMigration(ctx context.Context, in update.RunnerInput) error {
	root := filepath.Join(in.Preflight.ComposeRoot, ".graft-update", "backups", in.OperationID)
	if err := copyFile(filepath.Join(root, "config.snapshot"), filepath.Join(in.Preflight.ComposeRoot, ".env")); err != nil {
		return err
	}
	return compose(ctx, in, "up", "-d", "--no-deps", "--force-recreate", "server", "web")
}
func compose(ctx context.Context, in update.RunnerInput, args ...string) error {
	commandArgs := append([]string{"compose", "--env-file", ".env"}, composeFileArgs(in.Preflight.ComposeFiles)...)
	// #nosec G204 -- callers select only fixed runner lifecycle argument sets.
	command := exec.CommandContext(ctx, "docker", append(commandArgs, args...)...)
	command.Dir, command.Stdout, command.Stderr = in.Preflight.ComposeRoot, os.Stdout, os.Stderr
	tag, err := sharedOfficialTag(in.Preflight.ServerReference, in.Preflight.WebReference, in.Preflight.OfficialServerImage, in.Preflight.OfficialWebImage)
	if err != nil {
		return err
	}
	command.Env = append(os.Environ(), "GRAFT_IMAGE_TAG="+tag)
	return command.Run()
}

func composeFileArgs(files []string) []string {
	args := make([]string, 0, len(files)*composeFileArgumentCapacityMultiplier)
	for _, file := range files {
		args = append(args, "-f", file)
	}
	return args
}
func copyFile(source, target string) error {
	// #nosec G304 -- source and target are derived from the preflight-validated compose root.
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	// #nosec G304 -- source and target are derived from the preflight-validated compose root.
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, privateFilePermission)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
func digest(path string) (string, int64, error) {
	// #nosec G304 -- path is derived from the preflight-validated compose root.
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

// replaceRefs 只推进固定版本部署；跟随频道的部署必须保留其频道标签。
//
//nolint:cyclop // 两个镜像与官方仓库必须共同校验，拆分会破坏原子写入边界。
func replaceRefs(path string, preflight update.ComposePreflight) error {
	if preflight.DeploymentStrategy == update.DeploymentStrategyStableTracking || preflight.DeploymentStrategy == update.DeploymentStrategyBetaTracking {
		return nil
	}
	if preflight.DeploymentStrategy != update.DeploymentStrategyPinnedStable && preflight.DeploymentStrategy != update.DeploymentStrategyPinnedBeta {
		return errors.New("compose runner deployment strategy is invalid")
	}
	tag, err := sharedOfficialTag(preflight.ServerReference, preflight.WebReference, preflight.OfficialServerImage, preflight.OfficialWebImage)
	if err != nil {
		return err
	}
	// #nosec G304 -- .env path is derived from the preflight-validated compose root.
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	values := map[string]string{"GRAFT_IMAGE_TAG": tag}
	lines := strings.Split(string(contents), "\n")
	for index, line := range lines {
		for key, value := range values {
			if strings.HasPrefix(line, key+"=") {
				lines[index] = key + "=" + value
				delete(values, key)
			}
		}
	}
	if len(values) != 0 {
		return errors.New("official compose environment lacks the shared image tag")
	}
	temporary := path + ".graft-update-tmp"
	// #nosec G703 -- temporary is a fixed suffix under the preflight-validated compose root.
	if err := os.WriteFile(temporary, []byte(strings.Join(lines, "\n")), privateFilePermission); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func sharedOfficialTag(server, web, officialServer, officialWeb string) (string, error) {
	serverTag, ok := referenceTag(server, officialServer)
	if !ok {
		return "", errors.New("server image must use the official repository with an explicit version tag")
	}
	webTag, ok := referenceTag(web, officialWeb)
	if !ok {
		return "", errors.New("web image must use the official repository with an explicit version tag")
	}
	if serverTag != webTag {
		return "", errors.New("server and web image references must use the same version tag")
	}
	return serverTag, nil
}

func referenceTag(reference, officialImage string) (string, bool) {
	if reference == "" || officialImage == "" || strings.TrimSpace(reference) != reference || strings.TrimSpace(officialImage) != officialImage || strings.Contains(reference, "@") || strings.ContainsAny(reference, " \t\r\n") {
		return "", false
	}
	prefix := officialImage + ":"
	if !strings.HasPrefix(reference, prefix) {
		return "", false
	}
	tag := strings.TrimPrefix(reference, prefix)
	return tag, tag != "latest" && validImageTag(tag)
}

func validImageTag(tag string) bool {
	return imageTagPattern.MatchString(tag)
}

func verifyImageDigest(ctx context.Context, reference, wantDigest string) error {
	if !referenceHasExplicitTag(reference) || !immutableDigest(wantDigest) {
		return errors.New("image verification input is invalid")
	}
	// #nosec G204 -- 引用来自经 manifest 校验的官方明确镜像标签。
	command := exec.CommandContext(ctx, "docker", "image", "inspect", "--format", "{{json .RepoDigests}}", reference)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	contents, err := command.Output()
	if err != nil {
		if summary := strings.TrimSpace(stderr.String()); summary != "" {
			return fmt.Errorf("inspect pulled image: %w: %.256s", err, summary)
		}
		return err
	}
	var repoDigests []string
	if err := json.Unmarshal(contents, &repoDigests); err != nil {
		return fmt.Errorf("decode pulled image digests: %w", err)
	}
	if containsVerifiedRepoDigest(repoDigests, reference, wantDigest) {
		return nil
	}
	repository := reference[:strings.LastIndex(reference, ":")]
	if summary := strings.TrimSpace(stderr.String()); summary != "" {
		return fmt.Errorf("pulled image does not include verified digest %q: %.256s", repository+"@"+wantDigest, summary)
	}
	return fmt.Errorf("pulled image does not include verified digest %q", repository+"@"+wantDigest)
}

func referenceHasExplicitTag(reference string) bool {
	if reference == "" || strings.TrimSpace(reference) != reference || strings.Contains(reference, "@") || strings.ContainsAny(reference, " \t\r\n") {
		return false
	}
	lastSlash := strings.LastIndex(reference, "/")
	lastColon := strings.LastIndex(reference, ":")
	return lastColon > lastSlash && lastColon < len(reference)-1 && reference[lastColon+1:] != "latest"
}

func containsVerifiedRepoDigest(repoDigests []string, reference, wantDigest string) bool {
	repository := reference[:strings.LastIndex(reference, ":")]
	want := repository + "@" + wantDigest
	for _, actual := range repoDigests {
		if actual == want {
			return true
		}
	}
	return false
}

func immutableDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
