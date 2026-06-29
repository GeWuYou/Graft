package scheduler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"graft/server/internal/cronx"
)

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
	if err := ctx.Err(); err != nil {
		return err
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
		ID:             definition.ID,
		Key:            definition.TaskKey,
		JobKey:         definition.JobKey,
		TitleKey:       definition.TitleKey,
		Title:          definition.Title,
		DescriptionKey: definition.DescriptionKey,
		Description:    definition.Description,
		Schedule:       definition.CronExpression,
		Enabled:        definition.Enabled,
		Builtin:        definition.Builtin,
		ConfigJSON:     definition.ConfigJSON,
		ConfigSource:   definition.ConfigSource,
		CreatedAt:      definition.CreatedAt,
		UpdatedAt:      definition.UpdatedAt,
		DeletedAt:      definition.DeletedAt,
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
		jobSnapshot := jobDefinitionSnapshot(jobDefinition)
		snapshot.JobDefinition = &jobSnapshot
	}
	r.mu.RLock()
	_, snapshot.Running = r.running[definition.TaskKey]
	snapshot.NextRunAt = r.nextRunAtLocked(definition.TaskKey)
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

func (r *CronRuntime) nextRunAtLocked(key string) *time.Time {
	entryID, ok := r.entries[key]
	if !ok {
		return nil
	}
	next := r.cron.Entry(entryID).Next
	if next.IsZero() {
		return nil
	}
	return &next
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
	jobDefinition, err := r.requireKnownJob(ctx, definition.JobKey)
	if err != nil {
		return TaskRun{}, err
	}
	return r.runs.CreateRun(ctx, TaskRun{
		TaskKey:          definition.TaskKey,
		JobKey:           definition.JobKey,
		TaskTitle:        definition.Title,
		TaskTitleKey:     definition.TitleKey,
		JobTitle:         jobDefinition.Title,
		JobTitleKey:      jobDefinition.TitleKey,
		JobShortTitle:    jobDefinition.ShortTitle,
		JobShortTitleKey: jobDefinition.ShortTitleKey,
		JobCategory:      jobDefinition.Category,
		ModuleKey:        jobDefinition.ModuleKey,
		TaskBuiltin:      definition.Builtin,
		TriggerType:      trigger,
		Status:           RunStatusRunning,
		StartedAt:        startedAt,
		CreatedAt:        startedAt,
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
	trigger.Type = TriggerTypeManual
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
