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
)

const (
	defaultListLimit          = 20
	maxListLimit              = 100
	placeholderGrowthEstimate = 8
	adminFilterCapacity       = 4
	userFilterCapacity        = 3
	sortUpdatedDesc           = "updated_desc"
	sortPublishDesc           = "publish_desc"
	sortPinnedPublishDesc     = "pinned_publish_desc"
	statusPublished           = "published"
	statusArchived            = "archived"
)

type sqlDialect int

const (
	sqlDialectPostgres sqlDialect = iota
	sqlDialectSQLite
)

// SQLRepository persists Announcement Center state in module-owned SQL tables.
type SQLRepository struct {
	db          *sql.DB
	placeholder placeholderStyle
	dialect     sqlDialect
}

// NewSQLRepository creates a SQL-backed announcement repository.
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("announcement repository requires a non-nil sql db")
	}
	dialect := detectSQLDialect(db)
	return &SQLRepository{db: db, placeholder: placeholderStyleForDialect(dialect), dialect: dialect}, nil
}

// Ping verifies the repository can reach its SQL dependency.
func (r *SQLRepository) Ping(ctx context.Context) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	return r.db.PingContext(ctx)
}

// ListAdmin returns non-deleted announcements for the management list.
func (r *SQLRepository) ListAdmin(ctx context.Context, query ListQuery) (ListResult, error) {
	if err := r.ensureReady(); err != nil {
		return ListResult{}, err
	}
	query = normalizeListQuery(query)
	where, args, err := buildAdminWhere(query, r.dialect)
	if err != nil {
		return ListResult{}, err
	}

	//nolint:gosec // Predicates and ordering come from fixed fragments; values stay parameterized.
	countSQL := r.placeholder.rebind(fmt.Sprintf(`SELECT COUNT(*) FROM announcements WHERE %s`, strings.Join(where, " AND ")))
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count announcements: %w", err)
	}

	args = append(args, query.Limit, query.Offset)
	//nolint:gosec // Predicates and ordering come from fixed fragments; values stay parameterized.
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(fmt.Sprintf(`SELECT %s
		FROM announcements
		WHERE %s
		ORDER BY %s
		LIMIT ? OFFSET ?`, announcementColumns(), strings.Join(where, " AND "), adminOrderBy(query.Sort))), args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list announcements: %w", err)
	}
	defer closeRows(rows)

	items, err := scanAnnouncements(rows)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, Total: total}, nil
}

// ListCurrentUser returns currently visible announcements with read state for one user.
func (r *SQLRepository) ListCurrentUser(ctx context.Context, query UserListQuery) (UserListResult, error) {
	if err := r.ensureReady(); err != nil {
		return UserListResult{}, err
	}
	query, userDBID, err := normalizeUserListQuery(query)
	if err != nil {
		return UserListResult{}, err
	}
	where, args := buildUserVisibleWhere(query.Now, query.UnreadOnly)
	//nolint:gosec // Predicates come from fixed visibility fragments; values stay parameterized.
	countSQL := r.placeholder.rebind(fmt.Sprintf(`SELECT COUNT(*)
		FROM announcements a
		LEFT JOIN announcement_reads ar ON ar.announcement_id = a.id AND ar.user_id = ?
		WHERE %s`, strings.Join(where, " AND ")))
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, append([]any{userDBID}, args...)...).Scan(&total); err != nil {
		return UserListResult{}, fmt.Errorf("count current-user announcements: %w", err)
	}

	listArgs := append([]any{userDBID}, args...)
	listArgs = append(listArgs, query.Limit, query.Offset)
	//nolint:gosec // Predicates and ordering come from fixed fragments; values stay parameterized.
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(fmt.Sprintf(`SELECT %s, ar.read_at
		FROM announcements a
		LEFT JOIN announcement_reads ar ON ar.announcement_id = a.id AND ar.user_id = ?
		WHERE %s
		ORDER BY a.pinned DESC, a.publish_at DESC, a.id DESC
		LIMIT ? OFFSET ?`, prefixedAnnouncementColumns("a"), strings.Join(where, " AND "))), listArgs...)
	if err != nil {
		return UserListResult{}, fmt.Errorf("list current-user announcements: %w", err)
	}
	defer closeRows(rows)

	items, err := scanUserAnnouncements(rows)
	if err != nil {
		return UserListResult{}, err
	}
	return UserListResult{Items: items, Total: total}, nil
}

// Create inserts one management announcement.
func (r *SQLRepository) Create(ctx context.Context, input CreateInput) (Announcement, error) {
	if err := r.ensureReady(); err != nil {
		return Announcement{}, err
	}
	input = normalizeCreateInput(input)
	if input.Title == "" || input.Content == "" || input.Level == "" || input.Status == "" || input.DeliveryMode == "" {
		return Announcement{}, ErrInvalidInput
	}
	if input.ExpireAt != nil && input.PublishAt != nil && !input.ExpireAt.After(*input.PublishAt) {
		return Announcement{}, ErrInvalidInput
	}
	now := time.Now().UTC()
	return scanAnnouncement(r.db.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO announcements (
			title, content, level, status, delivery_mode, pinned, publish_at, expire_at, created_by, updated_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING `+announcementColumns()),
		input.Title,
		input.Content,
		input.Level,
		input.Status,
		input.DeliveryMode,
		input.Pinned,
		input.PublishAt,
		input.ExpireAt,
		input.ActorID,
		input.ActorID,
		now,
		now,
	))
}

// GetAdmin returns one non-deleted announcement by id.
func (r *SQLRepository) GetAdmin(ctx context.Context, id uint64) (Announcement, error) {
	if err := r.ensureReady(); err != nil {
		return Announcement{}, err
	}
	targetID, err := toDBID(id)
	if err != nil {
		return Announcement{}, err
	}
	item, err := scanAnnouncement(r.db.QueryRowContext(ctx, r.placeholder.rebind(`SELECT `+announcementColumns()+`
		FROM announcements
		WHERE id = ? AND deleted_at = 0`), targetID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Announcement{}, ErrAnnouncementNotFound
		}
		return Announcement{}, fmt.Errorf("get announcement: %w", err)
	}
	return item, nil
}

// Update replaces editable management fields for one non-deleted announcement.
func (r *SQLRepository) Update(ctx context.Context, id uint64, input UpdateInput) (Announcement, error) {
	if err := r.ensureReady(); err != nil {
		return Announcement{}, err
	}
	targetID, err := toDBID(id)
	if err != nil {
		return Announcement{}, err
	}
	input = normalizeUpdateInput(input)
	if !validUpdateInput(input) {
		return Announcement{}, ErrInvalidInput
	}
	item, err := r.updateAnnouncement(ctx, targetID, input)
	if err != nil {
		return Announcement{}, err
	}
	return item, nil
}

func validUpdateInput(input UpdateInput) bool {
	if input.Title == "" || input.Content == "" || input.Level == "" || input.DeliveryMode == "" {
		return false
	}
	return input.ExpireAt == nil || input.PublishAt == nil || input.ExpireAt.After(*input.PublishAt)
}

func (r *SQLRepository) updateAnnouncement(ctx context.Context, targetID int64, input UpdateInput) (Announcement, error) {
	item, err := scanAnnouncement(r.db.QueryRowContext(ctx, r.placeholder.rebind(`UPDATE announcements
		SET title = ?, content = ?, level = ?, delivery_mode = ?, pinned = ?, publish_at = ?, expire_at = ?, updated_by = ?, updated_at = ?
		WHERE id = ? AND deleted_at = 0
		RETURNING `+announcementColumns()),
		input.Title,
		input.Content,
		input.Level,
		input.DeliveryMode,
		input.Pinned,
		input.PublishAt,
		input.ExpireAt,
		input.ActorID,
		time.Now().UTC(),
		targetID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Announcement{}, ErrAnnouncementNotFound
		}
		return Announcement{}, fmt.Errorf("update announcement: %w", err)
	}
	return item, nil
}

// Publish marks one announcement published and records an effective publish time.
func (r *SQLRepository) Publish(ctx context.Context, id uint64, publishAt time.Time, actorID *uint64) (Announcement, error) {
	if err := r.ensureReady(); err != nil {
		return Announcement{}, err
	}
	targetID, err := toDBID(id)
	if err != nil {
		return Announcement{}, err
	}
	if publishAt.IsZero() {
		return Announcement{}, ErrInvalidInput
	}
	item, err := scanAnnouncement(r.db.QueryRowContext(ctx, r.placeholder.rebind(`UPDATE announcements
		SET status = ?, publish_at = ?, published_by = ?, archived_at = NULL, updated_by = ?, updated_at = ?
		WHERE id = ? AND deleted_at = 0
		RETURNING `+announcementColumns()),
		statusPublished,
		publishAt.UTC(),
		actorID,
		actorID,
		time.Now().UTC(),
		targetID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Announcement{}, ErrAnnouncementNotFound
		}
		return Announcement{}, fmt.Errorf("publish announcement: %w", err)
	}
	return item, nil
}

// Archive marks one non-deleted announcement archived.
func (r *SQLRepository) Archive(ctx context.Context, id uint64, actorID *uint64) (Announcement, error) {
	if err := r.ensureReady(); err != nil {
		return Announcement{}, err
	}
	targetID, err := toDBID(id)
	if err != nil {
		return Announcement{}, err
	}
	item, err := scanAnnouncement(r.db.QueryRowContext(ctx, r.placeholder.rebind(`UPDATE announcements
		SET status = ?, archived_at = ?, updated_by = ?, updated_at = ?
		WHERE id = ? AND deleted_at = 0
		RETURNING `+announcementColumns()),
		statusArchived,
		time.Now().UTC(),
		actorID,
		time.Now().UTC(),
		targetID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Announcement{}, ErrAnnouncementNotFound
		}
		return Announcement{}, fmt.Errorf("archive announcement: %w", err)
	}
	return item, nil
}

// Delete soft-deletes one non-deleted announcement.
func (r *SQLRepository) Delete(ctx context.Context, id uint64, actorID uint64, deletedAt time.Time) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	targetID, err := toDBID(id)
	if err != nil {
		return err
	}
	if deletedAt.IsZero() {
		return ErrInvalidInput
	}
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE announcements
		SET deleted_by = ?, deleted_at = ?, updated_by = ?, updated_at = ?
		WHERE id = ? AND deleted_at = 0`), nullableUint64(actorID), deletedAt.UTC().Unix(), nullableUint64(actorID), deletedAt.UTC(), targetID)
	if err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read delete announcement rows affected: %w", err)
	}
	if affected == 0 {
		return ErrAnnouncementNotFound
	}
	return nil
}

// MarkRead records one read fact for a currently visible announcement and user.
func (r *SQLRepository) MarkRead(ctx context.Context, userID uint64, announcementID uint64, readAt time.Time) (UserAnnouncement, error) {
	if err := r.ensureReady(); err != nil {
		return UserAnnouncement{}, err
	}
	userDBID, announcementDBID, err := readAccessIDs(userID, announcementID, readAt)
	if err != nil {
		return UserAnnouncement{}, err
	}
	now := time.Now().UTC()
	if _, err := r.getVisibleAnnouncement(ctx, announcementDBID, userDBID, now); err != nil {
		return UserAnnouncement{}, err
	}
	readAt = readAt.UTC()
	if _, err := r.db.ExecContext(ctx, r.placeholder.rebind(`INSERT INTO announcement_reads (
			announcement_id, user_id, read_at, created_at
		) VALUES (?, ?, ?, ?)
		ON CONFLICT (announcement_id, user_id) DO NOTHING`),
		announcementDBID,
		userDBID,
		readAt,
		readAt,
	); err != nil {
		return UserAnnouncement{}, fmt.Errorf("mark announcement read: %w", err)
	}
	return r.getVisibleAnnouncement(ctx, announcementDBID, userDBID, now)
}

// MarkAllRead records read facts for all currently visible unread announcements for one user.
func (r *SQLRepository) MarkAllRead(ctx context.Context, userID uint64, readAt time.Time, now time.Time) (int, error) {
	if err := r.ensureReady(); err != nil {
		return 0, err
	}
	userDBID, err := userReadDBID(userID, readAt, now)
	if err != nil {
		return 0, err
	}
	where, args := buildUserVisibleWhere(now.UTC(), true)
	insertArgs := append([]any{userDBID, readAt.UTC(), readAt.UTC(), userDBID}, args...)

	//nolint:gosec // Visibility predicates come from fixed fragments; values stay parameterized.
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(fmt.Sprintf(`INSERT INTO announcement_reads (
			announcement_id, user_id, read_at, created_at
		)
		SELECT a.id, ?, ?, ?
		FROM announcements a
		LEFT JOIN announcement_reads ar ON ar.announcement_id = a.id AND ar.user_id = ?
		WHERE %s
		ON CONFLICT (announcement_id, user_id) DO NOTHING`, strings.Join(where, " AND "))), insertArgs...)
	if err != nil {
		return 0, fmt.Errorf("mark all announcements read: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read mark-all announcement rows affected: %w", err)
	}
	return int(affected), nil
}

// UnreadCount counts currently visible unread announcements for one user.
func (r *SQLRepository) UnreadCount(ctx context.Context, userID uint64, now time.Time) (int, error) {
	if err := r.ensureReady(); err != nil {
		return 0, err
	}
	if now.IsZero() {
		return 0, ErrInvalidInput
	}
	userDBID, err := toDBID(userID)
	if err != nil {
		return 0, err
	}
	where, args := buildUserVisibleWhere(now.UTC(), true)
	countArgs := append([]any{userDBID}, args...)

	//nolint:gosec // Visibility predicates come from fixed fragments; values stay parameterized.
	countSQL := r.placeholder.rebind(fmt.Sprintf(`SELECT COUNT(*)
		FROM announcements a
		LEFT JOIN announcement_reads ar ON ar.announcement_id = a.id AND ar.user_id = ?
		WHERE %s`, strings.Join(where, " AND ")))
	var count int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread announcements: %w", err)
	}
	return count, nil
}

func (r *SQLRepository) ensureReady() error {
	if r == nil || r.db == nil {
		return errors.New("announcement repository is unavailable")
	}
	return nil
}

func normalizeCreateInput(input CreateInput) CreateInput {
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	input.Level = strings.TrimSpace(input.Level)
	input.Status = strings.TrimSpace(input.Status)
	input.DeliveryMode = strings.TrimSpace(input.DeliveryMode)
	if input.PublishAt != nil {
		publishAt := input.PublishAt.UTC()
		input.PublishAt = &publishAt
	}
	if input.ExpireAt != nil {
		expireAt := input.ExpireAt.UTC()
		input.ExpireAt = &expireAt
	}
	return input
}

func normalizeUpdateInput(input UpdateInput) UpdateInput {
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	input.Level = strings.TrimSpace(input.Level)
	input.DeliveryMode = strings.TrimSpace(input.DeliveryMode)
	if input.PublishAt != nil {
		publishAt := input.PublishAt.UTC()
		input.PublishAt = &publishAt
	}
	if input.ExpireAt != nil {
		expireAt := input.ExpireAt.UTC()
		input.ExpireAt = &expireAt
	}
	return input
}

func normalizeListQuery(query ListQuery) ListQuery {
	query.Status = strings.TrimSpace(query.Status)
	query.Level = strings.TrimSpace(query.Level)
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Sort = strings.TrimSpace(query.Sort)
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

func normalizeUserListQuery(query UserListQuery) (UserListQuery, int64, error) {
	if query.Now.IsZero() || query.UserID == 0 {
		return UserListQuery{}, 0, ErrInvalidInput
	}
	userID, err := toDBID(query.UserID)
	if err != nil {
		return UserListQuery{}, 0, err
	}
	query.Now = query.Now.UTC()
	if query.Limit <= 0 {
		query.Limit = defaultListLimit
	}
	if query.Limit > maxListLimit {
		query.Limit = maxListLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query, userID, nil
}

func buildAdminWhere(query ListQuery, dialect sqlDialect) ([]string, []any, error) {
	where := []string{"deleted_at = 0"}
	args := make([]any, 0, adminFilterCapacity)
	if query.Status != "" {
		args = append(args, query.Status)
		where = append(where, "status = ?")
	}
	if query.Level != "" {
		args = append(args, query.Level)
		where = append(where, "level = ?")
	}
	if query.Pinned != nil {
		args = append(args, *query.Pinned)
		where = append(where, "pinned = ?")
	}
	if query.Keyword != "" {
		keyword := strings.ToLower(query.Keyword)
		if dialect == sqlDialectSQLite {
			pattern := "%" + keyword + "%"
			args = append(args, pattern, pattern)
			where = append(where, "(LOWER(title) LIKE ? OR LOWER(content) LIKE ?)")
		} else {
			args = append(args, keyword)
			where = append(where, "to_tsvector('simple', title || ' ' || content) @@ plainto_tsquery('simple', ?)")
		}
	}
	return where, args, nil
}

func buildUserVisibleWhere(now time.Time, unreadOnly bool) ([]string, []any) {
	where := []string{
		"a.deleted_at = 0",
		"a.status = ?",
		"a.publish_at <= ?",
		"(a.expire_at IS NULL OR a.expire_at > ?)",
	}
	args := make([]any, 0, userFilterCapacity)
	args = append(args, statusPublished, now, now)
	if unreadOnly {
		where = append(where, "ar.read_at IS NULL")
	}
	return where, args
}

func adminOrderBy(sort string) string {
	switch sort {
	case sortPinnedPublishDesc:
		return "pinned DESC, publish_at DESC NULLS LAST, id DESC"
	case sortPublishDesc:
		return "publish_at DESC NULLS LAST, id DESC"
	case "", sortUpdatedDesc:
		return "updated_at DESC, id DESC"
	default:
		return "updated_at DESC, id DESC"
	}
}

func announcementColumns() string {
	return `id, title, content, level, status, delivery_mode, pinned, publish_at, published_by, archived_at, expire_at,
		created_by, updated_by, deleted_by, created_at, updated_at, deleted_at`
}

func prefixedAnnouncementColumns(prefix string) string {
	columns := strings.Split(announcementColumns(), ",")
	for index, column := range columns {
		columns[index] = prefix + "." + strings.TrimSpace(column)
	}
	return strings.Join(columns, ", ")
}

func scanAnnouncement(scanner interface{ Scan(dest ...any) error }) (Announcement, error) {
	var item Announcement
	var publishAt sql.NullTime
	var archivedAt sql.NullTime
	var expireAt sql.NullTime
	var publishedBy sql.NullInt64
	var createdBy sql.NullInt64
	var updatedBy sql.NullInt64
	var deletedBy sql.NullInt64
	if err := scanner.Scan(
		&item.ID,
		&item.Title,
		&item.Content,
		&item.Level,
		&item.Status,
		&item.DeliveryMode,
		&item.Pinned,
		&publishAt,
		&publishedBy,
		&archivedAt,
		&expireAt,
		&createdBy,
		&updatedBy,
		&deletedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return Announcement{}, err
	}
	if publishAt.Valid {
		item.PublishAt = &publishAt.Time
	}
	var err error
	item.PublishedBy, err = optionalUint64FromDBID(publishedBy)
	if err != nil {
		return Announcement{}, err
	}
	if archivedAt.Valid {
		item.ArchivedAt = &archivedAt.Time
	}
	if expireAt.Valid {
		item.ExpireAt = &expireAt.Time
	}
	item.CreatedBy, err = optionalUint64FromDBID(createdBy)
	if err != nil {
		return Announcement{}, err
	}
	item.UpdatedBy, err = optionalUint64FromDBID(updatedBy)
	if err != nil {
		return Announcement{}, err
	}
	item.DeletedBy, err = optionalUint64FromDBID(deletedBy)
	if err != nil {
		return Announcement{}, err
	}
	return item, nil
}

func optionalUint64FromDBID(value sql.NullInt64) (*uint64, error) {
	if !value.Valid {
		return nil, nil
	}
	converted, err := uint64FromDBID(value.Int64)
	if err != nil {
		return nil, err
	}
	return &converted, nil
}

func scanAnnouncements(rows *sql.Rows) ([]Announcement, error) {
	items := make([]Announcement, 0)
	for rows.Next() {
		item, err := scanAnnouncement(rows)
		if err != nil {
			return nil, fmt.Errorf("scan announcement row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate announcement rows: %w", err)
	}
	return items, nil
}

func scanUserAnnouncement(scanner interface{ Scan(dest ...any) error }) (UserAnnouncement, error) {
	var item UserAnnouncement
	var readAt sql.NullTime
	announcement, err := scanAnnouncement(rowScannerFunc(func(dest ...any) error {
		return scanner.Scan(append(dest, &readAt)...)
	}))
	if err != nil {
		return UserAnnouncement{}, err
	}
	item.Announcement = announcement
	if readAt.Valid {
		value := readAt.Time.UTC()
		item.ReadAt = &value
	}
	return item, nil
}

func scanUserAnnouncements(rows *sql.Rows) ([]UserAnnouncement, error) {
	items := make([]UserAnnouncement, 0)
	for rows.Next() {
		item, err := scanUserAnnouncement(rows)
		if err != nil {
			return nil, fmt.Errorf("scan current-user announcement row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate current-user announcement rows: %w", err)
	}
	return items, nil
}

func (r *SQLRepository) getVisibleAnnouncement(ctx context.Context, announcementID int64, userID int64, now time.Time) (UserAnnouncement, error) {
	where, args := buildUserVisibleWhere(now.UTC(), false)
	args = append([]any{userID}, args...)
	args = append(args, announcementID)
	where = append(where, "a.id = ?")

	//nolint:gosec // Visibility predicates come from fixed fragments; values stay parameterized.
	item, err := scanUserAnnouncement(r.db.QueryRowContext(ctx, r.placeholder.rebind(fmt.Sprintf(`SELECT %s, ar.read_at
		FROM announcements a
		LEFT JOIN announcement_reads ar ON ar.announcement_id = a.id AND ar.user_id = ?
		WHERE %s`, prefixedAnnouncementColumns("a"), strings.Join(where, " AND "))), args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserAnnouncement{}, ErrAnnouncementNotFound
		}
		return UserAnnouncement{}, fmt.Errorf("get current-user announcement: %w", err)
	}
	return item, nil
}

func closeRows(rows *sql.Rows) {
	if rows != nil {
		_ = rows.Close()
	}
}

func nullableUint64(value uint64) any {
	if value == 0 {
		return nil
	}
	return value
}

type placeholderStyle int

const (
	placeholderDollar placeholderStyle = iota
	placeholderQuestion
)

func detectSQLDialect(db *sql.DB) sqlDialect {
	if db == nil || db.Driver() == nil {
		return sqlDialectPostgres
	}
	driverType := strings.ToLower(reflect.TypeOf(db.Driver()).String())
	if strings.Contains(driverType, "sqlite") {
		return sqlDialectSQLite
	}
	return sqlDialectPostgres
}

func placeholderStyleForDialect(dialect sqlDialect) placeholderStyle {
	if dialect == sqlDialectSQLite {
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

func uint64FromDBID(value int64) (uint64, error) {
	if value < 0 {
		return 0, ErrInvalidInput
	}
	return uint64(value), nil
}

type rowScannerFunc func(dest ...any) error

func (f rowScannerFunc) Scan(dest ...any) error {
	return f(dest...)
}

func readAccessIDs(userID uint64, announcementID uint64, readAt time.Time) (int64, int64, error) {
	if announcementID == 0 {
		return 0, 0, ErrInvalidInput
	}
	userDBID, err := userReadDBID(userID, readAt, readAt)
	if err != nil {
		return 0, 0, err
	}
	announcementDBID, err := toDBID(announcementID)
	if err != nil {
		return 0, 0, err
	}
	return userDBID, announcementDBID, nil
}

func userReadDBID(userID uint64, readAt time.Time, now time.Time) (int64, error) {
	if userID == 0 || readAt.IsZero() || now.IsZero() {
		return 0, ErrInvalidInput
	}
	return toDBID(userID)
}

var _ Repository = (*SQLRepository)(nil)
