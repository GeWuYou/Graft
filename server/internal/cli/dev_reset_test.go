package cli

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/database"
	"graft/server/internal/pluginapi"
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
	originalResolveRBACBootstrap := devResetResolveRBACBootstrap
	originalResetAdmin := devResetAdmin
	defer func() {
		devResetLoadConfig = originalLoadConfig
		devResetOpenDB = originalOpenDB
		devResetCloseDB = originalCloseDB
		devResetNewAuthRepository = originalNewAuthRepository
		devResetResolveRBACBootstrap = originalResolveRBACBootstrap
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
	devResetNewAuthRepository = func(_ *sql.DB) (user.AuthRepositoryForReset, error) {
		steps = append(steps, "new-auth-repository")
		return userAuthRepositoryForResetStub{}, nil
	}
	devResetResolveRBACBootstrap = func(*database.Resources) (pluginapi.RBACBootstrapService, error) {
		steps = append(steps, "new-rbac-bootstrap")
		return rbacBootstrapServiceStub{}, nil
	}
	devResetAdmin = func(_ context.Context, _ user.AuthRepositoryForReset, _ pluginapi.RBACBootstrapService) error {
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
		"new-rbac-bootstrap",
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
	originalResolveRBACBootstrap := devResetResolveRBACBootstrap
	originalResetAdmin := devResetAdmin
	defer func() {
		devResetLoadConfig = originalLoadConfig
		devResetOpenDB = originalOpenDB
		devResetCloseDB = originalCloseDB
		devResetNewAuthRepository = originalNewAuthRepository
		devResetResolveRBACBootstrap = originalResolveRBACBootstrap
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
	devResetNewAuthRepository = func(_ *sql.DB) (user.AuthRepositoryForReset, error) {
		return userAuthRepositoryForResetStub{}, nil
	}
	devResetResolveRBACBootstrap = func(*database.Resources) (pluginapi.RBACBootstrapService, error) {
		return rbacBootstrapServiceStub{}, nil
	}
	devResetAdmin = func(context.Context, user.AuthRepositoryForReset, pluginapi.RBACBootstrapService) error {
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

type rbacBootstrapServiceStub struct{}

func (rbacBootstrapServiceStub) EnsureDefaultAdminAccess(context.Context, uint64, []pluginapi.PermissionSeed) error {
	return nil
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
