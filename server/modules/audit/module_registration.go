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

const (
	auditMenuOrderRoot     = 200
	auditMenuOrderOverview = 201
	auditMenuOrderLogs     = 202
)

func registerAuditPermissions(registry *permission.Registry, moduleName string) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:           auditcontract.AuditReadPermission.String(),
		DisplayKey:     "rbac.permissionCatalog.auditRead.display",
		DescriptionKey: "rbac.permissionCatalog.auditRead.description",
		Category:       "api",
		Module:         moduleName,
	})
	registry.Register(permission.Item{
		Code:           auditcontract.AuditManagePermission.String(),
		DisplayKey:     "rbac.permissionCatalog.auditManage.display",
		DescriptionKey: "rbac.permissionCatalog.auditManage.description",
		Category:       "api",
		Module:         moduleName,
	})
}

func registerAuditMenu(registry *menu.Registry, moduleName string) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       "audit.root",
		TitleKey:   auditcontract.AuditRootMenuTitle.String(),
		Path:       auditcontract.AuditMenuPath,
		Icon:       "secured",
		Order:      auditMenuOrderRoot,
		Permission: "",
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "audit.overview",
		TitleKey:   auditcontract.AuditOverviewMenuTitle.String(),
		Path:       auditcontract.AuditOverviewMenuPath,
		Icon:       "dashboard",
		Order:      auditMenuOrderOverview,
		Permission: auditcontract.AuditReadPermission.String(),
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "audit.logs",
		TitleKey:   auditcontract.AuditLogMenuTitle.String(),
		Path:       auditcontract.AuditLogsMenuPath,
		Icon:       "history",
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

func auditMessageKeys() []string {
	return []string{
		auditcontract.AuditRootMenuTitle.String(),
		auditcontract.AuditOverviewMenuTitle.String(),
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

func registerAuditService(ctx *module.Context, reader *Service) error {
	if ctx == nil || ctx.Services == nil {
		return errors.New("module context services are unavailable")
	}
	if reader == nil {
		return errors.New("audit service is unavailable")
	}

	return ctx.Services.RegisterSingleton((*auditReader)(nil), func(_ container.Resolver) (any, error) {
		return reader, nil
	})
}
