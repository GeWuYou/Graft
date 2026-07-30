package task

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"graft/server/internal/logger"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	taskcontract "graft/server/modules/task/contract"
	taskmodel "graft/server/modules/task/model"
	taskstore "graft/server/modules/task/store"
)

const (
	defaultWorkerCount          = 2
	defaultPollInterval         = 250 * time.Millisecond
	errorCodeExecutor           = "stage_executor_failed"
	errorCodeCancelled          = "cancelled"
	errorCodeMissingExec        = "stage_executor_unavailable"
	externalReceiptSHA256Length = 64
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
//
//nolint:cyclop // 提交必须在同一事务边界内校验冻结计划、幂等身份并持久化阶段。
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
	keyHash, fingerprint, err := submissionIdentity(input, plan)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	status := moduleapi.TaskStatusPending
	if input.ScheduledAt != nil && input.ScheduledAt.After(time.Now().UTC()) {
		status = moduleapi.TaskStatusScheduled
	}
	task := taskmodel.Task{
		Type: input.Type, Owner: input.Owner, Status: status, Input: input.Input,
		Metadata: input.Metadata, Plan: plan, State: json.RawMessage(`{}`),
		CreatedBy: nullableRequestedBy(input.RequestedBy), IdempotencyKeyHash: keyHash,
		SubmissionFingerprint: fingerprint, ScheduledAt: input.ScheduledAt,
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
	created, _, idempotent, err := r.repository.Create(ctx, taskstore.CreateInput{Task: task, Stages: stages})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("create task: %w", err)
	}
	if idempotent {
		return moduleapi.TaskReceipt{TaskID: created.ID, Status: created.Status}, nil
	}
	r.signalWake()
	r.publishTask(created.ID, taskcontract.TaskRealtimeEventCreated)
	return moduleapi.TaskReceipt{TaskID: created.ID, Status: created.Status}, nil
}

func submissionIdentity(input moduleapi.SubmitTaskInput, plan json.RawMessage) (*string, *string, error) {
	if input.IdempotencyKey == "" {
		return nil, nil, nil
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" || utf8.RuneCountInString(input.IdempotencyKey) > moduleapi.TaskIdempotencyKeyMaxRunes {
		return nil, nil, fmt.Errorf("%w: idempotency key must be non-blank and at most 128 characters", taskstore.ErrInvalidInput)
	}
	canonicalInput, err := canonicalSubmissionJSON(input.Input)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: canonicalize task input: %v", taskstore.ErrInvalidInput, err)
	}
	canonicalMetadata, err := canonicalSubmissionJSON(input.Metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: canonicalize task metadata: %v", taskstore.ErrInvalidInput, err)
	}
	canonicalPlan, err := canonicalSubmissionJSON(plan)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: canonicalize task plan: %v", taskstore.ErrInvalidInput, err)
	}
	var scheduledAt *string
	if input.ScheduledAt != nil {
		formatted := input.ScheduledAt.UTC().Format(time.RFC3339Nano)
		scheduledAt = &formatted
	}
	payload := struct {
		Type        moduleapi.TaskType  `json:"type"`
		Owner       moduleapi.TaskOwner `json:"owner"`
		RequestedBy uint64              `json:"requested_by"`
		Input       json.RawMessage     `json:"input"`
		Metadata    json.RawMessage     `json:"metadata"`
		Plan        json.RawMessage     `json:"plan"`
		ScheduledAt *string             `json:"scheduled_at"`
	}{Type: input.Type, Owner: input.Owner, RequestedBy: input.RequestedBy, Input: canonicalInput, Metadata: canonicalMetadata, Plan: canonicalPlan, ScheduledAt: scheduledAt}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: marshal task submission fingerprint: %v", taskstore.ErrInvalidInput, err)
	}
	keyDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(input.IdempotencyKey)))
	fingerprint := fmt.Sprintf("%x", sha256.Sum256(encoded))
	return &keyDigest, &fingerprint, nil
}

func canonicalSubmissionJSON(raw json.RawMessage) (json.RawMessage, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return json.RawMessage(`null`), nil
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, errors.New("multiple JSON values")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

// SettleExternalReceipt 通过 Task Runtime 边界接收短生命周期外部执行器的已绑定、无秘密回执。
// 调用方必须使用部署侧可信交付路径；该 capability 刻意不是 HTTP 或远程 agent API。
func (r *Runtime) SettleExternalReceipt(ctx context.Context, receipt moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	if r == nil || r.repository == nil {
		return moduleapi.ExternalReceiptSettlement{}, errors.New("task runtime repository is unavailable")
	}
	stage, expectation, err := r.boundExternalReceipt(ctx, receipt)
	if err != nil {
		return moduleapi.ExternalReceiptSettlement{}, err
	}
	settlement, err := r.repository.SettleExternalReceipt(ctx, taskstore.ExternalReceiptSettlementInput{
		TaskID: receipt.TaskID, StageID: stage.ID, ExecutorType: receipt.ExecutorType, Protocol: expectation.Protocol,
		OperationID: expectation.OperationID, Outcome: receipt.Outcome, FailureCode: receipt.FailureCode, IntegritySHA256: receipt.IntegritySHA256,
	})
	if err != nil {
		return moduleapi.ExternalReceiptSettlement{}, err
	}
	r.publishExternalReceiptSettlement(receipt, settlement)
	return moduleapi.ExternalReceiptSettlement{TaskID: settlement.TaskID, StageID: settlement.StageID, Status: settlement.Status, Idempotent: settlement.Idempotent}, nil
}

func (r *Runtime) boundExternalReceipt(ctx context.Context, receipt moduleapi.ExternalTaskReceipt) (taskmodel.Stage, *moduleapi.ExternalReceiptExpectation, error) {
	if err := validateExternalReceipt(receipt); err != nil {
		return taskmodel.Stage{}, nil, err
	}
	task, err := r.repository.Get(ctx, receipt.TaskID)
	if err != nil {
		return taskmodel.Stage{}, nil, err
	}
	var plan moduleapi.TaskPlan
	if err := json.Unmarshal(task.Plan, &plan); err != nil {
		return taskmodel.Stage{}, nil, fmt.Errorf("decode frozen task plan: %w", err)
	}
	stages, err := r.repository.ListStages(ctx, receipt.TaskID)
	if err != nil {
		return taskmodel.Stage{}, nil, err
	}
	return externalReceiptBinding(plan, stages, receipt)
}

func (r *Runtime) publishExternalReceiptSettlement(receipt moduleapi.ExternalTaskReceipt, settlement taskstore.ExternalReceiptSettlement) {
	if settlement.Idempotent {
		return
	}
	eventType := taskcontract.TaskRealtimeEventStageCompleted
	if receipt.Outcome != moduleapi.ExternalReceiptOutcomeSuccess {
		eventType = taskcontract.TaskRealtimeEventStageFailed
	}
	r.publishTask(receipt.TaskID, eventType)
}

func validateExternalReceipt(receipt moduleapi.ExternalTaskReceipt) error {
	if !externalReceiptIdentityValid(receipt) {
		return errors.New("external receipt identity is incomplete")
	}
	if !lowercaseSHA256(receipt.IntegritySHA256) {
		return errors.New("external receipt integrity digest must be a lowercase sha256")
	}
	return validateExternalReceiptOutcome(receipt)
}

func externalReceiptIdentityValid(receipt moduleapi.ExternalTaskReceipt) bool {
	return receipt.TaskID != 0 && strings.TrimSpace(string(receipt.ExecutorType)) != "" && strings.TrimSpace(receipt.Protocol) != "" && strings.TrimSpace(receipt.OperationID) != "" && len(receipt.Protocol) <= 128 && len(receipt.OperationID) <= 256 && len(receipt.FailureCode) <= 128
}

func lowercaseSHA256(value string) bool {
	if len(value) != externalReceiptSHA256Length {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func validateExternalReceiptOutcome(receipt moduleapi.ExternalTaskReceipt) error {
	switch receipt.Outcome {
	case moduleapi.ExternalReceiptOutcomeSuccess:
		if receipt.FailureCode != "" {
			return errors.New("successful external receipt cannot contain a failure code")
		}
	case moduleapi.ExternalReceiptOutcomeFailed, moduleapi.ExternalReceiptOutcomeNeedsAttention:
		if strings.TrimSpace(receipt.FailureCode) == "" {
			return errors.New("non-success external receipt requires a failure code")
		}
	default:
		return errors.New("external receipt outcome is unsupported")
	}
	return nil
}

func externalReceiptBinding(plan moduleapi.TaskPlan, stages []taskmodel.Stage, receipt moduleapi.ExternalTaskReceipt) (taskmodel.Stage, *moduleapi.ExternalReceiptExpectation, error) {
	if len(plan.Stages) != len(stages) {
		return taskmodel.Stage{}, nil, errors.New("frozen task plan does not match persisted stages")
	}
	for index, candidate := range plan.Stages {
		if candidate.ExternalReceipt == nil || candidate.ExecutorType != receipt.ExecutorType || candidate.ExternalReceipt.Protocol != receipt.Protocol || candidate.ExternalReceipt.OperationID != receipt.OperationID {
			continue
		}
		if index != len(plan.Stages)-1 {
			return taskmodel.Stage{}, nil, errors.New("external receipt stage must be final in task plan")
		}
		if stages[index].Key != candidate.Key || stages[index].ExecutorType != receipt.ExecutorType {
			return taskmodel.Stage{}, nil, errors.New("external receipt stage binding does not match persisted plan")
		}
		return stages[index], candidate.ExternalReceipt, nil
	}
	return taskmodel.Stage{}, nil, errors.New("external receipt does not match a frozen task plan expectation")
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
	return taskLogViews(logs, err)
}

// ListTaskLogsBefore 返回指定序列游标之前的日志，并按序列升序排列。
func (r *Runtime) ListTaskLogsBefore(ctx context.Context, taskID uint64, before int64, limit int) ([]moduleapi.TaskLogView, error) {
	logs, err := r.repository.ListLogsBefore(ctx, taskID, before, limit)
	return taskLogViews(logs, err)
}

// ListLatestTaskLogs 返回最近一页日志，并按序列升序排列。
func (r *Runtime) ListLatestTaskLogs(ctx context.Context, taskID uint64, limit int) ([]moduleapi.TaskLogView, error) {
	logs, err := r.repository.ListLatestLogs(ctx, taskID, limit)
	return taskLogViews(logs, err)
}

func taskLogViews(logs []taskmodel.Log, err error) ([]moduleapi.TaskLogView, error) {
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

// Cancel 持久化取消请求；本进程跟踪的 Stage 由执行器返回结果结算，失联的普通 Stage 则以 cancelled 释放业务资源占用。
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
			r.publishTask(taskID, taskcontract.TaskRealtimeEventCancelled)
		}
		return err
	}
	settled, err := r.cancelRunningTask(ctx, task)
	if err == nil {
		r.publishTask(taskID, taskcontract.TaskRealtimeEventCancelRequested)
		if settled {
			r.publishTask(taskID, taskcontract.TaskRealtimeEventCancelled)
		}
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

func (r *Runtime) cancelRunningTask(ctx context.Context, task taskmodel.Task) (bool, error) {
	r.mu.RLock()
	running, exists := r.running[task.ID]
	r.mu.RUnlock()
	if exists {
		return false, r.cancelTrackedStage(ctx, running)
	}
	stages, err := r.repository.ListStages(ctx, task.ID)
	if err != nil {
		return false, err
	}
	runningStage, err := untrackedRunningStage(stages)
	if err != nil {
		return false, err
	}
	if r.claimedStageWaitsForExternalReceipt(taskstore.StageClaim{Task: task, Stage: *runningStage}) {
		return false, r.appendEvent(ctx, task.ID, taskmodel.EventTypeCancelRequested)
	}
	if err := r.appendEvent(ctx, task.ID, taskmodel.EventTypeCancelRequested); err != nil {
		return false, err
	}
	now := time.Now().UTC()
	if err := r.repository.CancelUntrackedRunningStage(ctx, task.ID, runningStage.ID, now, durationSince(task.StartedAt, now)); err != nil {
		return false, err
	}
	if err := r.appendEvent(ctx, task.ID, taskmodel.EventTypeCancelled); err != nil {
		return false, err
	}
	return true, nil
}

func untrackedRunningStage(stages []taskmodel.Stage) (*taskmodel.Stage, error) {
	var runningStage *taskmodel.Stage
	for _, stage := range stages {
		if stage.Status != moduleapi.StageStatusRunning {
			continue
		}
		if runningStage != nil {
			return nil, taskstore.ErrStateConflict
		}
		current := stage
		runningStage = &current
	}
	if runningStage == nil {
		return nil, taskstore.ErrStateConflict
	}
	return runningStage, nil
}

func (r *Runtime) cancelTrackedStage(ctx context.Context, running runningStage) error {
	if err := r.cancelStage(ctx, running.executor, running.run); err != nil {
		return fmt.Errorf("cancel task stage: %w", err)
	}
	running.cancel()
	r.signalWake()
	return r.appendEvent(ctx, running.run.TaskID(), taskmodel.EventTypeCancelRequested)
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
	r.publishTask(taskID, taskcontract.TaskRealtimeEventRetryRequested)
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
	ticker := time.NewTicker(r.pollEvery)
	defer ticker.Stop()
	for {
		if err := r.runWorkerIteration(ctx); err != nil && !errors.Is(err, context.Canceled) {
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

// runWorkerIteration 将单次领取与执行包在可恢复边界内，避免一个未预期 panic 终止整个 worker goroutine。
func (r *Runtime) runWorkerIteration(ctx context.Context) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			r.logger.Error(ctx, "task worker panicked",
				logger.StringField(logger.FieldOperation, "task_worker"),
				logger.StringField("stacktrace", string(debug.Stack())),
			)
			err = fmt.Errorf("task worker panicked: %v", recovered)
		}
	}()
	return r.runOne(ctx)
}

func (r *Runtime) runOne(ctx context.Context) error {
	claim, found, err := r.repository.ClaimNextStage(ctx, time.Now().UTC())
	if err != nil || !found {
		return err
	}
	r.publishTask(claim.Task.ID, taskcontract.TaskRealtimeEventStageStarted)
	if r.claimedStageWaitsForExternalReceipt(claim) {
		return nil
	}
	executor, ok := r.executorFor(claim.Stage.ExecutorType)
	if !ok {
		return r.failClaim(ctx, claim, errorCodeMissingExec, "no executor registered for stage")
	}
	stageContext, cancel := context.WithCancel(ctx)
	run := &stageRun{runtime: r, task: claim.Task, stage: claim.Stage}
	r.addRunning(claim.Task.ID, runningStage{executor: executor, run: run, cancel: cancel})
	err = r.executeStage(stageContext, executor, run)
	cancel()
	finishErr := r.finishClaim(ctx, claim, err)
	r.removeRunning(claim.Task.ID)
	if finishErr == nil {
		eventType := taskcontract.TaskRealtimeEventStageCompleted
		if err != nil {
			eventType = taskcontract.TaskRealtimeEventStageFailed
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
	for index, stage := range plan.Stages {
		if err := r.validateStagePlan(stage, index, len(plan.Stages), seen); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) validateStagePlan(stage moduleapi.StagePlan, index int, total int, seen map[string]struct{}) error {
	if stage.Key == "" || stage.ExecutorType == "" || stage.RecoveryPolicy == "" {
		return errors.New("task plan stage is incomplete")
	}
	if _, exists := seen[stage.Key]; exists {
		return fmt.Errorf("task plan contains duplicate stage %q", stage.Key)
	}
	if stage.ExternalReceipt != nil {
		if !externalReceiptExpectationValid(stage.ExternalReceipt, index, total) {
			return errors.New("external receipt expectation must bind the final stage with a protocol and operation identity")
		}
	} else if _, exists := r.executorFor(stage.ExecutorType); !exists {
		return fmt.Errorf("task plan references unregistered stage executor %q", stage.ExecutorType)
	}
	seen[stage.Key] = struct{}{}
	return nil
}

// claimedStageWaitsForExternalReceipt 只依赖冻结 TaskPlan 识别外部阶段，避免把短生命周期 runner 误当作常驻本地执行器。
func (r *Runtime) claimedStageWaitsForExternalReceipt(claim taskstore.StageClaim) bool {
	index := claim.Stage.Sequence - 1
	if index < 0 {
		return false
	}
	var plan moduleapi.TaskPlan
	if err := json.Unmarshal(claim.Task.Plan, &plan); err != nil || index >= len(plan.Stages) {
		return false
	}
	stage := plan.Stages[index]
	return stage.ExternalReceipt != nil && stage.Key == claim.Stage.Key && stage.ExecutorType == claim.Stage.ExecutorType
}

func externalReceiptExpectationValid(expectation *moduleapi.ExternalReceiptExpectation, index int, total int) bool {
	return index == total-1 && strings.TrimSpace(expectation.Protocol) != "" && strings.TrimSpace(expectation.OperationID) != "" && len(expectation.Protocol) <= 128 && len(expectation.OperationID) <= 256
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
		r.runtime.publishTask(r.task.ID, taskcontract.TaskRealtimeEventLogAppended)
	}
	return err
}
