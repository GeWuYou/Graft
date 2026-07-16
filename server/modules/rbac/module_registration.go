package rbac

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/container"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	rbaccontract "graft/server/modules/rbac/contract"
	rbacstore "graft/server/modules/rbac/store"
)

// Module 是 MVP 阶段最小可用的 RBAC 模块。
//
// 当前实现同时承载两类稳定边界：
//   - 暴露 `moduleapi.Authorizer`，把权限判断收敛为统一后端安全边界
//   - 提供角色/权限只读管理路由，供 `web` 消费真实 RBAC 快照
type Module struct {
	repository rbacstore.Repository
}

// NewModule 创建最小 RBAC 模块。
func NewModule(repository rbacstore.Repository) *Module {
	return &Module{repository: repository}
}

// Register 注册跨模块可复用的授权服务。
//
// Register 阶段只做稳定能力暴露与管理只读路由装配，不执行任何后台行为或耗时初始化。
func (p *Module) Register(ctx *module.Context) error {
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	registerRBACPermissions(ctx.PermissionRegistry, moduleID)
	registerRBACMenu(ctx.MenuRegistry, moduleID)
	repository := p.repository
	if repository == nil {
		return errors.New("rbac repository is unavailable")
	}
	userService, securityReader, err := resolveUserServices(ctx)
	if err != nil {
		return err
	}
	if err := registerModuleServices(ctx, repository, securityReader); err != nil {
		return err
	}

	readService := managementReader{
		users: userService,
		rbac:  repository,
	}
	if err := registerDashboardWidgets(ctx, readService); err != nil {
		return err
	}
	writeService := managementWriter{
		users:    userService,
		rbac:     repository,
		auditBus: ctx.EventBus,
		logger:   ctx.Logger,
	}

	resolved, err := ctx.Services.Resolve((*moduleapi.AuthService)(nil))
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}

	authService, ok := resolved.(moduleapi.AuthService)
	if !ok {
		return fmt.Errorf("resolve auth service: unexpected type %T", resolved)
	}

	routeAuthorizer := authorizer{rbac: repository}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	registerManagementRoutes(
		ctx,
		moduleID,
		readService,
		writeService,
		managementGuards{
			roleRead:             httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.RoleReadPermission.String(), publisher),
			permissionRead:       httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.PermissionReadPermission.String(), publisher),
			roleCreate:           httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.RoleCreatePermission.String(), publisher),
			roleUpdate:           httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.RoleUpdatePermission.String(), publisher),
			roleStatus:           httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.RoleStatusUpdatePermission.String(), publisher),
			roleDelete:           httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.RoleDeletePermission.String(), publisher),
			rolePermissionAssign: httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.RolePermissionAssignPermission.String(), publisher),
			userRoleRead:         httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.UserRoleReadPermission.String(), publisher),
			userRoleAssign:       httpx.RequirePermission(ctx.I18n, authService, routeAuthorizer, rbaccontract.UserRoleAssignPermission.String(), publisher),
		},
	)

	return nil
}

// resolveUserServices 解析 RBAC 所需的用户服务和可选安全摘要读取器。
// 安全摘要读取器未注册时保留 nil，使 RBAC 的基础授权能力不依赖仪表盘统计；解析失败或类型不符时返回错误。
func resolveUserServices(ctx *module.Context) (moduleapi.UserService, moduleapi.UserSecurityReader, error) {
	resolvedUserService, err := ctx.Services.Resolve((*moduleapi.UserService)(nil))
	if err != nil {
		return nil, nil, fmt.Errorf("resolve user service: %w", err)
	}
	userService, ok := resolvedUserService.(moduleapi.UserService)
	if !ok {
		return nil, nil, fmt.Errorf("resolve user service: unexpected type %T", resolvedUserService)
	}

	resolvedSecurityReader, err := ctx.Services.Resolve((*moduleapi.UserSecurityReader)(nil))
	if errors.Is(err, container.ErrServiceNotRegistered) {
		return userService, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("resolve user security reader: %w", err)
	}
	securityReader, ok := resolvedSecurityReader.(moduleapi.UserSecurityReader)
	if !ok {
		return nil, nil, fmt.Errorf("resolve user security reader: unexpected type %T", resolvedSecurityReader)
	}
	return userService, securityReader, nil
}

// registerModuleServices 注册 RBAC 访问、bootstrap 与授权服务。
// 只有用户安全摘要读取器可用时才注册安全态势服务，避免把可选观测能力误当成授权运行时依赖。
func registerModuleServices(ctx *module.Context, repository rbacstore.Repository, users moduleapi.UserSecurityReader) error {
	if err := ctx.Services.RegisterSingleton((*moduleapi.RBACAccessService)(nil), func(_ container.Resolver) (any, error) {
		return accessService{rbac: repository}, nil
	}); err != nil {
		return err
	}
	if users != nil {
		if err := ctx.Services.RegisterSingleton((*moduleapi.RBACSecurityPostureService)(nil), func(_ container.Resolver) (any, error) {
			return accessService{rbac: repository, users: users}, nil
		}); err != nil {
			return err
		}
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RBACBootstrapService)(nil), func(_ container.Resolver) (any, error) {
		return bootstrapService{rbac: repository}, nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(_ container.Resolver) (any, error) {
		return authorizer{rbac: repository}, nil
	}); err != nil {
		return err
	}
	return nil
}

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range rbacMessageKeys() {
			matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key.String()))
			if len(matches) == 0 {
				return fmt.Errorf("register rbac module messages: locale resource %s missing key %s", locale, key)
			}
		}
	}

	return nil
}

// rbacMessageKeys 返回 RBAC 模块注册所需的消息键列表。
func rbacMessageKeys() []rbaccontract.MessageKey {
	return []rbaccontract.MessageKey{
		rbaccontract.RoleListMenuTitle,
		rbaccontract.PermissionListMenuTitle,
		rbaccontract.AuditRolePermissionsAdded,
		rbaccontract.AuditRolePermissionsRemoved,
	}
}

// Boot 当前没有额外运行时行为需要启动。
func (p *Module) Boot(_ *module.Context) error {
	return nil
}

// Shutdown 当前没有额外资源需要释放。
func (p *Module) Shutdown(_ *module.Context) error {
	return nil
}

type authorizer struct {
	rbac rbacstore.Repository
}

// Authorize 基于稳定 RBAC 仓储判断请求主体是否拥有指定权限。
func (a authorizer) Authorize(ctx context.Context, request moduleapi.RequestAuthContext, permission string) error {
	if request.User == nil || request.User.ID == 0 {
		return moduleapi.ErrUnauthenticated
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

	return moduleapi.ErrPermissionDenied
}

var _ moduleapi.Authorizer = authorizer{}
