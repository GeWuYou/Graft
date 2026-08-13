package registry

import (
	"errors"
	"fmt"

	containerdi "graft/server/internal/container"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	registrycontract "graft/server/modules/registry/contract"
)

// Module 拥有外部 Registry Connection 与 Artifact Repository 权威，不声明镜像仓库服务、路由或后台运行时。
type Module struct{ service *Service }

// NewModule 创建 Registry 基础设施模块。
func NewModule(service *Service) *Module { return &Module{service: service} }

// Register 注册 Registry 权限、导航、管理 HTTP API 与面向 Build 的窄解析能力。
//
//nolint:cyclop,mnd // 声明式权限和三个稳定 capability 必须在模块注册点集中可见。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil || ctx == nil || ctx.Services == nil || ctx.PermissionRegistry == nil || ctx.MenuRegistry == nil {
		return errors.New("registry module service is unavailable")
	}
	users, err := module.ResolveService[moduleapi.UserCandidateReader](ctx.Services, (*moduleapi.UserCandidateReader)(nil))
	if err != nil {
		return fmt.Errorf("resolve user candidate reader: %w", err)
	}
	m.service.bindUserCandidateReader(users)
	for _, item := range []permission.Item{
		{Code: registrycontract.ReadPermission, DisplayKey: "rbac.permissionCatalog.registryRead.display", DescriptionKey: "rbac.permissionCatalog.registryRead.description", Module: moduleID, Resource: "registry", Action: "read", RiskLevel: permission.RiskLevelLow, RiskCategory: permission.RiskCategoryRead},
		{Code: registrycontract.CreatePermission, DisplayKey: "rbac.permissionCatalog.registryCreate.display", DescriptionKey: "rbac.permissionCatalog.registryCreate.description", Module: moduleID, Resource: "registry", Action: "create", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategoryWrite},
		{Code: registrycontract.UpdatePermission, DisplayKey: "rbac.permissionCatalog.registryUpdate.display", DescriptionKey: "rbac.permissionCatalog.registryUpdate.description", Module: moduleID, Resource: "registry", Action: "update", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategoryWrite},
		{Code: registrycontract.DeletePermission, DisplayKey: "rbac.permissionCatalog.registryDelete.display", DescriptionKey: "rbac.permissionCatalog.registryDelete.description", Module: moduleID, Resource: "registry", Action: "delete", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategoryDestructive},
		{Code: registrycontract.VerifyPermission, DisplayKey: "rbac.permissionCatalog.registryVerify.display", DescriptionKey: "rbac.permissionCatalog.registryVerify.description", Module: moduleID, Resource: "registry", Action: "verify", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategorySecurity},
		{Code: registrycontract.AssignmentManagePermission, DisplayKey: "rbac.permissionCatalog.registryAssignmentManage.display", DescriptionKey: "rbac.permissionCatalog.registryAssignmentManage.description", Module: moduleID, Resource: "registry", Action: "assignment.manage", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategorySecurity},
	} {
		ctx.PermissionRegistry.Register(item)
	}
	ctx.MenuRegistry.Register(menu.Item{Code: "registry.list", ParentCode: "domain.infrastructure", Kind: menu.NodeKindEntry, TitleKey: registrycontract.MenuTitle, SectionKey: menu.SharedResourcesSectionKey, SectionTitleKey: menu.SharedResourcesSectionTitleKey, Path: registrycontract.MenuPath, Icon: registrycontract.MenuIcon, Order: registrycontract.MenuOrder, Permission: registrycontract.ReadPermission, Module: moduleID})
	if err := ctx.Services.RegisterSingleton((*moduleapi.RegistryDestinationResolver)(nil), func(containerdi.Resolver) (any, error) {
		return m.service, nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RegistryPublicationResolver)(nil), func(containerdi.Resolver) (any, error) {
		return m.service, nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RegistryArtifactCopyResolver)(nil), func(containerdi.Resolver) (any, error) {
		return m.service, nil
	}); err != nil {
		return err
	}
	return registerRegistryRoutes(ctx, m.service)
}

// Boot 不启动 Registry 自有后台进程。
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown 不需要释放 Registry 自有长生命周期资源。
func (*Module) Shutdown(*module.Context) error { return nil }
