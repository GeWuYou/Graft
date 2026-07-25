package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"graft/server/internal/event"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
)

// RolloutService 只编排人工确认的 Compose 更新；它保存 Update operation，Task 和 Backup 仍是外部能力。
type RolloutService struct {
	discovery         *Service
	operations        OperationStore
	coordinator       *ComposeExecutionCoordinator
	launcher          ComposeRunnerLauncher
	newOperation      func() string
	auditPublisher    event.Publisher
	receiptPollMu     sync.Mutex
	receiptPollCancel context.CancelFunc
	receiptPollDone   chan struct{}
	receiptPollClosed bool
	receiptPollEvery  time.Duration
}

const receiptPollInterval = 15 * time.Second

type receiptPolling struct {
	reader   ComposeRunnerReceiptReader
	ctx      context.Context
	done     chan struct{}
	interval time.Duration
}

var (
	errRolloutInvalidArgument = errors.New("compose update request is invalid")
	errRolloutPrecondition    = errors.New("compose update precondition is not met")
)

// SetAuditPublisher 注入 durable 审计事件发布端；Update 只发布领域证据，审计事实仍由 Audit 模块拥有。
func (s *RolloutService) SetAuditPublisher(publisher event.Publisher) { s.auditPublisher = publisher }

func newOperationID() string { return fmt.Sprintf("update-%d", time.Now().UTC().UnixNano()) }

// NewRolloutService 组合已注册的窄 capability 与受限 Docker launcher。
func NewRolloutService(discovery *Service, operations OperationStore, tasks moduleapi.TaskService, backups moduleapi.BackupService, launcher ComposeRunnerLauncher) *RolloutService {
	return &RolloutService{discovery: discovery, operations: operations, coordinator: NewComposeExecutionCoordinator(tasks, backups), launcher: launcher, newOperation: newOperationID}
}

// Start 只接受当前 catalog 中已验证的候选版本，随后仅启动一次 digest-pinned runner。
//
//nolint:cyclop // 版本、候选、镜像和跨模块 handoff 各自对应独立的升级安全门。
func (s *RolloutService) Start(ctx context.Context, requestedBy uint64, targetVersion string, candidateKeys ...string) (ComposeUpdateOperation, error) {
	if s == nil || s.discovery == nil || s.operations == nil || s.coordinator == nil || s.launcher == nil || requestedBy == 0 {
		return ComposeUpdateOperation{}, errors.New("compose update rollout is unavailable")
	}
	candidateKey := ""
	if len(candidateKeys) > 0 {
		candidateKey = strings.TrimSpace(candidateKeys[0])
	}
	status, preflight, err := s.confirmedPreflight(targetVersion, candidateKey)
	if err != nil {
		return ComposeUpdateOperation{}, err
	}
	operation := ComposeUpdateOperation{OperationID: s.newOperation(), SourceVersion: status.CurrentVersion, TargetVersion: targetVersion, RequestedBy: requestedBy, Outcome: ExecutionOutcomePlanning}
	handoff := backupHandoff(operation.OperationID, requestedBy, preflight.ComposeRoot)
	prepared, input, err := s.coordinator.Start(ctx, operation, requestedBy, handoff)
	if err != nil {
		return ComposeUpdateOperation{}, err
	}
	prepared.RequestedBy, prepared.Outcome = requestedBy, ExecutionOutcomePulling
	input.Preflight = preflight
	if err := s.persistAndLaunch(ctx, prepared, input); err != nil {
		return ComposeUpdateOperation{}, err
	}
	return prepared, nil
}

//nolint:cyclop // Each rejection is an independently auditable rollout safety gate.
func (s *RolloutService) confirmedPreflight(targetVersion, candidateKey string) (Status, ComposePreflight, error) {
	status := s.discovery.Status()
	if status.CacheStale || strings.TrimSpace(status.CheckError) != "" {
		return Status{}, ComposePreflight{}, fmt.Errorf("%w: fresh verified release catalog is required", errRolloutPrecondition)
	}
	if status.Profile.Capability != "compose_upgrade_available" || status.Latest == nil {
		return Status{}, ComposePreflight{}, fmt.Errorf("%w: rollout is unavailable for this installation", errRolloutPrecondition)
	}
	if minimum := strings.TrimSpace(status.Latest.MinimumSourceVersion); minimum != "" {
		current, currentErr := ParseVersion(status.CurrentVersion)
		minimumVersion, minimumErr := ParseVersion(minimum)
		if currentErr != nil || minimumErr != nil || current.Compare(minimumVersion) < 0 {
			return Status{}, ComposePreflight{}, fmt.Errorf("%w: current version does not meet the target minimum source version", errRolloutPrecondition)
		}
	}
	if strings.TrimSpace(targetVersion) == "" || targetVersion != status.Latest.Version {
		return Status{}, ComposePreflight{}, fmt.Errorf("%w: target version is not the currently verified release", errRolloutInvalidArgument)
	}
	preflight, err := composePreflight(status.Profile, *status.Latest, candidateKey)
	if err != nil {
		return Status{}, ComposePreflight{}, fmt.Errorf("%w: %w", errRolloutPrecondition, err)
	}
	return status, preflight, nil
}

// Close 释放 rollout 持有的 Docker client，供模块关闭阶段调用。
func (s *RolloutService) Close() error {
	if s == nil {
		return nil
	}
	s.receiptPollMu.Lock()
	s.receiptPollClosed = true
	cancel := s.receiptPollCancel
	done := s.receiptPollDone
	s.receiptPollCancel = nil
	s.receiptPollDone = nil
	s.receiptPollMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	if s.launcher == nil {
		return nil
	}
	return s.launcher.Close()
}

// StartReceiptPolling 在 server 重建后持续读取保留 runner 的日志回执。
// 当 service、launcher 或 receipt reader 不可用，或 service 已关闭时，此方法无副作用；否则会启动一个后台
// goroutine，按固定间隔读取并结算回执。调用 Close 会取消 polling context，并等待该 goroutine 退出后再关闭 launcher。
// 传入 ctx 被取消时，后台 goroutine 也会随之退出。
func (s *RolloutService) StartReceiptPolling(ctx context.Context) {
	polling, ok := s.prepareReceiptPolling(ctx)
	if !ok {
		return
	}
	go func() {
		s.runReceiptPolling(polling.ctx, polling)
	}()
}

func (s *RolloutService) prepareReceiptPolling(ctx context.Context) (receiptPolling, bool) {
	if s == nil {
		return receiptPolling{}, false
	}
	reader, ok := s.launcher.(ComposeRunnerReceiptReader)
	if !ok {
		return receiptPolling{}, false
	}
	pollCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.receiptPollMu.Lock()
	if s.receiptPollClosed || s.receiptPollCancel != nil {
		s.receiptPollMu.Unlock()
		cancel()
		return receiptPolling{}, false
	}
	s.receiptPollCancel = cancel
	s.receiptPollDone = done
	pollEvery := s.receiptPollEvery
	if pollEvery <= 0 {
		pollEvery = receiptPollInterval
	}
	s.receiptPollMu.Unlock()
	return receiptPolling{reader: reader, ctx: pollCtx, done: done, interval: pollEvery}, true
}

func (s *RolloutService) runReceiptPolling(ctx context.Context, polling receiptPolling) {
	defer close(polling.done)
	ticker := time.NewTicker(polling.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if receipts, err := polling.reader.ReadRunnerReceipts(ctx); err == nil {
				for _, receipt := range receipts {
					_, _ = s.settleReceiptAndCleanup(ctx, receipt)
				}
			}
		}
	}
}

func (s *RolloutService) persistAndLaunch(ctx context.Context, operation ComposeUpdateOperation, input RunnerInput) error {
	if err := s.operations.Create(ctx, operation); err != nil {
		s.publishAudit(ctx, operation, false, "operation_persist_failed")
		return err
	}
	if err := s.launcher.Launch(ctx, input); err != nil {
		operation.Outcome, operation.FailureCode = ExecutionOutcomeFailed, "runner_launch_failed"
		if cleanupErr := s.coordinator.CancelBeforeLaunch(ctx, operation); cleanupErr != nil {
			operation.FailureCode = "runner_launch_cleanup_failed"
			_ = s.operations.Settle(ctx, operation)
			s.publishAudit(ctx, operation, false, operation.FailureCode)
			return fmt.Errorf("launch compose update runner: %w; reconcile launch handoff: %w", err, cleanupErr)
		}
		_ = s.operations.Settle(ctx, operation)
		s.publishAudit(ctx, operation, false, operation.FailureCode)
		return fmt.Errorf("launch compose update runner: %w", err)
	}
	s.publishAudit(ctx, operation, true, "")
	return nil
}

// SettlePersistedReceipt 由目标 server 启动时调用，读取受限 receipt 后依序结算 Backup、Task 和 Update operation。
func (s *RolloutService) SettlePersistedReceipt(ctx context.Context, receipt RunnerReceipt) (ComposeUpdateOperation, error) {
	if s == nil || s.operations == nil || s.coordinator == nil || !runnerOperationID.MatchString(receipt.OperationID) {
		return ComposeUpdateOperation{}, errors.New("compose update receipt settlement is unavailable")
	}
	operation, err := s.operations.Get(ctx, receipt.OperationID)
	if err != nil {
		return ComposeUpdateOperation{}, err
	}
	if isTerminalOutcome(operation.Outcome) {
		return operation, nil
	}
	settled, err := s.coordinator.SettleReceipt(ctx, operation, receipt)
	if err != nil {
		s.publishAudit(ctx, operation, false, "receipt_settlement_failed")
		return ComposeUpdateOperation{}, err
	}
	if err := s.operations.Settle(ctx, settled); err != nil {
		s.publishAudit(ctx, settled, false, "operation_settlement_failed")
		return ComposeUpdateOperation{}, err
	}
	s.publishAudit(ctx, settled, settled.Outcome == ExecutionOutcomeSuccess, settled.FailureCode)
	return settled, nil
}

func (s *RolloutService) publishAudit(ctx context.Context, operation ComposeUpdateOperation, success bool, message string) {
	if s == nil || s.auditPublisher == nil {
		return
	}
	requestAuth, _ := moduleapi.RequestAuthContextFromContext(ctx)
	var operator *moduleapi.CurrentUser
	if requestAuth.User != nil {
		operator = requestAuth.User
	}
	payload := moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Operator: operator, Action: "platform.update.compose", ResourceType: "platform_update", ResourceID: operation.OperationID, ResourceName: operation.TargetVersion, StatusCode: http.StatusAccepted, Success: success, Message: strings.TrimSpace(message), Metadata: map[string]any{"source_version": operation.SourceVersion, "target_version": operation.TargetVersion, "task_id": operation.TaskID, "status": operation.Outcome}, CreatedAt: time.Now().UTC()}
	envelope, err := httpx.NewAuditEvent(moduleID, payload)
	if err != nil {
		return
	}
	_, _ = s.auditPublisher.Publish(ctx, envelope, event.PublishOptions{Delivery: event.DeliveryDurable})
}

func isTerminalOutcome(outcome ExecutionOutcome) bool {
	switch outcome {
	case ExecutionOutcomeSuccess, ExecutionOutcomeFailed, ExecutionOutcomeRecovered, ExecutionOutcomeNeedsAttention:
		return true
	default:
		return false
	}
}

// SettleAvailableReceipts 收敛目标 Compose 根目录下由 runner 留下的 receipt；无法解析的文件不影响 server 启动。
//
//nolint:cyclop // 日志 receipt 与 legacy durable receipt 是两个独立的恢复来源。
func (s *RolloutService) SettleAvailableReceipts(ctx context.Context) error {
	if s == nil || s.discovery == nil {
		return nil
	}
	if reader, ok := s.launcher.(ComposeRunnerReceiptReader); ok {
		if receipts, err := reader.ReadRunnerReceipts(ctx); err == nil {
			for _, receipt := range receipts {
				_, _ = s.settleReceiptAndCleanup(ctx, receipt)
			}
		}
	}
	profile := s.discovery.Status().Profile
	if profile.DetectedMode != "compose" {
		return nil
	}
	root := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_COMPOSE_ROOT"))
	if !filepath.IsAbs(root) {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(root, runnerReceiptDirectory))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read compose runner receipts: %w", err)
	}
	for _, entry := range entries {
		if err := s.settleReceiptEntry(ctx, root, entry); err != nil {
			return err
		}
	}
	return nil
}

func (s *RolloutService) settleReceiptEntry(ctx context.Context, root string, entry os.DirEntry) error {
	if !validRunnerReceiptEntry(entry) {
		return nil
	}
	path, receipt, err := readPersistedRunnerReceipt(root, entry.Name())
	if err != nil {
		return err
	}
	settled, err := s.settleReceiptAndCleanup(ctx, receipt)
	if err != nil && !errors.Is(err, errUpdateOperationNotFound) {
		return err
	}
	if err != nil || settled.Outcome != ExecutionOutcomeSuccess {
		return nil
	}
	// #nosec G703 -- path derives from a validated receipt filename below the configured absolute Compose root.
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove settled compose runner receipt: %w", err)
	}
	return nil
}

func (s *RolloutService) settleReceiptAndCleanup(ctx context.Context, receipt RunnerReceipt) (ComposeUpdateOperation, error) {
	settled, err := s.SettlePersistedReceipt(ctx, receipt)
	if err != nil {
		return ComposeUpdateOperation{}, err
	}
	if cleanup, ok := s.launcher.(ComposeRunnerReceiptCleanup); ok {
		_ = cleanup.RemoveRunner(ctx, receipt.OperationID)
	}
	return settled, nil
}

func validRunnerReceiptEntry(entry os.DirEntry) bool {
	if entry == nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || filepath.Base(entry.Name()) != entry.Name() || entry.Type()&os.ModeSymlink != 0 {
		return false
	}
	info, err := entry.Info()
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func readPersistedRunnerReceipt(root, name string) (string, RunnerReceipt, error) {
	path := filepath.Join(root, runnerReceiptDirectory, name)
	// #nosec G304,G703 -- name is a validated basename from the restricted receipt directory under the configured root.
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", RunnerReceipt{}, fmt.Errorf("read compose runner receipt: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return "", RunnerReceipt{}, fmt.Errorf("stat compose runner receipt: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", RunnerReceipt{}, errors.New("compose runner receipt is not a regular file")
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		return "", RunnerReceipt{}, fmt.Errorf("read compose runner receipt: %w", err)
	}
	var receipt RunnerReceipt
	if err := json.Unmarshal(contents, &receipt); err != nil {
		return "", RunnerReceipt{}, fmt.Errorf("decode compose runner receipt: %w", err)
	}
	return path, receipt, nil
}

func composePreflight(profile InstallationProfile, release Release, candidateKey string) (ComposePreflight, error) {
	root := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_COMPOSE_ROOT"))
	explicitRoot := root != ""
	composeFiles := []string{}
	if root == "" {
		for _, candidate := range profile.ComposeCandidates {
			if candidate.CandidateKey == strings.TrimSpace(candidateKey) {
				root = strings.TrimSpace(candidate.Root)
				composeFiles = append(composeFiles, candidate.ConfigFiles...)
				break
			}
		}
		if root == "" {
			return ComposePreflight{}, errors.New("a confirmed compose root candidate is required")
		}
	}
	if len(composeFiles) == 0 {
		if !explicitRoot {
			return ComposePreflight{}, errors.New("the confirmed compose root candidate must include its compose file sequence")
		}
		composeFiles = []string{filepath.Join(root, "compose.yml")}
	}
	value := ComposePreflight{DeclaredMode: profile.DeclaredMode, DetectedMode: profile.DetectedMode, ComposeRoot: root, Platform: "linux/amd64", DockerSocket: "/var/run/docker.sock", ComposeFiles: append([]string(nil), composeFiles...), BundledPostgres: true, OfficialServerImage: release.ServerImage, OfficialWebImage: release.WebImage, OfficialRunnerImage: release.RunnerImage, ServerDigest: release.ServerDigest, WebDigest: release.WebDigest, RunnerDigest: release.RunnerDigest, ServerReference: release.ServerRef, WebReference: release.WebRef, RunnerReference: release.RunnerRef}
	if err := ValidateComposePreflight(value); err != nil {
		return ComposePreflight{}, fmt.Errorf("preflight official compose rollout: %w", err)
	}
	return value, nil
}

func backupHandoff(operationID string, requestedBy uint64, root string) moduleapi.BackupRunnerHandoffPlan {
	artifactRoot := filepath.Join(root, ".graft-update", "backups", operationID)
	creator := requestedBy
	return moduleapi.BackupRunnerHandoffPlan{OperationID: operationID, Purpose: "before-update", RetainUntil: time.Now().UTC().Add(30 * 24 * time.Hour), CreatedBy: &creator, ArtifactRoot: artifactRoot, ConfigSnapshotRef: filepath.Join(artifactRoot, "config.snapshot"), DatabaseDumpRef: filepath.Join(artifactRoot, "database.dump")}
}
