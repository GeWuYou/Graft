package store

import (
	"context"
	"fmt"
	"time"

	"graft/server/internal/moduleapi"
)

// PromoteScheduledTasks makes due scheduled Tasks visible to workers as ready in one idempotent batch.
func (r *SQLRepository) PromoteScheduledTasks(ctx context.Context, now time.Time, limit int) (int, error) {
	if r == nil || r.db == nil || now.IsZero() || limit <= 0 {
		return 0, ErrInvalidInput
	}
	var result interface{ RowsAffected() (int64, error) }
	var err error
	if r.placeholder == placeholderQuestion {
		result, err = r.db.ExecContext(ctx, `UPDATE tasks SET status = ?, updated_at = ? WHERE status = ? AND id IN (SELECT id FROM tasks WHERE status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ? ORDER BY scheduled_at ASC, id ASC LIMIT ?)`, moduleapi.TaskStatusReady, now.UTC(), moduleapi.TaskStatusScheduled, moduleapi.TaskStatusScheduled, now.UTC(), limit)
	} else {
		result, err = r.db.ExecContext(ctx, r.placeholder.rebind(`WITH due AS (SELECT id FROM tasks WHERE status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ? ORDER BY scheduled_at ASC, id ASC LIMIT ?) UPDATE tasks SET status = ?, updated_at = ? FROM due WHERE tasks.id = due.id AND tasks.status = ?`), moduleapi.TaskStatusScheduled, now.UTC(), limit, moduleapi.TaskStatusReady, now.UTC(), moduleapi.TaskStatusScheduled)
	}
	if err != nil {
		return 0, fmt.Errorf("promote scheduled tasks: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count promoted scheduled tasks: %w", err)
	}
	return int(count), nil
}
