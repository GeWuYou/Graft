package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
	"graft/server/modules/task/state"
)

const (
	defaultPageLimit                      = 100
	maxPageLimit                          = 500
	externalReceiptSHA256Length           = 64
	interruptedCancellationEventFixedArgs = 3
)

// SQLRepository 将 Task Runtime 事实持久化到模块自有的 PostgreSQL 表中。
type SQLRepository struct {
	db          *sql.DB
	placeholder placeholderStyle
}

// databaseOperationError 以稳定的操作名和 SQLSTATE 供日志归类，同时通过错误链保留驱动错误，避免将数据库详情直接暴露给上层。
type databaseOperationError struct {
	operation string
	sqlState  string
	err       error
}

func (e *databaseOperationError) Error() string {
	if e.sqlState != "" {
		return fmt.Sprintf("task store database operation %s failed (sqlstate=%s)", e.operation, e.sqlState)
	}
	return fmt.Sprintf("task store database operation %s failed", e.operation)
}

func (e *databaseOperationError) Unwrap() error { return e.err }

func wrapDatabaseOperation(operation string, err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code != "" {
		return &databaseOperationError{operation: operation, sqlState: pgErr.Code, err: err}
	}
	return &databaseOperationError{operation: operation, err: err}
}

// NewSQLRepository 创建明确选择 SQL 方言语义的仓储；SQLite 方言仅用于保持单元测试的等价执行路径。
func NewSQLRepository(db *sql.DB, dialect SQLDialect) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("task repository requires a non-nil sql db")
	}
	placeholder, err := placeholderStyleForDialect(dialect)
	if err != nil {
		return nil, err
	}
	return &SQLRepository{db: db, placeholder: placeholder}, nil
}

// Create 原子保存冻结的 Task、按序 Stage 计划和创建事件，避免提交出不完整的执行计划。
// 返回的第三个值表示同一幂等提交已存在，调用方不应重复发布创建事件。
//
//nolint:gocognit,gocyclo,cyclop,funlen,revive // 事务必须原子处理幂等查询、插入竞争恢复、阶段和初始事件。
func (r *SQLRepository) Create(ctx context.Context, input CreateInput) (taskmodel.Task, []taskmodel.Stage, bool, error) {
	input, err := normalizeCreateInput(input)
	if err != nil {
		return taskmodel.Task{}, nil, false, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return taskmodel.Task{}, nil, false, wrapDatabaseOperation("task_create_begin", err)
	}
	defer rollback(tx)
	if err := r.lockOwner(ctx, tx, input.Task.Owner); err != nil {
		return taskmodel.Task{}, nil, false, err
	}
	//nolint:nestif // 必须在写入任何 Task 前检查现有的带键提交。
	if input.Task.IdempotencyKeyHash != nil {
		existing, found, findErr := r.findIdempotentTask(ctx, tx, input.Task)
		if findErr != nil {
			return taskmodel.Task{}, nil, false, findErr
		}
		if found {
			if existing.SubmissionFingerprint == nil || input.Task.SubmissionFingerprint == nil || *existing.SubmissionFingerprint != *input.Task.SubmissionFingerprint {
				return taskmodel.Task{}, nil, false, moduleapi.ErrTaskSubmissionConflict
			}
			return existing, nil, true, nil
		}
	}
	var reservedSubmission bool
	if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`SELECT EXISTS(SELECT 1 FROM task_submissions WHERE owner_type = ? AND owner_id = ? AND state = ?)`), input.Task.Owner.Type, input.Task.Owner.ID, moduleapi.TaskSubmissionStateReserved).Scan(&reservedSubmission); err != nil {
		return taskmodel.Task{}, nil, false, wrapDatabaseOperation("reserved_submission_check", err)
	}
	if reservedSubmission {
		return taskmodel.Task{}, nil, false, moduleapi.ErrTaskOwnerBusy
	}

	now := time.Now().UTC()
	input.Task.CreatedAt = now
	input.Task.UpdatedAt = now
	//nolint:nestif // 唯一索引竞争必须先解析为既有幂等 Task，再返回错误。
	if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO tasks (
		task_type, owner_type, owner_id, status, input_json, metadata_json, plan_json, state_json, activation_required,
		current_stage_key, created_by, idempotency_key_hash, submission_fingerprint, scheduled_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
		input.Task.Type,
		input.Task.Owner.Type,
		input.Task.Owner.ID,
		input.Task.Status,
		input.Task.Input,
		input.Task.Metadata,
		input.Task.Plan,
		input.Task.State,
		input.Task.ActivationRequired,
		input.Task.CurrentStageKey,
		input.Task.CreatedBy,
		input.Task.IdempotencyKeyHash,
		input.Task.SubmissionFingerprint,
		input.Task.ScheduledAt,
		now,
		now,
	).Scan(&input.Task.ID); err != nil {
		if input.Task.IdempotencyKeyHash != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				return taskmodel.Task{}, nil, false, fmt.Errorf("rollback duplicate task submission: %w", rollbackErr)
			}
			existing, found, findErr := r.findIdempotentTask(ctx, r.db, input.Task)
			if findErr != nil {
				return taskmodel.Task{}, nil, false, findErr
			}
			if found {
				if existing.SubmissionFingerprint == nil || input.Task.SubmissionFingerprint == nil || *existing.SubmissionFingerprint != *input.Task.SubmissionFingerprint {
					return taskmodel.Task{}, nil, false, moduleapi.ErrTaskSubmissionConflict
				}
				return existing, nil, true, nil
			}
		}
		return taskmodel.Task{}, nil, false, mapCreateTaskError(err)
	}

	stages := make([]taskmodel.Stage, 0, len(input.Stages))
	for _, current := range input.Stages {
		current.TaskID = input.Task.ID
		current.CreatedAt = now
		current.UpdatedAt = now
		if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO task_stages (
			task_id, stage_key, sequence, executor_type, external_execution, status, attempt, max_attempts, retry_backoff_ms,
			next_retry_at, input_json, coordination_group, leg_id, recovery_policy, result_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
			current.TaskID,
			current.Key,
			current.Sequence,
			current.ExecutorType,
			current.ExternalExecution,
			current.Status,
			current.Attempt,
			current.MaxAttempts,
			current.RetryBackoffMS,
			current.NextRetryAt,
			current.Input,
			current.CoordinationGroup,
			current.LegID,
			current.RecoveryPolicy,
			current.Result,
			now,
			now,
		).Scan(&current.ID); err != nil {
			return taskmodel.Task{}, nil, false, wrapDatabaseOperation("task_stage_insert", err)
		}
		stages = append(stages, current)
	}

	createdPayload := json.RawMessage(`{}`)
	if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO task_events (
		task_id, sequence, event_type, payload_json, created_at
	) VALUES (?, ?, ?, ?, ?) RETURNING id`),
		input.Task.ID,
		int64(1),
		taskmodel.EventTypeCreated,
		createdPayload,
		now,
	).Scan(new(uint64)); err != nil {
		return taskmodel.Task{}, nil, false, wrapDatabaseOperation("task_created_event_insert", err)
	}

	if err := tx.Commit(); err != nil {
		return taskmodel.Task{}, nil, false, wrapDatabaseOperation("task_create_commit", err)
	}
	return input.Task, stages, false, nil
}

func mapCreateTaskError(err error) error {
	if isActiveOwnerConflict(err) {
		return fmt.Errorf("insert task: %w", moduleapi.ErrTaskOwnerBusy)
	}
	return wrapDatabaseOperation("task_insert", err)
}

func isActiveOwnerConflict(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == "uq_tasks_active_owner"
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed: tasks.owner_type, tasks.owner_id")
}

type taskQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

//nolint:dupl // Task 与 Submission 的幂等查询故意保持同构，便于审计两种聚合的键语义。
func (r *SQLRepository) findIdempotentTask(ctx context.Context, queryer taskQueryer, task taskmodel.Task) (taskmodel.Task, bool, error) {
	requestedBy := uint64(0)
	if task.CreatedBy != nil {
		requestedBy = *task.CreatedBy
	}
	item, err := scanTask(queryer.QueryRowContext(ctx, r.placeholder.rebind(`SELECT `+taskColumns()+`
		FROM tasks
		WHERE task_type = ? AND owner_type = ? AND owner_id = ? AND COALESCE(created_by, 0) = ?
			AND idempotency_key_hash = ?`), task.Type, task.Owner.Type, task.Owner.ID, requestedBy, task.IdempotencyKeyHash))
	if errors.Is(err, sql.ErrNoRows) {
		return taskmodel.Task{}, false, nil
	}
	if err != nil {
		return taskmodel.Task{}, false, fmt.Errorf("find idempotent task: %w", err)
	}
	return item, true, nil
}

// Get 按稳定 ID 读取一个 Task。
func (r *SQLRepository) Get(ctx context.Context, taskID uint64) (taskmodel.Task, error) {
	if taskID == 0 {
		return taskmodel.Task{}, ErrInvalidInput
	}
	item, err := scanTask(r.db.QueryRowContext(ctx, r.placeholder.rebind(`SELECT `+taskColumns()+`
		FROM tasks WHERE id = ?`), taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return taskmodel.Task{}, ErrNotFound
	}
	if err != nil {
		return taskmodel.Task{}, fmt.Errorf("get task: %w", err)
	}
	return item, nil
}

// GetByIDs 按一次受控 IN 查询批量读取 Task，供列表消费者组装跨模块执行投影。
//
//nolint:cyclop // 批量输入校验、查询和扫描必须保持在同一 repository 边界。
func (r *SQLRepository) GetByIDs(ctx context.Context, taskIDs []uint64) ([]taskmodel.Task, error) {
	if r == nil || r.db == nil || len(taskIDs) == 0 || len(taskIDs) > 1000 {
		return nil, ErrInvalidInput
	}
	placeholders := make([]string, len(taskIDs))
	args := make([]any, len(taskIDs))
	for i, taskID := range taskIDs {
		if taskID == 0 {
			return nil, ErrInvalidInput
		}
		placeholders[i] = "?"
		args[i] = taskID
	}
	query := `SELECT ` + taskColumns() + ` FROM tasks WHERE id IN (` + strings.Join(placeholders, ", ") + `) ORDER BY created_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("get tasks by ids: %w", err)
	}
	defer closeRows(rows)
	items := make([]taskmodel.Task, 0, len(taskIDs))
	for rows.Next() {
		item, scanErr := scanTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks by ids: %w", err)
	}
	return items, nil
}

// List 使用 owner 索引返回按资源所有者隔离、可选筛选的 Task 历史分页及总数。
func (r *SQLRepository) List(ctx context.Context, filter moduleapi.TaskListFilter, limit int, offset int) ([]taskmodel.Task, int64, error) {
	if strings.TrimSpace(filter.Owner.Type) == "" || strings.TrimSpace(filter.Owner.ID) == "" || offset < 0 {
		return nil, 0, ErrInvalidInput
	}
	where, arguments := taskListWhere(filter)
	var total int64
	if err := r.db.QueryRowContext(ctx, r.placeholder.rebind(`SELECT COUNT(*) FROM tasks WHERE `+where), arguments...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count owner tasks: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+taskColumns()+`
		FROM tasks WHERE `+where+`
		ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`), append(arguments, normalizeLimit(limit), offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list owner tasks: %w", err)
	}
	defer closeRows(rows)
	items := make([]taskmodel.Task, 0)
	for rows.Next() {
		item, scanErr := scanTask(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate owner tasks: %w", err)
	}
	return items, total, nil
}

func taskListWhere(filter moduleapi.TaskListFilter) (string, []any) {
	clauses := []string{"owner_type = ?", "owner_id = ?"}
	arguments := []any{filter.Owner.Type, filter.Owner.ID}
	if filter.Type != nil {
		clauses = append(clauses, "task_type = ?")
		arguments = append(arguments, *filter.Type)
	}
	if filter.Status != nil {
		clauses = append(clauses, "status = ?")
		arguments = append(arguments, *filter.Status)
	}
	return strings.Join(clauses, " AND "), arguments
}

// ListStages 返回 Task 不可变的串行阶段计划，并保持执行顺序。
func (r *SQLRepository) ListStages(ctx context.Context, taskID uint64) ([]taskmodel.Stage, error) {
	if taskID == 0 {
		return nil, ErrInvalidInput
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+stageColumns()+`
		FROM task_stages WHERE task_id = ? ORDER BY sequence ASC, id ASC`), taskID)
	if err != nil {
		return nil, fmt.Errorf("list task stages: %w", err)
	}
	defer closeRows(rows)
	return scanStages(rows)
}

// ListEvents 按序列游标重放不能从当前状态推导的 Task 事件。
func (r *SQLRepository) ListEvents(ctx context.Context, taskID uint64, afterSequence int64, limit int) ([]taskmodel.Event, error) {
	if taskID == 0 || afterSequence < 0 {
		return nil, ErrInvalidInput
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+eventColumns()+`
		FROM task_events WHERE task_id = ? AND sequence > ? ORDER BY sequence ASC LIMIT ?`), taskID, afterSequence, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list task events: %w", err)
	}
	defer closeRows(rows)
	return scanEvents(rows)
}

// ListLogs 按序列游标重放 Task 日志。
func (r *SQLRepository) ListLogs(ctx context.Context, taskID uint64, afterSequence int64, limit int) ([]taskmodel.Log, error) {
	if taskID == 0 || afterSequence < 0 {
		return nil, ErrInvalidInput
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+logColumns()+`
		FROM task_logs WHERE task_id = ? AND sequence > ? ORDER BY sequence ASC LIMIT ?`), taskID, afterSequence, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list task logs: %w", err)
	}
	defer closeRows(rows)
	return scanLogs(rows)
}

// ListLogsBefore 返回指定游标之前的日志页，并保持调用方使用的正序展示语义。
func (r *SQLRepository) ListLogsBefore(ctx context.Context, taskID uint64, beforeSequence int64, limit int) ([]taskmodel.Log, error) {
	if taskID == 0 || beforeSequence <= 0 {
		return nil, ErrInvalidInput
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+logColumns()+`
		FROM task_logs WHERE task_id = ? AND sequence < ? ORDER BY sequence DESC LIMIT ?`), taskID, beforeSequence, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list task logs before cursor: %w", err)
	}
	defer closeRows(rows)
	logs, err := scanLogs(rows)
	if err != nil {
		return nil, err
	}
	reverseLogs(logs)
	return logs, nil
}

// ListLatestLogs 返回最近一页日志，并保持调用方使用的正序展示语义。
func (r *SQLRepository) ListLatestLogs(ctx context.Context, taskID uint64, limit int) ([]taskmodel.Log, error) {
	if taskID == 0 {
		return nil, ErrInvalidInput
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+logColumns()+`
		FROM task_logs WHERE task_id = ? ORDER BY sequence DESC LIMIT ?`), taskID, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list latest task logs: %w", err)
	}
	defer closeRows(rows)
	logs, err := scanLogs(rows)
	if err != nil {
		return nil, err
	}
	reverseLogs(logs)
	return logs, nil
}

func reverseLogs(logs []taskmodel.Log) {
	for left, right := 0, len(logs)-1; left < right; left, right = left+1, right-1 {
		logs[left], logs[right] = logs[right], logs[left]
	}
}

// TransitionTask 应用已校验的 compare-and-swap Task 状态迁移，状态冲突时不修改任何记录。
func (r *SQLRepository) TransitionTask(ctx context.Context, input TaskTransitionInput) error {
	if input.TaskID == 0 {
		return ErrInvalidInput
	}
	if err := state.ValidateTaskTransition(input.From, input.To); err != nil {
		return err
	}
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, current_stage_key = ?, failure_code = ?, failure_message = ?,
			started_at = COALESCE(?, started_at), finished_at = ?, duration_ms = ?, updated_at = ?
		WHERE id = ? AND status = ?`),
		input.To,
		input.CurrentStageKey,
		input.FailureCode,
		input.FailureMessage,
		input.StartedAt,
		input.FinishedAt,
		input.DurationMS,
		now,
		input.TaskID,
		input.From,
	)
	if err != nil {
		return fmt.Errorf("transition task: %w", err)
	}
	return expectOneAffected(result)
}

// TransitionStage 应用已校验的 compare-and-swap Stage 状态迁移，状态冲突时不修改任何记录。
func (r *SQLRepository) TransitionStage(ctx context.Context, input StageTransitionInput) error {
	if input.StageID == 0 || input.Attempt < 0 {
		return ErrInvalidInput
	}
	if err := state.ValidateStageTransition(input.From, input.To); err != nil {
		return err
	}
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, attempt = ?, next_retry_at = ?, result_json = ?, failure_code = ?, failure_message = ?,
			started_at = COALESCE(?, started_at), finished_at = ?, duration_ms = ?, updated_at = ?
		WHERE id = ? AND status = ?`),
		input.To,
		input.Attempt,
		input.NextRetryAt,
		normalizeJSON(input.Result),
		input.FailureCode,
		input.FailureMessage,
		input.StartedAt,
		input.FinishedAt,
		input.DurationMS,
		now,
		input.StageID,
		input.From,
	)
	if err != nil {
		return fmt.Errorf("transition task stage: %w", err)
	}
	return expectOneAffected(result)
}

// AppendEvent 持久化不能从当前状态推导的历史事实；realtime 通知必须在该事实成功写入后发布。
func (r *SQLRepository) AppendEvent(ctx context.Context, input AppendEventInput) (taskmodel.Event, error) {
	if input.TaskID == 0 || input.Sequence <= 0 || !validEventType(input.Type) {
		return taskmodel.Event{}, ErrInvalidInput
	}
	now := time.Now().UTC()
	item := taskmodel.Event{TaskID: input.TaskID, Sequence: input.Sequence, Type: input.Type, Payload: normalizeJSON(input.Payload), CreatedAt: now}
	if err := r.db.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO task_events (
		task_id, sequence, event_type, payload_json, created_at
	) VALUES (?, ?, ?, ?, ?) RETURNING id`),
		item.TaskID, item.Sequence, item.Type, item.Payload, item.CreatedAt,
	).Scan(&item.ID); err != nil {
		return taskmodel.Event{}, fmt.Errorf("append task event: %w", err)
	}
	return item, nil
}

// AppendLog 持久化一行执行器输出；日志大小限制由 worker 批处理入口统一执行。
func (r *SQLRepository) AppendLog(ctx context.Context, input AppendLogInput) (taskmodel.Log, error) {
	if input.TaskID == 0 || input.Sequence <= 0 || !validLogStream(input.Stream) || !validLogLevel(input.Level) || strings.TrimSpace(input.Line) == "" {
		return taskmodel.Log{}, ErrInvalidInput
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	item := taskmodel.Log{TaskID: input.TaskID, StageID: input.StageID, Sequence: input.Sequence, Stream: input.Stream, Level: input.Level, Line: input.Line, OccurredAt: input.OccurredAt.UTC()}
	if err := r.db.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO task_logs (
		task_id, stage_id, sequence, stream, level, line, occurred_at
	) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`),
		item.TaskID, item.StageID, item.Sequence, item.Stream, item.Level, item.Line, item.OccurredAt,
	).Scan(&item.ID); err != nil {
		return taskmodel.Log{}, fmt.Errorf("append task log: %w", err)
	}
	return item, nil
}

// ClaimNextStage 原子领取下一个可串行执行的 Stage 并持久化 running 状态；PostgreSQL 使用 SKIP LOCKED 防止并发 worker 重复执行，SQLite 保持等价测试语义。
func (r *SQLRepository) ClaimNextStage(ctx context.Context, now time.Time) (StageClaim, bool, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if r.placeholder == placeholderQuestion {
		return r.claimNextStageSQLite(ctx, now.UTC())
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return StageClaim{}, false, fmt.Errorf("begin stage claim transaction: %w", err)
	}
	defer rollback(tx)

	claim, found, err := r.claimNextStagePostgres(ctx, tx, now.UTC())
	if err != nil || !found {
		return claim, found, err
	}
	if err := tx.Commit(); err != nil {
		return StageClaim{}, false, fmt.Errorf("commit stage claim transaction: %w", err)
	}
	return claim, true, nil
}

func (r *SQLRepository) claimNextStagePostgres(ctx context.Context, tx *sql.Tx, now time.Time) (StageClaim, bool, error) {
	row := tx.QueryRowContext(ctx, r.placeholder.rebind(`SELECT `+taskColumnsFor("task")+`, `+stageColumnsFor("stage")+`
		FROM task_stages stage
		JOIN tasks task ON task.id = stage.task_id
		WHERE stage.status = ?
			AND stage.external_execution = false
			AND (stage.next_retry_at IS NULL OR stage.next_retry_at <= ?)
			AND task.status IN (?, ?)
			AND (task.scheduled_at IS NULL OR task.scheduled_at <= ?)
			AND NOT EXISTS (
				SELECT 1 FROM task_stages earlier
				WHERE earlier.task_id = stage.task_id AND earlier.sequence < stage.sequence
					AND earlier.status NOT IN (?, ?, ?)
					AND (stage.coordination_group = '' OR earlier.coordination_group = '' OR earlier.coordination_group <> stage.coordination_group)
			)
		ORDER BY task.created_at ASC, stage.sequence ASC, stage.id ASC
		FOR UPDATE OF task, stage SKIP LOCKED
		LIMIT 1`),
		moduleapi.StageStatusPending,
		now,
		moduleapi.TaskStatusReady, moduleapi.TaskStatusRunning,
		now,
		moduleapi.StageStatusSuccess, moduleapi.StageStatusSkipped, moduleapi.StageStatusCancelled,
	)
	return r.transitionClaimedStage(ctx, tx, row, now)
}

func (r *SQLRepository) claimNextStageSQLite(ctx context.Context, now time.Time) (StageClaim, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return StageClaim{}, false, fmt.Errorf("begin stage claim transaction: %w", err)
	}
	defer rollback(tx)
	// #nosec G202 -- projection fragments are static module-owned SQL identifiers, never caller input.
	row := tx.QueryRowContext(ctx, `SELECT `+taskColumnsFor("task")+`, `+stageColumnsFor("stage")+`
		FROM task_stages stage
		JOIN tasks task ON task.id = stage.task_id
		WHERE stage.status = ?
			AND stage.external_execution = false
			AND (stage.next_retry_at IS NULL OR stage.next_retry_at <= ?)
			AND task.status IN (?, ?)
			AND (task.scheduled_at IS NULL OR task.scheduled_at <= ?)
			AND NOT EXISTS (
				SELECT 1 FROM task_stages earlier
				WHERE earlier.task_id = stage.task_id AND earlier.sequence < stage.sequence
					AND earlier.status NOT IN (?, ?, ?)
					AND (stage.coordination_group = '' OR earlier.coordination_group = '' OR earlier.coordination_group <> stage.coordination_group)
			)
		ORDER BY task.created_at ASC, stage.sequence ASC, stage.id ASC
		LIMIT 1`,
		moduleapi.StageStatusPending,
		now,
		moduleapi.TaskStatusReady, moduleapi.TaskStatusRunning,
		now,
		moduleapi.StageStatusSuccess, moduleapi.StageStatusSkipped, moduleapi.StageStatusCancelled,
	)
	claim, found, err := r.transitionClaimedStage(ctx, tx, row, now)
	if err != nil || !found {
		return claim, found, err
	}
	if err := tx.Commit(); err != nil {
		return StageClaim{}, false, fmt.Errorf("commit stage claim transaction: %w", err)
	}
	return claim, true, nil
}

func (r *SQLRepository) transitionClaimedStage(ctx context.Context, tx *sql.Tx, row *sql.Row, now time.Time) (StageClaim, bool, error) {
	claim, err := scanStageClaim(row)
	if errors.Is(err, sql.ErrNoRows) {
		return StageClaim{}, false, nil
	}
	if err != nil {
		return StageClaim{}, false, fmt.Errorf("select claimable task stage: %w", err)
	}
	if err := r.markClaimTaskRunning(ctx, tx, &claim, now); err != nil {
		return StageClaim{}, false, err
	}
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, attempt = ?, started_at = COALESCE(started_at, ?), updated_at = ?
		WHERE id = ? AND status = ?`), moduleapi.StageStatusRunning, claim.Stage.Attempt+1, now, now, claim.Stage.ID, moduleapi.StageStatusPending)
	if err != nil {
		return StageClaim{}, false, fmt.Errorf("mark claimed stage running: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return StageClaim{}, false, err
	}
	claim.Stage.Status = moduleapi.StageStatusRunning
	claim.Stage.Attempt++
	claim.Stage.StartedAt = &now
	return claim, true, nil
}

func (r *SQLRepository) markClaimTaskRunning(ctx context.Context, tx *sql.Tx, claim *StageClaim, now time.Time) error {
	if claim.Task.Status != moduleapi.TaskStatusRunning {
		if err := state.ValidateTaskTransition(claim.Task.Status, moduleapi.TaskStatusRunning); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
			SET status = ?, current_stage_key = ?, started_at = COALESCE(started_at, ?), updated_at = ?
			WHERE id = ? AND status = ?`), moduleapi.TaskStatusRunning, claim.Stage.Key, now, now, claim.Task.ID, claim.Task.Status)
		if err != nil {
			return fmt.Errorf("mark claimed task running: %w", err)
		}
		if err := expectOneAffected(result); err != nil {
			return err
		}
		claim.Task.Status = moduleapi.TaskStatusRunning
		claim.Task.CurrentStageKey = stringPointer(claim.Stage.Key)
		claim.Task.StartedAt = &now
		return nil
	}
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET current_stage_key = ?, updated_at = ?
		WHERE id = ? AND status = ?`), claim.Stage.Key, now, claim.Task.ID, moduleapi.TaskStatusRunning)
	if err != nil {
		return fmt.Errorf("update claimed task stage: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return err
	}
	claim.Task.CurrentStageKey = stringPointer(claim.Stage.Key)
	return nil
}

// RequestCancellation 记录协作式取消请求；running Stage 保持不变，以便 worker 调用消费模块所有的 Cancel hook。
func (r *SQLRepository) RequestCancellation(ctx context.Context, taskID uint64, requestedAt time.Time) (taskmodel.Task, error) {
	if taskID == 0 {
		return taskmodel.Task{}, ErrInvalidInput
	}
	if requestedAt.IsZero() {
		requestedAt = time.Now().UTC()
	}
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET cancel_requested_at = COALESCE(cancel_requested_at, ?), updated_at = ?
		WHERE id = ? AND status IN (?, ?, ?, ?, ?)`),
		requestedAt.UTC(), requestedAt.UTC(), taskID,
		moduleapi.TaskStatusPending, moduleapi.TaskStatusReady, moduleapi.TaskStatusScheduled, moduleapi.TaskStatusRunning, moduleapi.TaskStatusNeedsAttention,
	)
	if err != nil {
		return taskmodel.Task{}, fmt.Errorf("request task cancellation: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return taskmodel.Task{}, err
	}
	return r.Get(ctx, taskID)
}

// CancelPendingTask 直接终止尚未领取的 Task，不调用任何 Stage 执行器。
func (r *SQLRepository) CancelPendingTask(ctx context.Context, taskID uint64, finishedAt time.Time, durationMS *int64) error {
	if taskID == 0 || finishedAt.IsZero() {
		return ErrInvalidInput
	}
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, finished_at = ?, duration_ms = ?, updated_at = ?
		WHERE id = ? AND status IN (?, ?, ?)`), moduleapi.TaskStatusCancelled, finishedAt.UTC(), durationMS, finishedAt.UTC(), taskID, moduleapi.TaskStatusPending, moduleapi.TaskStatusReady, moduleapi.TaskStatusScheduled)
	if err != nil {
		return fmt.Errorf("cancel pending task: %w", err)
	}
	return expectOneAffected(result)
}

// CancelUntrackedRunningStage 仅结算已经收到取消请求、但当前 Runtime 没有本地 worker 跟踪的普通 running Stage。
// cancelled 记录的是执行控制权已撤销，不能推断或伪造外部副作用已经成功或被回滚。
func (r *SQLRepository) CancelUntrackedRunningStage(ctx context.Context, taskID uint64, stageID uint64, finishedAt time.Time, stageDurationMS *int64, taskDurationMS *int64) error {
	if taskID == 0 || stageID == 0 || finishedAt.IsZero() {
		return ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin untracked running stage cancellation: %w", err)
	}
	defer rollback(tx)

	stageResult, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, finished_at = ?, duration_ms = ?, updated_at = ?
		WHERE id = ? AND task_id = ? AND status = ?`),
		moduleapi.StageStatusCancelled, finishedAt.UTC(), stageDurationMS, finishedAt.UTC(), stageID, taskID, moduleapi.StageStatusRunning)
	if err != nil {
		return fmt.Errorf("cancel untracked running stage: %w", err)
	}
	if err := expectOneAffected(stageResult); err != nil {
		return err
	}

	taskResult, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, current_stage_key = (SELECT stage_key FROM task_stages WHERE id = ?),
			finished_at = ?, duration_ms = ?, updated_at = ?
		WHERE id = ? AND status = ? AND cancel_requested_at IS NOT NULL`),
		moduleapi.TaskStatusCancelled, stageID, finishedAt.UTC(), taskDurationMS, finishedAt.UTC(), taskID, moduleapi.TaskStatusRunning)
	if err != nil {
		return fmt.Errorf("cancel untracked running task: %w", err)
	}
	if err := expectOneAffected(taskResult); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit untracked running stage cancellation: %w", err)
	}
	return nil
}

// CancelUntrackedRunningStages 原子取消同一 Task 下所有未被本 Runtime 跟踪的 running Stage；其耗时保持未知，避免伪造分布式执行时间。
func (r *SQLRepository) CancelUntrackedRunningStages(ctx context.Context, taskID uint64, finishedAt time.Time, taskDurationMS *int64) (int, error) {
	if taskID == 0 || finishedAt.IsZero() {
		return 0, ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin untracked running stages cancellation: %w", err)
	}
	defer rollback(tx)
	stageResult, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, finished_at = ?, duration_ms = NULL, updated_at = ?
		WHERE task_id = ? AND status = ?`), moduleapi.StageStatusCancelled, finishedAt.UTC(), finishedAt.UTC(), taskID, moduleapi.StageStatusRunning)
	if err != nil {
		return 0, fmt.Errorf("cancel untracked running stages: %w", err)
	}
	count, err := stageResult.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count cancelled untracked running stages: %w", err)
	}
	if count == 0 {
		return 0, ErrStateConflict
	}
	taskResult, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, finished_at = ?, duration_ms = ?, updated_at = ?
		WHERE id = ? AND status = ? AND cancel_requested_at IS NOT NULL`), moduleapi.TaskStatusCancelled, finishedAt.UTC(), taskDurationMS, finishedAt.UTC(), taskID, moduleapi.TaskStatusRunning)
	if err != nil {
		return 0, fmt.Errorf("cancel untracked running coordinated task: %w", err)
	}
	if err := expectOneAffected(taskResult); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit untracked running stages cancellation: %w", err)
	}
	return int(count), nil
}

// RetryStage 将操作员批准的 failed 或 unknown Stage 恢复为 pending。
// 运行时只在父 Task 为 needs_attention 时接受重试，防止绕过父任务生命周期。
func (r *SQLRepository) RetryStage(ctx context.Context, taskID uint64, stageID uint64, retryAt time.Time) (taskmodel.Stage, error) {
	if taskID == 0 || stageID == 0 || retryAt.IsZero() {
		return taskmodel.Stage{}, ErrInvalidInput
	}
	if err := r.resumeRetryableTask(ctx, taskID); err != nil {
		return taskmodel.Stage{}, err
	}
	if err := r.markStagePendingForRetry(ctx, taskID, stageID, retryAt); err != nil {
		return taskmodel.Stage{}, err
	}
	return r.getStage(ctx, taskID, stageID)
}

func (r *SQLRepository) resumeRetryableTask(ctx context.Context, taskID uint64) error {
	task, err := r.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Status != moduleapi.TaskStatusNeedsAttention && task.Status != moduleapi.TaskStatusFailed {
		return ErrStateConflict
	}
	if err := r.TransitionTask(ctx, TaskTransitionInput{TaskID: taskID, From: task.Status, To: moduleapi.TaskStatusRunning}); err != nil {
		return fmt.Errorf("resume task for stage retry: %w", err)
	}
	return nil
}

func (r *SQLRepository) markStagePendingForRetry(ctx context.Context, taskID uint64, stageID uint64, retryAt time.Time) error {
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, next_retry_at = ?, failure_code = NULL, failure_message = NULL,
			finished_at = NULL, duration_ms = NULL, updated_at = ?
		WHERE id = ? AND task_id = ? AND status IN (?, ?)`),
		moduleapi.StageStatusPending, retryAt.UTC(), retryAt.UTC(), stageID, taskID, moduleapi.StageStatusFailed, moduleapi.StageStatusUnknown,
	)
	if err != nil {
		return fmt.Errorf("retry task stage: %w", err)
	}
	return expectOneAffected(result)
}

func (r *SQLRepository) getStage(ctx context.Context, taskID uint64, stageID uint64) (taskmodel.Stage, error) {
	stages, err := r.ListStages(ctx, taskID)
	if err != nil {
		return taskmodel.Stage{}, err
	}
	for _, stage := range stages {
		if stage.ID == stageID {
			return stage, nil
		}
	}
	return taskmodel.Stage{}, ErrNotFound
}

// RescheduleStage 将可重试的 failed 阶段安排为下一次 pending 尝试，不改写尝试计数或终态历史详情。
func (r *SQLRepository) RescheduleStage(ctx context.Context, stageID uint64, retryAt time.Time) error {
	if stageID == 0 || retryAt.IsZero() {
		return ErrInvalidInput
	}
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, next_retry_at = ?, finished_at = NULL, duration_ms = NULL, updated_at = ?
		WHERE id = ? AND status = ?`), moduleapi.StageStatusPending, retryAt.UTC(), retryAt.UTC(), stageID, moduleapi.StageStatusFailed)
	if err != nil {
		return fmt.Errorf("reschedule task stage: %w", err)
	}
	return expectOneAffected(result)
}

// NextEventSequence 返回一个 Task 的下一个只追加事件序列号。
func (r *SQLRepository) NextEventSequence(ctx context.Context, taskID uint64) (int64, error) {
	return r.nextSequence(ctx, "task_events", taskID)
}

// NextLogSequence 返回一个 Task 的下一个只追加日志序列号。
func (r *SQLRepository) NextLogSequence(ctx context.Context, taskID uint64) (int64, error) {
	return r.nextSequence(ctx, "task_logs", taskID)
}

func (r *SQLRepository) nextSequence(ctx context.Context, table string, taskID uint64) (int64, error) {
	if taskID == 0 {
		return 0, ErrInvalidInput
	}
	var sequence int64
	query := `SELECT COALESCE(MAX(sequence), 0) + 1 FROM ` + table + ` WHERE task_id = ?`
	if err := r.db.QueryRowContext(ctx, r.placeholder.rebind(query), taskID).Scan(&sequence); err != nil {
		return 0, fmt.Errorf("next task sequence: %w", err)
	}
	return sequence, nil
}

// RecoverInterruptedStages 将需人工核验的 running Stage 标记为 unknown，因为重启进程无法证明外部副作用是否完成。
// 明确声明幂等的 Stage 才能回到 pending，进入受控重试流程。
func (r *SQLRepository) RecoverInterruptedStages(ctx context.Context, now time.Time) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin interrupted stage recovery: %w", err)
	}
	defer rollback(tx)
	cancelledCount, err := r.cancelInterruptedRequestedStages(ctx, tx, now)
	if err != nil {
		return 0, err
	}
	unknownCount, err := r.markManualRecoveryUnknown(ctx, tx, now)
	if err != nil {
		return 0, err
	}
	if err := r.markRecoveryTasks(ctx, tx, now); err != nil {
		return 0, err
	}
	if err := r.appendRecoveryEvents(ctx, tx, now); err != nil {
		return 0, err
	}
	retriedCount, err := r.rescheduleIdempotentRecovery(ctx, tx, now)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit interrupted stage recovery: %w", err)
	}
	return cancelledCount + unknownCount + retriedCount, nil
}

// SettleExternalReceipt 原子记录已绑定外部回执、更新其最终 Stage 并结算父 Task。
func (r *SQLRepository) SettleExternalReceipt(ctx context.Context, input ExternalReceiptSettlementInput) (ExternalReceiptSettlement, error) {
	if !validExternalReceiptSettlement(input) {
		return ExternalReceiptSettlement{}, ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ExternalReceiptSettlement{}, fmt.Errorf("begin external receipt settlement: %w", err)
	}
	defer rollback(tx)

	if settlement, found, err := r.existingExternalSettlement(ctx, tx, input); err != nil {
		return ExternalReceiptSettlement{}, err
	} else if found {
		return settlement, tx.Commit()
	}
	if err := r.lockSettlementTask(ctx, tx, input.TaskID); err != nil {
		return ExternalReceiptSettlement{}, err
	}
	// Recheck after acquiring the Task lock so concurrent submissions resolve as exact replays instead of conflicting writes.
	if settlement, found, err := r.existingExternalSettlement(ctx, tx, input); err != nil {
		return ExternalReceiptSettlement{}, err
	} else if found {
		return settlement, tx.Commit()
	}

	return r.persistExternalReceiptSettlement(ctx, tx, input)
}

func (r *SQLRepository) persistExternalReceiptSettlement(ctx context.Context, tx *sql.Tx, input ExternalReceiptSettlementInput) (ExternalReceiptSettlement, error) {
	status, stageStatus := settlementStatuses(input.Outcome)
	now := time.Now().UTC()
	if err := r.updateSettlementStage(ctx, tx, input, stageStatus, now); err != nil {
		return ExternalReceiptSettlement{}, err
	}
	if err := r.updateSettlementTask(ctx, tx, input, status, now); err != nil {
		return ExternalReceiptSettlement{}, err
	}
	if err := r.insertExternalReceipt(ctx, tx, input, status, now); err != nil {
		return ExternalReceiptSettlement{}, err
	}
	if err := r.appendExternalReceiptEvent(ctx, tx, input, status, now); err != nil {
		return ExternalReceiptSettlement{}, err
	}
	if err := tx.Commit(); err != nil {
		return ExternalReceiptSettlement{}, fmt.Errorf("commit external receipt settlement: %w", err)
	}
	return ExternalReceiptSettlement{TaskID: input.TaskID, StageID: input.StageID, Status: status}, nil
}

func (r *SQLRepository) existingExternalSettlement(ctx context.Context, tx *sql.Tx, input ExternalReceiptSettlementInput) (ExternalReceiptSettlement, bool, error) {
	existing, found, err := r.findExternalReceipt(ctx, tx, input.TaskID, input.OperationID)
	if err != nil {
		return ExternalReceiptSettlement{}, false, err
	}
	if !found {
		return ExternalReceiptSettlement{}, false, nil
	}
	if !sameExternalReceipt(existing, input) {
		return ExternalReceiptSettlement{}, false, ErrStateConflict
	}
	return ExternalReceiptSettlement{TaskID: existing.TaskID, StageID: existing.StageID, Status: existing.SettledStatus, Idempotent: true}, true, nil
}

func (r *SQLRepository) lockSettlementTask(ctx context.Context, tx *sql.Tx, taskID uint64) error {
	query := `SELECT id FROM tasks WHERE id = ?`
	if r.placeholder == placeholderDollar {
		query += ` FOR UPDATE`
	}
	var id uint64
	if err := tx.QueryRowContext(ctx, r.placeholder.rebind(query), taskID).Scan(&id); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("lock external receipt task: %w", err)
	}
	return nil
}

func (r *SQLRepository) findExternalReceipt(ctx context.Context, tx *sql.Tx, taskID uint64, operationID string) (taskmodel.ExternalReceipt, bool, error) {
	var item taskmodel.ExternalReceipt
	var executorType, protocol, outcome, settledStatus, settledStageStatus string
	var leaseID, failureCode sql.NullString
	err := tx.QueryRowContext(ctx, r.placeholder.rebind(`SELECT id, lease_id, task_id, stage_id, attempt, executor_type, receipt_protocol, operation_id, outcome, failure_code, integrity_sha256, settled_stage_status, settled_task_status, created_at
		FROM task_external_receipts WHERE task_id = ? AND operation_id = ? AND lease_id IS NULL`), taskID, operationID).Scan(
		&item.ID, &leaseID, &item.TaskID, &item.StageID, &item.Attempt, &executorType, &protocol, &item.OperationID, &outcome,
		&failureCode, &item.IntegritySHA256, &settledStageStatus, &settledStatus, &item.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return taskmodel.ExternalReceipt{}, false, nil
	}
	if err != nil {
		return taskmodel.ExternalReceipt{}, false, fmt.Errorf("find external receipt: %w", err)
	}
	item.ExecutorType = moduleapi.StageExecutorType(executorType)
	item.LeaseID = nullableString(leaseID)
	item.Protocol = protocol
	item.Outcome = moduleapi.ExternalReceiptOutcome(outcome)
	item.FailureCode = nullableString(failureCode)
	item.SettledStatus = moduleapi.TaskStatus(settledStatus)
	item.SettledStageStatus = moduleapi.StageStatus(settledStageStatus)
	return item, true, nil
}

func sameExternalReceipt(existing taskmodel.ExternalReceipt, input ExternalReceiptSettlementInput) bool {
	failureCode := ""
	if existing.FailureCode != nil {
		failureCode = *existing.FailureCode
	}
	return existing.StageID == input.StageID && existing.ExecutorType == input.ExecutorType && existing.Protocol == input.Protocol && existing.Outcome == input.Outcome && failureCode == input.FailureCode && existing.IntegritySHA256 == input.IntegritySHA256
}

func settlementStatuses(outcome moduleapi.ExternalReceiptOutcome) (moduleapi.TaskStatus, moduleapi.StageStatus) {
	switch outcome {
	case moduleapi.ExternalReceiptOutcomeSuccess:
		return moduleapi.TaskStatusSuccess, moduleapi.StageStatusSuccess
	case moduleapi.ExternalReceiptOutcomeFailed:
		return moduleapi.TaskStatusFailed, moduleapi.StageStatusFailed
	default:
		return moduleapi.TaskStatusNeedsAttention, moduleapi.StageStatusUnknown
	}
}

func (r *SQLRepository) updateSettlementStage(ctx context.Context, tx *sql.Tx, input ExternalReceiptSettlementInput, status moduleapi.StageStatus, now time.Time) error {
	var failureCode any
	if input.FailureCode != "" {
		failureCode = input.FailureCode
	}
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, failure_code = ?, failure_message = NULL, finished_at = ?, duration_ms = NULL, updated_at = ?
		WHERE id = ? AND task_id = ? AND executor_type = ? AND status IN (?, ?)`),
		status, failureCode, now, now, input.StageID, input.TaskID, input.ExecutorType, moduleapi.StageStatusRunning, moduleapi.StageStatusUnknown)
	if err != nil {
		return fmt.Errorf("settle external receipt stage: %w", err)
	}
	return expectOneAffected(result)
}

func (r *SQLRepository) updateSettlementTask(ctx context.Context, tx *sql.Tx, input ExternalReceiptSettlementInput, status moduleapi.TaskStatus, now time.Time) error {
	var failureCode any
	if input.FailureCode != "" {
		failureCode = input.FailureCode
	}
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, current_stage_key = (SELECT stage_key FROM task_stages WHERE id = ?), failure_code = ?, failure_message = NULL, finished_at = ?, duration_ms = NULL, updated_at = ?
		WHERE id = ? AND status IN (?, ?)`),
		status, input.StageID, failureCode, now, now, input.TaskID, moduleapi.TaskStatusRunning, moduleapi.TaskStatusNeedsAttention)
	if err != nil {
		return fmt.Errorf("settle external receipt task: %w", err)
	}
	return expectOneAffected(result)
}

func (r *SQLRepository) insertExternalReceipt(ctx context.Context, tx *sql.Tx, input ExternalReceiptSettlementInput, status moduleapi.TaskStatus, now time.Time) error {
	var failureCode any
	if input.FailureCode != "" {
		failureCode = input.FailureCode
	}
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`INSERT INTO task_external_receipts (
		task_id, stage_id, executor_type, receipt_protocol, operation_id, outcome, failure_code, integrity_sha256, settled_task_status, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`), input.TaskID, input.StageID, input.ExecutorType, input.Protocol, input.OperationID, input.Outcome, failureCode, input.IntegritySHA256, status, now); err != nil {
		return fmt.Errorf("insert external receipt: %w", err)
	}
	return nil
}

func (r *SQLRepository) appendExternalReceiptEvent(ctx context.Context, tx *sql.Tx, input ExternalReceiptSettlementInput, status moduleapi.TaskStatus, now time.Time) error {
	payload, err := json.Marshal(map[string]string{"operation_id": input.OperationID, "protocol": input.Protocol, "outcome": string(input.Outcome), "task_status": string(status)})
	if err != nil {
		return fmt.Errorf("marshal external receipt event: %w", err)
	}
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`INSERT INTO task_events (task_id, sequence, event_type, payload_json, created_at)
		SELECT ?, COALESCE(MAX(sequence), 0) + 1, ?, ?, ? FROM task_events WHERE task_id = ?`), input.TaskID, taskmodel.EventTypeExternalReceiptSettled, payload, now, input.TaskID); err != nil {
		return fmt.Errorf("append external receipt event: %w", err)
	}
	return nil
}

// cancelInterruptedRequestedStages 结算已请求取消且没有外部回执的中断 Stage；取消不宣称外部副作用已成功或被回滚。
func (r *SQLRepository) cancelInterruptedRequestedStages(ctx context.Context, tx *sql.Tx, now time.Time) (int, error) {
	stageResult, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, finished_at = ?, duration_ms = NULL, updated_at = ?
		WHERE status = ? AND task_id IN (
			SELECT id FROM tasks WHERE status = ? AND cancel_requested_at IS NOT NULL
		) AND NOT EXISTS (
			SELECT 1 FROM task_external_receipts receipt
			WHERE receipt.task_id = task_stages.task_id AND receipt.stage_id = task_stages.id
		) AND NOT EXISTS (
			SELECT 1 FROM task_external_execution_leases lease
			WHERE lease.task_id = task_stages.task_id AND lease.stage_id = task_stages.id
				AND lease.attempt = task_stages.attempt AND lease.state = ?
				AND lease.lease_expires_at > ? AND lease.absolute_deadline_at > ?
		)`), moduleapi.StageStatusCancelled, now.UTC(), now.UTC(), moduleapi.StageStatusRunning, moduleapi.TaskStatusRunning,
		moduleapi.ExternalExecutionLeaseStateClaimed, now.UTC(), now.UTC())
	if err != nil {
		return 0, fmt.Errorf("cancel interrupted requested stages: %w", err)
	}
	cancelledCount, err := stageResult.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count cancelled interrupted stages: %w", err)
	}
	if cancelledCount == 0 {
		return 0, nil
	}
	cancelledTaskIDs, err := r.cancelInterruptedRequestedTasks(ctx, tx, now)
	if err != nil {
		return 0, err
	}
	if err := r.appendInterruptedCancellationEvents(ctx, tx, now, cancelledTaskIDs); err != nil {
		return 0, err
	}
	return int(cancelledCount), nil
}

func (r *SQLRepository) cancelInterruptedRequestedTasks(ctx context.Context, tx *sql.Tx, now time.Time) ([]uint64, error) {
	rows, err := tx.QueryContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, finished_at = ?, duration_ms = NULL, updated_at = ?
		WHERE status = ? AND cancel_requested_at IS NOT NULL
			AND NOT EXISTS (SELECT 1 FROM task_stages WHERE task_stages.task_id = tasks.id AND task_stages.status = ?)
		RETURNING id`),
		moduleapi.TaskStatusCancelled, now.UTC(), now.UTC(), moduleapi.TaskStatusRunning, moduleapi.StageStatusRunning)
	if err != nil {
		return nil, fmt.Errorf("cancel interrupted requested tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var taskIDs []uint64
	for rows.Next() {
		var taskID uint64
		if err := rows.Scan(&taskID); err != nil {
			return nil, fmt.Errorf("scan cancelled interrupted task: %w", err)
		}
		taskIDs = append(taskIDs, taskID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cancelled interrupted tasks: %w", err)
	}
	return taskIDs, nil
}

func (r *SQLRepository) appendInterruptedCancellationEvents(ctx context.Context, tx *sql.Tx, now time.Time, taskIDs []uint64) error {
	if len(taskIDs) == 0 {
		return nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(taskIDs)), ",")
	args := make([]any, 0, len(taskIDs)+interruptedCancellationEventFixedArgs)
	args = append(args, taskmodel.EventTypeCancelled, json.RawMessage(`{}`), now.UTC())
	for _, taskID := range taskIDs {
		args = append(args, taskID)
	}
	query := fmt.Sprintf(`INSERT INTO task_events (task_id, sequence, event_type, payload_json, created_at)
		SELECT task.id, COALESCE((SELECT MAX(event.sequence) FROM task_events event WHERE event.task_id = task.id), 0) + 1, ?, ?, ?
		FROM tasks task
		WHERE task.id IN (%s)`, placeholders)
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(query), args...); err != nil {
		return fmt.Errorf("append interrupted task cancellation events: %w", err)
	}
	return nil
}

func (r *SQLRepository) markManualRecoveryUnknown(ctx context.Context, tx *sql.Tx, now time.Time) (int, error) {
	recoveryMessage := "stage outcome is unknown after task runtime restart"
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, failure_code = ?, failure_message = ?, finished_at = ?,
			duration_ms = NULL,
			updated_at = ?
		WHERE status = ? AND recovery_policy = ?
			AND NOT EXISTS (
				SELECT 1 FROM task_external_execution_leases lease
				WHERE lease.task_id = task_stages.task_id AND lease.stage_id = task_stages.id
					AND lease.attempt = task_stages.attempt AND lease.state = ?
					AND lease.lease_expires_at > ? AND lease.absolute_deadline_at > ?
			)`), moduleapi.StageStatusUnknown, "runner_interrupted", recoveryMessage, now.UTC(), now.UTC(),
		moduleapi.StageStatusRunning, moduleapi.StageRecoveryManualReconcile,
		moduleapi.ExternalExecutionLeaseStateClaimed, now.UTC(), now.UTC())
	if err != nil {
		return 0, fmt.Errorf("mark interrupted stages unknown: %w", err)
	}
	count64, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count interrupted stages: %w", err)
	}
	return int(count64), nil
}

func (r *SQLRepository) markRecoveryTasks(ctx context.Context, tx *sql.Tx, now time.Time) error {
	recoveryMessage := "stage outcome is unknown after task runtime restart"
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, failure_code = ?, failure_message = ?, finished_at = ?,
			duration_ms = NULL,
			updated_at = ?
		WHERE status = ? AND id IN (SELECT DISTINCT task_id FROM task_stages WHERE status = ? AND failure_code = ?)`),
		moduleapi.TaskStatusNeedsAttention, "runner_interrupted", recoveryMessage, now.UTC(), now.UTC(),
		moduleapi.TaskStatusRunning, moduleapi.StageStatusUnknown, "runner_interrupted",
	)
	if err != nil {
		return fmt.Errorf("mark interrupted tasks needs attention: %w", err)
	}
	if _, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("count interrupted tasks: %w", err)
	}
	return nil
}

func (r *SQLRepository) appendRecoveryEvents(ctx context.Context, tx *sql.Tx, now time.Time) error {
	query := fmt.Sprintf(`INSERT INTO task_events (
		task_id, sequence, event_type, payload_json, created_at
	) SELECT DISTINCT stage.task_id,
		COALESCE((SELECT MAX(event.sequence) FROM task_events event WHERE event.task_id = stage.task_id), 0) + 1,
		?, %s, %s
	FROM task_stages stage
	JOIN tasks task ON task.id = stage.task_id
	WHERE task.status = ? AND stage.status = ? AND stage.failure_code = ?
		AND NOT EXISTS (
			SELECT 1 FROM task_events required
			WHERE required.task_id = stage.task_id AND required.event_type = ?
				AND NOT EXISTS (
					SELECT 1 FROM task_events resolved
					WHERE resolved.task_id = required.task_id
						AND resolved.event_type = ?
						AND resolved.sequence > required.sequence
				)
		)`, r.jsonValuePlaceholder(), r.timestampValuePlaceholder())
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(query),
		taskmodel.EventTypeRecoveryRequired, json.RawMessage(`{}`), now.UTC(),
		moduleapi.TaskStatusNeedsAttention, moduleapi.StageStatusUnknown, "runner_interrupted", taskmodel.EventTypeRecoveryRequired, taskmodel.EventTypeRecoveryResolved,
	); err != nil {
		return fmt.Errorf("append interrupted stage recovery events: %w", err)
	}
	return nil
}

func (r *SQLRepository) rescheduleIdempotentRecovery(ctx context.Context, tx *sql.Tx, now time.Time) (int, error) {
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, next_retry_at = ?, failure_code = NULL, failure_message = NULL,
			finished_at = NULL, duration_ms = NULL, updated_at = ?
		WHERE status = ? AND recovery_policy = ?
			AND NOT EXISTS (
				SELECT 1 FROM task_external_execution_leases lease
				WHERE lease.task_id = task_stages.task_id AND lease.stage_id = task_stages.id
					AND lease.attempt = task_stages.attempt AND lease.state = ?
					AND lease.lease_expires_at > ? AND lease.absolute_deadline_at > ?
			)`),
		moduleapi.StageStatusPending, now.UTC(), now.UTC(), moduleapi.StageStatusRunning, moduleapi.StageRecoveryRetryIfIdempotent,
		moduleapi.ExternalExecutionLeaseStateClaimed, now.UTC(), now.UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("reschedule idempotent interrupted stages: %w", err)
	}
	retriedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count idempotent interrupted stages: %w", err)
	}
	return int(retriedCount), nil
}

// normalizeCreateInput 校验并规范化任务及其初始阶段配置。
// 返回规范化后的创建输入；输入无效时返回错误。
func normalizeCreateInput(input CreateInput) (CreateInput, error) {
	if err := normalizeTaskForCreate(&input.Task, len(input.Stages)); err != nil {
		return CreateInput{}, err
	}
	if err := normalizeStagesForCreate(input.Stages); err != nil {
		return CreateInput{}, err
	}
	return input, nil
}

// normalizeTaskForCreate 在创建前校验并规范化 Task；必须有类型、owner 和至少一个阶段，只允许 pending 或 scheduled，scheduled 必须带计划时间。
//
//nolint:cyclop // 创建前必须一次性校验身份、状态和计划时间，再规范化 JSON。
func normalizeTaskForCreate(task *taskmodel.Task, stageCount int) error {
	if task == nil || strings.TrimSpace(string(task.Type)) == "" || strings.TrimSpace(task.Owner.Type) == "" || strings.TrimSpace(task.Owner.ID) == "" || stageCount == 0 {
		return ErrInvalidInput
	}
	if task.Status != moduleapi.TaskStatusPending && task.Status != moduleapi.TaskStatusReady && task.Status != moduleapi.TaskStatusScheduled {
		return ErrInvalidInput
	}
	if task.Status == moduleapi.TaskStatusScheduled && task.ScheduledAt == nil {
		return ErrInvalidInput
	}
	task.Input = normalizeJSON(task.Input)
	task.Metadata = normalizeJSON(task.Metadata)
	task.Plan = normalizeJSON(task.Plan)
	task.State = normalizeJSON(task.State)
	return nil
}

// normalizeStagesForCreate 校验并规范化 Task 初始执行计划中的阶段集合。
func normalizeStagesForCreate(stages []taskmodel.Stage) error {
	seenKeys := make(map[string]struct{}, len(stages))
	seenLegIDs := make(map[string]struct{}, len(stages))
	for index := range stages {
		if err := normalizeStageForCreate(&stages[index], index+1, seenKeys, seenLegIDs); err != nil {
			return err
		}
	}
	return nil
}

// normalizeStageForCreate 验证并规范化创建任务所需的阶段数据，同时检查阶段序号和键的唯一性。
// 无效阶段或重复键返回 ErrInvalidInput；有效阶段的输入和结果 JSON 为空时规范化为 {}。
func normalizeStageForCreate(stage *taskmodel.Stage, sequence int, seenKeys map[string]struct{}, seenLegIDs map[string]struct{}) error {
	if !validInitialStage(stage, sequence) {
		return ErrInvalidInput
	}
	if _, exists := seenKeys[stage.Key]; exists {
		return ErrInvalidInput
	}
	seenKeys[stage.Key] = struct{}{}
	stage.CoordinationGroup, stage.LegID = strings.TrimSpace(stage.CoordinationGroup), strings.TrimSpace(stage.LegID)
	if (stage.CoordinationGroup == "") != (stage.LegID == "") {
		return ErrInvalidInput
	}
	if stage.LegID != "" {
		if _, exists := seenLegIDs[stage.LegID]; exists {
			return ErrInvalidInput
		}
		seenLegIDs[stage.LegID] = struct{}{}
	}
	stage.Input = normalizeJSON(stage.Input)
	stage.Result = normalizeJSON(stage.Result)
	return nil
}

// validInitialStage 判断阶段是否满足指定序号的初始 pending 约束。
func validInitialStage(stage *taskmodel.Stage, sequence int) bool {
	return stage != nil &&
		strings.TrimSpace(stage.Key) != "" &&
		strings.TrimSpace(string(stage.ExecutorType)) != "" &&
		stage.Sequence == sequence &&
		stage.Status == moduleapi.StageStatusPending &&
		stage.Attempt == 0 &&
		stage.MaxAttempts >= 1 &&
		stage.RetryBackoffMS >= 0 &&
		stage.RecoveryPolicy != ""
}

// normalizeJSON 将空 JSON 值规范化为空对象。
func normalizeJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return value
}

// normalizeLimit 将分页限制规范化到允许的范围内。
//
// 小于等于零的值使用默认分页限制，超过最大值的值使用最大分页限制。
func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultPageLimit
	}
	if limit > maxPageLimit {
		return maxPageLimit
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

// expectOneAffected 要求数据库操作恰好影响一行；影响数不符时返回状态冲突，防止静默覆盖并发更新。
func expectOneAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrStateConflict
	}
	return nil
}

// rollback 回滚事务并忽略回滚错误。
func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}

// closeRows 关闭数据库查询结果集并忽略关闭错误。
func closeRows(rows *sql.Rows) {
	_ = rows.Close()
}

// validEventType 判断事件类型是否属于任务仓储支持的历史事实集合。
func validEventType(value taskmodel.EventType) bool {
	switch value {
	case taskmodel.EventTypeCreated, taskmodel.EventTypeCancelRequested, taskmodel.EventTypeCancelled, taskmodel.EventTypeRetryRequested, taskmodel.EventTypeRetryScheduled, taskmodel.EventTypeRecoveryRequired, taskmodel.EventTypeRecoveryResolved, taskmodel.EventTypeExternalReceiptSettled:
		return true
	default:
		return false
	}
}

func validExternalReceiptSettlement(input ExternalReceiptSettlementInput) bool {
	if !externalReceiptSettlementIdentityValid(input) || !lowercaseReceiptSHA256(input.IntegritySHA256) {
		return false
	}
	return externalReceiptSettlementOutcomeValid(input)
}

func externalReceiptSettlementIdentityValid(input ExternalReceiptSettlementInput) bool {
	return input.TaskID != 0 && input.StageID != 0 && strings.TrimSpace(string(input.ExecutorType)) != "" && strings.TrimSpace(input.Protocol) != "" && strings.TrimSpace(input.OperationID) != "" && len(input.Protocol) <= 128 && len(input.OperationID) <= 256 && len(input.FailureCode) <= 128
}

func lowercaseReceiptSHA256(value string) bool {
	if len(value) != externalReceiptSHA256Length {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func externalReceiptSettlementOutcomeValid(input ExternalReceiptSettlementInput) bool {
	switch input.Outcome {
	case moduleapi.ExternalReceiptOutcomeSuccess:
		return input.FailureCode == ""
	case moduleapi.ExternalReceiptOutcomeFailed, moduleapi.ExternalReceiptOutcomeNeedsAttention:
		return strings.TrimSpace(input.FailureCode) != ""
	default:
		return false
	}
}

// validLogStream 判断 value 是否标识受支持的日志流。
func validLogStream(value string) bool {
	return value == "stdout" || value == "stderr" || value == "system"
}

// validLogLevel 判断 value 是否属于受支持的日志级别。
func validLogLevel(value string) bool {
	return value == "info" || value == "warn" || value == "error"
}
