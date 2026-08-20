package storeent

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	usercontract "graft/server/modules/user/contract"
	ent "graft/server/modules/user/ent"
	"graft/server/modules/user/ent/predicate"
	userent "graft/server/modules/user/ent/user"
	userstore "graft/server/modules/user/store"
)

type userRepository struct {
	client *ent.Client
	db     *sql.DB
}

// NewUserRepository 构建 user 模块基于 Ent 的用户仓储；客户端为空时返回错误。
func NewUserRepository(client *ent.Client, db ...*sql.DB) (userstore.UserRepository, error) {
	return newUserRepository(client, db...)
}

func newUserRepository(client *ent.Client, db ...*sql.DB) (*userRepository, error) {
	if client == nil {
		return nil, fmt.Errorf("user storeent requires a non-nil ent client")
	}

	repository := &userRepository{client: client}
	if len(db) > 0 {
		repository.db = db[0]
	}
	return repository, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uint64) (userstore.User, error) {
	entID, err := toEntID(id)
	if err != nil {
		if err == userstore.ErrInvalidID {
			return userstore.User{}, userstore.ErrUserNotFound
		}
		return userstore.User{}, err
	}

	record, err := r.client.User.Query().
		Where(
			userent.IDEQ(entID),
			userent.DeletedAtEQ(0),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return userstore.User{}, userstore.ErrUserNotFound
		}
		return userstore.User{}, fmt.Errorf("query user by id: %w", err)
	}

	return userstore.User{
		ID:                    toStoreID(record.ID),
		Username:              record.Username,
		Display:               record.Display,
		Status:                normalizeStoredUserStatus(record.Status),
		ProtectedDefaultAdmin: isProtectedDefaultAdminUsername(record.Username),
		CreatedAt:             record.CreatedAt,
		UpdatedAt:             record.UpdatedAt,
	}, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (userstore.User, error) {
	record, err := r.client.User.Query().
		Where(
			userent.UsernameEQ(username),
			userent.DeletedAtEQ(0),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return userstore.User{}, userstore.ErrUserNotFound
		}
		return userstore.User{}, fmt.Errorf("query user by username: %w", err)
	}

	return toStoreUser(record), nil
}

func (r *userRepository) List(ctx context.Context) ([]userstore.User, error) {
	records, err := r.client.User.Query().
		Where(userent.DeletedAtEQ(0)).
		Order(ent.Asc(userent.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]userstore.User, 0, len(records))
	for _, record := range records {
		users = append(users, userstore.User{
			ID:                    toStoreID(record.ID),
			Username:              record.Username,
			Display:               record.Display,
			Status:                normalizeStoredUserStatus(record.Status),
			ProtectedDefaultAdmin: isProtectedDefaultAdminUsername(record.Username),
			CreatedAt:             record.CreatedAt,
			UpdatedAt:             record.UpdatedAt,
		})
	}

	return users, nil
}

// ListPage 在数据库侧应用用户管理筛选、总数统计和稳定分页。
func (r *userRepository) ListPage(ctx context.Context, filter userstore.UserListFilter) ([]userstore.User, int, error) {
	if filter.Limit <= 0 || filter.Offset < 0 {
		return nil, 0, fmt.Errorf("invalid user page bounds")
	}

	predicates, empty, err := userListPredicates(filter)
	if err != nil {
		return nil, 0, err
	}
	if empty {
		return []userstore.User{}, 0, nil
	}

	query := r.client.User.Query().Where(predicates...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count paged users: %w", err)
	}
	records, err := query.Order(ent.Asc(userent.FieldID)).Offset(filter.Offset).Limit(filter.Limit).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list paged users: %w", err)
	}
	users := make([]userstore.User, 0, len(records))
	for _, record := range records {
		users = append(users, toStoreUser(record))
	}
	return users, total, nil
}

func userListPredicates(filter userstore.UserListFilter) ([]predicate.User, bool, error) {
	predicates := []predicate.User{userent.DeletedAtEQ(0)}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		predicates = append(predicates, userent.Or(userent.UsernameContainsFold(keyword), userent.DisplayContainsFold(keyword)))
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		predicates = append(predicates, userent.StatusEQ(status))
	}
	if filter.UserIDs == nil {
		return predicates, false, nil
	}
	if len(filter.UserIDs) == 0 {
		return nil, true, nil
	}
	ids := make([]int, 0, len(filter.UserIDs))
	for _, userID := range filter.UserIDs {
		id, err := toEntID(userID)
		if err != nil {
			return nil, false, fmt.Errorf("convert role-filter user id: %w", err)
		}
		ids = append(ids, id)
	}
	return append(predicates, userent.IDIn(ids...)), false, nil
}

// ListCandidates 在数据库侧完成候选搜索与分页，避免跨模块调用读取完整用户列表后再过滤。
func (r *userRepository) ListCandidates(ctx context.Context, query userstore.UserCandidateQuery) ([]userstore.User, int, error) {
	if query.Limit <= 0 || query.Offset < 0 {
		return nil, 0, fmt.Errorf("invalid user candidate page bounds")
	}

	predicates := []predicate.User{userent.DeletedAtEQ(0)}
	if search := strings.TrimSpace(query.Search); search != "" {
		predicates = append(predicates, userent.Or(userent.UsernameContainsFold(search), userent.DisplayContainsFold(search)))
	}

	users := r.client.User.Query().Where(predicates...)
	total, err := users.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count user candidates: %w", err)
	}
	records, err := users.Order(ent.Asc(userent.FieldID)).Offset(query.Offset).Limit(query.Limit).
		Select(userent.FieldID, userent.FieldUsername, userent.FieldDisplay, userent.FieldStatus).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list user candidates: %w", err)
	}
	items := make([]userstore.User, 0, len(records))
	for _, record := range records {
		items = append(items, userstore.User{ID: toStoreID(record.ID), Username: record.Username, Display: record.Display, Status: normalizeStoredUserStatus(record.Status)})
	}
	return items, total, nil
}

// ListSummariesByIDs 使用单次 ID 集合查询返回有效用户摘要，未命中或已删除用户不会伪造占位记录。
func (r *userRepository) ListSummariesByIDs(ctx context.Context, userIDs []uint64) ([]userstore.User, error) {
	if len(userIDs) == 0 {
		return []userstore.User{}, nil
	}
	ids := make([]int, 0, len(userIDs))
	for _, userID := range userIDs {
		id, err := toEntID(userID)
		if err != nil {
			return nil, fmt.Errorf("convert user summary id: %w", err)
		}
		ids = append(ids, id)
	}
	records, err := r.client.User.Query().
		Where(userent.DeletedAtEQ(0), userent.IDIn(ids...)).
		Order(ent.Asc(userent.FieldID)).
		Select(userent.FieldID, userent.FieldUsername, userent.FieldDisplay, userent.FieldStatus).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user summaries by ids: %w", err)
	}
	items := make([]userstore.User, 0, len(records))
	for _, record := range records {
		items = append(items, userstore.User{
			ID:       toStoreID(record.ID),
			Username: record.Username,
			Display:  record.Display,
			Status:   normalizeStoredUserStatus(record.Status),
		})
	}
	return items, nil
}

func (r *userRepository) ListSecuritySummaries(ctx context.Context, afterID uint64, limit int) ([]userstore.User, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("security summary limit must be positive")
	}
	afterEntID := 0
	if afterID != 0 {
		var err error
		afterEntID, err = toEntID(afterID)
		if err != nil {
			return nil, fmt.Errorf("convert security summary cursor: %w", err)
		}
	}
	records, err := r.client.User.Query().
		Where(userent.DeletedAtEQ(0), userent.IDGT(afterEntID)).
		Order(ent.Asc(userent.FieldID)).
		Limit(limit).
		Select(userent.FieldID, userent.FieldStatus).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list security user summaries: %w", err)
	}
	summaries := make([]userstore.User, 0, len(records))
	for _, record := range records {
		summaries = append(summaries, userstore.User{ID: toStoreID(record.ID), Status: normalizeStoredUserStatus(record.Status)})
	}
	return summaries, nil
}

func (r *userRepository) Count(ctx context.Context) (int, error) {
	total, err := r.client.User.Query().
		Where(userent.DeletedAtEQ(0)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return total, nil
}

func (r *userRepository) Create(ctx context.Context, input userstore.CreateUserInput) (userstore.User, error) {
	builder := r.client.User.Create().
		SetUsername(input.Username).
		SetDisplay(input.Display).
		SetStatus(normalizeStoredUserStatus(input.Status))
	if input.ActorID != 0 {
		builder = builder.SetCreatedBy(input.ActorID).SetUpdatedBy(input.ActorID)
	}

	record, err := builder.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return userstore.User{}, userstore.ErrUsernameConflict
		}
		return userstore.User{}, fmt.Errorf("create user: %w", err)
	}

	return toStoreUser(record), nil
}

func (r *userRepository) Update(ctx context.Context, input userstore.UpdateUserInput) (userstore.User, error) {
	id, err := toEntID(input.ID)
	if err != nil {
		if err == userstore.ErrInvalidID {
			return userstore.User{}, userstore.ErrUserNotFound
		}
		return userstore.User{}, err
	}

	builder := r.client.User.UpdateOneID(id).
		Where(userent.DeletedAtEQ(0)).
		SetUsername(input.Username).
		SetDisplay(input.Display)
	if input.ActorID != 0 {
		builder = builder.SetUpdatedBy(input.ActorID)
	}

	record, err := builder.Save(ctx)
	if err != nil {
		switch {
		case ent.IsConstraintError(err):
			return userstore.User{}, userstore.ErrUsernameConflict
		case ent.IsNotFound(err):
			return userstore.User{}, userstore.ErrUserNotFound
		default:
			return userstore.User{}, fmt.Errorf("update user: %w", err)
		}
	}

	return toStoreUser(record), nil
}

func (r *userRepository) SetStatus(ctx context.Context, input userstore.SetUserStatusInput) (userstore.User, error) {
	id, err := toEntID(input.ID)
	if err != nil {
		if err == userstore.ErrInvalidID {
			return userstore.User{}, userstore.ErrUserNotFound
		}
		return userstore.User{}, err
	}

	builder := r.client.User.UpdateOneID(id).
		Where(userent.DeletedAtEQ(0)).
		SetStatus(normalizeStoredUserStatus(input.Status))
	if input.ActorID != 0 {
		builder = builder.SetUpdatedBy(input.ActorID)
	}

	record, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return userstore.User{}, userstore.ErrUserNotFound
		}
		return userstore.User{}, fmt.Errorf("set user status: %w", err)
	}

	return toStoreUser(record), nil
}

func (r *userRepository) Delete(ctx context.Context, input userstore.DeleteUserInput) error {
	id, err := toEntID(input.ID)
	if err != nil {
		if err == userstore.ErrInvalidID {
			return userstore.ErrUserNotFound
		}
		return err
	}

	builder := r.client.User.UpdateOneID(id).
		Where(userent.DeletedAtEQ(0)).
		SetDeletedAt(input.DeletedAt.UTC().Unix()).
		SetDeletedBy(input.ActorID)
	if input.ActorID != 0 {
		builder = builder.SetUpdatedBy(input.ActorID)
	}

	if err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return userstore.ErrUserNotFound
		}
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

func normalizeStoredUserStatus(status string) string {
	switch status {
	case usercontract.UserStatusDisabled:
		return usercontract.UserStatusDisabled
	default:
		return usercontract.UserStatusEnabled
	}
}

// 结果包含用户 ID、用户名、显示名、状态、受保护的默认管理员标记以及创建和更新时间。
func toStoreUser(record *ent.User) userstore.User {
	return userstore.User{
		ID:                    toStoreID(record.ID),
		Username:              record.Username,
		Display:               record.Display,
		Status:                normalizeStoredUserStatus(record.Status),
		ProtectedDefaultAdmin: isProtectedDefaultAdminUsername(record.Username),
		CreatedAt:             record.CreatedAt,
		UpdatedAt:             record.UpdatedAt,
	}
}

// isProtectedDefaultAdminUsername 判断用户名是否属于受保护的默认管理员账号。
// @return true 如果用户名属于受保护的默认管理员账号，false 否则。
func isProtectedDefaultAdminUsername(username string) bool {
	return userstore.IsProtectedDefaultAdminUsername(username)
}
