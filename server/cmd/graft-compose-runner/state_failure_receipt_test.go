package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"graft/server/internal/moduleapi"
	"graft/server/modules/update"
)

func stateFailureInput() update.RunnerInput {
	return update.RunnerInput{ProtocolVersion: 2, OperationID: "update-state-failure", RunnerID: "runner-state-failure",
		TaskID: 71, SourceVersion: "1.0.0", TargetVersion: "1.1.0", ExecutionMode: "local",
		LaunchStageID: 101, LaunchAttempt: 2, Preflight: update.ComposePreflight{DeploymentStrategy: update.DeploymentStrategyBetaTracking}}
}

func decodeStateFailureReceipt(t *testing.T, output string) update.RunnerReceipt {
	t.Helper()
	line := strings.Split(output, "\n")[0]
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(line, update.RunnerReceiptLogMarker))
	if err != nil {
		t.Fatal(err)
	}
	var receipt update.RunnerReceipt
	if err := json.Unmarshal(payload, &receipt); err != nil {
		t.Fatal(err)
	}
	return receipt
}

func TestInitialStateWriteFailureEmitsTheBoundControllerReceipt(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	if err := os.WriteFile(root, []byte("blocked state directory"), privateFilePermission); err != nil {
		t.Fatal(err)
	}
	store, err := update.NewFileRunnerStateStore(root)
	if err != nil {
		t.Fatal(err)
	}
	input := stateFailureInput()
	reporter := &stateReporter{store: store, input: input, runnerID: input.RunnerID}
	failure := reporter.Initialize()
	if failure == nil {
		t.Fatal("blocked state directory did not reject initialization")
	}
	var output bytes.Buffer
	if err := writeRunnerStateFailureLogs(&output, input, failure); err != nil {
		t.Fatal(err)
	}
	receipt := decodeStateFailureReceipt(t, output.String())
	if receipt.OperationID != input.OperationID || receipt.RunnerID != input.RunnerID || receipt.LaunchStageID != input.LaunchStageID || receipt.LaunchAttempt != input.LaunchAttempt {
		t.Fatalf("failure receipt lost the frozen launch identity: %#v", receipt)
	}
	if receipt.ExecutionMode != "local" || receipt.MigrationStarted || update.ClassifyRunnerReceipt(receipt) != update.ExecutionOutcomeNeedsAttention {
		t.Fatalf("failure receipt changed execution facts: %#v", receipt)
	}
}

func TestIntermediateStateWriteFailureHasAUsableReceiptWhenFinalizationAlsoFails(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	store, err := update.NewFileRunnerStateStore(root)
	if err != nil {
		t.Fatal(err)
	}
	input := stateFailureInput()
	reporter := &stateReporter{store: store, input: input, runnerID: input.RunnerID}
	if err := reporter.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, root+"-retained"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte("blocked state directory"), privateFilePermission); err != nil {
		t.Fatal(err)
	}
	receipt, executionErr := update.ExecuteComposeRunner(t.Context(), input, &actions{reporter: reporter})
	if executionErr == nil {
		t.Fatal("intermediate state write unexpectedly succeeded")
	}
	receipt = withRunnerStateFailure(input, receipt, executionErr)
	if err := reporter.Finalize(receipt); err == nil {
		t.Fatal("finalization unexpectedly wrote the blocked state")
	}
	var output bytes.Buffer
	if err := writeRunnerReceiptLog(&output, receipt); err != nil {
		t.Fatal(err)
	}
	decoded := decodeStateFailureReceipt(t, output.String())
	if decoded.FailureCode != update.RunnerFailureCodeStateWriteFailed || decoded.MigrationStarted || decoded.BackupCompletion != nil {
		t.Fatalf("unclassified state error escaped the receipt: %#v", decoded)
	}
}

func TestStateWriteFailurePreservesMigrationAndBackupFacts(t *testing.T) {
	for _, migrated := range []bool{false, true} {
		receipt := update.RunnerReceipt{MigrationStarted: migrated, BackupCompletion: &moduleapi.CompleteBackupRunnerHandoffInput{TaskID: 71}}
		result := withRunnerStateFailure(stateFailureInput(), receipt, os.ErrPermission)
		if result.MigrationStarted != migrated || result.BackupCompletion != receipt.BackupCompletion || result.FailureStage != update.RunnerFailureStagePermissionDenied {
			t.Fatalf("state error rewrote known facts: %#v", result)
		}
	}
}

func TestRecoveredStateWriterPersistsTheNormalizedFailureReceipt(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	store, err := update.NewFileRunnerStateStore(root)
	if err != nil {
		t.Fatal(err)
	}
	input := stateFailureInput()
	reporter := &stateReporter{store: store, input: input, runnerID: input.RunnerID}
	if err := reporter.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, root+"-retained"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte("temporary state blockage"), privateFilePermission); err != nil {
		t.Fatal(err)
	}
	receipt, executionErr := update.ExecuteComposeRunner(t.Context(), input, &actions{reporter: reporter})
	if executionErr == nil {
		t.Fatal("temporary state blockage did not fail the phase report")
	}
	receipt = withRunnerStateFailure(input, receipt, executionErr)
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root+"-retained", root); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Finalize(receipt); err != nil {
		t.Fatal(err)
	}
	assertNormalizedFailureJournal(t, store, input.OperationID)
}

func assertNormalizedFailureJournal(t *testing.T, store *update.FileRunnerStateStore, operationID string) {
	t.Helper()
	persisted, err := store.Read()
	if err != nil || persisted.Phase != update.RunnerPhaseFailed || persisted.Receipt == nil || persisted.Error != update.RunnerFailureCodeStateWriteFailed {
		t.Fatalf("terminal failure journal=%#v err=%v", persisted, err)
	}
	if update.ClassifyRunnerReceipt(*persisted.Receipt) != update.ExecutionOutcomeNeedsAttention || persisted.Receipt.MigrationStarted {
		t.Fatalf("persisted receipt changed the migration boundary: %#v", persisted.Receipt)
	}
	events, err := store.ReadEvents(operationID, 0, 10)
	if err != nil || len(events) != 2 || events[1].Phase != update.RunnerPhaseFailed {
		t.Fatalf("terminal failure event=%#v err=%v", events, err)
	}
}

func TestFinalStateWriteFailurePreservesCompletedAndAlreadyClassifiedResults(t *testing.T) {
	for _, receipt := range []update.RunnerReceipt{{Succeeded: true, MigrationStarted: true},
		{FailureCode: "backup_failed", FailureStage: "env_snapshot"}} {
		result := withRunnerStateFailure(stateFailureInput(), receipt, errors.New("terminal state unavailable"))
		if !reflect.DeepEqual(result, receipt) {
			t.Fatalf("known outcome was replaced: %#v", result)
		}
	}
}
