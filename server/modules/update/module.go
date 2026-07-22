package update

import (
	"context"
	"errors"
	"fmt"
	"os"

	"graft/server/internal/cronx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	updatecontract "graft/server/modules/update/contract"
)

const (
	moduleID                    = "platform-update"
	platformUpdateMenuOrder     = 103
	platformUpdateCheckSchedule = "0 0 4 * * *"
)

// Module 拥有更新发现的注册、周期检查与 HTTP 读取面。
type Module struct {
	service    *Service
	operations OperationStore
	rollout    *RolloutService
}

// NewModule 创建 platform-update 模块。
func NewModule(operations OperationStore, cache DiscoveryCache) *Module {
	return &Module{service: NewServiceWithCache(GitHubReleaseProvider{Repository: os.Getenv("GRAFT_UPDATE_RELEASE_REPOSITORY")}, cache), operations: operations}
}

// Register 注册权限、菜单、读/check 路由和默认每日发现任务。
func (m *Module) Register(ctx *module.Context) error {
	if ctx == nil || m.service == nil || m.operations == nil {
		return errors.New("platform-update module context is unavailable")
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
	return registerRoutes(ctx, m.service, m.rollout)
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
	m.rollout.SetAuditBus(ctx.EventBus)
	return nil
}

// Boot 不需要预先网络检查，避免启动可用性依赖上游 GitHub。
func (m *Module) Boot(ctx *module.Context) error {
	if m == nil || m.rollout == nil || ctx == nil {
		return nil
	}
	return m.rollout.SettleAvailableReceipts(ctx.LifecycleContext)
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
		code updatecontract.PermissionCode
		key  string
	}{{updatecontract.UpdateReadPermission, "platformUpdateRead"}, {updatecontract.UpdateCheckPermission, "platformUpdateCheck"}, {updatecontract.UpdateManagePermission, "platformUpdateManage"}} {
		registry.Register(permission.Item{Code: item.code.String(), DisplayKey: "rbac.permissionCatalog." + item.key + ".display", DescriptionKey: "rbac.permissionCatalog." + item.key + ".description", Module: moduleID})
	}
	return nil
}
func registerMenu(registry *menu.Registry) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}
	registry.Register(menu.Item{Code: "platform-maintenance", ParentCode: "domain.platform", Kind: menu.NodeKindGroup, TitleKey: "menu.platform.maintenance", Icon: "platform-configuration", Order: platformUpdateMenuOrder, Module: moduleID})
	registry.Register(menu.Item{Code: "platform-update.center", ParentCode: "platform-maintenance", Kind: menu.NodeKindEntry, TitleKey: "menu.platform.update", Path: updatecontract.UpdateMenuPath, Icon: "platform-update", Order: platformUpdateMenuOrder, Permission: updatecontract.UpdateReadPermission.String(), Module: moduleID})
	return nil
}
