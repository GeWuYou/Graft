package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
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
