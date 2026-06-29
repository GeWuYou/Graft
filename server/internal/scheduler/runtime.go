package scheduler

import (
	"context"
	"errors"
	"fmt"
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
	// ErrTaskKeyConflict is returned when a scheduled task key is already in use.
	ErrTaskKeyConflict = errors.New("scheduler task key already exists")
	// ErrTaskTitleConflict is returned when a scheduled task title is already in use.
	ErrTaskTitleConflict = errors.New("scheduler task title already exists")
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
	Category       cronx.JobCategory
	TitleKey       string
	Title          string
	ShortTitleKey  string
	ShortTitle     string
	DescriptionKey string
	Description    string
	ConfigSchema   string
	DefaultConfig  string
	DefaultCron    string
	DefaultEnabled bool
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
	ID              uint64
	Key             string
	JobKey          string
	TitleKey        string
	Title           string
	DescriptionKey  string
	Description     string
	Schedule        string
	Enabled         bool
	Builtin         bool
	ConfigJSON      string
	ConfigSource    string
	EffectiveConfig string
	JobDefinition   *JobDefinitionSnapshot
	Running         bool
	LastRun         *TaskRun
	NextRunAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

// TaskRun is the persisted run-history model for scheduler runtime jobs.
type TaskRun struct {
	ID               uint64
	TaskKey          string
	JobKey           string
	TaskTitle        string
	TaskTitleKey     string
	JobTitle         string
	JobTitleKey      string
	JobShortTitle    string
	JobShortTitleKey string
	JobCategory      cronx.JobCategory
	ModuleKey        string
	TaskBuiltin      bool
	TriggerType      TriggerType
	Status           RunStatus
	ErrorMessage     string
	Result           string
	ResultJSON       string
	EffectiveConfig  string
	StartedAt        time.Time
	FinishedAt       *time.Time
	DurationMS       *int64
	CreatedAt        time.Time
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
	Category       cronx.JobCategory
	TitleKey       string
	Title          string
	ShortTitleKey  string
	ShortTitle     string
	DescriptionKey string
	Description    string
	ConfigSchema   string
	DefaultConfig  string
	DefaultCron    string
	DefaultEnabled bool
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
	TitleKey       string
	Title          string
	DescriptionKey string
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
	ReplaceTask(ctx context.Context, task TaskDefinition) (TaskDefinition, error)
	UpdateTask(ctx context.Context, key string, patch TaskMutation) (TaskDefinition, error)
	DeleteTask(ctx context.Context, key string) error
	SetTaskEnabled(ctx context.Context, key string, enabled bool) (TaskDefinition, error)
	ListTasks(ctx context.Context, query TaskListQuery) ([]TaskDefinition, int, error)
	GetTask(ctx context.Context, key string) (TaskDefinition, error)
	GetTaskByTitle(ctx context.Context, title string) (TaskDefinition, error)
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
	addSchedule     func(key string, schedule string, run func(context.Context) (TaskRun, error)) (cron.EntryID, error)
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
