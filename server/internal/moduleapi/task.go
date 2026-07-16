// Package moduleapi 定义 server 模块共享的稳定窄化契约。
package moduleapi

import (
	"context"
	"encoding/json"
	"time"
)

// TaskType 标识一种由消费者拥有的 Task 计划类型。
type TaskType string

// StageExecutorType 标识执行一个 Stage 的业务执行器类型。
type StageExecutorType string

// TaskStatus 标识持久化 Task 状态机的状态。
type TaskStatus string

const (
	// TaskStatusPending marks a submitted Task waiting for dispatch.
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusScheduled marks a Task that cannot run before its scheduled time.
	TaskStatusScheduled TaskStatus = "scheduled"
	// TaskStatusRunning marks a Task with an active Stage.
	TaskStatusRunning TaskStatus = "running"
	// TaskStatusSuccess marks a successfully completed Task.
	TaskStatusSuccess TaskStatus = "success"
	// TaskStatusFailed marks a Task with a terminal failed Stage.
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusCancelled marks a cooperatively cancelled Task.
	TaskStatusCancelled TaskStatus = "cancelled"
	// TaskStatusNeedsAttention marks a Task that needs operator reconciliation.
	TaskStatusNeedsAttention TaskStatus = "needs_attention"
)

// StageStatus 标识持久化 Stage 状态机的状态。
type StageStatus string

const (
	// StageStatusPending marks a Stage waiting for its turn.
	StageStatusPending StageStatus = "pending"
	// StageStatusRunning marks a Stage currently held by a worker.
	StageStatusRunning StageStatus = "running"
	// StageStatusSuccess marks a successfully completed Stage.
	StageStatusSuccess StageStatus = "success"
	// StageStatusFailed marks a terminal failed Stage.
	StageStatusFailed StageStatus = "failed"
	// StageStatusSkipped marks a plan Stage intentionally not run.
	StageStatusSkipped StageStatus = "skipped"
	// StageStatusCancelled marks a cooperatively cancelled Stage.
	StageStatusCancelled StageStatus = "cancelled"
	// StageStatusUnknown marks an interrupted Stage whose external result is indeterminate.
	StageStatusUnknown StageStatus = "unknown"
)

// StageRecoveryPolicy 定义运行时如何处理被中断的运行中 Stage。
type StageRecoveryPolicy string

const (
	// StageRecoveryManualReconcile requires an operator decision after an interrupted Stage.
	StageRecoveryManualReconcile StageRecoveryPolicy = "manual_reconcile"
	// StageRecoveryRetryIfIdempotent permits retry only when the consumer has declared the Stage idempotent.
	StageRecoveryRetryIfIdempotent StageRecoveryPolicy = "retry_if_idempotent"
)

// TaskOwner 标识拥有某个 Task 的业务资源。
type TaskOwner struct {
	Type string
	ID   string
}

// TaskListFilter 将 Task 历史限定到一个已授权所有者及可选 API 过滤条件。
type TaskListFilter struct {
	Owner  TaskOwner
	Type   *TaskType
	Status *TaskStatus
}

// StageRetryPolicy 固化已提交 TaskPlan 中某个 Stage 的重试策略。
type StageRetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
}

// StagePlan 定义 TaskPlan 中一个有序且不可变的 Stage。
type StagePlan struct {
	Key            string
	ExecutorType   StageExecutorType
	Input          json.RawMessage
	RetryPolicy    StageRetryPolicy
	RecoveryPolicy StageRecoveryPolicy
}

// TaskPlan 定义已提交 Task 的冻结有序 Stage 集合。
type TaskPlan struct {
	Stages []StagePlan
}

// SubmitTaskInput 提供创建新 Task 所需的消费者计划和资源身份。
type SubmitTaskInput struct {
	Type        TaskType
	Owner       TaskOwner
	RequestedBy uint64
	Input       json.RawMessage
	Metadata    json.RawMessage
	Plan        TaskPlan
	ScheduledAt *time.Time
}

// TaskReceipt 标识已接受的异步 Task 提交。
type TaskReceipt struct {
	TaskID uint64
	Status TaskStatus
}

// TaskService 向消费者模块暴露 Task Runtime 提交能力。
type TaskService interface {
	Submit(ctx context.Context, input SubmitTaskInput) (TaskReceipt, error)
	Cancel(ctx context.Context, taskID uint64) error
	RetryStage(ctx context.Context, taskID uint64, stageID uint64) error
}

// TaskQueryService 暴露 Task Runtime 读取能力，但不泄漏模块拥有的持久化实现。
type TaskQueryService interface {
	GetTask(ctx context.Context, taskID uint64) (TaskView, error)
	ListTasks(ctx context.Context, filter TaskListFilter, limit int, offset int) ([]TaskView, int64, error)
	ListTaskStages(ctx context.Context, taskID uint64) ([]TaskStageView, error)
	ListTaskEvents(ctx context.Context, taskID uint64, after int64, limit int) ([]TaskEventView, error)
	ListTaskLogs(ctx context.Context, taskID uint64, after int64, limit int) ([]TaskLogView, error)
}

// TaskView 是 Task Runtime 暴露的稳定 Task 读取模型。
type TaskView struct {
	ID              uint64
	Type            TaskType
	Owner           TaskOwner
	Status          TaskStatus
	CurrentStageKey *string
	CreatedBy       *uint64
	CreatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
	DurationMS      *int64
	FailureCode     *string
	FailureMessage  *string
}

// TaskStageView 是稳定的 Stage 时间线读取模型。
type TaskStageView struct {
	ID             uint64
	Key            string
	Sequence       int
	ExecutorType   StageExecutorType
	Status         StageStatus
	Attempt        int
	MaxAttempts    int
	RecoveryPolicy StageRecoveryPolicy
	StartedAt      *time.Time
	FinishedAt     *time.Time
	DurationMS     *int64
	FailureCode    *string
	FailureMessage *string
}

// TaskEventView 是一条不可由其它数据推导的持久化 Task 事实。
type TaskEventView struct {
	ID        uint64
	Sequence  int64
	Type      string
	Payload   json.RawMessage
	CreatedAt time.Time
}

// TaskLogView 是一条持久化的执行器输出记录。
type TaskLogView struct {
	ID         uint64
	TaskID     uint64
	StageID    *uint64
	Sequence   int64
	Stream     string
	Level      string
	Line       string
	OccurredAt time.Time
}

// StageRun 是提供给 StageExecutor 的有界执行句柄。
type StageRun interface {
	TaskID() uint64
	StageID() uint64
	Attempt() int
	Input() json.RawMessage
	CancellationRequested(ctx context.Context) bool
	AppendLog(ctx context.Context, entry TaskLogEntry) error
}

// TaskLogEntry 是由执行器产生、由 Task Runtime 拥有的一条日志记录。
type TaskLogEntry struct {
	Stream string
	Level  string
	Line   string
}

// StageExecutor 执行一个消费者拥有的 Stage，但不得直接修改 Task 状态。
type StageExecutor interface {
	Type() StageExecutorType
	Execute(ctx context.Context, run StageRun) error
	Cancel(ctx context.Context, run StageRun) error
}

// TaskOwnerAction 标识请求业务资源授权的操作。
type TaskOwnerAction string

const (
	// TaskOwnerActionView authorizes Task history, detail, event and log reads.
	TaskOwnerActionView TaskOwnerAction = "view"
	// TaskOwnerActionCancel authorizes a cancellation request.
	TaskOwnerActionCancel TaskOwnerAction = "cancel"
	// TaskOwnerActionRetry authorizes a manual Stage retry request.
	TaskOwnerActionRetry TaskOwnerAction = "retry"
)

// TaskOwnerAuthorizer 为通用 Task API 校验消费者拥有的资源授权。
type TaskOwnerAuthorizer interface {
	OwnerType() string
	AuthorizeTaskOwner(ctx context.Context, actor *CurrentUser, action TaskOwnerAction, owner TaskOwner) error
}

// TaskRuntimeRegistrar 在模块 Register 阶段接收消费者拥有的执行器和所有者授权器。
type TaskRuntimeRegistrar interface {
	RegisterStageExecutor(executor StageExecutor) error
	RegisterTaskOwnerAuthorizer(authorizer TaskOwnerAuthorizer) error
}

// TaskCapabilities 向 API 消费者暴露当前允许的 Task 详情操作。
type TaskCapabilities struct {
	Cancel      bool
	Retry       bool
	DownloadLog bool
}
