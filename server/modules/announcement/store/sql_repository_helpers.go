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

// normalizeCreateInput 规范化创建公告的输入，裁剪文本字段并将时间字段统一转换为 UTC。
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

// normalizeUpdateInput 规范化更新公告输入，裁剪文本字段并将时间字段转换为 UTC。
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

// normalizeListQuery 规范化公告列表查询条件。
// 去除状态、等级、关键词和排序字段的首尾空白，并修正分页参数：当 Limit 小于等于 0 时使用默认值，超过上限时截断为最大值，Offset 小于 0 时重置为 0。
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

// normalizeUserListQuery 校验并规范化用户列表查询条件。
// 它要求 Now 和 UserID 有效，将 UserID 转为数据库 ID，统一 Now 为 UTC，并对 Limit 和 Offset 应用默认值与边界限制。
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

// 结果始终包含删除标记条件，并按需追加状态、等级、置顶和关键词匹配条件。
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

// buildUserVisibleWhere 生成用户可见公告查询的 WHERE 条件。
// 条件包含未删除、已发布、发布时间已到且未过期的公告；当 unreadOnly 为 true 时，额外加入未读条件。
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

// 支持置顶优先发布时间、发布时间倒序和更新时间倒序；未知值使用更新时间倒序。
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

// announcementColumns 返回 announcements 表的列清单。
// 该列清单包含主键、内容字段、状态与投递信息、置顶与时间戳字段，以及创建、更新和删除相关用户 ID。
func announcementColumns() string {
	return `id, title, content, level, status, delivery_mode, pinned, publish_at, published_at, published_by, archived_at, expire_at,
		created_by, updated_by, deleted_by, created_at, updated_at, deleted_at`
}

// prefixedAnnouncementColumns 返回带有指定前缀的公告列名列表。
// 它会为 `announcementColumns` 中的每个列名前加上 `prefix + "."` 并用逗号分隔。
func prefixedAnnouncementColumns(prefix string) string {
	columns := strings.Split(announcementColumns(), ",")
	for index, column := range columns {
		columns[index] = prefix + "." + strings.TrimSpace(column)
	}
	return strings.Join(columns, ", ")
}

// scanAnnouncement 将单行结果扫描为公告对象，并把可空时间与用户 ID 转换为结构体字段。
// 可空时间字段会映射为对应的时间指针，数据库用户 ID 会转换为 uint64 指针。
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

// optionalUint64FromDBID 将可空的数据库 ID 转换为可选的 uint64 指针。
// @returns 当值无效时返回 nil；当值有效时返回转换后的 uint64 指针。
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

// scanAnnouncements 扫描并返回公告列表。
// 遍历查询结果并将每一行映射为 Announcement。
//
// @param rows 查询结果集。
// @returns 扫描得到的公告列表；或在扫描、迭代失败时返回错误。
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

// scanUserAnnouncement 扫描一条公告及其用户读取时间。
// 当读取时间存在时，返回的 ReadAt 会转换为 UTC。
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

// scanUserAnnouncements 扫描当前用户公告查询结果并返回列表。
// 解析过程中出现的扫描错误或行迭代错误会被包装后返回。
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

// closeRows 关闭非空的查询结果集。
func closeRows(rows *sql.Rows) {
	if rows != nil {
		_ = rows.Close()
	}
}

// nullableUint64 将值为 0 的无符号整数转换为 nil。
//
// @return value 为 0 时返回 nil；否则返回原值。
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

// detectSQLDialect 根据数据库驱动推断 SQL 方言。
// 当 db 或其驱动不可用时，默认返回 PostgreSQL 方言。
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

// 对 SQLite 返回 `?` 风格，其余方言返回 `$1, $2, ...` 风格。
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

// toDBID 将业务侧的 ID 转换为数据库 ID。
// @return 返回转换后的数据库 ID；当值为 0 或超出 int64 范围时返回 ErrInvalidInput。
func toDBID(value uint64) (int64, error) {
	if value == 0 || value > uint64(^uint64(0)>>1) {
		return 0, ErrInvalidInput
	}
	return int64(value), nil
}

// uint64FromDBID 将数据库中的整数 ID 转换为 uint64。
// @returns 值对应的 uint64，或在值为负数时返回错误。
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

// readAccessIDs 将用户和公告标识转换为读访问记录所需的数据库 ID。
// 当 announcementID 为 0 时返回 ErrInvalidInput；否则返回用户 ID 和公告 ID 对应的数据库 ID。
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

// userReadDBID 将用户标识转换为数据库 ID，并校验读写访问所需的输入有效性。
// 当 userID、readAt 或 now 无效时返回 ErrInvalidInput。
//
// @return userID 对应的数据库 ID；若输入无效则返回错误。
func userReadDBID(userID uint64, readAt time.Time, now time.Time) (int64, error) {
	if userID == 0 || readAt.IsZero() || now.IsZero() {
		return 0, ErrInvalidInput
	}
	return toDBID(userID)
}

var _ Repository = (*SQLRepository)(nil)
