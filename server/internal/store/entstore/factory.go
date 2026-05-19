package entstore

import (
	"errors"

	"graft/server/internal/ent"
	"graft/server/internal/store"
)

// Factory 是 store.Factory 的 Ent 实现。
//
// 它只负责装配仓储，不拥有传入 Ent 客户端的生命周期。
type Factory struct {
	userRepo store.UserRepository
	authRepo store.AuthRepository
}

// NewFactory 使用传入的 Ent 客户端装配各个仓储实现。
//
// 调用方必须保证 client 在整个仓储使用期间保持可用，并由更上层统一关闭。
func NewFactory(client *ent.Client) (*Factory, error) {
	if client == nil {
		return nil, errors.New("entstore factory requires a non-nil ent client")
	}

	return &Factory{
		userRepo: &userRepository{client: client},
		authRepo: &authRepository{client: client},
	}, nil
}

// Users 返回复用同一 Ent 客户端的用户仓储实现。
func (f *Factory) Users() store.UserRepository {
	return f.userRepo
}

// Auth 返回复用同一 Ent 客户端的认证仓储实现。
func (f *Factory) Auth() store.AuthRepository {
	return f.authRepo
}
