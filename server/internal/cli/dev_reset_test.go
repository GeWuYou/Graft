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
	"graft/server/internal/store"
)

type testDevResetFactory struct {
	auth store.AuthRepository
	rbac store.RBACRepository
}

func (f testDevResetFactory) Audit() store.AuditRepository {
	return nil
}

func (f testDevResetFactory) Users() store.UserRepository {
	return nil
}

func (f testDevResetFactory) Auth() store.AuthRepository {
	return f.auth
}

func (f testDevResetFactory) RBAC() store.RBACRepository {
	return f.rbac
}

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
	originalNewFactory := devResetNewFactory
	originalResetAdmin := devResetAdmin
	defer func() {
		devResetLoadConfig = originalLoadConfig
		devResetOpenDB = originalOpenDB
		devResetCloseDB = originalCloseDB
		devResetNewFactory = originalNewFactory
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
	devResetNewFactory = func(_ *database.Resources) (store.Factory, error) {
		steps = append(steps, "new-factory")
		return testDevResetFactory{
			auth: pluginTestAuthRepositoryStub{},
			rbac: pluginTestRBACRepositoryStub{},
		}, nil
	}
	devResetAdmin = func(_ context.Context, _ store.AuthRepository, _ store.RBACRepository) error {
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
		"new-factory",
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
	originalNewFactory := devResetNewFactory
	originalResetAdmin := devResetAdmin
	defer func() {
		devResetLoadConfig = originalLoadConfig
		devResetOpenDB = originalOpenDB
		devResetCloseDB = originalCloseDB
		devResetNewFactory = originalNewFactory
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
	devResetNewFactory = func(_ *database.Resources) (store.Factory, error) {
		return testDevResetFactory{
			auth: pluginTestAuthRepositoryStub{},
			rbac: pluginTestRBACRepositoryStub{},
		}, nil
	}
	devResetAdmin = func(context.Context, store.AuthRepository, store.RBACRepository) error {
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

type pluginTestAuthRepositoryStub struct{}

func (pluginTestAuthRepositoryStub) GetUserCredentialByUsername(context.Context, string) (store.UserCredential, error) {
	return store.UserCredential{}, nil
}

func (pluginTestAuthRepositoryStub) SetPasswordHash(context.Context, store.SetPasswordHashInput) error {
	return nil
}

func (pluginTestAuthRepositoryStub) EnsureUserCredential(context.Context, store.EnsureUserCredentialInput) (store.UserCredential, error) {
	return store.UserCredential{}, nil
}

func (pluginTestAuthRepositoryStub) CreateRefreshSession(context.Context, store.CreateRefreshSessionInput) (store.RefreshSession, error) {
	return store.RefreshSession{}, nil
}

func (pluginTestAuthRepositoryStub) GetRefreshSessionByTokenID(context.Context, string) (store.RefreshSession, error) {
	return store.RefreshSession{}, nil
}

func (pluginTestAuthRepositoryStub) RevokeRefreshSession(context.Context, store.RevokeRefreshSessionInput) error {
	return nil
}

func (pluginTestAuthRepositoryStub) RevokeRefreshSessionsByUserID(context.Context, store.RevokeRefreshSessionsByUserIDInput) error {
	return nil
}

func (pluginTestAuthRepositoryStub) RevokeOtherRefreshSessionsByUserID(context.Context, store.RevokeOtherRefreshSessionsInput) error {
	return nil
}

func (pluginTestAuthRepositoryStub) RevokeRefreshSessionByUserID(context.Context, store.RevokeRefreshSessionByUserIDInput) error {
	return nil
}

func (pluginTestAuthRepositoryStub) ListActiveRefreshSessionsByUserID(context.Context, store.ListActiveRefreshSessionsByUserIDInput) ([]store.RefreshSession, error) {
	return nil, nil
}

func (pluginTestAuthRepositoryStub) RotateRefreshSession(context.Context, store.RotateRefreshSessionInput) (store.RefreshSession, error) {
	return store.RefreshSession{}, nil
}

type pluginTestRBACRepositoryStub struct{}

func (pluginTestRBACRepositoryStub) EnsureRole(context.Context, store.EnsureRoleInput) (store.Role, error) {
	return store.Role{}, nil
}

func (pluginTestRBACRepositoryStub) EnsurePermission(context.Context, store.EnsurePermissionInput) (store.Permission, error) {
	return store.Permission{}, nil
}

func (pluginTestRBACRepositoryStub) CreateRole(context.Context, store.CreateRoleInput) (store.Role, error) {
	return store.Role{}, nil
}

func (pluginTestRBACRepositoryStub) UpdateRole(context.Context, store.UpdateRoleInput) (store.Role, error) {
	return store.Role{}, nil
}

func (pluginTestRBACRepositoryStub) AssignPermissionsToRole(context.Context, store.AssignPermissionsToRoleInput) error {
	return nil
}

func (pluginTestRBACRepositoryStub) ReplacePermissionsForRole(context.Context, store.ReplacePermissionsForRoleInput) error {
	return nil
}

func (pluginTestRBACRepositoryStub) AssignRoleToUser(context.Context, store.AssignRoleToUserInput) error {
	return nil
}

func (pluginTestRBACRepositoryStub) ReplaceRolesForUser(context.Context, store.ReplaceRolesForUserInput) error {
	return nil
}

func (pluginTestRBACRepositoryStub) GetRoleByID(context.Context, uint64) (store.Role, error) {
	return store.Role{}, nil
}

func (pluginTestRBACRepositoryStub) ListRolesByUserID(context.Context, uint64) ([]store.Role, error) {
	return nil, nil
}

func (pluginTestRBACRepositoryStub) ListRoles(context.Context) ([]store.Role, error) {
	return nil, nil
}

func (pluginTestRBACRepositoryStub) ListPermissionsByUserID(context.Context, uint64) ([]store.Permission, error) {
	return nil, nil
}

func (pluginTestRBACRepositoryStub) ListPermissions(context.Context) ([]store.Permission, error) {
	return nil, nil
}

func (pluginTestRBACRepositoryStub) ListRolePermissionBindings(context.Context, uint64) ([]store.RolePermissionBinding, error) {
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
