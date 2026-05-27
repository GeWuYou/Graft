package audit

import (
	"errors"
	"fmt"

	"graft/server/internal/container"
	auditcore "graft/server/internal/audit"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	auditcontract "graft/server/plugins/audit/contract"
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
		Code:       "audit.logs",
		Title:      "审计日志",
		TitleKey:   auditcontract.AuditLogMenuTitle.String(),
		Path:       auditcontract.AuditMenuPath,
		Icon:       "history",
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
				{Key: i18n.MessageKey(auditcontract.AuditLogMenuTitle.String()), Text: "审计日志"},
			},
		},
		{
			Namespace: "audit",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
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

func (p *Plugin) resolveRouteGuard(ctx *plugin.Context) (auditGuard, error) {
	if ctx == nil || ctx.Services == nil {
		return auditGuard{}, errors.New("plugin context services are unavailable")
	}

	resolvedAuthService, err := ctx.Services.Resolve((*pluginapi.AuthService)(nil))
	if err != nil {
		return auditGuard{}, fmt.Errorf("resolve auth service: %w", err)
	}
	authService, ok := resolvedAuthService.(pluginapi.AuthService)
	if !ok {
		return auditGuard{}, fmt.Errorf("resolve auth service: unexpected type %T", resolvedAuthService)
	}

	resolvedAuthorizer, err := ctx.Services.Resolve((*pluginapi.Authorizer)(nil))
	if err != nil {
		return auditGuard{}, fmt.Errorf("resolve route authorizer: %w", err)
	}
	authorizer, ok := resolvedAuthorizer.(pluginapi.Authorizer)
	if !ok {
		return auditGuard{}, fmt.Errorf("resolve route authorizer: unexpected type %T", resolvedAuthorizer)
	}

	return auditGuard{
		read: httpx.RequirePermission(ctx.I18n, authService, authorizer, auditcontract.AuditReadPermission.String()),
	}, nil
}

func registerAuditService(ctx *plugin.Context, reader *auditcore.Service) error {
	if ctx == nil || ctx.Services == nil {
		return errors.New("plugin context services are unavailable")
	}
	if reader == nil {
		return errors.New("audit service is unavailable")
	}

	return ctx.Services.RegisterSingleton((*auditReader)(nil), func(_ container.Resolver) (any, error) {
		return reader, nil
	})
}
