package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"graft/server/internal/logger"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
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

// Runtime 拥有 Task 提交、阶段串行分发和进程内 worker 生命周期；每次状态变化仍以 PostgreSQL 持久化事实为权威。
type Runtime struct {
	repository taskstore.Repository
	workers    int
	pollEvery  time.Duration

	mu              sync.RWMutex
	eventMu         sync.Mutex
	executors       map[moduleapi.StageExecutorType]moduleapi.StageExecutor
	authorizers     map[string]moduleapi.TaskOwnerAuthorizer
	running         map[uint64]runningStage
	cancel          context.CancelFunc
	wake            chan struct{}
	waitGroup       sync.WaitGroup
	realtimeTickets realtimeauth.Service
	realtimeHub     realtime.Hub
	topicIssuers    realtime.TopicIssuerRegistry
	logger          logger.AppLogger
}

type runningStage struct {
	executor moduleapi.StageExecutor
	run      moduleapi.StageRun
	cancel   context.CancelFunc
}

// NewRuntime 创建任务运行时，并初始化执行器、owner 授权器和 worker 唤醒通道。
func NewRuntime(repository taskstore.Repository) *Runtime {
	return &Runtime{
		repository:  repository,
		workers:     defaultWorkerCount,
		pollEvery:   defaultPollInterval,
		executors:   make(map[moduleapi.StageExecutorType]moduleapi.StageExecutor),
		authorizers: make(map[string]moduleapi.TaskOwnerAuthorizer),
		running:     make(map[uint64]runningStage),
		wake:        make(chan struct{}, 1),
		logger:      logger.NewAppLogger(nil),
	}
}

// SetAppLogger 绑定模块上下文注入的 AppLogger，使异步 Task 失败沿用同一关联与持久化策略。
func (r *Runtime) SetAppLogger(appLogger logger.AppLogger) {
	if r == nil || appLogger == nil {
		return
	}
	r.mu.Lock()
	r.logger = appLogger.Named("modules.task.runtime")
	r.mu.Unlock()
}

// RegisterStageExecutor 在 Boot 前注册一个消费模块所有的阶段执行器；启动后注册会被拒绝，避免 worker 看到不完整执行器集合。
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

// AuthorizeOwner 将资源授权委托给拥有该 Task 业务资源的消费模块，Task Runtime 不自行推断业务权限。
func (r *Runtime) AuthorizeOwner(ctx context.Context, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) error {
	r.mu.RLock()
	authorizer := r.authorizers[owner.Type]
	r.mu.RUnlock()
	if authorizer == nil {
		return errors.New("task owner authorizer is unavailable")
	}
	return authorizer.AuthorizeTaskOwner(ctx, actor, action, owner)
}

// RegisterTaskOwnerAuthorizer 注册消费模块所有的 Task 资源授权器，供通用任务接口在读取或操作前校验 owner 边界。
func (r *Runtime) RegisterTaskOwnerAuthorizer(authorizer moduleapi.TaskOwnerAuthorizer) error {
	if r == nil || authorizer == nil || authorizer.OwnerType() == "" {
		return errors.New("task owner authorizer is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.authorizers[authorizer.OwnerType()]; exists {
		return fmt.Errorf("task owner authorizer %q is already registered", authorizer.OwnerType())
	}
	r.authorizers[authorizer.OwnerType()] = authorizer
	return nil
}

// Submit 校验执行器引用并原子保存不可变 TaskPlan；成功 receipt 只证明 PostgreSQL 已提交，不代表任务已执行完成。
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
	r.publishTask(created.ID, "task.created")
	return moduleapi.TaskReceipt{TaskID: created.ID, Status: created.Status}, nil
}

// GetTask 返回一个已持久化的 Task 读模型；返回值来自仓储事实而不是 worker 内存状态。
func (r *Runtime) GetTask(ctx context.Context, taskID uint64) (moduleapi.TaskView, error) {
	task, err := r.repository.Get(ctx, taskID)
	if err != nil {
		return moduleapi.TaskView{}, err
	}
	return toTaskView(task), nil
}

// ListTasks 返回经调用方完成 owner 授权后的 Task 历史分页及总数。
func (r *Runtime) ListTasks(ctx context.Context, filter moduleapi.TaskListFilter, limit int, offset int) ([]moduleapi.TaskView, int64, error) {
	if r == nil || r.repository == nil {
		return nil, 0, taskstore.ErrInvalidInput
	}
	tasks, total, err := r.repository.List(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	items := make([]moduleapi.TaskView, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, toTaskView(task))
	}
	return items, total, nil
}

// ListTaskLogs 返回按持久化序列游标读取的 Task 日志重放分页。
func (r *Runtime) ListTaskLogs(ctx context.Context, taskID uint64, after int64, limit int) ([]moduleapi.TaskLogView, error) {
	logs, err := r.repository.ListLogs(ctx, taskID, after, limit)
	if err != nil {
		return nil, err
	}
	items := make([]moduleapi.TaskLogView, 0, len(logs))
	for _, log := range logs {
		items = append(items, moduleapi.TaskLogView{ID: log.ID, TaskID: log.TaskID, StageID: log.StageID, Sequence: log.Sequence, Stream: log.Stream, Level: log.Level, Line: log.Line, OccurredAt: log.OccurredAt})
	}
	return items, nil
}

// ListTaskStages 返回按阶段序号排序的持久化 Stage 时间线。
func (r *Runtime) ListTaskStages(ctx context.Context, taskID uint64) ([]moduleapi.TaskStageView, error) {
	stages, err := r.repository.ListStages(ctx, taskID)
	if err != nil {
		return nil, err
	}
	items := make([]moduleapi.TaskStageView, 0, len(stages))
	for _, stage := range stages {
		items = append(items, moduleapi.TaskStageView{ID: stage.ID, Key: stage.Key, Sequence: stage.Sequence, ExecutorType: stage.ExecutorType, Status: stage.Status, Attempt: stage.Attempt, MaxAttempts: stage.MaxAttempts, RecoveryPolicy: stage.RecoveryPolicy, StartedAt: stage.StartedAt, FinishedAt: stage.FinishedAt, DurationMS: stage.DurationMS, FailureCode: stage.FailureCode, FailureMessage: stage.FailureMessage})
	}
	return items, nil
}

// ListTaskEvents 返回不能从当前状态推导的持久化 Task 历史事实。
func (r *Runtime) ListTaskEvents(ctx context.Context, taskID uint64, after int64, limit int) ([]moduleapi.TaskEventView, error) {
	events, err := r.repository.ListEvents(ctx, taskID, after, limit)
	if err != nil {
		return nil, err
	}
	items := make([]moduleapi.TaskEventView, 0, len(events))
	for _, event := range events {
		items = append(items, moduleapi.TaskEventView{ID: event.ID, Sequence: event.Sequence, Type: string(event.Type), Payload: event.Payload, CreatedAt: event.CreatedAt})
	}
	return items, nil
}

// toTaskView 将持久化的任务模型转换为对外任务视图。
func toTaskView(task taskmodel.Task) moduleapi.TaskView {
	return moduleapi.TaskView{ID: task.ID, Type: task.Type, Owner: task.Owner, Status: task.Status, CurrentStageKey: task.CurrentStageKey, CreatedBy: task.CreatedBy, CreatedAt: task.CreatedAt, StartedAt: task.StartedAt, FinishedAt: task.FinishedAt, DurationMS: task.DurationMS, FailureCode: task.FailureCode, FailureMessage: task.FailureMessage}
}

// Cancel 持久化取消请求；若当前 Stage 正由消费模块执行，则继续转发取消信号，running 状态由执行结果决定。
func (r *Runtime) Cancel(ctx context.Context, taskID uint64) error {
	if r == nil || r.repository == nil {
		return errors.New("task runtime repository is unavailable")
	}
	task, err := r.repository.RequestCancellation(ctx, taskID, time.Now().UTC())
	if err != nil {
		return err
	}
	if task.Status != moduleapi.TaskStatusRunning {
		err := r.cancelNonRunningTask(ctx, task)
		if err == nil {
			r.publishTask(taskID, "task.cancelled")
		}
		return err
	}
	err = r.cancelRunningTask(ctx, taskID)
	if err == nil {
		r.publishTask(taskID, "task.cancel_requested")
	}
	return err
}

func (r *Runtime) cancelNonRunningTask(ctx context.Context, task taskmodel.Task) error {
	now := time.Now().UTC()
	switch task.Status {
	case moduleapi.TaskStatusPending, moduleapi.TaskStatusScheduled:
		if err := r.repository.CancelPendingTask(ctx, task.ID, now, durationSince(task.StartedAt, now)); err != nil {
			return err
		}
	case moduleapi.TaskStatusNeedsAttention:
		if err := r.repository.TransitionTask(ctx, taskstore.TaskTransitionInput{TaskID: task.ID, From: moduleapi.TaskStatusNeedsAttention, To: moduleapi.TaskStatusCancelled, CurrentStageKey: task.CurrentStageKey, FinishedAt: &now, DurationMS: durationSince(task.StartedAt, now)}); err != nil {
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
		if err := r.cancelStage(ctx, running.executor, running.run); err != nil {
			return fmt.Errorf("cancel task stage: %w", err)
		}
		running.cancel()
	}
	r.signalWake()
	return r.appendEvent(ctx, taskID, taskmodel.EventTypeCancelRequested)
}

func (r *Runtime) executeStage(ctx context.Context, executor moduleapi.StageExecutor, run moduleapi.StageRun) (err error) {
	return r.invokeStage(ctx, executor, run, "task_stage_execute", "stage executor panicked", func() error {
		return executor.Execute(ctx, run)
	})
}

func (r *Runtime) cancelStage(ctx context.Context, executor moduleapi.StageExecutor, run moduleapi.StageRun) (err error) {
	return r.invokeStage(ctx, executor, run, "task_stage_cancel", "stage executor cancellation panicked", func() error {
		return executor.Cancel(ctx, run)
	})
}

func (r *Runtime) invokeStage(
	ctx context.Context,
	executor moduleapi.StageExecutor,
	run moduleapi.StageRun,
	operation string,
	panicMessage string,
	invoke func() error,
) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%s: %v", panicMessage, recovered)
			r.logger.Error(ctx, panicMessage,
				logger.StringField(logger.FieldOperation, operation),
				logger.Uint64Field("task_id", run.TaskID()),
				logger.Uint64Field("stage_id", run.StageID()),
				logger.StringField("executor_type", string(executor.Type())),
				logger.StringField("stacktrace", string(debug.Stack())),
			)
		}
	}()
	return invoke()
}

// RetryStage 记录操作员批准的 unknown 或 failed 阶段重试；只有父 Task 处于可恢复状态时仓储才允许回到 pending。
func (r *Runtime) RetryStage(ctx context.Context, taskID uint64, stageID uint64) error {
	if r == nil || r.repository == nil {
		return errors.New("task runtime repository is unavailable")
	}
	recoveryResolved, err := r.stageNeedsRecoveryResolution(ctx, taskID, stageID)
	if err != nil {
		return err
	}
	if _, err := r.repository.RetryStage(ctx, taskID, stageID, time.Now().UTC()); err != nil {
		return err
	}
	if recoveryResolved {
		if err := r.appendEvent(ctx, taskID, taskmodel.EventTypeRecoveryResolved); err != nil {
			return err
		}
	}
	if err := r.appendEvent(ctx, taskID, taskmodel.EventTypeRetryRequested); err != nil {
		return err
	}
	r.signalWake()
	r.publishTask(taskID, "task.retry_requested")
	return nil
}

func (r *Runtime) stageNeedsRecoveryResolution(ctx context.Context, taskID uint64, stageID uint64) (bool, error) {
	stages, err := r.repository.ListStages(ctx, taskID)
	if err != nil {
		return false, err
	}
	for _, stage := range stages {
		if stage.ID == stageID {
			return stage.Status == moduleapi.StageStatusUnknown, nil
		}
	}
	return false, taskstore.ErrNotFound
}

// Start 在启动有界 worker 池前执行崩溃恢复；无法证明外部副作用完成的在途阶段会先进入 unknown。
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

// Stop 请求 worker 和运行中执行器停止并等待其 goroutine；无法确认完成的外部工作留待恢复为 unknown。
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
		_ = r.cancelStage(ctx, current.executor, current.run)
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
	defer func() {
		if recovered := recover(); recovered != nil {
			r.logger.Error(ctx, "task worker panicked",
				logger.StringField(logger.FieldOperation, "task_worker"),
				logger.StringField("stacktrace", string(debug.Stack())),
			)
		}
	}()
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
	r.publishTask(claim.Task.ID, "task.stage_started")
	executor, ok := r.executorFor(claim.Stage.ExecutorType)
	if !ok {
		return r.failClaim(ctx, claim, errorCodeMissingExec, "no executor registered for stage")
	}
	stageContext, cancel := context.WithCancel(ctx)
	run := &stageRun{runtime: r, task: claim.Task, stage: claim.Stage}
	r.addRunning(claim.Task.ID, runningStage{executor: executor, run: run, cancel: cancel})
	err = r.executeStage(stageContext, executor, run)
	cancel()
	r.removeRunning(claim.Task.ID)
	finishErr := r.finishClaim(ctx, claim, err)
	if finishErr == nil {
		eventType := "task.stage_completed"
		if err != nil {
			eventType = "task.stage_failed"
		}
		r.publishTask(claim.Task.ID, eventType)
	}
	return finishErr
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
	return r.repository.TransitionTask(ctx, taskstore.TaskTransitionInput{TaskID: claim.Task.ID, From: moduleapi.TaskStatusRunning, To: moduleapi.TaskStatusSuccess, CurrentStageKey: &claim.Stage.Key, FinishedAt: &now, DurationMS: durationSince(claim.Task.StartedAt, now)})
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
	return r.repository.TransitionTask(ctx, taskstore.TaskTransitionInput{TaskID: claim.Task.ID, From: moduleapi.TaskStatusRunning, To: moduleapi.TaskStatusFailed, CurrentStageKey: &claim.Stage.Key, FailureCode: &code, FailureMessage: &message, FinishedAt: &now, DurationMS: durationSince(claim.Task.StartedAt, now)})
}

func (r *Runtime) cancelClaim(ctx context.Context, claim taskstore.StageClaim) error {
	now := time.Now().UTC()
	if err := r.repository.TransitionStage(ctx, taskstore.StageTransitionInput{StageID: claim.Stage.ID, From: moduleapi.StageStatusRunning, To: moduleapi.StageStatusCancelled, Attempt: claim.Stage.Attempt, FinishedAt: &now, DurationMS: durationSince(claim.Stage.StartedAt, now)}); err != nil {
		return err
	}
	if err := r.repository.TransitionTask(ctx, taskstore.TaskTransitionInput{TaskID: claim.Task.ID, From: moduleapi.TaskStatusRunning, To: moduleapi.TaskStatusCancelled, CurrentStageKey: &claim.Stage.Key, FinishedAt: &now, DurationMS: durationSince(claim.Task.StartedAt, now)}); err != nil {
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

// nullableRequestedBy 将零值请求者标识转换为 nil，否则返回该标识的指针。
func nullableRequestedBy(value uint64) *uint64 {
	if value == 0 {
		return nil
	}
	return &value
}

// normalizedMaxAttempts 将小于等于零的最大尝试次数规范化为 1。
// 返回规范化后的最大尝试次数。
func normalizedMaxAttempts(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}

// hasPendingStages 判断是否仍有 pending 或 running 阶段；该结果用于决定 Task 是否可以进入终态。
func hasPendingStages(stages []taskmodel.Stage) bool {
	for _, stage := range stages {
		if stage.Status == moduleapi.StageStatusPending || stage.Status == moduleapi.StageStatusRunning {
			return true
		}
	}
	return false
}

// durationSince 返回起止时间之间的毫秒数；缺少开始时间时返回 nil，时钟回拨导致的负数会被截为零。
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
	if err == nil {
		r.runtime.publishTask(r.task.ID, "task.log_appended")
	}
	return err
}
