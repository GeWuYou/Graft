package entstore

import (
	"graft/server/internal/ent"
	"graft/server/internal/store"
)

// Factory is the Ent-backed implementation of store.Factory.
type Factory struct {
	userRepo store.UserRepository
}

// NewFactory wires the repository implementations backed by the provided Ent client.
func NewFactory(client *ent.Client) *Factory {
	return &Factory{
		userRepo: &userRepository{client: client},
	}
}

// Users returns the user repository implementation.
func (f *Factory) Users() store.UserRepository {
	return f.userRepo
}
