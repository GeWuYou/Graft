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

// Repository persists Task Runtime facts and provides the atomic ownership
// operations required by the in-process Task Runtime worker.
type Repository interface {
	Create(ctx context.Context, input CreateInput) (taskmodel.Task, []taskmodel.Stage, error)
	Get(ctx context.Context, taskID uint64) (taskmodel.Task, error)
	List(ctx context.Context, owner moduleapi.TaskOwner, limit int, offset int) ([]taskmodel.Task, int64, error)
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
}

// StageClaim is an exclusive worker lease represented by persisted running state.
// The task runtime does not persist a second lease record: a worker owns the
// claim only while the Stage remains running under compare-and-swap updates.
type StageClaim struct {
	Task  taskmodel.Task
	Stage taskmodel.Stage
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
