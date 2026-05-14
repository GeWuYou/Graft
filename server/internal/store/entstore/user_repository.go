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
		ID:        uint64(record.ID),
		Username:  record.Username,
		Display:   record.Display,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}, nil
}
