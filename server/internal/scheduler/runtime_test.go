package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/cronx"
)

// TestRegisterJobRejectsInvalidDeclarations 验证调度器会拒绝缺失执行入口或非法表达式的任务声明。
func TestRegisterJobRejectsInvalidDeclarations(t *testing.T) {
	runtime := New(zap.NewNop())

	if err := runtime.RegisterJob(cronx.Job{Name: "", Schedule: "* * * * * *", Run: func(context.Context) error { return nil }}); err == nil {
		t.Fatal("expected empty job name to fail")
	}
	if err := runtime.RegisterJob(cronx.Job{Name: "cleanup", Schedule: "", Run: func(context.Context) error { return nil }}); err == nil {
		t.Fatal("expected empty schedule to fail")
	}
	if err := runtime.RegisterJob(cronx.Job{Name: "cleanup", Schedule: "* * * * * *"}); err == nil {
		t.Fatal("expected missing run function to fail")
	}
}

// TestRegisterJobRejectsDuplicateName 验证重复任务名会在注册阶段显式失败。
func TestRegisterJobRejectsDuplicateName(t *testing.T) {
	runtime := New(zap.NewNop())
	job := cronx.Job{
		Name:     "cleanup",
		Schedule: "*/1 * * * * *",
		Run:      func(context.Context) error { return nil },
	}

	if err := runtime.RegisterJob(job); err != nil {
		t.Fatalf("register first job: %v", err)
	}
	if err := runtime.RegisterJob(job); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
}

// TestStartAndStopRunsRegisteredJob 验证最小调度器可以启动、执行一次任务并正常停止。
func TestStartAndStopRunsRegisteredJob(t *testing.T) {
	runtime := New(zap.NewNop())
	triggered := make(chan struct{}, 1)

	if err := runtime.RegisterJob(cronx.Job{
		Name:     "heartbeat",
		Schedule: "*/1 * * * * *",
		Run: func(_ context.Context) error {
			select {
			case triggered <- struct{}{}:
			default:
			}
			return nil
		},
	}); err != nil {
		t.Fatalf("register job: %v", err)
	}

	if err := runtime.Start(); err != nil {
		t.Fatalf("start runtime: %v", err)
	}
	defer func() {
		_ = runtime.Stop(context.Background())
	}()

	waitForSignal(t, triggered, 2500*time.Millisecond, "expected scheduled job to run")

	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("stop runtime: %v", err)
	}
}

// TestRemoveJobPreventsFutureExecution 验证移除任务后后续调度不会再次触发该任务。
func TestRemoveJobPreventsFutureExecution(t *testing.T) {
	runtime := New(zap.NewNop())
	triggered := make(chan struct{}, 2)

	if err := runtime.RegisterJob(cronx.Job{
		Name:     "cleanup",
		Schedule: "*/1 * * * * *",
		Run: func(_ context.Context) error {
			triggered <- struct{}{}
			return nil
		},
	}); err != nil {
		t.Fatalf("register job: %v", err)
	}
	if err := runtime.Start(); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	waitForSignal(t, triggered, 2500*time.Millisecond, "expected first scheduled execution")

	if err := runtime.RemoveJob("cleanup"); err != nil {
		t.Fatalf("remove job: %v", err)
	}

	assertNoSignal(t, triggered, 1200*time.Millisecond, "expected removed job not to run again")

	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("stop runtime: %v", err)
	}
}

// TestStopHonorsContextCancellation 验证 Stop 会把外部取消信号作为稳定错误返回。
func TestStopHonorsContextCancellation(t *testing.T) {
	runtime := New(zap.NewNop())
	if err := runtime.Start(); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	stopCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runtime.Stop(stopCtx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

// TestStopWithNilContextWaitsForInFlightJob 验证 nil ctx 会等待当前在途任务自然结束。
func TestStopWithNilContextWaitsForInFlightJob(t *testing.T) {
	runtime := New(zap.NewNop())
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	finished := make(chan struct{}, 1)

	if err := runtime.RegisterJob(cronx.Job{
		Name:     "blocking",
		Schedule: "*/1 * * * * *",
		Run: func(_ context.Context) error {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			select {
			case finished <- struct{}{}:
			default:
			}
			return nil
		},
	}); err != nil {
		t.Fatalf("register job: %v", err)
	}

	if err := runtime.Start(); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	waitForSignal(t, started, 2500*time.Millisecond, "expected scheduled job to start")

	stopDone := make(chan error, 1)
	var stopCtx context.Context
	go func() {
		stopDone <- runtime.Stop(stopCtx)
	}()

	assertNoStopResult(t, stopDone, 200*time.Millisecond)

	close(release)

	waitForSignal(t, finished, time.Second, "expected blocked job to finish after release")
	waitForStopResult(t, stopDone, time.Second)
}

func waitForSignal(t *testing.T, signal <-chan struct{}, timeout time.Duration, failureMessage string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(timeout):
		t.Fatal(failureMessage)
	}
}

func assertNoSignal(t *testing.T, signal <-chan struct{}, timeout time.Duration, failureMessage string) {
	t.Helper()

	select {
	case <-signal:
		t.Fatal(failureMessage)
	case <-time.After(timeout):
	}
}

func assertNoStopResult(t *testing.T, stopDone <-chan error, timeout time.Duration) {
	t.Helper()

	select {
	case err := <-stopDone:
		t.Fatalf("expected Stop(nil) to wait for in-flight job, got early result %v", err)
	case <-time.After(timeout):
	}
}

func waitForStopResult(t *testing.T, stopDone <-chan error, timeout time.Duration) {
	t.Helper()

	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("stop runtime: %v", err)
		}
	case <-time.After(timeout):
		t.Fatal("expected Stop(nil) to return after in-flight job finished")
	}
}
