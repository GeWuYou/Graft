package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"graft/server/internal/cronx"
)

// Runtime 暴露仓库内稳定的最小调度能力。
type Runtime interface {
	RegisterJob(job cronx.Job) error
	SeedBuiltinJobs(ctx context.Context, jobs []cronx.Job) error
	RemoveJob(name string) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	ListTasks(ctx context.Context) ([]TaskSnapshot, error)
	GetTask(ctx context.Context, key string) (TaskSnapshot, error)
	CreateTask(ctx context.Context, command TaskMutation) (TaskSnapshot, error)
	UpdateTask(ctx context.Context, key string, command TaskMutation) (TaskSnapshot, error)
	DeleteTask(ctx context.Context, key string) error
	SetTaskEnabled(ctx context.Context, key string, enabled bool) (TaskSnapshot, error)
	ListRuns(ctx context.Context, query RunListQuery) (RunListResult, error)
	GetRun(ctx context.Context, id uint64) (TaskRun, error)
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
	// TriggerTypeCron indicates a scheduler task run was started by its configured cron schedule.
	TriggerTypeCron TriggerType = "cron"
	// TriggerTypeManual indicates a scheduler task run was started by an explicit API/runtime request.
	TriggerTypeManual TriggerType = "manual"
	// TriggerTypeStartup indicates a scheduler task run was started during runtime startup.
	TriggerTypeStartup TriggerType = "startup"
)

// ErrTaskNotFound indicates the requested runtime job key is unknown.
var ErrTaskNotFound = errors.New("scheduler task not found")

// ErrTaskAlreadyRunning indicates the same task already has an active execution.
var ErrTaskAlreadyRunning = errors.New("scheduler task already running")

// ErrTaskImmutable indicates a protected builtin field was modified.
var ErrTaskImmutable = errors.New("scheduler task field is immutable")

// ErrTaskValidation indicates a persisted task definition is invalid.
var ErrTaskValidation = errors.New("scheduler task validation failed")

// TaskSnapshot is the internal service model consumed by later API routes.
type TaskSnapshot struct {
	ID                    uint64
	Key                   string
	Name                  string
	Owner                 string
	Module                string
	Type                  cronx.TaskType
	Title                 string
	Description           string
	DisplayMessageKey     string
	DescriptionMessageKey string
	Schedule              string
	Enabled               bool
	Builtin               bool
	ConfigJSON            string
	Running               bool
	LastRun               *TaskRun
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
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
	Result      string
	StartedAt   time.Time
	FinishedAt  *time.Time
	DurationMS  *int64
	CreatedAt   time.Time
}

// TaskDefinition is the DB-backed authority for one scheduled task.
type TaskDefinition struct {
	ID             uint64
	TaskKey        string
	TaskType       cronx.TaskType
	Title          string
	Description    string
	CronExpression string
	Enabled        bool
	Builtin        bool
	ConfigJSON     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

// TaskMutation carries create/update input before HTTP routes are bound.
type TaskMutation struct {
	TaskKey        string
	TaskType       cronx.TaskType
	Title          string
	Description    string
	CronExpression string
	Enabled        bool
	EnabledSet     bool
	ConfigJSON     string
}

// HTTPTaskConfig is the persisted config_json shape for user HTTP tasks.
type HTTPTaskConfig struct {
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           string            `json:"body,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
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
	FinishRun(ctx context.Context, id uint64, status RunStatus, finishedAt time.Time, resultSummary string, errorMessage string) (TaskRun, error)
	ListRuns(ctx context.Context, query RunListQuery) (RunListResult, error)
	LatestRunByTask(ctx context.Context, taskKey string) (TaskRun, bool, error)
	GetRun(ctx context.Context, id uint64) (TaskRun, error)
}

// TaskRepository persists scheduled_tasks task definitions.
type TaskRepository interface {
	SeedBuiltinTasks(ctx context.Context, tasks []TaskDefinition) error
	CreateTask(ctx context.Context, task TaskDefinition) (TaskDefinition, error)
	UpdateTask(ctx context.Context, key string, patch TaskMutation) (TaskDefinition, error)
	DeleteTask(ctx context.Context, key string) error
	SetTaskEnabled(ctx context.Context, key string, enabled bool) (TaskDefinition, error)
	ListTasks(ctx context.Context) ([]TaskDefinition, error)
	GetTask(ctx context.Context, key string) (TaskDefinition, error)
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
	tasks           TaskRepository
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

// SetTaskRepository binds the DB-backed task definition authority to the runtime.
func (r *CronRuntime) SetTaskRepository(repository TaskRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tasks = repository
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

	entryID, err := r.addCronFuncLocked(key, job.Schedule, func(runCtx context.Context) (TaskRun, error) {
		return r.runJob(runCtx, job, TriggerTypeCron)
	})
	if err != nil {
		return fmt.Errorf("register job %s: %w", job.Name, err)
	}

	r.entries[key] = entryID
	r.jobs[key] = job
	r.order = append(r.order, key)
	return nil
}

// SeedBuiltinJobs persists cron registry declarations as builtin system tasks.
func (r *CronRuntime) SeedBuiltinJobs(ctx context.Context, jobs []cronx.Job) error {
	definitions := make([]TaskDefinition, 0, len(jobs))
	for _, job := range jobs {
		if err := job.Validate(); err != nil {
			return err
		}
		if err := r.rememberBuiltin(job); err != nil {
			return err
		}
		definitions = append(definitions, builtinTaskDefinition(job, r.now()))
	}
	if r.tasks == nil {
		return nil
	}

	return r.tasks.SeedBuiltinTasks(ctx, definitions)
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
	if r.tasks != nil {
		definitions, err := r.tasks.ListTasks(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]TaskSnapshot, 0, len(definitions))
		for _, definition := range definitions {
			snapshot, err := r.snapshotDefinition(ctx, definition)
			if err != nil {
				return nil, err
			}
			items = append(items, snapshot)
		}
		return items, nil
	}

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
	if r.tasks != nil {
		definition, err := r.tasks.GetTask(ctx, key)
		if err != nil {
			return TaskSnapshot{}, err
		}
		return r.snapshotDefinition(ctx, definition)
	}

	job, ok := r.findJob(key)
	if !ok {
		return TaskSnapshot{}, ErrTaskNotFound
	}

	return r.snapshot(ctx, job)
}

// CreateTask persists and dynamically schedules one user HTTP task.
func (r *CronRuntime) CreateTask(ctx context.Context, command TaskMutation) (TaskSnapshot, error) {
	if r.tasks == nil {
		return TaskSnapshot{}, errors.New("scheduler task repository is unavailable")
	}
	definition, err := mutationToDefinition(command, r.now())
	if err != nil {
		return TaskSnapshot{}, err
	}
	created, err := r.tasks.CreateTask(ctx, definition)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if err := r.refreshDefinitionSchedule(created); err != nil {
		return TaskSnapshot{}, err
	}
	return r.snapshotDefinition(ctx, created)
}

// UpdateTask updates mutable task fields and refreshes the in-process schedule.
func (r *CronRuntime) UpdateTask(ctx context.Context, key string, command TaskMutation) (TaskSnapshot, error) {
	if r.tasks == nil {
		return TaskSnapshot{}, errors.New("scheduler task repository is unavailable")
	}
	updated, err := r.tasks.UpdateTask(ctx, key, command)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if err := r.refreshDefinitionSchedule(updated); err != nil {
		return TaskSnapshot{}, err
	}
	return r.snapshotDefinition(ctx, updated)
}

// DeleteTask soft-deletes a user task and removes its runtime schedule.
func (r *CronRuntime) DeleteTask(ctx context.Context, key string) error {
	if r.tasks == nil {
		return errors.New("scheduler task repository is unavailable")
	}
	if err := r.tasks.DeleteTask(ctx, key); err != nil {
		return err
	}
	return r.removeScheduleIfExists(key)
}

// SetTaskEnabled toggles one task and refreshes its runtime schedule.
func (r *CronRuntime) SetTaskEnabled(ctx context.Context, key string, enabled bool) (TaskSnapshot, error) {
	if r.tasks == nil {
		return TaskSnapshot{}, errors.New("scheduler task repository is unavailable")
	}
	updated, err := r.tasks.SetTaskEnabled(ctx, key, enabled)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if err := r.refreshDefinitionSchedule(updated); err != nil {
		return TaskSnapshot{}, err
	}
	return r.snapshotDefinition(ctx, updated)
}

// ListRuns returns persisted run history for one task.
func (r *CronRuntime) ListRuns(ctx context.Context, query RunListQuery) (RunListResult, error) {
	if r.runs == nil {
		return RunListResult{}, errors.New("scheduler run repository is unavailable")
	}
	if err := r.ensureKnownTask(ctx, query.TaskKey); err != nil {
		return RunListResult{}, err
	}

	return r.runs.ListRuns(ctx, query)
}

// GetRun returns one persisted run detail.
func (r *CronRuntime) GetRun(ctx context.Context, id uint64) (TaskRun, error) {
	if r.runs == nil {
		return TaskRun{}, errors.New("scheduler run repository is unavailable")
	}
	return r.runs.GetRun(ctx, id)
}

// RunOnce executes one visible runtime job immediately.
func (r *CronRuntime) RunOnce(ctx context.Context, key string) (TaskRun, error) {
	if r.tasks != nil {
		definition, err := r.tasks.GetTask(ctx, key)
		if err != nil {
			return TaskRun{}, err
		}
		return r.runDefinition(ctx, definition, TriggerTypeManual)
	}

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
	if r.tasks != nil {
		definitions, err := r.tasks.ListTasks(ctx)
		if err != nil {
			return err
		}
		for _, definition := range definitions {
			if err := r.refreshDefinitionScheduleLocked(definition); err != nil {
				return err
			}
		}
	}
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
		Enabled:               job.DefaultEnabled,
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

	finishedRun, finishErr := r.runs.FinishRun(ctx, run.ID, status, finishedAt, "", errorMessage)
	if finishErr != nil {
		return finishedRun, finishErr
	}
	if runErr != nil {
		return finishedRun, runErr
	}

	return finishedRun, nil
}

func (r *CronRuntime) rememberBuiltin(job cronx.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := job.RuntimeKey()
	if _, exists := r.jobs[key]; exists {
		return nil
	}
	r.jobs[key] = job
	r.order = append(r.order, key)
	return nil
}

func (r *CronRuntime) refreshDefinitionSchedule(definition TaskDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.refreshDefinitionScheduleLocked(definition)
}

func (r *CronRuntime) refreshDefinitionScheduleLocked(definition TaskDefinition) error {
	key := definition.TaskKey
	if entryID, ok := r.entries[key]; ok {
		r.cron.Remove(entryID)
		delete(r.entries, key)
	}
	if !definition.Enabled || definition.DeletedAt != nil {
		return nil
	}

	entryID, err := r.addCronFuncLocked(key, definition.CronExpression, func(runCtx context.Context) (TaskRun, error) {
		return r.runDefinition(runCtx, definition, TriggerTypeCron)
	})
	if err != nil {
		return err
	}
	r.entries[key] = entryID
	return nil
}

func (r *CronRuntime) addCronFuncLocked(key string, schedule string, run func(context.Context) (TaskRun, error)) (cron.EntryID, error) {
	return r.cron.AddFunc(schedule, func() {
		runCtx := r.jobContext()
		if runCtx == nil {
			r.logger.Error("scheduler job skipped because lifecycle context is unavailable", zap.String("job", key))
			return
		}
		if _, runErr := run(runCtx); runErr != nil {
			if errors.Is(runErr, ErrTaskAlreadyRunning) {
				r.logger.Warn("scheduler job skipped because task is already running", zap.String("job", key))
				return
			}
			r.logger.Error("scheduler job failed", zap.String("job", key), zap.Error(runErr))
		}
	})
}

func (r *CronRuntime) removeScheduleIfExists(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entryID, ok := r.entries[key]; ok {
		r.cron.Remove(entryID)
		delete(r.entries, key)
	}
	return nil
}

func (r *CronRuntime) ensureKnownTask(ctx context.Context, key string) error {
	if r.tasks != nil {
		_, err := r.tasks.GetTask(ctx, key)
		return err
	}
	if _, ok := r.findJob(key); !ok {
		return ErrTaskNotFound
	}
	return nil
}

func (r *CronRuntime) snapshotDefinition(ctx context.Context, definition TaskDefinition) (TaskSnapshot, error) {
	snapshot := TaskSnapshot{
		ID:          definition.ID,
		Key:         definition.TaskKey,
		Name:        definition.TaskKey,
		Owner:       ownerForDefinition(definition),
		Module:      ownerForDefinition(definition),
		Type:        definition.TaskType,
		Title:       definition.Title,
		Description: definition.Description,
		Schedule:    definition.CronExpression,
		Enabled:     definition.Enabled,
		Builtin:     definition.Builtin,
		ConfigJSON:  definition.ConfigJSON,
		CreatedAt:   definition.CreatedAt,
		UpdatedAt:   definition.UpdatedAt,
		DeletedAt:   definition.DeletedAt,
	}
	if definition.Builtin {
		if job, ok := r.findJob(definition.TaskKey); ok {
			snapshot.Name = job.Name
			snapshot.Owner = job.RuntimeOwner()
			snapshot.Module = job.Module
			snapshot.DisplayMessageKey = job.DisplayMessageKey
			snapshot.DescriptionMessageKey = job.DescriptionMessageKey
		}
	}
	r.mu.RLock()
	_, snapshot.Running = r.running[definition.TaskKey]
	r.mu.RUnlock()
	if r.runs == nil {
		return snapshot, nil
	}
	latest, ok, err := r.runs.LatestRunByTask(ctx, definition.TaskKey)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if ok {
		snapshot.LastRun = &latest
	}
	return snapshot, nil
}

func (r *CronRuntime) runDefinition(ctx context.Context, definition TaskDefinition, trigger TriggerType) (TaskRun, error) {
	if definition.Builtin {
		job, ok := r.findJob(definition.TaskKey)
		if !ok {
			return TaskRun{}, ErrTaskNotFound
		}
		job.Schedule = definition.CronExpression
		job.DefaultEnabled = definition.Enabled
		return r.runJob(ctx, job, trigger)
	}
	return r.runHTTPTask(ctx, definition, trigger)
}

func (r *CronRuntime) runHTTPTask(ctx context.Context, definition TaskDefinition, trigger TriggerType) (TaskRun, error) {
	if r.runs == nil {
		return TaskRun{}, errors.New("scheduler run repository is unavailable")
	}
	if err := validateHTTPDefinition(definition); err != nil {
		return TaskRun{}, err
	}
	if err := r.markRunning(definition.TaskKey); err != nil {
		return TaskRun{}, err
	}
	defer r.markFinished(definition.TaskKey)

	startedAt := r.now()
	run, err := r.runs.CreateRun(ctx, TaskRun{
		TaskKey:     definition.TaskKey,
		TaskName:    definition.Title,
		Owner:       ownerForDefinition(definition),
		Module:      moduleForDefinition(definition),
		TaskType:    definition.TaskType,
		TriggerType: trigger,
		Status:      RunStatusRunning,
		StartedAt:   startedAt,
		CreatedAt:   startedAt,
	})
	if err != nil {
		return TaskRun{}, err
	}

	resultSummary, errorMessage := executeHTTPTask(ctx, definition.ConfigJSON)
	finishedAt := r.now()
	status := RunStatusSuccess
	if errorMessage != "" {
		status = RunStatusFailed
	}
	finishedRun, finishErr := r.runs.FinishRun(ctx, run.ID, status, finishedAt, resultSummary, errorMessage)
	if finishErr != nil {
		return finishedRun, finishErr
	}
	if errorMessage != "" {
		return finishedRun, errors.New(errorMessage)
	}
	return finishedRun, nil
}

func builtinTaskDefinition(job cronx.Job, now time.Time) TaskDefinition {
	title := strings.TrimSpace(job.DisplayMessageKey)
	if title == "" {
		title = job.RuntimeKey()
	}
	description := strings.TrimSpace(job.DescriptionMessageKey)
	if description == "" {
		description = title
	}
	return TaskDefinition{
		TaskKey:        job.RuntimeKey(),
		TaskType:       cronx.TaskTypeSystem,
		Title:          title,
		Description:    description,
		CronExpression: job.Schedule,
		Enabled:        job.DefaultEnabled,
		Builtin:        true,
		ConfigJSON:     "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func mutationToDefinition(command TaskMutation, now time.Time) (TaskDefinition, error) {
	taskType := command.TaskType
	if taskType == "" {
		taskType = cronx.TaskTypeHTTP
	}
	definition := TaskDefinition{
		TaskKey:        command.TaskKey,
		TaskType:       taskType,
		Title:          command.Title,
		Description:    command.Description,
		CronExpression: command.CronExpression,
		Enabled:        command.Enabled,
		Builtin:        false,
		ConfigJSON:     command.ConfigJSON,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := validateDefinition(definition); err != nil {
		return TaskDefinition{}, err
	}
	return definition, nil
}

func validateDefinition(definition TaskDefinition) error {
	if definition.TaskKey == "" || definition.CronExpression == "" || definition.Title == "" {
		return ErrTaskValidation
	}
	if definition.TaskType != cronx.TaskTypeSystem && definition.TaskType != cronx.TaskTypeHTTP {
		return ErrTaskValidation
	}
	if err := validateCronExpression(definition.CronExpression); err != nil {
		return err
	}
	if definition.TaskType == cronx.TaskTypeHTTP {
		return validateHTTPDefinition(definition)
	}
	return nil
}

func validateCronExpression(expression string) error {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(expression); err != nil {
		return fmt.Errorf("%w: invalid cron expression", ErrTaskValidation)
	}
	return nil
}

func validateHTTPDefinition(definition TaskDefinition) error {
	var config HTTPTaskConfig
	if err := json.Unmarshal([]byte(definition.ConfigJSON), &config); err != nil {
		return fmt.Errorf("%w: invalid http config", ErrTaskValidation)
	}
	_, err := normalizeHTTPTaskConfig(config)
	return err
}

func normalizeHTTPTaskConfig(config HTTPTaskConfig) (HTTPTaskConfig, error) {
	method := strings.ToUpper(strings.TrimSpace(config.Method))
	if method == "" {
		method = http.MethodGet
	}
	if method != http.MethodGet && method != http.MethodPost {
		return HTTPTaskConfig{}, fmt.Errorf("%w: unsupported http method", ErrTaskValidation)
	}
	parsed, err := url.Parse(strings.TrimSpace(config.URL))
	if err != nil || parsed == nil || parsed.Host == "" {
		return HTTPTaskConfig{}, fmt.Errorf("%w: invalid http url", ErrTaskValidation)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return HTTPTaskConfig{}, fmt.Errorf("%w: unsupported http scheme", ErrTaskValidation)
	}
	timeout := config.TimeoutSeconds
	if timeout == 0 {
		timeout = 30
	}
	if timeout < 1 || timeout > 120 {
		return HTTPTaskConfig{}, fmt.Errorf("%w: invalid http timeout", ErrTaskValidation)
	}
	config.Method = method
	config.URL = parsed.String()
	config.TimeoutSeconds = timeout
	return config, nil
}

func executeHTTPTask(ctx context.Context, rawConfig string) (string, string) {
	var config HTTPTaskConfig
	if err := json.Unmarshal([]byte(rawConfig), &config); err != nil {
		return "", err.Error()
	}
	config, err := normalizeHTTPTaskConfig(config)
	if err != nil {
		return "", err.Error()
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()
	var body io.Reader
	if config.Method == http.MethodPost {
		body = strings.NewReader(config.Body)
	}
	request, err := http.NewRequestWithContext(timeoutCtx, config.Method, config.URL, body)
	if err != nil {
		return "", err.Error()
	}
	for key, value := range config.Headers {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err.Error()
	}
	defer func() {
		_ = response.Body.Close()
	}()
	limited, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if err != nil {
		return "", err.Error()
	}
	summary := fmt.Sprintf("HTTP %d %s", response.StatusCode, strings.TrimSpace(string(limited)))
	if response.StatusCode >= http.StatusBadRequest {
		return summary, fmt.Sprintf("http status %d", response.StatusCode)
	}
	return summary, ""
}

func ownerForDefinition(definition TaskDefinition) string {
	if definition.Builtin {
		return "system"
	}
	return "scheduler"
}

func moduleForDefinition(definition TaskDefinition) string {
	if definition.Builtin {
		return "system"
	}
	return "scheduler"
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
