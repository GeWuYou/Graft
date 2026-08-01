package update

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileRunnerStateStorePublishesVerifiedMonotonicSnapshots(t *testing.T) {
	store, err := NewFileRunnerStateStore(t.TempDir())
	if err != nil {
		t.Fatalf("new state store: %v", err)
	}
	input := RunnerInput{OperationID: "update-state-1", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Preflight: ComposePreflight{DeploymentStrategy: DeploymentStrategyBetaTracking}}
	ready := NewRunnerState(input, "runner-state-1", RunnerPhaseReady, 0, "runner_accepted", "", RunnerState{})
	if err := store.Write(ready); err != nil {
		t.Fatalf("write ready state: %v", err)
	}
	if store.enforceOwnership {
		t.Fatal("local state store must not require production volume ownership")
	}
	preflight := NewRunnerState(input, "runner-state-1", RunnerPhasePreflight, 5, "checking_environment", "", ready)
	if err := store.Write(preflight); err != nil {
		t.Fatalf("write preflight state: %v", err)
	}
	got, err := store.Read()
	if err != nil || got.Revision != 2 || got.Phase != RunnerPhasePreflight || got.Digest == "" {
		t.Fatalf("state = %#v, %v", got, err)
	}
	if _, err := os.Stat(store.root + "/events/00000000000000000002.json"); err != nil {
		t.Fatalf("state event: %v", err)
	}
	if err := store.Write(ready); err == nil {
		t.Fatal("expected stale state revision to be rejected")
	}
}

func TestFileRunnerStateStoreRetriesAfterEventPublishFailure(t *testing.T) {
	store, err := NewFileRunnerStateStore(t.TempDir())
	if err != nil {
		t.Fatalf("new state store: %v", err)
	}
	input := RunnerInput{OperationID: "update-state-retry", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Preflight: ComposePreflight{DeploymentStrategy: DeploymentStrategyBetaTracking}}
	ready := NewRunnerState(input, "runner-state-retry", RunnerPhaseReady, 0, "runner_accepted", "", RunnerState{})
	if err := store.Write(ready); err != nil {
		t.Fatalf("write ready state: %v", err)
	}
	next := NewRunnerState(input, "runner-state-retry", RunnerPhasePreflight, 5, "checking_environment", "", ready)
	eventPath := filepath.Join(store.root, "events", "00000000000000000002.json")
	failOnce := true
	store.renameFile = func(oldPath, newPath string) error {
		if failOnce && newPath == eventPath {
			failOnce = false
			return errors.New("event publish unavailable")
		}
		return os.Rename(oldPath, newPath)
	}
	if err := store.Write(next); err == nil || !strings.Contains(err.Error(), "publish runner state event") {
		t.Fatalf("event publish failure = %v", err)
	}
	if got, err := store.Read(); err != nil || got.Revision != ready.Revision {
		t.Fatalf("current snapshot after failed publish = %#v, %v", got, err)
	}
	if err := store.Write(next); err != nil {
		t.Fatalf("retry state write: %v", err)
	}
	assertRunnerStateEventMatchesCurrent(t, store, eventPath, next)
}

func assertRunnerStateEventMatchesCurrent(t *testing.T, store *FileRunnerStateStore, eventPath string, want RunnerState) {
	t.Helper()
	current, err := os.ReadFile(filepath.Join(store.root, "current.json"))
	if err != nil {
		t.Fatalf("read current snapshot: %v", err)
	}
	// #nosec G304 -- eventPath 由 t.TempDir() 创建的测试状态卷推导。
	event, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatalf("read published event: %v", err)
	}
	if string(current) != strings.TrimSuffix(string(event), "\n") {
		t.Fatalf("event and current snapshots differ: event=%s current=%s", event, current)
	}
	var eventState RunnerState
	if err := json.Unmarshal(event, &eventState); err != nil {
		t.Fatalf("decode published event: %v", err)
	}
	if eventState.Revision != want.Revision || eventState.OperationID != want.OperationID {
		t.Fatalf("published event state = %#v", eventState)
	}
}

func TestNewFileRunnerStateStoreEnforcesOwnershipOnlyForOfficialVolume(t *testing.T) {
	store, err := NewFileRunnerStateStore(RunnerStateRoot)
	if err != nil {
		t.Fatalf("new official state store: %v", err)
	}
	if !store.enforceOwnership {
		t.Fatal("official state volume must enforce server ownership")
	}
}

func TestFileRunnerStateStoreRejectsCorruptedSnapshot(t *testing.T) {
	store, err := NewFileRunnerStateStore(t.TempDir())
	if err != nil {
		t.Fatalf("new state store: %v", err)
	}
	state := NewRunnerState(RunnerInput{OperationID: "update-state-2", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Preflight: ComposePreflight{DeploymentStrategy: DeploymentStrategyBetaTracking}}, "runner-state-2", RunnerPhaseSuccess, 100, "update_completed", "", RunnerState{})
	if err := store.Write(state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := os.WriteFile(store.root+"/current.json", []byte(`{"schema_version":1}`), 0o600); err != nil {
		t.Fatalf("corrupt state: %v", err)
	}
	if _, err := store.Read(); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected corrupted state error, got %v", err)
	}
}

func TestFileRunnerStateStoreRejectsDigestMismatch(t *testing.T) {
	store, err := NewFileRunnerStateStore(t.TempDir())
	if err != nil {
		t.Fatalf("new state store: %v", err)
	}
	state := NewRunnerState(RunnerInput{OperationID: "update-state-integrity", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Preflight: ComposePreflight{DeploymentStrategy: DeploymentStrategyBetaTracking}}, "runner-state-integrity", RunnerPhaseSuccess, 100, "update_completed", "", RunnerState{})
	if err := store.Write(state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(store.root, "current.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	contents = append(contents[:len(contents)-2], []byte("x\"}")...)
	// #nosec G703 -- store.root is created from t.TempDir().
	if err := os.WriteFile(filepath.Join(store.root, "current.json"), contents, 0o600); err != nil {
		t.Fatalf("tamper state: %v", err)
	}
	if _, err := store.Read(); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("expected digest mismatch, got %v", err)
	}
}

func TestFileRunnerStateStoreRejectsInvalidDeploymentStrategy(t *testing.T) {
	store, err := NewFileRunnerStateStore(t.TempDir())
	if err != nil {
		t.Fatalf("new state store: %v", err)
	}
	state := NewRunnerState(RunnerInput{OperationID: "update-state-strategy", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Preflight: ComposePreflight{DeploymentStrategy: DeploymentStrategyBetaTracking}}, "runner-state-strategy", RunnerPhaseReady, 0, "runner_accepted", "", RunnerState{})
	state.Strategy = "unsupported"
	if err := store.Write(state); err == nil {
		t.Fatal("expected invalid deployment strategy rejection")
	}
}

func TestRolloutReadsActiveRunnerStateBeforeDatabaseHistory(t *testing.T) {
	store, err := NewFileRunnerStateStore(t.TempDir())
	if err != nil {
		t.Fatalf("new state store: %v", err)
	}
	input := RunnerInput{OperationID: "update-state-3", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Preflight: ComposePreflight{DeploymentStrategy: DeploymentStrategyBetaTracking}}
	state := NewRunnerState(input, "runner-state-3", RunnerPhasePullImages, 30, "pulling_images", "", RunnerState{})
	if err := store.Write(state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	rollout := &RolloutService{stateStore: store}
	view, err := rollout.GetOperation(t.Context(), state.OperationID)
	if err != nil || view.Phase != RunnerPhasePullImages || view.Progress != 30 || view.RunnerID != state.RunnerID {
		t.Fatalf("runner state projection = %#v, %v", view, err)
	}
}

func TestRolloutGetOperationReturnsCorruptRunnerStateError(t *testing.T) {
	store, err := NewFileRunnerStateStore(t.TempDir())
	if err != nil {
		t.Fatalf("new state store: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.root, "current.json"), []byte(`{"schema_version":1}`), 0o600); err != nil {
		t.Fatalf("write corrupt state: %v", err)
	}
	rollout := &RolloutService{stateStore: store}
	if _, err := rollout.GetOperation(t.Context(), "update-state-corrupt"); err == nil || !strings.Contains(err.Error(), "read runner state") {
		t.Fatalf("get corrupt runner state error = %v", err)
	}
}

func TestFileRunnerStateStoreRejectsConcurrentOperation(t *testing.T) {
	store, err := NewFileRunnerStateStore(t.TempDir())
	if err != nil {
		t.Fatalf("new state store: %v", err)
	}
	first := NewRunnerState(RunnerInput{OperationID: "update-state-4", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Preflight: ComposePreflight{DeploymentStrategy: DeploymentStrategyBetaTracking}}, "runner-state-4", RunnerPhaseBackup, 15, "creating_backup", "", RunnerState{})
	if err := store.Write(first); err != nil {
		t.Fatalf("write active state: %v", err)
	}
	second := NewRunnerState(RunnerInput{OperationID: "update-state-5", SourceVersion: "1.1.0", TargetVersion: "1.2.0", Preflight: ComposePreflight{DeploymentStrategy: DeploymentStrategyBetaTracking}}, "runner-state-5", RunnerPhaseReady, 0, "runner_accepted", "", RunnerState{})
	if err := store.Write(second); err == nil {
		t.Fatal("expected active operation exclusion")
	}
}
