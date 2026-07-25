package event

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	deliveryPending    = "pending"
	deliveryProcessing = "processing"
	deliveryDelivered  = "delivered"
	deliveryFailed     = "failed"
)

// OutboxStore 持有 durable event 和按 consumer 独立恢复的 delivery 状态。
//
// Append 在一个数据库事务中写入 event_outbox 与 event_deliveries；业务模块若需要
// 与自己的事实记录共用事务，应在该业务事务中调用同等 SQL，而不是在提交后补写事件。
type OutboxStore interface {
	Append(context.Context, Event, []string) (Receipt, error)
	Claim(context.Context, string, time.Time, time.Duration, int) ([]ClaimedDelivery, error)
	Complete(context.Context, ClaimedDelivery) error
	Retry(context.Context, ClaimedDelivery, time.Time, error) error
	Fail(context.Context, ClaimedDelivery, error) error
}

// TransactionalOutboxStore 支持把 Outbox 写入调用方已有的业务事务。
type TransactionalOutboxStore interface {
	OutboxStore
	AppendTx(context.Context, *sql.Tx, Event, []string) error
}

// ClaimedDelivery 是被单个 runtime 实例以租约占有的 consumer 投递。
type ClaimedDelivery struct {
	Event      Event
	ConsumerID string
	Attempt    int
	claimOwner string
}

// SQLRepository 是 PostgreSQL Outbox 的 database/sql 实现。
type SQLRepository struct{ db *sql.DB }

// NewSQLRepository 创建使用运行时共享 SQL 连接池的 Outbox store。
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("event outbox database is required")
	}
	return &SQLRepository{db: db}, nil
}

// Append 原子写入 event 及其当前已注册 consumer 的投递记录。
func (r *SQLRepository) Append(ctx context.Context, event Event, consumerIDs []string) (Receipt, error) {
	if r == nil || r.db == nil {
		return Receipt{}, errors.New("event outbox repository is unavailable")
	}
	if len(consumerIDs) == 0 {
		return Receipt{}, fmt.Errorf("%w: %s", ErrNoHandlers, event.Type)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Receipt{}, fmt.Errorf("begin event outbox transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := r.AppendTx(ctx, tx, event, consumerIDs); err != nil {
		return Receipt{}, err
	}
	if err := tx.Commit(); err != nil {
		return Receipt{}, fmt.Errorf("commit event outbox transaction: %w", err)
	}
	return Receipt{EventID: event.ID, Delivery: DeliveryDurable}, nil
}

// AppendTx 将 event 与 consumer delivery 写入调用方提供的事务。
//
// 业务 owner 可以把自己的状态变更和本方法置于同一事务，从而避免业务事实与
// Outbox 的 dual-write 缺口；consumerIDs 必须来自 Runtime 冻结后的 handler 注册表。
func (r *SQLRepository) AppendTx(ctx context.Context, tx *sql.Tx, event Event, consumerIDs []string) error {
	if r == nil || r.db == nil {
		return errors.New("event outbox repository is unavailable")
	}
	if tx == nil {
		return errors.New("event outbox transaction is required")
	}
	if len(consumerIDs) == 0 {
		return fmt.Errorf("%w: %s", ErrNoHandlers, event.Type)
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO event_outbox (
  event_id, event_type, version, source, payload, metadata, occurred_at, created_at,
  correlation_id, causation_id, idempotency_key
) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, $9, $10, $11)
ON CONFLICT (event_id) DO NOTHING`,
		event.ID, event.Type, event.Version, event.Source, []byte(event.Payload), nullableJSON(event.Metadata),
		event.OccurredAt, event.CreatedAt, nullString(event.CorrelationID), nullString(event.CausationID), nullString(event.IdempotencyKey),
	)
	if err != nil {
		return fmt.Errorf("append event outbox row: %w", err)
	}
	for _, consumerID := range consumerIDs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO event_deliveries (event_id, consumer_id, status, available_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $4, $4)
ON CONFLICT (event_id, consumer_id) DO NOTHING`, event.ID, consumerID, deliveryPending, event.CreatedAt); err != nil {
			return fmt.Errorf("append event delivery for %s: %w", consumerID, err)
		}
	}
	return nil
}

// Claim 以 SKIP LOCKED 获取可恢复投递，并把它们租给本 runtime 实例。
func (r *SQLRepository) Claim(ctx context.Context, owner string, now time.Time, lease time.Duration, limit int) ([]ClaimedDelivery, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("event outbox repository is unavailable")
	}
	if limit <= 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `
WITH candidates AS (
  SELECT d.event_id, d.consumer_id
  FROM event_deliveries d
  WHERE (d.status = $1 AND d.available_at <= $2)
     OR (d.status = $3 AND d.lease_expires_at <= $2)
  ORDER BY d.available_at, d.created_at
  FOR UPDATE SKIP LOCKED
  LIMIT $4
), claimed AS (
  UPDATE event_deliveries d
  SET status = $3, attempt_count = d.attempt_count + 1, lease_owner = $5,
      lease_expires_at = $6, updated_at = $2
  FROM candidates c
  WHERE d.event_id = c.event_id AND d.consumer_id = c.consumer_id
  RETURNING d.event_id, d.consumer_id, d.attempt_count
)
SELECT o.event_id, o.event_type, o.version, o.source, o.payload, o.metadata, o.occurred_at,
       o.created_at, o.correlation_id, o.causation_id, o.idempotency_key,
       c.consumer_id, c.attempt_count
FROM claimed c JOIN event_outbox o ON o.event_id = c.event_id`,
		deliveryPending, now, deliveryProcessing, limit, owner, now.Add(lease),
	)
	if err != nil {
		return nil, fmt.Errorf("claim event deliveries: %w", err)
	}
	defer func() { _ = rows.Close() }()
	claimed := make([]ClaimedDelivery, 0, limit)
	for rows.Next() {
		var item ClaimedDelivery
		var eventType string
		var version int
		var metadata []byte
		var correlationID, causationID, idempotencyKey sql.NullString
		if err := rows.Scan(&item.Event.ID, &eventType, &version, &item.Event.Source, &item.Event.Payload, &metadata,
			&item.Event.OccurredAt, &item.Event.CreatedAt, &correlationID, &causationID, &idempotencyKey,
			&item.ConsumerID, &item.Attempt); err != nil {
			return nil, fmt.Errorf("scan claimed event delivery: %w", err)
		}
		if version <= 0 || version > math.MaxUint16 {
			return nil, fmt.Errorf("claimed event %s has invalid version %d", item.Event.ID, version)
		}
		item.Event.Type = Type(eventType)
		item.Event.Version = uint16(version)
		item.Event.Metadata = append(item.Event.Metadata[:0], metadata...)
		item.Event.CorrelationID = correlationID.String
		item.Event.CausationID = causationID.String
		item.Event.IdempotencyKey = idempotencyKey.String
		item.claimOwner = owner
		claimed = append(claimed, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed event deliveries: %w", err)
	}
	return claimed, nil
}

// Complete 只接受仍属于当前 claim 的 processing delivery，避免过期 worker 覆盖新 owner 的结果。
func (r *SQLRepository) Complete(ctx context.Context, delivery ClaimedDelivery) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE event_deliveries
SET status = $1, delivered_at = NOW(), lease_owner = NULL, lease_expires_at = NULL, updated_at = NOW()
WHERE event_id = $2 AND consumer_id = $3 AND status = $4 AND attempt_count = $5 AND lease_owner = $6`,
		deliveryDelivered, delivery.Event.ID, delivery.ConsumerID, deliveryProcessing, delivery.Attempt, delivery.claimOwner)
	if err != nil {
		return fmt.Errorf("complete event delivery: %w", err)
	}
	if changed, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("inspect completed event delivery: %w", err)
	} else if changed != 1 {
		return ErrClaimLost
	}
	return nil
}

// Retry 释放失败投递，下一次 claim 会在 availableAt 后重新执行同一 consumer。
func (r *SQLRepository) Retry(ctx context.Context, delivery ClaimedDelivery, availableAt time.Time, cause error) error {
	message := ""
	if cause != nil {
		message = cause.Error()
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE event_deliveries
SET status = $1, available_at = $2, lease_owner = NULL, lease_expires_at = NULL,
    last_error = $3, updated_at = NOW()
WHERE event_id = $4 AND consumer_id = $5 AND status = $6 AND attempt_count = $7 AND lease_owner = $8`,
		deliveryPending, availableAt, message, delivery.Event.ID, delivery.ConsumerID, deliveryProcessing, delivery.Attempt, delivery.claimOwner)
	if err != nil {
		return fmt.Errorf("retry event delivery: %w", err)
	}
	if changed, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("inspect retried event delivery: %w", err)
	} else if changed != 1 {
		return ErrClaimLost
	}
	return nil
}

// Fail 将仍归当前 claim 的 delivery 记录为最终失败，后续轮询不会再次执行它。
func (r *SQLRepository) Fail(ctx context.Context, delivery ClaimedDelivery, cause error) error {
	message := ""
	if cause != nil {
		message = cause.Error()
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE event_deliveries
SET status = $1, lease_owner = NULL, lease_expires_at = NULL, last_error = $2,
    failed_at = NOW(), updated_at = NOW()
WHERE event_id = $3 AND consumer_id = $4 AND status = $5 AND attempt_count = $6 AND lease_owner = $7`,
		deliveryFailed, message, delivery.Event.ID, delivery.ConsumerID, deliveryProcessing, delivery.Attempt, delivery.claimOwner)
	if err != nil {
		return fmt.Errorf("fail event delivery: %w", err)
	}
	if changed, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("inspect failed event delivery: %w", err)
	} else if changed != 1 {
		return ErrClaimLost
	}
	return nil
}

func nullableJSON(value []byte) any {
	if len(value) == 0 {
		return []byte(`{}`)
	}
	return value
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
