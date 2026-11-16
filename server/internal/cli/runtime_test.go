package cli

import (
	"bytes"
	"context"
	"encoding/json"
	taskstore "graft/server/modules/task/store"
	"testing"
)

func TestRuntimeModePreflightJSONPreservesBlockedReport(t *testing.T) {
	original := runtimeModePreflight
	t.Cleanup(func() { runtimeModePreflight = original })
	runtimeModePreflight = func(_ context.Context, mode string) (taskstore.ExecutionModePreflight, error) {
		return taskstore.ExecutionModePreflight{CurrentMode: "agent", RequestedMode: mode, Ready: false, BlockingTasks: 2, DiagnosticCode: "execution_mode_has_unsettled_work"}, nil
	}
	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"runtime", "mode", "preflight", "--mode", "local", "--format", "json"})
	if err := command.Execute(); err == nil {
		t.Fatal("blocked cutover returned success")
	}
	var report taskstore.ExecutionModePreflight
	decoder := json.NewDecoder(&output)
	if err := decoder.Decode(&report); err != nil {
		t.Fatalf("JSON report missing: %s %v", output.String(), err)
	}
	if report.Ready || report.CurrentMode != "agent" || report.RequestedMode != "local" || report.BlockingTasks != 2 {
		t.Fatalf("report=%#v", report)
	}
}

func TestRuntimeModePreflightRejectsUnsupportedInputBeforeDatabase(t *testing.T) {
	for _, arguments := range [][]string{{"--mode", "invalid"}, {"--mode", "local", "--format", "yaml"}} {
		command := NewRootCommand()
		command.SetArgs(append([]string{"runtime", "mode", "preflight"}, arguments...))
		if err := command.Execute(); err == nil {
			t.Fatalf("accepted=%v", arguments)
		}
	}
}
