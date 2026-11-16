package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"graft/server/modules/update"
)

func persistExpiredControllerFixture(t *testing.T, store update.RunnerStateWriter, state update.RunnerState) update.RunnerState {
	t.Helper()
	state.LeaseHeartbeatAt, state.LeaseExpiresAt = time.Now().UTC().Add(-10*time.Minute), time.Now().UTC().Add(-5*time.Minute)
	if err := store.Write(state); err != nil {
		t.Fatal(err)
	}
	persisted, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	return persisted
}

func recoverControllerFixture(t *testing.T, store update.RunnerStateWriter, input update.RunnerRecoveryInput) update.RunnerState {
	t.Helper()
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := recoverTerminatedRunnerWithStore(base64.RawStdEncoding.EncodeToString(payload), store, io.Discard); err != nil {
		t.Fatal(err)
	}
	result, err := store.Read()
	if err != nil || result.Receipt == nil {
		t.Fatalf("recovered controller receipt unavailable: %#v / %v", result, err)
	}
	return result
}

func localRecoveryFixture(t *testing.T, root string, store update.RunnerStateWriter, scenario string) update.RunnerRecoveryInput {
	t.Helper()
	recovery := update.RunnerRecoveryInput{OperationID: "update-local-recovery", RunnerID: "runner-before",
		SourceVersion: "1.0.0", TargetVersion: "1.1.0", Strategy: string(update.DeploymentStrategyBetaTracking), ExecutionMode: "local", LaunchStageID: 30, LaunchAttempt: 3}
	if scenario == "missing" {
		return recovery
	}
	input := update.RunnerInput{OperationID: recovery.OperationID, SourceVersion: recovery.SourceVersion,
		TargetVersion: recovery.TargetVersion, ExecutionMode: "local", Preflight: update.ComposePreflight{DeploymentStrategy: update.DeploymentStrategyBetaTracking}}
	state := update.NewRunnerState(input, recovery.RunnerID, update.RunnerPhaseReady, 0, "runner_accepted", "", update.RunnerState{})
	state = persistExpiredControllerFixture(t, store, state)
	if scenario == "trusted" {
		recovery.State = &state
		// 已验证的快照拥有恢复模式，兼容缺少新增模式字段的恢复材料。
		recovery.ExecutionMode = ""
		return recovery
	}
	if err := os.WriteFile(filepath.Join(root, "current.json"), []byte(`{"schema_version":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	recovery.Corrupt = true
	return recovery
}
