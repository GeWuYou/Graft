// Package moduleapi contains stable, narrow contracts shared by server modules.
package moduleapi

import (
	"context"
	"encoding/json"
	"time"
)

// TaskType identifies one consumer-owned Task plan type.
type TaskType string

// StageExecutorType identifies the business executor that runs one Stage.
type StageExecutorType string

// TaskStatus identifies the persisted Task state-machine state.
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

// StageStatus identifies the persisted Stage state-machine state.
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

// StageRecoveryPolicy defines how the runtime handles an interrupted running Stage.
type StageRecoveryPolicy string

const (
	// StageRecoveryManualReconcile requires an operator decision after an interrupted Stage.
	StageRecoveryManualReconcile StageRecoveryPolicy = "manual_reconcile"
	// StageRecoveryRetryIfIdempotent permits retry only when the consumer has declared the Stage idempotent.
	StageRecoveryRetryIfIdempotent StageRecoveryPolicy = "retry_if_idempotent"
)

// TaskOwner identifies the business resource that owns one Task.
type TaskOwner struct {
	Type string
	ID   string
}

// StageRetryPolicy freezes retry policy for one Stage in a submitted TaskPlan.
type StageRetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
}

// StagePlan defines one ordered, immutable Stage in a TaskPlan.
type StagePlan struct {
	Key            string
	ExecutorType   StageExecutorType
	Input          json.RawMessage
	RetryPolicy    StageRetryPolicy
	RecoveryPolicy StageRecoveryPolicy
}

// TaskPlan defines the frozen ordered stages for one submitted Task.
type TaskPlan struct {
	Stages []StagePlan
}

// SubmitTaskInput supplies the consumer-owned plan and resource identity for a new Task.
type SubmitTaskInput struct {
	Type        TaskType
	Owner       TaskOwner
	RequestedBy uint64
	Input       json.RawMessage
	Metadata    json.RawMessage
	Plan        TaskPlan
	ScheduledAt *time.Time
}

// TaskReceipt identifies an accepted asynchronous Task submission.
type TaskReceipt struct {
	TaskID uint64
	Status TaskStatus
}

// TaskService exposes the Task Runtime submission capability to consumer modules.
type TaskService interface {
	Submit(ctx context.Context, input SubmitTaskInput) (TaskReceipt, error)
}

// StageRun is the bounded execution handle supplied to a StageExecutor.
type StageRun interface {
	TaskID() uint64
	StageID() uint64
	Attempt() int
	Input() json.RawMessage
	CancellationRequested(ctx context.Context) bool
	AppendLog(ctx context.Context, entry TaskLogEntry) error
}

// TaskLogEntry is one executor-produced log record owned by the Task Runtime.
type TaskLogEntry struct {
	Stream string
	Level  string
	Line   string
}

// StageExecutor executes one consumer-owned Stage without directly changing Task state.
type StageExecutor interface {
	Type() StageExecutorType
	Execute(ctx context.Context, run StageRun) error
	Cancel(ctx context.Context, run StageRun) error
}

// TaskOwnerAction identifies the operation whose resource authorization is requested.
type TaskOwnerAction string

const (
	// TaskOwnerActionView authorizes Task history, detail, event and log reads.
	TaskOwnerActionView TaskOwnerAction = "view"
	// TaskOwnerActionCancel authorizes a cancellation request.
	TaskOwnerActionCancel TaskOwnerAction = "cancel"
	// TaskOwnerActionRetry authorizes a manual Stage retry request.
	TaskOwnerActionRetry TaskOwnerAction = "retry"
)

// TaskOwnerAuthorizer verifies consumer-owned resource authorization for generic Task APIs.
type TaskOwnerAuthorizer interface {
	OwnerType() string
	AuthorizeTaskOwner(ctx context.Context, actor *CurrentUser, action TaskOwnerAction, owner TaskOwner) error
}

// TaskRuntimeRegistrar accepts consumer-owned executors and owner authorizers during module Register.
type TaskRuntimeRegistrar interface {
	RegisterStageExecutor(executor StageExecutor) error
	RegisterTaskOwnerAuthorizer(authorizer TaskOwnerAuthorizer) error
}

// TaskCapabilities exposes the currently permitted Task Detail operations to API consumers.
type TaskCapabilities struct {
	Cancel      bool
	Retry       bool
	DownloadLog bool
}
