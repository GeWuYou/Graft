package network

import (
	"errors"
	"fmt"

	"graft/server/internal/container"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	networkcontract "graft/server/modules/network/contract"
)

const (
	moduleID         = "platform-network"
	networkMenuOrder = 102
)

// Module 拥有平台主动 HTTP(S) 访问策略、client factory 和固定诊断目标注册表。
type Module struct {
	service *Service
}

// NewModule 创建 platform-network 模块实例。
func NewModule() *Module { return &Module{} }

// Register 注册出站策略定义和稳定消费能力；策略读取始终经由 System Config 快照，不读取进程环境变量。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || ctx == nil || ctx.ConfigRegistry == nil || ctx.Services == nil {
		return errors.New("platform-network module context is unavailable")
	}
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	if err := registerPermissions(ctx.PermissionRegistry); err != nil {
		return err
	}
	if err := registerMenu(ctx.MenuRegistry); err != nil {
		return err
	}
	if err := registerOutboundConfig(ctx.ConfigRegistry); err != nil {
		return fmt.Errorf("register outbound network config: %w", err)
	}
	configs, err := module.ResolveService[moduleapi.ModuleConfigManager](ctx.Services, (*moduleapi.ModuleConfigManager)(nil))
	if err != nil {
		return fmt.Errorf("resolve module config manager: %w", err)
	}
	provider, err := NewPolicyProvider(configs)
	if err != nil {
		return err
	}
	factory, err := NewHTTPClientFactory(provider)
	if err != nil {
		return err
	}
	diagnostics := NewDiagnosticRegistry()
	m.service = NewService(configs, diagnostics)
	if err := ctx.Services.RegisterSingleton((*moduleapi.OutboundNetworkProvider)(nil), func(container.Resolver) (any, error) { return provider, nil }); err != nil {
		return fmt.Errorf("register outbound network provider: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.OutboundHTTPClientFactory)(nil), func(container.Resolver) (any, error) { return factory, nil }); err != nil {
		return fmt.Errorf("register outbound HTTP client factory: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.OutboundDiagnosticRegistry)(nil), func(container.Resolver) (any, error) { return diagnostics, nil }); err != nil {
		return fmt.Errorf("register outbound diagnostic registry: %w", err)
	}
	return registerNetworkRoutes(ctx, m.service)
}

// Boot 不在启动时访问外网，避免平台启动可用性依赖远程目标。
func (m *Module) Boot(_ *module.Context) error { return nil }

// Shutdown 不持有额外资源，网络连接由每个 HTTP client 自己管理。
func (m *Module) Shutdown(_ *module.Context) error { return nil }

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}
	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range []i18n.MessageKey{"menu.platform.network", "network.outbound.title", "network.outbound.description", "network.outbound.authentication.description", "network.diagnosticTargets.platformUpdate"} {
			if len(localizer.RegisteredMessageResources(locale, key)) == 0 {
				return fmt.Errorf("platform-network locale resource missing %s for %s", key, locale)
			}
		}
	}
	return nil
}

func registerPermissions(registry *permission.Registry) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}
	for _, item := range []struct {
		code                            networkcontract.PermissionCode
		action, riskLevel, riskCategory string
	}{
		{networkcontract.NetworkReadPermission, "read", permission.RiskLevelLow, permission.RiskCategoryRead},
		{networkcontract.NetworkWritePermission, "write", permission.RiskLevelHigh, permission.RiskCategorySecurity},
		{networkcontract.NetworkDiagnosePermission, "diagnose", permission.RiskLevelMedium, permission.RiskCategoryWrite},
	} {
		registry.Register(permission.Item{Code: item.code.String(), DisplayKey: "rbac.permissionCatalog." + string(item.code) + ".display", DescriptionKey: "rbac.permissionCatalog." + string(item.code) + ".description", Module: moduleID, Resource: "platform-network", Action: item.action, RiskLevel: item.riskLevel, RiskCategory: item.riskCategory})
	}
	return nil
}

func registerMenu(registry *menu.Registry) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}
	registry.Register(menu.Item{Code: "platform-network", ParentCode: "domain.platform", Kind: menu.NodeKindEntry, TitleKey: "menu.platform.network", Path: networkcontract.NetworkMenuPath, Icon: "network", Order: networkMenuOrder, Permission: networkcontract.NetworkReadPermission.String(), Module: moduleID})
	return nil
}
