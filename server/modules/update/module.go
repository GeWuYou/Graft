package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	"graft/server/internal/realtime"
	updatecontract "graft/server/modules/update/contract"
)

const (
	moduleID                    = "platform-update"
	platformUpdateMenuOrder     = 103
	platformUpdateCheckSchedule = "0 0 */4 * * *"
	deploymentProfileTimeout    = 2 * time.Second
)

// Module 拥有更新发现的注册、周期检查与 HTTP 读取面。
type Module struct {
	service     *Service
	operations  OperationStore
	diagnostics FailureDiagnosticStore
	rollout     *RolloutService
	repository  string
}

// NewModule 创建 platform-update 模块。
func NewModule(operations OperationStore, diagnostics FailureDiagnosticStore, cache DiscoveryCache) *Module {
	repository := os.Getenv("GRAFT_UPDATE_RELEASE_REPOSITORY")
	if repository == "" {
		repository = defaultReleaseRepository
	}
	return &Module{service: NewServiceWithCache(GitHubReleaseProvider{Repository: repository}, cache), operations: operations, diagnostics: diagnostics, repository: repository}
}

// Register 注册权限、菜单、读/check 路由和默认每日发现任务。
//
//nolint:cyclop // 注册顺序定义 Update 的启动依赖与失败边界，保持线性可审计。
func (m *Module) Register(ctx *module.Context) error {
	if err := m.validateRegistration(ctx); err != nil {
		return err
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
	if err := m.configureOutboundNetwork(ctx); err != nil {
		return err
	}
	if err := m.configureDeploymentRuntime(ctx); err != nil {
		return err
	}
	if err := registerUpdateTaskOwnerAuthorizer(ctx); err != nil {
		return err
	}
	if err := m.configureRollout(ctx); err != nil {
		return err
	}
	if ctx.CronRegistry != nil {
		ctx.CronRegistry.Register(cronx.Job{Name: "platform-update.check", Key: "platform-update.check", ModuleKey: moduleID, Module: moduleID, Category: cronx.JobCategoryMaintenance, TitleKey: "scheduledTask.platformUpdateCheck.title", DescriptionKey: "scheduledTask.platformUpdateCheck.description", Schedule: platformUpdateCheckSchedule, DefaultEnabled: true, Handler: func(runCtx context.Context, _ string) (cronx.JobRunResult, error) {
			status := m.service.Check(runCtx)
			if status.CheckError != "" {
				return cronx.JobRunResult{Summary: status.CheckError, Stage: "failed", AffectedResource: "platform_update"}, fmt.Errorf("check platform update: %s", status.CheckError)
			}
			return cronx.JobRunResult{Summary: "platform update check completed", Stage: "completed", AffectedResource: "platform_update"}, nil
		}})
	}
	return registerRoutes(ctx, m.service, m.rollout, m.diagnostics)
}

func (m *Module) configureOutboundNetwork(ctx *module.Context) error {
	factory, err := module.ResolveService[moduleapi.OutboundHTTPClientFactory](ctx.Services, (*moduleapi.OutboundHTTPClientFactory)(nil))
	if err != nil {
		return fmt.Errorf("resolve outbound HTTP client factory: %w", err)
	}
	diagnostics, err := module.ResolveService[moduleapi.OutboundDiagnosticRegistry](ctx.Services, (*moduleapi.OutboundDiagnosticRegistry)(nil))
	if err != nil {
		return fmt.Errorf("resolve outbound diagnostic registry: %w", err)
	}
	consumers, err := module.ResolveService[moduleapi.OutboundNetworkConsumerRegistry](ctx.Services, (*moduleapi.OutboundNetworkConsumerRegistry)(nil))
	if err != nil {
		return fmt.Errorf("resolve outbound network consumer registry: %w", err)
	}
	m.service.provider = GitHubReleaseProvider{Repository: m.repository, ClientFactory: factory}
	if err := diagnostics.RegisterOutboundDiagnosticTarget(platformUpdateDiagnosticTarget{factory: factory}); err != nil {
		return fmt.Errorf("register platform update outbound diagnostic target: %w", err)
	}
	if err := consumers.RegisterOutboundNetworkConsumer(platformUpdateOutboundNetworkConsumer{}); err != nil {
		return fmt.Errorf("register platform update outbound network consumer: %w", err)
	}
	return nil
}

func (m *Module) validateRegistration(ctx *module.Context) error {
	if ctx == nil || m.service == nil || m.operations == nil || m.diagnostics == nil {
		return errors.New("platform-update module context is unavailable")
	}
	return nil
}

func registerUpdateTaskOwnerAuthorizer(ctx *module.Context) error {
	registrar, err := module.ResolveService[moduleapi.TaskRuntimeRegistrar](ctx.Services, (*moduleapi.TaskRuntimeRegistrar)(nil))
	if err != nil {
		return fmt.Errorf("resolve task runtime registrar: %w", err)
	}
	authorizer, err := module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
	if err != nil {
		return fmt.Errorf("resolve authorizer: %w", err)
	}
	if err := registrar.RegisterTaskOwnerAuthorizer(platformUpdateTaskOwnerAuthorizer{authorizer: authorizer}); err != nil {
		return fmt.Errorf("register platform update task owner authorizer: %w", err)
	}
	return nil
}

func (m *Module) configureDeploymentRuntime(ctx *module.Context) error {
	runtime, err := module.ResolveService[moduleapi.DeploymentRuntime](ctx.Services, (*moduleapi.DeploymentRuntime)(nil))
	if err != nil {
		return fmt.Errorf("resolve deployment runtime: %w", err)
	}
	m.service.profile = func() InstallationProfile {
		profileCtx, cancel := context.WithTimeout(ctx.LifecycleContext, deploymentProfileTimeout)
		defer cancel()
		return installationProfile(runtime.Current(profileCtx))
	}
	return nil
}

func (m *Module) configureRollout(ctx *module.Context) error {
	tasks, err := module.ResolveService[moduleapi.TaskService](ctx.Services, (*moduleapi.TaskService)(nil))
	if err != nil {
		return fmt.Errorf("resolve task service: %w", err)
	}
	backups, err := module.ResolveService[moduleapi.BackupService](ctx.Services, (*moduleapi.BackupService)(nil))
	if err != nil {
		return fmt.Errorf("resolve backup service: %w", err)
	}
	launcher, err := NewDockerComposeRunnerLauncher()
	if err != nil {
		return err
	}
	m.rollout = NewRolloutService(m.service, m.operations, tasks, backups, launcher)
	runtime, err := module.ResolveService[moduleapi.DeploymentRuntime](ctx.Services, (*moduleapi.DeploymentRuntime)(nil))
	if err != nil {
		return fmt.Errorf("resolve deployment runtime: %w", err)
	}
	m.rollout.SetDeploymentRuntime(runtime)
	if ctx.Config == nil {
		return errors.New("platform-update config is unavailable")
	}
	m.rollout.SetBackupArtifactRoot(ctx.Config.Backup.ArtifactRoot)
	m.rollout.SetFailureDiagnosticStore(m.diagnostics)
	m.rollout.SetAuditPublisher(ctx.EventPublisher, ctx.Logger)
	m.rollout.SetAppLogger(ctx.AppLogger)
	hub, err := module.ResolveService[realtime.Hub](ctx.Services, (*realtime.Hub)(nil))
	switch {
	case err == nil:
		m.rollout.SetRealtimePublisher(hub)
	case errors.Is(err, container.ErrServiceNotRegistered):
		// realtime hub 是可选能力；未注册时保持更新模块可用。
	default:
		return fmt.Errorf("resolve realtime hub: %w", err)
	}
	return nil
}

// Boot 不需要预先网络检查，避免启动可用性依赖上游 GitHub。
func (m *Module) Boot(ctx *module.Context) error {
	if m == nil || m.rollout == nil || ctx == nil {
		return nil
	}
	if err := m.rollout.ReconcileRunnerState(ctx.LifecycleContext); err != nil && ctx.Logger != nil {
		ctx.Logger.Warn("platform update runner state reconciliation deferred", zap.Error(err))
	}
	m.rollout.StartRunnerStateProjection(ctx.LifecycleContext)
	return nil
}

// Shutdown 释放 rollout 持有的 Docker client，避免模块生命周期结束后遗留连接。
func (m *Module) Shutdown(_ *module.Context) error {
	if m == nil || m.rollout == nil {
		return nil
	}
	return m.rollout.Close()
}

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}
	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range []i18n.MessageKey{
			"menu.platform.maintenance",
			"menu.platform.update",
			"scheduledTask.platformUpdateCheck.title",
			"scheduledTask.platformUpdateCheck.description",
			"update.operation.start.invalid_target",
			"update.operation.start.catalog_stale",
			"update.operation.start.installation_unavailable",
			"update.operation.start.source_version_unsupported",
			"update.operation.start.compose_candidate_invalid",
			"update.operation.start.compose_preflight_failed",
			"update.operation.start.operation_start_failed",
			"update.diagnostics.deployment_runtime_unavailable",
		} {
			if len(localizer.RegisteredMessageResources(locale, key)) == 0 {
				return fmt.Errorf("platform-update locale resource missing %s for %s", key, locale)
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
		code         updatecontract.PermissionCode
		key          string
		action       string
		riskLevel    string
		riskCategory string
	}{{updatecontract.UpdateReadPermission, "platformUpdateRead", "read", permission.RiskLevelLow, permission.RiskCategoryRead}, {updatecontract.UpdateCheckPermission, "platformUpdateCheck", "check", permission.RiskLevelMedium, permission.RiskCategoryWrite}, {updatecontract.UpdateManagePermission, "platformUpdateManage", "manage", permission.RiskLevelHigh, permission.RiskCategorySecurity}} {
		registry.Register(permission.Item{Code: item.code.String(), DisplayKey: "rbac.permissionCatalog." + item.key + ".display", DescriptionKey: "rbac.permissionCatalog." + item.key + ".description", Module: moduleID, Resource: "platform-update", Action: item.action, RiskLevel: item.riskLevel, RiskCategory: item.riskCategory})
	}
	return nil
}
func registerMenu(registry *menu.Registry) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}
	registry.Register(menu.Item{Code: "platform-maintenance", ParentCode: "domain.platform", Kind: menu.NodeKindGroup, TitleKey: "menu.platform.maintenance", Icon: "system-maintenance", Order: platformUpdateMenuOrder, Module: moduleID})
	registry.Register(menu.Item{Code: "platform-update.center", ParentCode: "platform-maintenance", Kind: menu.NodeKindEntry, TitleKey: "menu.platform.update", Path: updatecontract.UpdateMenuPath, Icon: "platform-update", Order: platformUpdateMenuOrder, Permission: updatecontract.UpdateReadPermission.String(), Module: moduleID})
	return nil
}
