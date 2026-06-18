// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	defaultListLimit          = 20
	maxListLimit              = 100
	placeholderGrowthEstimate = 8
)

// SQLRepository persists Notification Center state in module-owned SQL tables.
type SQLRepository struct {
	db          *sql.DB
	placeholder placeholderStyle
}

// NewSQLRepository creates a SQL-backed notification repository.
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("notification repository requires a non-nil sql db")
	}

	return &SQLRepository{db: db, placeholder: detectPlaceholderStyle(db)}, nil
}

// CreateEvent inserts one notification fact. When dedupe_key is present, an existing event is reused.
func (r *SQLRepository) CreateEvent(ctx context.Context, input CreateEventInput) (Event, bool, error) {
	if err := r.ensureReady(); err != nil {
		return Event{}, false, err
	}
	input, err := validateCreateEventInput(input)
	if err != nil {
		return Event{}, false, err
	}

	if event, found, err := r.findExistingDedupedEvent(ctx, input.DedupeKey); err != nil || found {
		return event, found, err
	}

	return r.createNewEvent(ctx, input)
}

func (r *SQLRepository) ensureReady() error {
	if r == nil || r.db == nil {
		return errors.New("notification repository is unavailable")
	}
	return nil
}

func validateCreateEventInput(input CreateEventInput) (CreateEventInput, error) {
	input = normalizeEventInput(input)
	if (input.TitleKey == "" && input.Title == "") ||
		(input.MessageKey == "" && input.Message == "") ||
		input.Severity == "" ||
		input.Category == "" ||
		input.SourceModule == "" ||
		input.EventType == "" ||
		input.OccurredAt.IsZero() {
		return CreateEventInput{}, ErrInvalidInput
	}
	return input, nil
}

func (r *SQLRepository) createNewEvent(ctx context.Context, input CreateEventInput) (Event, bool, error) {
	event, err := r.insertEvent(ctx, input, time.Now().UTC())
	if err == nil {
		return event, false, nil
	}
	if input.DedupeKey == "" {
		return Event{}, false, fmt.Errorf("create notification event: %w", err)
	}
	return r.resolveDedupeInsertConflict(ctx, input.DedupeKey, err)
}

func (r *SQLRepository) findExistingDedupedEvent(ctx context.Context, dedupeKey string) (Event, bool, error) {
	if dedupeKey == "" {
		return Event{}, false, nil
	}
	event, err := r.findEventByDedupeKey(ctx, dedupeKey)
	if err == nil {
		return event, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Event{}, false, fmt.Errorf("query notification event by dedupe key: %w", err)
	}
	return Event{}, false, nil
}

func (r *SQLRepository) resolveDedupeInsertConflict(ctx context.Context, dedupeKey string, insertErr error) (Event, bool, error) {
	if !isUniqueViolation(insertErr) {
		return Event{}, false, fmt.Errorf("create notification event: %w", insertErr)
	}
	event, findErr := r.findEventByDedupeKey(ctx, dedupeKey)
	if findErr != nil {
		return Event{}, false, fmt.Errorf("re-query notification event after dedupe conflict: %w", findErr)
	}
	return event, true, nil
}

func (r *SQLRepository) insertEvent(ctx context.Context, input CreateEventInput, createdAt time.Time) (Event, error) {
	return scanEvent(r.db.QueryRowContext(
		ctx,
		r.placeholder.rebind(`INSERT INTO notification_events (
			title_key, title, message_key, message, category_key, source_key, level_key, event_type_key,
			resource_type_key, action_label_key, action_label, severity, category, source_module, event_type,
			resource_type, resource_id, resource_name, navigation_kind, navigation_payload, metadata,
			dedupe_key, occurred_at, expires_at, created_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?
		)
		RETURNING id, title_key, title, message_key, message, category_key, source_key, level_key, event_type_key,
			resource_type_key, action_label_key, action_label, severity, category, source_module, event_type,
			resource_type, resource_id, resource_name, navigation_kind, navigation_payload, metadata,
			dedupe_key, occurred_at, expires_at, created_at`),
		input.TitleKey,
		input.Title,
		input.MessageKey,
		input.Message,
		input.CategoryKey,
		input.SourceKey,
		input.LevelKey,
		input.EventTypeKey,
		input.ResourceTypeKey,
		input.ActionLabelKey,
		input.ActionLabel,
		input.Severity,
		input.Category,
		input.SourceModule,
		input.EventType,
		input.ResourceType,
		input.ResourceID,
		input.ResourceName,
		input.NavigationKind,
		jsonBytes(input.NavigationPayload),
		jsonBytes(input.Metadata),
		nullableString(input.DedupeKey),
		input.OccurredAt.UTC(),
		input.ExpiresAt,
		createdAt,
	))
}

func (r *SQLRepository) findEventByDedupeKey(ctx context.Context, dedupeKey string) (Event, error) {
	return scanEvent(r.db.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT id, title_key, title, message_key, message, category_key, source_key, level_key,
			event_type_key, resource_type_key, action_label_key, action_label, severity, category, source_module, event_type,
			resource_type, resource_id, resource_name, navigation_kind, navigation_payload, metadata,
			dedupe_key, occurred_at, expires_at, created_at
		FROM notification_events
		WHERE dedupe_key = ?`),
		strings.TrimSpace(dedupeKey),
	))
}

// CreateDeliveries inserts one or more user delivery rows.
func (r *SQLRepository) CreateDeliveries(ctx context.Context, inputs []CreateDeliveryInput) ([]Delivery, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("notification repository is unavailable")
	}
	if len(inputs) == 0 {
		return nil, ErrInvalidInput
	}

	deliveryInputs := make([]deliveryInsertInput, 0, len(inputs))
	for _, input := range inputs {
		deliveryInput, err := validateDeliveryInput(input)
		if err != nil {
			return nil, err
		}
		deliveryInputs = append(deliveryInputs, deliveryInput)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin notification delivery transaction: %w", err)
	}
	defer rollbackTx(tx)

	now := time.Now().UTC()
	deliveries := make([]Delivery, 0, len(deliveryInputs))
	for _, deliveryInput := range deliveryInputs {
		delivery, err := r.createDelivery(ctx, tx, deliveryInput, now)
		if err != nil {
			return nil, fmt.Errorf("create notification delivery: %w", err)
		}
		deliveries = append(deliveries, delivery)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit notification deliveries: %w", err)
	}
	return deliveries, nil
}

func (r *SQLRepository) createDelivery(
	ctx context.Context,
	tx *sql.Tx,
	input deliveryInsertInput,
	createdAt time.Time,
) (Delivery, error) {
	return scanDelivery(tx.QueryRowContext(
		ctx,
		r.placeholder.rebind(`INSERT INTO notification_deliveries (
			event_id, recipient_user_id, target_type, target_ref, created_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (event_id, recipient_user_id) DO UPDATE SET
			target_type = excluded.target_type,
			target_ref = excluded.target_ref
		RETURNING id, event_id, recipient_user_id, target_type, target_ref, read_at, deleted_at, created_at`),
		input.eventID,
		input.recipientUserID,
		input.targetType,
		input.targetRef,
		createdAt,
	))
}

type deliveryInsertInput struct {
	eventID         int64
	recipientUserID int64
	targetType      string
	targetRef       string
}

func validateDeliveryInput(input CreateDeliveryInput) (deliveryInsertInput, error) {
	input.TargetType = strings.TrimSpace(input.TargetType)
	input.TargetRef = strings.TrimSpace(input.TargetRef)
	if input.EventID == 0 || input.RecipientUserID == 0 || input.TargetType == "" || input.TargetRef == "" {
		return deliveryInsertInput{}, ErrInvalidInput
	}
	eventID, err := toDBID(input.EventID)
	if err != nil {
		return deliveryInsertInput{}, err
	}
	recipientUserID, err := toDBID(input.RecipientUserID)
	if err != nil {
		return deliveryInsertInput{}, err
	}
	return deliveryInsertInput{
		eventID:         eventID,
		recipientUserID: recipientUserID,
		targetType:      input.TargetType,
		targetRef:       input.TargetRef,
	}, nil
}

// List returns current-user visible notifications.
func (r *SQLRepository) List(ctx context.Context, query ListQuery) (ListResult, error) {
	if r == nil || r.db == nil {
		return ListResult{}, errors.New("notification repository is unavailable")
	}
	query = normalizeListQuery(query)
	if query.RecipientUserID == 0 {
		return ListResult{}, ErrInvalidInput
	}
	recipientUserID, err := toDBID(query.RecipientUserID)
	if err != nil {
		return ListResult{}, err
	}

	where, args, err := buildListWhere(query)
	if err != nil {
		return ListResult{}, err
	}
	args[0] = recipientUserID

	//nolint:gosec // Query predicates come from buildListWhere's fixed fragments; values stay parameterized.
	countSQL := r.placeholder.rebind(fmt.Sprintf(`SELECT COUNT(*)
		FROM notification_deliveries d
		INNER JOIN notification_events e ON e.id = d.event_id
		WHERE %s`, strings.Join(where, " AND ")))
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count notifications: %w", err)
	}

	args = append(args, query.Limit, query.Offset)
	//nolint:gosec // Query predicates come from buildListWhere's fixed fragments; values stay parameterized.
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(fmt.Sprintf(`SELECT
			e.id, e.title_key, e.title, e.message_key, e.message, e.category_key, e.source_key, e.level_key,
			e.event_type_key, e.resource_type_key, e.action_label_key, e.action_label, e.severity, e.category,
			e.source_module, e.event_type, e.resource_type, e.resource_id, e.resource_name,
			e.navigation_kind, e.navigation_payload, e.metadata, e.dedupe_key, e.occurred_at, e.expires_at, e.created_at,
			d.id, d.event_id, d.recipient_user_id, d.target_type, d.target_ref, d.read_at, d.deleted_at, d.created_at
		FROM notification_deliveries d
		INNER JOIN notification_events e ON e.id = d.event_id
		WHERE %s
		ORDER BY e.occurred_at DESC, d.id DESC
		LIMIT ? OFFSET ?`, strings.Join(where, " AND "))), args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list notifications: %w", err)
	}
	defer closeRows(rows)

	items, err := scanNotifications(rows)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, Total: total}, nil
}

// Get returns one current-user visible notification by delivery id.
func (r *SQLRepository) Get(ctx context.Context, recipientUserID uint64, deliveryID uint64) (Notification, error) {
	if r == nil || r.db == nil {
		return Notification{}, errors.New("notification repository is unavailable")
	}
	if recipientUserID == 0 || deliveryID == 0 {
		return Notification{}, ErrInvalidInput
	}
	recipientID, err := toDBID(recipientUserID)
	if err != nil {
		return Notification{}, err
	}
	targetID, err := toDBID(deliveryID)
	if err != nil {
		return Notification{}, err
	}

	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT
			e.id, e.title_key, e.title, e.message_key, e.message, e.category_key, e.source_key, e.level_key,
			e.event_type_key, e.resource_type_key, e.action_label_key, e.action_label, e.severity, e.category,
			e.source_module, e.event_type, e.resource_type, e.resource_id, e.resource_name,
			e.navigation_kind, e.navigation_payload, e.metadata, e.dedupe_key, e.occurred_at, e.expires_at, e.created_at,
			d.id, d.event_id, d.recipient_user_id, d.target_type, d.target_ref, d.read_at, d.deleted_at, d.created_at
		FROM notification_deliveries d
		INNER JOIN notification_events e ON e.id = d.event_id
		WHERE d.id = ? AND d.recipient_user_id = ? AND d.deleted_at = 0`), targetID, recipientID)
	if err != nil {
		return Notification{}, fmt.Errorf("get notification: %w", err)
	}
	defer closeRows(rows)

	items, err := scanNotifications(rows)
	if err != nil {
		return Notification{}, err
	}
	if len(items) == 0 {
		return Notification{}, ErrDeliveryNotFound
	}
	return items[0], nil
}

// UnreadCount returns the non-deleted unread delivery count for one user.
func (r *SQLRepository) UnreadCount(ctx context.Context, recipientUserID uint64) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("notification repository is unavailable")
	}
	if recipientUserID == 0 {
		return 0, ErrInvalidInput
	}
	id, err := toDBID(recipientUserID)
	if err != nil {
		return 0, err
	}

	var count int
	if err := r.db.QueryRowContext(ctx, r.placeholder.rebind(`SELECT COUNT(*)
		FROM notification_deliveries
		WHERE recipient_user_id = ? AND read_at IS NULL AND deleted_at = 0`), id).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

// MarkRead marks one current-user delivery as read.
func (r *SQLRepository) MarkRead(ctx context.Context, recipientUserID uint64, deliveryID uint64, readAt time.Time) (Delivery, error) {
	if r == nil || r.db == nil {
		return Delivery{}, errors.New("notification repository is unavailable")
	}
	recipientID, targetID, err := deliveryAccessIDs(recipientUserID, deliveryID, readAt)
	if err != nil {
		return Delivery{}, err
	}
	delivery, err := r.getDelivery(ctx, targetID, recipientID)
	if err != nil {
		return Delivery{}, err
	}
	markAt := readAt.UTC()
	if delivery.ReadAt != nil {
		markAt = delivery.ReadAt.UTC()
	}

	result, err := r.db.ExecContext(
		ctx,
		r.placeholder.rebind(`UPDATE notification_deliveries
		SET read_at = ?
		WHERE id = ? AND recipient_user_id = ? AND deleted_at = 0`),
		markAt,
		targetID,
		recipientID,
	)
	if err != nil {
		return Delivery{}, fmt.Errorf("mark notification delivery read: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Delivery{}, fmt.Errorf("read mark notification rows affected: %w", err)
	}
	if affected == 0 {
		return Delivery{}, ErrDeliveryNotFound
	}
	return r.getDelivery(ctx, targetID, recipientID)
}

// MarkAllRead marks all non-deleted unread deliveries for one user as read.
func (r *SQLRepository) MarkAllRead(ctx context.Context, recipientUserID uint64, readAt time.Time) (int, error) {
	return r.MarkAllReadMatching(ctx, ListQuery{RecipientUserID: recipientUserID, Status: "unread"}, readAt)
}

// MarkAllReadMatching marks all non-deleted unread deliveries matching current-user filters as read.
func (r *SQLRepository) MarkAllReadMatching(ctx context.Context, query ListQuery, readAt time.Time) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("notification repository is unavailable")
	}
	query = normalizeListQuery(query)
	query.Status = "unread"
	if query.RecipientUserID == 0 || readAt.IsZero() {
		return 0, ErrInvalidInput
	}
	id, err := toDBID(query.RecipientUserID)
	if err != nil {
		return 0, err
	}
	where, args, err := buildListWhere(query)
	if err != nil {
		return 0, err
	}
	args[0] = id
	args = append([]any{readAt.UTC()}, args...)

	//nolint:gosec // Query predicates come from buildListWhere's fixed fragments; values stay parameterized.
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(fmt.Sprintf(`UPDATE notification_deliveries
		SET read_at = COALESCE(read_at, ?)
		WHERE id IN (
			SELECT d.id
			FROM notification_deliveries d
			INNER JOIN notification_events e ON e.id = d.event_id
			WHERE %s
		)`, strings.Join(where, " AND "))), args...)
	if err != nil {
		return 0, fmt.Errorf("mark all notification deliveries read: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read mark-all notification rows affected: %w", err)
	}
	return int(affected), nil
}

// DeleteDelivery soft-deletes one current-user delivery.
func (r *SQLRepository) DeleteDelivery(ctx context.Context, recipientUserID uint64, deliveryID uint64, deletedAt time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("notification repository is unavailable")
	}
	recipientID, targetID, err := deliveryAccessIDs(recipientUserID, deliveryID, deletedAt)
	if err != nil {
		return err
	}
	if _, err := r.getDelivery(ctx, targetID, recipientID); err != nil {
		return err
	}
	deleteEpoch := deletedAt.UTC().Unix()
	if deleteEpoch <= 0 {
		return ErrInvalidInput
	}

	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE notification_deliveries
		SET deleted_at = ?
		WHERE id = ? AND recipient_user_id = ? AND deleted_at = 0`), deleteEpoch, targetID, recipientID)
	if err != nil {
		return fmt.Errorf("delete notification delivery: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read delete notification rows affected: %w", err)
	}
	if affected == 0 {
		return ErrDeliveryNotFound
	}
	return nil
}

func deliveryAccessIDs(recipientUserID uint64, deliveryID uint64, eventTime time.Time) (int64, int64, error) {
	if recipientUserID == 0 || deliveryID == 0 || eventTime.IsZero() {
		return 0, 0, ErrInvalidInput
	}
	recipientID, err := toDBID(recipientUserID)
	if err != nil {
		return 0, 0, err
	}
	targetID, err := toDBID(deliveryID)
	if err != nil {
		return 0, 0, err
	}
	return recipientID, targetID, nil
}

func normalizeEventInput(input CreateEventInput) CreateEventInput {
	input.TitleKey = strings.TrimSpace(input.TitleKey)
	input.Title = strings.TrimSpace(input.Title)
	input.MessageKey = strings.TrimSpace(input.MessageKey)
	input.Message = strings.TrimSpace(input.Message)
	input.CategoryKey = strings.TrimSpace(input.CategoryKey)
	input.SourceKey = strings.TrimSpace(input.SourceKey)
	input.LevelKey = strings.TrimSpace(input.LevelKey)
	input.EventTypeKey = strings.TrimSpace(input.EventTypeKey)
	input.ResourceTypeKey = strings.TrimSpace(input.ResourceTypeKey)
	input.ActionLabelKey = strings.TrimSpace(input.ActionLabelKey)
	input.ActionLabel = strings.TrimSpace(input.ActionLabel)
	input.Severity = strings.TrimSpace(input.Severity)
	input.Category = strings.TrimSpace(input.Category)
	input.SourceModule = strings.TrimSpace(input.SourceModule)
	input.EventType = strings.TrimSpace(input.EventType)
	input.ResourceType = strings.TrimSpace(input.ResourceType)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.ResourceName = strings.TrimSpace(input.ResourceName)
	input.NavigationKind = strings.TrimSpace(input.NavigationKind)
	input.DedupeKey = strings.TrimSpace(input.DedupeKey)
	input.OccurredAt = input.OccurredAt.UTC()
	if input.ExpiresAt != nil {
		expiresAt := input.ExpiresAt.UTC()
		input.ExpiresAt = &expiresAt
	}
	return input
}

func normalizeListQuery(query ListQuery) ListQuery {
	query.Status = strings.TrimSpace(query.Status)
	query.Severity = strings.TrimSpace(query.Severity)
	query.Category = strings.TrimSpace(query.Category)
	query.SourceModule = strings.TrimSpace(query.SourceModule)
	if query.Limit <= 0 {
		query.Limit = defaultListLimit
	}
	if query.Limit > maxListLimit {
		query.Limit = maxListLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query
}

func buildListWhere(query ListQuery) ([]string, []any, error) {
	where := []string{"d.recipient_user_id = ?", "d.deleted_at = 0"}
	args := []any{query.RecipientUserID}
	switch query.Status {
	case "", "all":
	case "unread":
		where = append(where, "d.read_at IS NULL")
	case "read":
		where = append(where, "d.read_at IS NOT NULL")
	default:
		return nil, nil, ErrInvalidInput
	}
	if query.Severity != "" {
		args = append(args, query.Severity)
		where = append(where, "e.severity = ?")
	}
	if query.Category != "" {
		args = append(args, query.Category)
		where = append(where, "e.category = ?")
	}
	if query.SourceModule != "" {
		args = append(args, query.SourceModule)
		where = append(where, "e.source_module = ?")
	}
	if query.OccurredFrom != nil {
		args = append(args, query.OccurredFrom.UTC())
		where = append(where, "e.occurred_at >= ?")
	}
	if query.OccurredTo != nil {
		args = append(args, query.OccurredTo.UTC())
		where = append(where, "e.occurred_at <= ?")
	}
	return where, args, nil
}

func scanEvent(scanner interface{ Scan(dest ...any) error }) (Event, error) {
	var event Event
	var navigationPayload []byte
	var metadata []byte
	var dedupeKey sql.NullString
	if err := scanner.Scan(
		&event.ID,
		&event.TitleKey,
		&event.Title,
		&event.MessageKey,
		&event.Message,
		&event.CategoryKey,
		&event.SourceKey,
		&event.LevelKey,
		&event.EventTypeKey,
		&event.ResourceTypeKey,
		&event.ActionLabelKey,
		&event.ActionLabel,
		&event.Severity,
		&event.Category,
		&event.SourceModule,
		&event.EventType,
		&event.ResourceType,
		&event.ResourceID,
		&event.ResourceName,
		&event.NavigationKind,
		&navigationPayload,
		&metadata,
		&dedupeKey,
		&event.OccurredAt,
		&event.ExpiresAt,
		&event.CreatedAt,
	); err != nil {
		return Event{}, err
	}
	event.NavigationPayload = append(event.NavigationPayload[:0], navigationPayload...)
	event.Metadata = append(event.Metadata[:0], metadata...)
	if dedupeKey.Valid {
		event.DedupeKey = dedupeKey.String
	}
	return event, nil
}

func scanDelivery(scanner interface{ Scan(dest ...any) error }) (Delivery, error) {
	var delivery Delivery
	var readAt sql.NullTime
	if err := scanner.Scan(
		&delivery.ID,
		&delivery.EventID,
		&delivery.RecipientUserID,
		&delivery.TargetType,
		&delivery.TargetRef,
		&readAt,
		&delivery.DeletedAt,
		&delivery.CreatedAt,
	); err != nil {
		return Delivery{}, err
	}
	if readAt.Valid {
		delivery.ReadAt = &readAt.Time
	}
	return delivery, nil
}

func (r *SQLRepository) getDelivery(ctx context.Context, deliveryID int64, recipientUserID int64) (Delivery, error) {
	delivery, err := scanDelivery(r.db.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT id, event_id, recipient_user_id, target_type, target_ref, read_at, deleted_at, created_at
		FROM notification_deliveries
		WHERE id = ? AND recipient_user_id = ? AND deleted_at = 0`),
		deliveryID,
		recipientUserID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Delivery{}, ErrDeliveryNotFound
		}
		return Delivery{}, fmt.Errorf("get notification delivery: %w", err)
	}
	return delivery, nil
}

func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func closeRows(rows *sql.Rows) {
	if rows != nil {
		_ = rows.Close()
	}
}

func scanNotifications(rows *sql.Rows) ([]Notification, error) {
	items := make([]Notification, 0)
	for rows.Next() {
		var item Notification
		var navigationPayload []byte
		var metadata []byte
		var dedupeKey sql.NullString
		var readAt sql.NullTime
		if err := rows.Scan(
			&item.Event.ID,
			&item.Event.TitleKey,
			&item.Event.Title,
			&item.Event.MessageKey,
			&item.Event.Message,
			&item.Event.CategoryKey,
			&item.Event.SourceKey,
			&item.Event.LevelKey,
			&item.Event.EventTypeKey,
			&item.Event.ResourceTypeKey,
			&item.Event.ActionLabelKey,
			&item.Event.ActionLabel,
			&item.Event.Severity,
			&item.Event.Category,
			&item.Event.SourceModule,
			&item.Event.EventType,
			&item.Event.ResourceType,
			&item.Event.ResourceID,
			&item.Event.ResourceName,
			&item.Event.NavigationKind,
			&navigationPayload,
			&metadata,
			&dedupeKey,
			&item.Event.OccurredAt,
			&item.Event.ExpiresAt,
			&item.Event.CreatedAt,
			&item.Delivery.ID,
			&item.Delivery.EventID,
			&item.Delivery.RecipientUserID,
			&item.Delivery.TargetType,
			&item.Delivery.TargetRef,
			&readAt,
			&item.Delivery.DeletedAt,
			&item.Delivery.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification row: %w", err)
		}
		item.Event.NavigationPayload = append(item.Event.NavigationPayload[:0], navigationPayload...)
		item.Event.Metadata = append(item.Event.Metadata[:0], metadata...)
		if dedupeKey.Valid {
			item.Event.DedupeKey = dedupeKey.String
		}
		if readAt.Valid {
			item.Delivery.ReadAt = &readAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification rows: %w", err)
	}
	return items, nil
}

func jsonBytes(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return append([]byte(nil), raw...)
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

type placeholderStyle int

const (
	placeholderDollar placeholderStyle = iota
	placeholderQuestion
)

func detectPlaceholderStyle(db *sql.DB) placeholderStyle {
	if db == nil || db.Driver() == nil {
		return placeholderDollar
	}
	driverType := strings.ToLower(reflect.TypeOf(db.Driver()).String())
	if strings.Contains(driverType, "sqlite") {
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

func toDBID(value uint64) (int64, error) {
	if value == 0 || value > uint64(^uint64(0)>>1) {
		return 0, ErrInvalidInput
	}
	return int64(value), nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	type sqlStateCarrier interface {
		SQLState() string
	}
	var stateErr sqlStateCarrier
	if errors.As(err, &stateErr) && stateErr.SQLState() == "23505" {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "sqlstate 23505") ||
		strings.Contains(message, "unique constraint failed")
}

var _ Repository = (*SQLRepository)(nil)
