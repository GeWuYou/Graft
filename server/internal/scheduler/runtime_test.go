package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/cronx"
)

type runRepositoryRecorder struct {
	created []TaskRun
	updated []TaskRun
	latest  map[string]TaskRun
	nextID  uint64
}

func newRunRepositoryRecorder() *runRepositoryRecorder {
	return &runRepositoryRecorder{latest: make(map[string]TaskRun), nextID: 1}
}

func (r *runRepositoryRecorder) CreateRun(_ context.Context, run TaskRun) (TaskRun, error) {
	run.ID = r.nextID
	r.nextID++
	r.created = append(r.created, run)
	r.latest[run.TaskKey] = run
	return run, nil
}

func (r *runRepositoryRecorder) FinishRun(_ context.Context, id uint64, status RunStatus, finishedAt time.Time, resultSummary string, errorMessage string) (TaskRun, error) {
	for _, run := range r.created {
		if run.ID != id {
			continue
		}

		run.Status = status
		run.Error = errorMessage
		run.Result = resultSummary
		run.FinishedAt = &finishedAt
		duration := int64(0)
		run.DurationMS = &duration
		r.updated = append(r.updated, run)
		r.latest[run.TaskKey] = run
		return run, nil
	}

	return TaskRun{}, errors.New("run not found")
}

func (r *runRepositoryRecorder) ListRuns(_ context.Context, query RunListQuery) (RunListResult, error) {
	items := make([]TaskRun, 0)
	for _, run := range r.updated {
		if run.TaskKey == query.TaskKey {
			items = append(items, run)
		}
	}
	return RunListResult{Items: items, Total: len(items)}, nil
}

func (r *runRepositoryRecorder) LatestRunByTask(_ context.Context, taskKey string) (TaskRun, bool, error) {
	run, ok := r.latest[taskKey]
	return run, ok, nil
}

func (r *runRepositoryRecorder) GetRun(_ context.Context, id uint64) (TaskRun, error) {
	for _, run := range r.updated {
		if run.ID == id {
			return run, nil
		}
	}
	return TaskRun{}, ErrTaskNotFound
}

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

// TestListTasksReturnsRuntimeJobSnapshots 验证运行时快照会保留任务声明中的展示与 owner 元数据。
func TestListTasksReturnsRuntimeJobSnapshots(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)

	if err := runtime.RegisterJob(cronx.Job{
		Name:                  "audit.audit-log-retention-cleanup",
		Key:                   "audit.audit-log-retention-cleanup",
		Owner:                 "audit",
		Type:                  cronx.TaskTypeSystem,
		DisplayMessageKey:     "scheduledTask.auditLogRetention.title",
		DescriptionMessageKey: "scheduledTask.auditLogRetention.description",
		Schedule:              "*/1 * * * * *",
		DefaultEnabled:        true,
		Module:                "audit",
		Run:                   func(context.Context) error { return nil },
	}); err != nil {
		t.Fatalf("register job: %v", err)
	}

	items, err := runtime.ListTasks(context.Background())
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one task, got %d", len(items))
	}
	item := items[0]
	if item.Key != "audit.audit-log-retention-cleanup" || item.Owner != "audit" || item.Type != cronx.TaskTypeSystem {
		t.Fatalf("unexpected task snapshot: %#v", item)
	}
	if item.DisplayMessageKey != "scheduledTask.auditLogRetention.title" || item.DescriptionMessageKey == "" {
		t.Fatalf("expected display metadata, got %#v", item)
	}
	if !item.Enabled {
		t.Fatal("expected runtime job to be default-enabled")
	}
}

// TestRunOncePersistsManualRunHistory 验证手动运行会写入运行历史并完成成功状态。
func TestRunOncePersistsManualRunHistory(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	triggered := false

	if err := runtime.RegisterJob(cronx.Job{
		Name:           "manual",
		Schedule:       "*/1 * * * * *",
		DefaultEnabled: true,
		Run: func(context.Context) error {
			triggered = true
			return nil
		},
	}); err != nil {
		t.Fatalf("register job: %v", err)
	}

	run, err := runtime.RunOnce(context.Background(), "manual")
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !triggered {
		t.Fatal("expected manual run to execute job")
	}
	if run.TriggerType != TriggerTypeManual || run.Status != RunStatusSuccess {
		t.Fatalf("expected successful manual run, got %#v", run)
	}
	if len(repo.created) != 1 || len(repo.updated) != 1 {
		t.Fatalf("expected one persisted run lifecycle, got created=%d updated=%d", len(repo.created), len(repo.updated))
	}
}

// TestRunOnceRejectsConcurrentSameTask 验证同一任务运行中再次手动触发会返回冲突式错误。
func TestRunOnceRejectsConcurrentSameTask(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	started := make(chan struct{}, 1)
	release := make(chan struct{})

	if err := runtime.RegisterJob(cronx.Job{
		Name:     "blocking",
		Schedule: "*/1 * * * * *",
		Run: func(context.Context) error {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			return nil
		},
	}); err != nil {
		t.Fatalf("register job: %v", err)
	}

	firstDone := make(chan error, 1)
	go func() {
		_, err := runtime.RunOnce(context.Background(), "blocking")
		firstDone <- err
	}()

	waitForSignal(t, started, time.Second, "expected first manual run to start")

	if _, err := runtime.RunOnce(context.Background(), "blocking"); !errors.Is(err, ErrTaskAlreadyRunning) {
		t.Fatalf("expected already-running conflict, got %v", err)
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first manual run failed: %v", err)
	}
}

// TestStartAndStopRunsRegisteredJob 验证最小调度器可以启动、执行一次任务并正常停止。
func TestStartAndStopRunsRegisteredJob(t *testing.T) {
	runtime := New(zap.NewNop())
	triggered := make(chan struct{}, 1)
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if err := runtime.Start(runCtx); err != nil {
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
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	if err := runtime.Start(runCtx); err != nil {
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
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	if err := runtime.Start(runCtx); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	stopCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runtime.Stop(stopCtx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

// TestStopCancelsJobLifecycleContext 验证显式 Stop 会取消运行中任务绑定的 lifecycle ctx。
func TestStopCancelsJobLifecycleContext(t *testing.T) {
	runtime := New(zap.NewNop())
	runCtx := context.Background()
	started := make(chan context.Context, 1)
	finished := make(chan struct{}, 1)

	if err := runtime.RegisterJob(cronx.Job{
		Name:     "watch",
		Schedule: "*/1 * * * * *",
		Run: func(ctx context.Context) error {
			select {
			case started <- ctx:
			default:
			}
			<-ctx.Done()
			select {
			case finished <- struct{}{}:
			default:
			}
			return nil
		},
	}); err != nil {
		t.Fatalf("register job: %v", err)
	}

	if err := runtime.Start(runCtx); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	jobCtx := waitForJobContext(t, started, 2500*time.Millisecond, "expected scheduled job to start")
	if jobCtx == nil {
		t.Fatal("expected job to receive lifecycle context")
	}
	if jobCtx.Err() != nil {
		t.Fatalf("expected job lifecycle context to be active, got %v", jobCtx.Err())
	}

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- runtime.Stop(context.Background())
	}()

	waitForContextDone(jobCtx, t, time.Second, "expected stop to cancel job lifecycle context")

	waitForSignal(t, finished, time.Second, "expected job to observe lifecycle cancellation")
	waitForStopResult(t, stopDone, time.Second)
}

// TestStopWithNilContextWaitsForInFlightJob 验证 nil ctx 会等待当前在途任务自然结束。
func TestStopWithNilContextWaitsForInFlightJob(t *testing.T) {
	runtime := New(zap.NewNop())
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	finished := make(chan struct{}, 1)
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if err := runtime.Start(runCtx); err != nil {
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

func waitForJobContext(t *testing.T, signal <-chan context.Context, timeout time.Duration, failureMessage string) context.Context {
	t.Helper()

	select {
	case ctx := <-signal:
		return ctx
	case <-time.After(timeout):
		t.Fatal(failureMessage)
		return nil
	}
}

func waitForContextDone(ctx context.Context, t *testing.T, timeout time.Duration, failureMessage string) {
	t.Helper()

	select {
	case <-ctx.Done():
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
