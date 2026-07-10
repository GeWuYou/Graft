package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
	taskstore "graft/server/modules/task/store"
)

const (
	defaultWorkerCount   = 2
	defaultPollInterval  = 250 * time.Millisecond
	errorCodeExecutor    = "stage_executor_failed"
	errorCodeCancelled   = "cancelled"
	errorCodeMissingExec = "stage_executor_unavailable"
)

// Runtime owns Task submission, serial Stage dispatch, and process-local worker
// lifecycle. PostgreSQL remains the durable authority for every state change.
type Runtime struct {
	repository taskstore.Repository
	workers    int
	pollEvery  time.Duration

	mu        sync.RWMutex
	eventMu   sync.Mutex
	executors map[moduleapi.StageExecutorType]moduleapi.StageExecutor
	running   map[uint64]runningStage
	cancel    context.CancelFunc
	wake      chan struct{}
	waitGroup sync.WaitGroup
}

type runningStage struct {
	executor moduleapi.StageExecutor
	run      moduleapi.StageRun
	cancel   context.CancelFunc
}

// NewRuntime creates a Task Runtime with a bounded in-process worker pool.
func NewRuntime(repository taskstore.Repository) *Runtime {
	return &Runtime{
		repository: repository,
		workers:    defaultWorkerCount,
		pollEvery:  defaultPollInterval,
		executors:  make(map[moduleapi.StageExecutorType]moduleapi.StageExecutor),
		running:    make(map[uint64]runningStage),
		wake:       make(chan struct{}, 1),
	}
}

// RegisterStageExecutor installs one consumer-owned Stage executor before Boot.
func (r *Runtime) RegisterStageExecutor(executor moduleapi.StageExecutor) error {
	if r == nil || executor == nil || executor.Type() == "" {
		return errors.New("task stage executor is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.executors[executor.Type()]; exists {
		return fmt.Errorf("task stage executor %q is already registered", executor.Type())
	}
	r.executors[executor.Type()] = executor
	return nil
}

// RegisterTaskOwnerAuthorizer is reserved for generic Task HTTP APIs. It is a
// no-op in the runtime-only batch, where no generic HTTP route exists yet.
func (r *Runtime) RegisterTaskOwnerAuthorizer(authorizer moduleapi.TaskOwnerAuthorizer) error {
	if r == nil || authorizer == nil || authorizer.OwnerType() == "" {
		return errors.New("task owner authorizer is required")
	}
	return nil
}

// Submit validates consumer-owned executor references before atomically storing
// the immutable TaskPlan. The dispatcher observes it through PostgreSQL.
func (r *Runtime) Submit(ctx context.Context, input moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	if r == nil || r.repository == nil {
		return moduleapi.TaskReceipt{}, errors.New("task runtime repository is unavailable")
	}
	if err := r.validatePlan(input.Plan); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	plan, err := json.Marshal(input.Plan)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal task plan: %w", err)
	}
	status := moduleapi.TaskStatusPending
	if input.ScheduledAt != nil && input.ScheduledAt.After(time.Now().UTC()) {
		status = moduleapi.TaskStatusScheduled
	}
	task := taskmodel.Task{
		Type: input.Type, Owner: input.Owner, Status: status, Input: input.Input,
		Metadata: input.Metadata, Plan: plan, State: json.RawMessage(`{}`),
		CreatedBy: nullableRequestedBy(input.RequestedBy), ScheduledAt: input.ScheduledAt,
	}
	stages := make([]taskmodel.Stage, 0, len(input.Plan.Stages))
	for index, stage := range input.Plan.Stages {
		stages = append(stages, taskmodel.Stage{
			Key: stage.Key, Sequence: index + 1, ExecutorType: stage.ExecutorType,
			Status: moduleapi.StageStatusPending, MaxAttempts: normalizedMaxAttempts(stage.RetryPolicy.MaxAttempts),
			RetryBackoffMS: stage.RetryPolicy.Backoff.Milliseconds(), Input: stage.Input,
			RecoveryPolicy: stage.RecoveryPolicy, Result: json.RawMessage(`{}`),
		})
	}
	created, _, err := r.repository.Create(ctx, taskstore.CreateInput{Task: task, Stages: stages})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("create task: %w", err)
	}
	r.signalWake()
	return moduleapi.TaskReceipt{TaskID: created.ID, Status: created.Status}, nil
}

// Cancel persists a cancellation request and forwards a cancellation signal to a
// currently running consumer executor when one owns the active Stage.
func (r *Runtime) Cancel(ctx context.Context, taskID uint64) error {
	if r == nil || r.repository == nil {
		return errors.New("task runtime repository is unavailable")
	}
	task, err := r.repository.RequestCancellation(ctx, taskID, time.Now().UTC())
	if err != nil {
		return err
	}
	if task.Status != moduleapi.TaskStatusRunning {
		return r.cancelNonRunningTask(ctx, task)
	}
	return r.cancelRunningTask(ctx, taskID)
}

func (r *Runtime) cancelNonRunningTask(ctx context.Context, task taskmodel.Task) error {
	now := time.Now().UTC()
	switch task.Status {
	case moduleapi.TaskStatusPending, moduleapi.TaskStatusScheduled:
		if err := r.repository.CancelPendingTask(ctx, task.ID, now, durationSince(task.StartedAt, now)); err != nil {
			return err
		}
	case moduleapi.TaskStatusNeedsAttention:
		if err := r.repository.TransitionTask(ctx, taskstore.TaskTransitionInput{TaskID: task.ID, From: moduleapi.TaskStatusNeedsAttention, To: moduleapi.TaskStatusCancelled, FinishedAt: &now, DurationMS: durationSince(task.StartedAt, now)}); err != nil {
			return err
		}
	default:
		return taskstore.ErrStateConflict
	}
	return r.appendEvent(ctx, task.ID, taskmodel.EventTypeCancelled)
}

func (r *Runtime) cancelRunningTask(ctx context.Context, taskID uint64) error {
	r.mu.RLock()
	running, exists := r.running[taskID]
	r.mu.RUnlock()
	if exists {
		if err := running.executor.Cancel(ctx, running.run); err != nil {
			return fmt.Errorf("cancel task stage: %w", err)
		}
		running.cancel()
	}
	r.signalWake()
	return r.appendEvent(ctx, taskID, taskmodel.EventTypeCancelRequested)
}

// RetryStage records an operator-approved retry for an unknown or failed Stage.
func (r *Runtime) RetryStage(ctx context.Context, taskID uint64, stageID uint64) error {
	if r == nil || r.repository == nil {
		return errors.New("task runtime repository is unavailable")
	}
	if _, err := r.repository.RetryStage(ctx, taskID, stageID, time.Now().UTC()); err != nil {
		return err
	}
	if err := r.appendEvent(ctx, taskID, taskmodel.EventTypeRetryRequested); err != nil {
		return err
	}
	r.signalWake()
	return nil
}

// Start performs crash recovery before starting the bounded worker pool.
func (r *Runtime) Start(ctx context.Context) error {
	if r == nil || r.repository == nil || ctx == nil {
		return errors.New("task runtime start dependencies are unavailable")
	}
	if _, err := r.repository.RecoverInterruptedStages(ctx, time.Now().UTC()); err != nil {
		return err
	}
	r.mu.Lock()
	if r.cancel != nil {
		r.mu.Unlock()
		return errors.New("task runtime is already started")
	}
	workerContext, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	workers := r.workers
	r.mu.Unlock()
	for index := 0; index < workers; index++ {
		r.waitGroup.Add(1)
		go r.worker(workerContext)
	}
	r.signalWake()
	return nil
}

// Stop asks workers and running executors to stop, then waits for their owned
// goroutines. In-flight external work remains conservatively recoverable as unknown.
func (r *Runtime) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	cancel := r.cancel
	r.cancel = nil
	running := make([]runningStage, 0, len(r.running))
	for _, current := range r.running {
		running = append(running, current)
	}
	r.mu.Unlock()
	if cancel == nil {
		return nil
	}
	for _, current := range running {
		_ = current.executor.Cancel(ctx, current.run)
		current.cancel()
	}
	cancel()
	done := make(chan struct{})
	go func() { r.waitGroup.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runtime) worker(ctx context.Context) {
	defer r.waitGroup.Done()
	ticker := time.NewTicker(r.pollEvery)
	defer ticker.Stop()
	for {
		if err := r.runOne(ctx); err != nil && !errors.Is(err, context.Canceled) {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-r.wake:
		case <-ticker.C:
		}
	}
}

func (r *Runtime) runOne(ctx context.Context) error {
	claim, found, err := r.repository.ClaimNextStage(ctx, time.Now().UTC())
	if err != nil || !found {
		return err
	}
	executor, ok := r.executorFor(claim.Stage.ExecutorType)
	if !ok {
		return r.failClaim(ctx, claim, errorCodeMissingExec, "no executor registered for stage")
	}
	stageContext, cancel := context.WithCancel(ctx)
	run := &stageRun{runtime: r, task: claim.Task, stage: claim.Stage}
	r.addRunning(claim.Task.ID, runningStage{executor: executor, run: run, cancel: cancel})
	err = executor.Execute(stageContext, run)
	cancel()
	r.removeRunning(claim.Task.ID)
	return r.finishClaim(ctx, claim, err)
}

func (r *Runtime) finishClaim(ctx context.Context, claim taskstore.StageClaim, executeErr error) error {
	if task, err := r.repository.Get(ctx, claim.Task.ID); err == nil && task.CancelRequestedAt != nil {
		return r.cancelClaim(ctx, claim)
	}
	if executeErr == nil {
		return r.completeClaim(ctx, claim)
	}
	return r.failClaim(ctx, claim, errorCodeExecutor, executeErr.Error())
}

func (r *Runtime) completeClaim(ctx context.Context, claim taskstore.StageClaim) error {
	now := time.Now().UTC()
	if err := r.repository.TransitionStage(ctx, taskstore.StageTransitionInput{StageID: claim.Stage.ID, From: moduleapi.StageStatusRunning, To: moduleapi.StageStatusSuccess, Attempt: claim.Stage.Attempt, FinishedAt: &now, DurationMS: durationSince(claim.Stage.StartedAt, now)}); err != nil {
		return err
	}
	stages, err := r.repository.ListStages(ctx, claim.Task.ID)
	if err != nil {
		return err
	}
	if hasPendingStages(stages) {
		r.signalWake()
		return nil
	}
	return r.repository.TransitionTask(ctx, taskstore.TaskTransitionInput{TaskID: claim.Task.ID, From: moduleapi.TaskStatusRunning, To: moduleapi.TaskStatusSuccess, FinishedAt: &now, DurationMS: durationSince(claim.Task.StartedAt, now)})
}

func (r *Runtime) failClaim(ctx context.Context, claim taskstore.StageClaim, code string, message string) error {
	now := time.Now().UTC()
	if claim.Stage.Attempt < claim.Stage.MaxAttempts && claim.Stage.RecoveryPolicy == moduleapi.StageRecoveryRetryIfIdempotent {
		nextRetry := now.Add(time.Duration(claim.Stage.RetryBackoffMS) * time.Millisecond)
		if err := r.repository.TransitionStage(ctx, taskstore.StageTransitionInput{StageID: claim.Stage.ID, From: moduleapi.StageStatusRunning, To: moduleapi.StageStatusFailed, Attempt: claim.Stage.Attempt, NextRetryAt: &nextRetry, FailureCode: &code, FailureMessage: &message, FinishedAt: &now, DurationMS: durationSince(claim.Stage.StartedAt, now)}); err != nil {
			return err
		}
		if err := r.repository.RescheduleStage(ctx, claim.Stage.ID, nextRetry); err != nil {
			return err
		}
		if err := r.appendEvent(ctx, claim.Task.ID, taskmodel.EventTypeRetryScheduled); err != nil {
			return err
		}
		r.signalWake()
		return nil
	}
	if err := r.repository.TransitionStage(ctx, taskstore.StageTransitionInput{StageID: claim.Stage.ID, From: moduleapi.StageStatusRunning, To: moduleapi.StageStatusFailed, Attempt: claim.Stage.Attempt, FailureCode: &code, FailureMessage: &message, FinishedAt: &now, DurationMS: durationSince(claim.Stage.StartedAt, now)}); err != nil {
		return err
	}
	return r.repository.TransitionTask(ctx, taskstore.TaskTransitionInput{TaskID: claim.Task.ID, From: moduleapi.TaskStatusRunning, To: moduleapi.TaskStatusFailed, FailureCode: &code, FailureMessage: &message, FinishedAt: &now, DurationMS: durationSince(claim.Task.StartedAt, now)})
}

func (r *Runtime) cancelClaim(ctx context.Context, claim taskstore.StageClaim) error {
	now := time.Now().UTC()
	if err := r.repository.TransitionStage(ctx, taskstore.StageTransitionInput{StageID: claim.Stage.ID, From: moduleapi.StageStatusRunning, To: moduleapi.StageStatusCancelled, Attempt: claim.Stage.Attempt, FinishedAt: &now, DurationMS: durationSince(claim.Stage.StartedAt, now)}); err != nil {
		return err
	}
	if err := r.repository.TransitionTask(ctx, taskstore.TaskTransitionInput{TaskID: claim.Task.ID, From: moduleapi.TaskStatusRunning, To: moduleapi.TaskStatusCancelled, FinishedAt: &now, DurationMS: durationSince(claim.Task.StartedAt, now)}); err != nil {
		return err
	}
	return r.appendEvent(ctx, claim.Task.ID, taskmodel.EventTypeCancelled)
}

func (r *Runtime) validatePlan(plan moduleapi.TaskPlan) error {
	if len(plan.Stages) == 0 {
		return errors.New("task plan must contain at least one stage")
	}
	seen := make(map[string]struct{}, len(plan.Stages))
	for _, stage := range plan.Stages {
		if stage.Key == "" || stage.ExecutorType == "" || stage.RecoveryPolicy == "" {
			return errors.New("task plan stage is incomplete")
		}
		if _, exists := seen[stage.Key]; exists {
			return fmt.Errorf("task plan contains duplicate stage %q", stage.Key)
		}
		if _, exists := r.executorFor(stage.ExecutorType); !exists {
			return fmt.Errorf("task plan references unregistered stage executor %q", stage.ExecutorType)
		}
		seen[stage.Key] = struct{}{}
	}
	return nil
}

func (r *Runtime) executorFor(executorType moduleapi.StageExecutorType) (moduleapi.StageExecutor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	executor, exists := r.executors[executorType]
	return executor, exists
}

func (r *Runtime) addRunning(taskID uint64, stage runningStage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running[taskID] = stage
}

func (r *Runtime) removeRunning(taskID uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.running, taskID)
}

func (r *Runtime) appendEvent(ctx context.Context, taskID uint64, eventType taskmodel.EventType) error {
	r.eventMu.Lock()
	defer r.eventMu.Unlock()
	sequence, err := r.repository.NextEventSequence(ctx, taskID)
	if err != nil {
		return err
	}
	_, err = r.repository.AppendEvent(ctx, taskstore.AppendEventInput{TaskID: taskID, Sequence: sequence, Type: eventType})
	return err
}

func (r *Runtime) signalWake() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func nullableRequestedBy(value uint64) *uint64 {
	if value == 0 {
		return nil
	}
	return &value
}

func normalizedMaxAttempts(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}

func hasPendingStages(stages []taskmodel.Stage) bool {
	for _, stage := range stages {
		if stage.Status == moduleapi.StageStatusPending || stage.Status == moduleapi.StageStatusRunning {
			return true
		}
	}
	return false
}

func durationSince(startedAt *time.Time, finishedAt time.Time) *int64 {
	if startedAt == nil {
		return nil
	}
	duration := finishedAt.Sub(*startedAt).Milliseconds()
	if duration < 0 {
		duration = 0
	}
	return &duration
}

type stageRun struct {
	runtime *Runtime
	task    taskmodel.Task
	stage   taskmodel.Stage
	logMu   sync.Mutex
}

func (r *stageRun) TaskID() uint64 { return r.task.ID }

func (r *stageRun) StageID() uint64 { return r.stage.ID }

func (r *stageRun) Attempt() int { return r.stage.Attempt }

func (r *stageRun) Input() json.RawMessage { return append(json.RawMessage(nil), r.stage.Input...) }

func (r *stageRun) CancellationRequested(ctx context.Context) bool {
	task, err := r.runtime.repository.Get(ctx, r.task.ID)
	return err == nil && task.CancelRequestedAt != nil
}

func (r *stageRun) AppendLog(ctx context.Context, entry moduleapi.TaskLogEntry) error {
	r.logMu.Lock()
	defer r.logMu.Unlock()
	sequence, err := r.runtime.repository.NextLogSequence(ctx, r.task.ID)
	if err != nil {
		return err
	}
	stageID := r.stage.ID
	_, err = r.runtime.repository.AppendLog(ctx, taskstore.AppendLogInput{TaskID: r.task.ID, StageID: &stageID, Sequence: sequence, Stream: entry.Stream, Level: entry.Level, Line: entry.Line, OccurredAt: time.Now().UTC()})
	return err
}
