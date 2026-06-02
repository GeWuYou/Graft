package audit

import (
	"errors"
	"fmt"

	auditcore "graft/server/internal/audit"
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

func registerAuditPermissions(registry *permission.Registry, pluginName string) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:        auditcontract.AuditReadPermission.String(),
		Name:        "Read Audit Logs",
		Description: "Allows reading audit-log records and filters.",
		Category:    "api",
		Plugin:      pluginName,
	})
}

func registerAuditMenu(registry *menu.Registry, pluginName string) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       "audit.root",
		Title:      "安全审计",
		TitleKey:   auditcontract.AuditRootMenuTitle.String(),
		Path:       auditcontract.AuditMenuPath,
		Icon:       "secured",
		Order:      auditMenuOrderRoot,
		Permission: "",
		Plugin:     pluginName,
	})

	registry.Register(menu.Item{
		Code:       "audit.overview",
		Title:      "概览",
		TitleKey:   auditcontract.AuditOverviewMenuTitle.String(),
		Path:       auditcontract.AuditOverviewMenuPath,
		Icon:       "dashboard",
		Order:      auditMenuOrderOverview,
		Permission: auditcontract.AuditReadPermission.String(),
		Plugin:     pluginName,
	})

	registry.Register(menu.Item{
		Code:       "audit.logs",
		Title:      "审计日志",
		TitleKey:   auditcontract.AuditLogMenuTitle.String(),
		Path:       auditcontract.AuditLogsMenuPath,
		Icon:       "history",
		Order:      auditMenuOrderLogs,
		Permission: auditcontract.AuditReadPermission.String(),
		Plugin:     pluginName,
	})
}

func registerAuditMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, registration := range []i18n.Registration{
		{
			Namespace: "audit",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(auditcontract.AuditRootMenuTitle.String()), Text: "安全审计"},
				{Key: i18n.MessageKey(auditcontract.AuditOverviewMenuTitle.String()), Text: "概览"},
				{Key: i18n.MessageKey(auditcontract.AuditLogMenuTitle.String()), Text: "审计日志"},
			},
		},
		{
			Namespace: "audit",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(auditcontract.AuditRootMenuTitle.String()), Text: "Security Audit"},
				{Key: i18n.MessageKey(auditcontract.AuditOverviewMenuTitle.String()), Text: "Overview"},
				{Key: i18n.MessageKey(auditcontract.AuditLogMenuTitle.String()), Text: "Audit Logs"},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register audit plugin messages: %w", err)
		}
	}

	return nil
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
		read: httpx.RequirePermission(ctx.I18n, authService, authorizer, auditcontract.AuditReadPermission.String(), publisher),
	}, nil
}

func registerAuditService(ctx *module.Context, reader *auditcore.Service) error {
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
