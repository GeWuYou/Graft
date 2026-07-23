package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"graft/server/internal/eventbus"
	"graft/server/internal/moduleapi"
)

// RolloutService 只编排人工确认的 Compose 更新；它保存 Update operation，Task 和 Backup 仍是外部能力。
type RolloutService struct {
	discovery    *Service
	operations   OperationStore
	coordinator  *ComposeExecutionCoordinator
	launcher     ComposeRunnerLauncher
	newOperation func() string
	auditBus     eventbus.Bus
}

var (
	errRolloutInvalidArgument = errors.New("compose update request is invalid")
	errRolloutPrecondition    = errors.New("compose update precondition is not met")
)

// SetAuditBus 注入审计事件发布端；Update 只发布领域证据，审计事实仍由 Audit 模块拥有。
func (s *RolloutService) SetAuditBus(bus eventbus.Bus) { s.auditBus = bus }

func newOperationID() string { return fmt.Sprintf("update-%d", time.Now().UTC().UnixNano()) }

// NewRolloutService 组合已注册的窄 capability 与受限 Docker launcher。
func NewRolloutService(discovery *Service, operations OperationStore, tasks moduleapi.TaskService, backups moduleapi.BackupService, launcher ComposeRunnerLauncher) *RolloutService {
	return &RolloutService{discovery: discovery, operations: operations, coordinator: NewComposeExecutionCoordinator(tasks, backups), launcher: launcher, newOperation: newOperationID}
}

// Start 要求操作者确认当前 catalog 中精确的候选版本，随后仅启动一次 digest-pinned runner。
func (s *RolloutService) Start(ctx context.Context, requestedBy uint64, targetVersion, confirmation string) (ComposeUpdateOperation, error) {
	if s == nil || s.discovery == nil || s.operations == nil || s.coordinator == nil || s.launcher == nil || requestedBy == 0 {
		return ComposeUpdateOperation{}, errors.New("compose update rollout is unavailable")
	}
	status, preflight, err := s.confirmedPreflight(targetVersion, confirmation)
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
func (s *RolloutService) confirmedPreflight(targetVersion, confirmation string) (Status, ComposePreflight, error) {
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
	if strings.TrimSpace(targetVersion) == "" || targetVersion != status.Latest.Version || confirmation != targetVersion {
		return Status{}, ComposePreflight{}, fmt.Errorf("%w: target version requires an exact manual confirmation", errRolloutInvalidArgument)
	}
	preflight, err := composePreflight(status.Profile, *status.Latest)
	if err != nil {
		return Status{}, ComposePreflight{}, fmt.Errorf("%w: %w", errRolloutPrecondition, err)
	}
	return status, preflight, nil
}

// Close 释放 rollout 持有的 Docker client，供模块关闭阶段调用。
func (s *RolloutService) Close() error {
	if s == nil || s.launcher == nil {
		return nil
	}
	return s.launcher.Close()
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
	if s == nil || s.auditBus == nil {
		return
	}
	requestAuth, _ := moduleapi.RequestAuthContextFromContext(ctx)
	var operator *moduleapi.CurrentUser
	if requestAuth.User != nil {
		operator = requestAuth.User
	}
	_ = s.auditBus.Publish(ctx, eventbus.Event{Name: string(moduleapi.AuditRecordEventName), Source: moduleID, Payload: moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Operator: operator, Action: "platform.update.compose", ResourceType: "platform_update", ResourceID: operation.OperationID, ResourceName: operation.TargetVersion, StatusCode: http.StatusAccepted, Success: success, Message: strings.TrimSpace(message), Metadata: map[string]any{"source_version": operation.SourceVersion, "target_version": operation.TargetVersion, "task_id": operation.TaskID, "status": operation.Outcome}}, OccurredAt: time.Now().UTC()})
}

// SettleAvailableReceipts 收敛目标 Compose 根目录下由 runner 留下的 receipt；无法解析的文件不影响 server 启动。
func (s *RolloutService) SettleAvailableReceipts(ctx context.Context) error {
	if s == nil || s.discovery == nil {
		return nil
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
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || filepath.Base(entry.Name()) != entry.Name() {
		return nil
	}
	path := filepath.Join(root, runnerReceiptDirectory, entry.Name())
	// #nosec G304,G703 -- entry 由受限 receipt 目录读取，且其 basename 已校验，不接受 HTTP 路径输入。
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read compose runner receipt: %w", err)
	}
	var receipt RunnerReceipt
	if err := json.Unmarshal(contents, &receipt); err != nil {
		return fmt.Errorf("decode compose runner receipt: %w", err)
	}
	settled, err := s.SettlePersistedReceipt(ctx, receipt)
	if err != nil && !errors.Is(err, errUpdateOperationNotFound) {
		return err
	}
	if err == nil && settled.Outcome == ExecutionOutcomeSuccess {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove settled compose runner receipt: %w", err)
		}
	}
	return nil
}

func composePreflight(profile InstallationProfile, release Release) (ComposePreflight, error) {
	root := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_COMPOSE_ROOT"))
	value := ComposePreflight{DeclaredMode: profile.DeclaredMode, DetectedMode: profile.DetectedMode, ComposeRoot: root, Platform: "linux/amd64", DockerSocket: "/var/run/docker.sock", ComposeFiles: []string{filepath.Join(root, "compose.yml")}, BundledPostgres: true, OfficialServerImage: release.ServerImage, OfficialWebImage: release.WebImage, OfficialRunnerImage: release.RunnerImage, ServerDigest: release.ServerDigest, WebDigest: release.WebDigest, RunnerDigest: release.RunnerDigest, ServerReference: release.ServerRef, WebReference: release.WebRef, RunnerReference: release.RunnerRef}
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
