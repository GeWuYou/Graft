// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/database"
	"graft/server/internal/moduleapi"
	"graft/server/modules/rbac"
	"graft/server/modules/user"
)

var (
	devResetLoadConfig        = config.Load
	devResetOpenDB            = database.Open
	devResetCloseDB           = database.Close
	devResetNewAuthRepository = user.NewAuthRepositoryForReset
	devResetAdmin             = func(ctx context.Context, authRepo user.AuthRepositoryForReset, rbac moduleapi.RBACBootstrapService) error {
		return user.ResetDefaultAdminForDevelopment(
			ctx,
			authRepo,
			rbac,
		)
	}
	devResetResolveRBACBootstrap = func(resources *database.Resources) (moduleapi.RBACBootstrapService, error) {
		repo, err := rbac.NewRepositoryForReset(resources.SQL)
		if err != nil {
			return nil, err
		}
		return rbac.NewBootstrapServiceForReset(repo), nil
	}
)

func newDevResetAdminCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-admin",
		Short: "Reset the default admin back to the development-first-login state",
		Long: "graft dev reset-admin is a dev-only helper for local verification. " +
			"It ensures the default graft admin exists, resets its password to graft-admin, and marks must_change_password=true.",
		Example:      "  graft dev reset-admin",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDevResetAdmin(cmd)
		},
	}
}

func runDevResetAdmin(cmd *cobra.Command) (err error) {
	cfg, err := devResetLoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if !isDevelopmentAppEnv(cfg.App.Env) {
		return fmt.Errorf("dev reset-admin is only available in local/test environments, got %q", cfg.App.Env)
	}

	resources, err := devResetOpenDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if closeErr := devResetCloseDB(resources); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close database: %w", closeErr))
		}
	}()

	authRepo, err := devResetNewAuthRepository(resources.SQL)
	if err != nil {
		return fmt.Errorf("create user auth repository: %w", err)
	}
	rbacBootstrap, err := devResetResolveRBACBootstrap(resources)
	if err != nil {
		return fmt.Errorf("create rbac bootstrap service: %w", err)
	}

	if err := devResetAdmin(cmd.Context(), authRepo, rbacBootstrap); err != nil {
		return fmt.Errorf("reset default admin: %w", err)
	}

	if _, err := cmd.OutOrStdout().Write([]byte("default admin reset: username=graft password=graft-admin must_change_password=true\n")); err != nil {
		return fmt.Errorf("write reset-admin result: %w", err)
	}

	return err
}

func isDevelopmentAppEnv(env string) bool {
	switch strings.TrimSpace(env) {
	case "local", "test":
		return true
	default:
		return false
	}
}
