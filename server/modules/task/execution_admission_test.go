package task

import (
	"context"
	"errors"
	"strings"
	"testing"

	"graft/server/internal/moduleapi"
	taskstore "graft/server/modules/task/store"
)

// 执行所有权监视的准入回归：一次瞬时探测失败不得锁定准入（真实缺陷：单次 ping 失败
// 永久锁定 execution_ownership_lost，代理领取被 503 拒绝，已接受的任务被搁浅）。

func newAdmissionRuntimeForTest(t *testing.T) (*Runtime, *taskstore.SQLRepository, *taskstore.ExecutionOwnership) {
	t.Helper()
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	admissionConfig := configForAdmissionTest()
	runtime.configureLocalExecution(&admissionConfig)
	if err := repository.CheckExecutionMode(context.Background(), "agent"); err != nil {
		t.Fatalf("align execution mode row: %v", err)
	}
	// 与 localRuntimeForTest 相同的准入就绪语义：不启动后台 worker，用 runOne 驱动已接受的任务。
	runtime.executionAdmissionReady = true
	ownership := taskstore.NewExecutionOwnership()
	runtime.ownership = ownership
	return runtime, repository, ownership
}

func assertAdmissionRefused(t *testing.T, err error, diagnostic string) {
	t.Helper()
	if !errors.Is(err, moduleapi.ErrTaskExecutionUnavailable) || !strings.Contains(err.Error(), diagnostic) {
		t.Fatalf("admission refusal error = %v, want unavailable with %s", err, diagnostic)
	}
}

func assertAdmissionProjection(t *testing.T, admission moduleapi.TaskExecutionAdmission, ready bool, diagnostic string) {
	t.Helper()
	if admission.Ready != ready || admission.DiagnosticCode != diagnostic {
		t.Fatalf("admission projection = %#v, want ready=%v/%s", admission, ready, diagnostic)
	}
}

func TestExecutionAdmissionToleratesTransientOwnershipPingFailures(t *testing.T) {
	runtime, repository, ownership := newAdmissionRuntimeForTest(t)
	receipt, err := runtime.Submit(context.Background(), testSubmitInput(1, 1))
	if err != nil {
		t.Fatalf("submit accepted task: %v", err)
	}
	// N-1 次瞬时探测失败：准入保持就绪，已接受的任务不被搁浅（真实缺陷的级联起点）。
	for failures := 1; failures < taskstore.ExecutionOwnershipDegradedAfter; failures++ {
		if degraded := ownership.ProbeFailed(); degraded {
			t.Fatalf("transient ping failure %d degraded admission", failures)
		}
	}
	if err := runtime.requireExecutionAdmission(); err != nil {
		t.Fatalf("transient ownership ping failures must not refuse admission: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("accepted task must still execute across transient failures: %v", err)
	}
	if task, err := repository.Get(context.Background(), receipt.TaskID); err != nil || task.Status != "success" {
		t.Fatalf("accepted task after transient failures = %#v err=%v", task, err)
	}
	// N 次连续失败进入退化：新的执行请求（提交与代理领取）以独立诊断码拒绝。
	if !ownership.ProbeFailed() {
		t.Fatal("expected degraded admission after the failure threshold")
	}
	assertAdmissionRefused(t, runtime.requireExecutionAdmission(), taskstore.ExecutionOwnershipAdmissionDegraded)
	assertAdmissionRefused(t, mustClaimError(t, runtime), taskstore.ExecutionOwnershipAdmissionDegraded)
	assertAdmissionProjection(t, runtime.ExecutionAdmission(), false, taskstore.ExecutionOwnershipAdmissionDegraded)
	// 探测恢复：准入自动回到就绪，无需重启运行时。
	ownership.ProbeSucceeded()
	if err := runtime.requireExecutionAdmission(); err != nil {
		t.Fatalf("admission must self-heal after ping recovery: %v", err)
	}
	assertAdmissionProjection(t, runtime.ExecutionAdmission(), true, "execution_ready")
	// 真实的所有权冲突（他方运行时持有执行权）按既有语义永久失去。
	ownership.OwnershipConflict()
	assertAdmissionRefused(t, runtime.requireExecutionAdmission(), "execution_ownership_lost")
	assertAdmissionProjection(t, runtime.ExecutionAdmission(), false, "execution_ownership_lost")
}

// mustClaimError 驱动一次代理领取并返回其错误；退化与失去所有权都应在此处被拒绝。
func mustClaimError(t *testing.T, runtime *Runtime) error {
	t.Helper()
	_, err := runtime.ClaimExternalExecution(context.Background(), moduleapi.ExternalExecutionClaimRequest{
		RuntimeTargetID: 42, ProviderID: "docker", Capability: "compose_execution", CapabilityVersion: "docker/v1",
	})
	return err
}
