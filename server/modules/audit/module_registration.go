package audit

import (
	"errors"
	"fmt"

	"graft/server/internal/container"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	auditcontract "graft/server/modules/audit/contract"
)

const auditMenuOrderLogs = 2

// registerAuditPermissions 注册审计模块的读取和管理权限条目。
func registerAuditPermissions(registry *permission.Registry, moduleName string) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:           auditcontract.AuditReadPermission.String(),
		DisplayKey:     "rbac.permissionCatalog.auditRead.display",
		DescriptionKey: "rbac.permissionCatalog.auditRead.description",
		Module:         moduleName,
		Resource:       "audit",
		Action:         "read",
		RiskLevel:      permission.RiskLevelLow,
		RiskCategory:   permission.RiskCategoryRead,
	})
	registry.Register(permission.Item{
		Code:           auditcontract.AuditManagePermission.String(),
		DisplayKey:     "rbac.permissionCatalog.auditManage.display",
		DescriptionKey: "rbac.permissionCatalog.auditManage.description",
		Module:         moduleName,
		Resource:       "audit",
		Action:         "manage",
		RiskLevel:      permission.RiskLevelHigh,
		RiskCategory:   permission.RiskCategorySecurity,
	})
}

// registerAuditMenu 注册审计日志菜单项，并为其配置审计读取权限。
func registerAuditMenu(registry *menu.Registry, moduleName string) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       "audit.logs",
		ParentCode: "domain.security",
		Kind:       menu.NodeKindEntry,
		TitleKey:   auditcontract.AuditLogMenuTitle.String(),
		Path:       auditcontract.AuditLogsMenuPath,
		Icon:       "audit-trail",
		Order:      auditMenuOrderLogs,
		Permission: auditcontract.AuditReadPermission.String(),
		Module:     moduleName,
	})
}

func registerAuditMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range auditMessageKeys() {
			matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
			if len(matches) == 0 {
				return fmt.Errorf("register audit module messages: locale resource %s missing key %s", locale, key)
			}
		}
	}

	return nil
}

// auditMessageKeys 返回审计模块所需的本地化消息键。
func auditMessageKeys() []string {
	return []string{
		auditcontract.AuditRootMenuTitle.String(),
		auditcontract.AuditLogMenuTitle.String(),
		auditcontract.AuditTargetLabelUser.String(),
		auditcontract.AuditTargetLabelRole.String(),
		auditcontract.AuditTargetLabelPermission.String(),
		auditcontract.AuditTargetLabelAudit.String(),
		auditcontract.AuditTargetLabelServerStatus.String(),
		auditcontract.AuditTargetLabelAuth.String(),
	}
}

func (p *Module) resolveRouteGuard(ctx *module.Context) (auditGuard, error) {
	if ctx == nil || ctx.Services == nil {
		return auditGuard{}, errors.New("module context services are unavailable")
	}

	resolvedAuthService, err := ctx.Services.Resolve((*moduleapi.AuthService)(nil))
	if err != nil {
		return auditGuard{}, fmt.Errorf("resolve auth service: %w", err)
	}
	authService, ok := resolvedAuthService.(moduleapi.AuthService)
	if !ok {
		return auditGuard{}, fmt.Errorf("resolve auth service: unexpected type %T", resolvedAuthService)
	}

	resolvedAuthorizer, err := ctx.Services.Resolve((*moduleapi.Authorizer)(nil))
	if err != nil {
		return auditGuard{}, fmt.Errorf("resolve route authorizer: %w", err)
	}
	authorizer, ok := resolvedAuthorizer.(moduleapi.Authorizer)
	if !ok {
		return auditGuard{}, fmt.Errorf("resolve route authorizer: unexpected type %T", resolvedAuthorizer)
	}

	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	return auditGuard{
		read:   httpx.RequirePermission(ctx.I18n, authService, authorizer, auditcontract.AuditReadPermission.String(), publisher),
		manage: httpx.RequirePermission(ctx.I18n, authService, authorizer, auditcontract.AuditManagePermission.String(), publisher),
	}, nil
}

// registerAuditService 将审计服务及其安全读取器注册到模块服务容器。
func registerAuditService(ctx *module.Context, reader *Service) error {
	if ctx == nil || ctx.Services == nil {
		return errors.New("module context services are unavailable")
	}
	if reader == nil {
		return errors.New("audit service is unavailable")
	}

	if err := ctx.Services.RegisterSingleton((*auditReader)(nil), func(_ container.Resolver) (any, error) {
		return reader, nil
	}); err != nil {
		return err
	}
	return ctx.Services.RegisterSingleton((*moduleapi.AuditSecurityReader)(nil), func(_ container.Resolver) (any, error) {
		return auditSecurityReader{reader: reader}, nil
	})
}
