// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"graft/server/internal/cronx"
)

// Runtime exposes the repository-stable scheduler capability.
type Runtime interface {
	RegisterJob(job cronx.Job) error
	SeedBuiltinJobs(ctx context.Context, jobs []cronx.Job) error
	RemoveJob(name string) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	ListJobDefinitions(ctx context.Context) ([]JobDefinitionSnapshot, error)
	GetJobDefinition(ctx context.Context, key string) (JobDefinitionSnapshot, error)
	ListTasks(ctx context.Context, query TaskListQuery) (TaskListResult, error)
	GetTask(ctx context.Context, key string) (TaskSnapshot, error)
	CreateTask(ctx context.Context, command TaskMutation) (TaskSnapshot, error)
	UpdateTask(ctx context.Context, key string, command TaskMutation) (TaskSnapshot, error)
	DeleteTask(ctx context.Context, key string) error
	SetTaskEnabled(ctx context.Context, key string, enabled bool) (TaskSnapshot, error)
	ListRuns(ctx context.Context, query RunListQuery) (RunListResult, error)
	GetRun(ctx context.Context, id uint64) (TaskRun, error)
	RunOnce(ctx context.Context, key string) (TaskRun, error)
	RunOnceWithTrigger(ctx context.Context, key string, trigger RunTrigger) (TaskRun, error)
	RunAction(ctx context.Context, taskKey string, actionKey string, configJSON string) (JobActionResult, error)
}

// DefaultConfigResolver resolves administrator-overridden defaults for jobs whose defaults are system-config backed.
type DefaultConfigResolver interface {
	ResolveDefaultConfig(ctx context.Context, key string) (string, error)
}

// RunFailureNotifier observes persisted failed scheduler runs.
type RunFailureNotifier interface {
	NotifyRunFailed(ctx context.Context, run TaskRun)
}

// RunSuccessNotifier observes persisted successful scheduler runs.
type RunSuccessNotifier interface {
	NotifyRunSucceeded(ctx context.Context, run TaskRun, trigger RunTrigger)
}

// RunStatus records the result state of one runtime job execution.
type RunStatus string

const (
	// RunStatusRunning means the job execution has been created but not finished.
	RunStatusRunning RunStatus = "running"
	// RunStatusSuccess means the job execution finished without a handler error.
	RunStatusSuccess RunStatus = "success"
	// RunStatusFailed means the job execution finished with a handler error.
	RunStatusFailed RunStatus = "failed"
)

// TriggerType records why a runtime job execution started.
type TriggerType string

const (
	// TriggerTypeCron records a run started by cron scheduling.
	TriggerTypeCron TriggerType = "cron"
	// TriggerTypeManual records a run started by an explicit API request.
	TriggerTypeManual TriggerType = "manual"
	// TriggerTypeStartup records a run started during scheduler startup.
	TriggerTypeStartup TriggerType = "startup"
)

// RunTrigger records scheduler-domain trigger metadata without request-layer dependencies.
type RunTrigger struct {
	Type          TriggerType
	TriggerUserID uint64
}

var (
	// ErrTaskNotFound is returned when a scheduled task or run cannot be found.
	ErrTaskNotFound = errors.New("scheduler task not found")
	// ErrJobDefinitionNotFound is returned when a scheduled task references an unknown job definition.
	ErrJobDefinitionNotFound = errors.New("scheduler job definition not found")
	// ErrJobActionNotFound is returned when a job definition action is unknown.
	ErrJobActionNotFound = errors.New("scheduler job action not found")
	// ErrTaskAlreadyRunning is returned when a manual run is requested while the task is active.
	ErrTaskAlreadyRunning = errors.New("scheduler task already running")
	// ErrTaskImmutable is returned when a caller tries to change builtin or identity fields.
	ErrTaskImmutable = errors.New("scheduler task field is immutable")
	// ErrTaskValidation is returned when task, job, or cron input is invalid.
	ErrTaskValidation = errors.New("scheduler task validation failed")
)

var reservedTaskKeys = map[string]struct{}{
	"jobs": {},
	"runs": {},
}

const (
	taskConfigSourceSystem = "system"
	taskConfigSourceUser   = "user"
	runFailureNotifyTTL    = 3 * time.Second
)

// JobDefinitionSnapshot describes one persisted, creatable scheduler job type.
type JobDefinitionSnapshot struct {
	ID             uint64
	JobKey         string
	ModuleKey      string
	TitleKey       string
	Title          string
	DescriptionKey string
	Description    string
	ConfigSchema   string
	DefaultConfig  string
	DefaultCron    string
	Enabled        bool
	Actions        []JobActionSnapshot
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

// JobActionSnapshot describes one backend-defined Job Definition action.
type JobActionSnapshot struct {
	Key            string
	TitleKey       string
	Title          string
	DescriptionKey string
	Description    string
}

// TaskSnapshot is the internal service model for scheduled task instances.
type TaskSnapshot struct {
	ID                    uint64
	Key                   string
	JobKey                string
	ModuleKey             string
	Name                  string
	Title                 string
	Description           string
	DisplayMessageKey     string
	DescriptionMessageKey string
	Schedule              string
	Enabled               bool
	Builtin               bool
	ConfigJSON            string
	EffectiveConfig       string
	Running               bool
	LastRun               *TaskRun
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
}

// TaskRun is the persisted run-history model for scheduler runtime jobs.
type TaskRun struct {
	ID              uint64
	TaskKey         string
	JobKey          string
	TaskName        string
	Owner           string
	Module          string
	TriggerType     TriggerType
	Status          RunStatus
	Error           string
	Result          string
	ResultJSON      string
	EffectiveConfig string
	StartedAt       time.Time
	FinishedAt      *time.Time
	DurationMS      *int64
	CreatedAt       time.Time
}

// JobActionResult is the non-persisted result of a backend-defined Job Definition action.
type JobActionResult struct {
	ActionKey       string
	TaskKey         string
	JobKey          string
	Result          cronx.JobRunResult
	EffectiveConfig string
}

type actionExecution struct {
	definition    TaskDefinition
	jobDefinition JobDefinition
	action        JobActionSnapshot
	job           cronx.Job
}

// JobDefinition is the DB-backed authority for one creatable job type.
type JobDefinition struct {
	ID             uint64
	JobKey         string
	ModuleKey      string
	TitleKey       string
	Title          string
	DescriptionKey string
	Description    string
	ConfigSchema   string
	DefaultConfig  string
	DefaultCron    string
	Enabled        bool
	Actions        []JobActionSnapshot
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

// TaskDefinition is the DB-backed authority for one scheduled task instance.
type TaskDefinition struct {
	ID             uint64
	TaskKey        string
	JobKey         string
	ModuleKey      string
	Title          string
	Description    string
	CronExpression string
	Enabled        bool
	Builtin        bool
	ConfigJSON     string
	ConfigSource   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

// TaskMutation carries create/update input before HTTP routes are bound.
type TaskMutation struct {
	TaskKey        string
	JobKey         string
	Title          string
	Description    string
	CronExpression string
	Enabled        bool
	EnabledSet     bool
	ConfigJSON     string
}

// TaskListQuery scopes scheduled task lookup.
type TaskListQuery struct {
	Limit  int
	Offset int
}

// TaskListResult contains one page of scheduled tasks plus a total count.
type TaskListResult struct {
	Items []TaskSnapshot
	Total int
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

// RunRepository persists execution history for scheduled task runs.
type RunRepository interface {
	CreateRun(ctx context.Context, run TaskRun) (TaskRun, error)
	FinishRun(ctx context.Context, command RunFinishCommand) (TaskRun, error)
	ListRuns(ctx context.Context, query RunListQuery) (RunListResult, error)
	LatestRunByTask(ctx context.Context, taskKey string) (TaskRun, bool, error)
	GetRun(ctx context.Context, id uint64) (TaskRun, error)
}

// RunFinishCommand captures the persisted result for one completed execution.
type RunFinishCommand struct {
	ID            uint64
	Status        RunStatus
	FinishedAt    time.Time
	ResultJSON    string
	ResultSummary string
	ErrorMessage  string
}

// TaskRepository persists user-created and builtin scheduled task instances.
type TaskRepository interface {
	SeedBuiltinTasks(ctx context.Context, tasks []TaskDefinition) error
	CreateTask(ctx context.Context, task TaskDefinition) (TaskDefinition, error)
	UpdateTask(ctx context.Context, key string, patch TaskMutation) (TaskDefinition, error)
	DeleteTask(ctx context.Context, key string) error
	SetTaskEnabled(ctx context.Context, key string, enabled bool) (TaskDefinition, error)
	ListTasks(ctx context.Context, query TaskListQuery) ([]TaskDefinition, int, error)
	GetTask(ctx context.Context, key string) (TaskDefinition, error)
}

// JobDefinitionRepository persists module-registered scheduler job definitions.
type JobDefinitionRepository interface {
	SyncJobDefinitions(ctx context.Context, definitions []JobDefinition) error
	ListJobDefinitions(ctx context.Context) ([]JobDefinition, error)
	GetJobDefinition(ctx context.Context, key string) (JobDefinition, error)
}

// CronRuntime is the in-process scheduler backed by robfig/cron.
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
	jobDefinitions  JobDefinitionRepository
	defaultConfigs  DefaultConfigResolver
	failureNotifier RunFailureNotifier
	successNotifier RunSuccessNotifier
	now             func() time.Time
}

// New constructs an in-process cron runtime with an optional run repository.
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

// SetTaskRepository attaches the scheduled task persistence backend.
func (r *CronRuntime) SetTaskRepository(repository TaskRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks = repository
}

// SetJobDefinitionRepository attaches the job definition persistence backend.
func (r *CronRuntime) SetJobDefinitionRepository(repository JobDefinitionRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobDefinitions = repository
}

// SetDefaultConfigResolver attaches the optional system-config backed default resolver.
func (r *CronRuntime) SetDefaultConfigResolver(resolver DefaultConfigResolver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultConfigs = resolver
}

// SetRunFailureNotifier attaches a non-blocking observer for persisted failed runs.
func (r *CronRuntime) SetRunFailureNotifier(notifier RunFailureNotifier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failureNotifier = notifier
}

// SetRunSuccessNotifier attaches a non-blocking observer for persisted successful manual runs.
func (r *CronRuntime) SetRunSuccessNotifier(notifier RunSuccessNotifier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.successNotifier = notifier
}

// RegisterJob adds an in-memory job handler declaration to the runtime.
func (r *CronRuntime) RegisterJob(job cronx.Job) error {
	if err := validateJob(job); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := job.RuntimeKey()
	if _, exists := r.jobs[key]; exists {
		return fmt.Errorf("job already registered: %s", key)
	}
	r.jobs[key] = job
	r.order = append(r.order, key)
	return nil
}

// SeedBuiltinJobs syncs module-registered jobs and their builtin scheduled task instances.
func (r *CronRuntime) SeedBuiltinJobs(ctx context.Context, jobs []cronx.Job) error {
	definitions := make([]JobDefinition, 0, len(jobs))
	tasks := make([]TaskDefinition, 0, len(jobs))
	for _, job := range jobs {
		if err := r.RegisterJob(job); err != nil {
			var duplicateErr error
			if strings.Contains(err.Error(), "job already registered") {
				duplicateErr = nil
			} else {
				duplicateErr = err
			}
			if duplicateErr != nil {
				return duplicateErr
			}
		}
		definition, err := r.jobDefinitionFromJob(ctx, job)
		if err != nil {
			return err
		}
		definitions = append(definitions, definition)
		task, err := r.builtinTaskDefinition(ctx, job)
		if err != nil {
			return err
		}
		tasks = append(tasks, task)
	}
	if r.jobDefinitions != nil {
		if err := r.jobDefinitions.SyncJobDefinitions(ctx, definitions); err != nil {
			return err
		}
	}
	if r.tasks != nil {
		return r.tasks.SeedBuiltinTasks(ctx, tasks)
	}
	return nil
}

func (r *CronRuntime) builtinTaskDefinition(ctx context.Context, job cronx.Job) (TaskDefinition, error) {
	task := builtinTaskDefinition(job, r.now())
	if r.tasks == nil {
		return task, nil
	}
	existing, err := r.tasks.GetTask(ctx, task.TaskKey)
	if errors.Is(err, ErrTaskNotFound) {
		return task, nil
	}
	if err != nil {
		return TaskDefinition{}, err
	}
	if !existing.Builtin {
		return task, nil
	}
	if existing.ConfigSource != taskConfigSourceUser {
		if historicalUserOverride, err := sanitizeHistoricalUserOverride(job, task.ConfigJSON, existing.ConfigJSON); err != nil {
			return TaskDefinition{}, err
		} else if historicalUserOverride != "" {
			task.ConfigJSON = historicalUserOverride
			task.ConfigSource = taskConfigSourceUser
			return task, nil
		}
		return task, nil
	}
	configJSON, err := sanitizeConfigJSON(job.RuntimeConfigSchema(), existing.ConfigJSON)
	if err != nil {
		return TaskDefinition{}, err
	}
	task.ConfigJSON = configJSON
	task.ConfigSource = taskConfigSourceUser
	return task, nil
}

func sanitizeHistoricalUserOverride(job cronx.Job, seededDefault string, existingConfig string) (string, error) {
	existingConfig = strings.TrimSpace(existingConfig)
	if existingConfig == "" {
		return "", nil
	}
	if sameJSONObject(existingConfig, "{}") || sameJSONObject(existingConfig, seededDefault) {
		return "", nil
	}
	configJSON, err := sanitizeConfigJSON(job.RuntimeConfigSchema(), existingConfig)
	if err != nil {
		return "", err
	}
	if sameJSONObject(configJSON, "{}") || sameJSONObject(configJSON, seededDefault) {
		return "", nil
	}
	return configJSON, nil
}

// RemoveJob removes a registered in-memory job and any active cron schedule for it.
func (r *CronRuntime) RemoveJob(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entryID, ok := r.entries[name]; ok {
		r.cron.Remove(entryID)
		delete(r.entries, name)
	}
	if _, ok := r.jobs[name]; !ok {
		return errors.New("job not found")
	}
	delete(r.jobs, name)
	r.order = removeKey(r.order, name)
	return nil
}

// ListJobDefinitions returns the creatable scheduler job definitions.
func (r *CronRuntime) ListJobDefinitions(ctx context.Context) ([]JobDefinitionSnapshot, error) {
	if r.jobDefinitions != nil {
		definitions, err := r.jobDefinitions.ListJobDefinitions(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]JobDefinitionSnapshot, 0, len(definitions))
		for _, definition := range definitions {
			enriched, err := r.enrichJobDefinition(ctx, definition)
			if err != nil {
				return nil, err
			}
			items = append(items, jobDefinitionSnapshot(enriched))
		}
		return items, nil
	}

	r.mu.RLock()
	jobs := make([]cronx.Job, 0, len(r.order))
	for _, key := range r.order {
		jobs = append(jobs, r.jobs[key])
	}
	r.mu.RUnlock()

	items := make([]JobDefinitionSnapshot, 0, len(jobs))
	for _, job := range jobs {
		definition, err := r.jobDefinitionFromJob(ctx, job)
		if err != nil {
			return nil, err
		}
		items = append(items, jobDefinitionSnapshot(definition))
	}
	return items, nil
}

// GetJobDefinition returns one creatable scheduler job definition.
func (r *CronRuntime) GetJobDefinition(ctx context.Context, key string) (JobDefinitionSnapshot, error) {
	definition, err := r.requireKnownJob(ctx, key)
	if err != nil {
		return JobDefinitionSnapshot{}, err
	}
	return jobDefinitionSnapshot(definition), nil
}

// ListTasks returns active scheduled task instances.
func (r *CronRuntime) ListTasks(ctx context.Context, query TaskListQuery) (TaskListResult, error) {
	if r.tasks == nil {
		return TaskListResult{}, errors.New("scheduler task repository is unavailable")
	}
	definitions, total, err := r.tasks.ListTasks(ctx, query)
	if err != nil {
		return TaskListResult{}, err
	}
	items := make([]TaskSnapshot, 0, len(definitions))
	for _, definition := range definitions {
		snapshot, err := r.snapshotDefinition(ctx, definition)
		if err != nil {
			return TaskListResult{}, err
		}
		items = append(items, snapshot)
	}
	return TaskListResult{Items: items, Total: total}, nil
}

// GetTask returns one active scheduled task instance by key.
func (r *CronRuntime) GetTask(ctx context.Context, key string) (TaskSnapshot, error) {
	if r.tasks == nil {
		return TaskSnapshot{}, errors.New("scheduler task repository is unavailable")
	}
	definition, err := r.tasks.GetTask(ctx, key)
	if err != nil {
		return TaskSnapshot{}, err
	}
	return r.snapshotDefinition(ctx, definition)
}

// CreateTask persists and schedules a user-created scheduled task instance.
func (r *CronRuntime) CreateTask(ctx context.Context, command TaskMutation) (TaskSnapshot, error) {
	if r.tasks == nil {
		return TaskSnapshot{}, errors.New("scheduler task repository is unavailable")
	}
	job, err := r.requireKnownJob(ctx, command.JobKey)
	if err != nil {
		return TaskSnapshot{}, err
	}
	definition, err := mutationToDefinition(command, job, r.now())
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

// UpdateTask updates mutable scheduled task fields and refreshes its cron schedule.
func (r *CronRuntime) UpdateTask(ctx context.Context, key string, command TaskMutation) (TaskSnapshot, error) {
	if r.tasks == nil {
		return TaskSnapshot{}, errors.New("scheduler task repository is unavailable")
	}
	if command.JobKey != "" {
		if _, err := r.requireKnownJob(ctx, command.JobKey); err != nil {
			return TaskSnapshot{}, err
		}
	}
	existing, err := r.tasks.GetTask(ctx, key)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if err := validateTaskPatch(key, existing, command); err != nil {
		return TaskSnapshot{}, err
	}
	next := applyTaskPatch(existing, command)
	if err := r.validateTaskConfig(ctx, next); err != nil {
		return TaskSnapshot{}, err
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

// DeleteTask soft-deletes a user-created scheduled task and removes its cron schedule.
func (r *CronRuntime) DeleteTask(ctx context.Context, key string) error {
	if r.tasks == nil {
		return errors.New("scheduler task repository is unavailable")
	}
	if err := r.tasks.DeleteTask(ctx, key); err != nil {
		return err
	}
	return r.removeScheduleIfExists(key)
}

// SetTaskEnabled toggles a scheduled task and refreshes its cron schedule.
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

// ListRuns returns a page of run history for one scheduled task.
func (r *CronRuntime) ListRuns(ctx context.Context, query RunListQuery) (RunListResult, error) {
	if r.runs == nil {
		return RunListResult{}, errors.New("scheduler run repository is unavailable")
	}
	if err := r.ensureKnownTask(ctx, query.TaskKey); err != nil {
		return RunListResult{}, err
	}
	return r.runs.ListRuns(ctx, query)
}

// GetRun returns one persisted run-history record by id.
func (r *CronRuntime) GetRun(ctx context.Context, id uint64) (TaskRun, error) {
	if r.runs == nil {
		return TaskRun{}, errors.New("scheduler run repository is unavailable")
	}
	return r.runs.GetRun(ctx, id)
}

// RunOnce starts one manual execution for a scheduled task.
func (r *CronRuntime) RunOnce(ctx context.Context, key string) (TaskRun, error) {
	return r.RunOnceWithTrigger(ctx, key, RunTrigger{Type: TriggerTypeManual})
}

// RunOnceWithTrigger starts one manual execution with scheduler-domain trigger metadata.
func (r *CronRuntime) RunOnceWithTrigger(ctx context.Context, key string, trigger RunTrigger) (TaskRun, error) {
	if r.tasks == nil {
		return TaskRun{}, errors.New("scheduler task repository is unavailable")
	}
	trigger = normalizeManualRunTrigger(trigger)
	definition, err := r.tasks.GetTask(ctx, key)
	if err != nil {
		return TaskRun{}, err
	}
	return r.runDefinition(ctx, definition, trigger)
}

// RunAction executes one backend-defined Job Definition action without writing run history.
func (r *CronRuntime) RunAction(ctx context.Context, taskKey string, actionKey string, configJSON string) (JobActionResult, error) {
	execution, err := r.resolveActionExecution(ctx, taskKey, actionKey)
	if err != nil {
		return JobActionResult{}, err
	}
	if err := r.markRunning(execution.definition.TaskKey); err != nil {
		return JobActionResult{}, err
	}
	defer r.markFinished(execution.definition.TaskKey)

	taskConfig, err := r.taskConfigForEffective(execution.definition)
	if err != nil {
		return JobActionResult{}, err
	}
	effectiveConfig, err := actionEffectiveConfigJSON(execution.jobDefinition, taskConfig, configJSON)
	if err != nil {
		return JobActionResult{}, err
	}
	if validationErr := ValidateConfigJSON(execution.jobDefinition.ConfigSchema, effectiveConfig); validationErr != nil {
		return JobActionResult{}, validationErr
	}
	result, runErr := invokeJobAction(ctx, execution.job, execution.action.Key, effectiveConfig)
	_, _ = completeJobRunResult(&result, runErr)
	if runErr != nil {
		return jobActionResult(execution, result, effectiveConfig), runErr
	}
	return jobActionResult(execution, result, effectiveConfig), nil
}

// Start schedules persisted enabled tasks and starts the cron engine.
func (r *CronRuntime) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("lifecycle context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return nil
	}
	if r.tasks != nil {
		definitions, _, err := r.tasks.ListTasks(ctx, TaskListQuery{})
		if err != nil {
			return err
		}
		for _, definition := range definitions {
			if err := r.refreshDefinitionScheduleLocked(definition); err != nil {
				return err
			}
		}
	}
	r.lifecycleCtx, r.lifecycleCancel = context.WithCancel(ctx)
	r.cron.Start()
	r.started = true
	return nil
}

// Stop cancels runtime-owned contexts and stops the cron engine.
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

func (r *CronRuntime) requireKnownJob(ctx context.Context, key string) (JobDefinition, error) {
	if strings.TrimSpace(key) == "" {
		return JobDefinition{}, ErrJobDefinitionNotFound
	}
	if r.jobDefinitions != nil {
		definition, err := r.jobDefinitions.GetJobDefinition(ctx, key)
		if err != nil {
			return JobDefinition{}, err
		}
		if !definition.Enabled || definition.DeletedAt != nil {
			return JobDefinition{}, ErrJobDefinitionNotFound
		}
		return r.enrichJobDefinition(ctx, definition)
	}
	job, ok := r.findJob(key)
	if !ok {
		return JobDefinition{}, ErrJobDefinitionNotFound
	}
	return r.jobDefinitionFromJob(ctx, job)
}

func (r *CronRuntime) snapshotDefinition(ctx context.Context, definition TaskDefinition) (TaskSnapshot, error) {
	snapshot := TaskSnapshot{
		ID:          definition.ID,
		Key:         definition.TaskKey,
		JobKey:      definition.JobKey,
		ModuleKey:   definition.ModuleKey,
		Name:        definition.TaskKey,
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
	if jobDefinition, err := r.requireKnownJob(ctx, definition.JobKey); err == nil {
		taskConfig, taskConfigErr := r.taskConfigForEffective(definition)
		if taskConfigErr != nil {
			return TaskSnapshot{}, taskConfigErr
		}
		effectiveConfig, mergeErr := effectiveConfigJSON(jobDefinition.DefaultConfig, taskConfig)
		if mergeErr != nil {
			return TaskSnapshot{}, mergeErr
		}
		snapshot.EffectiveConfig = effectiveConfig
	}
	if job, ok := r.findJob(definition.JobKey); ok {
		snapshot.Name = job.RuntimeKey()
		snapshot.DisplayMessageKey = job.RuntimeTitleKey()
		snapshot.DescriptionMessageKey = job.RuntimeDescriptionKey()
		if snapshot.ModuleKey == "" {
			snapshot.ModuleKey = job.Module
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

func (r *CronRuntime) runDefinition(ctx context.Context, definition TaskDefinition, trigger RunTrigger) (TaskRun, error) {
	if r.runs == nil {
		return TaskRun{}, errors.New("scheduler run repository is unavailable")
	}
	if err := validateDefinition(definition); err != nil {
		return TaskRun{}, err
	}
	job, ok := r.findJob(definition.JobKey)
	if !ok {
		return TaskRun{}, ErrJobDefinitionNotFound
	}
	if err := r.markRunning(definition.TaskKey); err != nil {
		return TaskRun{}, err
	}
	defer r.markFinished(definition.TaskKey)

	run, err := r.createStartedRun(ctx, definition, trigger.Type)
	if err != nil {
		return TaskRun{}, err
	}

	effectiveConfig, err := r.effectiveConfigForRun(ctx, definition)
	if err != nil {
		return TaskRun{}, err
	}
	jobResult, runErr := job.Invoke(ctx, effectiveConfig)
	finishedRun, finishErr := r.finishRun(ctx, run.ID, trigger, jobResult, runErr)
	if finishErr != nil {
		return finishedRun, finishErr
	}
	finishedRun.EffectiveConfig = effectiveConfig
	if runErr != nil {
		return finishedRun, runErr
	}
	return finishedRun, nil
}

func (r *CronRuntime) createStartedRun(ctx context.Context, definition TaskDefinition, trigger TriggerType) (TaskRun, error) {
	startedAt := r.now()
	return r.runs.CreateRun(ctx, TaskRun{
		TaskKey:     definition.TaskKey,
		JobKey:      definition.JobKey,
		TaskName:    definition.Title,
		Owner:       definition.ModuleKey,
		Module:      definition.ModuleKey,
		TriggerType: trigger,
		Status:      RunStatusRunning,
		StartedAt:   startedAt,
		CreatedAt:   startedAt,
	})
}

func (r *CronRuntime) effectiveConfigForRun(ctx context.Context, definition TaskDefinition) (string, error) {
	jobDefinition, err := r.requireKnownJob(ctx, definition.JobKey)
	if err != nil {
		return "", err
	}
	taskConfig, err := r.taskConfigForEffective(definition)
	if err != nil {
		return "", err
	}
	effectiveConfig, err := effectiveConfigJSON(jobDefinition.DefaultConfig, taskConfig)
	if err != nil {
		return "", err
	}
	if validationErr := ValidateConfigJSON(jobDefinition.ConfigSchema, effectiveConfig); validationErr != nil {
		return "", validationErr
	}
	return effectiveConfig, nil
}

func (r *CronRuntime) finishRun(ctx context.Context, id uint64, trigger RunTrigger, result cronx.JobRunResult, runErr error) (TaskRun, error) {
	command := r.runFinishCommand(id, result, runErr)
	finished, err := r.runs.FinishRun(finishRunContext(ctx), command)
	if err == nil && finished.Status == RunStatusFailed {
		r.notifyRunFailed(ctx, finished)
	}
	if err == nil && finished.Status == RunStatusSuccess && trigger.Type == TriggerTypeManual {
		r.notifyRunSucceeded(ctx, finished, trigger)
	}
	return finished, err
}

func (r *CronRuntime) runFinishCommand(id uint64, result cronx.JobRunResult, runErr error) RunFinishCommand {
	status, errorMessage := completeJobRunResult(&result, runErr)
	resultJSON, resultSummary := encodeJobRunResult(result)
	return RunFinishCommand{
		ID:            id,
		Status:        status,
		FinishedAt:    r.now(),
		ResultJSON:    resultJSON,
		ResultSummary: resultSummary,
		ErrorMessage:  errorMessage,
	}
}

func completeJobRunResult(result *cronx.JobRunResult, runErr error) (RunStatus, string) {
	if runErr == nil {
		if result.Stage == "" {
			result.Stage = "completed"
		}
		return RunStatusSuccess, ""
	}
	errorMessage := runErr.Error()
	if result.Summary == "" {
		result.Summary = errorMessage
	}
	if result.Stage == "" {
		result.Stage = "failed"
	}
	return RunStatusFailed, errorMessage
}

func normalizeManualRunTrigger(trigger RunTrigger) RunTrigger {
	if trigger.Type == "" {
		trigger.Type = TriggerTypeManual
	}
	if trigger.Type != TriggerTypeManual {
		trigger.Type = TriggerTypeManual
	}
	return trigger
}

func finishRunContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

func (r *CronRuntime) notifyRunFailed(ctx context.Context, run TaskRun) {
	r.mu.RLock()
	notifier := r.failureNotifier
	r.mu.RUnlock()
	if notifier == nil {
		return
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				r.logger.Error("scheduler run failure notifier panicked",
					zap.String("task", run.TaskKey),
					zap.Uint64("runID", run.ID),
					zap.Any("panic", recovered),
				)
			}
		}()
		notifyCtx, cancel := context.WithTimeout(finishRunContext(ctx), runFailureNotifyTTL)
		defer cancel()
		notifier.NotifyRunFailed(notifyCtx, run)
	}()
}

func (r *CronRuntime) notifyRunSucceeded(ctx context.Context, run TaskRun, trigger RunTrigger) {
	r.mu.RLock()
	notifier := r.successNotifier
	r.mu.RUnlock()
	if notifier == nil {
		return
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				r.logger.Error("scheduler run success notifier panicked",
					zap.String("task", run.TaskKey),
					zap.Uint64("runID", run.ID),
					zap.Any("panic", recovered),
				)
			}
		}()
		notifyCtx, cancel := context.WithTimeout(finishRunContext(ctx), runFailureNotifyTTL)
		defer cancel()
		notifier.NotifyRunSucceeded(notifyCtx, run, trigger)
	}()
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
		return r.runDefinition(runCtx, definition, RunTrigger{Type: TriggerTypeCron})
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
			r.logger.Error("scheduler job skipped because lifecycle context is unavailable", zap.String("task", key))
			return
		}
		if _, runErr := run(runCtx); runErr != nil {
			if errors.Is(runErr, ErrTaskAlreadyRunning) {
				r.logger.Warn("scheduler job skipped because task is already running", zap.String("task", key))
				return
			}
			r.logger.Error("scheduler job failed", zap.String("task", key), zap.Error(runErr))
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
	if r.tasks == nil {
		return errors.New("scheduler task repository is unavailable")
	}
	_, err := r.tasks.GetTask(ctx, key)
	return err
}

func jobDefinitionFromJob(job cronx.Job, now time.Time) JobDefinition {
	return JobDefinition{
		JobKey:         job.RuntimeKey(),
		ModuleKey:      job.RuntimeOwner(),
		TitleKey:       job.RuntimeTitleKey(),
		Title:          job.RuntimeTitle(),
		DescriptionKey: job.RuntimeDescriptionKey(),
		Description:    job.RuntimeDescription(),
		ConfigSchema:   job.RuntimeConfigSchema(),
		DefaultConfig:  job.RuntimeDefaultConfig(),
		DefaultCron:    strings.TrimSpace(job.Schedule),
		Enabled:        true,
		Actions:        jobActionsFromJob(job),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (r *CronRuntime) jobDefinitionFromJob(ctx context.Context, job cronx.Job) (JobDefinition, error) {
	definition := jobDefinitionFromJob(job, r.now())
	defaultConfig, err := r.resolveJobDefaultConfig(ctx, job)
	if err != nil {
		return JobDefinition{}, err
	}
	definition.DefaultConfig = defaultConfig
	return definition, nil
}

func builtinTaskDefinition(job cronx.Job, now time.Time) TaskDefinition {
	return TaskDefinition{
		TaskKey:        job.RuntimeKey(),
		JobKey:         job.RuntimeKey(),
		ModuleKey:      job.RuntimeOwner(),
		Title:          job.RuntimeTitle(),
		Description:    job.RuntimeDescription(),
		CronExpression: strings.TrimSpace(job.Schedule),
		Enabled:        job.DefaultEnabled,
		Builtin:        true,
		ConfigJSON:     job.RuntimeDefaultConfig(),
		ConfigSource:   taskConfigSourceSystem,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func mutationToDefinition(command TaskMutation, job JobDefinition, now time.Time) (TaskDefinition, error) {
	configJSON := strings.TrimSpace(command.ConfigJSON)
	if configJSON == "" {
		configJSON = "{}"
	}
	definition := TaskDefinition{
		TaskKey:        strings.TrimSpace(command.TaskKey),
		JobKey:         strings.TrimSpace(command.JobKey),
		ModuleKey:      strings.TrimSpace(job.ModuleKey),
		Title:          strings.TrimSpace(command.Title),
		Description:    strings.TrimSpace(command.Description),
		CronExpression: strings.TrimSpace(command.CronExpression),
		Enabled:        command.Enabled,
		Builtin:        false,
		ConfigJSON:     configJSON,
		ConfigSource:   taskConfigSourceUser,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := validateDefinition(definition); err != nil {
		return TaskDefinition{}, err
	}
	if err := validateEffectiveConfig(job, definition.ConfigJSON); err != nil {
		return TaskDefinition{}, err
	}
	return definition, nil
}

func (r *CronRuntime) validateTaskConfig(ctx context.Context, definition TaskDefinition) error {
	job, err := r.requireKnownJob(ctx, definition.JobKey)
	if err != nil {
		return err
	}
	taskConfig, err := r.taskConfigForEffective(definition)
	if err != nil {
		return err
	}
	return validateEffectiveConfig(job, taskConfig)
}

func validateEffectiveConfig(job JobDefinition, configJSON string) error {
	effectiveConfig, err := effectiveConfigJSON(job.DefaultConfig, configJSON)
	if err != nil {
		return err
	}
	return ValidateConfigJSON(job.ConfigSchema, effectiveConfig)
}

func actionEffectiveConfigJSON(job JobDefinition, taskConfig string, requestConfig string) (string, error) {
	return mergeConfigJSONObjects(job.DefaultConfig, taskConfig, requestConfig)
}

func (r *CronRuntime) resolveActionExecution(ctx context.Context, taskKey string, actionKey string) (actionExecution, error) {
	if r.tasks == nil {
		return actionExecution{}, errors.New("scheduler task repository is unavailable")
	}
	definition, err := r.tasks.GetTask(ctx, taskKey)
	if err != nil {
		return actionExecution{}, err
	}
	if err := validateDefinition(definition); err != nil {
		return actionExecution{}, err
	}
	jobDefinition, err := r.requireKnownJob(ctx, definition.JobKey)
	if err != nil {
		return actionExecution{}, err
	}
	action, ok := findJobAction(jobDefinition.Actions, actionKey)
	if !ok {
		return actionExecution{}, ErrJobActionNotFound
	}
	job, ok := r.findJob(definition.JobKey)
	if !ok {
		return actionExecution{}, ErrJobDefinitionNotFound
	}
	return actionExecution{
		definition:    definition,
		jobDefinition: jobDefinition,
		action:        action,
		job:           job,
	}, nil
}

func jobActionResult(execution actionExecution, result cronx.JobRunResult, effectiveConfig string) JobActionResult {
	return JobActionResult{
		ActionKey:       strings.TrimSpace(execution.action.Key),
		TaskKey:         execution.definition.TaskKey,
		JobKey:          execution.definition.JobKey,
		Result:          result,
		EffectiveConfig: effectiveConfig,
	}
}

func validateJob(job cronx.Job) error {
	if err := job.Validate(); err != nil {
		return err
	}
	if err := validateCronExpression(job.Schedule); err != nil {
		return err
	}
	if !isJSONObject(job.RuntimeDefaultConfig()) {
		return fmt.Errorf("%w: invalid default config", ErrTaskValidation)
	}
	if !isJSONObject(job.RuntimeConfigSchema()) {
		return fmt.Errorf("%w: invalid config schema", ErrTaskValidation)
	}
	for _, action := range job.Actions {
		if strings.TrimSpace(action.Key) == "" {
			return fmt.Errorf("%w: invalid job action key", ErrTaskValidation)
		}
		if action.Handler == nil {
			return fmt.Errorf("%w: job action handler is required", ErrTaskValidation)
		}
	}
	return nil
}

func validateDefinition(definition TaskDefinition) error {
	if strings.TrimSpace(definition.TaskKey) == "" ||
		strings.TrimSpace(definition.JobKey) == "" ||
		strings.TrimSpace(definition.ModuleKey) == "" ||
		strings.TrimSpace(definition.CronExpression) == "" ||
		strings.TrimSpace(definition.Title) == "" {
		return ErrTaskValidation
	}
	if _, reserved := reservedTaskKeys[strings.TrimSpace(definition.TaskKey)]; reserved {
		return fmt.Errorf("%w: reserved task key", ErrTaskValidation)
	}
	if err := validateCronExpression(definition.CronExpression); err != nil {
		return err
	}
	if !isJSONObject(definition.ConfigJSON) {
		return fmt.Errorf("%w: invalid config json", ErrTaskValidation)
	}
	return nil
}

func validateJobDefinition(definition JobDefinition) error {
	if strings.TrimSpace(definition.JobKey) == "" ||
		strings.TrimSpace(definition.ModuleKey) == "" ||
		strings.TrimSpace(definition.Title) == "" ||
		strings.TrimSpace(definition.DefaultCron) == "" {
		return ErrTaskValidation
	}
	if err := validateCronExpression(definition.DefaultCron); err != nil {
		return err
	}
	if !isJSONObject(definition.ConfigSchema) || !isJSONObject(definition.DefaultConfig) {
		return fmt.Errorf("%w: invalid job definition json", ErrTaskValidation)
	}
	if err := ValidateConfigSchema(definition.ConfigSchema); err != nil {
		return err
	}
	if err := ValidateConfigJSON(definition.ConfigSchema, definition.DefaultConfig); err != nil {
		return err
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

func isJSONObject(value string) bool {
	var decoded map[string]any
	return json.Unmarshal([]byte(strings.TrimSpace(value)), &decoded) == nil
}

func sameJSONObject(left string, right string) bool {
	var leftDecoded map[string]any
	var rightDecoded map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(left)), &leftDecoded) != nil {
		return false
	}
	if json.Unmarshal([]byte(strings.TrimSpace(right)), &rightDecoded) != nil {
		return false
	}
	return reflect.DeepEqual(leftDecoded, rightDecoded)
}

func (r *CronRuntime) resolveJobDefaultConfig(ctx context.Context, job cronx.Job) (string, error) {
	key := strings.TrimSpace(job.DefaultConfigKey)
	if key == "" {
		return job.RuntimeDefaultConfig(), nil
	}
	resolver := r.defaultConfigResolver()
	if resolver == nil {
		return job.RuntimeDefaultConfig(), nil
	}
	config, err := resolver.ResolveDefaultConfig(ctx, key)
	if err != nil {
		return "", fmt.Errorf("resolve scheduler job default config %s: %w", key, err)
	}
	config = strings.TrimSpace(config)
	if config == "" {
		config = "{}"
	}
	if !isJSONObject(config) {
		return "", fmt.Errorf("%w: invalid default config", ErrTaskValidation)
	}
	return config, nil
}

func (r *CronRuntime) defaultConfigResolver() DefaultConfigResolver {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultConfigs
}

func (r *CronRuntime) taskConfigForEffective(definition TaskDefinition) (string, error) {
	configJSON := strings.TrimSpace(definition.ConfigJSON)
	if configJSON == "" {
		configJSON = "{}"
	}
	if !definition.Builtin {
		return configJSON, nil
	}
	if definition.ConfigSource != taskConfigSourceUser {
		return "{}", nil
	}
	return configJSON, nil
}

func effectiveConfigJSON(defaultConfig string, taskConfig string) (string, error) {
	return mergeConfigJSONObjects(defaultConfig, taskConfig)
}

func mergeConfigJSONObjects(items ...string) (string, error) {
	merged := make(map[string]any)
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			trimmed = "{}"
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
			return "", fmt.Errorf("%w: invalid config json", ErrTaskValidation)
		}
		for key, value := range decoded {
			merged[key] = value
		}
	}
	encoded, err := json.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("%w: encode effective config", ErrTaskValidation)
	}
	return string(encoded), nil
}

func encodeJobRunResult(result cronx.JobRunResult) (string, string) {
	if result.Summary == "" {
		result.Summary = result.Stage
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "{}", result.Summary
	}
	return string(encoded), result.Summary
}

func jobDefinitionSnapshot(definition JobDefinition) JobDefinitionSnapshot {
	return JobDefinitionSnapshot(definition)
}

func (r *CronRuntime) enrichJobDefinition(ctx context.Context, definition JobDefinition) (JobDefinition, error) {
	if job, ok := r.findJob(definition.JobKey); ok {
		definition.Actions = jobActionsFromJob(job)
		defaultConfig, err := r.resolveJobDefaultConfig(ctx, job)
		if err != nil {
			return JobDefinition{}, err
		}
		definition.DefaultConfig = defaultConfig
	}
	return definition, nil
}

func jobActionsFromJob(job cronx.Job) []JobActionSnapshot {
	actions := make([]JobActionSnapshot, 0, len(job.Actions))
	for _, action := range job.Actions {
		actions = append(actions, JobActionSnapshot{
			Key:            strings.TrimSpace(action.Key),
			TitleKey:       strings.TrimSpace(action.TitleKey),
			Title:          strings.TrimSpace(action.Title),
			DescriptionKey: strings.TrimSpace(action.DescriptionKey),
			Description:    strings.TrimSpace(action.Description),
		})
	}
	return actions
}

func findJobAction(actions []JobActionSnapshot, key string) (JobActionSnapshot, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return JobActionSnapshot{}, false
	}
	for _, action := range actions {
		if strings.TrimSpace(action.Key) == key {
			return action, true
		}
	}
	return JobActionSnapshot{}, false
}

func invokeJobAction(ctx context.Context, job cronx.Job, actionKey string, configJSON string) (cronx.JobRunResult, error) {
	actionKey = strings.TrimSpace(actionKey)
	for _, action := range job.Actions {
		if strings.TrimSpace(action.Key) == actionKey {
			if action.Handler == nil {
				return cronx.JobRunResult{}, fmt.Errorf("%w: job action handler is required", ErrTaskValidation)
			}
			return action.Handler(ctx, configJSON)
		}
	}
	return job.Invoke(ctx, configJSON)
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
		if value == key {
			return append(values[:index], values[index+1:]...)
		}
	}
	return values
}
