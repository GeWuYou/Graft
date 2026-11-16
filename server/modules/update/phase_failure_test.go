package update

import (
	"context"
	"errors"
	"testing"
)

func TestLocalControllerPhaseFailuresStopAtTheirOwningBoundary(t *testing.T) {
	for _, scenario := range []struct {
		stage, code string
		migration   bool
		outcome     ExecutionOutcome
	}{
		{"backup", runnerFailureBackup, false, ExecutionOutcomeFailed},
		{"compose pull", runnerFailurePull, false, ExecutionOutcomeRecovered},
		{"verify images", runnerFailureImageVerify, false, ExecutionOutcomeRecovered},
		{"stop server web", runnerFailureStopServices, false, ExecutionOutcomeRecovered},
		{"bootstrap migrate up", runnerFailureMigration, true, ExecutionOutcomeNeedsAttention},
		{"compose recreate server web", runnerFailureRecreate, true, ExecutionOutcomeNeedsAttention},
		{"docker health", runnerFailureDockerHealth, true, ExecutionOutcomeNeedsAttention},
		{"healthz", runnerFailureHealthz, true, ExecutionOutcomeNeedsAttention},
		{"local execution readiness", runnerFailureExecutionReady, true, ExecutionOutcomeNeedsAttention},
	} {
		t.Run(scenario.stage, func(t *testing.T) {
			input := fixtureRunnerInput(t.TempDir())
			input.ExecutionMode = "local"
			actions := &tracingRunnerActions{failAt: scenario.stage, recover: true}
			receipt, err := ExecuteComposeRunner(t.Context(), input, actions)
			if err == nil || receipt.Succeeded || receipt.FailureCode != scenario.code || receipt.MigrationStarted != scenario.migration {
				t.Fatalf("phase failure evidence = %#v / %v", receipt, err)
			}
			if ClassifyRunnerReceipt(receipt) != scenario.outcome || actions.trace[len(actions.trace)-1] != scenario.stage {
				t.Fatalf("phase failure crossed its recovery boundary: %#v / %#v", receipt, actions.trace)
			}
		})
	}
}

func TestLocalControllerCancellationRetainsTheMigrationBoundary(t *testing.T) {
	for _, stage := range []string{"backup", "bootstrap migrate up", "compose recreate server web", "healthz"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			input := fixtureRunnerInput(t.TempDir())
			input.ExecutionMode = "local"
			actions := &tracingRunnerActions{failAt: stage, failWith: context.Canceled, onFailure: cancel, recover: true}
			receipt, err := ExecuteComposeRunner(ctx, input, actions)
			migration := stage != "backup"
			if !errors.Is(err, context.Canceled) || receipt.Succeeded || receipt.MigrationStarted != migration || receipt.RecoveryCompleted {
				t.Fatalf("cancelled phase lost its boundary: %#v / %v", receipt, err)
			}
			if migration && ClassifyRunnerReceipt(receipt) != ExecutionOutcomeNeedsAttention {
				t.Fatalf("cancelled migration was presented as safely recoverable: %#v", receipt)
			}
		})
	}
}
