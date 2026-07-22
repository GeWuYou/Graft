// Package model 定义 Task Runtime 所有的持久化事实模型。
package model

import (
	"encoding/json"
	"time"

	"graft/server/internal/moduleapi"
)

// Task 表示消费模块提交的一次持久化 TaskPlan 执行；状态和结果字段由 Runtime 持续更新。
type Task struct {
	ID                uint64
	Type              moduleapi.TaskType
	Owner             moduleapi.TaskOwner
	Status            moduleapi.TaskStatus
	Input             json.RawMessage
	Metadata          json.RawMessage
	Plan              json.RawMessage
	State             json.RawMessage
	CurrentStageKey   *string
	CreatedBy         *uint64
	ScheduledAt       *time.Time
	CancelRequestedAt *time.Time
	StartedAt         *time.Time
	FinishedAt        *time.Time
	DurationMS        *int64
	FailureCode       *string
	FailureMessage    *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Stage 表示冻结计划中的一个阶段及其当前执行状态；计划字段创建后不因重试而重写。
type Stage struct {
	ID             uint64
	TaskID         uint64
	Key            string
	Sequence       int
	ExecutorType   moduleapi.StageExecutorType
	Status         moduleapi.StageStatus
	Attempt        int
	MaxAttempts    int
	RetryBackoffMS int64
	NextRetryAt    *time.Time
	Input          json.RawMessage
	RecoveryPolicy moduleapi.StageRecoveryPolicy
	Result         json.RawMessage
	FailureCode    *string
	FailureMessage *string
	StartedAt      *time.Time
	FinishedAt     *time.Time
	DurationMS     *int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Event 记录不能从 Task 和 Stage 当前状态推导的生命周期、重试、取消或恢复事实。
type Event struct {
	ID        uint64
	TaskID    uint64
	Sequence  int64
	Type      EventType
	Payload   json.RawMessage
	CreatedAt time.Time
}

// EventType 标识除 Task 和 Stage 行之外保留的有限事件历史类型。
type EventType string

// EventType 类型常量标识 Task 和 Stage 行无法推导的历史事实。
const (
	EventTypeCreated                EventType = "created"
	EventTypeCancelRequested        EventType = "cancel_requested"
	EventTypeCancelled              EventType = "cancelled"
	EventTypeRetryRequested         EventType = "retry_requested"
	EventTypeRetryScheduled         EventType = "retry_scheduled"
	EventTypeRecoveryRequired       EventType = "recovery_required"
	EventTypeRecoveryResolved       EventType = "recovery_resolved"
	EventTypeExternalReceiptSettled EventType = "external_receipt_settled"
)

// ExternalReceipt 保存由 Task Runtime 拥有的不可变、无秘密外部执行结算事实。
type ExternalReceipt struct {
	ID              uint64
	TaskID          uint64
	StageID         uint64
	ExecutorType    moduleapi.StageExecutorType
	Protocol        string
	OperationID     string
	Outcome         moduleapi.ExternalReceiptOutcome
	FailureCode     *string
	IntegritySHA256 string
	SettledStatus   moduleapi.TaskStatus
	CreatedAt       time.Time
}

// Log 表示一条按序排列的 Stage 输出或系统诊断记录。
type Log struct {
	ID         uint64
	TaskID     uint64
	StageID    *uint64
	Sequence   int64
	Stream     string
	Level      string
	Line       string
	OccurredAt time.Time
}
