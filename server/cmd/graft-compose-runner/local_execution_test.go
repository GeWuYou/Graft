package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
			recovery := localRecoveryFixture(t, root, store, scenario)
			state := recoverControllerFixture(t, store, recovery)
			if state.ExecutionMode != "local" || state.Receipt.ExecutionMode != "local" || !isTerminalPhase(state.Phase) {
				t.Fatalf("local terminal recovery evidence was lost: %#v", state)
			}
			if state.Receipt.LaunchStageID != 30 || state.Receipt.LaunchAttempt != 3 {
				t.Fatalf("recovery receipt lost current retry binding: %#v", state.Receipt)
			}
			if state.Receipt.MigrationStarted != (scenario == "corrupt") {
				t.Fatalf("recovery uncertainty boundary was lost: %#v", state.Receipt)
			}
		})
	}
}
