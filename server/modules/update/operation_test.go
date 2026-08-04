package update

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	containertypes "github.com/moby/moby/api/types/container"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"graft/server/internal/buildinfo"
	"graft/server/internal/event"
	"graft/server/internal/httpx"
	"graft/server/internal/logger"
	"graft/server/internal/moduleapi"
)

type deploymentRuntimeStub struct {
	context moduleapi.DeploymentContext
	err     error
}

func (s deploymentRuntimeStub) Current(context.Context) moduleapi.DeploymentContext { return s.context }

func (s deploymentRuntimeStub) Freeze(_ context.Context, request moduleapi.DeploymentFreezeRequest) (moduleapi.DeploymentSnapshot, error) {
	if s.err != nil {
		return moduleapi.DeploymentSnapshot{}, s.err
	}
	for _, candidate := range s.context.ComposeCandidates() {
		if request.CandidateKey == "" || request.CandidateKey == candidate.Key() {
			return moduleapi.NewDeploymentSnapshot(s.context, candidate, "test-fingerprint"), nil
		}
	}
	return moduleapi.DeploymentSnapshot{}, errors.New("candidate unavailable")
}

func composeDeploymentRuntime(root string) moduleapi.DeploymentRuntime {
	candidate := moduleapi.NewDeploymentComposeCandidate("compose-test", root, []string{filepath.Join(root, "compose.yml")}, "graft", "high", nil)
	return deploymentRuntimeStub{context: moduleapi.NewDeploymentContext("compose", "explicit_config", false, []moduleapi.DeploymentComposeCandidate{candidate}, nil)}
}

func TestComposeExecutionCoordinatorSettlesReceiptAndDoesNotExposeBackupRefs(t *testing.T) {
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 41, Status: moduleapi.TaskStatusPending}}
	backups := &stubBackupService{}
	coordinator := NewComposeExecutionCoordinator(tasks, backups)
	operation, input, err := coordinator.Start(context.Background(), ComposeUpdateOperation{OperationID: "update-41", SourceVersion: "v1.0.0", TargetVersion: "v1.1.0"}, 9, testBackupPlan("update-41"))
	if err != nil {
		t.Fatalf("start compose update: %v", err)
	}
	if input.TaskID != 41 || backups.plan.TaskID != 41 || tasks.plan.Plan.Stages[0].ExternalReceipt.OperationID != operation.OperationID {
		t.Fatalf("operation linkage was not frozen: %#v %#v", operation, backups.plan)
	}
	receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: operation.OperationID, Succeeded: true, BackupCompletion: &moduleapi.CompleteBackupRunnerHandoffInput{OperationID: operation.OperationID, TaskID: operation.TaskID, ConfigSnapshotSHA256: testDigest('a'), ConfigSnapshotBytes: 3, DatabaseDumpSHA256: testDigest('b'), DatabaseDumpBytes: 5}}
	settled, err := coordinator.SettleReceipt(context.Background(), operation, receipt)
	if err != nil {
		t.Fatalf("settle compose receipt: %v", err)
	}
	if settled.Outcome != ExecutionOutcomeSuccess || settled.BackupID != 8 || tasks.external.Outcome != moduleapi.ExternalReceiptOutcomeSuccess || tasks.external.IntegritySHA256 == "" {
		t.Fatalf("unexpected settled operation: %#v / %#v", settled, tasks.external)
	}
	if backups.completion.OperationID != operation.OperationID || backups.completion.TaskID != operation.TaskID {
		t.Fatalf("backup completion binding lost: %#v", backups.completion)
	}
}

func TestComposeExecutionCoordinatorMarksPostMigrationFailureNeedsAttention(t *testing.T) {
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 52, Status: moduleapi.TaskStatusPending}}
	coordinator := NewComposeExecutionCoordinator(tasks, &stubBackupService{})
	operation := ComposeUpdateOperation{OperationID: "update-52", SourceVersion: "v1.0.0", TargetVersion: "v1.1.0", TaskID: 52}
	settled, err := coordinator.SettleReceipt(context.Background(), operation, RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: operation.OperationID, MigrationStarted: true, FailureCode: "healthz_failed"})
	if err != nil {
		t.Fatalf("settle post-migration failure: %v", err)
	}
	if settled.Outcome != ExecutionOutcomeNeedsAttention || tasks.external.Outcome != moduleapi.ExternalReceiptOutcomeNeedsAttention {
		t.Fatalf("post-migration receipt was not retained for attention: %#v %#v", settled, tasks.external)
	}
}

func TestComposeExecutionCoordinatorPreservesVerifiedRunnerIDWhenReceiptOmitsIt(t *testing.T) {
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 54, Status: moduleapi.TaskStatusPending}}
	coordinator := NewComposeExecutionCoordinator(tasks, &stubBackupService{})
	operation := ComposeUpdateOperation{OperationID: "update-54", RunnerID: "runner-update-54", SourceVersion: "v1.0.0", TargetVersion: "v1.1.0", TaskID: 54}
	settled, err := coordinator.SettleReceipt(t.Context(), operation, RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: operation.OperationID, Succeeded: true})
	if err != nil {
		t.Fatalf("settle receipt: %v", err)
	}
	if settled.RunnerID != operation.RunnerID {
		t.Fatalf("runner ID = %q, want %q", settled.RunnerID, operation.RunnerID)
	}
}

func TestComposeExecutionCoordinatorSettlesLegacyRunnerReceiptWithItsProtocol(t *testing.T) {
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 53, Status: moduleapi.TaskStatusPending}}
	coordinator := NewComposeExecutionCoordinator(tasks, &stubBackupService{})
	operation := ComposeUpdateOperation{OperationID: "update-53", SourceVersion: "v1.0.0", TargetVersion: "v1.1.0", TaskID: 53}

	if _, err := coordinator.SettleReceipt(t.Context(), operation, RunnerReceipt{ProtocolVersion: legacyRunnerProtocolVersion, OperationID: operation.OperationID, Succeeded: true}); err != nil {
		t.Fatalf("settle legacy compose receipt: %v", err)
	}
	if tasks.external.Protocol != legacyRunnerProtocol {
		t.Fatalf("legacy receipt protocol = %q, want %q", tasks.external.Protocol, legacyRunnerProtocol)
	}
}

func TestComposeExecutionCoordinatorRejectsForgedBackupReceiptBinding(t *testing.T) {
	coordinator := NewComposeExecutionCoordinator(&stubTaskService{}, &stubBackupService{})
	_, err := coordinator.SettleReceipt(context.Background(), ComposeUpdateOperation{OperationID: "update-53", SourceVersion: "v1.0.0", TargetVersion: "v1.1.0", TaskID: 53}, RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: "update-53", FailureCode: "pull_failed", BackupCompletion: &moduleapi.CompleteBackupRunnerHandoffInput{OperationID: "other", TaskID: 53}})
	if err == nil {
		t.Fatal("expected forged backup completion rejection")
	}
}

func TestComposeExecutionCoordinatorCancelsPreparedHandoffWhenItsBindingIsInvalid(t *testing.T) {
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 66}}
	backups := &stubBackupService{prepared: &moduleapi.BackupRunnerHandoffPlan{OperationID: "other", TaskID: 66}}
	coordinator := NewComposeExecutionCoordinator(tasks, backups)
	_, _, err := coordinator.Start(t.Context(), ComposeUpdateOperation{OperationID: "update-66", SourceVersion: "v1.0.0", TargetVersion: "v1.1.0"}, 9, testBackupPlan("update-66"))
	if err == nil || !strings.Contains(err.Error(), "prepared backup handoff does not match update operation") {
		t.Fatalf("expected invalid prepared handoff rejection, got %v", err)
	}
	if tasks.canceled != 66 || backups.canceled.OperationID != "update-66" || backups.canceled.TaskID != 66 {
		t.Fatalf("invalid prepared handoff was not cancelled through owners: task=%d backup=%#v", tasks.canceled, backups.canceled)
	}
}

func TestRunnerReceiptDoesNotSerializeBackupStorageReferences(t *testing.T) {
	receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: "update-54", BackupCompletion: &moduleapi.CompleteBackupRunnerHandoffInput{OperationID: "update-54", TaskID: 54, ConfigSnapshotSHA256: testDigest('a'), DatabaseDumpSHA256: testDigest('b')}}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal runner receipt: %v", err)
	}
	if string(encoded) == "" || containsAny(string(encoded), "storage_ref", "database_dump_ref", "config_snapshot_ref", "/var/") {
		t.Fatalf("runner receipt exposes backup storage data: %s", encoded)
	}
}

func TestRolloutRequiresCurrentVerifiedTargetAndPersistsLauncherOperation(t *testing.T) {
	root := t.TempDir()
	t.Setenv(imageTagEnv, "beta")
	discovery := NewService(nil)
	discovery.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0"} }
	discovery.profile = func() InstallationProfile {
		return InstallationProfile{DeclaredMode: "compose", DetectedMode: "compose", Capability: "compose_upgrade_available"}
	}
	discovery.latest = &Release{Version: "1.1.0", Channel: "beta", ServerImage: "ghcr.io/gewuyou/graft-server", WebImage: "ghcr.io/gewuyou/graft-web", RunnerImage: "ghcr.io/gewuyou/graft-compose-runner", ServerDigest: "sha256:" + strings.Repeat("a", 64), WebDigest: "sha256:" + strings.Repeat("b", 64), RunnerDigest: "sha256:" + strings.Repeat("c", 64)}
	discovery.latest.ServerRef = discovery.latest.ServerImage + "@" + discovery.latest.ServerDigest
	discovery.latest.WebRef = discovery.latest.WebImage + "@" + discovery.latest.WebDigest
	discovery.latest.RunnerRef = discovery.latest.RunnerImage + "@" + discovery.latest.RunnerDigest
	fresh := time.Now().UTC()
	discovery.lastSuccessfulAt = &fresh
	operations := &memoryOperationStore{}
	launcher := &recordingLauncher{}
	backups := &stubBackupService{}
	rollout := NewRolloutService(discovery, operations, &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 77}}, backups, launcher)
	discovery.profile = func() InstallationProfile {
		return InstallationProfile{DeclaredMode: "compose", DetectedMode: "compose", Capability: "manual_guidance_blocked"}
	}
	if _, err := rollout.Start(t.Context(), StartRolloutInput{RequestedBy: 9, TargetVersion: "1.1.0"}); !errors.Is(err, errRolloutInstallationUnavailable) {
		t.Fatalf("start with blocked installation error = %v, want installation unavailable", err)
	}
	if len(operations.items) != 0 || launcher.input.OperationID != "" {
		t.Fatalf("blocked installation created a rollout side effect: operations=%#v launcher=%#v", operations.items, launcher.input)
	}
	discovery.profile = func() InstallationProfile {
		return InstallationProfile{DeclaredMode: "compose", DetectedMode: "compose", Capability: "compose_upgrade_available"}
	}
	if _, err := rollout.Start(t.Context(), StartRolloutInput{RequestedBy: 9, TargetVersion: "1.1.0"}); !errors.Is(err, errRolloutInstallationUnavailable) {
		t.Fatalf("start without deployment runtime error = %v, want installation unavailable", err)
	}
	rollout.SetDeploymentRuntime(composeDeploymentRuntime(root))
	rollout.SetBackupArtifactRoot("/var/lib/graft/backups")
	rollout.newOperation = func() string { return "update-77" }
	if _, err := rollout.Start(t.Context(), StartRolloutInput{RequestedBy: 9, TargetVersion: "1.0.9"}); err == nil {
		t.Fatal("expected target version rejection")
	}
	operation, err := rollout.Start(t.Context(), StartRolloutInput{RequestedBy: 9, TargetVersion: "1.1.0"})
	if err != nil {
		t.Fatalf("start rollout: %v", err)
	}
	if operation.Outcome != ExecutionOutcomePlanning || operation.TaskID != 77 || launcher.input.Preflight.ComposeRoot != root {
		t.Fatalf("unexpected rollout operation: %#v / %#v", operation, launcher.input)
	}
	if launcher.input.Preflight.ComposeRoot != root || launcher.input.OperationID != operation.OperationID || operations.items[operation.OperationID].TaskID != 77 {
		t.Fatalf("runner input or persisted operation lost constrained identity: %#v", operations.items)
	}
	if launcher.input.BackupArtifactRoot != "/var/lib/graft/backups/update-77" || backups.plan.ArtifactRoot != launcher.input.BackupArtifactRoot {
		t.Fatalf("backup handoff did not use the server artifact root: input=%q plan=%q", launcher.input.BackupArtifactRoot, backups.plan.ArtifactRoot)
	}
}

func TestRolloutLaunchFailureCancelsTaskAndBackupHandoffThroughCapabilities(t *testing.T) {
	root := t.TempDir()
	t.Setenv(imageTagEnv, "beta")
	discovery := NewService(nil)
	discovery.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0"} }
	discovery.profile = func() InstallationProfile {
		return InstallationProfile{DeclaredMode: "compose", DetectedMode: "compose", Capability: "compose_upgrade_available"}
	}
	discovery.latest = &Release{Version: "1.1.0", Channel: "beta", ServerImage: "ghcr.io/gewuyou/graft-server", WebImage: "ghcr.io/gewuyou/graft-web", RunnerImage: "ghcr.io/gewuyou/graft-compose-runner", ServerDigest: "sha256:" + strings.Repeat("a", 64), WebDigest: "sha256:" + strings.Repeat("b", 64), RunnerDigest: "sha256:" + strings.Repeat("c", 64)}
	discovery.latest.ServerRef = discovery.latest.ServerImage + "@" + discovery.latest.ServerDigest
	discovery.latest.WebRef = discovery.latest.WebImage + "@" + discovery.latest.WebDigest
	discovery.latest.RunnerRef = discovery.latest.RunnerImage + "@" + discovery.latest.RunnerDigest
	fresh := time.Now().UTC()
	discovery.lastSuccessfulAt = &fresh
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 78}}
	backups := &stubBackupService{}
	operations := &memoryOperationStore{}
	rollout := NewRolloutService(discovery, operations, tasks, backups, &recordingLauncher{launchErr: errors.New("docker unavailable")})
	rollout.SetDeploymentRuntime(composeDeploymentRuntime(root))
	rollout.SetBackupArtifactRoot("/var/lib/graft/backups")
	rollout.newOperation = func() string { return "update-78" }
	if _, err := rollout.Start(t.Context(), StartRolloutInput{RequestedBy: 9, TargetVersion: "1.1.0"}); err == nil {
		t.Fatal("expected launcher failure")
	}
	if tasks.canceled != 78 || backups.canceled.OperationID != "update-78" || backups.canceled.TaskID != 78 {
		t.Fatalf("owner capability cleanup was not requested: task=%d backup=%#v", tasks.canceled, backups.canceled)
	}
	if item := operations.items["update-78"]; item.Outcome != ExecutionOutcomeFailed || item.FailureCode != "runner_launch_failed" {
		t.Fatalf("operation failure was not persisted: %#v", item)
	}
}

func TestSettlePersistedReceiptShortCircuitsTerminalOperation(t *testing.T) {
	operations := &memoryOperationStore{items: map[string]ComposeUpdateOperation{
		"update-83": {OperationID: "update-83", SourceVersion: "1.0.0", TargetVersion: "1.1.0", TaskID: 83, Outcome: ExecutionOutcomeSuccess},
	}}
	tasks := &stubTaskService{}
	backups := &stubBackupService{}
	rollout := NewRolloutService(NewService(nil), operations, tasks, backups, &recordingLauncher{})

	settled, err := rollout.SettlePersistedReceipt(t.Context(), RunnerReceipt{OperationID: "update-83"})
	if err != nil {
		t.Fatalf("settle terminal operation: %v", err)
	}
	if settled != operations.items["update-83"] {
		t.Fatalf("terminal operation changed during replay: got %#v want %#v", settled, operations.items["update-83"])
	}
	if tasks.external.TaskID != 0 || backups.completion.OperationID != "" {
		t.Fatalf("terminal receipt replay triggered side effects: task=%#v backup=%#v", tasks.external, backups.completion)
	}
}

func TestLogTerminalFailureWithoutRequestIDStillWritesAppLog(t *testing.T) {
	core, observed := observer.New(zap.ErrorLevel)
	rollout := NewRolloutService(NewService(nil), &memoryOperationStore{}, &stubTaskService{}, &stubBackupService{}, &recordingLauncher{})
	rollout.SetAppLogger(logger.NewAppLogger(zap.New(core)))

	rollout.logTerminalFailure(t.Context(), ComposeUpdateOperation{OperationID: "update-84", TaskID: 84, TargetVersion: "1.1.0", Outcome: ExecutionOutcomeFailed}, RunnerReceipt{FailureCode: runnerFailureBackup, FailureStage: string(RunnerBackupFailureStageConfigSnapshot), FailureDetail: string(RunnerBackupFailureDetailPermissionDenied)})

	entries := observed.All()
	if len(entries) != 1 || entries[0].Message != runnerFailureDiagnosticSummary {
		t.Fatalf("expected terminal failure app log without request ID, got %#v", entries)
	}
	fields := entries[0].ContextMap()
	if fields["failure_stage"] != string(RunnerBackupFailureStageConfigSnapshot) || fields[logger.FieldError] != "deployment environment snapshot was denied by deployment filesystem permissions" {
		t.Fatalf("expected bounded terminal diagnostic in app log, got %#v", fields)
	}
}

func TestSettleAvailableReceiptsCleansRunnerOnlyAfterSuccessfulSettlement(t *testing.T) {
	discovery := NewService(nil)
	discovery.profile = func() InstallationProfile { return InstallationProfile{DetectedMode: "compose"} }
	operations := &memoryOperationStore{items: map[string]ComposeUpdateOperation{
		"update-84": {OperationID: "update-84", SourceVersion: "1.0.0", TargetVersion: "1.1.0", TaskID: 84, Outcome: ExecutionOutcomePulling},
	}}
	launcher := &receiptReaderCleanupLauncher{receipts: []RunnerReceipt{
		{ProtocolVersion: runnerProtocolVersion, OperationID: "update-84", Succeeded: true, BackupCompletion: &moduleapi.CompleteBackupRunnerHandoffInput{OperationID: "update-84", TaskID: 84, ConfigSnapshotSHA256: testDigest('a'), ConfigSnapshotBytes: 3, DatabaseDumpSHA256: testDigest('b'), DatabaseDumpBytes: 5}},
		{OperationID: "update-missing", Succeeded: true},
	}}
	rollout := NewRolloutService(discovery, operations, &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 84}}, &stubBackupService{}, launcher)
	if err := rollout.SettleAvailableReceipts(t.Context()); err == nil || !strings.Contains(err.Error(), "update operation not found") {
		t.Fatalf("settle available receipts error = %v, want invalid receipt to remain observable", err)
	}
	if !slices.Equal(launcher.removed, []string{"update-84"}) {
		t.Fatalf("removed runners = %#v, want only successfully settled runner", launcher.removed)
	}
}

func TestReceiptPollingCloseCancelsAndWaitsForReader(t *testing.T) {
	launcher := &blockingReceiptReaderLauncher{started: make(chan struct{}), closed: make(chan struct{})}
	rollout := NewRolloutService(NewService(nil), &memoryOperationStore{}, &stubTaskService{}, &stubBackupService{}, launcher)
	rollout.receiptPollEvery = time.Millisecond
	rollout.StartReceiptPolling(t.Context())
	select {
	case <-launcher.started:
	case <-time.After(time.Second):
		t.Fatal("receipt polling did not start")
	}
	if err := rollout.Close(); err != nil {
		t.Fatalf("close rollout: %v", err)
	}
	select {
	case <-launcher.closed:
	default:
		t.Fatal("launcher was closed before polling reader exited")
	}
	rollout.StartReceiptPolling(t.Context())
}

func TestReceiptPollingStartAndCloseAreSafeConcurrently(t *testing.T) {
	launcher := &blockingReceiptReaderLauncher{started: make(chan struct{}), closed: make(chan struct{})}
	rollout := NewRolloutService(NewService(nil), &memoryOperationStore{}, &stubTaskService{}, &stubBackupService{}, launcher)
	rollout.receiptPollEvery = time.Millisecond
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			rollout.StartReceiptPolling(t.Context())
		}()
	}
	group.Add(1)
	go func() {
		defer group.Done()
		_ = rollout.Close()
	}()
	group.Wait()
}

func TestSettleReceiptWithoutCleanupInterfaceDoesNotRequireRunnerCleanup(t *testing.T) {
	operations := &memoryOperationStore{items: map[string]ComposeUpdateOperation{
		"update-85": {OperationID: "update-85", SourceVersion: "1.0.0", TargetVersion: "1.1.0", TaskID: 85, Outcome: ExecutionOutcomeSuccess},
	}}
	rollout := NewRolloutService(NewService(nil), operations, &stubTaskService{}, &stubBackupService{}, &recordingLauncher{})
	if _, err := rollout.settleReceiptAndCleanup(t.Context(), RunnerReceipt{OperationID: "update-85"}); err != nil {
		t.Fatalf("settle receipt without optional cleanup: %v", err)
	}
}

func TestComposePreflightPreservesFrozenCandidateConfigFiles(t *testing.T) {
	root := t.TempDir()
	serverImage := "ghcr.io/gewuyou/graft-server"
	webImage := "ghcr.io/gewuyou/graft-web"
	runnerImage := "ghcr.io/gewuyou/graft-compose-runner"
	serverDigest := "sha256:" + strings.Repeat("a", 64)
	webDigest := "sha256:" + strings.Repeat("b", 64)
	runnerDigest := "sha256:" + strings.Repeat("c", 64)
	release := Release{Version: "1.1.0-beta.1", ServerImage: serverImage, WebImage: webImage, RunnerImage: runnerImage, ServerDigest: serverDigest, WebDigest: webDigest, RunnerDigest: runnerDigest, ServerRef: serverImage + "@" + serverDigest, WebRef: webImage + "@" + webDigest, RunnerRef: runnerImage + "@" + runnerDigest}
	files := []string{filepath.Join(root, "compose.yaml"), filepath.Join(root, "overrides", "web.yml")}
	candidate := moduleapi.NewDeploymentComposeCandidate("compose-selected", root, files, "graft", "high", nil)
	snapshot := moduleapi.NewDeploymentSnapshot(moduleapi.NewDeploymentContext("compose", "docker_discovered", false, []moduleapi.DeploymentComposeCandidate{candidate}, nil), candidate, "test-fingerprint")
	preflight, err := composePreflight(snapshot, release, ResolvedDeploymentStrategy{ImageTag: "beta", Mode: DeploymentStrategyBetaTracking, Channel: "beta", Tracking: true})
	if err != nil {
		t.Fatalf("build compose preflight: %v", err)
	}
	if !slices.Equal(preflight.ComposeFiles, files) {
		t.Fatalf("candidate config file sequence was not passed to runner input: got %#v want %#v", preflight.ComposeFiles, files)
	}
	if preflight.ServerReference != serverImage+":v"+release.Version || preflight.WebReference != webImage+":v"+release.Version {
		t.Fatalf("preflight image references = (%q, %q), want canonical v-prefixed release tags", preflight.ServerReference, preflight.WebReference)
	}
}

func TestComposePreflightUsesFrozenUniqueCandidate(t *testing.T) {
	root := t.TempDir()
	release := Release{Version: "1.1.0-beta.1", ServerImage: "ghcr.io/gewuyou/graft-server", WebImage: "ghcr.io/gewuyou/graft-web", RunnerImage: "ghcr.io/gewuyou/graft-compose-runner", ServerDigest: "sha256:" + strings.Repeat("a", 64), WebDigest: "sha256:" + strings.Repeat("b", 64), RunnerDigest: "sha256:" + strings.Repeat("c", 64)}
	release.ServerRef, release.WebRef, release.RunnerRef = release.ServerImage+"@"+release.ServerDigest, release.WebImage+"@"+release.WebDigest, release.RunnerImage+"@"+release.RunnerDigest
	candidate := moduleapi.NewDeploymentComposeCandidate("compose-unique", root, []string{filepath.Join(root, "compose.yml")}, "graft", "high", nil)
	snapshot := moduleapi.NewDeploymentSnapshot(moduleapi.NewDeploymentContext("compose", "docker_discovered", false, []moduleapi.DeploymentComposeCandidate{candidate}, nil), candidate, "test-fingerprint")
	preflight, err := composePreflight(snapshot, release, ResolvedDeploymentStrategy{ImageTag: "beta", Mode: DeploymentStrategyBetaTracking, Channel: "beta", Tracking: true})
	if err != nil || preflight.ComposeRoot != root {
		t.Fatalf("expected unique candidate preflight, got %#v, %v", preflight, err)
	}
}

func TestRolloutRejectsStaleCatalogBeforeCreatingTask(t *testing.T) {
	discovery := NewService(nil)
	discovery.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0"} }
	discovery.profile = func() InstallationProfile {
		return InstallationProfile{DeclaredMode: "compose", DetectedMode: "compose", Capability: "compose_upgrade_available"}
	}
	discovery.latest = &Release{Version: "1.1.0"}
	old := time.Now().UTC().Add(-discoveryCacheStaleAfter - time.Minute)
	discovery.lastSuccessfulAt = &old
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 99}}
	rollout := NewRolloutService(discovery, &memoryOperationStore{}, tasks, &stubBackupService{}, &recordingLauncher{})
	if _, err := rollout.Start(t.Context(), StartRolloutInput{RequestedBy: 9, TargetVersion: "1.1.0", CandidateKey: "1.1.0"}); err == nil {
		t.Fatal("expected stale release catalog to reject rollout")
	}
	if tasks.plan.Type != "" {
		t.Fatal("stale catalog must not submit a task")
	}
}

func TestRolloutRejectsBelowManifestMinimumSourceVersion(t *testing.T) {
	discovery := NewService(nil)
	discovery.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0"} }
	discovery.profile = func() InstallationProfile {
		return InstallationProfile{DeclaredMode: "compose", DetectedMode: "compose", Capability: "compose_upgrade_available"}
	}
	discovery.latest = &Release{Version: "1.1.0", MinimumSourceVersion: "1.0.1"}
	fresh := time.Now().UTC()
	discovery.lastSuccessfulAt = &fresh
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 100}}
	rollout := NewRolloutService(discovery, &memoryOperationStore{}, tasks, &stubBackupService{}, &recordingLauncher{})
	if _, err := rollout.Start(t.Context(), StartRolloutInput{RequestedBy: 9, TargetVersion: "1.1.0", CandidateKey: "1.1.0"}); err == nil {
		t.Fatal("expected minimum source version to reject rollout")
	}
	if tasks.plan.Type != "" {
		t.Fatal("ineligible source version must not submit a task")
	}
}

func TestComposeRunnerContainerConfigUsesOnlyFrozenMountsAndDigestImage(t *testing.T) {
	input := fixtureRunnerInput("/opt/graft")
	config, host := composeRunnerContainerConfig(input, "/opt/graft/.graft-update/inputs/fixture-operation-1.json", "graft-update-state")
	if config.Image != input.Preflight.RunnerReference || len(config.Env) != 1 || config.Env[0] != "GRAFT_UPDATE_RUNNER_INPUT_B64=/opt/graft/.graft-update/inputs/fixture-operation-1.json" {
		t.Fatalf("runner config is not constrained: %#v", config)
	}
	if len(host.Binds) != 3 || host.Binds[0] != "/opt/graft:/opt/graft:rw" || host.Binds[1] != "/var/run/docker.sock:/var/run/docker.sock:rw" || host.Binds[2] != "graft-update-state:"+RunnerStateRoot+":rw" || host.NetworkMode != "none" || host.AutoRemove {
		t.Fatalf("runner host config is not constrained: %#v", host)
	}
}

func TestComposeRunnerContainerConfigAllowsOnlyChownForStateOwnership(t *testing.T) {
	_, host := composeRunnerContainerConfig(fixtureRunnerInput("/opt/graft"), "runner-input", "graft-update-state")
	if len(host.CapDrop) != 1 || host.CapDrop[0] != "ALL" || len(host.CapAdd) != 1 || host.CapAdd[0] != "CHOWN" || len(host.SecurityOpt) != 1 || host.SecurityOpt[0] != "no-new-privileges:true" {
		t.Fatalf("runner capability hardening changed: %#v", host)
	}
}

func TestComposeRunnerRecoveryContainerConfigOnlyMountsStateVolume(t *testing.T) {
	state := NewRunnerState(RunnerInput{OperationID: "update-recovery-config", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Preflight: ComposePreflight{DeploymentStrategy: DeploymentStrategyBetaTracking}}, "runner-recovery-config", RunnerPhaseReady, 0, "runner_accepted", "", RunnerState{})
	config, host := composeRunnerRecoveryContainerConfig(RunnerRecoveryInput{OperationID: state.OperationID, RunnerID: state.RunnerID, SourceVersion: state.SourceVersion, TargetVersion: state.TargetVersion, Strategy: state.Strategy, State: &state}, "ghcr.io/gewuyou/graft-compose-runner@sha256:"+strings.Repeat("a", 64), "bound-state", "graft-update-state")
	if config.Image == "" || len(config.Env) != 1 || config.Env[0] != "GRAFT_UPDATE_RUNNER_RECOVERY_STATE_B64=bound-state" || config.Labels["io.graft.update.recovery"] != "true" {
		t.Fatalf("recovery runner config is not bound: %#v", config)
	}
	assertRecoveryRunnerHardening(t, config.User, host)
}

func assertRecoveryRunnerHardening(t *testing.T, user string, host containertypes.HostConfig) {
	t.Helper()
	if user != "0:0" || len(host.Binds) != 1 || host.Binds[0] != "graft-update-state:"+RunnerStateRoot+":rw" || host.NetworkMode != "none" || !host.ReadonlyRootfs || host.AutoRemove {
		t.Fatalf("recovery runner host config exceeds state-only scope: %#v", host)
	}
	if len(host.CapDrop) != 1 || host.CapDrop[0] != "ALL" || len(host.CapAdd) != 2 || host.CapAdd[0] != "CHOWN" || host.CapAdd[1] != "DAC_OVERRIDE" || len(host.SecurityOpt) != 1 || host.SecurityOpt[0] != "no-new-privileges:true" {
		t.Fatalf("recovery runner capability hardening changed: %#v", host)
	}
}

func TestSQLOperationStorePersistsHistoryWithoutReceiptContent(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE update_operations (operation_id TEXT PRIMARY KEY, runner_id TEXT, request_id TEXT, source_version TEXT, target_version TEXT, deployment_strategy TEXT, task_id INTEGER, backup_id INTEGER, requested_by INTEGER, status TEXT, receipt_integrity_sha256 TEXT, failure_code TEXT, recovery_completed BOOLEAN, recovery_claim_id TEXT, recovery_claimed_at TIMESTAMP, created_at TIMESTAMP, started_at TIMESTAMP, updated_at TIMESTAMP, finished_at TIMESTAMP);
CREATE TABLE update_failure_diagnostics (request_id TEXT PRIMARY KEY, operation_id TEXT, task_id INTEGER, target_version TEXT, failure_code TEXT, failure_stage TEXT, summary TEXT, detail TEXT, occurred_at TIMESTAMP)`); err != nil {
		t.Fatalf("create update operations: %v", err)
	}
	store, err := newSQLOperationStore(db)
	if err != nil {
		t.Fatalf("new operation store: %v", err)
	}
	created := ComposeUpdateOperation{OperationID: "update-history-1", RunnerID: "runner-history-1", SourceVersion: "1.0.0", TargetVersion: "1.1.0", DeploymentStrategy: DeploymentStrategyBetaTracking, TaskID: 9, RequestedBy: 3, Outcome: ExecutionOutcomePulling}
	if err := store.Create(t.Context(), created); err != nil {
		t.Fatalf("create operation: %v", err)
	}
	created.Outcome, created.BackupID, created.FailureCode, created.ReceiptIntegritySHA256 = ExecutionOutcomeNeedsAttention, 7, "healthz_failed", strings.Repeat("a", 64)
	if err := store.Settle(t.Context(), created); err != nil {
		t.Fatalf("settle operation: %v", err)
	}
	loaded, err := store.Get(t.Context(), created.OperationID)
	if err != nil {
		t.Fatalf("get operation: %v", err)
	}
	if loaded.RunnerID != "runner-history-1" || loaded.DeploymentStrategy != DeploymentStrategyBetaTracking || loaded.Outcome != ExecutionOutcomeNeedsAttention || loaded.BackupID != 7 || loaded.FailureCode != "healthz_failed" || loaded.ReceiptIntegritySHA256 != strings.Repeat("a", 64) {
		t.Fatalf("unexpected durable history: %#v", loaded)
	}
}

func TestSQLOperationStoreRejectsUnsupportedDeploymentStrategy(t *testing.T) {
	store, err := newSQLOperationStore(&sql.DB{})
	if err != nil {
		t.Fatalf("new operation store: %v", err)
	}
	err = store.Create(t.Context(), ComposeUpdateOperation{
		OperationID:        "update-invalid-mode",
		SourceVersion:      "1.0.0",
		TargetVersion:      "1.1.0",
		DeploymentStrategy: "unsupported",
		TaskID:             1,
		Outcome:            ExecutionOutcomePlanning,
	})
	if err == nil || err.Error() != "update operation is invalid" {
		t.Fatalf("expected invalid deployment strategy to be rejected, got %v", err)
	}
}

func TestSettlePersistedReceiptStoresControlledOperationDiagnostic(t *testing.T) {
	operations := &memoryOperationStore{items: map[string]ComposeUpdateOperation{
		"update-86": {OperationID: "update-86", RequestID: "request-86", SourceVersion: "1.0.0", TargetVersion: "1.1.0", TaskID: 86, RequestedBy: 9, Outcome: ExecutionOutcomePulling},
	}}
	diagnostics := &failureDiagnosticStoreRecorder{}
	rollout := NewRolloutService(NewService(nil), operations, &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 86}}, &stubBackupService{}, &recordingLauncher{})
	rollout.SetFailureDiagnosticStore(diagnostics)
	_, err := rollout.SettlePersistedReceipt(t.Context(), RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: "update-86", FailureCode: runnerFailureBackup, FailureStage: string(RunnerBackupFailureStageConfigSnapshot), FailureDetail: string(RunnerBackupFailureDetailPermissionDenied)})
	if err != nil {
		t.Fatalf("settle failed receipt: %v", err)
	}
	if diagnostics.value.OperationID != "update-86" || diagnostics.value.RequestID != "request-86" || diagnostics.value.FailureCode != rolloutFailureRunnerTerminal || diagnostics.value.FailureStage != string(RunnerBackupFailureStageConfigSnapshot) {
		t.Fatalf("unexpected controlled terminal diagnostic: %#v", diagnostics.value)
	}
	if diagnostics.value.Detail != "deployment environment snapshot was denied by deployment filesystem permissions" || strings.Contains(diagnostics.value.Detail, "stderr") {
		t.Fatalf("terminal diagnostic must not expose runner logs: %q", diagnostics.value.Detail)
	}
}

func TestRolloutAuditCarriesRequestIDAndLogsPublishFailure(t *testing.T) {
	observed, logs := observer.New(zap.ErrorLevel)
	publisher := &rolloutAuditPublisher{err: errors.New("outbox unavailable")}
	rollout := NewRolloutService(nil, nil, nil, nil, nil)
	rollout.SetAuditPublisher(publisher, zap.New(observed))
	ctx := httpx.WithRequestAuditContext(context.Background(), httpx.RequestAuditContext{
		RequestID: "req-update-1",
		Method:    "POST",
		Route:     "/api/platform/update/compose",
		ClientIP:  "198.51.100.7",
		UserAgent: "update-test",
	})

	rollout.publishAudit(ctx, ComposeUpdateOperation{OperationID: "update-audit-1", SourceVersion: "1.0.0", TargetVersion: "1.1.0", DeploymentStrategy: DeploymentStrategyBetaTracking, TaskID: 7, Outcome: ExecutionOutcomePulling}, true, "")
	if publisher.options.Delivery != event.DeliveryDurable {
		t.Fatalf("expected durable audit delivery, got %q", publisher.options.Delivery)
	}
	var payload moduleapi.AuditEvent
	if err := json.Unmarshal(publisher.event.Payload, &payload); err != nil {
		t.Fatalf("decode audit payload: %v", err)
	}
	if payload.RequestID != "req-update-1" || payload.RequestMethod != "POST" || payload.RequestPath != "/api/platform/update/compose" || payload.IP != "198.51.100.7" || payload.UserAgent != "update-test" {
		t.Fatalf("expected request correlation in audit payload, got %#v", payload)
	}
	if payload.Metadata["deployment_strategy"] != string(DeploymentStrategyBetaTracking) {
		t.Fatalf("expected deployment strategy audit snapshot, got %#v", payload.Metadata)
	}
	if entries := logs.FilterMessage("publish update rollout audit event failed").All(); len(entries) != 1 {
		t.Fatalf("expected one audit publish failure log, got %#v", entries)
	}
}

type memoryOperationStore struct {
	items          map[string]ComposeUpdateOperation
	recoveryClaims map[string]string
}

type rolloutAuditPublisher struct {
	event   event.Event
	options event.PublishOptions
	err     error
}

func (p *rolloutAuditPublisher) Publish(_ context.Context, current event.Event, options event.PublishOptions) (event.Receipt, error) {
	p.event = current
	p.options = options
	return event.Receipt{}, p.err
}

func (p *rolloutAuditPublisher) PublishAsync(current event.Event, options event.PublishOptions) (event.Receipt, error) {
	return p.Publish(context.Background(), current, options)
}

func (*rolloutAuditPublisher) PublishBatch(context.Context, []event.Event, event.PublishOptions) event.BatchReceipt {
	return event.BatchReceipt{}
}

func (s *memoryOperationStore) Create(_ context.Context, item ComposeUpdateOperation) error {
	if s.items == nil {
		s.items = map[string]ComposeUpdateOperation{}
	}
	s.items[item.OperationID] = item
	return nil
}
func (s *memoryOperationStore) Get(_ context.Context, id string) (ComposeUpdateOperation, error) {
	item, ok := s.items[id]
	if !ok {
		return ComposeUpdateOperation{}, errUpdateOperationNotFound
	}
	return item, nil
}
func (s *memoryOperationStore) List(_ context.Context, limit int) ([]ComposeUpdateOperation, error) {
	items := make([]ComposeUpdateOperation, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
		if len(items) == limit {
			break
		}
	}
	return items, nil
}
func (s *memoryOperationStore) Settle(_ context.Context, item ComposeUpdateOperation) error {
	s.items[item.OperationID] = item
	return nil
}
func (s *memoryOperationStore) ClaimRecovery(_ context.Context, operationID, claimID string) (bool, error) {
	if _, ok := s.items[operationID]; !ok {
		return false, errUpdateOperationNotFound
	}
	if s.recoveryClaims == nil {
		s.recoveryClaims = map[string]string{}
	}
	if s.recoveryClaims[operationID] != "" {
		return false, nil
	}
	s.recoveryClaims[operationID] = claimID
	return true, nil
}
func (s *memoryOperationStore) ReleaseRecoveryClaim(_ context.Context, operationID, claimID string) error {
	if s.recoveryClaims[operationID] == claimID {
		delete(s.recoveryClaims, operationID)
	}
	return nil
}
func (s *memoryOperationStore) RecoveryClaim(_ context.Context, operationID string) (string, error) {
	if _, ok := s.items[operationID]; !ok {
		return "", errUpdateOperationNotFound
	}
	return s.recoveryClaims[operationID], nil
}

type recordingLauncher struct {
	input     RunnerInput
	launchErr error
}

func (l *recordingLauncher) Launch(_ context.Context, input RunnerInput) error {
	l.input = input
	if l.launchErr != nil {
		return l.launchErr
	}
	return nil
}

func (*recordingLauncher) Close() error { return nil }

type recoveryLauncher struct {
	failures                []RunnerFailureEvidence
	recovered               RunnerState
	recoveryErr             error
	recoveryContainerErr    error
	failureReads            int
	recoveryContainerExists bool
}

func (*recoveryLauncher) Launch(context.Context, RunnerInput) error { return nil }
func (*recoveryLauncher) Close() error                              { return nil }
func (l *recoveryLauncher) RecoveryContainerExists(context.Context, string, string) (bool, error) {
	return l.recoveryContainerExists, l.recoveryContainerErr
}
func (l *recoveryLauncher) ReadRunnerFailures(context.Context) ([]RunnerFailureEvidence, error) {
	l.failureReads++
	return l.failures, nil
}
func (l *recoveryLauncher) LaunchRecovery(_ context.Context, input RunnerRecoveryInput, _, _ string) error {
	if input.State != nil {
		l.recovered = *input.State
	} else {
		l.recovered = RunnerState{OperationID: input.OperationID, RunnerID: input.RunnerID}
	}
	if l.recoveryErr == nil {
		l.recoveryContainerExists = true
	}
	return l.recoveryErr
}

type receiptReaderCleanupLauncher struct {
	receipts []RunnerReceipt
	removed  []string
}

type blockingReceiptReaderLauncher struct {
	mu      sync.Mutex
	started chan struct{}
	closed  chan struct{}
}

func (*blockingReceiptReaderLauncher) Launch(context.Context, RunnerInput) error { return nil }
func (l *blockingReceiptReaderLauncher) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
	return nil
}
func (l *blockingReceiptReaderLauncher) ReadRunnerReceipts(ctx context.Context) ([]RunnerReceipt, error) {
	l.mu.Lock()
	select {
	case <-l.started:
	default:
		close(l.started)
	}
	l.mu.Unlock()
	<-ctx.Done()
	return nil, ctx.Err()
}

func (*receiptReaderCleanupLauncher) Launch(context.Context, RunnerInput) error { return nil }
func (*receiptReaderCleanupLauncher) Close() error                              { return nil }
func (l *receiptReaderCleanupLauncher) ReadRunnerReceipts(context.Context) ([]RunnerReceipt, error) {
	return l.receipts, nil
}
func (l *receiptReaderCleanupLauncher) RemoveRunner(_ context.Context, operationID string) error {
	l.removed = append(l.removed, operationID)
	return nil
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

type stubTaskService struct {
	receipt  moduleapi.TaskReceipt
	plan     moduleapi.SubmitTaskInput
	external moduleapi.ExternalTaskReceipt
	canceled uint64
}

func (s *stubTaskService) Submit(_ context.Context, input moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	s.plan = input
	return s.receipt, nil
}
func (s *stubTaskService) SettleExternalReceipt(_ context.Context, receipt moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	s.external = receipt
	return moduleapi.ExternalReceiptSettlement{TaskID: receipt.TaskID}, nil
}
func (s *stubTaskService) Cancel(_ context.Context, taskID uint64) error {
	s.canceled = taskID
	return nil
}
func (*stubTaskService) RetryStage(context.Context, uint64, uint64) error { return nil }

type taskQueryStub struct {
	tasks map[uint64]moduleapi.TaskView
	err   error
}

func (s taskQueryStub) GetTask(_ context.Context, taskID uint64) (moduleapi.TaskView, error) {
	if s.err != nil {
		return moduleapi.TaskView{}, s.err
	}
	task, ok := s.tasks[taskID]
	if !ok {
		return moduleapi.TaskView{}, errors.New("task not found")
	}
	return task, nil
}

func (taskQueryStub) ListTasks(context.Context, moduleapi.TaskListFilter, int, int) ([]moduleapi.TaskView, int64, error) {
	return nil, 0, nil
}
func (taskQueryStub) ListTaskStages(context.Context, uint64) ([]moduleapi.TaskStageView, error) {
	return nil, nil
}
func (taskQueryStub) ListTaskEvents(context.Context, uint64, int64, int) ([]moduleapi.TaskEventView, error) {
	return nil, nil
}
func (taskQueryStub) ListTaskLogs(context.Context, uint64, int64, int) ([]moduleapi.TaskLogView, error) {
	return nil, nil
}

type stubBackupService struct {
	plan       moduleapi.BackupRunnerHandoffPlan
	prepared   *moduleapi.BackupRunnerHandoffPlan
	completion moduleapi.CompleteBackupRunnerHandoffInput
	canceled   moduleapi.BackupRunnerHandoffPlan
}

func (s *stubBackupService) Create(context.Context, moduleapi.CreateBackupInput) (moduleapi.Backup, error) {
	return moduleapi.Backup{}, errors.New("unused")
}
func (s *stubBackupService) PrepareRunnerHandoff(_ context.Context, plan moduleapi.BackupRunnerHandoffPlan) (moduleapi.BackupRunnerHandoffPlan, error) {
	s.plan = plan
	if s.prepared != nil {
		return *s.prepared, nil
	}
	return plan, nil
}
func (s *stubBackupService) CancelRunnerHandoff(_ context.Context, operationID string, taskID uint64) error {
	s.canceled = moduleapi.BackupRunnerHandoffPlan{OperationID: operationID, TaskID: taskID}
	return nil
}
func (s *stubBackupService) CompleteRunnerHandoff(_ context.Context, input moduleapi.CompleteBackupRunnerHandoffInput) (moduleapi.BackupRunnerHandoffCompletion, error) {
	s.completion = input
	return moduleapi.BackupRunnerHandoffCompletion{BackupID: 8, OperationID: input.OperationID, TaskID: input.TaskID}, nil
}
func (*stubBackupService) Get(context.Context, uint64) (moduleapi.Backup, error) {
	return moduleapi.Backup{}, errors.New("unused")
}
func (*stubBackupService) RecordRestoreEvidence(context.Context, moduleapi.RecordBackupRestoreInput) (moduleapi.Backup, error) {
	return moduleapi.Backup{}, errors.New("unused")
}

func testBackupPlan(operationID string) moduleapi.BackupRunnerHandoffPlan {
	return moduleapi.BackupRunnerHandoffPlan{OperationID: operationID, Purpose: "before-update", RetainUntil: time.Now().Add(time.Hour), ArtifactRoot: "/var/lib/graft/backups", ConfigSnapshotRef: "/var/lib/graft/backups/config", DatabaseDumpRef: "/var/lib/graft/backups/dump"}
}
func testDigest(character rune) string { return string(makeDigest(character)) }
func makeDigest(character rune) []rune {
	value := make([]rune, 64)
	for index := range value {
		value[index] = character
	}
	return value
}
