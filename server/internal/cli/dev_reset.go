package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/database"
	"graft/server/internal/store"
	"graft/server/internal/store/entstore"
	rbacplugin "graft/server/plugins/rbac"
	rbacstoreadapter "graft/server/plugins/rbac/storeadapter"
	"graft/server/plugins/user"
	"graft/server/plugins/user/storeadapter"
)

var (
	devResetLoadConfig = config.Load
	devResetOpenDB     = database.Open
	devResetCloseDB    = database.Close
	devResetNewFactory = func(resources *database.Resources) (store.Factory, error) {
		return entstore.NewFactory(resources.Client)
	}
	devResetAdmin = func(ctx context.Context, authRepo store.AuthRepository, rbacRepo store.RBACRepository) error {
		return user.ResetDefaultAdminForDevelopment(
			ctx,
			storeadapter.NewAuthRepositoryAdapter(authRepo),
			rbacplugin.NewBootstrapService(rbacstoreadapter.NewInternalRepositoryAdapter(rbacRepo)),
		)
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

	factory, err := devResetNewFactory(resources)
	if err != nil {
		return fmt.Errorf("create ent store factory: %w", err)
	}

	if err := devResetAdmin(cmd.Context(), factory.Auth(), factory.RBAC()); err != nil {
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
