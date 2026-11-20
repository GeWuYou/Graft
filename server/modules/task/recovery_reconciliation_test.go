package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
	taskstore "graft/server/modules/task/store"
)

// 本文件回归策略驱动的有界自动恢复：租约中断幂等阶段在预算内自动重置为可领取，
// 预算耗尽或已无法领取时收敛为带稳定诊断的 needs_attention；manual_reconcile 与显式
// needs_attention 结论不进入自动恢复。恢复事件必须恰好一次。

func TestReconcileAutoResumesInterruptedIdempotentStageWithinBudget(t *testing.T) {
	fixture := submitAndExpireIdempotentExternalTask(t, 2)
	assertStageState(t, fixture.repository, fixture.taskID, moduleapi.StageStatusUnknown, "external_execution_lease_expired")

	fixture.runtime.reconcileInterruptedRecovery(context.Background())
	assertStageState(t, fixture.repository, fixture.taskID, moduleapi.StageStatusPending, "")
	if task := mustTask(t, fixture.repository, fixture.taskID); task.Status != moduleapi.TaskStatusRunning || task.FailureCode != nil || task.FailureClass != nil {
		t.Fatalf("task after automatic resume = %#v", task)
	}
	assertRecoveryEventCounts(t, fixture.repository, fixture.taskID, 1, 1)

	// 重领沿用同一冻结操作身份的外部执行路径，幂等重放语义保持不变。
	replayed := claimTestExternalExecution(t, fixture.runtime)
	if replayed.ID == fixture.lease.ID || replayed.Attempt != fixture.lease.Attempt+1 || replayed.OperationID != fixture.lease.OperationID {
		t.Fatalf("replayed lease = %#v original = %#v", replayed, fixture.lease)
	}
	if settlement, err := fixture.runtime.SettleExternalExecution(context.Background(), testExternalExecutionReceipt(replayed, moduleapi.ExternalReceiptOutcomeSuccess, "")); err != nil || settlement.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("settle resumed external stage = %#v err=%v", settlement, err)
	}

	// 收敛后的第二次巡检不产生新事实，事件保持恰好一次。
	fixture.runtime.reconcileInterruptedRecovery(context.Background())
	if task := mustTask(t, fixture.repository, fixture.taskID); task.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("task after idempotent replay = %#v", task)
	}
	assertRecoveryEventCounts(t, fixture.repository, fixture.taskID, 1, 1)
}

func TestReconcileConvergesBudgetExhaustedStageToNeedsAttention(t *testing.T) {
	fixture := submitAndExpireIdempotentExternalTask(t, 1)

	fixture.runtime.reconcileInterruptedRecovery(context.Background())
	assertBudgetExhaustedDiagnostics(t, fixture.repository, fixture.taskID)
	assertRecoveryEventCounts(t, fixture.repository, fixture.taskID, 1, 0)

	// 预算耗尽后的第二次巡检不得重复恢复事件，也不得改变已收敛的诊断事实。
	fixture.runtime.reconcileInterruptedRecovery(context.Background())
	assertRecoveryEventCounts(t, fixture.repository, fixture.taskID, 1, 0)
	assertStageState(t, fixture.repository, fixture.taskID, moduleapi.StageStatusUnknown, taskstore.RecoveryBudgetExhaustedFailureCode)
	if _, err := fixture.runtime.ClaimExternalExecution(context.Background(), moduleapi.ExternalExecutionClaimRequest{
		RuntimeTargetID: 42, ProviderID: "docker", Capability: "compose_execution", CapabilityVersion: "docker/v1",
	}); !errors.Is(err, moduleapi.ErrExternalExecutionNotFound) {
		t.Fatalf("claim over-budget stage error = %v, want %v", err, moduleapi.ErrExternalExecutionNotFound)
	}
}

// assertBudgetExhaustedDiagnostics 校验预算耗尽收敛后的阶段与任务稳定诊断投影。
func assertBudgetExhaustedDiagnostics(t *testing.T, repository *taskstore.SQLRepository, taskID uint64) {
	t.Helper()
	stages := mustStages(t, repository, taskID)
	stage := stages[0]
	if stage.Status != moduleapi.StageStatusUnknown || stage.Attempt != 1 {
		t.Fatalf("budget exhausted stage = %#v", stage)
	}
	if stage.FailureCode == nil || *stage.FailureCode != taskstore.RecoveryBudgetExhaustedFailureCode {
		t.Fatalf("budget exhausted failure code = %v", stage.FailureCode)
	}
	if stage.FailureClass == nil || *stage.FailureClass != moduleapi.ExecutionFailureClassInternal || stage.FailureMessage == nil || *stage.FailureMessage == "" || stage.FailureStage != nil {
		t.Fatalf("budget exhausted diagnostics = code:%v class:%v stage:%v", stage.FailureCode, stage.FailureClass, stage.FailureStage)
	}
	task := mustTask(t, repository, taskID)
	if task.Status != moduleapi.TaskStatusNeedsAttention || task.FailureCode == nil || *task.FailureCode != taskstore.RecoveryBudgetExhaustedFailureCode || task.FailureClass == nil || *task.FailureClass != moduleapi.ExecutionFailureClassInternal {
		t.Fatalf("budget exhausted task = %#v", task)
	}
}

func TestReconcileConvergesUnclaimablePendingIdempotentStage(t *testing.T) {
	runtime, repository, taskID := submitIdempotentExternalTask(t, 1)
	lease := claimTestExternalExecution(t, runtime)
	// 复现启动恢复缺陷：以超过租约窗口的时间执行启动恢复，会把以最后一次尝试被领取的
	// 幂等阶段重置为 pending；外部领取路径要求 attempt < max_attempts，该阶段将永远无法被领取。
	recoveryAt := lease.AbsoluteDeadlineAt.Add(time.Second)
	if count, err := repository.RecoverInterruptedStages(context.Background(), recoveryAt); err != nil || count != 1 {
		t.Fatalf("startup recovery of expired idempotent stage = count:%d err:%v", count, err)
	}
	assertStageState(t, repository, taskID, moduleapi.StageStatusPending, "")
	if stage := mustStages(t, repository, taskID)[0]; stage.Attempt != 1 || stage.MaxAttempts != 1 {
		t.Fatalf("unclaimable pending stage = %#v", stage)
	}
	if task := mustTask(t, repository, taskID); task.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("task with unclaimable pending stage = %#v", task)
	}

	runtime.reconcileInterruptedRecovery(context.Background())
	assertStageState(t, repository, taskID, moduleapi.StageStatusUnknown, taskstore.RecoveryBudgetExhaustedFailureCode)
	if task := mustTask(t, repository, taskID); task.Status != moduleapi.TaskStatusNeedsAttention || task.FailureCode == nil || *task.FailureCode != taskstore.RecoveryBudgetExhaustedFailureCode {
		t.Fatalf("task after unclaimable pending convergence = %#v", task)
	}
	// 启动恢复路径没有写入 recovery_required；预算耗尽收敛必须补记恰好一条。
	assertRecoveryEventCounts(t, repository, taskID, 1, 0)
	runtime.reconcileInterruptedRecovery(context.Background())
	assertRecoveryEventCounts(t, repository, taskID, 1, 0)
}

func TestReconcileSkipsManualReconcileAndExplicitNeedsAttention(t *testing.T) {
	runtime, repository := newRuntimeForTest(t)

	// manual_reconcile 阶段被中断后保持人工恢复事实，不自动重置。
	input := testSubmitInput(1, 2)
	input.Plan.Stages[0].RecoveryPolicy = moduleapi.StageRecoveryManualReconcile
	input.Plan.Stages[0].ExternalExecution = testExternalExecutionExpectation("manual-interrupted-operation")
	if _, err := runtime.Submit(context.Background(), input); err != nil {
		t.Fatalf("submit manual external task: %v", err)
	}
	manual := claimTestExternalExecution(t, runtime)
	expireClaimedExternalLease(t, repository, manual)
	runtime.reconcileInterruptedRecovery(context.Background())
	assertStageState(t, repository, manual.TaskID, moduleapi.StageStatusUnknown, "external_execution_lease_expired")
	if task := mustTask(t, repository, manual.TaskID); task.Status != moduleapi.TaskStatusNeedsAttention {
		t.Fatalf("manual task after reconcile = %#v", task)
	}
	assertRecoveryEventCounts(t, repository, manual.TaskID, 1, 0)

	// 显式 needs_attention 结论携带 Provider 判定语义，不属于运行时中断，不自动重放。
	idempotentInput := testSubmitInput(1, 2)
	idempotentInput.Plan.Stages[0].ExternalExecution = testExternalExecutionExpectation("explicit-attention-operation")
	if _, err := runtime.Submit(context.Background(), idempotentInput); err != nil {
		t.Fatalf("submit explicit attention task: %v", err)
	}
	explicit := claimTestExternalExecution(t, runtime)
	receipt := testExternalExecutionReceipt(explicit, moduleapi.ExternalReceiptOutcomeNeedsAttention, "provider_operation_failed")
	if settlement, err := runtime.SettleExternalExecution(context.Background(), receipt); err != nil || settlement.Status != moduleapi.TaskStatusNeedsAttention {
		t.Fatalf("explicit needs attention settlement = %#v err=%v", settlement, err)
	}
	runtime.reconcileInterruptedRecovery(context.Background())
	assertStageState(t, repository, explicit.TaskID, moduleapi.StageStatusUnknown, "provider_operation_failed")
	if task := mustTask(t, repository, explicit.TaskID); task.Status != moduleapi.TaskStatusNeedsAttention {
		t.Fatalf("explicit attention task after reconcile = %#v", task)
	}
	assertRecoveryEventCounts(t, repository, explicit.TaskID, 0, 0)
}

func TestResumeInterruptedRecoveryRestoresAllLegsAtomically(t *testing.T) {
	runtime, repository := newRuntimeForTest(t)
	legs := createIdempotentExternalLegs(t, repository, []string{"amd64", "arm64"}, []int{2, 2})
	first := claimTestExternalExecution(t, runtime)
	claimTestExternalExecution(t, runtime)
	// 两个 leg 的租约窗口几乎同时到期；一次过期收敛同时中断两条 leg。
	if count, err := repository.ExpireExternalExecutionLeases(context.Background(), first.AbsoluteDeadlineAt.Add(time.Second), 10); err != nil || count != 2 {
		t.Fatalf("expire interrupted legs = count:%d err:%v", count, err)
	}
	for _, stage := range mustStages(t, repository, legs.ID) {
		if stage.Status != moduleapi.StageStatusUnknown {
			t.Fatalf("interrupted leg = %#v", stage)
		}
	}

	resumed, err := repository.ResumeInterruptedRecovery(context.Background(), legs.ID, time.Now().UTC())
	if err != nil || resumed != 2 {
		t.Fatalf("resume interrupted legs = %d err=%v", resumed, err)
	}
	for _, stage := range mustStages(t, repository, legs.ID) {
		if stage.Status != moduleapi.StageStatusPending || stage.FailureCode != nil || stage.FailureClass != nil {
			t.Fatalf("resumed leg = %#v", stage)
		}
	}
	if task := mustTask(t, repository, legs.ID); task.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("task after atomic leg resume = %#v", task)
	}
	// 一次 Task 级恢复只追加一条 recovery_resolved，与恢复事务原子可见。
	assertRecoveryEventCounts(t, repository, legs.ID, 2, 1)
}

func TestReconcileCandidateSelectionReachesNewerWorkPastBlockedTasks(t *testing.T) {
	runtime, repository := newRuntimeForTest(t)
	// 100 个混合预算 Task：可恢复 leg + 已收敛的超预算 leg。修正前它们的可恢复 leg 占满
	// 最旧优先的有界窗口，且每次恢复都被 ErrRecoveryBlocked 拒绝，永远饿死更晚的可恢复任务。
	for index := 0; index < 100; index++ {
		blocked := createIdempotentExternalLegs(t, repository,
			[]string{fmt.Sprintf("blocked-%d-a", index), fmt.Sprintf("blocked-%d-b", index)}, []int{3, 1})
		first := claimTestExternalExecution(t, runtime)
		_ = claimTestExternalExecution(t, runtime)
		if count, err := repository.ExpireExternalExecutionLeases(context.Background(), first.AbsoluteDeadlineAt.Add(time.Second), 10); err != nil || count != 2 {
			t.Fatalf("expire blocked legs = count:%d err:%v", count, err)
		}
		blockedStages := mustStages(t, repository, blocked.ID)
		if err := repository.ConvergeRecoveryBudgetExhausted(context.Background(), taskstore.RecoveryBudgetExhaustionInput{TaskID: blocked.ID, StageID: blockedStages[1].ID, Now: time.Now().UTC()}); err != nil {
			t.Fatalf("converge blocked leg: %v", err)
		}
	}
	newerID := submitIdempotentExternalTaskOn(t, runtime, 2)
	newerLease := claimTestExternalExecution(t, runtime)
	expireClaimedExternalLease(t, repository, newerLease)
	candidates, err := repository.ListInterruptedRecoveryCandidates(context.Background(), 100)
	if err != nil {
		t.Fatalf("list candidates past blocked tasks: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Task.ID != newerID {
		t.Fatalf("blocked tasks starved newer work: %d candidates, want only task %d", len(candidates), newerID)
	}
	runtime.reconcileInterruptedRecovery(context.Background())
	if task := mustTask(t, repository, newerID); task.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("newer task after reconcile = %#v", task)
	}
}

func TestResumeInterruptedRecoveryRespectsCancellationRecordedAfterSelection(t *testing.T) {
	runtime, repository, taskID := submitIdempotentExternalTask(t, 2)
	lease := claimTestExternalExecution(t, runtime)
	expireClaimedExternalLease(t, repository, lease)
	// 候选选定（unknown + 租约过期）之后、恢复事务之前，取消请求落库且取消流程尚未完成。
	if _, err := repository.RequestCancellation(context.Background(), taskID, time.Now().UTC()); err != nil {
		t.Fatalf("record cancellation: %v", err)
	}
	if _, err := repository.ResumeInterruptedRecovery(context.Background(), taskID, time.Now().UTC()); !errors.Is(err, taskstore.ErrStateConflict) {
		t.Fatalf("resume with recorded cancellation error = %v, want %v", err, taskstore.ErrStateConflict)
	}
	if task := mustTask(t, repository, taskID); task.Status != moduleapi.TaskStatusNeedsAttention || task.CancelRequestedAt == nil {
		t.Fatalf("task after refused resume = %#v", task)
	}
	assertStageState(t, repository, taskID, moduleapi.StageStatusUnknown, "external_execution_lease_expired")
	assertRecoveryEventCounts(t, repository, taskID, 1, 0)
	// 巡检不再触碰已请求取消的 Task；既有取消流程照常完成。
	runtime.reconcileInterruptedRecovery(context.Background())
	assertRecoveryEventCounts(t, repository, taskID, 1, 0)
	if err := runtime.Cancel(context.Background(), taskID); err != nil {
		t.Fatalf("cancel refused task: %v", err)
	}
	if task := mustTask(t, repository, taskID); task.Status != moduleapi.TaskStatusCancelled {
		t.Fatalf("cancelled task after refusal = %#v", task)
	}
}

func TestReconcileKeepsBlockedLegVisibleWhenAnyLegExhaustsBudget(t *testing.T) {
	runtime, repository := newRuntimeForTest(t)
	// leg-amd64 仍有预算（max 3）；leg-arm64 一次领取后即耗尽预算（max 1）。一次过期收敛同时中断两条 leg。
	legs := createIdempotentExternalLegs(t, repository, []string{"amd64", "arm64"}, []int{3, 1})
	first := claimTestExternalExecution(t, runtime)
	_ = claimTestExternalExecution(t, runtime)
	if count, err := repository.ExpireExternalExecutionLeases(context.Background(), first.AbsoluteDeadlineAt.Add(time.Second), 10); err != nil || count != 2 {
		t.Fatalf("expire interrupted legs = count:%d err:%v", count, err)
	}

	// 收敛先于恢复：预算耗尽的 leg 保持可见，整任务恢复被整体拒绝，父任务不得回到 running。
	runtime.reconcileInterruptedRecovery(context.Background())
	assertRecoveryLegState(t, repository, legs.ID, recoveryLegExpectation{key: "build-amd64", status: moduleapi.StageStatusUnknown, failureCode: "external_execution_lease_expired", attempt: 1, maxAttempts: 3})
	assertRecoveryLegState(t, repository, legs.ID, recoveryLegExpectation{key: "build-arm64", status: moduleapi.StageStatusUnknown, failureCode: taskstore.RecoveryBudgetExhaustedFailureCode, attempt: 1, maxAttempts: 1})
	if task := mustTask(t, repository, legs.ID); task.Status != moduleapi.TaskStatusNeedsAttention {
		t.Fatalf("task with blocked leg after reconcile = %#v", task)
	}
	assertRecoveryEventCounts(t, repository, legs.ID, 2, 0)

	// 第二次巡检不得把阻塞任务拉回 running，也不得重复恢复事件。
	runtime.reconcileInterruptedRecovery(context.Background())
	assertRecoveryLegState(t, repository, legs.ID, recoveryLegExpectation{key: "build-amd64", status: moduleapi.StageStatusUnknown, failureCode: "external_execution_lease_expired", attempt: 1, maxAttempts: 3})
	assertRecoveryLegState(t, repository, legs.ID, recoveryLegExpectation{key: "build-arm64", status: moduleapi.StageStatusUnknown, failureCode: taskstore.RecoveryBudgetExhaustedFailureCode, attempt: 1, maxAttempts: 1})
	if task := mustTask(t, repository, legs.ID); task.Status != moduleapi.TaskStatusNeedsAttention {
		t.Fatalf("task with blocked leg after second reconcile = %#v", task)
	}
	assertRecoveryEventCounts(t, repository, legs.ID, 2, 0)
}

// recoveryLegExpectation 描述一次 leg 断言的期望状态、失败码与预算事实。
type recoveryLegExpectation struct {
	key         string
	status      moduleapi.StageStatus
	failureCode string
	attempt     int
	maxAttempts int
}

// assertRecoveryLegState 校验指定 leg 的持久化状态、失败码与预算事实。
func assertRecoveryLegState(t *testing.T, repository *taskstore.SQLRepository, taskID uint64, expect recoveryLegExpectation) {
	t.Helper()
	for _, stage := range mustStages(t, repository, taskID) {
		if stage.Key != expect.key {
			continue
		}
		if stage.Status != expect.status || stage.Attempt != expect.attempt || stage.MaxAttempts != expect.maxAttempts {
			t.Fatalf("leg %s = status:%s attempt:%d max:%d, want %s/%d/%d", expect.key, stage.Status, stage.Attempt, stage.MaxAttempts, expect.status, expect.attempt, expect.maxAttempts)
		}
		if stage.FailureCode == nil || *stage.FailureCode != expect.failureCode {
			t.Fatalf("leg %s failure code = %v, want %q", expect.key, stage.FailureCode, expect.failureCode)
		}
		return
	}
	t.Fatalf("leg %s missing from task %d", expect.key, taskID)
}

func TestResumeInterruptedRecoveryRejectsTaskWithoutResumableStages(t *testing.T) {
	runtime, repository := newRuntimeForTest(t)
	input := testSubmitInput(1, 2)
	input.Plan.Stages[0].ExternalExecution = testExternalExecutionExpectation("non-resumable-operation")
	if _, err := runtime.Submit(context.Background(), input); err != nil {
		t.Fatalf("submit non-resumable task: %v", err)
	}
	lease := claimTestExternalExecution(t, runtime)
	receipt := testExternalExecutionReceipt(lease, moduleapi.ExternalReceiptOutcomeNeedsAttention, "provider_operation_failed")
	if _, err := runtime.SettleExternalExecution(context.Background(), receipt); err != nil {
		t.Fatalf("settle explicit needs attention: %v", err)
	}

	if _, err := repository.ResumeInterruptedRecovery(context.Background(), lease.TaskID, time.Now().UTC()); err != taskstore.ErrStateConflict {
		t.Fatalf("resume without resumable stages error = %v, want %v", err, taskstore.ErrStateConflict)
	}
	// 无可恢复阶段时必须整体回滚：任务保持 needs_attention，不产生恢复事件。
	if task := mustTask(t, repository, lease.TaskID); task.Status != moduleapi.TaskStatusNeedsAttention {
		t.Fatalf("task after rejected resume = %#v", task)
	}
	assertRecoveryEventCounts(t, repository, lease.TaskID, 0, 0)
}

// interruptedIdempotentFixture 持有已中断幂等外部任务的完整夹具事实。
type interruptedIdempotentFixture struct {
	runtime    *Runtime
	repository *taskstore.SQLRepository
	taskID     uint64
	lease      moduleapi.ExternalExecutionLease
}

// submitIdempotentExternalTask 提交单阶段幂等外部执行任务并返回 Runtime、仓储与任务标识。
func submitIdempotentExternalTask(t *testing.T, attempts int) (*Runtime, *taskstore.SQLRepository, uint64) {
	t.Helper()
	runtime, repository := newRuntimeForTest(t)
	taskID := submitIdempotentExternalTaskOn(t, runtime, attempts)
	return runtime, repository, taskID
}

// submitIdempotentExternalTaskOn 在给定 Runtime 上提交单阶段幂等外部执行任务。
func submitIdempotentExternalTaskOn(t *testing.T, runtime *Runtime, attempts int) uint64 {
	t.Helper()
	input := testSubmitInput(1, attempts)
	input.Plan.Stages[0].ExternalExecution = testExternalExecutionExpectation("recovery-operation-" + t.Name())
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("submit idempotent external task: %v", err)
	}
	return receipt.TaskID
}

// submitAndExpireIdempotentExternalTask 提交幂等外部任务，领取一次并让租约过期，构造真实的运行时中断状态。
func submitAndExpireIdempotentExternalTask(t *testing.T, attempts int) interruptedIdempotentFixture {
	t.Helper()
	runtime, repository, taskID := submitIdempotentExternalTask(t, attempts)
	lease := claimTestExternalExecution(t, runtime)
	expireClaimedExternalLease(t, repository, lease)
	return interruptedIdempotentFixture{runtime: runtime, repository: repository, taskID: taskID, lease: lease}
}

// createIdempotentExternalLegs 通过仓储直接创建携带幂等外部 leg 的协调任务；冻结计划必须与持久化 leg 对齐。
func createIdempotentExternalLegs(t *testing.T, repository *taskstore.SQLRepository, legIDs []string, legMaxAttempts []int) taskmodel.Task {
	t.Helper()
	stages := make([]taskmodel.Stage, 0, len(legIDs))
	planStages := make([]moduleapi.StagePlan, 0, len(legIDs))
	for index, legID := range legIDs {
		stages = append(stages, taskmodel.Stage{Key: "build-" + legID, Sequence: index + 1, ExecutorType: "test.executor", CoordinationGroup: "build-platforms", LegID: legID, Status: moduleapi.StageStatusPending, MaxAttempts: legMaxAttempts[index], Input: json.RawMessage(`{}`), RecoveryPolicy: moduleapi.StageRecoveryRetryIfIdempotent, Result: json.RawMessage(`{}`), ExternalExecution: true})
		planStages = append(planStages, moduleapi.StagePlan{Key: "build-" + legID, ExecutorType: "test.executor", CoordinationGroup: "build-platforms", LegID: legID, Input: json.RawMessage(`{}`), RecoveryPolicy: moduleapi.StageRecoveryRetryIfIdempotent, ExternalExecution: testExternalExecutionExpectation("recovery-leg-" + legID)})
	}
	plan, err := json.Marshal(moduleapi.TaskPlan{Stages: planStages})
	if err != nil {
		t.Fatalf("marshal recovery leg plan: %v", err)
	}
	created, _, _, err := repository.Create(context.Background(), taskstore.CreateInput{Task: taskmodel.Task{Type: "test.recovery", Owner: moduleapi.TaskOwner{Type: "test", ID: "recovery-legs-" + t.Name() + "-" + legIDs[0]}, Status: moduleapi.TaskStatusReady, Input: json.RawMessage(`{}`), Metadata: json.RawMessage(`{}`), Plan: plan, State: json.RawMessage(`{}`)}, Stages: stages})
	if err != nil {
		t.Fatalf("create idempotent external legs: %v", err)
	}
	return created
}

// expireClaimedExternalLease 以超过租约与绝对截止时间的时间收敛一次已领取的外部租约。
func expireClaimedExternalLease(t *testing.T, repository *taskstore.SQLRepository, lease moduleapi.ExternalExecutionLease) {
	t.Helper()
	if count, err := repository.ExpireExternalExecutionLeases(context.Background(), lease.AbsoluteDeadlineAt.Add(time.Second), 10); err != nil || count != 1 {
		t.Fatalf("expire claimed external lease = count:%d err:%v", count, err)
	}
}

// assertStageState 校验首个阶段的持久化状态与失败码。
func assertStageState(t *testing.T, repository *taskstore.SQLRepository, taskID uint64, status moduleapi.StageStatus, failureCode string) {
	t.Helper()
	stages := mustStages(t, repository, taskID)
	if len(stages) != 1 {
		t.Fatalf("stage count = %d, want 1", len(stages))
	}
	if stages[0].Status != status {
		t.Fatalf("stage status = %q, want %q", stages[0].Status, status)
	}
	if failureCode == "" {
		if stages[0].FailureCode != nil || stages[0].FailureClass != nil || stages[0].FailureStage != nil {
			t.Fatalf("stage failure diagnostics not cleared = %#v", stages[0])
		}
		return
	}
	if stages[0].FailureCode == nil || *stages[0].FailureCode != failureCode {
		t.Fatalf("stage failure code = %v, want %q", stages[0].FailureCode, failureCode)
	}
}

// mustStages 读取任务的全量持久化阶段。
func mustStages(t *testing.T, repository *taskstore.SQLRepository, taskID uint64) []taskmodel.Stage {
	t.Helper()
	stages, err := repository.ListStages(context.Background(), taskID)
	if err != nil {
		t.Fatalf("list task stages: %v", err)
	}
	return stages
}

// assertRecoveryEventCounts 校验恢复事件的恰好一次语义。
func assertRecoveryEventCounts(t *testing.T, repository *taskstore.SQLRepository, taskID uint64, required int, resolved int) {
	t.Helper()
	events, err := repository.ListEvents(context.Background(), taskID, 0, 100)
	if err != nil {
		t.Fatalf("list task events: %v", err)
	}
	var requiredCount, resolvedCount int
	for _, event := range events {
		switch event.Type {
		case taskmodel.EventTypeRecoveryRequired:
			requiredCount++
		case taskmodel.EventTypeRecoveryResolved:
			resolvedCount++
		}
	}
	if requiredCount != required || resolvedCount != resolved {
		t.Fatalf("recovery events = required:%d resolved:%d, want required:%d resolved:%d", requiredCount, resolvedCount, required, resolved)
	}
}
