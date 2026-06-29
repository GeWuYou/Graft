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
		"(a.publish_at IS NULL OR a.publish_at <= ?)",
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
	return `id, title, content, level, status, delivery_mode, pinned, publish_at, published_at, published_by, archived_at, expire_at,
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
	var publishedAt sql.NullTime
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
		&publishedAt,
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
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
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
