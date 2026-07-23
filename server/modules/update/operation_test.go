package update

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"graft/server/internal/buildinfo"
	"graft/server/internal/moduleapi"
)

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
	if err == nil {
		t.Fatal("expected invalid prepared handoff rejection")
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

func TestRolloutRequiresExactConfirmationAndPersistsLauncherOperation(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GRAFT_UPDATE_COMPOSE_ROOT", root)
	discovery := NewService(nil)
	discovery.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0"} }
	discovery.profile = func() InstallationProfile {
		return InstallationProfile{DeclaredMode: "compose", DetectedMode: "compose", Capability: "compose_upgrade_available"}
	}
	discovery.latest = &Release{Version: "1.1.0", ServerImage: "ghcr.io/gewuyou/graft-server", WebImage: "ghcr.io/gewuyou/graft-web", RunnerImage: "ghcr.io/gewuyou/graft-compose-runner", ServerDigest: "sha256:" + strings.Repeat("a", 64), WebDigest: "sha256:" + strings.Repeat("b", 64), RunnerDigest: "sha256:" + strings.Repeat("c", 64)}
	discovery.latest.ServerRef = discovery.latest.ServerImage + "@" + discovery.latest.ServerDigest
	discovery.latest.WebRef = discovery.latest.WebImage + "@" + discovery.latest.WebDigest
	discovery.latest.RunnerRef = discovery.latest.RunnerImage + "@" + discovery.latest.RunnerDigest
	fresh := time.Now().UTC()
	discovery.lastSuccessfulAt = &fresh
	operations := &memoryOperationStore{}
	launcher := &recordingLauncher{}
	rollout := NewRolloutService(discovery, operations, &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 77}}, &stubBackupService{}, launcher)
	rollout.newOperation = func() string { return "update-77" }
	if _, err := rollout.Start(t.Context(), 9, "1.1.0", "wrong"); err == nil {
		t.Fatal("expected confirmation rejection")
	}
	operation, err := rollout.Start(t.Context(), 9, "1.1.0", "1.1.0")
	if err != nil {
		t.Fatalf("start rollout: %v", err)
	}
	if operation.Outcome != ExecutionOutcomePulling || operation.TaskID != 77 || launcher.input.Preflight.ComposeRoot != root {
		t.Fatalf("unexpected rollout operation: %#v / %#v", operation, launcher.input)
	}
	if filepath.Dir(filepath.Dir(filepath.Dir(launcher.inputPath))) != root || operations.items[operation.OperationID].TaskID != 77 {
		t.Fatalf("runner input or persisted operation lost constrained identity: %q %#v", launcher.inputPath, operations.items)
	}
}

func TestRolloutLaunchFailureCancelsTaskAndBackupHandoffThroughCapabilities(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GRAFT_UPDATE_COMPOSE_ROOT", root)
	discovery := NewService(nil)
	discovery.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0"} }
	discovery.profile = func() InstallationProfile {
		return InstallationProfile{DeclaredMode: "compose", DetectedMode: "compose", Capability: "compose_upgrade_available"}
	}
	discovery.latest = &Release{Version: "1.1.0", ServerImage: "ghcr.io/gewuyou/graft-server", WebImage: "ghcr.io/gewuyou/graft-web", RunnerImage: "ghcr.io/gewuyou/graft-compose-runner", ServerDigest: "sha256:" + strings.Repeat("a", 64), WebDigest: "sha256:" + strings.Repeat("b", 64), RunnerDigest: "sha256:" + strings.Repeat("c", 64)}
	discovery.latest.ServerRef = discovery.latest.ServerImage + "@" + discovery.latest.ServerDigest
	discovery.latest.WebRef = discovery.latest.WebImage + "@" + discovery.latest.WebDigest
	discovery.latest.RunnerRef = discovery.latest.RunnerImage + "@" + discovery.latest.RunnerDigest
	fresh := time.Now().UTC()
	discovery.lastSuccessfulAt = &fresh
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 78}}
	backups := &stubBackupService{}
	operations := &memoryOperationStore{}
	rollout := NewRolloutService(discovery, operations, tasks, backups, &recordingLauncher{launchErr: errors.New("docker unavailable")})
	rollout.newOperation = func() string { return "update-78" }
	if _, err := rollout.Start(t.Context(), 9, "1.1.0", "1.1.0"); err == nil {
		t.Fatal("expected launcher failure")
	}
	if tasks.canceled != 78 || backups.canceled.OperationID != "update-78" || backups.canceled.TaskID != 78 {
		t.Fatalf("owner capability cleanup was not requested: task=%d backup=%#v", tasks.canceled, backups.canceled)
	}
	if item := operations.items["update-78"]; item.Outcome != ExecutionOutcomeFailed || item.FailureCode != "runner_launch_failed" {
		t.Fatalf("operation failure was not persisted: %#v", item)
	}
}

func TestSettleReceiptEntryDeletesOnlySuccessfulSettlements(t *testing.T) {
	root := t.TempDir()
	operations := &memoryOperationStore{items: map[string]ComposeUpdateOperation{
		"update-81": {OperationID: "update-81", SourceVersion: "1.0.0", TargetVersion: "1.1.0", TaskID: 81, Outcome: ExecutionOutcomePulling},
		"update-82": {OperationID: "update-82", SourceVersion: "1.0.0", TargetVersion: "1.1.0", TaskID: 82, Outcome: ExecutionOutcomePulling},
	}}
	rollout := NewRolloutService(NewService(nil), operations, &stubTaskService{}, &stubBackupService{}, &recordingLauncher{})
	for _, receipt := range []RunnerReceipt{
		{ProtocolVersion: runnerProtocolVersion, OperationID: "update-81", Succeeded: true},
		{ProtocolVersion: runnerProtocolVersion, OperationID: "update-82", FailureCode: "pull_failed"},
	} {
		input := fixtureRunnerInput(root)
		input.OperationID = receipt.OperationID
		if err := persistRunnerReceipt(input, receipt); err != nil {
			t.Fatalf("persist receipt %s: %v", receipt.OperationID, err)
		}
		entry := mustReceiptEntry(t, root, receipt.OperationID+".json")
		if err := rollout.settleReceiptEntry(t.Context(), root, entry); err != nil {
			t.Fatalf("settle receipt %s: %v", receipt.OperationID, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, runnerReceiptDirectory, "update-81.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("successful receipt should be removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, runnerReceiptDirectory, "update-82.json")); err != nil {
		t.Fatalf("failed receipt should remain for retry, got %v", err)
	}
}

func mustReceiptEntry(t *testing.T, root, name string) os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, runnerReceiptDirectory))
	if err != nil {
		t.Fatalf("list receipts: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == name {
			return entry
		}
	}
	t.Fatalf("receipt %s was not created", name)
	return nil
}

func TestRolloutRejectsStaleCatalogBeforeCreatingTask(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GRAFT_UPDATE_COMPOSE_ROOT", root)
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
	if _, err := rollout.Start(t.Context(), 9, "1.1.0", "1.1.0"); err == nil {
		t.Fatal("expected stale release catalog to reject rollout")
	}
	if tasks.plan.Type != "" {
		t.Fatal("stale catalog must not submit a task")
	}
}

func TestRolloutRejectsBelowManifestMinimumSourceVersion(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GRAFT_UPDATE_COMPOSE_ROOT", root)
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
	if _, err := rollout.Start(t.Context(), 9, "1.1.0", "1.1.0"); err == nil {
		t.Fatal("expected minimum source version to reject rollout")
	}
	if tasks.plan.Type != "" {
		t.Fatal("ineligible source version must not submit a task")
	}
}

func TestComposeRunnerContainerConfigUsesOnlyFrozenMountsAndDigestImage(t *testing.T) {
	input := fixtureRunnerInput("/opt/graft")
	config, host := composeRunnerContainerConfig(input, "/opt/graft/.graft-update/inputs/fixture-operation-1.json")
	if config.Image != input.Preflight.RunnerReference || len(config.Env) != 1 || config.Env[0] != "GRAFT_UPDATE_RUNNER_INPUT=/opt/graft/.graft-update/inputs/fixture-operation-1.json" {
		t.Fatalf("runner config is not constrained: %#v", config)
	}
	if len(host.Binds) != 2 || host.Binds[0] != "/opt/graft:/opt/graft:rw" || host.Binds[1] != "/var/run/docker.sock:/var/run/docker.sock:rw" || host.NetworkMode != "none" {
		t.Fatalf("runner host config is not constrained: %#v", host)
	}
}

func TestSQLOperationStorePersistsHistoryWithoutReceiptContent(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE update_operations (operation_id TEXT PRIMARY KEY, source_version TEXT, target_version TEXT, task_id INTEGER, backup_id INTEGER, requested_by INTEGER, status TEXT, receipt_integrity_sha256 TEXT, failure_code TEXT, recovery_completed BOOLEAN, created_at TIMESTAMP, started_at TIMESTAMP, finished_at TIMESTAMP)`); err != nil {
		t.Fatalf("create update operations: %v", err)
	}
	store, err := newSQLOperationStore(db)
	if err != nil {
		t.Fatalf("new operation store: %v", err)
	}
	created := ComposeUpdateOperation{OperationID: "update-history-1", SourceVersion: "1.0.0", TargetVersion: "1.1.0", TaskID: 9, RequestedBy: 3, Outcome: ExecutionOutcomePulling}
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
	if loaded.Outcome != ExecutionOutcomeNeedsAttention || loaded.BackupID != 7 || loaded.FailureCode != "healthz_failed" || loaded.ReceiptIntegritySHA256 != strings.Repeat("a", 64) {
		t.Fatalf("unexpected durable history: %#v", loaded)
	}
}

type memoryOperationStore struct {
	items map[string]ComposeUpdateOperation
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
func (s *memoryOperationStore) List(context.Context, int) ([]ComposeUpdateOperation, error) {
	return nil, nil
}
func (s *memoryOperationStore) Settle(_ context.Context, item ComposeUpdateOperation) error {
	s.items[item.OperationID] = item
	return nil
}

type recordingLauncher struct {
	input     RunnerInput
	inputPath string
	launchErr error
}

func (l *recordingLauncher) Launch(_ context.Context, input RunnerInput) error {
	l.input = input
	if l.launchErr != nil {
		return l.launchErr
	}
	path, err := persistRunnerInput(input)
	l.inputPath = path
	return err
}

func (*recordingLauncher) Close() error { return nil }

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
