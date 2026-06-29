package scheduler

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	defaultRunListLimit  = 20
	defaultTaskListLimit = 20
	maxTaskListLimit     = 100
	maxSQLRunID          = 1<<63 - 1
)

// SQLJobDefinitionRepository stores scheduler job definitions in SQL.
type SQLJobDefinitionRepository struct {
	db *sql.DB
}

// NewSQLJobDefinitionRepository 创建一个基于 SQL 的作业定义仓库。若 db 为空，则返回错误。
func NewSQLJobDefinitionRepository(db *sql.DB) (*SQLJobDefinitionRepository, error) {
	if db == nil {
		return nil, errors.New("scheduler job definition repository requires a non-nil sql db")
	}
	return &SQLJobDefinitionRepository{db: db}, nil
}

func (r *SQLJobDefinitionRepository) ensureAvailable() error {
	if r == nil || r.db == nil {
		return errors.New("scheduler job definition repository is unavailable")
	}
	return nil
}

// SQLTaskRepository stores scheduled task instances in SQL.
type SQLTaskRepository struct {
	db *sql.DB
}

// NewSQLTaskRepository builds a SQL-backed task repository.
// 当 db 为空时返回错误；否则返回已初始化的仓库。
func NewSQLTaskRepository(db *sql.DB) (*SQLTaskRepository, error) {
	if db == nil {
		return nil, errors.New("scheduler task repository requires a non-nil sql db")
	}
	return &SQLTaskRepository{db: db}, nil
}

func (r *SQLTaskRepository) ensureTaskAvailable() error {
	if r == nil || r.db == nil {
		return errors.New("scheduler task repository is unavailable")
	}
	return nil
}

// SQLRunRepository stores scheduler run history in SQL.
type SQLRunRepository struct {
	db *sql.DB
}

// NewSQLRunRepository builds a SQL-backed scheduler run repository.
// db 不能为空。
func NewSQLRunRepository(db *sql.DB) (*SQLRunRepository, error) {
	if db == nil {
		return nil, errors.New("scheduler run repository requires a non-nil sql db")
	}
	return &SQLRunRepository{db: db}, nil
}

func (r *SQLRunRepository) ensureAvailable() error {
	if r == nil || r.db == nil {
		return errors.New("scheduler run repository is unavailable")
	}
	return nil
}

func (r *SQLRunRepository) sqlRunID(id uint64) (int64, error) {
	if err := r.ensureAvailable(); err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, errors.New("scheduler run id is required")
	}
	if id > maxSQLRunID {
		return 0, errors.New("scheduler run id is too large")
	}
	sqlID, err := strconv.ParseInt(strconv.FormatUint(id, 10), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("convert scheduler run id: %w", err)
	}
	return sqlID, nil
}

func validateRunFinish(status RunStatus, finishedAt time.Time) error {
	if status == "" {
		return errors.New("scheduler run status is required")
	}
	if finishedAt.IsZero() {
		return errors.New("scheduler run finished_at is required")
	}
	return nil
}

// normalizeRunListQuery 验证并规范化运行列表查询条件。
// 当 TaskKey 为空时返回错误；当 Limit 小于等于 0 时使用默认值；当 Offset 小于 0 时将其设为 0。
func normalizeRunListQuery(query RunListQuery) (RunListQuery, error) {
	if query.TaskKey == "" {
		return RunListQuery{}, errors.New("scheduler run task key is required")
	}
	if query.Limit <= 0 {
		query.Limit = defaultRunListLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query, nil
}

// normalizeTaskListQuery 规范化任务列表查询参数，补齐默认分页并限制偏移量范围。
// 返回处理后的查询参数。
func normalizeTaskListQuery(query TaskListQuery) TaskListQuery {
	if query.Limit <= 0 {
		query.Limit = defaultTaskListLimit
	} else if query.Limit > maxTaskListLimit {
		query.Limit = maxTaskListLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query
}

type rowScanner interface {
	Scan(dest ...any) error
}

type rowsScanner interface {
	rowScanner
	Close() error
	Err() error
	Next() bool
}

// collectRows 迭代 rows 并将扫描结果收集到切片中。
//
// 当扫描过程或行迭代出现错误时返回相应错误；迭代结束后会检查 rows 的错误状态。
//
// 返回收集到的元素切片，或在发生错误时返回 nil 和错误。
func collectRows[T any](rows rowsScanner, scan func(rowScanner) (T, error), iterateLabel string) ([]T, error) {
	defer func() {
		_ = rows.Close()
	}()

	items := make([]T, 0)
	for rows.Next() {
		item, scanErr := scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", iterateLabel, err)
	}
	return items, nil
}

// taskRunIDFromSQL 将数据库中的任务运行 ID 转换为 uint64。
// 当 ID 无效或转换失败时返回错误。
func taskRunIDFromSQL(id int64) (uint64, error) {
	if id <= 0 {
		return 0, errors.New("scheduler id from database is invalid")
	}
	runID, err := strconv.ParseUint(strconv.FormatInt(id, 10), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("convert scheduler id from database: %w", err)
	}
	return runID, nil
}

// mapScheduledTaskWriteError 将调度任务写入冲突错误映射为领域错误。
// 当错误对应任务键或任务标题的唯一约束冲突时，返回相应的冲突错误；
// 其他错误原样返回。
//
// @returns
// 映射后的错误，或原始错误。
func mapScheduledTaskWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return mapScheduledTaskUniqueConstraint(pgErr.ConstraintName, err)
	}
	message := err.Error()
	if !isUniqueConstraintErrorText(message) {
		return err
	}
	if containsTaskKeyConstraint(message) {
		return ErrTaskKeyConflict
	}
	if containsTaskTitleConstraint(message) {
		return ErrTaskTitleConflict
	}
	return err
}

func mapScheduledTaskUniqueConstraint(constraintName string, fallback error) error {
	switch constraintName {
	case "scheduled_tasks_task_key_key", "scheduled_tasks_task_key_live_key":
		return ErrTaskKeyConflict
	case "scheduled_tasks_title_active_key", "scheduled_tasks_title_live_key":
		return ErrTaskTitleConflict
	default:
		return fallback
	}
}

func containsTaskKeyConstraint(message string) bool {
	return strings.Contains(message, "scheduled_tasks_task_key_key") ||
		strings.Contains(message, "scheduled_tasks_task_key_live_key")
}

func containsTaskTitleConstraint(message string) bool {
	return strings.Contains(message, "scheduled_tasks_title_active_key") ||
		strings.Contains(message, "scheduled_tasks_title_live_key")
}

// isUniqueConstraintErrorText 判断消息是否包含唯一约束冲突文本。
// 当消息包含“duplicate key”或“violates unique constraint”时，返回 true，否则返回 false。
func isUniqueConstraintErrorText(message string) bool {
	return strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "violates unique constraint")
}

// requireAffectedScheduledTask 检查写入操作是否影响了至少一条调度任务记录。
//
// 当未影响任何行时返回 ErrTaskNotFound；当读取受影响行数失败时返回带上下文的错误。
func requireAffectedScheduledTask(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read scheduled task affected rows: %w", err)
	}
	if affected == 0 {
		return ErrTaskNotFound
	}
	return nil
}
