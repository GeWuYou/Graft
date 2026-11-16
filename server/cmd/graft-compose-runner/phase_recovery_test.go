package main

import (
	"testing"

	"graft/server/modules/update"
)

func TestInterruptedLocalControllerRecoveryPreservesItsDurablePhase(t *testing.T) {
	for _, scenario := range []struct {
		phase     update.RunnerPhase
		message   string
		migration bool
	}{
		{update.RunnerPhaseBackup, "creating_backup", false},
		{update.RunnerPhaseMigration, "running_migrations", true},
		{update.RunnerPhaseStartServices, "starting_services", true},
		{update.RunnerPhaseHealthCheck, "checking_health", true},
	} {
		t.Run(string(scenario.phase), func(t *testing.T) {
			store, err := update.NewFileRunnerStateStore(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			input := update.RunnerInput{OperationID: "update-phase-recovery", SourceVersion: "1.0.0", TargetVersion: "1.1.0",
				ExecutionMode: "local", Preflight: update.ComposePreflight{DeploymentStrategy: update.DeploymentStrategyBetaTracking}}
			state := update.NewRunnerState(input, "runner-original", scenario.phase, 70, scenario.message, "", update.RunnerState{})
			state = persistExpiredControllerFixture(t, store, state)
			recovery := update.RunnerRecoveryInput{OperationID: state.OperationID, RunnerID: state.RunnerID,
				SourceVersion: state.SourceVersion, TargetVersion: state.TargetVersion, Strategy: state.Strategy,
				ExecutionMode: "local", LaunchStageID: 7, LaunchAttempt: 2, State: &state}
			result := recoverControllerFixture(t, store, recovery)
			if result.Receipt.MigrationStarted != scenario.migration {
				t.Fatalf("durable recovery evidence = %#v", result)
			}
			if scenario.migration && update.ClassifyRunnerReceipt(*result.Receipt) != update.ExecutionOutcomeNeedsAttention {
				t.Fatalf("post-migration interruption lost manual recovery requirement: %#v", result.Receipt)
			}
			if scenario.migration && result.Receipt.FailureDetail != "interrupted_during_or_after_migration" {
				t.Fatalf("post-migration diagnostic described a pre-migration interruption: %#v", result.Receipt)
			}
		})
	}
}
