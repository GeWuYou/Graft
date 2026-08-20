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

// SQLDialect 选择仓储连接使用的占位符和 JSON/时间值转换语法。
type SQLDialect string

const (
	// SQLDialectPostgres 使用 PostgreSQL 位置占位符和类型转换。
	SQLDialectPostgres SQLDialect = "postgres"
	// SQLDialectSQLite 供单元测试使用 SQLite 问号占位符。
	SQLDialectSQLite SQLDialect = "sqlite"
)

func placeholderStyleForDialect(dialect SQLDialect) (placeholderStyle, error) {
	switch dialect {
	case SQLDialectPostgres:
		return placeholderDollar, nil
	case SQLDialectSQLite:
		return placeholderQuestion, nil
	default:
		return 0, fmt.Errorf("unsupported task repository sql dialect %q", dialect)
	}
}

func (s placeholderStyle) rebind(query string) string {
	if s == placeholderQuestion {
		return query
	}
	var builder strings.Builder
	builder.Grow(len(query) + placeholderGrowthEstimate)
	index := 1
	for queryIndex := 0; queryIndex < len(query); queryIndex++ {
		current := query[queryIndex]
		if current == '?' && queryIndex+1 < len(query) && query[queryIndex+1] == '?' {
			builder.WriteByte('?')
			queryIndex++
			continue
		}
		if current != '?' {
			builder.WriteByte(current)
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
	return `id, task_type, owner_type, owner_id, status, input_json, metadata_json, plan_json, state_json, activation_required,
		current_stage_key, created_by, idempotency_key_hash, submission_fingerprint, scheduled_at, cancel_requested_at, started_at, finished_at, duration_ms,
		failure_code, failure_message, created_at, updated_at`
}

// taskColumnsFor 返回带指定表别名的任务表列名列表。
//
//nolint:dupl // Task 与 Stage 投影刻意保持同构，以便关联扫描顺序可独立审计。
func taskColumnsFor(alias string) string {
	return alias + `.id, ` + alias + `.task_type, ` + alias + `.owner_type, ` + alias + `.owner_id, ` + alias + `.status, ` + alias + `.input_json, ` + alias + `.metadata_json, ` + alias + `.plan_json, ` + alias + `.state_json,
		` + alias + `.activation_required, ` + alias + `.current_stage_key, ` + alias + `.created_by, ` + alias + `.idempotency_key_hash, ` + alias + `.submission_fingerprint, ` + alias + `.scheduled_at, ` + alias + `.cancel_requested_at, ` + alias + `.started_at, ` + alias + `.finished_at, ` + alias + `.duration_ms,
		` + alias + `.failure_code, ` + alias + `.failure_message, ` + alias + `.created_at, ` + alias + `.updated_at`
}

func submissionColumns() string {
	return `id, task_type, owner_type, owner_id, requested_by, idempotency_key_hash, submission_fingerprint,
		state, submission_version, lease_ttl_ms, lease_renewable, lease_token_hash, lease_expires_at, absolute_deadline_at, prerequisite_kind,
		prerequisite_ref, task_id, terminal_reason, created_at, updated_at, activated_at, terminal_at`
}

// stageColumns 返回从 Stage 表选择的逗号分隔列名。
func stageColumns() string {
	return `id, task_id, stage_key, sequence, executor_type, external_execution, status, attempt, max_attempts, retry_backoff_ms,
		next_retry_at, input_json, coordination_group, leg_id, recovery_policy, result_json, failure_code, failure_message, started_at,
		finished_at, duration_ms, created_at, updated_at`
}

// stageColumnsFor 返回使用指定表别名限定的 Stage 逗号分隔列名。
//
//nolint:dupl // Task 与 Stage 投影刻意保持同构，以便关联扫描顺序可独立审计。
func stageColumnsFor(alias string) string {
	return alias + `.id, ` + alias + `.task_id, ` + alias + `.stage_key, ` + alias + `.sequence, ` + alias + `.executor_type, ` + alias + `.external_execution, ` + alias + `.status, ` + alias + `.attempt, ` + alias + `.max_attempts, ` + alias + `.retry_backoff_ms,
		` + alias + `.next_retry_at, ` + alias + `.input_json, ` + alias + `.coordination_group, ` + alias + `.leg_id, ` + alias + `.recovery_policy, ` + alias + `.result_json, ` + alias + `.failure_code, ` + alias + `.failure_message, ` + alias + `.started_at,
		` + alias + `.finished_at, ` + alias + `.duration_ms, ` + alias + `.created_at, ` + alias + `.updated_at`
}

// eventColumns 返回从事件表选择的逗号分隔列名。
func eventColumns() string {
	return `id, task_id, sequence, event_type, payload_json, created_at`
}

// logColumns 返回日志表的逗号分隔列名。
func logColumns() string {
	return `id, task_id, stage_id, sequence, stream, level, line, occurred_at`
}

// scanTask 将数据库行扫描为 taskmodel.Task，并规范化 JSON 与可空字段；扫描失败时返回原始错误。
func scanTask(scanner interface{ Scan(dest ...any) error }) (taskmodel.Task, error) {
	var item taskmodel.Task
	var taskType string
	var status string
	var input, metadata, plan, state []byte
	var currentStageKey, idempotencyKeyHash, submissionFingerprint, failureCode, failureMessage sql.NullString
	var createdBy sql.NullInt64
	var scheduledAt, cancelRequestedAt, startedAt, finishedAt sql.NullTime
	var durationMS sql.NullInt64
	if err := scanner.Scan(
		&item.ID, &taskType, &item.Owner.Type, &item.Owner.ID, &status, &input, &metadata, &plan, &state, &item.ActivationRequired,
		&currentStageKey, &createdBy, &idempotencyKeyHash, &submissionFingerprint, &scheduledAt, &cancelRequestedAt, &startedAt, &finishedAt, &durationMS,
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
	item.IdempotencyKeyHash = nullableString(idempotencyKeyHash)
	item.SubmissionFingerprint = nullableString(submissionFingerprint)
	item.ScheduledAt = nullableTime(scheduledAt)
	item.CancelRequestedAt = nullableTime(cancelRequestedAt)
	item.StartedAt = nullableTime(startedAt)
	item.FinishedAt = nullableTime(finishedAt)
	item.DurationMS = nullableInt64(durationMS)
	item.FailureCode = nullableString(failureCode)
	item.FailureMessage = nullableString(failureMessage)
	return item, nil
}

func scanSubmission(scanner interface{ Scan(dest ...any) error }) (taskmodel.Submission, error) {
	var item taskmodel.Submission
	var taskType, state string
	var requestedBy, taskID sql.NullInt64
	var idempotencyKeyHash, fingerprint, prerequisiteRef, terminalReason sql.NullString
	var leaseTTLMS int64
	var activatedAt, terminalAt sql.NullTime
	if err := scanner.Scan(&item.ID, &taskType, &item.Owner.Type, &item.Owner.ID, &requestedBy, &idempotencyKeyHash, &fingerprint,
		&state, &item.Version, &leaseTTLMS, &item.LeaseRenewable, &item.LeaseTokenHash, &item.LeaseExpiresAt, &item.AbsoluteDeadlineAt, &item.PrerequisiteKind,
		&prerequisiteRef, &taskID, &terminalReason, &item.CreatedAt, &item.UpdatedAt, &activatedAt, &terminalAt); err != nil {
		return taskmodel.Submission{}, err
	}
	item.Type, item.State, item.LeaseTTL = moduleapi.TaskType(taskType), moduleapi.TaskSubmissionState(state), time.Duration(leaseTTLMS)*time.Millisecond
	item.RequestedBy, item.IdempotencyKeyHash, item.SubmissionFingerprint = nullableUint64(requestedBy), nullableString(idempotencyKeyHash), nullableString(fingerprint)
	item.PrerequisiteRef, item.TerminalReason, item.ActivatedAt, item.TerminalAt = nullableString(prerequisiteRef), nullableString(terminalReason), nullableTime(activatedAt), nullableTime(terminalAt)
	if taskID.Valid && taskID.Int64 > 0 {
		value := uint64(taskID.Int64)
		item.TaskID = &value
	}
	return item, nil
}

// scanStageClaim 将关联查询得到的 Task 与 Stage 记录扫描为 StageClaim。
// 记录无法读取时返回空 StageClaim 和扫描错误。
func scanStageClaim(scanner interface{ Scan(dest ...any) error }) (StageClaim, error) {
	var claim StageClaim
	var taskType, taskStatus string
	var taskInput, taskMetadata, taskPlan, taskState []byte
	var taskCurrentStageKey, taskIdempotencyKeyHash, taskSubmissionFingerprint, taskFailureCode, taskFailureMessage sql.NullString
	var taskCreatedBy sql.NullInt64
	var taskScheduledAt, taskCancelRequestedAt, taskStartedAt, taskFinishedAt sql.NullTime
	var taskDurationMS sql.NullInt64
	var stageExecutorType, stageStatus, stageCoordinationGroup, stageLegID, stageRecoveryPolicy string
	var stageInput, stageResult []byte
	var stageNextRetryAt, stageStartedAt, stageFinishedAt sql.NullTime
	var stageFailureCode, stageFailureMessage sql.NullString
	var stageDurationMS sql.NullInt64
	if err := scanner.Scan(
		&claim.Task.ID, &taskType, &claim.Task.Owner.Type, &claim.Task.Owner.ID, &taskStatus, &taskInput, &taskMetadata, &taskPlan, &taskState, &claim.Task.ActivationRequired,
		&taskCurrentStageKey, &taskCreatedBy, &taskIdempotencyKeyHash, &taskSubmissionFingerprint, &taskScheduledAt, &taskCancelRequestedAt, &taskStartedAt, &taskFinishedAt, &taskDurationMS,
		&taskFailureCode, &taskFailureMessage, &claim.Task.CreatedAt, &claim.Task.UpdatedAt,
		&claim.Stage.ID, &claim.Stage.TaskID, &claim.Stage.Key, &claim.Stage.Sequence, &stageExecutorType, &claim.Stage.ExternalExecution, &stageStatus, &claim.Stage.Attempt, &claim.Stage.MaxAttempts, &claim.Stage.RetryBackoffMS,
		&stageNextRetryAt, &stageInput, &stageCoordinationGroup, &stageLegID, &stageRecoveryPolicy, &stageResult, &stageFailureCode, &stageFailureMessage, &stageStartedAt,
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
	claim.Task.IdempotencyKeyHash = nullableString(taskIdempotencyKeyHash)
	claim.Task.SubmissionFingerprint = nullableString(taskSubmissionFingerprint)
	claim.Task.ScheduledAt = nullableTime(taskScheduledAt)
	claim.Task.CancelRequestedAt = nullableTime(taskCancelRequestedAt)
	claim.Task.StartedAt = nullableTime(taskStartedAt)
	claim.Task.FinishedAt = nullableTime(taskFinishedAt)
	claim.Task.DurationMS = nullableInt64(taskDurationMS)
	claim.Task.FailureCode = nullableString(taskFailureCode)
	claim.Task.FailureMessage = nullableString(taskFailureMessage)
	claim.Stage.ExecutorType = moduleapi.StageExecutorType(stageExecutorType)
	claim.Stage.CoordinationGroup = stageCoordinationGroup
	claim.Stage.LegID = stageLegID
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
	var executorType, status, coordinationGroup, legID, recoveryPolicy string
	var input, result []byte
	var nextRetryAt, startedAt, finishedAt sql.NullTime
	var failureCode, failureMessage sql.NullString
	var durationMS sql.NullInt64
	if err := scanner.Scan(
		&item.ID, &item.TaskID, &item.Key, &item.Sequence, &executorType, &item.ExternalExecution, &status, &item.Attempt, &item.MaxAttempts, &item.RetryBackoffMS,
		&nextRetryAt, &input, &coordinationGroup, &legID, &recoveryPolicy, &result, &failureCode, &failureMessage, &startedAt,
		&finishedAt, &durationMS, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return taskmodel.Stage{}, err
	}
	item.ExecutorType = moduleapi.StageExecutorType(executorType)
	item.CoordinationGroup = coordinationGroup
	item.LegID = legID
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

// scanLogs 从查询结果读取任务日志，并将可空阶段标识转换为指针；扫描或迭代失败时返回错误。
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

// nullableString 在值有效时返回字符串指针，否则返回 nil。
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

// nullableInt64 在值有效时返回整数指针，否则返回 nil。
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

// stringPointer 返回给定字符串的独立指针。
func stringPointer(value string) *string {
	return &value
}
