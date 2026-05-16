package rbac

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/container"
	"graft/server/internal/httpx"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
	rbaccontract "graft/server/plugins/rbac/contract"
)

// Plugin 是 MVP 阶段最小可用的 RBAC 插件。
//
// 当前实现同时承载两类稳定边界：
//   - 暴露 `pluginapi.Authorizer`，把权限判断收敛为统一后端安全边界
//   - 提供角色/权限只读管理路由，供 `web` 消费真实 RBAC 快照
type Plugin struct{}

// NewPlugin 创建最小 RBAC 插件。
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name 返回插件稳定标识。
func (p *Plugin) Name() string {
	return "rbac"
}

// Version 返回当前插件版本。
func (p *Plugin) Version() string {
	return "0.1.0"
}

// DependsOn 声明当前最小授权插件依赖用户插件已完成认证主体解析。
func (p *Plugin) DependsOn() []string {
	return []string{"user"}
}

// Register 注册跨插件可复用的授权服务。
//
// Register 阶段只做稳定能力暴露与管理只读路由装配，不执行任何后台行为或耗时初始化。
func (p *Plugin) Register(ctx *plugin.Context) error {
	registerRBACPermissions(ctx.PermissionRegistry, p.Name())
	registerRBACMenu(ctx.MenuRegistry, p.Name())
	repository := ctx.Stores.RBAC()
	readService := managementReader{rbac: repository}
	writeService := managementWriter{rbac: repository}

	if err := ctx.Services.RegisterSingleton((*pluginapi.Authorizer)(nil), func(_ container.Resolver) (any, error) {
		return authorizer{rbac: repository}, nil
	}); err != nil {
		return err
	}

	resolved, err := ctx.Services.Resolve((*pluginapi.AuthService)(nil))
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}

	authService, ok := resolved.(pluginapi.AuthService)
	if !ok {
		return fmt.Errorf("resolve auth service: unexpected type %T", resolved)
	}

	routeAuthorizer := authorizer{rbac: repository}
	registerManagementRoutes(
		ctx,
		p.Name(),
		readService,
		writeService,
		managementGuards{
			roleRead:             httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.RoleReadPermission.String()),
			permissionRead:       httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.PermissionReadPermission.String()),
			roleCreate:           httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.RoleCreatePermission.String()),
			roleUpdate:           httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.RoleUpdatePermission.String()),
			rolePermissionAssign: httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.RolePermissionAssignPermission.String()),
			userRoleAssign:       httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.UserRoleAssignPermission.String()),
		},
	)

	return nil
}

// Boot 当前没有额外运行时行为需要启动。
func (p *Plugin) Boot(_ *plugin.Context) error {
	return nil
}

// Shutdown 当前没有额外资源需要释放。
func (p *Plugin) Shutdown(_ *plugin.Context) error {
	return nil
}

type authorizer struct {
	rbac store.RBACRepository
}

// Authorize 基于稳定 RBAC 仓储判断请求主体是否拥有指定权限。
func (a authorizer) Authorize(ctx context.Context, request pluginapi.RequestAuthContext, permission string) error {
	if request.User == nil || request.User.ID == 0 {
		return pluginapi.ErrUnauthenticated
	}
	if strings.TrimSpace(permission) == "" {
		return nil
	}
	if a.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}

	permissions, err := a.rbac.ListPermissionsByUserID(ctx, request.User.ID)
	if err != nil {
		return err
	}
	for _, granted := range permissions {
		if granted.Code == permission {
			return nil
		}
	}

	return pluginapi.ErrPermissionDenied
}

var _ pluginapi.Authorizer = authorizer{}
