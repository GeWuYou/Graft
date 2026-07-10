package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/database"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
	"graft/server/modules/auth"
	authstore "graft/server/modules/auth/store"
	"graft/server/modules/rbac"
	"graft/server/modules/user"
)

var (
	devResetLoadConfig        = config.Load
	devResetOpenDB            = database.Open
	devResetCloseDB           = database.Close
	devResetNewUserIdentity   = user.NewIdentityProviderForDevelopmentReset
	devResetNewAuthRepository = auth.NewRepositoryForDevelopmentReset
	devResetNewLocalizer      = func(cfg config.I18nConfig) (*i18n.Service, error) { return i18n.New(cfg) }
	devResetAdmin             = func(ctx context.Context, authRepo authstore.AuthRepository, identity moduleapi.UserIdentityProvider, localizer *i18n.Service, rbac moduleapi.RBACBootstrapService) error {
		return auth.ResetDefaultAdminForDevelopment(ctx, authRepo, identity, localizer, rbac, user.DefaultAdminPermissionItems())
	}
	devResetResolveRBACBootstrap = func(resources *database.Resources) (moduleapi.RBACBootstrapService, error) {
		repo, err := rbac.NewRepositoryForReset(resources.SQL)
		if err != nil {
			return nil, err
		}
		return rbac.NewBootstrapServiceForReset(repo), nil
	}
)

// newDevResetAdminCommand 创建仅限开发环境使用的命令，用于将默认管理员恢复为首次登录状态。
// 返回配置完成的 reset-admin 命令。
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

// runDevResetAdmin 在本地或测试环境中初始化重置所需服务，将默认管理员恢复为开发环境首次登录状态，并输出重置结果。
// 数据库关闭失败时，会将该错误与执行过程中的错误合并返回。
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

	identity, err := devResetNewUserIdentity(resources.SQL)
	if err != nil {
		return fmt.Errorf("create user identity provider: %w", err)
	}
	authRepo, err := devResetNewAuthRepository(resources.SQL, identity)
	if err != nil {
		return fmt.Errorf("create user auth repository: %w", err)
	}
	localizer, err := devResetNewLocalizer(cfg.I18n)
	if err != nil {
		return fmt.Errorf("create i18n service: %w", err)
	}
	rbacBootstrap, err := devResetResolveRBACBootstrap(resources)
	if err != nil {
		return fmt.Errorf("create rbac bootstrap service: %w", err)
	}

	if err := devResetAdmin(cmd.Context(), authRepo, identity, localizer, rbacBootstrap); err != nil {
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
