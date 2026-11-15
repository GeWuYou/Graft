package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	composetypes "github.com/compose-spec/compose-go/v2/types"

	"graft/server/modules/update"
)

func TestControllerLoadsTheFrozenDeploymentProjectName(t *testing.T) {
	root := t.TempDir()
	composeFile := filepath.Join(root, "compose.yml")
	definition := "services:\n  server:\n    image: alpine:3.24.4\n  web:\n    image: alpine:3.24.4\n"
	if err := os.WriteFile(composeFile, []byte(definition), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("GRAFT_EXECUTION_MODE=local\nCOMPOSE_PROJECT_NAME=legacy-project\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := update.RunnerInput{ExecutionMode: "local", Preflight: update.ComposePreflight{ComposeRoot: root, ProjectName: "frozen-project", ComposeFiles: []string{composeFile}}}
	project, err := loadRunnerComposeProject(context.Background(), input, "v1.0.0")
	if err != nil || project.Name != "frozen-project" {
		t.Fatalf("frozen project identity lost: %#v / %v", project, err)
	}
	input.Preflight.ProjectName = ""
	project, err = loadRunnerComposeProject(context.Background(), input, "v1.0.0")
	if err != nil || project.Name != "legacy-project" {
		t.Fatalf("legacy project environment changed: %#v / %v", project, err)
	}
}

func TestControllerRejectsExecutionModeDrift(t *testing.T) {
	for _, scenario := range []struct {
		name, frozen, configured string
		allowed                  bool
		agentActive              bool
	}{
		{name: "legacy missing mode", allowed: true, agentActive: true},
		{name: "explicit Agent", frozen: "agent", configured: "agent", allowed: true, agentActive: true},
		{name: "Agent profile missing", frozen: "agent", configured: "agent"},
		{name: "local", frozen: "local", configured: "local", allowed: true},
		{name: "local changed to Agent", frozen: "local", configured: "agent"},
		{name: "legacy changed to local", configured: "local"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			project := &composetypes.Project{Environment: composetypes.Mapping{"GRAFT_EXECUTION_MODE": scenario.configured}}
			if scenario.agentActive {
				project.Services = composetypes.Services{"docker-runtime-agent": {Name: "docker-runtime-agent"}}
			}
			err := validateProjectExecutionMode(project, update.RunnerInput{ExecutionMode: scenario.frozen})
			if (err == nil) != scenario.allowed {
				t.Fatalf("mode drift result = %v, allowed %v", err, scenario.allowed)
			}
		})
	}
}

func TestLocalRecoveryPersistsModeAcrossTrustedMissingAndCorruptState(t *testing.T) {
	for _, scenario := range []string{"trusted", "missing", "corrupt"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			store, err := update.NewFileRunnerStateStore(root)
			if err != nil {
				t.Fatal(err)
			}
			recovery := update.RunnerRecoveryInput{OperationID: "update-local-recovery", RunnerID: "runner-before",
				SourceVersion: "1.0.0", TargetVersion: "1.1.0", Strategy: string(update.DeploymentStrategyBetaTracking), ExecutionMode: "local", LaunchStageID: 30, LaunchAttempt: 3}
			if scenario != "missing" {
				input := update.RunnerInput{OperationID: recovery.OperationID, SourceVersion: recovery.SourceVersion,
					TargetVersion: recovery.TargetVersion, ExecutionMode: "local", Preflight: update.ComposePreflight{DeploymentStrategy: update.DeploymentStrategyBetaTracking}}
				state := update.NewRunnerState(input, recovery.RunnerID, update.RunnerPhaseReady, 0, "runner_accepted", "", update.RunnerState{})
				state.LeaseHeartbeatAt = time.Now().UTC().Add(-10 * time.Minute)
				state.LeaseExpiresAt = time.Now().UTC().Add(-5 * time.Minute)
				if err := store.Write(state); err != nil {
					t.Fatal(err)
				}
				state, err = store.Read()
				if err != nil {
					t.Fatal(err)
				}
				if scenario == "trusted" {
					recovery.State = &state
					// 已验证的快照拥有恢复模式，兼容缺少新增模式字段的恢复材料。
					recovery.ExecutionMode = ""
				} else {
					if err := os.WriteFile(filepath.Join(root, "current.json"), []byte(`{"schema_version":2}`), 0o600); err != nil {
						t.Fatal(err)
					}
					recovery.Corrupt = true
				}
			}
			body, err := json.Marshal(recovery)
			if err != nil {
				t.Fatal(err)
			}
			if err := recoverTerminatedRunnerWithStore(base64.RawStdEncoding.EncodeToString(body), store, io.Discard); err != nil {
				t.Fatal(err)
			}
			state, err := store.Read()
			if err != nil || state.ExecutionMode != "local" || state.Receipt == nil || state.Receipt.ExecutionMode != "local" || !isTerminalPhase(state.Phase) {
				t.Fatalf("local terminal recovery evidence was lost: %#v / %v", state, err)
			}
			if state.Receipt.LaunchStageID != 30 || state.Receipt.LaunchAttempt != 3 {
				t.Fatalf("recovery receipt lost current retry binding: %#v", state.Receipt)
			}
		})
	}
}
