package update

import (
	"errors"
	"os"
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
	contents, err := os.ReadFile(store.root + "/current.json")
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	contents = append(contents[:len(contents)-2], []byte("x\"}")...)
	if err := os.WriteFile(store.root+"/current.json", contents, 0o600); err != nil {
		t.Fatalf("tamper state: %v", err)
	}
	if _, err := store.Read(); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("expected digest mismatch, got %v", err)
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
