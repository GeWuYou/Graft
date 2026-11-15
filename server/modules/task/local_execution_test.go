package task

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/dockerexecution"
	"graft/server/internal/moduleapi"
	taskstore "graft/server/modules/task/store"
)

type localConnectionReader struct{}

func (localConnectionReader) GetProviderConnection(_ context.Context, id int64) (moduleapi.RuntimeTargetProviderConnection, error) {
	return moduleapi.RuntimeTargetProviderConnection{TargetID: id, Provider: "docker", Endpoint: "unix:///var/run/docker.sock", ConnectionKind: "unix_socket"}, nil
}

func localRuntimeForTest(t *testing.T) (*Runtime, *taskstore.SQLRepository) {
	t.Helper()
	runtime, repository := newRuntimeForTest(t)
	if err := repository.CheckExecutionMode(context.Background(), "local"); err != nil {
		t.Fatal(err)
	}
	runtime.configureLocalExecution(&config.Config{Execution: config.ExecutionConfig{Mode: "local"}, Container: config.ContainerConfig{DockerEndpoint: "unix:///var/run/docker.sock"}})
	runtime.localExecution.connections = localConnectionReader{}
	runtime.executionAdmissionReady = true
	return runtime, repository
}

func localSubmitInput(owner, key string) moduleapi.SubmitTaskInput {
	return moduleapi.SubmitTaskInput{Type: "test.local", Owner: moduleapi.TaskOwner{Type: "test", ID: owner}, Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{Key: "run", ExecutorType: "local.test", Input: json.RawMessage(`{"application_id":"app"}`), RecoveryPolicy: moduleapi.StageRecoveryManualReconcile, RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, ExternalExecution: &moduleapi.ExternalExecutionExpectation{RuntimeTargetID: 1, ProviderID: "docker", Capability: "compose_execution", CapabilityVersion: "docker/v1", Protocol: "application-compose/v1", OperationID: "application.compose.up.v1", LeaseTTL: time.Second, AbsoluteDeadline: time.Minute, ResourceKeys: []string{key}}}}}}
}

func TestLocalExecutionFreezesModeAndUsesTransportNeutralSDK(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	called := false
	runtime.localExecution.execute = func(_ context.Context, _ dockerexecution.Environment, request dockerexecution.Request) dockerexecution.Result {
		called = true
		if request.OperationID != "application.compose.up.v1" {
			t.Fatalf("request=%#v", request)
		}
		return dockerexecution.Result{Outcome: "success"}
	}
	receipt, err := runtime.Submit(context.Background(), localSubmitInput("one", "compose:one"))
	if err != nil {
		t.Fatal(err)
	}
	task := mustTask(t, repository, receipt.TaskID)
	var plan moduleapi.TaskPlan
	if err := json.Unmarshal(task.Plan, &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Stages[0].ExternalExecution != nil || plan.Stages[0].LocalExecution == nil || plan.Stages[0].LocalExecution.Version != localExecutionVersion {
		t.Fatalf("plan=%#v", plan)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called || mustTask(t, repository, receipt.TaskID).Status != moduleapi.TaskStatusSuccess {
		t.Fatal("local SDK operation was not completed")
	}
}

func TestLocalExecutionPersistsCrossOwnerConflictsAndAllowsUnrelatedResources(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	receipt, err := runtime.Submit(context.Background(), localSubmitInput("start", "container:canonical"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Submit(context.Background(), localSubmitInput("stop", "container:canonical")); !errors.Is(err, moduleapi.ErrTaskOwnerBusy) {
		t.Fatalf("same-resource conflict=%v", err)
	}
	if _, err := runtime.Submit(context.Background(), localSubmitInput("other", "container:other")); err != nil {
		t.Fatalf("unrelated resource=%v", err)
	}
	claimTaskStage(t, repository)
	if _, err := repository.RecoverInterruptedStages(context.Background(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if mustTask(t, repository, receipt.TaskID).Status != moduleapi.TaskStatusNeedsAttention {
		t.Fatal("uncertain operation was not preserved")
	}
	if _, err := runtime.Submit(context.Background(), localSubmitInput("retry-other-owner", "container:canonical")); !errors.Is(err, moduleapi.ErrTaskOwnerBusy) {
		t.Fatalf("uncertain resource released=%v", err)
	}
}

func TestLocalExecutionUnknownOutcomeSurvivesCancellation(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	runtime.localExecution.execute = func(_ context.Context, _ dockerexecution.Environment, _ dockerexecution.Request) dockerexecution.Result {
		if err := runtime.Cancel(context.Background(), 1); err != nil {
			t.Fatal(err)
		}
		return dockerexecution.Result{Outcome: "needs_attention", FailureCode: "agent_execution_interrupted"}
	}
	receipt, err := runtime.Submit(context.Background(), localSubmitInput("cancel", "compose:cancel"))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	task := mustTask(t, repository, receipt.TaskID)
	if task.Status != moduleapi.TaskStatusNeedsAttention || task.FailureCode == nil || *task.FailureCode != "runner_interrupted" {
		t.Fatalf("task=%#v", task)
	}
}

func TestExecutionModeCutoverKeepsHistoryAndRejectsUnsettledWork(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	runtime.localExecution.execute = func(context.Context, dockerexecution.Environment, dockerexecution.Request) dockerexecution.Result {
		return dockerexecution.Result{Outcome: "success"}
	}
	receipt, err := runtime.Submit(context.Background(), localSubmitInput("mode", "compose:mode"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := repository.PreflightExecutionMode(context.Background(), "agent")
	if err != nil || report.Ready || report.BlockingTasks != 1 {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if err := repository.CheckExecutionMode(context.Background(), "agent"); err == nil {
		t.Fatal("changed mode with active local task")
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := repository.CheckExecutionMode(context.Background(), "agent"); err != nil {
		t.Fatal(err)
	}
	if mustTask(t, repository, receipt.TaskID).Status != moduleapi.TaskStatusSuccess {
		t.Fatal("terminal history changed")
	}
	if _, err := runtime.Submit(context.Background(), localSubmitInput("stale", "compose:stale")); err == nil {
		t.Fatal("stale local server submitted after mode switch")
	}
}

func TestLocalStageRejectsChangedAttemptBeforeMaterialOrEffects(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	receipt, err := runtime.Submit(context.Background(), localSubmitInput("fence", "compose:fence"))
	if err != nil {
		t.Fatal(err)
	}
	claim, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC())
	if err != nil || !found {
		t.Fatalf("claim=%v %v", found, err)
	}
	if _, err := repository.RecoverInterruptedStages(context.Background(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	stage, ok := localStage(claim.Task, claim.Stage)
	if !ok {
		t.Fatal("missing frozen local binding")
	}
	executor := localStageExecutor{runtime: runtime, task: claim.Task, claimed: claim.Stage, stage: stage}
	run := &stageRun{runtime: runtime, task: claim.Task, stage: claim.Stage}
	called := false
	runtime.localExecution.execute = func(context.Context, dockerexecution.Environment, dockerexecution.Request) dockerexecution.Result {
		called = true
		return dockerexecution.Result{Outcome: "success"}
	}
	if err := executor.Execute(context.Background(), run); !errors.Is(err, taskstore.ErrStateConflict) {
		t.Fatalf("stale attempt=%v", err)
	}
	if called {
		t.Fatal("stale attempt performed SDK effects")
	}
	if mustTask(t, repository, receipt.TaskID).Status != moduleapi.TaskStatusNeedsAttention {
		t.Fatal("recovery fact changed")
	}
}

func TestLocalOwnershipLossRejectsNewSubmissionsAndReportsUnavailable(t *testing.T) {
	runtime, _ := localRuntimeForTest(t)
	lost := make(chan struct{})
	runtime.ownershipLost = lost
	close(lost)
	admission := runtime.ExecutionAdmission()
	if admission.Ready || admission.DiagnosticCode != "execution_ownership_lost" {
		t.Fatalf("admission=%#v", admission)
	}
	if _, err := runtime.Submit(context.Background(), localSubmitInput("lost", "compose:lost")); err == nil {
		t.Fatal("lost owner accepted new work")
	}
}

func TestLocalIdempotentReplayPrecedesChangedResourceAdmission(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	input := localSubmitInput("replay", "compose:replay")
	input.IdempotencyKey = "same-request"
	first, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	// Existing receipt is readable even if the target resolver later fails.
	runtime.localExecution.connections = nil
	second, err := runtime.Submit(context.Background(), input)
	if err != nil || second != first {
		t.Fatalf("replay=%#v %v", second, err)
	}
	stages, err := repository.ListStages(context.Background(), first.TaskID)
	if err != nil || len(stages) != 1 {
		t.Fatalf("duplicate task stages=%#v %v", stages, err)
	}
}

func TestModeSwitchAllowsModeUnboundNativeTasks(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Submit(context.Background(), testSubmitInput(1, 1)); err != nil {
		t.Fatal(err)
	}
	report, err := repository.PreflightExecutionMode(context.Background(), "agent")
	if err != nil || !report.Ready {
		t.Fatalf("native task blocked cutover: %#v %v", report, err)
	}
	if err := repository.CheckExecutionMode(context.Background(), "agent"); err != nil {
		t.Fatal(err)
	}
}

func TestStaleRuntimeCannotReserveAfterPersistedModeCutover(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	if err := repository.CheckExecutionMode(context.Background(), "agent"); err != nil {
		t.Fatal(err)
	}
	_, err := runtime.BeginSubmission(context.Background(), moduleapi.BeginTaskSubmissionInput{Task: localSubmitInput("stale-reservation", "compose:stale"), Policy: moduleapi.TaskSubmissionPolicy{LeaseTTL: time.Minute, AbsoluteDeadline: time.Hour, RenewBefore: time.Second, AllowRenew: true, PrerequisiteKind: "test"}})
	if !errors.Is(err, moduleapi.ErrTaskExecutionUnavailable) {
		t.Fatalf("stale reservation=%v", err)
	}
}

func TestControllerReceiptReconcilesCrashAfterLaunchBeforeSuccess(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	input := localSubmitInput("controller-crash", "platform-update")
	input.Plan.Stages[0].Key = "controller_launch"
	final := externalReceiptSubmitInput().Plan.Stages[0]
	final.Key = "controller_result"
	final.ExternalReceipt.LaunchStageKey = "controller_launch"
	input.Plan.Stages = append(input.Plan.Stages, final)
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	claim, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC())
	if err != nil || !found {
		t.Fatalf("claim=%t %v", found, err)
	}
	// The independent Controller has started, but the launching worker has not committed success.
	if _, err := repository.RecoverInterruptedStages(context.Background(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	proof := externalReceipt(receipt.TaskID, moduleapi.ExternalReceiptOutcomeSuccess, "")
	proof.LaunchStageID = claim.Stage.ID
	proof.LaunchAttempt = claim.Stage.Attempt
	stale := proof
	stale.LaunchAttempt++
	if _, err := runtime.SettleExternalReceipt(context.Background(), stale); err == nil {
		t.Fatal("stale Controller attempt settled")
	}
	settlement, err := runtime.SettleExternalReceipt(context.Background(), proof)
	if err != nil || settlement.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("settlement=%#v %v", settlement, err)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil || len(stages) != 2 || stages[0].Status != moduleapi.StageStatusSuccess || stages[1].Status != moduleapi.StageStatusSuccess {
		t.Fatalf("stages=%#v %v", stages, err)
	}
	replay, err := runtime.SettleExternalReceipt(context.Background(), proof)
	if err != nil || !replay.Idempotent {
		t.Fatalf("receipt replay=%#v %v", replay, err)
	}
}

func TestControllerReceiptCannotSettleUnstartedLaunch(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	input := localSubmitInput("unstarted-controller", "platform-update")
	input.Plan.Stages[0].Key = "controller_launch"
	final := externalReceiptSubmitInput().Plan.Stages[0]
	final.Key = "controller_result"
	final.ExternalReceipt.LaunchStageKey = "controller_launch"
	input.Plan.Stages = append(input.Plan.Stages, final)
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	proof := externalReceipt(receipt.TaskID, moduleapi.ExternalReceiptOutcomeSuccess, "")
	proof.LaunchStageID = stages[0].ID
	proof.LaunchAttempt = 1
	if _, err := runtime.SettleExternalReceipt(context.Background(), proof); err == nil {
		t.Fatal("unstarted launch accepted Controller proof")
	}
	if mustTask(t, repository, receipt.TaskID).Status != moduleapi.TaskStatusReady {
		t.Fatal("invalid proof changed task")
	}
}
