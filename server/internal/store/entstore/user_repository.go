package entstore

import (
	"context"
	"fmt"
	"math"

	"graft/server/internal/ent"
	entuser "graft/server/internal/ent/user"
	"graft/server/internal/store"
)

type userRepository struct {
	client *ent.Client
}

func (r *userRepository) GetByID(ctx context.Context, id uint64) (store.User, error) {
	if id == 0 {
		return store.User{}, store.ErrUserNotFound
	}
	if id > math.MaxInt {
		return store.User{}, store.ErrUserNotFound
	}

	record, err := r.client.User.Query().
		Where(entuser.IDEQ(int(id))).
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
