package build

import (
	"errors"
	"fmt"

	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	buildcontract "graft/server/modules/build/contract"
	buildstore "graft/server/modules/build/store"
)

const moduleID = "build"

// Module 声明 Build domain 的生命周期边界，并在 Register 阶段接入其 Task executor 与 HTTP API。
type Module struct {
	service    *Service
	repository buildstore.Repository
}

// NewModule 创建由 Task Runtime 消费的无常驻 Build 模块。
func NewModule(repository buildstore.Repository) *Module { return &Module{repository: repository} }

// Register 注册 Build 权限、导航、Task executor 和 HTTP API，不启动独立 worker。
//
//nolint:cyclop // 显式解析跨模块 capability 保持 Build 的依赖与注册顺序可审计。
func (m *Module) Register(ctx *module.Context) error {
	if ctx == nil || ctx.PermissionRegistry == nil || ctx.MenuRegistry == nil {
		return errors.New("build module registries are unavailable")
	}
	items := []permission.Item{
		{Code: buildcontract.BuildReadPermission, DisplayKey: "rbac.permissionCatalog.buildRead.display", DescriptionKey: "rbac.permissionCatalog.buildRead.description", Module: moduleID, Resource: "build", Action: "read", RiskLevel: permission.RiskLevelLow, RiskCategory: permission.RiskCategoryRead},
		{Code: buildcontract.BuildCreatePermission, DisplayKey: "rbac.permissionCatalog.buildCreate.display", DescriptionKey: "rbac.permissionCatalog.buildCreate.description", Module: moduleID, Resource: "build", Action: "create", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategoryWrite},
		{Code: buildcontract.BuildCancelPermission, DisplayKey: "rbac.permissionCatalog.buildCancel.display", DescriptionKey: "rbac.permissionCatalog.buildCancel.description", Module: moduleID, Resource: "build", Action: "cancel", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategoryDestructive},
		{Code: buildcontract.BuildRetryPermission, DisplayKey: "rbac.permissionCatalog.buildRetry.display", DescriptionKey: "rbac.permissionCatalog.buildRetry.description", Module: moduleID, Resource: "build", Action: "retry", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategoryWrite},
	}
	for _, item := range items {
		ctx.PermissionRegistry.Register(item)
	}
	ctx.MenuRegistry.Register(menu.Item{Code: "build.jobs", ParentCode: "domain.build", Kind: menu.NodeKindEntry, TitleKey: "menu.build.jobs.title", Path: "/build/jobs", Icon: "build", Order: 1, Permission: buildcontract.BuildReadPermission, Module: moduleID})
	contexts, err := module.ResolveService[moduleapi.ApplicationBuildContextResolver](ctx.Services, (*moduleapi.ApplicationBuildContextResolver)(nil))
	if err != nil {
		return fmt.Errorf("resolve application build context resolver: %w", err)
	}
	tasks, err := module.ResolveService[moduleapi.TaskReservationService](ctx.Services, (*moduleapi.TaskReservationService)(nil))
	if err != nil {
		return fmt.Errorf("resolve task service: %w", err)
	}
	registrar, err := module.ResolveService[moduleapi.TaskRuntimeRegistrar](ctx.Services, (*moduleapi.TaskRuntimeRegistrar)(nil))
	if err != nil {
		return fmt.Errorf("resolve task runtime registrar: %w", err)
	}
	docker, err := module.ResolveService[moduleapi.DockerImageBuildCapability](ctx.Services, (*moduleapi.DockerImageBuildCapability)(nil))
	if err != nil {
		return fmt.Errorf("resolve Docker image build capability: %w", err)
	}
	service, err := NewService(contexts, tasks, docker, m.repository)
	if err != nil {
		return err
	}
	if err := registerBuildTaskExecutor(registrar, m.repository, docker); err != nil {
		return err
	}
	m.service = service
	return registerRoutes(ctx, service)
}

// Boot 当前无常驻构建资源。
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown 当前无模块自有资源。
func (*Module) Shutdown(*module.Context) error { return nil }
