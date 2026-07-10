// Package model defines the Task Runtime's module-owned persisted facts.
package model

import (
	"encoding/json"
	"time"

	"graft/server/internal/moduleapi"
)

// Task is one persisted execution of a consumer-owned TaskPlan.
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

// Stage is one immutable plan entry and its current execution state.
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

// Event records a non-derivable Task lifecycle, retry, cancellation, or recovery fact.
type Event struct {
	ID        uint64
	TaskID    uint64
	Sequence  int64
	Type      EventType
	Payload   json.RawMessage
	CreatedAt time.Time
}

// EventType identifies the limited event history retained in addition to Task and Stage rows.
type EventType string

// Event type constants identify facts that Task and Stage rows cannot derive.
const (
	EventTypeCreated          EventType = "created"
	EventTypeCancelRequested  EventType = "cancel_requested"
	EventTypeCancelled        EventType = "cancelled"
	EventTypeRetryRequested   EventType = "retry_requested"
	EventTypeRetryScheduled   EventType = "retry_scheduled"
	EventTypeRecoveryRequired EventType = "recovery_required"
	EventTypeRecoveryResolved EventType = "recovery_resolved"
)

// Log is one ordered Stage output or system diagnostic record.
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
