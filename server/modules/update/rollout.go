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
	"graft/server/internal/realtime"
)

// RolloutService 只编排人工确认的 Compose 更新；它保存 Update operation，Task 和 Backup 仍是外部能力。
type RolloutService struct {
	discovery          *Service
	operations         OperationStore
	diagnostics        FailureDiagnosticStore
	coordinator        *ComposeExecutionCoordinator
	launcher           ComposeRunnerLauncher
	deployment         moduleapi.DeploymentRuntime
	newOperation       func() string
	auditPublisher     event.Publisher
	logger             *zap.Logger
	appLogger          logger.AppLogger
	realtime           realtime.Publisher
	backupArtifactRoot string
	stateStore         RunnerStateStore
	startMu            sync.Mutex
	statePollMu        sync.Mutex
	statePollCancel    context.CancelFunc
	statePollDone      chan struct{}
	statePollClosed    bool
	statePollEvery     time.Duration
	receiptPollMu      sync.Mutex
	receiptPollCancel  context.CancelFunc
	receiptPollDone    chan struct{}
	receiptPollClosed  bool
	receiptPollEvery   time.Duration
}

const receiptPollInterval = 15 * time.Second
const maxActiveOperationScan = 100

// runnerStatePollInterval 仅保持 server 投影的新鲜度，不构成 runner 执行心跳或生命周期租约。
const runnerStatePollInterval = 2 * time.Second

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
	errRolloutImageTagUnconfigured     = errors.New("compose image tag is unconfigured")
	errRolloutImageTagInvalid          = errors.New("compose image tag is invalid")
	errRolloutCandidateConfirmation    = errors.New("compose update candidate confirmation is required")
	errRolloutNoEligibleRelease        = errors.New("no eligible newer compose update release")
	errRolloutSourceVersionUnsupported = errors.New("compose update source version is unsupported")
	errRolloutComposeCandidateInvalid  = errors.New("compose update candidate is invalid")
	errRolloutComposePreflightFailed   = errors.New("compose update preflight failed")
	errRunnerStateUnavailable          = errors.New("runner state is unavailable")
	errActiveUpdateOperationNotFound   = errors.New("active update operation not found")
	errRecoveryConflict                = errors.New("runner recovery precondition is not met")
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

// SetBackupArtifactRoot 注入 Backup 模块在 server 容器内可见的唯一工件根目录。
func (s *RolloutService) SetBackupArtifactRoot(root string) {
	if s != nil {
		s.backupArtifactRoot = filepath.Clean(strings.TrimSpace(root))
	}
}

func newOperationID() string { return fmt.Sprintf("update-%d", time.Now().UTC().UnixNano()) }

// NewRolloutService 组合已注册的窄 capability 与受限 Docker launcher。
func NewRolloutService(discovery *Service, operations OperationStore, tasks moduleapi.TaskService, backups moduleapi.BackupService, launcher ComposeRunnerLauncher) *RolloutService {
	stateStore, _ := NewFileRunnerStateStore(RunnerStateRoot)
	return &RolloutService{discovery: discovery, operations: operations, coordinator: NewComposeExecutionCoordinator(tasks, backups), launcher: launcher, newOperation: newOperationID, stateStore: stateStore}
}

// SetRunnerStateStore 注入只读 runner 状态源；server 永不通过该边界写入生命周期阶段。
func (s *RolloutService) SetRunnerStateStore(store RunnerStateStore) {
	if s != nil {
		s.stateStore = store
	}
}

// SetDeploymentRuntime 注入唯一可解释部署事实的能力；每次升级启动前都必须重新冻结快照。
func (s *RolloutService) SetDeploymentRuntime(runtime moduleapi.DeploymentRuntime) {
	if s != nil {
		s.deployment = runtime
	}
}

// SetFailureDiagnosticStore 注入 operation 终态使用的受控诊断存储；runner 的原始输出不进入该边界。
func (s *RolloutService) SetFailureDiagnosticStore(store FailureDiagnosticStore) {
	if s != nil {
		s.diagnostics = store
	}
}

// StartRolloutInput 承载一次 Compose 更新启动的显式请求语义，避免候选 Compose 根与更新策略共享位置参数。
type StartRolloutInput struct {
	RequestedBy   uint64
	TargetVersion string
	CandidateKey  string
}

// Start 只接受当前 catalog 中已验证的候选版本，随后仅启动一次 digest-pinned runner。
//
//nolint:cyclop // 版本、候选、镜像和跨模块 handoff 各自对应独立的升级安全门。
func (s *RolloutService) Start(ctx context.Context, input StartRolloutInput) (ComposeUpdateOperation, error) {
	if s == nil || s.discovery == nil || s.operations == nil || s.coordinator == nil || s.launcher == nil || input.RequestedBy == 0 {
		return ComposeUpdateOperation{}, newRolloutStartFailure(rolloutFailureOperationStartFailed, "availability", "", errors.New("compose update rollout is unavailable"))
	}
	s.startMu.Lock()
	defer s.startMu.Unlock()
	if err := s.ensureNoActiveRunner(); err != nil {
		return ComposeUpdateOperation{}, newRolloutStartFailure(rolloutFailureOperationStartFailed, "runner_state", "", err)
	}
	candidateKey := strings.TrimSpace(input.CandidateKey)
	status, preflight, mode, err := s.confirmedPreflight(ctx, input.TargetVersion, candidateKey)
	if err != nil {
		code := classifyRolloutPreflightFailure(err)
		return ComposeUpdateOperation{}, newRolloutStartFailure(code, "preflight", "", err)
	}
	operationID := s.newOperation()
	operation := ComposeUpdateOperation{OperationID: operationID, RunnerID: "runner-" + strings.TrimPrefix(operationID, "update-"), RequestID: rolloutRequestID(ctx), SourceVersion: status.CurrentVersion, TargetVersion: input.TargetVersion, DeploymentStrategy: mode, RequestedBy: input.RequestedBy, Outcome: ExecutionOutcomePlanning}
	if !filepath.IsAbs(s.backupArtifactRoot) {
		return ComposeUpdateOperation{}, newRolloutStartFailure(rolloutFailureOperationStartFailed, "backup", operation.OperationID, errors.New("backup artifact root is unavailable"))
	}
	handoff := backupHandoff(operation.OperationID, input.RequestedBy, s.backupArtifactRoot)
	prepared, runnerInput, err := s.coordinator.Start(ctx, operation, input.RequestedBy, handoff)
	if err != nil {
		return ComposeUpdateOperation{}, newRolloutStartFailure(rolloutFailureOperationStartFailed, "handoff", operation.OperationID, err)
	}
	prepared.RequestedBy, prepared.Outcome = input.RequestedBy, ExecutionOutcomePlanning
	runnerInput.Preflight = preflight
	if !filepath.IsAbs(runnerInput.BackupArtifactRoot) {
		return ComposeUpdateOperation{}, newRolloutStartFailure(rolloutFailureOperationStartFailed, "backup", operation.OperationID, errors.New("prepared backup artifact root is unavailable"))
	}
	if err := s.persistAndLaunch(ctx, prepared, runnerInput); err != nil {
		return ComposeUpdateOperation{}, err
	}
	return prepared, nil
}

func (s *RolloutService) ensureNoActiveRunner() error {
	if s == nil || s.stateStore == nil {
		return nil
	}
	state, err := s.stateStore.Read()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read runner state before launch: %w", err)
	}
	if !isTerminalRunnerPhase(state.Phase) {
		return errors.New("a platform update runner is already active")
	}
	return nil
}

// GetOperation 返回活动 runner 快照，或已结算的数据库终态历史。
//
//nolint:cyclop // runner 状态读取、不可用投影和终态历史有不同 authority，必须保留可审计分支。
func (s *RolloutService) GetOperation(ctx context.Context, operationID string) (OperationView, error) {
	if s == nil || !runnerOperationID.MatchString(operationID) {
		return OperationView{}, errors.New("update operation identity is invalid")
	}
	if s.stateStore != nil {
		state, err := s.stateStore.Read()
		switch {
		case err == nil && state.OperationID == operationID:
			if view, terminated := s.runnerTerminationView(ctx, state); terminated {
				return view, nil
			}
			return updateOperationViewFromRunnerState(state), nil
		case err != nil && !errors.Is(err, os.ErrNotExist):
			if s.logger != nil {
				s.logger.Warn("platform update runner state read failed", zap.Error(err))
			}
			return OperationView{}, fmt.Errorf("%w: %v", errRunnerStateUnavailable, err)
		}
	}
	if s.operations == nil {
		return OperationView{}, errors.New("update operation store is unavailable")
	}
	operation, err := s.operations.Get(ctx, operationID)
	if err != nil {
		return OperationView{}, err
	}
	if !isTerminalOutcome(operation.Outcome) {
		return updateOperationViewFromUnavailableRunnerState(operation), nil
	}
	return updateOperationViewFromHistory(operation), nil
}

// runnerTerminationView 投影保留 runner 的退出证据，但不改写 runner 所有的生命周期状态。
//
//nolint:cyclop,gocognit,gocyclo // Docker 证据、持久操作和受控诊断分别属于独立的不可降级安全门。
func (s *RolloutService) runnerTerminationView(ctx context.Context, state RunnerState) (OperationView, bool) {
	reader, ok := s.launcher.(ComposeRunnerFailureReader)
	if !ok || s.operations == nil {
		return OperationView{}, false
	}
	failures, err := reader.ReadRunnerFailures(ctx)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("platform update runner failure inspection deferred", zap.Error(err))
		}
		return OperationView{}, false
	}
	for _, evidence := range failures {
		if evidence.OperationID != state.OperationID || !runnerFailureMatchesState(evidence, state) {
			continue
		}
		operation, getErr := s.operations.Get(ctx, state.OperationID)
		if getErr != nil || isTerminalOutcome(operation.Outcome) {
			return OperationView{}, false
		}
		if s.diagnostics != nil && operation.RequestID != "" && operation.RequestedBy != 0 {
			diagnostic := runnerTerminatedFailureDiagnostic(operation, evidence)
			if createErr := s.diagnostics.CreateFailureDiagnostic(ctx, diagnostic, operation.RequestedBy); createErr != nil && s.logger != nil {
				s.logger.Error("persist terminated platform update runner diagnostic failed", zap.String("module", moduleID), zap.String("operation_id", state.OperationID), zap.Error(createErr))
			}
		}
		return updateOperationViewFromTerminatedRunner(state), true
	}
	return OperationView{}, false
}

// GetActiveOperation 返回 runner 当前接管的操作；缺失状态卷时保留受控不可用投影，不能伪造 READY 进度。
//
//nolint:cyclop // runner 快照优先，状态缺失时才回退到受限数据库请求记录，分支对应不同事实来源。
func (s *RolloutService) GetActiveOperation(ctx context.Context) (*OperationView, error) {
	if s == nil {
		return nil, errors.New("update operation service is unavailable")
	}
	if s.stateStore != nil {
		state, err := s.stateStore.Read()
		switch {
		case err == nil && !isTerminalRunnerPhase(state.Phase):
			if view, terminated := s.runnerTerminationView(ctx, state); terminated {
				return &view, nil
			}
			view := updateOperationViewFromRunnerState(state)
			return &view, nil
		case err != nil && !errors.Is(err, os.ErrNotExist):
			return nil, fmt.Errorf("%w: %v", errRunnerStateUnavailable, err)
		}
	}
	if s.operations == nil {
		return nil, errors.New("update operation store is unavailable")
	}
	items, err := s.operations.List(ctx, maxActiveOperationScan)
	if err != nil {
		return nil, err
	}
	for _, operation := range items {
		if !isTerminalOutcome(operation.Outcome) {
			view := updateOperationViewFromUnavailableRunnerState(operation)
			return &view, nil
		}
	}
	return nil, errActiveUpdateOperationNotFound
}

// Recover 只在已验证 runner 异常退出且尚未迁移时，启动一次性终态恢复 runner。
// 它不执行 Compose 操作，也不由 server 伪造或改写 runner 生命周期快照。
//
//nolint:cyclop,gocognit,gocyclo // 状态、operation、退出证据和一次性 launcher 的绑定必须按序 fail closed。
func (s *RolloutService) Recover(ctx context.Context, operationID string) (ComposeUpdateOperation, error) {
	if s == nil || !runnerOperationID.MatchString(operationID) || s.stateStore == nil || s.operations == nil {
		return ComposeUpdateOperation{}, errors.New("update operation recovery is unavailable")
	}
	recoveryLauncher, ok := s.launcher.(ComposeRunnerRecoveryLauncher)
	if !ok {
		return ComposeUpdateOperation{}, errors.New("update operation recovery launcher is unavailable")
	}
	s.startMu.Lock()
	defer s.startMu.Unlock()
	state, err := s.stateStore.Read()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ComposeUpdateOperation{}, errUpdateOperationNotFound
		}
		return ComposeUpdateOperation{}, fmt.Errorf("%w: %v", errRunnerStateUnavailable, err)
	}
	if state.OperationID != operationID || isTerminalRunnerPhase(state.Phase) || phaseOrdinal(state.Phase) >= phaseOrdinal(RunnerPhaseMigration) {
		return ComposeUpdateOperation{}, errRecoveryConflict
	}
	operation, err := s.operations.Get(ctx, operationID)
	if err != nil {
		return ComposeUpdateOperation{}, err
	}
	if isTerminalOutcome(operation.Outcome) || operation.RunnerID != state.RunnerID {
		return ComposeUpdateOperation{}, errRecoveryConflict
	}
	failureReader, ok := s.launcher.(ComposeRunnerFailureReader)
	if !ok {
		return ComposeUpdateOperation{}, errors.New("update operation failure reader is unavailable")
	}
	failures, err := failureReader.ReadRunnerFailures(ctx)
	if err != nil {
		return ComposeUpdateOperation{}, fmt.Errorf("inspect terminated compose runner: %w", err)
	}
	matched := false
	for _, evidence := range failures {
		if evidence.OperationID == operationID && runnerFailureMatchesState(evidence, state) {
			matched = true
			break
		}
	}
	if !matched {
		return ComposeUpdateOperation{}, errRecoveryConflict
	}
	recoveryImage, err := s.recoveryRunnerImage()
	if err != nil {
		return ComposeUpdateOperation{}, err
	}
	if err := recoveryLauncher.LaunchRecovery(ctx, state, recoveryImage); err != nil {
		return ComposeUpdateOperation{}, fmt.Errorf("launch terminated compose runner recovery: %w", err)
	}
	return operation, nil
}

func runnerFailureMatchesState(evidence RunnerFailureEvidence, state RunnerState) bool {
	return evidence.RunnerID == "" || evidence.RunnerID == state.RunnerID
}

func (s *RolloutService) recoveryRunnerImage() (string, error) {
	image := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_RECOVERY_RUNNER_IMAGE"))
	const officialRecoveryRunnerPrefix = "ghcr.io/gewuyou/graft-compose-runner@sha256:"
	if !strings.HasPrefix(image, officialRecoveryRunnerPrefix) || !validDigest(strings.TrimPrefix(image, "ghcr.io/gewuyou/graft-compose-runner@")) {
		return "", errors.New("a digest-pinned recovery runner image is required")
	}
	return image, nil
}

// GetOperationEvents 返回 runner 状态卷中按 revision 回放的受控节点日志。
func (s *RolloutService) GetOperationEvents(ctx context.Context, operationID string, afterRevision uint64, limit int) ([]RunnerOperationEvent, error) {
	if s == nil || !runnerOperationID.MatchString(operationID) {
		return nil, errors.New("update operation identity is invalid")
	}
	view, err := s.GetOperation(ctx, operationID)
	if err != nil {
		return nil, err
	}
	if !view.StateAvailable {
		return nil, errRunnerStateUnavailable
	}
	reader, ok := s.stateStore.(RunnerStateEventReader)
	if !ok {
		return nil, errRunnerStateUnavailable
	}
	return reader.ReadEvents(operationID, afterRevision, limit)
}

// ListOperations 返回数据库中的终态业务历史，活动操作只能通过 GetOperation 读取状态卷快照。
func (s *RolloutService) ListOperations(ctx context.Context, limit int) ([]OperationView, error) {
	if s == nil || s.operations == nil {
		return nil, errors.New("update operation store is unavailable")
	}
	items, err := s.operations.List(ctx, limit)
	if err != nil {
		return nil, err
	}
	views := make([]OperationView, 0, len(items))
	for _, item := range items {
		if isTerminalOutcome(item.Outcome) {
			views = append(views, updateOperationViewFromHistory(item))
		}
	}
	return views, nil
}

//nolint:cyclop,gocognit,gocyclo,revive // 每个拒绝分支都是可独立审计的发布安全门。
func (s *RolloutService) confirmedPreflight(ctx context.Context, targetVersion, candidateKey string) (Status, ComposePreflight, DeploymentStrategy, error) {
	status := s.discovery.Status()
	if status.CacheStale || strings.TrimSpace(status.CheckError) != "" {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: fresh verified release catalog is required", errRolloutCatalogStale)
	}
	if status.Profile.Capability != "compose_upgrade_available" {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: rollout is unavailable for this installation", errRolloutInstallationUnavailable)
	}
	strategy, configured := configuredDeploymentStrategy()
	if strings.TrimSpace(strategy.ImageTag) == "" {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: configure %s in the deployment .env", errRolloutImageTagUnconfigured, imageTagEnv)
	}
	if !configured {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: %s must be latest, beta, or a pinned release tag", errRolloutImageTagInvalid, imageTagEnv)
	}
	if status.Profile.ComposeRootConfirmationRequired && strings.TrimSpace(candidateKey) == "" {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: select a discovered compose candidate", errRolloutCandidateConfirmation)
	}
	if strategy.Tracking && status.Latest == nil {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: the configured tracking channel has no newer verified release", errRolloutNoEligibleRelease)
	}
	release, found := verifiedRelease(status.AvailableReleases, targetVersion)
	if !found {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: target version is not in the verified release catalog", errRolloutInvalidArgument)
	}
	if release.Channel != strategy.Channel {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: target channel does not match the deployment image tag", errRolloutInvalidArgument)
	}
	if strategy.Tracking && (status.Latest == nil || release.Version != status.Latest.Version) {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: target version is not the tracking channel's latest verified release", errRolloutInvalidArgument)
	}
	current, currentErr := ParseVersion(status.CurrentVersion)
	target, targetErr := ParseVersion(release.Version)
	if currentErr != nil || targetErr != nil || target.Compare(current) <= 0 {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: target version must be newer than the running version", errRolloutNoEligibleRelease)
	}
	if minimum := strings.TrimSpace(release.MinimumSourceVersion); minimum != "" {
		current, currentErr := ParseVersion(status.CurrentVersion)
		minimumVersion, minimumErr := ParseVersion(minimum)
		if currentErr != nil || minimumErr != nil || current.Compare(minimumVersion) < 0 {
			return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: current version does not meet the target minimum source version", errRolloutSourceVersionUnsupported)
		}
	}
	if s.deployment == nil {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: deployment runtime is unavailable", errRolloutInstallationUnavailable)
	}
	snapshot, err := s.deployment.Freeze(ctx, moduleapi.DeploymentFreezeRequest{CandidateKey: candidateKey})
	if err != nil {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: %w", errRolloutComposeCandidateInvalid, err)
	}
	preflight, err := composePreflight(snapshot, release, strategy)
	if err != nil {
		return Status{}, ComposePreflight{}, "", fmt.Errorf("%w: %w", errRolloutComposePreflightFailed, err)
	}
	return status, preflight, strategy.Mode, nil
}

func verifiedRelease(catalog []Release, version string) (Release, bool) {
	for _, release := range catalog {
		if release.Version == strings.TrimSpace(version) {
			return release, true
		}
	}
	return Release{}, false
}

//nolint:cyclop // 每个稳定预检失败都映射到一个明确的公开错误码。
func classifyRolloutPreflightFailure(err error) string {
	switch {
	case errors.Is(err, errRolloutInvalidArgument):
		return rolloutFailureInvalidTarget
	case errors.Is(err, errRolloutCatalogStale):
		return rolloutFailureCatalogStale
	case errors.Is(err, errRolloutInstallationUnavailable):
		return rolloutFailureInstallationUnavailable
	case errors.Is(err, errRolloutImageTagUnconfigured):
		return rolloutFailureImageTagUnconfigured
	case errors.Is(err, errRolloutImageTagInvalid):
		return rolloutFailureImageTagInvalid
	case errors.Is(err, errRolloutCandidateConfirmation):
		return rolloutFailureCandidateConfirmationRequired
	case errors.Is(err, errRolloutNoEligibleRelease):
		return rolloutFailureNoEligibleRelease
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
	s.statePollMu.Lock()
	s.statePollClosed = true
	stateCancel := s.statePollCancel
	stateDone := s.statePollDone
	s.statePollCancel = nil
	s.statePollDone = nil
	s.statePollMu.Unlock()
	if stateCancel != nil {
		stateCancel()
	}
	if stateDone != nil {
		<-stateDone
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

// StartRunnerStateProjection 将 runner 所有的快照发布到既有实时通道。
// server 停止只会暂停投影；状态卷仍为事实源，并在下次启动时重新收敛。
func (s *RolloutService) StartRunnerStateProjection(ctx context.Context) {
	if s == nil || s.stateStore == nil {
		return
	}
	pollCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.statePollMu.Lock()
	if s.statePollClosed || s.statePollCancel != nil {
		s.statePollMu.Unlock()
		cancel()
		return
	}
	s.statePollCancel = cancel
	s.statePollDone = done
	interval := s.statePollEvery
	if interval <= 0 {
		interval = runnerStatePollInterval
	}
	s.statePollMu.Unlock()
	go func() {
		defer close(done)
		s.runRunnerStateProjection(pollCtx, interval)
	}()
}

func (s *RolloutService) runRunnerStateProjection(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var publishedOperation string
	var publishedRevision uint64
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, err := s.readRunnerState()
			if err != nil {
				s.logRunnerStateProjectionError(err)
				continue
			}
			if state.OperationID != publishedOperation || state.Revision != publishedRevision {
				s.publishRunnerState(state)
				publishedOperation, publishedRevision = state.OperationID, state.Revision
			}
			s.settleRunnerState(ctx, state)
		}
	}
}

func (s *RolloutService) logRunnerStateProjectionError(err error) {
	if s.logger != nil && !errors.Is(err, os.ErrNotExist) {
		s.logger.Warn("platform update runner state projection deferred", zap.Error(err))
	}
}

func (s *RolloutService) settleRunnerState(ctx context.Context, state RunnerState) {
	if !isTerminalRunnerPhase(state.Phase) || state.Receipt == nil {
		return
	}
	if state.Receipt.OperationID != state.OperationID || state.Receipt.RunnerID != state.RunnerID {
		if s.logger != nil {
			s.logger.Warn("platform update runner terminal receipt does not match state snapshot")
		}
		return
	}
	if _, err := s.SettlePersistedReceipt(ctx, *state.Receipt); err != nil && s.logger != nil {
		s.logger.Warn("platform update runner terminal settlement deferred", zap.Error(err))
	}
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
			if err := s.reconcileAvailableReceipts(ctx, polling.reader); err != nil {
				s.logReceiptReconciliationDeferred()
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
	if persisted, getErr := s.operations.Get(ctx, settled.OperationID); getErr == nil {
		settled = persisted
	}
	s.persistTerminalFailureDiagnostic(ctx, settled, receipt)
	s.logTerminalFailure(ctx, settled, receipt)
	s.publishAudit(ctx, settled, settled.Outcome == ExecutionOutcomeSuccess, settled.FailureCode)
	return settled, nil
}

func (s *RolloutService) logTerminalFailure(ctx context.Context, operation ComposeUpdateOperation, receipt RunnerReceipt) {
	if s == nil || s.appLogger == nil || (operation.Outcome != ExecutionOutcomeFailed && operation.Outcome != ExecutionOutcomeNeedsAttention) {
		return
	}
	stage, detail := runnerReceiptFailureDiagnostic(receipt)
	s.appLogger.Named("modules.update.rollout").Error(ctx, runnerFailureDiagnosticSummary,
		logger.StringField(logger.FieldOperation, "platform_update.runner_terminal_failure"),
		logger.StringField(logger.FieldRequestID, operation.RequestID),
		logger.StringField("operation_id", operation.OperationID),
		logger.Uint64Field("task_id", operation.TaskID),
		logger.StringField("target_version", operation.TargetVersion),
		logger.StringField("failure_code", receipt.FailureCode),
		logger.StringField("failure_stage", stage),
		logger.StringField("status", string(operation.Outcome)),
		logger.StringField(logger.FieldError, detail),
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
	payload := moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Operator: operator, Action: "platform.update.compose", ResourceType: "platform_update", ResourceID: operation.OperationID, ResourceName: operation.TargetVersion, RequestID: requestAudit.RequestID, RequestMethod: requestAudit.Method, RequestPath: requestAudit.Route, IP: requestAudit.ClientIP, UserAgent: requestAudit.UserAgent, StatusCode: http.StatusAccepted, Success: success, Message: strings.TrimSpace(message), Metadata: map[string]any{"source_version": operation.SourceVersion, "target_version": operation.TargetVersion, "deployment_strategy": operation.DeploymentStrategy, "task_id": operation.TaskID, "status": operation.Outcome}, CreatedAt: time.Now().UTC()}
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
	reader, ok := s.launcher.(ComposeRunnerReceiptReader)
	if !ok {
		return nil
	}
	return s.reconcileAvailableReceipts(ctx, reader)
}

// SetRealtimePublisher 注入 core realtime topic publisher；Update 只发布已持久化的操作快照。
func (s *RolloutService) SetRealtimePublisher(publisher realtime.Publisher) {
	if s != nil {
		s.realtime = publisher
	}
}

// ReconcileRunnerState 只读取 runner 状态卷；活动阶段不写回数据库。
// 当 runner 已持久化终态 receipt 时，server 幂等结算其派生的 Task、Backup 和历史记录。
func (s *RolloutService) ReconcileRunnerState(ctx context.Context) error {
	if s == nil || s.stateStore == nil {
		return nil
	}
	state, err := s.readRunnerState()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read runner state: %w", err)
	}
	s.publishRunnerState(state)
	if !isTerminalRunnerPhase(state.Phase) || state.Receipt == nil {
		return nil
	}
	if state.Receipt.OperationID != state.OperationID || state.Receipt.RunnerID != state.RunnerID {
		return errors.New("runner terminal receipt does not match state snapshot")
	}
	_, err = s.SettlePersistedReceipt(ctx, *state.Receipt)
	return err
}

func (s *RolloutService) readRunnerState() (RunnerState, error) {
	if s == nil || s.stateStore == nil {
		return RunnerState{}, os.ErrNotExist
	}
	return s.stateStore.Read()
}

func (s *RolloutService) publishRunnerState(state RunnerState) {
	if s == nil || s.realtime == nil || !runnerOperationID.MatchString(state.OperationID) {
		return
	}
	view := updateOperationViewFromRunnerState(state)
	s.realtime.Publish(updateOperationTopic(state.OperationID), struct {
		Event     RunnerOperationEvent `json:"event"`
		Operation OperationView        `json:"operation"`
	}{Event: newRunnerOperationEvent(state), Operation: view})
}

func (s *RolloutService) reconcileAvailableReceipts(ctx context.Context, reader ComposeRunnerReceiptReader) error {
	receipts, err := reader.ReadRunnerReceipts(ctx)
	if err != nil {
		return fmt.Errorf("read retained compose runner receipts: %w", err)
	}
	var settlementErr error
	for _, receipt := range receipts {
		if _, err := s.settleReceiptAndCleanup(ctx, receipt); err != nil {
			settlementErr = errors.Join(settlementErr, fmt.Errorf("settle retained compose runner receipt: %w", err))
		}
	}
	return settlementErr
}

// logReceiptReconciliationDeferred 保留固定、无敏感信息的运维信号；下一轮轮询会继续读取未删除的 runner receipt。
func (s *RolloutService) logReceiptReconciliationDeferred() {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Warn("platform update runner receipt reconciliation deferred",
		zap.String("module", moduleID),
		zap.String("reason", "runner_receipt_reconciliation_failed"),
	)
}

func updateOperationTopic(operationID string) string {
	return "platform.update.operations." + operationID
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

func composePreflight(snapshot moduleapi.DeploymentSnapshot, release Release, strategy ResolvedDeploymentStrategy) (ComposePreflight, error) {
	candidate := snapshot.Candidate()
	root := candidate.Root()
	composeFiles := candidate.ConfigFiles()
	if root == "" || len(composeFiles) == 0 {
		return ComposePreflight{}, fmt.Errorf("%w: frozen compose candidate is incomplete", errRolloutComposeCandidateInvalid)
	}
	value := ComposePreflight{DeclaredMode: snapshot.Context().Mode(), DeploymentStrategy: strategy.Mode, ImageTag: strategy.ImageTag, DetectedMode: snapshot.Context().Mode(), ComposeRoot: root, Platform: "linux/amd64", DockerSocket: "/var/run/docker.sock", ComposeFiles: append([]string(nil), composeFiles...), BundledPostgres: true, OfficialServerImage: release.ServerImage, OfficialWebImage: release.WebImage, OfficialRunnerImage: release.RunnerImage, ServerDigest: release.ServerDigest, WebDigest: release.WebDigest, RunnerDigest: release.RunnerDigest, ServerReference: release.ServerImage + ":v" + release.Version, WebReference: release.WebImage + ":v" + release.Version, RunnerReference: release.RunnerRef}
	if err := ValidateComposePreflight(value); err != nil {
		return ComposePreflight{}, fmt.Errorf("preflight official compose rollout: %w", err)
	}
	return value, nil
}

func backupHandoff(operationID string, requestedBy uint64, root string) moduleapi.BackupRunnerHandoffPlan {
	artifactRoot := filepath.Join(root, operationID)
	creator := requestedBy
	return moduleapi.BackupRunnerHandoffPlan{OperationID: operationID, Purpose: "before-update", RetainUntil: time.Now().UTC().Add(30 * 24 * time.Hour), CreatedBy: &creator, ArtifactRoot: artifactRoot, ConfigSnapshotRef: filepath.Join(artifactRoot, "config.snapshot"), DatabaseDumpRef: filepath.Join(artifactRoot, "database.dump")}
}
