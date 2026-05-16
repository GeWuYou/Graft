package entstore

import (
	"context"
	"fmt"

	"graft/server/internal/ent"
	entuser "graft/server/internal/ent/user"
	"graft/server/internal/store"
)

type userRepository struct {
	client *ent.Client
}

// GetByID 按 ID 查询用户，并将 Ent 模型转换为对上层稳定的 store.User。
func (r *userRepository) GetByID(ctx context.Context, id uint64) (store.User, error) {
	entID, err := toEntID(id)
	if err != nil {
		if err == store.ErrInvalidID {
			return store.User{}, store.ErrUserNotFound
		}
		return store.User{}, err
	}

	record, err := r.client.User.Query().
		Where(entuser.IDEQ(entID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return store.User{}, store.ErrUserNotFound
		}
		return store.User{}, fmt.Errorf("query user by id: %w", err)
	}

	return store.User{
		ID:        toStoreID(record.ID),
		Username:  record.Username,
		Display:   record.Display,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}, nil
}

// List 按稳定顺序返回当前全部用户记录，供最小只读用户列表契约复用。
func (r *userRepository) List(ctx context.Context) ([]store.User, error) {
	records, err := r.client.User.Query().
		Order(ent.Asc(entuser.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]store.User, 0, len(records))
	for _, record := range records {
		users = append(users, store.User{
			ID:        toStoreID(record.ID),
			Username:  record.Username,
			Display:   record.Display,
			CreatedAt: record.CreatedAt,
			UpdatedAt: record.UpdatedAt,
		})
	}

	return users, nil
}
