package auth

import (
	"database/sql"
	"fmt"

	"graft/server/internal/moduleapi"
	authstore "graft/server/modules/auth/store"
	"graft/server/modules/auth/storeent"
)

// NewRepositoryForDevelopmentReset exposes the auth-owned persistence entry
// NewRepositoryForDevelopmentReset creates the authentication repository used by the development-only reset command.
// It returns an error if the persistence client or authentication repository cannot be created.
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
