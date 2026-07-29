package update

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/event"
	"graft/server/internal/httpx"
	"graft/server/internal/logger"
	"graft/server/internal/moduleapi"
)

// RolloutService 只编排人工确认的 Compose 更新；它保存 Update operation，Task 和 Backup 仍是外部能力。
type RolloutService struct {
	discovery         *Service
	operations        OperationStore
	diagnostics       FailureDiagnosticStore
	coordinator       *ComposeExecutionCoordinator
	launcher          ComposeRunnerLauncher
	newOperation      func() string
	auditPublisher    event.Publisher
	logger            *zap.Logger
	appLogger         logger.AppLogger
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
	errRolloutInvalidArgument          = errors.New("compose update request is invalid")
	errRolloutCatalogStale             = errors.New("compose update release catalog is stale")
	errRolloutInstallationUnavailable  = errors.New("compose update installation is unavailable")
	errRolloutSourceVersionUnsupported = errors.New("compose update source version is unsupported")
	errRolloutComposeCandidateInvalid  = errors.New("compose update candidate is invalid")
	errRolloutComposePreflightFailed   = errors.New("compose update preflight failed")
)

// SetAuditPublisher 注入 durable 审计事件发布端和失败日志器；Update 只发布领域证据，审计事实仍由 Audit 模块拥有。
func (s *RolloutService) SetAuditPublisher(publisher event.Publisher, logger *zap.Logger) {
	if s == nil {
		return
	}
	s.auditPublisher = publisher
	s.logger = logger
}

// SetAppLogger 注入 canonical App Log writer；runner 终态失败必须保留原始 request ID 供应用日志查询。
func (s *RolloutService) SetAppLogger(appLogger logger.AppLogger) {
	if s != nil {
		s.appLogger = appLogger
	}
}

func newOperationID() string { return fmt.Sprintf("update-%d", time.Now().UTC().UnixNano()) }

// NewRolloutService 组合已注册的窄 capability 与受限 Docker launcher。
func NewRolloutService(discovery *Service, operations OperationStore, tasks moduleapi.TaskService, backups moduleapi.BackupService, launcher ComposeRunnerLauncher) *RolloutService {
	return &RolloutService{discovery: discovery, operations: operations, coordinator: NewComposeExecutionCoordinator(tasks, backups), launcher: launcher, newOperation: newOperationID}
}

// SetFailureDiagnosticStore 注入 operation 终态使用的受控诊断存储；runner 的原始输出不进入该边界。
func (s *RolloutService) SetFailureDiagnosticStore(store FailureDiagnosticStore) {
	if s != nil {
		s.diagnostics = store
	}
}

// Start 只接受当前 catalog 中已验证的候选版本，随后仅启动一次 digest-pinned runner。
//
//nolint:cyclop // 版本、候选、镜像和跨模块 handoff 各自对应独立的升级安全门。
func (s *RolloutService) Start(ctx context.Context, requestedBy uint64, targetVersion string, candidateKeys ...string) (ComposeUpdateOperation, error) {
	if s == nil || s.discovery == nil || s.operations == nil || s.coordinator == nil || s.launcher == nil || requestedBy == 0 {
		return ComposeUpdateOperation{}, newRolloutStartFailure(rolloutFailureOperationStartFailed, "availability", "", errors.New("compose update rollout is unavailable"))
	}
	candidateKey := ""
	requestedPolicy := ""
	if len(candidateKeys) > 0 {
		candidateKey = strings.TrimSpace(candidateKeys[0])
	}
	if len(candidateKeys) > 1 {
		requestedPolicy = strings.TrimSpace(candidateKeys[1])
	}
	status, preflight, policy, err := s.confirmedPreflight(targetVersion, candidateKey, requestedPolicy)
	if err != nil {
		code := classifyRolloutPreflightFailure(err)
		return ComposeUpdateOperation{}, newRolloutStartFailure(code, "preflight", "", err)
	}
	operation := ComposeUpdateOperation{OperationID: s.newOperation(), RequestID: rolloutRequestID(ctx), SourceVersion: status.CurrentVersion, TargetVersion: targetVersion, UpdatePolicy: policy, RequestedBy: requestedBy, Outcome: ExecutionOutcomePlanning}
	handoff := backupHandoff(operation.OperationID, requestedBy, preflight.ComposeRoot)
	prepared, input, err := s.coordinator.Start(ctx, operation, requestedBy, handoff)
	if err != nil {
		return ComposeUpdateOperation{}, newRolloutStartFailure(rolloutFailureOperationStartFailed, "handoff", operation.OperationID, err)
	}
	prepared.RequestedBy, prepared.Outcome = requestedBy, ExecutionOutcomePulling
	input.Preflight = preflight
	if err := s.persistAndLaunch(ctx, prepared, input); err != nil {
		return ComposeUpdateOperation{}, err
	}
	return prepared, nil
}

//nolint:cyclop,gocognit,gocyclo,revive // 每个拒绝分支都是可独立审计的发布安全门。
func (s *RolloutService) confirmedPreflight(targetVersion, candidateKey, requestedPolicy string) (Status, ComposePreflight, UpdatePolicy, error) {
	status := s.discovery.Status()
	if status.CacheStale || strings.TrimSpace(status.CheckError) != "" {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: fresh verified release catalog is required", errRolloutCatalogStale)
	}
	if status.Profile.Capability != "compose_upgrade_available" || status.Latest == nil {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: rollout is unavailable for this installation", errRolloutInstallationUnavailable)
	}
	policy, initialized := configuredUpdatePolicy()
	if initialized && requestedPolicy != "" || !initialized && requestedPolicy == "" {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: update policy initialization is invalid", errRolloutInvalidArgument)
	}
	if !initialized {
		var ok bool
		policy, ok = parseUpdatePolicy(requestedPolicy)
		if !ok {
			return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: update policy is invalid", errRolloutInvalidArgument)
		}
	}
	if policy == UpdatePolicyManual {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: manual policy does not permit automated rollout", errRolloutInvalidArgument)
	}
	release, found := verifiedRelease(status.AvailableReleases, targetVersion)
	if !found {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: target version is not in the verified release catalog", errRolloutInvalidArgument)
	}
	if (policy == UpdatePolicyStable || policy == UpdatePolicyBeta) && release.Channel != string(policy) {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: target channel does not match update policy", errRolloutInvalidArgument)
	}
	if (policy == UpdatePolicyStable || policy == UpdatePolicyBeta) && (status.Latest == nil || release.Version != status.Latest.Version) {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: target version is not the policy-selected verified release", errRolloutInvalidArgument)
	}
	if minimum := strings.TrimSpace(release.MinimumSourceVersion); minimum != "" {
		current, currentErr := ParseVersion(status.CurrentVersion)
		minimumVersion, minimumErr := ParseVersion(minimum)
		if currentErr != nil || minimumErr != nil || current.Compare(minimumVersion) < 0 {
			return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: current version does not meet the target minimum source version", errRolloutSourceVersionUnsupported)
		}
	}
	preflight, err := composePreflight(status.Profile, release, policy, candidateKey)
	if err != nil {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: %w", errRolloutComposePreflightFailed, err)
	}
	return status, preflight, policy, nil
}

func verifiedRelease(catalog []Release, version string) (Release, bool) {
	for _, release := range catalog {
		if release.Version == strings.TrimSpace(version) {
			return release, true
		}
	}
	return Release{}, false
}

func classifyRolloutPreflightFailure(err error) string {
	switch {
	case errors.Is(err, errRolloutInvalidArgument):
		return rolloutFailureInvalidTarget
	case errors.Is(err, errRolloutCatalogStale):
		return rolloutFailureCatalogStale
	case errors.Is(err, errRolloutInstallationUnavailable):
		return rolloutFailureInstallationUnavailable
	case errors.Is(err, errRolloutSourceVersionUnsupported):
		return rolloutFailureSourceVersionUnsupported
	case errors.Is(err, errRolloutComposeCandidateInvalid):
		return rolloutFailureComposeCandidateInvalid
	default:
		return rolloutFailureComposePreflightFailed
	}
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
		return newRolloutStartFailure(rolloutFailureOperationStartFailed, "operation_persist", operation.OperationID, fmt.Errorf("persist update operation: %w", err))
	}
	if err := s.launcher.Launch(ctx, input); err != nil {
		operation.Outcome, operation.FailureCode = ExecutionOutcomeFailed, "runner_launch_failed"
		if cleanupErr := s.coordinator.CancelBeforeLaunch(ctx, operation); cleanupErr != nil {
			operation.FailureCode = "runner_launch_cleanup_failed"
			_ = s.operations.Settle(ctx, operation)
			return newRolloutStartFailure(rolloutFailureOperationStartFailed, "runner_launch", operation.OperationID, fmt.Errorf("launch compose update runner: %w; reconcile launch handoff: %w", err, cleanupErr))
		}
		_ = s.operations.Settle(ctx, operation)
		return newRolloutStartFailure(rolloutFailureOperationStartFailed, "runner_launch", operation.OperationID, fmt.Errorf("launch compose update runner: %w", err))
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
	s.persistTerminalFailureDiagnostic(ctx, settled, receipt)
	s.logTerminalFailure(ctx, settled, receipt)
	s.publishAudit(ctx, settled, settled.Outcome == ExecutionOutcomeSuccess, settled.FailureCode)
	return settled, nil
}

func (s *RolloutService) logTerminalFailure(ctx context.Context, operation ComposeUpdateOperation, receipt RunnerReceipt) {
	if s == nil || s.appLogger == nil || operation.RequestID == "" || (operation.Outcome != ExecutionOutcomeFailed && operation.Outcome != ExecutionOutcomeNeedsAttention) {
		return
	}
	s.appLogger.Named("modules.update.rollout").Error(ctx, runnerFailureDiagnosticSummary,
		logger.StringField(logger.FieldOperation, "platform_update.runner_terminal_failure"),
		logger.StringField(logger.FieldRequestID, operation.RequestID),
		logger.StringField("operation_id", operation.OperationID),
		logger.Uint64Field("task_id", operation.TaskID),
		logger.StringField("target_version", operation.TargetVersion),
		logger.StringField("failure_code", receipt.FailureCode),
		logger.StringField("status", string(operation.Outcome)),
	)
}

func rolloutRequestID(ctx context.Context) string {
	audit, ok := httpx.RequestAuditContextFromContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(audit.RequestID)
}

func (s *RolloutService) persistTerminalFailureDiagnostic(ctx context.Context, operation ComposeUpdateOperation, receipt RunnerReceipt) {
	if s == nil || s.diagnostics == nil || operation.RequestID == "" || operation.RequestedBy == 0 {
		return
	}
	if operation.Outcome != ExecutionOutcomeFailed && operation.Outcome != ExecutionOutcomeNeedsAttention {
		return
	}
	diagnostic := runnerTerminalFailureDiagnostic(operation, receipt)
	if err := s.diagnostics.CreateFailureDiagnostic(ctx, diagnostic, operation.RequestedBy); err != nil && s.logger != nil {
		s.logger.Error("persist terminal platform update failure diagnostic failed", zap.String("module", moduleID), zap.String("operation_id", operation.OperationID), zap.Error(err))
	}
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
	requestAudit, _ := httpx.RequestAuditContextFromContext(ctx)
	payload := moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Operator: operator, Action: "platform.update.compose", ResourceType: "platform_update", ResourceID: operation.OperationID, ResourceName: operation.TargetVersion, RequestID: requestAudit.RequestID, RequestMethod: requestAudit.Method, RequestPath: requestAudit.Route, IP: requestAudit.ClientIP, UserAgent: requestAudit.UserAgent, StatusCode: http.StatusAccepted, Success: success, Message: strings.TrimSpace(message), Metadata: map[string]any{"source_version": operation.SourceVersion, "target_version": operation.TargetVersion, "task_id": operation.TaskID, "status": operation.Outcome}, CreatedAt: time.Now().UTC()}
	envelope, err := httpx.NewAuditEvent(moduleID, payload)
	if err != nil {
		s.logAuditPublishFailure(operation, err)
		return
	}
	if _, err := s.auditPublisher.Publish(ctx, envelope, event.PublishOptions{Delivery: event.DeliveryDurable}); err != nil {
		s.logAuditPublishFailure(operation, err)
	}
}

func (s *RolloutService) logAuditPublishFailure(operation ComposeUpdateOperation, err error) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Error("publish update rollout audit event failed",
		zap.String("module", moduleID),
		zap.String("operation_id", operation.OperationID),
		zap.Error(err),
	)
}

func isTerminalOutcome(outcome ExecutionOutcome) bool {
	switch outcome {
	case ExecutionOutcomeSuccess, ExecutionOutcomeFailed, ExecutionOutcomeRecovered, ExecutionOutcomeNeedsAttention:
		return true
	default:
		return false
	}
}

// SettleAvailableReceipts 收敛带稳定协议 label 的保留 runner 容器日志；无法读取日志不影响 server 启动。
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

func composePreflight(profile InstallationProfile, release Release, policy UpdatePolicy, candidateKey string) (ComposePreflight, error) {
	root := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_COMPOSE_ROOT"))
	explicitRoot := root != ""
	composeFiles := []string{}
	if root == "" {
		root, composeFiles = confirmedComposeCandidate(profile, candidateKey)
		if root == "" {
			return ComposePreflight{}, fmt.Errorf("%w: a confirmed compose root candidate is required", errRolloutComposeCandidateInvalid)
		}
	}
	if len(composeFiles) == 0 {
		if !explicitRoot {
			return ComposePreflight{}, fmt.Errorf("%w: the confirmed compose root candidate must include its compose file sequence", errRolloutComposeCandidateInvalid)
		}
		composeFiles = []string{filepath.Join(root, "compose.yml")}
	}
	value := ComposePreflight{DeclaredMode: profile.DeclaredMode, UpdatePolicy: policy, DetectedMode: profile.DetectedMode, ComposeRoot: root, Platform: "linux/amd64", DockerSocket: "/var/run/docker.sock", ComposeFiles: append([]string(nil), composeFiles...), BundledPostgres: true, OfficialServerImage: release.ServerImage, OfficialWebImage: release.WebImage, OfficialRunnerImage: release.RunnerImage, ServerDigest: release.ServerDigest, WebDigest: release.WebDigest, RunnerDigest: release.RunnerDigest, ServerReference: release.ServerImage + ":" + release.Version, WebReference: release.WebImage + ":" + release.Version, RunnerReference: release.RunnerRef}
	if err := ValidateComposePreflight(value); err != nil {
		return ComposePreflight{}, fmt.Errorf("preflight official compose rollout: %w", err)
	}
	return value, nil
}

func confirmedComposeCandidate(profile InstallationProfile, candidateKey string) (string, []string) {
	if !profile.ComposeRootConfirmationRequired && len(profile.ComposeCandidates) == 1 && profile.ComposeCandidates[0].Confidence == "high" && strings.TrimSpace(candidateKey) == "" {
		candidateKey = profile.ComposeCandidates[0].CandidateKey
	}
	for _, candidate := range profile.ComposeCandidates {
		if candidate.CandidateKey == strings.TrimSpace(candidateKey) {
			return strings.TrimSpace(candidate.Root), append([]string(nil), candidate.ConfigFiles...)
		}
	}
	return "", nil
}

func backupHandoff(operationID string, requestedBy uint64, root string) moduleapi.BackupRunnerHandoffPlan {
	artifactRoot := filepath.Join(root, ".graft-update", "backups", operationID)
	creator := requestedBy
	return moduleapi.BackupRunnerHandoffPlan{OperationID: operationID, Purpose: "before-update", RetainUntil: time.Now().UTC().Add(30 * 24 * time.Hour), CreatedBy: &creator, ArtifactRoot: artifactRoot, ConfigSnapshotRef: filepath.Join(artifactRoot, "config.snapshot"), DatabaseDumpRef: filepath.Join(artifactRoot, "database.dump")}
}
