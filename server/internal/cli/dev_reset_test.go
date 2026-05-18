package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/database"
	"graft/server/internal/ent"
	"graft/server/internal/store"
	"graft/server/plugins/user"
	userstore "graft/server/plugins/user/store"
)

func TestRunDevResetAdminRejectsNonDevelopmentEnv(t *testing.T) {
	originalLoadConfig := devResetLoadConfig
	originalOpenDB := devResetOpenDB
	defer func() {
		devResetLoadConfig = originalLoadConfig
		devResetOpenDB = originalOpenDB
	}()

	devResetLoadConfig = func() (*config.Config, error) {
		return &config.Config{App: config.AppConfig{Env: "production"}}, nil
	}
	devResetOpenDB = func(config.DatabaseConfig) (*database.Resources, error) {
		t.Fatal("database should not be opened for non-development env")
		return nil, nil
	}

	err := runDevResetAdmin(&cobra.Command{})
	if err == nil {
		t.Fatal("expected reset-admin env guard error")
	}
	if !strings.Contains(err.Error(), "only available in local/test environments") {
		t.Fatalf("expected development env guard, got %v", err)
	}
}

func TestRunDevResetAdminResetsDefaultAdmin(t *testing.T) {
	originalLoadConfig := devResetLoadConfig
	originalOpenDB := devResetOpenDB
	originalCloseDB := devResetCloseDB
	originalNewAuthRepository := devResetNewAuthRepository
	originalResolveRBACRepository := devResetResolveRBACRepository
	originalResetAdmin := devResetAdmin
	defer func() {
		devResetLoadConfig = originalLoadConfig
		devResetOpenDB = originalOpenDB
		devResetCloseDB = originalCloseDB
		devResetNewAuthRepository = originalNewAuthRepository
		devResetResolveRBACRepository = originalResolveRBACRepository
		devResetAdmin = originalResetAdmin
	}()

	var steps []string
	devResetLoadConfig = func() (*config.Config, error) {
		steps = append(steps, "load-config")
		return testDevResetConfig("local"), nil
	}
	devResetOpenDB = func(cfg config.DatabaseConfig) (*database.Resources, error) {
		steps = append(steps, "open-db:"+cfg.URL)
		return &database.Resources{}, nil
	}
	devResetCloseDB = func(*database.Resources) error {
		steps = append(steps, "close-db")
		return nil
	}
	devResetNewAuthRepository = func(_ *ent.Client) (user.AuthRepositoryForReset, error) {
		steps = append(steps, "new-auth-repository")
		return userAuthRepositoryForResetStub{}, nil
	}
	devResetResolveRBACRepository = func(*database.Resources) (store.RBACRepository, error) {
		steps = append(steps, "new-rbac-repository")
		return rbacRepositoryForBootstrapStub{}, nil
	}
	devResetAdmin = func(_ context.Context, _ user.AuthRepositoryForReset, _ store.RBACRepository) error {
		steps = append(steps, "reset-admin")
		return nil
	}

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	err := runDevResetAdmin(cmd)
	if err != nil {
		t.Fatalf("run reset-admin: %v", err)
	}

	expectedSteps := []string{
		"load-config",
		"open-db:" + testDevResetDatabaseURL(),
		"new-auth-repository",
		"new-rbac-repository",
		"reset-admin",
		"close-db",
	}
	if strings.Join(steps, "|") != strings.Join(expectedSteps, "|") {
		t.Fatalf("expected steps %v, got %v", expectedSteps, steps)
	}
	if !strings.Contains(stdout.String(), "username=graft password=graft-admin must_change_password=true") {
		t.Fatalf("expected reset-admin output, got %q", stdout.String())
	}
}

func TestRunDevResetAdminWrapsResetFailure(t *testing.T) {
	originalLoadConfig := devResetLoadConfig
	originalOpenDB := devResetOpenDB
	originalCloseDB := devResetCloseDB
	originalNewAuthRepository := devResetNewAuthRepository
	originalResolveRBACRepository := devResetResolveRBACRepository
	originalResetAdmin := devResetAdmin
	defer func() {
		devResetLoadConfig = originalLoadConfig
		devResetOpenDB = originalOpenDB
		devResetCloseDB = originalCloseDB
		devResetNewAuthRepository = originalNewAuthRepository
		devResetResolveRBACRepository = originalResolveRBACRepository
		devResetAdmin = originalResetAdmin
	}()

	devResetLoadConfig = func() (*config.Config, error) {
		return testDevResetConfig("test"), nil
	}
	devResetOpenDB = func(config.DatabaseConfig) (*database.Resources, error) {
		return &database.Resources{}, nil
	}
	devResetCloseDB = func(*database.Resources) error {
		return nil
	}
	devResetNewAuthRepository = func(_ *ent.Client) (user.AuthRepositoryForReset, error) {
		return userAuthRepositoryForResetStub{}, nil
	}
	devResetResolveRBACRepository = func(*database.Resources) (store.RBACRepository, error) {
		return rbacRepositoryForBootstrapStub{}, nil
	}
	devResetAdmin = func(context.Context, user.AuthRepositoryForReset, store.RBACRepository) error {
		return errors.New("boom")
	}

	err := runDevResetAdmin(&cobra.Command{})
	if err == nil {
		t.Fatal("expected reset-admin failure")
	}
	if !strings.Contains(err.Error(), "reset default admin") {
		t.Fatalf("expected reset-admin context, got %v", err)
	}
}

type userAuthRepositoryForResetStub struct{}

func (userAuthRepositoryForResetStub) GetUserCredentialByUsername(context.Context, string) (userstore.UserCredential, error) {
	return userstore.UserCredential{}, nil
}

func (userAuthRepositoryForResetStub) SetPasswordHash(context.Context, userstore.SetPasswordHashInput) error {
	return nil
}

func (userAuthRepositoryForResetStub) EnsureUserCredential(context.Context, userstore.EnsureUserCredentialInput) (userstore.UserCredential, error) {
	return userstore.UserCredential{}, nil
}

func (userAuthRepositoryForResetStub) CreateRefreshSession(context.Context, userstore.CreateRefreshSessionInput) (userstore.RefreshSession, error) {
	return userstore.RefreshSession{}, nil
}

func (userAuthRepositoryForResetStub) GetRefreshSessionByTokenID(context.Context, string) (userstore.RefreshSession, error) {
	return userstore.RefreshSession{}, nil
}

func (userAuthRepositoryForResetStub) RevokeRefreshSession(context.Context, userstore.RevokeRefreshSessionInput) error {
	return nil
}

func (userAuthRepositoryForResetStub) RevokeRefreshSessionsByUserID(context.Context, userstore.RevokeRefreshSessionsByUserIDInput) error {
	return nil
}

func (userAuthRepositoryForResetStub) RevokeOtherRefreshSessionsByUserID(context.Context, userstore.RevokeOtherRefreshSessionsInput) error {
	return nil
}

func (userAuthRepositoryForResetStub) RevokeRefreshSessionByUserID(context.Context, userstore.RevokeRefreshSessionByUserIDInput) error {
	return nil
}

func (userAuthRepositoryForResetStub) ListActiveRefreshSessionsByUserID(context.Context, userstore.ListActiveRefreshSessionsByUserIDInput) ([]userstore.RefreshSession, error) {
	return nil, nil
}

func (userAuthRepositoryForResetStub) RotateRefreshSession(context.Context, userstore.RotateRefreshSessionInput) (userstore.RefreshSession, error) {
	return userstore.RefreshSession{}, nil
}

type rbacRepositoryForBootstrapStub struct{}

func (rbacRepositoryForBootstrapStub) EnsureRole(context.Context, store.EnsureRoleInput) (store.Role, error) {
	return store.Role{}, nil
}

func (rbacRepositoryForBootstrapStub) EnsurePermission(context.Context, store.EnsurePermissionInput) (store.Permission, error) {
	return store.Permission{}, nil
}

func (rbacRepositoryForBootstrapStub) CreateRole(context.Context, store.CreateRoleInput) (store.Role, error) {
	return store.Role{}, nil
}

func (rbacRepositoryForBootstrapStub) UpdateRole(context.Context, store.UpdateRoleInput) (store.Role, error) {
	return store.Role{}, nil
}

func (rbacRepositoryForBootstrapStub) AssignPermissionsToRole(context.Context, store.AssignPermissionsToRoleInput) error {
	return nil
}

func (rbacRepositoryForBootstrapStub) ReplacePermissionsForRole(context.Context, store.ReplacePermissionsForRoleInput) error {
	return nil
}

func (rbacRepositoryForBootstrapStub) AssignRoleToUser(context.Context, store.AssignRoleToUserInput) error {
	return nil
}

func (rbacRepositoryForBootstrapStub) ReplaceRolesForUser(context.Context, store.ReplaceRolesForUserInput) error {
	return nil
}

func (rbacRepositoryForBootstrapStub) GetRoleByID(context.Context, uint64) (store.Role, error) {
	return store.Role{}, nil
}

func (rbacRepositoryForBootstrapStub) ListRolesByUserID(context.Context, uint64) ([]store.Role, error) {
	return nil, nil
}

func (rbacRepositoryForBootstrapStub) ListRoles(context.Context) ([]store.Role, error) {
	return nil, nil
}

func (rbacRepositoryForBootstrapStub) ListPermissionsByUserID(context.Context, uint64) ([]store.Permission, error) {
	return nil, nil
}

func (rbacRepositoryForBootstrapStub) ListPermissions(context.Context) ([]store.Permission, error) {
	return nil, nil
}

func (rbacRepositoryForBootstrapStub) ListRolePermissionBindings(context.Context, uint64) ([]store.RolePermissionBinding, error) {
	return nil, nil
}

func testDevResetConfig(env string) *config.Config {
	return &config.Config{
		App:      config.AppConfig{Env: env},
		Database: config.DatabaseConfig{Driver: "postgres", URL: testDevResetDatabaseURL()},
	}
}

func testDevResetDatabaseURL() string {
	return "postgres://" + "graft:" + "***" + "@localhost:5432/graft?sslmode=disable"
}
