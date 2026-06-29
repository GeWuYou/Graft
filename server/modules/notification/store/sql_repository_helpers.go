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

func (r *SQLRepository) ensureReady() error {
	if r == nil || r.db == nil {
		return errors.New("notification repository is unavailable")
	}
	return nil
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
