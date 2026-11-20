package store

// recovery_repository.go 拥有被中断幂等阶段的自动恢复仓储路径：预算内原子恢复，预算耗尽稳定收敛。
// manual_reconcile 阶段与显式 needs_attention 结论不进入这些路径，保持人工恢复事实。

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
)

const (
	// RecoveryBudgetExhaustedFailureCode 是自动恢复预算耗尽的稳定失败码；它与外部执行结算码共用同一命名约束。
	RecoveryBudgetExhaustedFailureCode = "auto_recovery_budget_exhausted"
	// recoveryBudgetExhaustedMessage 是预算耗尽收敛的稳定脱敏摘要；不携带原始 Provider 诊断或本机路径。
	recoveryBudgetExhaustedMessage = "automatic recovery budget exhausted for interrupted stage"
	// interruptedRecoveryFailureCode 是租约过期中断写入 Stage 的稳定失败码，也是自动恢复唯一认可的恢复信号。
	interruptedRecoveryFailureCode = "external_execution_lease_expired"
)

// RecoveryBudgetExhaustionInput 描述一次预算耗尽收敛的 Task 与 Stage 绑定。
type RecoveryBudgetExhaustionInput struct {
	TaskID  uint64
	StageID uint64
	Now     time.Time
}

// ListInterruptedRecoveryCandidates 返回可自动恢复的被中断幂等阶段候选：
// 租约过期导致的 unknown 阶段（预算内恢复或预算耗尽收敛），以及启动恢复遗留、已无法领取的 pending 超预算阶段。
// manual_reconcile 阶段、显式 needs_attention 结论和已请求取消的 Task 都不是候选。
func (r *SQLRepository) ListInterruptedRecoveryCandidates(ctx context.Context, limit int) ([]StageClaim, error) {
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+taskColumnsFor("task")+`, `+stageColumnsFor("stage")+`
		FROM task_stages stage
		JOIN tasks task ON task.id = stage.task_id
		WHERE task.cancel_requested_at IS NULL
			AND stage.recovery_policy = ?
			AND stage.external_execution = true
			AND (
				(stage.status = ? AND stage.failure_code = ? AND task.status = ?)
				OR (stage.status = ? AND stage.attempt >= stage.max_attempts AND task.status = ?)
			)
		ORDER BY task.created_at ASC, stage.sequence ASC, stage.id ASC
		LIMIT ?`),
		moduleapi.StageRecoveryRetryIfIdempotent,
		moduleapi.StageStatusUnknown, interruptedRecoveryFailureCode, moduleapi.TaskStatusNeedsAttention,
		moduleapi.StageStatusPending, moduleapi.TaskStatusRunning,
		normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list interrupted recovery candidates: %w", err)
	}
	defer closeRows(rows)
	items := make([]StageClaim, 0)
	for rows.Next() {
		item, scanErr := scanStageClaim(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan interrupted recovery candidate: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate interrupted recovery candidates: %w", err)
	}
	return items, nil
}

// ResumeInterruptedRecovery 在同一事务内恢复 needs_attention 父 Task 及其全部可自动重试的租约中断幂等阶段，
// 并原子追加一条 recovery_resolved 事件；没有任何可恢复阶段时回滚整个恢复，保留人工恢复事实。
// 该路径复用 RetryStage 的状态机语义（Task needs_attention -> running，Stage unknown -> pending 并清除失败诊断），
// 但不推进尝试计数；外部领取路径仍按 attempt < max_attempts 有界发放。
func (r *SQLRepository) ResumeInterruptedRecovery(ctx context.Context, taskID uint64, retryAt time.Time) (int, error) {
	if taskID == 0 || retryAt.IsZero() {
		return 0, ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin interrupted recovery resume: %w", err)
	}
	defer rollback(tx)
	if err := r.resumeRetryableTask(ctx, tx, taskID); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, next_retry_at = ?, failure_code = NULL, failure_message = NULL,
			failure_stage = NULL, failure_class = NULL,
			finished_at = NULL, duration_ms = NULL, updated_at = ?
		WHERE task_id = ? AND status = ? AND recovery_policy = ? AND failure_code = ?
			AND attempt < max_attempts`),
		moduleapi.StageStatusPending, retryAt.UTC(), retryAt.UTC(), taskID,
		moduleapi.StageStatusUnknown, moduleapi.StageRecoveryRetryIfIdempotent, interruptedRecoveryFailureCode)
	if err != nil {
		return 0, fmt.Errorf("resume interrupted recovery stages: %w", err)
	}
	count64, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count resumed interrupted recovery stages: %w", err)
	}
	if count64 == 0 {
		return 0, ErrStateConflict
	}
	if err := r.appendRecoveryResolvedEvent(ctx, tx, taskID, int(count64), retryAt, time.Now().UTC()); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit interrupted recovery resume: %w", err)
	}
	return int(count64), nil
}

// appendRecoveryResolvedEvent 在恢复事务内追加一条 recovery_resolved 事件，使恢复事实与状态迁移原子可见。
func (r *SQLRepository) appendRecoveryResolvedEvent(ctx context.Context, tx *sql.Tx, taskID uint64, resumedStages int, retryAt time.Time, now time.Time) error {
	payload, err := json.Marshal(map[string]any{"resumed_stages": resumedStages, "retry_at": retryAt.UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return fmt.Errorf("marshal interrupted recovery resolved event: %w", err)
	}
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`INSERT INTO task_events (task_id, sequence, event_type, payload_json, created_at)
		SELECT ?, COALESCE(MAX(sequence), 0) + 1, ?, `+r.jsonValuePlaceholder()+`, `+r.timestampValuePlaceholder()+` FROM task_events WHERE task_id = ?`),
		taskID, taskmodel.EventTypeRecoveryResolved, payload, now, taskID); err != nil {
		return fmt.Errorf("append interrupted recovery resolved event: %w", err)
	}
	return nil
}

// ConvergeRecoveryBudgetExhausted 把超出重试预算的被中断幂等阶段收敛为稳定诊断的 unknown/needs_attention：
// 已无法领取的 pending 阶段回到 unknown，unknown 阶段仅重写预算耗尽诊断；父任务收敛为 needs_attention。
// 未解决的 recovery_required 事实保持恰好一条，事件幂等由既有未解决守卫保证。
func (r *SQLRepository) ConvergeRecoveryBudgetExhausted(ctx context.Context, input RecoveryBudgetExhaustionInput) error {
	if input.TaskID == 0 || input.StageID == 0 || input.Now.IsZero() {
		return ErrInvalidInput
	}
	now := input.Now.UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin recovery budget exhaustion: %w", err)
	}
	defer rollback(tx)
	if err := r.markRecoveryBudgetExhaustedStage(ctx, tx, input, now); err != nil {
		return err
	}
	if err := r.markRecoveryBudgetExhaustedTask(ctx, tx, input, now); err != nil {
		return err
	}
	if err := r.appendBudgetExhaustedRecoveryRequired(ctx, tx, input, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit recovery budget exhaustion: %w", err)
	}
	return nil
}

// markRecoveryBudgetExhaustedStage 只在阶段仍处于 pending/unknown 且确已超预算时写入稳定诊断；
// 迟到的回执结算或并发领取会命中零行，按状态冲突回滚以保留结算事实。
func (r *SQLRepository) markRecoveryBudgetExhaustedStage(ctx context.Context, tx *sql.Tx, input RecoveryBudgetExhaustionInput, now time.Time) error {
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, failure_code = ?, failure_stage = NULL, failure_class = ?, failure_message = ?,
			finished_at = COALESCE(finished_at, ?), duration_ms = NULL, updated_at = ?
		WHERE id = ? AND task_id = ? AND recovery_policy = ? AND attempt >= max_attempts
			AND status IN (?, ?)`),
		moduleapi.StageStatusUnknown, RecoveryBudgetExhaustedFailureCode, string(moduleapi.ExecutionFailureClassInternal),
		recoveryBudgetExhaustedMessage, now, now, input.StageID, input.TaskID,
		moduleapi.StageRecoveryRetryIfIdempotent, moduleapi.StageStatusPending, moduleapi.StageStatusUnknown)
	if err != nil {
		return fmt.Errorf("mark recovery budget exhausted stage: %w", err)
	}
	return expectOneAffected(result)
}

// markRecoveryBudgetExhaustedTask 把父任务收敛为 needs_attention 并写入同一组稳定诊断；
// 任务已被操作员取消或进入其他终态时命中零行，按状态冲突回滚。
func (r *SQLRepository) markRecoveryBudgetExhaustedTask(ctx context.Context, tx *sql.Tx, input RecoveryBudgetExhaustionInput, now time.Time) error {
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, failure_code = ?, failure_stage = NULL, failure_class = ?, failure_message = ?,
			finished_at = ?, duration_ms = NULL, updated_at = ?
		WHERE id = ? AND status IN (?, ?)`),
		moduleapi.TaskStatusNeedsAttention, RecoveryBudgetExhaustedFailureCode, string(moduleapi.ExecutionFailureClassInternal),
		recoveryBudgetExhaustedMessage, now, now, input.TaskID,
		moduleapi.TaskStatusRunning, moduleapi.TaskStatusNeedsAttention)
	if err != nil {
		return fmt.Errorf("mark recovery budget exhausted task: %w", err)
	}
	return expectOneAffected(result)
}

// appendBudgetExhaustedRecoveryRequired 在缺少未解决 recovery_required 事实时补记一条预算耗尽恢复事件；
// 租约过期已写入的 recovery_required 不会被重复追加。
func (r *SQLRepository) appendBudgetExhaustedRecoveryRequired(ctx context.Context, tx *sql.Tx, input RecoveryBudgetExhaustionInput, now time.Time) error {
	payload, err := json.Marshal(map[string]any{"stage_id": input.StageID, "reason": RecoveryBudgetExhaustedFailureCode})
	if err != nil {
		return fmt.Errorf("marshal recovery budget exhausted event: %w", err)
	}
	query := fmt.Sprintf(`INSERT INTO task_events (task_id, sequence, event_type, payload_json, created_at)
		SELECT ?, COALESCE((SELECT MAX(sequence) FROM task_events WHERE task_id = ?), 0) + 1, ?, %s, %s
		WHERE NOT EXISTS (
			SELECT 1 FROM task_events required
			WHERE required.task_id = ? AND required.event_type = ?
				AND NOT EXISTS (
					SELECT 1 FROM task_events resolved
					WHERE resolved.task_id = required.task_id
						AND resolved.event_type = ?
						AND resolved.sequence > required.sequence
				)
		)`, r.jsonValuePlaceholder(), r.timestampValuePlaceholder())
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(query),
		input.TaskID, input.TaskID, taskmodel.EventTypeRecoveryRequired, payload, now,
		input.TaskID, taskmodel.EventTypeRecoveryRequired, taskmodel.EventTypeRecoveryResolved); err != nil {
		return fmt.Errorf("append recovery budget exhausted event: %w", err)
	}
	return nil
}
