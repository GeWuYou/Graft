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

// NewSQLJobDefinitionRepository creates the SQL-backed job definition repository.
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

// NewSQLTaskRepository creates the SQL-backed scheduled task repository.
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

// NewSQLRunRepository creates the SQL-backed run-history repository.
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

func isUniqueConstraintErrorText(message string) bool {
	return strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "violates unique constraint")
}

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
