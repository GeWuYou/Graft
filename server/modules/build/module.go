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

const (
	moduleID                = "build"
	buildMenuOrderJobs      = 1
	buildMenuOrderArtifacts = 2
)

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
	ctx.MenuRegistry.Register(menu.Item{Code: "build.jobs", ParentCode: "domain.build", Kind: menu.NodeKindEntry, TitleKey: "menu.build.jobs.title", Path: "/build/jobs", Icon: "build", Order: buildMenuOrderJobs, Permission: buildcontract.BuildReadPermission, Module: moduleID})
	ctx.MenuRegistry.Register(menu.Item{Code: "build.artifacts", ParentCode: "domain.build", Kind: menu.NodeKindEntry, TitleKey: "menu.build.artifacts.title", Path: "/build/artifacts", Icon: "image-artifact", Order: buildMenuOrderArtifacts, Permission: buildcontract.BuildReadPermission, Module: moduleID})
	contexts, err := module.ResolveService[moduleapi.ApplicationBuildContextResolver](ctx.Services, (*moduleapi.ApplicationBuildContextResolver)(nil))
	if err != nil {
		return fmt.Errorf("resolve application build context resolver: %w", err)
	}
	submissions, err := module.ResolveService[moduleapi.TaskSubmissionService](ctx.Services, (*moduleapi.TaskSubmissionService)(nil))
	if err != nil {
		return fmt.Errorf("resolve task service: %w", err)
	}
	taskBatch, err := module.ResolveService[moduleapi.TaskBatchQueryService](ctx.Services, (*moduleapi.TaskBatchQueryService)(nil))
	if err != nil {
		return fmt.Errorf("resolve task batch query service: %w", err)
	}
	registrar, err := module.ResolveService[moduleapi.TaskRuntimeRegistrar](ctx.Services, (*moduleapi.TaskRuntimeRegistrar)(nil))
	if err != nil {
		return fmt.Errorf("resolve task runtime registrar: %w", err)
	}
	docker, err := module.ResolveService[moduleapi.DockerImageBuildCapability](ctx.Services, (*moduleapi.DockerImageBuildCapability)(nil))
	if err != nil {
		return fmt.Errorf("resolve Docker image build capability: %w", err)
	}
	targetDocker, _ := module.ResolveService[moduleapi.TargetBoundDockerImageBuildCapability](ctx.Services, (*moduleapi.TargetBoundDockerImageBuildCapability)(nil))
	targetReader, _ := module.ResolveService[moduleapi.BuildRuntimeTargetReader](ctx.Services, (*moduleapi.BuildRuntimeTargetReader)(nil))
	service, err := NewService(contexts, submissions, taskBatch, docker, m.repository)
	if err != nil {
		return err
	}
	configureBuildV2Submission(ctx, service)
	publication, _ := module.ResolveService[moduleapi.TargetBoundDockerImagePublicationCapability](ctx.Services, (*moduleapi.TargetBoundDockerImagePublicationCapability)(nil))
	manifestPublication, _ := module.ResolveService[moduleapi.TargetBoundOCIManifestPublicationCapability](ctx.Services, (*moduleapi.TargetBoundOCIManifestPublicationCapability)(nil))
	snapshotDelivery, _ := module.ResolveService[moduleapi.TargetBoundWorkspaceSnapshotDeliveryCapability](ctx.Services, (*moduleapi.TargetBoundWorkspaceSnapshotDeliveryCapability)(nil))
	conformance, _ := module.ResolveService[moduleapi.TargetBoundProviderExecutionConformanceCapability](ctx.Services, (*moduleapi.TargetBoundProviderExecutionConformanceCapability)(nil))
	provider, _ := module.ResolveService[moduleapi.TargetBoundDockerBuildProvider](ctx.Services, (*moduleapi.TargetBoundDockerBuildProvider)(nil))
	registryPublication, _ := module.ResolveService[moduleapi.RegistryPublicationResolver](ctx.Services, (*moduleapi.RegistryPublicationResolver)(nil))
	if err := registerBuildTaskExecutor(registrar, m.repository, docker, targetDocker, publication, manifestPublication, snapshotDelivery, conformance, provider, registryPublication, service.intents, targetReader); err != nil {
		return err
	}
	if err := registerSnapshotMaterializationCleanupJob(ctx.CronRegistry, service); err != nil {
		return err
	}
	m.service = service
	return registerRoutes(ctx, service)
}

// configureBuildV2Submission 在仓库迁移 legacy read 期间刻意保持可选；生产注册
// 提供全部 authority，focused legacy test 不必构造第二套 fake graph。
func configureBuildV2Submission(ctx *module.Context, service *Service) {
	if ctx == nil || ctx.Services == nil || service == nil {
		return
	}
	snapshots, snapshotErr := module.ResolveService[moduleapi.ApplicationWorkspaceSnapshotResolver](ctx.Services, (*moduleapi.ApplicationWorkspaceSnapshotResolver)(nil))
	targets, targetErr := module.ResolveService[moduleapi.BuildRuntimeTargetReader](ctx.Services, (*moduleapi.BuildRuntimeTargetReader)(nil))
	assignments, assignmentErr := module.ResolveService[moduleapi.RuntimeTargetBuildAssignmentReader](ctx.Services, (*moduleapi.RuntimeTargetBuildAssignmentReader)(nil))
	registry, registryErr := module.ResolveService[moduleapi.RegistryDestinationResolver](ctx.Services, (*moduleapi.RegistryDestinationResolver)(nil))
	if snapshotErr == nil && targetErr == nil && assignmentErr == nil && registryErr == nil {
		service.ConfigureV2Submission(snapshots, targets, assignments, registry)
	}
}

// Boot 当前无常驻构建资源。
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown 当前无模块自有资源。
func (*Module) Shutdown(*module.Context) error { return nil }
