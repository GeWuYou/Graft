// Package moduleapi 定义 server 模块共享的稳定窄化契约。
package moduleapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrTaskSubmissionConflict 表示同一幂等键被用于内容不同的 Task 提交。
	ErrTaskSubmissionConflict = errors.New("task submission conflict")
	// ErrTaskOwnerBusy 表示同一业务资源已有 pending、scheduled、running 或 needs_attention Task 占用执行权。
	ErrTaskOwnerBusy = errors.New("task owner already has an active task")
	// ErrCoordinatedTaskUnsupported 表示当前 Task Runtime 尚未启用分布式 leg 执行器。
	ErrCoordinatedTaskUnsupported = errors.New("coordinated task execution is not enabled")
	// ErrExternalExecutionNotFound 表示当前认证后的目标与能力没有可领取的外部 Stage。
	ErrExternalExecutionNotFound = errors.New("external execution lease not found")
)

const (
	// TaskIdempotencyKeyMaxRunes 是 Task Runtime 接受的幂等提交键最大字符数。
	TaskIdempotencyKeyMaxRunes = 128
	// TaskOwnerIDMaxRunes 与 tasks.owner_id 的持久化长度保持一致，供跨模块 owner 构造与校验复用。
	TaskOwnerIDMaxRunes = 191
)

// TaskType 标识一种由消费者拥有的 Task 计划类型。
type TaskType string

// StageExecutorType 标识执行一个 Stage 的业务执行器类型。
type StageExecutorType string

// ExecutionFailureClass 是执行器向 Task Runtime 报告的稳定失败分类。它只表达恢复
// 语义，不能携带 Provider、凭据或基础设施的实现细节。
type ExecutionFailureClass string

const (
	// ExecutionFailureClassTransient 表示可由既有幂等重试策略处理的短暂失败。
	ExecutionFailureClassTransient ExecutionFailureClass = "transient"
	// ExecutionFailureClassPermanent 表示确定性业务或输入失败。
	ExecutionFailureClassPermanent ExecutionFailureClass = "permanent"
	// ExecutionFailureClassConfiguration 表示需要纠正受控配置的失败。
	ExecutionFailureClassConfiguration ExecutionFailureClass = "configuration"
	// ExecutionFailureClassAuthorization 表示授权或凭据作用域不满足执行要求。
	ExecutionFailureClassAuthorization ExecutionFailureClass = "authorization"
	// ExecutionFailureClassProvider 表示目标 Provider 未能满足冻结执行契约。
	ExecutionFailureClassProvider ExecutionFailureClass = "provider"
	// ExecutionFailureClassInfrastructure 表示需要重验冻结执行环境的基础设施失败。
	ExecutionFailureClassInfrastructure ExecutionFailureClass = "infrastructure"
	// ExecutionFailureClassInternal 表示系统内部失败，需要保留人工恢复事实。
	ExecutionFailureClassInternal ExecutionFailureClass = "internal"
	// ExecutionFailureClassUnknown 表示外部结果无法被安全确认。
	ExecutionFailureClassUnknown ExecutionFailureClass = "unknown"
)

// RecoveryDisposition 是 Task Runtime 对受限执行失败结果允许作出的状态裁决。
type RecoveryDisposition string

const (
	// RecoveryDispositionRetry 使用 Stage 已冻结的幂等重试策略。
	RecoveryDispositionRetry RecoveryDisposition = "retry"
	// RecoveryDispositionFailed 将 Task 结算为确定失败。
	RecoveryDispositionFailed RecoveryDisposition = "failed"
	// RecoveryDispositionNeedsAttention 保留未知或安全敏感结果，禁止自动重试。
	RecoveryDispositionNeedsAttention RecoveryDisposition = "needs_attention"
)

// ExecutionFailure 是 StageExecutor 返回给 Task Runtime 的脱敏结构化失败结果。
// Runtime 独占状态转换；消费者只能提供稳定 code、分类和恢复 disposition。
type ExecutionFailure struct {
	Code        string
	Class       ExecutionFailureClass
	Disposition RecoveryDisposition
	Cause       error
}

// Error 实现 error，避免将 Cause 以外的执行实现细节暴露到 Task 状态之外。
func (f *ExecutionFailure) Error() string {
	if f == nil || f.Cause == nil {
		return "stage execution failed"
	}
	return f.Cause.Error()
}

// Unwrap 保持 callers 可使用 errors.Is 和 errors.As 检查底层失败。
func (f *ExecutionFailure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.Cause
}

// TaskStatus 标识持久化 Task 状态机的状态。
type TaskStatus string

const (
	// TaskStatusPending 表示已提交但等待调度的 Task。
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusReady 表示已完整物化、可由 worker 领取的 Task。
	TaskStatusReady TaskStatus = "ready"
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

// TaskSubmissionState 标识 Task 物化前提交聚合的有限生命周期。
type TaskSubmissionState string

const (
	// TaskSubmissionStateReserved 表示提交持有租约、尚未物化 Task。
	TaskSubmissionStateReserved TaskSubmissionState = "reserved"
	// TaskSubmissionStateActivated 表示提交已原子物化为 Task。
	TaskSubmissionStateActivated TaskSubmissionState = "activated"
	// TaskSubmissionStateDiscarded 表示调用方显式终结提交。
	TaskSubmissionStateDiscarded TaskSubmissionState = "discarded"
	// TaskSubmissionStateExpired 表示提交因租约或绝对截止时间到期而终结。
	TaskSubmissionStateExpired TaskSubmissionState = "expired"
)

// TaskSubmissionPolicy 在开始提交时冻结短生命周期 reservation 的时限。
type TaskSubmissionPolicy struct {
	LeaseTTL         time.Duration
	AbsoluteDeadline time.Duration
	RenewBefore      time.Duration
	AllowRenew       bool
	PrerequisiteKind string
}

// BeginTaskSubmissionInput 描述创建 Task 前的持久化提交事实。
type BeginTaskSubmissionInput struct {
	Task   SubmitTaskInput
	Policy TaskSubmissionPolicy
}

// TaskSubmission 表示跨模块可见、但不泄漏持久化实现的 Submission 读取模型。
type TaskSubmission struct {
	ID                 string
	TaskType           TaskType
	Owner              TaskOwner
	RequestedBy        *uint64
	State              TaskSubmissionState
	SubmissionVersion  int64
	LeaseTTL           time.Duration
	LeaseRenewable     bool
	LeaseExpiresAt     time.Time
	AbsoluteDeadlineAt time.Time
	PrerequisiteKind   string
	PrerequisiteRef    *string
	TaskID             *uint64
	TerminalReason     *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ActivatedAt        *time.Time
	TerminalAt         *time.Time
}

// TaskSubmissionHandle 是调用方持有的短期授权凭据；LeaseToken 不会被持久化为明文。
type TaskSubmissionHandle struct {
	Submission TaskSubmission
	LeaseToken string
}

// TaskSubmissionWriter 在 Task Runtime 持有的同一 SQL 事务中写入模块私有前置条件。
// 实现不得提交、回滚或在返回后使用 transaction。
type TaskSubmissionWriter interface {
	MaterializeTaskSubmission(context.Context, *sql.Tx, TaskSubmission) (string, error)
}

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
	Key          string
	ExecutorType StageExecutorType
	// CoordinationGroup 仅由 CoordinatedTaskService 填充，用于允许同一协调组内的 Stage 并行领取。
	CoordinationGroup string
	// LegID 在一个协调任务内唯一，关联冻结的 Builder 和平台证据。
	LegID             string
	Input             json.RawMessage
	RetryPolicy       StageRetryPolicy
	RecoveryPolicy    StageRecoveryPolicy
	ExternalExecution *ExternalExecutionExpectation
	// ExternalReceipt 是自更新 Controller 完成迁移前保留的 final-stage bridge；新外部执行必须使用 ExternalExecution。
	ExternalReceipt *ExternalReceiptExpectation
}

// CoordinatedLegPlan 是未来由 Task Runtime 协调的单个平台执行 leg；它只携带
// 稳定资源身份和受控输入，不携带 endpoint、凭据或执行命令。
type CoordinatedLegPlan struct {
	ID                string
	Platform          string
	BuilderInstanceID string
	RuntimeTargetID   int64
	Input             json.RawMessage
}

// CoordinatedTaskPlan 定义多 leg 的聚合身份和版本化协调契约。Task Runtime 拥有
// leg 的取消、重试、恢复和聚合终态，Build 只拥有计划和产物事实。
type CoordinatedTaskPlan struct {
	Version           string
	AggregateStageKey string
	Legs              []CoordinatedLegPlan
}

// ValidateCoordinatedTaskPlan 校验分布式 leg 契约的稳定身份和平台输入。
//
//nolint:cyclop // 校验必须显式覆盖每个 leg 的身份、平台唯一性与运行目标绑定。
func ValidateCoordinatedTaskPlan(plan *CoordinatedTaskPlan) error {
	if plan == nil {
		return nil
	}
	if plan.Version == "" || plan.AggregateStageKey == "" || len(plan.Legs) < 2 {
		return errors.New("coordinated task plan is incomplete")
	}
	seen := make(map[string]struct{}, len(plan.Legs))
	platforms := make(map[string]struct{}, len(plan.Legs))
	for _, leg := range plan.Legs {
		if leg.ID == "" || leg.Platform == "" || leg.BuilderInstanceID == "" || leg.RuntimeTargetID < 1 {
			return errors.New("coordinated task leg is incomplete")
		}
		if _, exists := seen[leg.ID]; exists {
			return errors.New("coordinated task plan contains duplicate leg")
		}
		if _, exists := platforms[leg.Platform]; exists {
			return errors.New("coordinated task plan contains duplicate platform")
		}
		seen[leg.ID], platforms[leg.Platform] = struct{}{}, struct{}{}
	}
	return nil
}

// ExternalReceiptExpectation 冻结短生命周期外部执行器结算最终 Stage 时必须匹配的身份；它不包含凭据，认证由部署侧文件与进程边界承担。
type ExternalReceiptExpectation struct {
	Protocol    string
	OperationID string
}

// ExternalExecutionExpectation 冻结一个可由 Runtime Agent 领取的 Stage 执行身份；它不包含 endpoint、凭据或命令。
type ExternalExecutionExpectation struct {
	RuntimeTargetID   int64
	ProviderID        string
	Capability        string
	CapabilityVersion string
	Protocol          string
	OperationID       string
	PayloadSHA256     string
	LeaseTTL          time.Duration
	AbsoluteDeadline  time.Duration
}

// ExternalExecutionLeaseState 标识 Task Runtime 持有的外部 Stage attempt 租约状态。
type ExternalExecutionLeaseState string

const (
	// ExternalExecutionLeaseStateClaimed 表示 Runtime Agent 持有当前 fenced attempt。
	ExternalExecutionLeaseStateClaimed ExternalExecutionLeaseState = "claimed"
	// ExternalExecutionLeaseStateSettled 表示当前 attempt 已由完全匹配的 receipt 结算。
	ExternalExecutionLeaseStateSettled ExternalExecutionLeaseState = "settled"
	// ExternalExecutionLeaseStateExpired 表示租约到期且外部结果无法由 Task Runtime 确认。
	ExternalExecutionLeaseStateExpired ExternalExecutionLeaseState = "expired"
)

// ExternalExecutionClaimRequest 将 claim 限定到经 Runtime Target 认证后的目标和能力绑定。
type ExternalExecutionClaimRequest struct {
	RuntimeTargetID   int64
	ProviderID        string
	Capability        string
	CapabilityVersion string
}

// ExternalExecutionLease 是 Agent 可见的无秘密执行句柄；FenceToken 只在首次 claim 时返回明文。
type ExternalExecutionLease struct {
	ID                    string
	TaskID                uint64
	StageID               uint64
	Attempt               int
	ExecutorType          StageExecutorType
	RuntimeTargetID       int64
	ProviderID            string
	Capability            string
	CapabilityVersion     string
	Protocol              string
	OperationID           string
	PayloadSHA256         string
	Input                 json.RawMessage
	FenceToken            string
	State                 ExternalExecutionLeaseState
	LeaseTTL              time.Duration
	LeaseExpiresAt        time.Time
	AbsoluteDeadlineAt    time.Time
	CancellationRequested bool
}

// ExternalExecutionLeaseHandle 提供续租、日志与结算所需的 fenced lease 身份。
type ExternalExecutionLeaseHandle struct {
	LeaseID    string
	FenceToken string
}

// ExternalExecutionLogBatch 是一次有界、按请求顺序追加的 Stage 日志。
type ExternalExecutionLogBatch struct {
	Handle  ExternalExecutionLeaseHandle
	Entries []TaskLogEntry
}

// ExternalExecutionMaterialRequest 描述一次已围栏 lease 对瞬时执行材料的读取。
// Input 只包含已持久化的 provider-neutral 领域意图；解析器返回的材料不得由 Task Runtime 持久化或记录。
type ExternalExecutionMaterialRequest struct {
	TaskID          uint64
	StageID         uint64
	Attempt         int
	ExecutorType    StageExecutorType
	RuntimeTargetID int64
	OperationID     string
	Input           json.RawMessage
}

// ExternalExecutionMaterial 是只在 mTLS Agent transport 中传递的瞬时执行材料。
type ExternalExecutionMaterial struct {
	Protocol string
	Payload  json.RawMessage
}

// ExternalExecutionMaterialResolver 由领域模块实现，把 provider-neutral 意图解析为一次性执行材料。
type ExternalExecutionMaterialResolver interface {
	Type() StageExecutorType
	ResolveExternalExecutionMaterial(context.Context, ExternalExecutionMaterialRequest) (ExternalExecutionMaterial, error)
}

// ExternalExecutionResult 是 Agent 在 receipt 前提交的瞬时领域结果。Task Runtime
// 只验证 lease fence 并转交领域 recorder，不持久化 Payload。
type ExternalExecutionResult struct {
	Handle   ExternalExecutionLeaseHandle
	Protocol string
	Payload  json.RawMessage
}

// ExternalExecutionResultRequest 为领域 recorder 补充已验证的 Stage 身份。
// Input 仍是冻结的 provider-neutral 意图，Result 只在当前调用期间可见。
type ExternalExecutionResultRequest struct {
	TaskID          uint64
	StageID         uint64
	Attempt         int
	ExecutorType    StageExecutorType
	RuntimeTargetID int64
	OperationID     string
	Input           json.RawMessage
	Protocol        string
	Result          json.RawMessage
}

// ExternalExecutionResultRecorder 由领域模块实现，幂等解释 Agent 结果并写入领域事实。
type ExternalExecutionResultRecorder interface {
	Type() StageExecutorType
	RecordExternalExecutionResult(context.Context, ExternalExecutionResultRequest) error
}

// ExternalExecutionReceipt 是绑定 lease/fence 的受限外部执行结论。
type ExternalExecutionReceipt struct {
	Handle          ExternalExecutionLeaseHandle
	Outcome         ExternalReceiptOutcome
	FailureCode     string
	IntegritySHA256 string
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

// RuntimeAgentExecutionGateway 是 Agent transport adapter 消费的 Task-owned 外部执行能力。
type RuntimeAgentExecutionGateway interface {
	ClaimExternalExecution(context.Context, ExternalExecutionClaimRequest) (ExternalExecutionLease, error)
	InspectExternalExecution(context.Context, ExternalExecutionLeaseHandle) (ExternalExecutionLease, error)
	RenewExternalExecution(context.Context, ExternalExecutionLeaseHandle) (ExternalExecutionLease, error)
	ResolveExternalExecutionMaterial(context.Context, ExternalExecutionLeaseHandle) (ExternalExecutionMaterial, error)
	RecordExternalExecutionResult(context.Context, ExternalExecutionResult) error
	AppendExternalExecutionLogs(context.Context, ExternalExecutionLogBatch) error
	SettleExternalExecution(context.Context, ExternalExecutionReceipt) (ExternalReceiptSettlement, error)
	ExpireExternalExecutions(context.Context, int) (int, error)
}

// TaskPlan 定义已提交 Task 的冻结有序 Stage 集合。
type TaskPlan struct {
	Stages       []StagePlan
	Coordination *CoordinatedTaskPlan
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

// TaskSubmissionService 向前置条件模块暴露 Submission 的租约、物化和终结能力。
type TaskSubmissionService interface {
	BeginSubmission(context.Context, BeginTaskSubmissionInput) (TaskSubmissionHandle, error)
	RenewSubmissionLease(context.Context, TaskSubmissionHandle) (TaskSubmissionHandle, error)
	MaterializeSubmission(context.Context, TaskSubmissionHandle, SubmitTaskInput, TaskSubmissionWriter) (TaskReceipt, error)
	DiscardSubmission(context.Context, TaskSubmissionHandle, string) error
	ExpireSubmissions(context.Context, int) (int, error)
	GetSubmission(context.Context, string) (TaskSubmission, error)
}

// TaskService 向消费者模块暴露 Task Runtime 提交能力。
type TaskService interface {
	Submit(ctx context.Context, input SubmitTaskInput) (TaskReceipt, error)
	SettleExternalReceipt(ctx context.Context, receipt ExternalTaskReceipt) (ExternalReceiptSettlement, error)
	Cancel(ctx context.Context, taskID uint64) error
	RetryStage(ctx context.Context, taskID uint64, stageID uint64) error
}

// CoordinatedTaskService 是多 leg 任务的独立能力边界；只有完成持久化协调器注册后，Task Runtime 才应对外实现可执行语义。
// 调用方不得把它降级成普通 Stage 的本地循环。
type CoordinatedTaskService interface {
	SubmitCoordinated(ctx context.Context, input SubmitTaskInput) (TaskReceipt, error)
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
	RegisterExternalExecutionMaterialResolver(resolver ExternalExecutionMaterialResolver) error
	RegisterExternalExecutionResultRecorder(recorder ExternalExecutionResultRecorder) error
}

// TaskCapabilities 向 API 消费者暴露当前允许的 Task 详情操作。
type TaskCapabilities struct {
	Cancel      bool
	Retry       bool
	DownloadLog bool
}
