package scheduler

import (
	"context"
	"errors"

	"graft/server/internal/cronx"
)

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
	if err := r.ensureTaskKeyAvailable(ctx, definition.TaskKey); err != nil {
		return TaskSnapshot{}, err
	}
	if err := r.ensureTaskTitleAvailable(ctx, definition.Title, definition.TaskKey); err != nil {
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
	if err := r.ensureTaskTitleAvailable(ctx, next.Title, key); err != nil {
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

func (r *CronRuntime) ensureTaskKeyAvailable(ctx context.Context, key string) error {
	_, err := r.tasks.GetTask(ctx, key)
	switch {
	case err == nil:
		return ErrTaskKeyConflict
	case errors.Is(err, ErrTaskNotFound):
		return nil
	default:
		return err
	}
}

func (r *CronRuntime) ensureTaskTitleAvailable(ctx context.Context, title string, currentKey string) error {
	existing, err := r.tasks.GetTaskByTitle(ctx, title)
	switch {
	case err == nil && existing.TaskKey != currentKey:
		return ErrTaskTitleConflict
	case err == nil:
		return nil
	case errors.Is(err, ErrTaskNotFound):
		return nil
	default:
		return err
	}
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
