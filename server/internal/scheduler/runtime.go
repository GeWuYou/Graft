package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"graft/server/internal/cronx"
)

// Runtime 暴露仓库内稳定的最小调度能力。
type Runtime interface {
	RegisterJob(job cronx.Job) error
	RemoveJob(name string) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	ListTasks(ctx context.Context) ([]TaskSnapshot, error)
	GetTask(ctx context.Context, key string) (TaskSnapshot, error)
	ListRuns(ctx context.Context, query RunListQuery) (RunListResult, error)
	RunOnce(ctx context.Context, key string) (TaskRun, error)
}

// RunStatus records the result state of one runtime job execution.
type RunStatus string

const (
	// RunStatusRunning indicates a scheduler task run has started but not finished.
	RunStatusRunning RunStatus = "running"
	// RunStatusSuccess indicates a scheduler task run completed successfully.
	RunStatusSuccess RunStatus = "success"
	// RunStatusFailed indicates a scheduler task run completed with an error.
	RunStatusFailed RunStatus = "failed"
)

// TriggerType records why a runtime job execution started.
type TriggerType string

const (
	// TriggerTypeSchedule indicates a scheduler task run was started by its configured cron schedule.
	TriggerTypeSchedule TriggerType = "schedule"
	// TriggerTypeManual indicates a scheduler task run was started by an explicit API/runtime request.
	TriggerTypeManual TriggerType = "manual"
)

// ErrTaskNotFound indicates the requested runtime job key is unknown.
var ErrTaskNotFound = errors.New("scheduler task not found")

// ErrTaskAlreadyRunning indicates the same task already has an active execution.
var ErrTaskAlreadyRunning = errors.New("scheduler task already running")

// TaskSnapshot is the internal service model consumed by later API routes.
type TaskSnapshot struct {
	Key                   string
	Name                  string
	Owner                 string
	Module                string
	Type                  cronx.TaskType
	DisplayMessageKey     string
	DescriptionMessageKey string
	Schedule              string
	DefaultEnabled        bool
	Running               bool
	LastRun               *TaskRun
}

// TaskRun is the persisted run-history model for scheduler runtime jobs.
type TaskRun struct {
	ID          uint64
	TaskKey     string
	TaskName    string
	Owner       string
	Module      string
	TaskType    cronx.TaskType
	TriggerType TriggerType
	Status      RunStatus
	Error       string
	StartedAt   time.Time
	FinishedAt  *time.Time
	DurationMS  *int64
	CreatedAt   time.Time
}

// RunListQuery scopes run-history lookup for one task.
type RunListQuery struct {
	TaskKey string
	Limit   int
	Offset  int
}

// RunListResult contains one page of run history plus a total count.
type RunListResult struct {
	Items []TaskRun
	Total int
}

// RunRepository persists scheduler_task_runs records.
type RunRepository interface {
	CreateRun(ctx context.Context, run TaskRun) (TaskRun, error)
	FinishRun(ctx context.Context, id uint64, status RunStatus, finishedAt time.Time, errorMessage string) (TaskRun, error)
	ListRuns(ctx context.Context, query RunListQuery) (RunListResult, error)
	LatestRunByTask(ctx context.Context, taskKey string) (TaskRun, bool, error)
}

// CronRuntime 是基于 robfig/cron/v3 的最小进程内调度器封装。
//
// 它把底层 cron 细节留在包内部，对外只保留显式 job 注册、启动、停止与
// 移除语义，避免业务模块直接依赖第三方调度器实现。
type CronRuntime struct {
	logger *zap.Logger

	mu      sync.RWMutex
	cron    *cron.Cron
	started bool
	entries map[string]cron.EntryID
	jobs    map[string]cronx.Job
	order   []string
	running map[string]struct{}

	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc
	runs            RunRepository
	now             func() time.Time
}

// New 创建一个新的最小调度器运行时。
func New(logger *zap.Logger, repositories ...RunRepository) *CronRuntime {
	if logger == nil {
		logger = zap.NewNop()
	}
	var runs RunRepository
	if len(repositories) > 0 {
		runs = repositories[0]
	}

	return &CronRuntime{
		logger:  logger,
		cron:    cron.New(cron.WithSeconds()),
		entries: make(map[string]cron.EntryID),
		jobs:    make(map[string]cronx.Job),
		order:   make([]string, 0),
		running: make(map[string]struct{}),
		runs:    runs,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// RegisterJob 注册一个显式调度任务。
func (r *CronRuntime) RegisterJob(job cronx.Job) error {
	if err := job.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := job.RuntimeKey()
	if _, exists := r.entries[key]; exists {
		return fmt.Errorf("job already registered: %s", key)
	}

	entryID, err := r.cron.AddFunc(job.Schedule, func() {
		runCtx := r.jobContext()
		if runCtx == nil {
			r.logger.Error("scheduler job skipped because lifecycle context is unavailable",
				zap.String("job", key),
				zap.String("module", job.Module),
			)
			return
		}

		if _, runErr := r.runJob(runCtx, job, TriggerTypeSchedule); runErr != nil {
			if errors.Is(runErr, ErrTaskAlreadyRunning) {
				r.logger.Warn("scheduler job skipped because task is already running",
					zap.String("job", key),
					zap.String("module", job.Module),
				)
				return
			}
			r.logger.Error("scheduler job failed",
				zap.String("job", key),
				zap.String("module", job.Module),
				zap.Error(runErr),
			)
		}
	})
	if err != nil {
		return fmt.Errorf("register job %s: %w", job.Name, err)
	}

	r.entries[key] = entryID
	r.jobs[key] = job
	r.order = append(r.order, key)
	return nil
}

// RemoveJob 按稳定名称移除一个已注册任务。
func (r *CronRuntime) RemoveJob(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entryID, ok := r.entries[name]
	if !ok {
		return errors.New("job not found")
	}

	r.cron.Remove(entryID)
	delete(r.entries, name)
	delete(r.jobs, name)
	r.order = removeKey(r.order, name)
	return nil
}

// ListTasks returns visible runtime job snapshots for later API routes.
func (r *CronRuntime) ListTasks(ctx context.Context) ([]TaskSnapshot, error) {
	r.mu.RLock()
	jobs := make([]cronx.Job, 0, len(r.order))
	for _, key := range r.order {
		jobs = append(jobs, r.jobs[key])
	}
	r.mu.RUnlock()

	items := make([]TaskSnapshot, 0, len(jobs))
	for _, job := range jobs {
		snapshot, err := r.snapshot(ctx, job)
		if err != nil {
			return nil, err
		}
		items = append(items, snapshot)
	}

	return items, nil
}

// GetTask returns one visible runtime job snapshot.
func (r *CronRuntime) GetTask(ctx context.Context, key string) (TaskSnapshot, error) {
	job, ok := r.findJob(key)
	if !ok {
		return TaskSnapshot{}, ErrTaskNotFound
	}

	return r.snapshot(ctx, job)
}

// ListRuns returns persisted run history for one task.
func (r *CronRuntime) ListRuns(ctx context.Context, query RunListQuery) (RunListResult, error) {
	if r.runs == nil {
		return RunListResult{}, errors.New("scheduler run repository is unavailable")
	}
	if _, ok := r.findJob(query.TaskKey); !ok {
		return RunListResult{}, ErrTaskNotFound
	}

	return r.runs.ListRuns(ctx, query)
}

// RunOnce executes one visible runtime job immediately.
func (r *CronRuntime) RunOnce(ctx context.Context, key string) (TaskRun, error) {
	job, ok := r.findJob(key)
	if !ok {
		return TaskRun{}, ErrTaskNotFound
	}
	if ctx == nil {
		ctx = context.Background()
	}

	return r.runJob(ctx, job, TriggerTypeManual)
}

// Start 绑定生命周期上下文并启动当前调度器。
func (r *CronRuntime) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("lifecycle context is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return nil
	}

	r.lifecycleCtx, r.lifecycleCancel = context.WithCancel(ctx)
	r.cron.Start()
	r.started = true
	return nil
}

// Stop 停止当前调度器并等待在途任务结束。
//
// 若传入的 ctx 为 nil，则无限期等待所有已启动任务完成；若 ctx 非 nil，
// 则在 ctx 取消时立即返回 ctx.Err()，但底层任务仍会继续执行到自然结束。
func (r *CronRuntime) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return nil
	}

	stopCtx := r.cron.Stop()
	r.started = false
	lifecycleCancel := r.lifecycleCancel
	r.lifecycleCtx = nil
	r.lifecycleCancel = nil
	r.mu.Unlock()

	if lifecycleCancel != nil {
		lifecycleCancel()
	}

	if ctx == nil {
		<-stopCtx.Done()
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-stopCtx.Done():
		return nil
	}
}

func (r *CronRuntime) jobContext() context.Context {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.lifecycleCtx
}

func (r *CronRuntime) findJob(key string) (cronx.Job, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.jobs[key]
	return job, ok
}

func (r *CronRuntime) snapshot(ctx context.Context, job cronx.Job) (TaskSnapshot, error) {
	key := job.RuntimeKey()
	snapshot := TaskSnapshot{
		Key:                   key,
		Name:                  job.Name,
		Owner:                 job.RuntimeOwner(),
		Module:                job.Module,
		Type:                  job.RuntimeType(),
		DisplayMessageKey:     job.DisplayMessageKey,
		DescriptionMessageKey: job.DescriptionMessageKey,
		Schedule:              job.Schedule,
		DefaultEnabled:        job.DefaultEnabled,
	}

	r.mu.RLock()
	_, snapshot.Running = r.running[key]
	r.mu.RUnlock()

	if r.runs == nil {
		return snapshot, nil
	}

	latest, ok, err := r.runs.LatestRunByTask(ctx, key)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if ok {
		snapshot.LastRun = &latest
	}

	return snapshot, nil
}

func (r *CronRuntime) runJob(ctx context.Context, job cronx.Job, trigger TriggerType) (TaskRun, error) {
	if err := job.Validate(); err != nil {
		return TaskRun{}, err
	}

	key := job.RuntimeKey()
	if err := r.markRunning(key); err != nil {
		return TaskRun{}, err
	}
	defer r.markFinished(key)

	if r.runs == nil {
		if err := job.Run(ctx); err != nil {
			return TaskRun{}, err
		}
		return TaskRun{
			TaskKey:     key,
			TaskName:    job.Name,
			Owner:       job.RuntimeOwner(),
			Module:      job.Module,
			TaskType:    job.RuntimeType(),
			TriggerType: trigger,
			Status:      RunStatusSuccess,
			StartedAt:   r.now(),
			CreatedAt:   r.now(),
		}, nil
	}

	startedAt := r.now()
	run, err := r.runs.CreateRun(ctx, TaskRun{
		TaskKey:     key,
		TaskName:    job.Name,
		Owner:       job.RuntimeOwner(),
		Module:      job.Module,
		TaskType:    job.RuntimeType(),
		TriggerType: trigger,
		Status:      RunStatusRunning,
		StartedAt:   startedAt,
		CreatedAt:   startedAt,
	})
	if err != nil {
		return TaskRun{}, err
	}

	runErr := job.Run(ctx)
	finishedAt := r.now()
	status := RunStatusSuccess
	errorMessage := ""
	if runErr != nil {
		status = RunStatusFailed
		errorMessage = runErr.Error()
	}

	finishedRun, finishErr := r.runs.FinishRun(ctx, run.ID, status, finishedAt, errorMessage)
	if finishErr != nil {
		return finishedRun, finishErr
	}
	if runErr != nil {
		return finishedRun, runErr
	}

	return finishedRun, nil
}

func (r *CronRuntime) markRunning(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.running[key]; exists {
		return ErrTaskAlreadyRunning
	}

	r.running[key] = struct{}{}
	return nil
}

func (r *CronRuntime) markFinished(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.running, key)
}

func removeKey(values []string, key string) []string {
	for index, value := range values {
		if value != key {
			continue
		}

		return append(values[:index], values[index+1:]...)
	}

	return values
}
