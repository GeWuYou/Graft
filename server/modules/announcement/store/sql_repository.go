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

// SQLRepository 将公告记录和用户阅读事实持久化到公告模块自有表中，并统一处理 SQL 方言和软删除边界。
type SQLRepository struct {
	db          *sql.DB
	placeholder placeholderStyle
	dialect     sqlDialect
}

// NewSQLRepository 使用给定数据库连接创建公告仓储；数据库连接为空时拒绝构建。
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("announcement repository requires a non-nil sql db")
	}
	dialect := detectSQLDialect(db)
	return &SQLRepository{db: db, placeholder: placeholderStyleForDialect(dialect), dialect: dialect}, nil
}

// Ping 验证公告仓储可以访问底层 SQL 依赖。
func (r *SQLRepository) Ping(ctx context.Context) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	return r.db.PingContext(ctx)
}

// ListAdmin 返回管理端未删除公告，并按查询条件提供稳定分页结果。
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

// ListCurrentUser 返回指定用户当前可见的公告及阅读状态；可见性由发布时间、过期时间和软删除条件共同决定。
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

// Create 写入一条管理端公告；调用方应在服务层完成状态、时间和内容约束校验。
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

// GetAdmin 按 ID 返回一条未删除公告；不存在或已删除记录统一返回未找到错误。
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

// Update 更新一条未删除公告的可编辑字段，不覆盖生命周期状态和删除事实。
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

// Publish 将公告标记为已发布并记录本次发布动作时间；状态转换约束由服务层负责。
func (r *SQLRepository) Publish(
	ctx context.Context,
	id uint64,
	publishAt *time.Time,
	publishedAt time.Time,
	actorID *uint64,
) (Announcement, error) {
	if err := r.ensureReady(); err != nil {
		return Announcement{}, err
	}
	targetID, err := toDBID(id)
	if err != nil {
		return Announcement{}, err
	}
	if publishedAt.IsZero() {
		return Announcement{}, ErrInvalidInput
	}
	var effectivePublishAt *time.Time
	if publishAt != nil {
		normalized := publishAt.UTC()
		effectivePublishAt = &normalized
	}
	publishedAt = publishedAt.UTC()
	item, err := scanAnnouncement(r.db.QueryRowContext(ctx, r.placeholder.rebind(`UPDATE announcements
		SET status = ?, publish_at = ?, published_at = ?, published_by = ?, archived_at = NULL, updated_by = ?, updated_at = ?
		WHERE id = ? AND deleted_at = 0
		RETURNING `+announcementColumns()),
		statusPublished,
		effectivePublishAt,
		publishedAt,
		actorID,
		actorID,
		publishedAt,
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

// Archive 将一条未删除公告标记为已归档，使其退出当前用户可见范围。
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

// Delete 软删除一条未删除公告；删除不会抹去公告的生命周期和审计字段。
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

// MarkRead 为当前可见公告记录指定用户的阅读事实；重复标记保持已有阅读时间语义。
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

// MarkAllRead 为指定用户当前可见的未读公告批量记录阅读事实，并返回实际影响数量。
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

// UnreadCount 统计指定用户当前可见且未读的公告数量。
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
