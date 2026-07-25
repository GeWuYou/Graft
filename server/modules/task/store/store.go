// Package store 定义 Task Runtime 的持久化契约和 SQL 实现边界。
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
)

var (
	// ErrInvalidInput 表示输入无法表达合法的 Task Runtime 持久化事实。
	ErrInvalidInput = errors.New("invalid task store input")
	// ErrNotFound 表示请求的 Task Runtime 记录不存在。
	ErrNotFound = errors.New("task runtime record not found")
	// ErrStateConflict 表示 compare-and-swap 更新时持久化的前置状态已被其他执行者改变。
	ErrStateConflict = errors.New("task runtime state conflict")
)

// Repository 持久化 Task Runtime 事实，并提供进程内 worker 所需的原子领取、状态迁移和历史追加操作。
type Repository interface {
	Create(ctx context.Context, input CreateInput) (taskmodel.Task, []taskmodel.Stage, bool, error)
	Get(ctx context.Context, taskID uint64) (taskmodel.Task, error)
	List(ctx context.Context, filter moduleapi.TaskListFilter, limit int, offset int) ([]taskmodel.Task, int64, error)
	ListStages(ctx context.Context, taskID uint64) ([]taskmodel.Stage, error)
	ListEvents(ctx context.Context, taskID uint64, afterSequence int64, limit int) ([]taskmodel.Event, error)
	ListLogs(ctx context.Context, taskID uint64, afterSequence int64, limit int) ([]taskmodel.Log, error)
	TransitionTask(ctx context.Context, input TaskTransitionInput) error
	TransitionStage(ctx context.Context, input StageTransitionInput) error
	AppendEvent(ctx context.Context, input AppendEventInput) (taskmodel.Event, error)
	AppendLog(ctx context.Context, input AppendLogInput) (taskmodel.Log, error)
	ClaimNextStage(ctx context.Context, now time.Time) (StageClaim, bool, error)
	RequestCancellation(ctx context.Context, taskID uint64, requestedAt time.Time) (taskmodel.Task, error)
	CancelPendingTask(ctx context.Context, taskID uint64, finishedAt time.Time, durationMS *int64) error
	RetryStage(ctx context.Context, taskID uint64, stageID uint64, retryAt time.Time) (taskmodel.Stage, error)
	RescheduleStage(ctx context.Context, stageID uint64, retryAt time.Time) error
	NextEventSequence(ctx context.Context, taskID uint64) (int64, error)
	NextLogSequence(ctx context.Context, taskID uint64) (int64, error)
	RecoverInterruptedStages(ctx context.Context, now time.Time) (int, error)
	SettleExternalReceipt(ctx context.Context, input ExternalReceiptSettlementInput) (ExternalReceiptSettlement, error)
}

// StageClaim 是由持久化 running 状态表示的 worker 领取结果。
// Task Runtime 不另建租约记录；数据库行锁与状态迁移共同防止并发 worker 重复领取同一阶段。
type StageClaim struct {
	Task  taskmodel.Task
	Stage taskmodel.Stage
}

// CreateInput 在一个数据库事务中冻结 Task 及其串行 Stage 计划，确保调度器不会观察到半成品计划。
type CreateInput struct {
	Task   taskmodel.Task
	Stages []taskmodel.Stage
}

// TaskTransitionInput 描述一次 compare-and-swap Task 状态迁移。
type TaskTransitionInput struct {
	TaskID          uint64
	From            moduleapi.TaskStatus
	To              moduleapi.TaskStatus
	CurrentStageKey *string
	FailureCode     *string
	FailureMessage  *string
	StartedAt       *time.Time
	FinishedAt      *time.Time
	DurationMS      *int64
}

// StageTransitionInput 描述一次 compare-and-swap Stage 状态迁移。
type StageTransitionInput struct {
	StageID        uint64
	From           moduleapi.StageStatus
	To             moduleapi.StageStatus
	Attempt        int
	NextRetryAt    *time.Time
	Result         json.RawMessage
	FailureCode    *string
	FailureMessage *string
	StartedAt      *time.Time
	FinishedAt     *time.Time
	DurationMS     *int64
}

// AppendEventInput 描述一条不可由当前状态推导的 Task 历史事实。
type AppendEventInput struct {
	TaskID   uint64
	Sequence int64
	Type     taskmodel.EventType
	Payload  json.RawMessage
}

// AppendLogInput 描述一条有序的执行器输出或 Task Runtime 诊断日志。
type AppendLogInput struct {
	TaskID     uint64
	StageID    *uint64
	Sequence   int64
	Stream     string
	Level      string
	Line       string
	OccurredAt time.Time
}

// ExternalReceiptSettlementInput 是 Task Runtime 公共 capability 接受的完整绑定、无秘密回执。
type ExternalReceiptSettlementInput struct {
	TaskID          uint64
	StageID         uint64
	ExecutorType    moduleapi.StageExecutorType
	Protocol        string
	OperationID     string
	Outcome         moduleapi.ExternalReceiptOutcome
	FailureCode     string
	IntegritySHA256 string
}

// ExternalReceiptSettlement 记录持久化结论及是否重放了完全相同的回执。
type ExternalReceiptSettlement struct {
	TaskID     uint64
	StageID    uint64
	Status     moduleapi.TaskStatus
	Idempotent bool
}
