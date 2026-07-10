package auth

import (
	"database/sql"
	"fmt"

	"graft/server/internal/moduleapi"
	authstore "graft/server/modules/auth/store"
	"graft/server/modules/auth/storeent"
)

// NewRepositoryForDevelopmentReset exposes the auth-owned persistence entry
// used by the development-only reset command.
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
