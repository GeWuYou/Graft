package auth

import (
	"database/sql"
	"fmt"

	authstore "graft/server/modules/auth/store"
	"graft/server/modules/user/storeent"

	"go.uber.org/zap"
)

// NewRepositoryForDevelopmentReset exposes the auth-owned persistence entry
// used by the development-only reset command. It remains an adapter over the
// legacy user Ent model until the auth schema migration batch lands.
func NewRepositoryForDevelopmentReset(sqlDB *sql.DB) (authstore.AuthRepository, error) {
	runtime, err := storeent.NewRuntime(sqlDB, zap.NewNop())
	if err != nil {
		return nil, fmt.Errorf("build legacy auth persistence runtime: %w", err)
	}

	repository, err := runtime.NewAuthRepository()
	if err != nil {
		return nil, fmt.Errorf("build auth repository: %w", err)
	}

	return repository, nil
}
