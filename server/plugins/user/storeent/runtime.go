package storeent

import (
	"database/sql"
	"fmt"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"go.uber.org/zap"

	ent "graft/server/plugins/user/ent"
)

// Runtime owns the user plugin's shared Ent client wiring for one process.
//
// The client reuses the core-owned *sql.DB pool. Core keeps ownership of closing
// that pool, so this runtime only builds the plugin-local Ent surface.
type Runtime struct {
	client *ent.Client
}

// NewRuntime builds the user plugin's Ent runtime on top of the shared SQL pool.
func NewRuntime(sqlDB *sql.DB) (*Runtime, error) {
	if sqlDB == nil {
		return nil, fmt.Errorf("user storeent runtime requires a non-nil sql db")
	}

	driver := entsql.OpenDB("postgres", sqlDB)
	return &Runtime{
		client: ent.NewClient(
			ent.Driver(driver),
			ent.Log(func(args ...any) {
				message := strings.TrimSpace(fmt.Sprint(args...))
				if message == "" {
					return
				}

				zap.L().Debug("ent debug",
					zap.String("plugin", "user"),
					zap.String("component", "ent"),
					zap.String("message", message),
				)
			}),
		),
	}, nil
}

// NewUserRepository builds the plugin-owned user repository from the shared Ent client.
func (r *Runtime) NewUserRepository() (*userRepository, error) {
	return newUserRepository(r.client)
}

// NewAuthRepository builds the plugin-owned auth/session repository from the shared Ent client.
func (r *Runtime) NewAuthRepository() (*authRepository, error) {
	return newAuthRepository(r.client)
}

// Client exposes the shared Ent client for the narrow cases that still need direct client access.
func (r *Runtime) Client() *ent.Client {
	return r.client
}
