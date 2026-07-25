// Package moduleapi 定义 server 模块共享的稳定窄化契约。
package moduleapi

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrTaskSubmissionConflict 表示同一幂等键被用于内容不同的 Task 提交。
	ErrTaskSubmissionConflict = errors.New("task submission conflict")
)

const (
	// TaskIdempotencyKeyMaxRunes 是 Task Runtime 接受的幂等提交键最大字符数。
	TaskIdempotencyKeyMaxRunes = 128
)

// TaskType 标识一种由消费者拥有的 Task 计划类型。
type TaskType string

// StageExecutorType 标识执行一个 Stage 的业务执行器类型。
type StageExecutorType string

// TaskStatus 标识持久化 Task 状态机的状态。
type TaskStatus string

const (
	// TaskStatusPending 表示已提交但等待调度的 Task。
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusScheduled 表示 Task 尚未到达计划执行时间。
	TaskStatusScheduled TaskStatus = "scheduled"
	// TaskStatusRunning 表示 Task 当前有正在执行的 Stage。
	TaskStatusRunning TaskStatus = "running"
	// TaskStatusSuccess 表示 Task 已成功完成。
	TaskStatusSuccess TaskStatus = "success"
	// TaskStatusFailed 表示 Task 因终态失败的 Stage 结束。
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusCancelled 表示 Task 已被协作式取消。
	TaskStatusCancelled TaskStatus = "cancelled"
	// TaskStatusNeedsAttention 表示 Task 需要操作员人工对账或处置。
	TaskStatusNeedsAttention TaskStatus = "needs_attention"
)

// StageStatus 标识持久化 Stage 状态机的状态。
type StageStatus string

const (
	// StageStatusPending 表示等待执行顺序轮到自己的 Stage。
	StageStatusPending StageStatus = "pending"
	// StageStatusRunning 表示当前由 worker 持有并执行的 Stage。
	StageStatusRunning StageStatus = "running"
	// StageStatusSuccess 表示 Stage 已成功完成。
	StageStatusSuccess StageStatus = "success"
	// StageStatusFailed 表示已进入失败终态的 Stage。
	StageStatusFailed StageStatus = "failed"
	// StageStatusSkipped 表示计划中被明确跳过、未实际执行的 Stage。
	StageStatusSkipped StageStatus = "skipped"
	// StageStatusCancelled 表示 Stage 已被协作式取消。
	StageStatusCancelled StageStatus = "cancelled"
	// StageStatusUnknown 表示 Stage 被中断且外部执行结果无法确定。
	StageStatusUnknown StageStatus = "unknown"
)

// StageRecoveryPolicy 定义运行时如何处理被中断的运行中 Stage。
type StageRecoveryPolicy string

const (
	// StageRecoveryManualReconcile 要求中断后由操作员决定如何处置 Stage。
	StageRecoveryManualReconcile StageRecoveryPolicy = "manual_reconcile"
	// StageRecoveryRetryIfIdempotent 仅在消费者声明 Stage 幂等时允许重试。
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
	Key             string
	ExecutorType    StageExecutorType
	Input           json.RawMessage
	RetryPolicy     StageRetryPolicy
	RecoveryPolicy  StageRecoveryPolicy
	ExternalReceipt *ExternalReceiptExpectation
}

// ExternalReceiptExpectation 冻结短生命周期外部执行器结算最终 Stage 时必须匹配的身份；它不包含凭据，认证由部署侧文件与进程边界承担。
type ExternalReceiptExpectation struct {
	Protocol    string
	OperationID string
}

// ExternalReceiptOutcome 标识外部执行回执允许携带的受限终态结论。
type ExternalReceiptOutcome string

const (
	// ExternalReceiptOutcomeSuccess 表示受限外部操作已成功完成。
	ExternalReceiptOutcomeSuccess ExternalReceiptOutcome = "success"
	// ExternalReceiptOutcomeFailed 表示受限外部操作失败但无需保留人工对账状态。
	ExternalReceiptOutcomeFailed ExternalReceiptOutcome = "failed"
	// ExternalReceiptOutcomeNeedsAttention 表示外部观察结果必须保留给操作者人工对账。
	ExternalReceiptOutcomeNeedsAttention ExternalReceiptOutcome = "needs_attention"
)

// ExternalTaskReceipt 是通过 Task Runtime capability 提交的版本化、无秘密外部执行事实；它刻意排除命令、环境、日志和任意结果载荷。
type ExternalTaskReceipt struct {
	TaskID          uint64
	ExecutorType    StageExecutorType
	Protocol        string
	OperationID     string
	Outcome         ExternalReceiptOutcome
	FailureCode     string
	IntegritySHA256 string
}

// ExternalReceiptSettlement 返回已提交外部回执的持久 Task Runtime 结论。
type ExternalReceiptSettlement struct {
	TaskID     uint64
	StageID    uint64
	Status     TaskStatus
	Idempotent bool
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
	// IdempotencyKey 是调用方提供的不透明提交键；为空时保持历史的非幂等提交语义。
	IdempotencyKey string
	Input          json.RawMessage
	Metadata       json.RawMessage
	Plan           TaskPlan
	ScheduledAt    *time.Time
}

// TaskReceipt 标识已接受的异步 Task 提交。
type TaskReceipt struct {
	TaskID uint64
	Status TaskStatus
}

// TaskService 向消费者模块暴露 Task Runtime 提交能力。
type TaskService interface {
	Submit(ctx context.Context, input SubmitTaskInput) (TaskReceipt, error)
	SettleExternalReceipt(ctx context.Context, receipt ExternalTaskReceipt) (ExternalReceiptSettlement, error)
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
