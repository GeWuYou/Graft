package auth

import (
	"database/sql"
	"fmt"

	"graft/server/internal/moduleapi"
	authstore "graft/server/modules/auth/store"
	"graft/server/modules/auth/storeent"
)

// NewRepositoryForDevelopmentReset 创建仅供开发环境重置命令使用的 auth 持久化仓储；数据库客户端或仓储初始化失败时返回错误。
func NewRepositoryForDevelopmentReset(sqlDB *sql.DB, identity moduleapi.UserIdentityProvider) (authstore.AuthRepository, error) {
	client, err := storeent.NewClient(sqlDB)
	if err != nil {
		return nil, fmt.Errorf("build auth persistence client: %w", err)
	}

	repository, err := storeent.NewAuthRepository(client, identity)
	if err != nil {
		return nil, fmt.Errorf("build auth repository: %w", err)
	}

	return repository, nil
}
