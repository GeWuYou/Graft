package storeent

import (
	"context"
	"fmt"

	usercontract "graft/server/modules/user/contract"
	ent "graft/server/modules/user/ent"
	userent "graft/server/modules/user/ent/user"
	userstore "graft/server/modules/user/store"
)

type userRepository struct {
	client *ent.Client
}

// NewUserRepository builds the user module's Ent-backed user repository.
func NewUserRepository(client *ent.Client) (userstore.UserRepository, error) {
	return newUserRepository(client)
}

func newUserRepository(client *ent.Client) (*userRepository, error) {
	if client == nil {
		return nil, fmt.Errorf("user storeent requires a non-nil ent client")
	}

	return &userRepository{client: client}, nil
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
		SetStatus(normalizeStoredUserStatus(input.Status)).
		SetPasswordHash(input.PasswordHash).
		SetMustChangePassword(input.MustChangePassword)
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
