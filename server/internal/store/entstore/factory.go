package entstore

import (
	"graft/server/internal/ent"
	"graft/server/internal/store"
)

// Factory 是 store.Factory 的 Ent 实现。
type Factory struct {
	userRepo store.UserRepository
}

// NewFactory 使用传入的 Ent 客户端装配各个仓储实现。
func NewFactory(client *ent.Client) *Factory {
	return &Factory{
		userRepo: &userRepository{client: client},
	}
}

// Users 返回用户仓储实现。
func (f *Factory) Users() store.UserRepository {
	return f.userRepo
}
