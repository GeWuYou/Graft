package update

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunnerExecutionModePreservesLegacyInputs(t *testing.T) {
	for _, mode := range []string{"", "agent", "local"} {
		t.Run(mode, func(t *testing.T) {
			input := fixtureRunnerInput(t.TempDir())
			input.ExecutionMode = mode
			if err := ValidateRunnerInput(input); err != nil {
				t.Fatalf("validate runner mode %q: %v", mode, err)
			}
			want := mode
			if want == "" {
				want = "agent"
			}
			if input.EffectiveExecutionMode() != want {
				t.Fatalf("effective mode = %q, want %q", input.EffectiveExecutionMode(), want)
			}
		})
	}
	input := fixtureRunnerInput(t.TempDir())
	input.ExecutionMode = "other"
	if err := ValidateRunnerInput(input); err == nil {
		t.Fatal("unknown execution mode accepted")
	}
}

func TestLocalControllerRequiresExecutionReadinessAfterServerHealth(t *testing.T) {
	for _, failure := range []string{"", "local execution readiness"} {
		t.Run(failure, func(t *testing.T) {
			input := fixtureRunnerInput(t.TempDir())
			input.ExecutionMode = "local"
			input.LaunchStageID, input.LaunchAttempt = 20, 2
			actions := &tracingRunnerActions{failAt: failure}
			receipt, err := ExecuteComposeRunner(context.Background(), input, actions)
			if receipt.ExecutionMode != "local" || actions.trace[len(actions.trace)-1] != "local execution readiness" {
				t.Fatalf("local mode or readiness sequence lost: %#v / %#v", receipt, actions.trace)
			}
			if receipt.LaunchStageID != 20 || receipt.LaunchAttempt != 2 {
				t.Fatalf("receipt lost its launch attempt: %#v", receipt)
			}
			if failure == "" && (err != nil || !receipt.Succeeded) {
				t.Fatalf("local controller failed: %#v / %v", receipt, err)
			}
			if failure != "" && (err == nil || receipt.FailureCode != runnerFailureExecutionReady || ClassifyRunnerReceipt(receipt) != ExecutionOutcomeNeedsAttention) {
				t.Fatalf("local readiness failure must require recovery: %#v / %v", receipt, err)
			}
		})
	}
}

func TestLegacyControllerKeepsExistingHealthSequence(t *testing.T) {
	input := fixtureRunnerInput(t.TempDir())
	actions := &tracingRunnerActions{}
	if _, err := ExecuteComposeRunner(context.Background(), input, actions); err != nil {
		t.Fatal(err)
	}
	if actions.trace[len(actions.trace)-1] != "healthz" {
		t.Fatalf("legacy update added a local readiness requirement: %#v", actions.trace)
	}
}

func TestRunnerStateAndReceiptKeepFrozenExecutionMode(t *testing.T) {
	input := fixtureRunnerInput(t.TempDir())
	input.ExecutionMode = "local"
	state := NewRunnerState(input, "runner-mode-1", RunnerPhaseReady, 0, "runner_accepted", "", RunnerState{})
	input.ExecutionMode = "agent"
	next := NewRunnerState(input, state.RunnerID, RunnerPhaseBackup, 15, "creating_backup", "", state)
	if next.ExecutionMode != "local" {
		t.Fatalf("controller mode changed after restart: %#v", next)
	}
	next.Receipt = &RunnerReceipt{ExecutionMode: "agent"}
	if err := validateRunnerState(next); err == nil {
		t.Fatal("receipt mode mismatch accepted")
	}
}

func TestRunnerReceiptRejectsUnknownModeWhileReadingLegacy(t *testing.T) {
	for _, mode := range []string{"", "agent", "local", "invalid"} {
		receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: "mode-receipt", ExecutionMode: mode}
		body, err := json.Marshal(receipt)
		if err != nil {
			t.Fatal(err)
		}
		_, valid := parseRunnerReceiptLog(RunnerReceiptLogMarker + base64.RawStdEncoding.EncodeToString(body))
		if valid != (mode != "invalid") {
			t.Fatalf("receipt mode %q validity = %v", mode, valid)
		}
		if mode == "" && strings.Contains(string(body), "execution_mode") {
			t.Fatal("legacy receipt integrity representation changed")
		}
	}
}
