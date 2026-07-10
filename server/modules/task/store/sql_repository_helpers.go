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

// placeholderStyleFor determines the SQL placeholder style for a database driver. It returns
// placeholderQuestion for SQLite drivers and placeholderDollar for other drivers or nil databases.
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

func (r *SQLRepository) jsonValuePlaceholder() string {
	if r.placeholder == placeholderDollar {
		return "?::jsonb"
	}
	return "?"
}

func (r *SQLRepository) timestampValuePlaceholder() string {
	if r.placeholder == placeholderDollar {
		return "?::timestamptz"
	}
	return "?"
}

// taskColumns 返回任务表的固定列名列表。
func taskColumns() string {
	return `id, task_type, owner_type, owner_id, status, input_json, metadata_json, plan_json, state_json,
		current_stage_key, created_by, scheduled_at, cancel_requested_at, started_at, finished_at, duration_ms,
		failure_code, failure_message, created_at, updated_at`
}

// taskColumnsFor 返回带指定表别名的任务表列名列表。
func taskColumnsFor(alias string) string {
	return alias + `.id, ` + alias + `.task_type, ` + alias + `.owner_type, ` + alias + `.owner_id, ` + alias + `.status, ` + alias + `.input_json, ` + alias + `.metadata_json, ` + alias + `.plan_json, ` + alias + `.state_json,
		` + alias + `.current_stage_key, ` + alias + `.created_by, ` + alias + `.scheduled_at, ` + alias + `.cancel_requested_at, ` + alias + `.started_at, ` + alias + `.finished_at, ` + alias + `.duration_ms,
		` + alias + `.failure_code, ` + alias + `.failure_message, ` + alias + `.created_at, ` + alias + `.updated_at`
}

// stageColumns returns the comma-separated column names selected from the stage table.
func stageColumns() string {
	return `id, task_id, stage_key, sequence, executor_type, status, attempt, max_attempts, retry_backoff_ms,
		next_retry_at, input_json, recovery_policy, result_json, failure_code, failure_message, started_at,
		finished_at, duration_ms, created_at, updated_at`
}

// stageColumnsFor returns a comma-separated list of stage columns qualified with the specified table alias.
func stageColumnsFor(alias string) string {
	return alias + `.id, ` + alias + `.task_id, ` + alias + `.stage_key, ` + alias + `.sequence, ` + alias + `.executor_type, ` + alias + `.status, ` + alias + `.attempt, ` + alias + `.max_attempts, ` + alias + `.retry_backoff_ms,
		` + alias + `.next_retry_at, ` + alias + `.input_json, ` + alias + `.recovery_policy, ` + alias + `.result_json, ` + alias + `.failure_code, ` + alias + `.failure_message, ` + alias + `.started_at,
		` + alias + `.finished_at, ` + alias + `.duration_ms, ` + alias + `.created_at, ` + alias + `.updated_at`
}

// eventColumns returns the comma-separated column names selected from the event table.
func eventColumns() string {
	return `id, task_id, sequence, event_type, payload_json, created_at`
}

// logColumns returns the comma-separated list of log table column names.
func logColumns() string {
	return `id, task_id, stage_id, sequence, stream, level, line, occurred_at`
}

// scanTask scans a database row into a taskmodel.Task, normalizing JSON fields and nullable values.
// It returns the populated task or the scanning error.
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

// scanStageClaim scans a joined task and stage record into a StageClaim.
// It returns an empty StageClaim and the scanning error if the record cannot be read.
func scanStageClaim(scanner interface{ Scan(dest ...any) error }) (StageClaim, error) {
	var claim StageClaim
	var taskType, taskStatus string
	var taskInput, taskMetadata, taskPlan, taskState []byte
	var taskCurrentStageKey, taskFailureCode, taskFailureMessage sql.NullString
	var taskCreatedBy sql.NullInt64
	var taskScheduledAt, taskCancelRequestedAt, taskStartedAt, taskFinishedAt sql.NullTime
	var taskDurationMS sql.NullInt64
	var stageExecutorType, stageStatus, stageRecoveryPolicy string
	var stageInput, stageResult []byte
	var stageNextRetryAt, stageStartedAt, stageFinishedAt sql.NullTime
	var stageFailureCode, stageFailureMessage sql.NullString
	var stageDurationMS sql.NullInt64
	if err := scanner.Scan(
		&claim.Task.ID, &taskType, &claim.Task.Owner.Type, &claim.Task.Owner.ID, &taskStatus, &taskInput, &taskMetadata, &taskPlan, &taskState,
		&taskCurrentStageKey, &taskCreatedBy, &taskScheduledAt, &taskCancelRequestedAt, &taskStartedAt, &taskFinishedAt, &taskDurationMS,
		&taskFailureCode, &taskFailureMessage, &claim.Task.CreatedAt, &claim.Task.UpdatedAt,
		&claim.Stage.ID, &claim.Stage.TaskID, &claim.Stage.Key, &claim.Stage.Sequence, &stageExecutorType, &stageStatus, &claim.Stage.Attempt, &claim.Stage.MaxAttempts, &claim.Stage.RetryBackoffMS,
		&stageNextRetryAt, &stageInput, &stageRecoveryPolicy, &stageResult, &stageFailureCode, &stageFailureMessage, &stageStartedAt,
		&stageFinishedAt, &stageDurationMS, &claim.Stage.CreatedAt, &claim.Stage.UpdatedAt,
	); err != nil {
		return StageClaim{}, err
	}
	claim.Task.Type = moduleapi.TaskType(taskType)
	claim.Task.Status = moduleapi.TaskStatus(taskStatus)
	claim.Task.Input = normalizeJSON(taskInput)
	claim.Task.Metadata = normalizeJSON(taskMetadata)
	claim.Task.Plan = normalizeJSON(taskPlan)
	claim.Task.State = normalizeJSON(taskState)
	claim.Task.CurrentStageKey = nullableString(taskCurrentStageKey)
	claim.Task.CreatedBy = nullableUint64(taskCreatedBy)
	claim.Task.ScheduledAt = nullableTime(taskScheduledAt)
	claim.Task.CancelRequestedAt = nullableTime(taskCancelRequestedAt)
	claim.Task.StartedAt = nullableTime(taskStartedAt)
	claim.Task.FinishedAt = nullableTime(taskFinishedAt)
	claim.Task.DurationMS = nullableInt64(taskDurationMS)
	claim.Task.FailureCode = nullableString(taskFailureCode)
	claim.Task.FailureMessage = nullableString(taskFailureMessage)
	claim.Stage.ExecutorType = moduleapi.StageExecutorType(stageExecutorType)
	claim.Stage.Status = moduleapi.StageStatus(stageStatus)
	claim.Stage.Input = normalizeJSON(stageInput)
	claim.Stage.RecoveryPolicy = moduleapi.StageRecoveryPolicy(stageRecoveryPolicy)
	claim.Stage.Result = normalizeJSON(stageResult)
	claim.Stage.NextRetryAt = nullableTime(stageNextRetryAt)
	claim.Stage.FailureCode = nullableString(stageFailureCode)
	claim.Stage.FailureMessage = nullableString(stageFailureMessage)
	claim.Stage.StartedAt = nullableTime(stageStartedAt)
	claim.Stage.FinishedAt = nullableTime(stageFinishedAt)
	claim.Stage.DurationMS = nullableInt64(stageDurationMS)
	return claim, nil
}

// scanStages 扫描查询结果中的所有阶段记录并返回阶段列表。
// 如果单行扫描或结果集迭代失败，则返回错误。
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

// 对可空时间、字符串和时长字段进行指针化，并规范化阶段的 JSON 字段。
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

// scanEvents 扫描事件查询结果，并将其转换为任务事件列表；扫描或迭代过程中发生错误时返回错误。
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

// scanLogs reads task log records from rows and converts nullable stage identifiers to pointers.
// It returns an error if scanning a record or iterating through the rows fails.
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

// nullableString returns a pointer to the string value when it is valid, or nil otherwise.
func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

// nullableUint64 将有效且非负的 sql.NullInt64 值转换为 uint64 指针；无效或负值返回 nil。
func nullableUint64(value sql.NullInt64) *uint64 {
	if !value.Valid || value.Int64 < 0 {
		return nil
	}
	result := uint64(value.Int64)
	return &result
}

// nullableInt64 returns a pointer to the integer value when it is valid, or nil otherwise.
func nullableInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

// nullableTime 返回有效数据库时间值的 UTC 时间指针；无效值返回 nil。
func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}

// stringPointer returns a pointer to the provided string value.
func stringPointer(value string) *string {
	return &value
}
