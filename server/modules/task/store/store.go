// Package store defines Task Runtime persistence contracts and SQL implementation.
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
	// ErrInvalidInput reports input that cannot represent a persisted Task Runtime fact.
	ErrInvalidInput = errors.New("invalid task store input")
	// ErrNotFound reports a requested Task Runtime record that does not exist.
	ErrNotFound = errors.New("task runtime record not found")
	// ErrStateConflict reports a compare-and-swap update whose prior state no longer matches.
	ErrStateConflict = errors.New("task runtime state conflict")
)

// Repository persists Task Runtime facts. It intentionally exposes no worker-claim operation in this batch.
type Repository interface {
	Create(ctx context.Context, input CreateInput) (taskmodel.Task, []taskmodel.Stage, error)
	Get(ctx context.Context, taskID uint64) (taskmodel.Task, error)
	ListStages(ctx context.Context, taskID uint64) ([]taskmodel.Stage, error)
	ListEvents(ctx context.Context, taskID uint64, afterSequence int64, limit int) ([]taskmodel.Event, error)
	ListLogs(ctx context.Context, taskID uint64, afterSequence int64, limit int) ([]taskmodel.Log, error)
	TransitionTask(ctx context.Context, input TaskTransitionInput) error
	TransitionStage(ctx context.Context, input StageTransitionInput) error
	AppendEvent(ctx context.Context, input AppendEventInput) (taskmodel.Event, error)
	AppendLog(ctx context.Context, input AppendLogInput) (taskmodel.Log, error)
}

// CreateInput freezes a Task and its serial Stage plan in one database transaction.
type CreateInput struct {
	Task   taskmodel.Task
	Stages []taskmodel.Stage
}

// TaskTransitionInput is one compare-and-swap Task state transition.
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

// StageTransitionInput is one compare-and-swap Stage state transition.
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

// AppendEventInput records a non-derivable Task history fact.
type AppendEventInput struct {
	TaskID   uint64
	Sequence int64
	Type     taskmodel.EventType
	Payload  json.RawMessage
}

// AppendLogInput records one ordered executor output or Task Runtime diagnostic line.
type AppendLogInput struct {
	TaskID     uint64
	StageID    *uint64
	Sequence   int64
	Stream     string
	Level      string
	Line       string
	OccurredAt time.Time
}
