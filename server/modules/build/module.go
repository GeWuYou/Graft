package build

import (
	"errors"

	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/permission"
	buildcontract "graft/server/modules/build/contract"
)

const moduleID = "build"

// Module 声明 Build domain 的生命周期边界；具体 executor 和 API 在后续阶段接入。
type Module struct{}

// NewModule 创建无常驻资源的 Build 模块。
func NewModule() *Module { return &Module{} }

// Register 注册 Build 权限和导航入口，不启动构建行为。
func (*Module) Register(ctx *module.Context) error {
	if ctx == nil || ctx.PermissionRegistry == nil || ctx.MenuRegistry == nil {
		return errors.New("build module registries are unavailable")
	}
	items := []permission.Item{
		{Code: buildcontract.BuildReadPermission, DisplayKey: "rbac.permissionCatalog.buildRead.display", DescriptionKey: "rbac.permissionCatalog.buildRead.description", Module: moduleID},
		{Code: buildcontract.BuildCreatePermission, DisplayKey: "rbac.permissionCatalog.buildCreate.display", DescriptionKey: "rbac.permissionCatalog.buildCreate.description", Module: moduleID},
		{Code: buildcontract.BuildCancelPermission, DisplayKey: "rbac.permissionCatalog.buildCancel.display", DescriptionKey: "rbac.permissionCatalog.buildCancel.description", Module: moduleID},
		{Code: buildcontract.BuildRetryPermission, DisplayKey: "rbac.permissionCatalog.buildRetry.display", DescriptionKey: "rbac.permissionCatalog.buildRetry.description", Module: moduleID},
	}
	for _, item := range items {
		ctx.PermissionRegistry.Register(item)
	}
	ctx.MenuRegistry.Register(menu.Item{Code: "build.jobs", ParentCode: "domain.build", Kind: menu.NodeKindEntry, TitleKey: "menu.build.jobs.title", Path: "/build/jobs", Icon: "build", Order: 1, Permission: buildcontract.BuildReadPermission, Module: moduleID})
	return nil
}

// Boot 当前无常驻构建资源。
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown 当前无模块自有资源。
func (*Module) Shutdown(*module.Context) error { return nil }
