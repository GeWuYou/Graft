package storeent

import (
	"context"
	"fmt"

	ent "graft/server/plugins/user/ent"
	userent "graft/server/plugins/user/ent/user"
	userstore "graft/server/plugins/user/store"
)

type userRepository struct {
	client *ent.Client
}

// NewUserRepository builds the user plugin's Ent-backed user repository.
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
		Where(userent.IDEQ(entID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return userstore.User{}, userstore.ErrUserNotFound
		}
		return userstore.User{}, fmt.Errorf("query user by id: %w", err)
	}

	return userstore.User{
		ID:        toStoreID(record.ID),
		Username:  record.Username,
		Display:   record.Display,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}, nil
}

func (r *userRepository) List(ctx context.Context) ([]userstore.User, error) {
	records, err := r.client.User.Query().
		Order(ent.Asc(userent.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]userstore.User, 0, len(records))
	for _, record := range records {
		users = append(users, userstore.User{
			ID:        toStoreID(record.ID),
			Username:  record.Username,
			Display:   record.Display,
			CreatedAt: record.CreatedAt,
			UpdatedAt: record.UpdatedAt,
		})
	}

	return users, nil
}
