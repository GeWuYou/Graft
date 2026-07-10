package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
)

type placeholderStyle int

const (
	placeholderDollar placeholderStyle = iota
	placeholderQuestion
	placeholderGrowthEstimate = 16
)

func placeholderStyleFor(db *sql.DB) placeholderStyle {
	if db != nil && strings.Contains(strings.ToLower(fmt.Sprintf("%T", db.Driver())), "sqlite") {
		return placeholderQuestion
	}
	return placeholderDollar
}

func (s placeholderStyle) rebind(query string) string {
	if s == placeholderQuestion {
		return query
	}
	var builder strings.Builder
	builder.Grow(len(query) + placeholderGrowthEstimate)
	index := 1
	for _, current := range query {
		if current != '?' {
			builder.WriteRune(current)
			continue
		}
		builder.WriteByte('$')
		builder.WriteString(strconv.Itoa(index))
		index++
	}
	return builder.String()
}

func taskColumns() string {
	return `id, task_type, owner_type, owner_id, status, input_json, metadata_json, plan_json, state_json,
		current_stage_key, created_by, scheduled_at, cancel_requested_at, started_at, finished_at, duration_ms,
		failure_code, failure_message, created_at, updated_at`
}

func stageColumns() string {
	return `id, task_id, stage_key, sequence, executor_type, status, attempt, max_attempts, retry_backoff_ms,
		next_retry_at, input_json, recovery_policy, result_json, failure_code, failure_message, started_at,
		finished_at, duration_ms, created_at, updated_at`
}

func eventColumns() string {
	return `id, task_id, sequence, event_type, payload_json, created_at`
}

func logColumns() string {
	return `id, task_id, stage_id, sequence, stream, level, line, occurred_at`
}

func scanTask(scanner interface{ Scan(dest ...any) error }) (taskmodel.Task, error) {
	var item taskmodel.Task
	var taskType string
	var status string
	var input, metadata, plan, state []byte
	var currentStageKey, failureCode, failureMessage sql.NullString
	var createdBy sql.NullInt64
	var scheduledAt, cancelRequestedAt, startedAt, finishedAt sql.NullTime
	var durationMS sql.NullInt64
	if err := scanner.Scan(
		&item.ID, &taskType, &item.Owner.Type, &item.Owner.ID, &status, &input, &metadata, &plan, &state,
		&currentStageKey, &createdBy, &scheduledAt, &cancelRequestedAt, &startedAt, &finishedAt, &durationMS,
		&failureCode, &failureMessage, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return taskmodel.Task{}, err
	}
	item.Type = moduleapi.TaskType(taskType)
	item.Status = moduleapi.TaskStatus(status)
	item.Input = normalizeJSON(input)
	item.Metadata = normalizeJSON(metadata)
	item.Plan = normalizeJSON(plan)
	item.State = normalizeJSON(state)
	item.CurrentStageKey = nullableString(currentStageKey)
	item.CreatedBy = nullableUint64(createdBy)
	item.ScheduledAt = nullableTime(scheduledAt)
	item.CancelRequestedAt = nullableTime(cancelRequestedAt)
	item.StartedAt = nullableTime(startedAt)
	item.FinishedAt = nullableTime(finishedAt)
	item.DurationMS = nullableInt64(durationMS)
	item.FailureCode = nullableString(failureCode)
	item.FailureMessage = nullableString(failureMessage)
	return item, nil
}

func scanStages(rows *sql.Rows) ([]taskmodel.Stage, error) {
	items := make([]taskmodel.Stage, 0)
	for rows.Next() {
		item, err := scanStage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task stages: %w", err)
	}
	return items, nil
}

func scanStage(scanner interface{ Scan(dest ...any) error }) (taskmodel.Stage, error) {
	var item taskmodel.Stage
	var executorType, status, recoveryPolicy string
	var input, result []byte
	var nextRetryAt, startedAt, finishedAt sql.NullTime
	var failureCode, failureMessage sql.NullString
	var durationMS sql.NullInt64
	if err := scanner.Scan(
		&item.ID, &item.TaskID, &item.Key, &item.Sequence, &executorType, &status, &item.Attempt, &item.MaxAttempts, &item.RetryBackoffMS,
		&nextRetryAt, &input, &recoveryPolicy, &result, &failureCode, &failureMessage, &startedAt,
		&finishedAt, &durationMS, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return taskmodel.Stage{}, err
	}
	item.ExecutorType = moduleapi.StageExecutorType(executorType)
	item.Status = moduleapi.StageStatus(status)
	item.Input = normalizeJSON(input)
	item.RecoveryPolicy = moduleapi.StageRecoveryPolicy(recoveryPolicy)
	item.Result = normalizeJSON(result)
	item.NextRetryAt = nullableTime(nextRetryAt)
	item.FailureCode = nullableString(failureCode)
	item.FailureMessage = nullableString(failureMessage)
	item.StartedAt = nullableTime(startedAt)
	item.FinishedAt = nullableTime(finishedAt)
	item.DurationMS = nullableInt64(durationMS)
	return item, nil
}

func scanEvents(rows *sql.Rows) ([]taskmodel.Event, error) {
	items := make([]taskmodel.Event, 0)
	for rows.Next() {
		var item taskmodel.Event
		var eventType string
		var payload []byte
		if err := rows.Scan(&item.ID, &item.TaskID, &item.Sequence, &eventType, &payload, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan task event: %w", err)
		}
		item.Type = taskmodel.EventType(eventType)
		item.Payload = normalizeJSON(payload)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task events: %w", err)
	}
	return items, nil
}

func scanLogs(rows *sql.Rows) ([]taskmodel.Log, error) {
	items := make([]taskmodel.Log, 0)
	for rows.Next() {
		var item taskmodel.Log
		var stageID sql.NullInt64
		if err := rows.Scan(&item.ID, &item.TaskID, &stageID, &item.Sequence, &item.Stream, &item.Level, &item.Line, &item.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan task log: %w", err)
		}
		item.StageID = nullableUint64(stageID)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task logs: %w", err)
	}
	return items, nil
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func nullableUint64(value sql.NullInt64) *uint64 {
	if !value.Valid || value.Int64 < 0 {
		return nil
	}
	result := uint64(value.Int64)
	return &result
}

func nullableInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}
